package container

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"os"
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

func (m *Manager) runtime() string { return m.Cfg.ContainerRuntime(m.P) }
func (m *Manager) name() string    { return m.Cfg.ContainerBlock(m.P).Name }

// run executes a runtime subcommand (`<runtime> args...`) through the Runner.
func (m *Manager) run(ctx context.Context, args ...string) error {
	return m.R.Run(ctx, m.runtime(), args...)
}

// output captures a runtime subcommand's stdout through the Runner.
func (m *Manager) output(ctx context.Context, args ...string) ([]byte, error) {
	return m.R.Output(ctx, m.runtime(), args...)
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
		fmt.Fprintf(w, "  docker         : mode=%s composeFile=%s\n", cfg.Docker.Mode, m.composeFile())
	}
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
// actionable error before any deploy step runs. Under --dry-run it only echoes.
func (m *Manager) Reachable(ctx context.Context) error {
	if _, err := m.output(ctx, "version"); err != nil {
		return fmt.Errorf("cannot reach the %s runtime %q (is it installed and running?): %w", platformTitle(m.P), m.runtime(), err)
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
	return m.prepPSK()
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
// uses a systemd quadlet unit; docker uses compose (default) or a bare run.
func (m *Manager) Deploy(ctx context.Context, role config.Role) error {
	id := m.Cfg.ResolveNode(role)
	if m.P == config.Podman {
		return m.deployPodman(ctx, id)
	}
	return m.deployDocker(ctx, id)
}

func (m *Manager) deployPodman(ctx context.Context, id config.NodeIdentity) error {
	if err := m.checkPodmanEUID(); err != nil {
		return err
	}
	unit := filepath.ToSlash(filepath.Join(m.Cfg.Podman.QuadletDir, m.name()+".container"))
	if m.isDryRun() {
		m.logf("[Info] would write quadlet unit %s (skipped under --dry-run).", unit)
	} else {
		if err := os.MkdirAll(m.Cfg.Podman.QuadletDir, 0o755); err != nil {
			return fmt.Errorf("create quadlet dir %q: %w", m.Cfg.Podman.QuadletDir, err)
		}
		if err := os.WriteFile(unit, render.Quadlet(m.Cfg, id), 0o600); err != nil {
			return fmt.Errorf("write quadlet unit %q: %w", unit, err)
		}
		m.logf("[ OK ] wrote quadlet unit %s", unit)
	}
	if err := m.systemctl(ctx, "daemon-reload"); err != nil {
		return err
	}
	return m.systemctl(ctx, "start", m.name()+".service")
}

func (m *Manager) deployDocker(ctx context.Context, id config.NodeIdentity) error {
	if m.Cfg.Docker.Mode == "run" {
		return m.run(ctx, render.RunArgs(m.Cfg, id)...)
	}
	file := m.composeFile()
	if m.isDryRun() {
		m.logf("[Info] would write compose file %s (skipped under --dry-run).", file)
	} else {
		if err := os.WriteFile(file, render.Compose(m.Cfg, id), 0o600); err != nil {
			return fmt.Errorf("write compose file %q: %w", file, err)
		}
		m.logf("[ OK ] wrote compose file %s", file)
	}
	return m.run(ctx, "compose", "-f", file, "up", "-d")
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
	if m.Cfg.Docker.Mode == "run" {
		return m.stopAndRemove(ctx)
	}
	file := m.composeFile()
	if m.isDryRun() || fileExists(file) {
		return m.run(ctx, "compose", "-f", file, "down")
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
	if m.Cfg.Docker.Mode != "run" {
		file := m.composeFile()
		if m.isDryRun() || fileExists(file) {
			if err := m.run(ctx, "compose", "-f", file, "ps"); err != nil {
				m.logf("[WARN] compose ps failed: %v", err)
			}
		}
	}
	return m.run(ctx, "ps", "--all", "--filter", "name="+m.name())
}

// Logs streams the broker container's logs (`<runtime> logs -f <name>`).
func (m *Manager) Logs(ctx context.Context) error {
	return m.run(ctx, "logs", "-f", m.name())
}

// CLI opens an interactive Solace CLI session inside the container.
func (m *Manager) CLI(ctx context.Context) error {
	return m.R.RunInteractive(ctx, m.runtime(), "exec", "-it", m.name(), "cli", "-A")
}

// Shell opens an interactive shell inside the container.
func (m *Manager) Shell(ctx context.Context) error {
	return m.R.RunInteractive(ctx, m.runtime(), "exec", "-it", m.name(), "bash")
}

// --- helpers ----------------------------------------------------------------

// systemctl runs `systemctl [--user] args...` through the Runner, honoring the
// rootless (`--user`) vs rootful mode derived in config.
func (m *Manager) systemctl(ctx context.Context, args ...string) error {
	if u := m.Cfg.Podman.SystemctlUser; u != "" {
		args = append([]string{u}, args...)
	}
	return m.R.Run(ctx, "systemctl", args...)
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
