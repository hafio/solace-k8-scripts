package container

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"path/filepath"
	"strings"

	"solace/internal/config"
	"solace/internal/engine"
	"solace/internal/render"
)

// Manager deploys and operates the single broker container on THIS host -- the
// container analog of k8s.Cluster. Where k8s.Cluster drives every pod in a
// namespace from one control point, a container host runs one broker, so the
// Manager's operations are node-local and the HA coordination (leader,
// redundancy) is a per-host handshake handled in package broker. Every mutating
// command routes through the engine.Runner, so --dry-run echoes without running.
type Manager struct {
	R   engine.Runner
	Cfg *config.Config
	P   config.Platform
	Log func(format string, args ...any) // progress -> stderr; nil discards
	Out io.Writer                        // user-facing output; nil -> os.Stdout
	In  io.Reader                        // reserved for prompts; nil -> os.Stdin

	// Resolve reports whether a hostname resolves; a seam over net.LookupHost so
	// Check/PrepHost DNS probes are testable. NewManager sets the default.
	Resolve func(host string) bool
	// GenPSK returns a freshly generated redundancy pre-shared key. NewManager sets
	// a crypto/rand + base64 default; tests inject a fixed value.
	GenPSK func() (string, error)
	// Geteuid reports this process's effective uid; a seam over os.Geteuid so the
	// podman rootless/rootful guard is testable off a POSIX host (Windows returns
	// -1). NewManager sets the default.
	Geteuid func() int
	// EnvPath is the resolved env-file path, so PrepHost can write a generated PSK
	// back into it (the analog of 002-host-prep.sh's sed rewrite).
	EnvPath string

	// Restart pre-approves bouncing an already-running broker when the deploy
	// artifact changed (the --restart flag).
	Restart bool
	// Confirm asks the operator a yes/no question. nil declines, which is what a
	// non-interactive run must do: re-deploying should never bounce a running
	// broker unattended. The CLI wires it to the same prompt style as delete.
	Confirm func(question string) bool
}

// NewManager builds a container host Manager over the given runner, config and
// platform. It defaults Resolve to a net.LookupHost seam and GenPSK to a
// crypto/rand + standard-base64 generator (matching 002-host-prep.sh's
// `openssl rand -base64 60`). In and EnvPath are left for the caller to set.
func NewManager(r engine.Runner, cfg *config.Config, p config.Platform, log func(string, ...any), out io.Writer) *Manager {
	return &Manager{
		R:       r,
		Cfg:     cfg,
		P:       p,
		Log:     log,
		Out:     out,
		Resolve: func(host string) bool { _, err := net.LookupHost(host); return err == nil },
		GenPSK:  defaultGenPSK,
		Geteuid: os.Geteuid,
	}
}

func (m *Manager) logf(format string, args ...any) {
	if m.Log != nil {
		m.Log(format, args...)
	}
}

// out returns the user-facing sink, defaulting to os.Stdout.
func (m *Manager) out() io.Writer {
	if m.Out != nil {
		return m.Out
	}
	return os.Stdout
}

// isDryRun reports whether the runner only echoes (engine.Echo), so file writes
// and DNS/euid probes are previewed rather than performed.
func (m *Manager) isDryRun() bool { _, ok := m.R.(engine.Echo); return ok }

// runtime is the configured runtime command (docker.runtime / podman.runtime):
// argv[0] plus any leading arguments that precede every call's own.
func (m *Manager) runtime() config.Command { return m.Cfg.ContainerRuntime(m.P) }
func (m *Manager) name() string            { return m.Cfg.ContainerBlock(m.P).Name }

// run executes a runtime subcommand (`<runtime> args...`) through the Runner.
func (m *Manager) run(ctx context.Context, args ...string) error {
	r := m.runtime()
	return m.R.Run(ctx, r.Name(), r.Args(args...)...)
}

// output captures a runtime subcommand's stdout through the Runner.
func (m *Manager) output(ctx context.Context, args ...string) ([]byte, error) {
	r := m.runtime()
	return m.R.Output(ctx, r.Name(), r.Args(args...)...)
}

// composeCmd is the compose invocation (docker only): whatever docker.compose
// names -- a host with only the standalone v1 binary sets it to `docker-compose`
// -- defaulting to the runtime's own `compose` subcommand. ApplyDefaults fills it
// too, so the fallback here only matters for a hand-built config (the same
// arrangement composeFile uses).
func (m *Manager) composeCmd() config.Command {
	if len(m.Cfg.Docker.Compose) > 0 {
		return m.Cfg.Docker.Compose
	}
	rt := m.runtime()
	compose := make(config.Command, 0, len(rt)+1)
	compose = append(compose, rt...)
	return append(compose, "compose")
}

// compose runs a compose subcommand (`<compose> args...`) through the Runner.
func (m *Manager) compose(ctx context.Context, args ...string) error {
	c := m.composeCmd()
	return m.R.Run(ctx, c.Name(), c.Args(args...)...)
}

// --- Check ------------------------------------------------------------------

// Check prints the resolved container configuration, probes the runtime, and
// resolves the broker hostname(s). It never mutates host or container state.
func (m *Manager) Check(ctx context.Context) error {
	m.CheckEnv()
	if err := m.Reachable(ctx); err != nil {
		return err
	}
	return m.checkDNS(ctx)
}

// CheckEnv writes the effective container configuration, mirroring k8s.CheckEnv.
func (m *Manager) CheckEnv() {
	w := m.out()
	cfg := m.Cfg
	cb := cfg.ContainerBlock(m.P)
	nw := cfg.NetworkBlock(m.P)

	mode := "standalone (single broker)"
	if cfg.RedundancyEnabled() {
		mode = "HA redundancy group (primary + backup + monitor)"
	}

	fmt.Fprintf(w, "Solace broker deployment (%s):\n", platformTitle(m.P))
	fmt.Fprintf(w, "  container      : name=%s runtime=%s\n", cb.Name, cfg.ContainerRuntime(m.P))
	fmt.Fprintf(w, "  redundancy     : %s\n", mode)
	fmt.Fprintf(w, "  image          : %s\n", orNone(cfg.Image.Ref()))
	if cfg.Image.User != "" || cfg.Image.Pass != "" {
		fmt.Fprintf(w, "  registry login : user=%s password=%s (prep logs in with `%s login`)\n",
			orNone(cfg.Image.User), setOrMissing(cfg.Image.Pass), cfg.ContainerRuntime(m.P).Name())
	}
	fmt.Fprintf(w, "  data dir       : %s\n", cb.DataDir)
	fmt.Fprintf(w, "  run user       : %s\n", cb.RunUser)
	if nw.Mode == "host" {
		fmt.Fprintln(w, "  network        : host")
	} else {
		fmt.Fprintf(w, "  network        : bridge ports=%d\n", len(nw.Ports))
	}
	fmt.Fprintf(w, "  admin          : user=%s password=%s\n", cfg.Admin.User, setOrMissing(cfg.Admin.Pass))
	if cfg.TLS.Cert != "" || cfg.TLS.CertKey != "" {
		fmt.Fprintf(w, "  tls            : cert=%s key=%s cas=%d\n", orNone(cfg.TLS.Cert), setOrMissing(cfg.TLS.CertKey), len(cfg.TLS.CAs))
	} else {
		fmt.Fprintln(w, "  tls            : (not configured)")
	}
	if m.P == config.Podman {
		fmt.Fprintf(w, "  podman         : rootless=%t quadletDir=%s\n", cfg.Podman.Rootless, cfg.Podman.QuadletDir)
	} else {
		fmt.Fprintf(w, "  docker         : compose=%s composeFile=%s\n", m.composeCmd(), m.composeFile())
	}
	fmt.Fprintf(w, "  secrets        : %s\n", secretSummary(m.P, render.ContainerSecrets(cfg)))
	if cfg.RedundancyEnabled() {
		n := cfg.Nodes
		fmt.Fprintf(w, "  primary        : %s (%s)\n", n.Primary.Name, orNone(n.Primary.IP))
		fmt.Fprintf(w, "  backup         : %s (%s)\n", n.Backup.Name, orNone(n.Backup.IP))
		fmt.Fprintf(w, "  monitor        : %s (%s)\n", n.Monitor.Name, orNone(n.Monitor.IP))
		fmt.Fprintf(w, "  psk            : %s\n", setOrMissing(n.PSK))
	} else {
		fmt.Fprintf(w, "  node           : %s\n", orNone(cfg.Nodes.Primary.Name))
	}
}

// Reachable probes `<runtime> version` so a missing/stopped engine fails with an
// actionable error before any deploy step runs. On docker it also probes the
// configured compose command, since every deploy goes through it and the plugin
// is a separate install from the engine. Under --dry-run it only echoes.
func (m *Manager) Reachable(ctx context.Context) error {
	if _, err := m.output(ctx, "version"); err != nil {
		return fmt.Errorf("cannot reach the %s runtime %q (is it installed and running?): %w", platformTitle(m.P), m.runtime(), err)
	}
	if m.P == config.Docker {
		c := m.composeCmd()
		if _, err := m.R.Output(ctx, c.Name(), c.Args("version")...); err != nil {
			return fmt.Errorf("cannot run the compose command %q (install the docker compose plugin, "+
				"or set docker.compose to this host's standalone 'docker-compose' binary): %w", c, err)
		}
	}
	return nil
}

// checkDNS resolves the broker hostname(s): in HA every node name must resolve
// (fail loud on any miss, matching 002-host-prep.sh); standalone warns only on
// the single name (it is used just as the routername). Skipped under --dry-run,
// where no real lookups make sense.
func (m *Manager) checkDNS(ctx context.Context) error {
	_ = ctx
	if m.isDryRun() {
		fmt.Fprintln(m.out(), "  dns            : skipped (--dry-run)")
		return nil
	}
	w := m.out()
	if !m.Cfg.RedundancyEnabled() {
		name := m.Cfg.Nodes.Primary.Name
		if name == "" || m.Resolve(name) {
			fmt.Fprintf(w, "  [ OK ] broker hostname resolves: %s\n", orNone(name))
		} else {
			m.logf("[WARN] broker hostname does not resolve: %s (standalone -- used only as the routername, usually fine)", name)
		}
		return nil
	}
	failed := 0
	for _, n := range []struct{ role, name string }{
		{"primary", m.Cfg.Nodes.Primary.Name},
		{"backup", m.Cfg.Nodes.Backup.Name},
		{"monitor", m.Cfg.Nodes.Monitor.Name},
	} {
		if m.Resolve(n.name) {
			fmt.Fprintf(w, "  [ OK ] %s hostname resolves: %s\n", n.role, n.name)
			continue
		}
		fmt.Fprintf(w, "  [ERROR] %s hostname does NOT resolve: %s\n", n.role, n.name)
		failed++
	}
	if failed > 0 {
		return fmt.Errorf("%d redundancy-group hostname(s) do not resolve; add them to DNS or /etc/hosts on this host", failed)
	}
	return nil
}

// --- PrepHost ---------------------------------------------------------------

// PrepHost prepares this host for the broker container: it creates and chowns
// the data directory, resolves the broker hostname(s), and (HA only) ensures a
// redundancy PSK exists -- generating one and writing it back into the env file
// when absent. It ports bash/docker-podman/002-host-prep.sh.
func (m *Manager) PrepHost(ctx context.Context) error {
	if m.P == config.Podman && m.Cfg.Podman.Rootless && m.Geteuid() == 0 {
		m.logf("[WARN] podman.rootless=true but running as root; run prep as the target rootless user so subuid/subgid mapping matches the deploy.")
	}

	cb := m.Cfg.ContainerBlock(m.P)
	m.logf("Preparing data directory %s", cb.DataDir)
	if err := m.R.Run(ctx, "mkdir", "-p", cb.DataDir); err != nil {
		return fmt.Errorf("create data dir %q: %w", cb.DataDir, err)
	}
	if m.P == config.Podman && m.Cfg.Podman.Rootless {
		// Rootless podman: the container's uid:gid maps through subuid/subgid, so
		// enter the user namespace to apply the ownership the container will see.
		if err := m.run(ctx, "unshare", "chown", cb.RunUser, cb.DataDir); err != nil {
			return fmt.Errorf("chown data dir %q (rootless): %w", cb.DataDir, err)
		}
	} else if err := m.R.Run(ctx, "chown", cb.RunUser, cb.DataDir); err != nil {
		return fmt.Errorf("chown data dir %q to %q: %w", cb.DataDir, cb.RunUser, err)
	}

	if err := m.checkDNS(ctx); err != nil {
		return err
	}
	if err := m.registryLogin(ctx); err != nil {
		return err
	}
	return m.prepPSK()
}

// registryLogin authenticates this host to the image registry when credentials are
// configured. Containers have no analog of the k8s image-pull Secret, so without
// this the credentials in the env file did nothing here and a private-registry pull
// simply failed unauthenticated. The password is fed on stdin, so it never reaches
// an argv or the --dry-run echo (§3).
func (m *Manager) registryLogin(ctx context.Context) error {
	user, pass := m.Cfg.Image.User, m.Cfg.Image.Pass
	if user == "" && pass == "" {
		return nil
	}
	if user == "" || pass == "" {
		return fmt.Errorf("image.user and image.pass must both be set to log in to the registry (one of them is empty)")
	}
	registry := m.Cfg.Image.Registry
	args := []string{"login", "--username", user, "--password-stdin"}
	if registry != "" {
		args = append(args, registry)
	}
	m.logf("Logging in to registry %s as %s", orNone(registry), user)
	r := m.runtime()
	if err := m.R.RunInput(ctx, []byte(pass), r.Name(), r.Args(args...)...); err != nil {
		return fmt.Errorf("%s login to registry %s failed: %w", platformTitle(m.P), orNone(registry), err)
	}
	return nil
}

// prepPSK ensures a redundancy PSK exists (HA only). If nodes.psk is already set
// it is left unchanged (with a reminder to keep it identical across hosts). If
// empty, a PSK is generated and written back into the env file -- but never
// under --dry-run, which must not write files or echo secret bytes.
func (m *Manager) prepPSK() error {
	if !m.Cfg.RedundancyEnabled() {
		m.logf("[Info] standalone mode -- no redundancy PSK needed; skipping.")
		return nil
	}
	if m.Cfg.Nodes.PSK != "" {
		m.logf("[Info] nodes.psk is already set -- leaving it unchanged.")
		m.logf("[WARN] the SAME psk must be present in the env file on all three hosts.")
		return nil
	}
	if m.isDryRun() {
		m.logf("[Info] nodes.psk is empty -- a PSK would be generated and written to %s (skipped under --dry-run).", m.EnvPath)
		return nil
	}
	psk, err := m.GenPSK()
	if err != nil {
		return fmt.Errorf("generate redundancy PSK: %w", err)
	}
	return m.writePSK(psk)
}

// writePSK stores a generated PSK by rewriting the `psk:` line under the env
// file's `nodes:` block, preserving everything else. If no such line exists it
// prints the value for the operator to place rather than guessing where to
// insert it. The env file now holds a secret, so it is written back mode 0600.
func (m *Manager) writePSK(psk string) error {
	if m.EnvPath == "" {
		return fmt.Errorf("cannot store generated PSK: env-file path is unknown")
	}
	raw, err := os.ReadFile(m.EnvPath)
	if err != nil {
		return fmt.Errorf("read env file %q to store PSK: %w", m.EnvPath, err)
	}
	updated, replaced := replacePSKLine(string(raw), psk)
	if !replaced {
		m.logf("[WARN] no nodes.psk line found in %s; add this line under nodes: and copy it to all three hosts:", m.EnvPath)
		fmt.Fprintf(m.out(), "  psk: %q\n", psk)
		return nil
	}
	if err := os.WriteFile(m.EnvPath, []byte(updated), 0o600); err != nil {
		return fmt.Errorf("write PSK back to env file %q: %w", m.EnvPath, err)
	}
	m.logf("[ OK ] generated a redundancy PSK and wrote it to %s", m.EnvPath)
	m.logf("[WARN] copy the SAME psk into the env file on the OTHER two hosts (it must match).")
	return nil
}

// --- Deploy -----------------------------------------------------------------

// Deploy renders and starts the broker container for role's identity. Podman
// uses a systemd quadlet unit; docker uses a compose file. The broker's secret
// settings are externalized first, since the rendered artifact only references
// them by name.
func (m *Manager) Deploy(ctx context.Context, role config.Role) error {
	id := m.Cfg.ResolveNode(role)
	if m.P == config.Podman {
		if err := m.checkPodmanEUID(); err != nil {
			return err
		}
	}
	if err := m.prepareSecrets(ctx); err != nil {
		return err
	}
	if m.P == config.Podman {
		return m.deployPodman(ctx, id)
	}
	return m.deployDocker(ctx, id)
}

// prepareSecrets externalizes the broker's secret settings so the deploy artifact
// can reference them instead of carrying them: podman loads each into its own
// secret store, docker writes the 0600 files backing its compose secrets. An
// empty value fails loud here rather than deploying a broker with no password --
// except under --dry-run, which must stay previewable before `prep host` has
// generated the HA pre-shared key.
func (m *Manager) prepareSecrets(ctx context.Context) error {
	secrets := render.ContainerSecrets(m.Cfg)
	// Skipped under --dry-run so a preview stays possible before `prep host` has
	// generated the PSK; `--gen-secrets-only` runs the same check, since the script
	// it prints is meant to be executed.
	if !m.isDryRun() {
		if err := render.SecretPreflight(m.Cfg, m.P); err != nil {
			return err
		}
	}
	if m.P == config.Podman {
		return m.createPodmanSecrets(ctx, secrets)
	}
	return m.writeDockerSecrets(secrets)
}

// createPodmanSecrets loads each secret into podman's secret store, feeding the
// value on stdin so it never reaches an argv or the --dry-run echo (§3).
// --replace makes a redeploy with a rotated value idempotent.
func (m *Manager) createPodmanSecrets(ctx context.Context, secrets []render.ContainerSecret) error {
	r := m.runtime()
	for _, s := range secrets {
		if err := m.R.RunInput(ctx, []byte(s.Value), r.Name(),
			r.Args("secret", "create", "--replace", s.Name, "-")...); err != nil {
			return fmt.Errorf("create podman secret %q from %s: %w", s.Name, s.ConfigKey, err)
		}
	}
	return nil
}

// writeDockerSecrets writes the file-backed sources of the compose secrets, 0600
// inside a 0700 directory beside the compose file (the path the rendered compose
// file points at). Skipped under --dry-run, which must not write files or echo
// secret bytes.
func (m *Manager) writeDockerSecrets(secrets []render.ContainerSecret) error {
	dir := m.secretsDir()
	if m.isDryRun() {
		m.logf("[Info] would write %d secret file(s) to %s (skipped under --dry-run).", len(secrets), dir)
		return nil
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create secrets dir %q: %w", dir, err)
	}
	for _, s := range secrets {
		path := filepath.Join(dir, s.Name)
		if err := os.WriteFile(path, []byte(s.Value), 0o600); err != nil {
			return fmt.Errorf("write secret file %q: %w", path, err)
		}
	}
	m.logf("[ OK ] wrote %d secret file(s) to %s", len(secrets), dir)
	return nil
}

// secretsDir is where docker's compose-secret sources live: beside the compose
// file, so the rendered `file: ./<dir>/<name>` reference resolves relative to it.
func (m *Manager) secretsDir() string {
	return filepath.Join(filepath.Dir(m.composeFile()), render.SecretsDirName)
}

// writeArtifact writes a rendered deploy artifact, reporting whether it differs
// from what was already on disk. An unchanged artifact is not rewritten, which is
// what lets Deploy tell "nothing to do" apart from "the running broker is now
// stale and needs a bounce". Under --dry-run nothing is written and the artifact
// counts as changed, so the preview shows the work a real run would do.
func (m *Manager) writeArtifact(path string, body []byte, what string, dirMode os.FileMode) (bool, error) {
	if m.isDryRun() {
		m.logf("[Info] would write %s %s (skipped under --dry-run).", what, path)
		return true, nil
	}
	if existing, err := os.ReadFile(path); err == nil && bytes.Equal(existing, body) {
		m.logf("[Info] %s %s is already up to date.", what, path)
		return false, nil
	}
	if dirMode != 0 {
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, dirMode); err != nil {
			return false, fmt.Errorf("create %s dir %q: %w", what, dir, err)
		}
	}
	if err := os.WriteFile(path, body, 0o600); err != nil {
		return false, fmt.Errorf("write %s %q: %w", what, path, err)
	}
	m.logf("[ OK ] wrote %s %s", what, path)
	return true, nil
}

// approveRestart decides whether an already-running broker may be bounced to pick
// up a changed artifact. --restart pre-approves it; otherwise the Confirm seam
// asks. A non-interactive run without --restart declines, so a scripted deploy
// leaves the new artifact in place and says so rather than dropping messaging
// traffic on its own. Deliberately not --yes: bouncing a live broker is its own
// decision, the same separation --purge has from --yes.
func (m *Manager) approveRestart(what string) bool {
	if m.Restart {
		return true
	}
	if m.Confirm == nil {
		return false
	}
	return m.Confirm(fmt.Sprintf("Restart %s now to apply the change?", what))
}

// staleWarning tells the operator the artifact is applied but the running broker
// is still on the old one, and how to finish the job.
func (m *Manager) staleWarning(what string) {
	m.logf("[WARN] %s is applied, but the running broker still uses the previous one.", what)
	m.logf("[WARN] re-run with --restart (or restart it yourself) to pick up the change.")
}

func (m *Manager) deployPodman(ctx context.Context, id config.NodeIdentity) error {
	unit := filepath.ToSlash(filepath.Join(m.Cfg.Podman.QuadletDir, m.name()+".container"))
	svc := m.name() + ".service"
	changed, err := m.writeArtifact(unit, render.Quadlet(m.Cfg, id), "quadlet unit", 0o755)
	if err != nil {
		return err
	}
	// daemon-reload runs either way: a unit that was already correct may still be
	// unknown to systemd (a fresh host, or a manual removal).
	if err := m.systemctl(ctx, "daemon-reload"); err != nil {
		return err
	}
	if !m.serviceActive(ctx) {
		return m.systemctl(ctx, "start", svc)
	}
	// `systemctl start` on an active unit is a no-op, so an already-running broker
	// would silently keep the old image. Restart is the only way to apply a change.
	if !changed {
		m.logf("[Info] %s is already active on this unit -- nothing to do.", svc)
		return nil
	}
	if !m.approveRestart(svc) {
		m.staleWarning("the quadlet unit")
		return nil
	}
	return m.systemctl(ctx, "restart", svc)
}

func (m *Manager) deployDocker(ctx context.Context, id config.NodeIdentity) error {
	file := m.composeFile()
	changed, err := m.writeArtifact(file, render.Compose(m.Cfg, id), "compose file", 0)
	if err != nil {
		return err
	}
	if !m.containerRunning(ctx) {
		return m.compose(ctx, "-f", file, "up", "-d")
	}
	// `compose up -d` recreates the container when the file changed, which bounces
	// a running broker -- the same hazard podman has, so it takes the same consent.
	if !changed {
		m.logf("[Info] container %s is already running on this compose file -- nothing to do.", m.name())
		return nil
	}
	if !m.approveRestart("container " + m.name()) {
		m.staleWarning("the compose file")
		return nil
	}
	return m.compose(ctx, "-f", file, "up", "-d")
}

// serviceActive reports whether this host's broker unit is already active, so
// Deploy can restart it instead of issuing a no-op start. Under --dry-run no host
// state is probed (matching checkPodmanEUID/checkDNS) and the answer is "not
// active", so the preview shows the plain start path.
func (m *Manager) serviceActive(ctx context.Context) bool {
	if m.isDryRun() {
		return false
	}
	out, err := m.systemctlOutput(ctx, "is-active", m.name()+".service")
	return err == nil && strings.TrimSpace(string(out)) == "active"
}

// containerRunning reports whether this host's broker container is up. Same
// dry-run rule as serviceActive.
func (m *Manager) containerRunning(ctx context.Context) bool {
	if m.isDryRun() {
		return false
	}
	out, err := m.output(ctx, "ps", "--quiet", "--filter", "name="+m.name(), "--filter", "status=running")
	return err == nil && strings.TrimSpace(string(out)) != ""
}

// --- Delete -----------------------------------------------------------------

// Delete stops and removes the broker container (and its unit/compose artifact),
// then optionally removes the data directory when purge is set.
func (m *Manager) Delete(ctx context.Context, purge bool) error {
	var err error
	if m.P == config.Podman {
		err = m.deletePodman(ctx)
	} else {
		err = m.deleteDocker(ctx)
	}
	if err != nil {
		return err
	}
	if purge {
		return m.purgeData(ctx)
	}
	m.logf("[Info] data directory %s kept (pass --purge to remove it).", m.Cfg.ContainerBlock(m.P).DataDir)
	return nil
}

func (m *Manager) deletePodman(ctx context.Context) error {
	svc := m.name() + ".service"
	if err := m.systemctl(ctx, "stop", svc); err != nil {
		m.logf("[WARN] stopping %s failed (already stopped?): %v", svc, err)
	}
	unit := filepath.ToSlash(filepath.Join(m.Cfg.Podman.QuadletDir, m.name()+".container"))
	if m.isDryRun() {
		m.logf("[Info] would remove quadlet unit %s (skipped under --dry-run).", unit)
	} else if err := os.Remove(unit); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove quadlet unit %q: %w", unit, err)
	} else {
		m.logf("[ OK ] removed quadlet unit %s", unit)
	}
	return m.systemctl(ctx, "daemon-reload")
}

func (m *Manager) deleteDocker(ctx context.Context) error {
	file := m.composeFile()
	if m.isDryRun() || fileExists(file) {
		return m.compose(ctx, "-f", file, "down")
	}
	// No compose file on disk: fall back to stop/rm by container name.
	return m.stopAndRemove(ctx)
}

func (m *Manager) stopAndRemove(ctx context.Context) error {
	if err := m.run(ctx, "stop", m.name()); err != nil {
		m.logf("[WARN] stopping container %s failed (already stopped?): %v", m.name(), err)
	}
	return m.run(ctx, "rm", m.name())
}

func (m *Manager) purgeData(ctx context.Context) error {
	dir := m.Cfg.ContainerBlock(m.P).DataDir
	m.logf("Removing data directory %s", dir)
	if m.P == config.Podman && m.Cfg.Podman.Rootless {
		return m.run(ctx, "unshare", "rm", "-rf", dir)
	}
	return m.R.Run(ctx, "rm", "-rf", dir)
}

// --- Status / Logs / CLI / Shell --------------------------------------------

// Status reports the container's state. Podman also shows its systemd unit.
func (m *Manager) Status(ctx context.Context) error {
	if m.P == config.Podman {
		svc := m.name() + ".service"
		if err := m.systemctl(ctx, "status", svc, "--no-pager"); err != nil {
			m.logf("[WARN] systemctl status %s reported non-zero (unit not active?): %v", svc, err)
		}
		return m.run(ctx, "ps", "--all", "--filter", "name="+m.name())
	}
	file := m.composeFile()
	if m.isDryRun() || fileExists(file) {
		if err := m.compose(ctx, "-f", file, "ps"); err != nil {
			m.logf("[WARN] compose ps failed: %v", err)
		}
	}
	return m.run(ctx, "ps", "--all", "--filter", "name="+m.name())
}

// Describe prints detailed inspection output for this host's broker, the container
// analog of `kubectl describe pod`: `<runtime> inspect` carries the health state,
// restart count, exit reason, mounts and resource limits that Status's one-line
// listing does not. Podman additionally shows the installed unit definition, which
// answers "what did we actually deploy" -- the summary is already in Status.
func (m *Manager) Describe(ctx context.Context) error {
	if m.P == config.Podman {
		svc := m.name() + ".service"
		if err := m.systemctl(ctx, "cat", svc); err != nil {
			m.logf("[WARN] systemctl cat %s failed (unit not installed?): %v", svc, err)
		}
	}
	return m.run(ctx, "inspect", m.name())
}

// Logs streams the broker container's logs (`<runtime> logs -f <name>`).
func (m *Manager) Logs(ctx context.Context) error {
	return m.run(ctx, "logs", "-f", m.name())
}

// CopyFrom copies files out of the broker container into the working directory,
// mirroring the k8s verb: each file is attempted, failures are reported per file,
// and the command exits non-zero if any failed.
func (m *Manager) CopyFrom(ctx context.Context, files []string) error {
	if len(files) == 0 {
		return fmt.Errorf("no files specified to copy from the broker")
	}
	t := NewTransport(m.R, m.Cfg, m.P)
	var failed int
	for _, f := range files {
		local := path.Base(f)
		m.logf("copying %s from container %s", f, m.name())
		if err := t.Download(ctx, config.Primary, f, local); err != nil {
			fmt.Fprintf(m.out(), "  [ERROR] %s: %v\n", f, err)
			failed++
			continue
		}
		fmt.Fprintf(m.out(), "  [ OK ] %s -> %s\n", f, local)
	}
	if failed > 0 {
		return fmt.Errorf("%d of %d file(s) failed to copy from the broker", failed, len(files))
	}
	return nil
}

// CopyInto copies local files into destDir inside the broker container.
func (m *Manager) CopyInto(ctx context.Context, files []string, destDir string) error {
	if len(files) == 0 {
		return fmt.Errorf("no files specified to copy into the broker")
	}
	if destDir == "" {
		destDir = "."
	}
	t := NewTransport(m.R, m.Cfg, m.P)
	var failed int
	for _, f := range files {
		m.logf("copying %s into container %s:%s", f, m.name(), destDir)
		if err := t.UploadFile(ctx, config.Primary, f, destDir); err != nil {
			fmt.Fprintf(m.out(), "  [ERROR] %s: %v\n", f, err)
			failed++
			continue
		}
		fmt.Fprintf(m.out(), "  [ OK ] %s -> %s:%s\n", f, m.name(), destDir)
	}
	if failed > 0 {
		return fmt.Errorf("%d of %d file(s) failed to copy into the broker", failed, len(files))
	}
	return nil
}

// CLI opens an interactive Solace CLI session inside the container.
func (m *Manager) CLI(ctx context.Context) error {
	r := m.runtime()
	return m.R.RunInteractive(ctx, r.Name(), r.Args("exec", "-it", m.name(), "cli", "-A")...)
}

// Shell opens an interactive shell inside the container.
func (m *Manager) Shell(ctx context.Context) error {
	r := m.runtime()
	return m.R.RunInteractive(ctx, r.Name(), r.Args("exec", "-it", m.name(), "bash")...)
}

// --- helpers ----------------------------------------------------------------

// systemctl runs `systemctl [--user] args...` through the Runner, honoring the
// rootless (`--user`) vs rootful mode derived in config.
func (m *Manager) systemctl(ctx context.Context, args ...string) error {
	return m.R.Run(ctx, "systemctl", m.systemctlArgs(args)...)
}

// systemctlOutput captures a systemctl subcommand's stdout, for the state probes
// (`is-active`) whose answer picks the next step rather than being shown.
func (m *Manager) systemctlOutput(ctx context.Context, args ...string) ([]byte, error) {
	return m.R.Output(ctx, "systemctl", m.systemctlArgs(args)...)
}

// systemctlArgs prepends the `--user` token when the config is rootless, honoring
// the rootless vs rootful mode derived in config.
func (m *Manager) systemctlArgs(args []string) []string {
	if u := m.Cfg.Podman.SystemctlUser; u != "" {
		return append([]string{u}, args...)
	}
	return args
}

// checkPodmanEUID enforces the rootless/rootful invariant before a real deploy:
// rootless must not run as root and rootful must. It is skipped under --dry-run
// and on platforms without a meaningful euid (Windows returns -1), so the deploy
// stays previewable everywhere.
func (m *Manager) checkPodmanEUID() error {
	if m.isDryRun() {
		return nil
	}
	euid := m.Geteuid()
	if euid < 0 {
		return nil
	}
	if m.Cfg.Podman.Rootless && euid == 0 {
		return fmt.Errorf("podman.rootless=true but running as root; run as the target rootless user (no sudo)")
	}
	if !m.Cfg.Podman.Rootless && euid != 0 {
		return fmt.Errorf("rootful podman requires root; re-run with sudo or set podman.rootless=true")
	}
	return nil
}

// composeFile is the docker compose path, defaulting an empty config value to
// docker-compose.yml (config leaves it empty; the manager owns the default).
func (m *Manager) composeFile() string {
	if f := m.Cfg.Docker.ComposeFile; f != "" {
		return f
	}
	return "docker-compose.yml"
}

// defaultGenPSK returns a base64 pre-shared key from 60 crypto-random bytes,
// matching 002-host-prep.sh's `openssl rand -base64 60`.
func defaultGenPSK() (string, error) {
	b := make([]byte, 60)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("read random bytes for PSK: %w", err)
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// replacePSKLine rewrites the `psk:` line inside the top-level `nodes:` block of
// a YAML document, preserving its indentation and every other line. It returns
// the updated content and whether a line was replaced. Scoping to the nodes:
// block avoids touching an unrelated `psk:` key elsewhere in the file.
func replacePSKLine(content, psk string) (string, bool) {
	lines := strings.Split(content, "\n")
	inNodes := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") && indent == 0 {
			inNodes = strings.HasPrefix(trimmed, "nodes:")
			continue
		}
		if inNodes && indent > 0 && strings.HasPrefix(trimmed, "psk:") {
			lines[i] = fmt.Sprintf("%spsk: %q", line[:indent], psk)
			return strings.Join(lines, "\n"), true
		}
	}
	return content, false
}

// fileExists reports whether p exists (used to pick compose-down vs stop/rm).
func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// platformTitle is the display name for the platform in Check output/errors.
func platformTitle(p config.Platform) string {
	if p == config.Podman {
		return "Podman"
	}
	return "Docker"
}

// orNone renders an empty string as "(none)" -- a package-local copy of the k8s
// report formatter (one small copy per package, not over-DRY'd across packages).
func orNone(s string) string {
	if s == "" {
		return "(none)"
	}
	return s
}

// setOrMissing renders a secret's presence without echoing it.
func setOrMissing(s string) string {
	if s == "" {
		return "MISSING"
	}
	return "set"
}

// secretSummary reports the externalized secrets by name, whether each value is
// present, and the mechanism this platform stores them in. Values never appear.
func secretSummary(p config.Platform, secrets []render.ContainerSecret) string {
	store := "podman secret store"
	if p == config.Docker {
		store = "compose secret files"
	}
	parts := make([]string, 0, len(secrets))
	for _, s := range secrets {
		parts = append(parts, s.Name+"="+setOrMissing(s.Value))
	}
	return fmt.Sprintf("%s (%s)", strings.Join(parts, " "), store)
}
