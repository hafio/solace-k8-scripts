package container

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"solace/internal/config"
	"solace/internal/engine"
	"solace/internal/render"
)

// These tests exercise the container host Manager over two runners: engine.Echo
// (dry-run) to assert the echoed command sequence portably (no euid/file/DNS side
// effects), and the capturing capRunner from transport_test.go to inspect exact
// argv and the real file-write paths that dry-run skips. capRunner/capCall/eqArgs
// live in transport_test.go (same package).

// ctrCfg builds a container config for platform p with a full node table. TLS is
// left unset (Check's "(not configured)" branch); tests that need it set it.
func ctrCfg(p config.Platform, redundancy string) *config.Config {
	c := &config.Config{Redundancy: redundancy}
	c.Image.Repo = "solace/solace-pubsub-standard"
	c.Image.Tag = "latest"
	c.Admin.User = "admin"
	c.Admin.Pass = "secret-pass"
	c.Nodes = config.Nodes{
		Primary: config.Node{Name: "pri-host", IP: "10.0.0.1"},
		Backup:  config.Node{Name: "bkp-host", IP: "10.0.0.2"},
		Monitor: config.Node{Name: "mon-host", IP: "10.0.0.3"},
	}
	// The container default scaling tier and the CPU it fixes. Scaling.CPU is
	// derived by ApplyDefaults, which this hand-built config deliberately skips
	// (executors must work without config.Load), so it is set alongside the tier
	// it comes from -- otherwise the rendered artifacts would carry no resource
	// caps and these tests would exercise the renderers' fail-safe branch instead
	// of the deploy path a real run takes.
	c.Scaling.MaxConnections = 1000
	c.Scaling.CPU = "2"
	switch p {
	case config.Podman:
		c.Podman.Runtime = config.Command{"podman"}
		c.Podman.Container.Name = "sol-pod"
		c.Podman.Container.RunUser = "1000:1000"
		c.Podman.Container.Mem = "6898m"
		c.Podman.Container.DataDir = "/opt/solace/data"
		c.Podman.QuadletDir = "/etc/containers/systemd"
		c.Podman.Network.Mode = "host"
	default:
		c.Docker.Runtime = config.Command{"docker"}
		c.Docker.Mode = "compose"
		c.Docker.Container.Name = "solace"
		c.Docker.Container.RunUser = "0:0"
		c.Docker.Container.Mem = "6898m"
		c.Docker.Container.DataDir = "/opt/solace/data"
		c.Docker.Network.Mode = "host"
	}
	return c
}

// newEchoMgr builds a Manager over engine.Echo, routing echoed commands, report
// output, and progress to one buffer. Resolve is stubbed (dry-run skips it).
func newEchoMgr(cfg *config.Config, p config.Platform) (*Manager, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	m := NewManager(engine.Echo{W: buf}, cfg, p, func(f string, a ...any) { fmt.Fprintf(buf, f+"\n", a...) }, buf)
	m.Resolve = func(string) bool { return true }
	return m, buf
}

// newCapMgr builds a Manager over the capturing capRunner (non-dry): file writes,
// euid checks, and DNS run for real, so Resolve defaults to always-resolves.
func newCapMgr(cfg *config.Config, p config.Platform) (*Manager, *capRunner, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	rr := &capRunner{}
	m := NewManager(rr, cfg, p, func(f string, a ...any) { fmt.Fprintf(buf, f+"\n", a...) }, buf)
	m.Resolve = func(string) bool { return true }
	return m, rr, buf
}

// hasCall reports whether rr captured a Run/RunInput call to name with exactly args.
func hasCall(rr *capRunner, name string, args []string) bool {
	for _, c := range rr.calls {
		if c.name == name && eqArgs(c.args, args) {
			return true
		}
	}
	return false
}

// assertMode checks the permission bits of a file the manager wrote. Skipped on
// Windows, whose filesystem carries no POSIX mode -- the check matters on the
// hosts that actually run a broker.
func assertMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	if runtime.GOOS == "windows" {
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Errorf("%s mode = %#o, want %#o", path, got, want)
	}
}

// withUser prepends the systemctl --user token when the config is rootless.
func withUser(cfg *config.Config, args ...string) []string {
	if cfg.Podman.SystemctlUser != "" {
		return append([]string{cfg.Podman.SystemctlUser}, args...)
	}
	return args
}

// --- Check ------------------------------------------------------------------

func TestManagerCheckDryRun(t *testing.T) {
	cases := []struct {
		p          config.Platform
		redundancy string
		title      string
		mode       string
	}{
		{config.Docker, "yes", "Docker", "HA redundancy group"},
		{config.Docker, "no", "Docker", "standalone (single broker)"},
		{config.Podman, "yes", "Podman", "HA redundancy group"},
		{config.Podman, "no", "Podman", "standalone (single broker)"},
	}
	for _, tc := range cases {
		cfg := ctrCfg(tc.p, tc.redundancy)
		if tc.redundancy == "yes" {
			cfg.TLS.Cert, cfg.TLS.CertKey = "server.pem", "server.key" // exercise the tls-configured branch
		}
		m, buf := newEchoMgr(cfg, tc.p)
		if err := m.Check(context.Background()); err != nil {
			t.Fatalf("%s/%s Check: %v", tc.p, tc.redundancy, err)
		}
		out := buf.String()
		for _, want := range []string{
			"Solace broker deployment (" + tc.title + ")",
			tc.mode,
			"+ " + cfg.ContainerRuntime(tc.p).String() + " version",
			"skipped (--dry-run)",
		} {
			if !strings.Contains(out, want) {
				t.Errorf("%s/%s Check missing %q:\n%s", tc.p, tc.redundancy, want, out)
			}
		}
	}
}

func TestManagerCheckDNSFailsLoudInHA(t *testing.T) {
	m, _, buf := newCapMgr(ctrCfg(config.Docker, "yes"), config.Docker)
	m.Resolve = func(host string) bool { return host != "bkp-host" }
	if err := m.Check(context.Background()); err == nil {
		t.Fatal("Check must fail when a redundancy hostname does not resolve")
	}
	if !strings.Contains(buf.String(), "does NOT resolve: bkp-host") {
		t.Errorf("Check should name the failing host:\n%s", buf.String())
	}
}

func TestManagerCheckStandaloneDNSWarnsOnly(t *testing.T) {
	m, _, _ := newCapMgr(ctrCfg(config.Docker, "no"), config.Docker)
	m.Resolve = func(string) bool { return false } // unresolved
	if err := m.Check(context.Background()); err != nil {
		t.Fatalf("standalone Check must not fail on an unresolved name: %v", err)
	}
}

// --- PrepHost ---------------------------------------------------------------

func TestManagerPrepHostDryRunDoesNotWritePSK(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, "sample.yaml")
	original := "redundancy: yes\nnodes:\n  primary:\n    name: pri-host\n  psk:\n"
	if err := os.WriteFile(envFile, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	m, buf := newEchoMgr(ctrCfg(config.Docker, "yes"), config.Docker)
	m.EnvPath = envFile
	m.GenPSK = func() (string, error) { t.Fatal("GenPSK must not run under --dry-run"); return "", nil }
	if err := m.PrepHost(context.Background()); err != nil {
		t.Fatalf("PrepHost: %v", err)
	}
	if got, _ := os.ReadFile(envFile); string(got) != original {
		t.Errorf("env file must be unchanged under --dry-run:\n%s", got)
	}
	out := buf.String()
	for _, want := range []string{"+ mkdir -p /opt/solace/data", "+ chown 0:0 /opt/solace/data"} {
		if !strings.Contains(out, want) {
			t.Errorf("PrepHost should echo %q:\n%s", want, out)
		}
	}
}

func TestManagerPrepHostWritesPSK(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, "sample.yaml")
	original := "redundancy: yes\nreplication:\n  psk: KEEP-ME\nnodes:\n  primary:\n    name: pri-host\n  psk:\n"
	if err := os.WriteFile(envFile, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	m, rr, _ := newCapMgr(ctrCfg(config.Docker, "yes"), config.Docker)
	m.EnvPath = envFile
	m.GenPSK = func() (string, error) { return "TESTPSK123", nil }
	if err := m.PrepHost(context.Background()); err != nil {
		t.Fatalf("PrepHost: %v", err)
	}
	got, _ := os.ReadFile(envFile)
	if !strings.Contains(string(got), `psk: "TESTPSK123"`) {
		t.Errorf("PrepHost should write the generated PSK into nodes:\n%s", got)
	}
	if !strings.Contains(string(got), "psk: KEEP-ME") {
		t.Errorf("PrepHost must not touch the replication psk:\n%s", got)
	}
	if !hasCall(rr, "mkdir", []string{"-p", "/opt/solace/data"}) {
		t.Errorf("PrepHost should mkdir the data dir:\n%+v", rr.calls)
	}
	if !hasCall(rr, "chown", []string{"0:0", "/opt/solace/data"}) {
		t.Errorf("PrepHost should chown the data dir:\n%+v", rr.calls)
	}
}

func TestManagerPrepHostRootlessUsesUnshareChown(t *testing.T) {
	cfg := ctrCfg(config.Podman, "no") // standalone -> PSK step is skipped
	cfg.Podman.Rootless = true
	m, rr, _ := newCapMgr(cfg, config.Podman)
	if err := m.PrepHost(context.Background()); err != nil {
		t.Fatalf("PrepHost: %v", err)
	}
	if !hasCall(rr, "podman", []string{"unshare", "chown", "1000:1000", "/opt/solace/data"}) {
		t.Errorf("rootless PrepHost should chown via `podman unshare`:\n%+v", rr.calls)
	}
}

// --- rootless nofile --------------------------------------------------------

func setNoFile(cfg *config.Config, p config.Platform, v string) {
	if p == config.Podman {
		cfg.Podman.Container.Ulimits.NoFile = v
		return
	}
	cfg.Docker.Container.Ulimits.NoFile = v
}

// rootlessNoFileMgr builds a rootless podman Manager whose `ulimit -Hn` probe
// answers hardLimit. capRunner returns one canned stdout for every Output call,
// which `podman info` also receives -- harmless, since Preflight reads only its
// error.
func rootlessNoFileMgr(want, hardLimit string) (*Manager, *capRunner, *bytes.Buffer) {
	cfg := ctrCfg(config.Podman, "no") // standalone -> the PSK step is skipped
	cfg.Podman.Rootless = true
	setNoFile(cfg, config.Podman, want)
	m, rr, buf := newCapMgr(cfg, config.Podman)
	rr.out = []byte(hardLimit)
	return m, rr, buf
}

func TestPrepHostRootlessNoFileSufficient(t *testing.T) {
	m, rr, buf := rootlessNoFileMgr("2448:1048576", "1048576\n")
	if err := m.PrepHost(context.Background()); err != nil {
		t.Fatalf("PrepHost: %v", err)
	}
	if !hasCall(rr, "sh", []string{"-c", "ulimit -Hn"}) {
		t.Errorf("rootless prep should probe this user's hard nofile limit:\n%+v", rr.calls)
	}
	if !strings.Contains(buf.String(), "hard nofile limit: 1048576") {
		t.Errorf("prep should report the limit it found:\n%s", buf)
	}
}

// TestPrepHostRootlessNoFileTooLow is the point of the check: a rootless
// container cannot raise nofile past the user's hard limit, so prep stops before
// deploying a broker that would silently run under-provisioned.
func TestPrepHostRootlessNoFileTooLow(t *testing.T) {
	m, _, _ := rootlessNoFileMgr("2448:1048576", "1024\n")
	err := m.PrepHost(context.Background())
	if err == nil {
		t.Fatal("a hard limit below the configured one must fail prep")
	}
	// Both numbers and the exact remedy, so the message is actionable on its own.
	for _, want := range []string{"1024", "1048576", "2448", "limits.d", "hard nofile", "soft nofile", "log out"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("nofile error should mention %q, got: %v", want, err)
		}
	}
}

func TestPrepHostRootlessNoFileUnlimited(t *testing.T) {
	m, _, buf := rootlessNoFileMgr("2448:1048576", "unlimited\n")
	if err := m.PrepHost(context.Background()); err != nil {
		t.Fatalf("an unlimited hard limit satisfies any request: %v", err)
	}
	if !strings.Contains(buf.String(), "unlimited") {
		t.Errorf("prep should report the unlimited case:\n%s", buf)
	}
}

func TestPrepHostRootlessNoFileUnreadable(t *testing.T) {
	m, _, _ := rootlessNoFileMgr("2448:1048576", "not-a-number\n")
	if err := m.PrepHost(context.Background()); err == nil || !strings.Contains(err.Error(), "cannot parse") {
		t.Errorf("an unreadable limit must fail loud rather than be assumed fine, got: %v", err)
	}
}

// TestPrepHostRootlessNoFileUnsetSkips covers the hand-built config the executors
// are handed: with no configured limit there is nothing to assert against, so the
// probe never runs.
func TestPrepHostRootlessNoFileUnsetSkips(t *testing.T) {
	m, rr, _ := rootlessNoFileMgr("", "1024\n")
	if err := m.PrepHost(context.Background()); err != nil {
		t.Fatalf("PrepHost: %v", err)
	}
	if hasCall(rr, "sh", []string{"-c", "ulimit -Hn"}) {
		t.Errorf("no configured nofile means no probe:\n%+v", rr.calls)
	}
}

// TestPrepHostRootfulSkipsNoFile: a privileged engine raises the limit itself, so
// the user's own hard limit does not bound the container.
func TestPrepHostRootfulSkipsNoFile(t *testing.T) {
	for _, p := range []config.Platform{config.Podman, config.Docker} {
		cfg := ctrCfg(p, "no")
		cfg.Podman.Rootless = false
		setNoFile(cfg, p, "2448:1048576")
		m, rr, _ := newCapMgr(cfg, p)
		rr.out = []byte("1024\n") // far below, and deliberately not consulted
		if err := m.PrepHost(context.Background()); err != nil {
			t.Fatalf("%s: PrepHost: %v", p, err)
		}
		if hasCall(rr, "sh", []string{"-c", "ulimit -Hn"}) {
			t.Errorf("%s: only rootless podman is bounded by the user's limit:\n%+v", p, rr.calls)
		}
	}
}

func TestPrepHostRootlessNoFileDryRun(t *testing.T) {
	cfg := ctrCfg(config.Podman, "no")
	cfg.Podman.Rootless = true
	setNoFile(cfg, config.Podman, "2448:1048576")
	m, buf := newEchoMgr(cfg, config.Podman)
	if err := m.PrepHost(context.Background()); err != nil {
		t.Fatalf("PrepHost: %v", err)
	}
	// The Echo runner answers nothing, so the probe is shown and the assertion
	// skipped -- the same shape Preflight uses.
	out := buf.String()
	if !strings.Contains(out, "ulimit -Hn") {
		t.Errorf("dry-run should echo the probe:\n%s", out)
	}
	if !strings.Contains(out, "nofile         : skipped (--dry-run)") {
		t.Errorf("dry-run should say the assertion was skipped:\n%s", out)
	}
}

func TestSplitLimit(t *testing.T) {
	for _, tc := range []struct {
		in         string
		soft, hard int
	}{
		{"2448:1048576", 2448, 1048576},
		{"1024", 1024, 1024},
		{"-1", 0, 0}, // unlimited: nothing to assert against
		{"-1:-1", 0, 0},
		{"", 0, 0},
		{"bogus", 0, 0},
		{" 2448 : 1048576 ", 2448, 1048576},
	} {
		soft, hard := splitLimit(tc.in)
		if soft != tc.soft || hard != tc.hard {
			t.Errorf("splitLimit(%q) = %d,%d want %d,%d", tc.in, soft, hard, tc.soft, tc.hard)
		}
	}
}

// --- Deploy -----------------------------------------------------------------

func TestManagerDeployDockerComposeWritesFile(t *testing.T) {
	dir := t.TempDir()
	cfg := ctrCfg(config.Docker, "no")
	cfg.Docker.ComposeFile = filepath.Join(dir, "compose.yml")
	m, rr, _ := newCapMgr(cfg, config.Docker)
	if err := m.Deploy(context.Background(), config.Primary); err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	if !fileExists(cfg.Docker.ComposeFile) {
		t.Error("Deploy should write the compose file")
	}
	// --force-recreate is part of the create path: a stopped container would
	// otherwise be started with the credentials it was created with.
	if !hasCall(rr, "docker", []string{"compose", "-f", cfg.Docker.ComposeFile, "up", "-d", "--force-recreate"}) {
		t.Errorf("Deploy should compose up:\n%+v", rr.calls)
	}
}

// TestManagerDockerComposeCommandOverride covers the standalone-binary form: a
// host without the compose plugin sets docker.compose, and every compose call has
// to go through it rather than the runtime.
func TestManagerDockerComposeCommandOverride(t *testing.T) {
	dir := t.TempDir()
	cfg := ctrCfg(config.Docker, "no")
	cfg.Docker.ComposeFile = filepath.Join(dir, "compose.yml")
	cfg.Docker.Compose = config.Command{"docker-compose"}
	m, rr, _ := newCapMgr(cfg, config.Docker)
	if err := m.Deploy(context.Background(), config.Primary); err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	if !hasCall(rr, "docker-compose", []string{"-f", cfg.Docker.ComposeFile, "up", "-d", "--force-recreate"}) {
		t.Errorf("Deploy should use the configured compose command:\n%+v", rr.calls)
	}
}

// TestManagerDockerCheckProbesCompose pins the compose probe: the plugin is a
// separate install from the engine, so a reachable docker with no compose must
// fail at check time rather than at deploy time.
func TestManagerDockerCheckProbesCompose(t *testing.T) {
	m, buf := newEchoMgr(ctrCfg(config.Docker, "no"), config.Docker)
	if err := m.Check(context.Background()); err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !strings.Contains(buf.String(), "+ docker compose version") {
		t.Errorf("docker Check should probe the compose command:\n%s", buf.String())
	}
}

func TestManagerDockerCheckFailsWhenComposeMissing(t *testing.T) {
	m, rr, _ := newCapMgr(ctrCfg(config.Docker, "no"), config.Docker)
	// Fail only the compose probe, so the engine still looks reachable and the
	// error has to be the compose-specific one.
	rr.outFail = failOn("compose")
	err := m.Check(context.Background())
	if err == nil {
		t.Fatal("Check must fail when the compose command cannot run")
	}
	if !strings.Contains(err.Error(), "docker.compose") {
		t.Errorf("error should point at the docker.compose override, got: %v", err)
	}
}

// --- describe / copy parity -------------------------------------------------

// TestManagerDescribe covers the container analog of `kubectl describe pod`:
// `<runtime> inspect` on both platforms, plus the installed unit on podman (the
// unit's summary is already in Status, so `cat` answers a different question).
func TestManagerDescribe(t *testing.T) {
	t.Run("docker", func(t *testing.T) {
		m, rr, _ := newCapMgr(ctrCfg(config.Docker, "no"), config.Docker)
		if err := m.Describe(context.Background()); err != nil {
			t.Fatalf("Describe: %v", err)
		}
		if !hasCall(rr, "docker", []string{"inspect", "solace"}) {
			t.Errorf("Describe should inspect the container:\n%+v", rr.calls)
		}
	})
	t.Run("podman also shows the unit", func(t *testing.T) {
		cfg := ctrCfg(config.Podman, "no")
		m, rr, _ := newCapMgr(cfg, config.Podman)
		if err := m.Describe(context.Background()); err != nil {
			t.Fatalf("Describe: %v", err)
		}
		if !hasCall(rr, "systemctl", withUser(cfg, "cat", "sol-pod.service")) {
			t.Errorf("podman Describe should show the installed unit:\n%+v", rr.calls)
		}
		if !hasCall(rr, "podman", []string{"inspect", "sol-pod"}) {
			t.Errorf("podman Describe should inspect the container:\n%+v", rr.calls)
		}
	})
	t.Run("a missing unit is tolerated", func(t *testing.T) {
		m, rr, buf := newCapMgr(ctrCfg(config.Podman, "no"), config.Podman)
		rr.fail = failOn("cat")
		if err := m.Describe(context.Background()); err != nil {
			t.Fatalf("Describe should tolerate an uninstalled unit: %v", err)
		}
		if !strings.Contains(buf.String(), "systemctl cat") {
			t.Errorf("Describe should warn about the missing unit:\n%s", buf.String())
		}
	})
}

// TestManagerCopy covers the copy verbs the container tree previously lacked. The
// per-file reporting and the non-zero exit on any failure mirror the k8s verbs.
func TestManagerCopy(t *testing.T) {
	t.Run("from", func(t *testing.T) {
		m, rr, buf := newCapMgr(ctrCfg(config.Docker, "no"), config.Docker)
		if err := m.CopyFrom(context.Background(), []string{"/var/lib/solace/jail/logs/debug.log"}); err != nil {
			t.Fatalf("CopyFrom: %v", err)
		}
		if !hasCall(rr, "docker", []string{"cp", "solace:/var/lib/solace/jail/logs/debug.log", "debug.log"}) {
			t.Errorf("CopyFrom should cp out of the container:\n%+v", rr.calls)
		}
		if !strings.Contains(buf.String(), "[ OK ]") {
			t.Errorf("CopyFrom should report each file:\n%s", buf.String())
		}
	})
	t.Run("into", func(t *testing.T) {
		m, rr, _ := newCapMgr(ctrCfg(config.Podman, "no"), config.Podman)
		if err := m.CopyInto(context.Background(), []string{"setup.cli"}, "/tmp"); err != nil {
			t.Fatalf("CopyInto: %v", err)
		}
		if !hasCall(rr, "podman", []string{"cp", "setup.cli", "sol-pod:/tmp"}) {
			t.Errorf("CopyInto should cp into the container:\n%+v", rr.calls)
		}
	})
	t.Run("no files is an error", func(t *testing.T) {
		m, _, _ := newCapMgr(ctrCfg(config.Docker, "no"), config.Docker)
		if err := m.CopyFrom(context.Background(), nil); err == nil {
			t.Error("CopyFrom with no files should error")
		}
		if err := m.CopyInto(context.Background(), nil, ""); err == nil {
			t.Error("CopyInto with no files should error")
		}
	})
	t.Run("a failed file makes the command fail", func(t *testing.T) {
		m, rr, buf := newCapMgr(ctrCfg(config.Docker, "no"), config.Docker)
		rr.fail = failOn("cp")
		if err := m.CopyFrom(context.Background(), []string{"a.log", "b.log"}); err == nil {
			t.Error("CopyFrom should fail when a file could not be copied")
		}
		if !strings.Contains(buf.String(), "[ERROR]") {
			t.Errorf("CopyFrom should report the failing file:\n%s", buf.String())
		}
	})
}

// --- registry login ---------------------------------------------------------

// TestManagerPrepHostRegistryLogin pins the credentials that used to be silently
// ignored on containers: prep now logs in, with the password on stdin so it never
// reaches an argv or the dry-run echo.
func TestManagerPrepHostRegistryLogin(t *testing.T) {
	cfg := ctrCfg(config.Docker, "no")
	cfg.Image.Registry = "registry.example.com"
	cfg.Image.User = "repo-user"
	cfg.Image.Pass = "repo-pass"
	m, rr, _ := newCapMgr(cfg, config.Docker)
	if err := m.PrepHost(context.Background()); err != nil {
		t.Fatalf("PrepHost: %v", err)
	}
	want := []string{"login", "--username", "repo-user", "--password-stdin", "registry.example.com"}
	if !hasCall(rr, "docker", want) {
		t.Errorf("PrepHost should log in to the registry:\n%+v", rr.calls)
	}
	for _, c := range rr.calls {
		for _, a := range c.args {
			if strings.Contains(a, "repo-pass") {
				t.Errorf("the registry password must not reach an argv: %s %v", c.name, c.args)
			}
		}
	}
}

func TestManagerPrepHostNoLoginWithoutCreds(t *testing.T) {
	m, rr, _ := newCapMgr(ctrCfg(config.Podman, "no"), config.Podman)
	if err := m.PrepHost(context.Background()); err != nil {
		t.Fatalf("PrepHost: %v", err)
	}
	for _, c := range rr.calls {
		if len(c.args) > 0 && c.args[0] == "login" {
			t.Errorf("no credentials means no login attempt:\n%+v", rr.calls)
		}
	}
}

func TestManagerPrepHostRejectsHalfCredentials(t *testing.T) {
	cfg := ctrCfg(config.Docker, "no")
	cfg.Image.User = "repo-user" // pass left empty
	m, _, _ := newCapMgr(cfg, config.Docker)
	err := m.PrepHost(context.Background())
	if err == nil || !strings.Contains(err.Error(), "image.user and image.pass") {
		t.Fatalf("PrepHost err = %v, want the both-or-neither credentials error", err)
	}
}

// --- redeploy: compare, then restart only with consent ----------------------

// TestManagerRedeployUnchangedIsNoOp covers the both-platforms "nothing to do"
// arm: a re-deploy that renders the same artifact against a running broker must
// not touch it at all.
func TestManagerRedeployUnchangedIsNoOp(t *testing.T) {
	t.Run("podman", func(t *testing.T) {
		dir := t.TempDir()
		cfg := ctrCfg(config.Podman, "no")
		cfg.Podman.QuadletDir = dir
		m, rr, buf := newCapMgr(cfg, config.Podman)
		m.Geteuid = func() int { return -1 }
		rr.out = []byte("active\n") // the unit is already running
		if err := os.WriteFile(filepath.Join(dir, "sol-pod.container"),
			render.Quadlet(cfg, cfg.ResolveNode(config.Primary)), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := m.Deploy(context.Background(), config.Primary); err != nil {
			t.Fatalf("Deploy: %v", err)
		}
		if hasCall(rr, "systemctl", withUser(cfg, "restart", "sol-pod.service")) {
			t.Errorf("an unchanged unit must not bounce the broker:\n%+v", rr.calls)
		}
		if hasCall(rr, "systemctl", withUser(cfg, "start", "sol-pod.service")) {
			t.Errorf("an active unit needs no start:\n%+v", rr.calls)
		}
		if !strings.Contains(buf.String(), "nothing to do") {
			t.Errorf("Deploy should report there was nothing to do:\n%s", buf.String())
		}
	})
	t.Run("docker", func(t *testing.T) {
		dir := t.TempDir()
		cfg := ctrCfg(config.Docker, "no")
		cfg.Docker.ComposeFile = filepath.Join(dir, "compose.yml")
		if err := os.WriteFile(cfg.Docker.ComposeFile,
			render.Compose(cfg, cfg.ResolveNode(config.Primary)), 0o600); err != nil {
			t.Fatal(err)
		}
		m, rr, buf := newCapMgr(cfg, config.Docker)
		rr.out = []byte("solace\n") // ps lists this container by name -> running
		if err := m.Deploy(context.Background(), config.Primary); err != nil {
			t.Fatalf("Deploy: %v", err)
		}
		for _, c := range rr.calls {
			if containsStr(c.args, "up") {
				t.Errorf("an unchanged compose file must not recreate the container:\n%+v", rr.calls)
			}
		}
		if !strings.Contains(buf.String(), "nothing to do") {
			t.Errorf("Deploy should report there was nothing to do:\n%s", buf.String())
		}
	})
}

// TestManagerRedeployChangedNeedsConsent is the core of the upgrade fix: a
// changed artifact against a running broker must not bounce it silently, and
// --restart is what applies it. Podman's old behaviour was worse than silent --
// `systemctl start` on an active unit is a no-op, so the broker kept the old
// image while the command reported success.
func TestManagerRedeployChangedNeedsConsent(t *testing.T) {
	setup := func(t *testing.T, p config.Platform) (*config.Config, string) {
		t.Helper()
		dir := t.TempDir()
		cfg := ctrCfg(p, "no")
		if p == config.Podman {
			cfg.Podman.QuadletDir = dir
			// An artifact from an older image tag: the on-disk one differs from what
			// this config now renders.
			old := ctrCfg(p, "no")
			old.Image.Tag = "previous"
			if err := os.WriteFile(filepath.Join(dir, "sol-pod.container"),
				render.Quadlet(old, old.ResolveNode(config.Primary)), 0o600); err != nil {
				t.Fatal(err)
			}
			return cfg, "sol-pod.service"
		}
		cfg.Docker.ComposeFile = filepath.Join(dir, "compose.yml")
		old := ctrCfg(p, "no")
		old.Image.Tag = "previous"
		old.Docker.ComposeFile = cfg.Docker.ComposeFile
		if err := os.WriteFile(cfg.Docker.ComposeFile,
			render.Compose(old, old.ResolveNode(config.Primary)), 0o600); err != nil {
			t.Fatal(err)
		}
		return cfg, ""
	}

	t.Run("podman declines without consent", func(t *testing.T) {
		cfg, svc := setup(t, config.Podman)
		m, rr, buf := newCapMgr(cfg, config.Podman)
		m.Geteuid = func() int { return -1 }
		rr.out = []byte("active\n")
		if err := m.Deploy(context.Background(), config.Primary); err != nil {
			t.Fatalf("Deploy: %v", err)
		}
		if hasCall(rr, "systemctl", withUser(cfg, "restart", svc)) {
			t.Errorf("no consent means no bounce:\n%+v", rr.calls)
		}
		if !strings.Contains(buf.String(), "still uses the previous one") {
			t.Errorf("Deploy should warn the running broker is now stale:\n%s", buf.String())
		}
		if !strings.Contains(buf.String(), "--restart") {
			t.Errorf("the warning should name the flag that finishes the job:\n%s", buf.String())
		}
	})
	t.Run("podman restarts with --restart", func(t *testing.T) {
		cfg, svc := setup(t, config.Podman)
		m, rr, _ := newCapMgr(cfg, config.Podman)
		m.Geteuid = func() int { return -1 }
		m.Restart = true
		rr.out = []byte("active\n")
		if err := m.Deploy(context.Background(), config.Primary); err != nil {
			t.Fatalf("Deploy: %v", err)
		}
		if !hasCall(rr, "systemctl", withUser(cfg, "restart", svc)) {
			t.Errorf("--restart should restart the active unit:\n%+v", rr.calls)
		}
	})
	t.Run("podman restarts when the prompt is accepted", func(t *testing.T) {
		cfg, svc := setup(t, config.Podman)
		m, rr, _ := newCapMgr(cfg, config.Podman)
		m.Geteuid = func() int { return -1 }
		asked := ""
		m.Confirm = func(q string) bool { asked = q; return true }
		rr.out = []byte("active\n")
		if err := m.Deploy(context.Background(), config.Primary); err != nil {
			t.Fatalf("Deploy: %v", err)
		}
		if !strings.Contains(asked, svc) {
			t.Errorf("the prompt should name what gets restarted, got %q", asked)
		}
		if !hasCall(rr, "systemctl", withUser(cfg, "restart", svc)) {
			t.Errorf("an accepted prompt should restart:\n%+v", rr.calls)
		}
	})
	t.Run("docker declines without consent", func(t *testing.T) {
		cfg, _ := setup(t, config.Docker)
		m, rr, buf := newCapMgr(cfg, config.Docker)
		rr.out = []byte("solace\n")
		if err := m.Deploy(context.Background(), config.Primary); err != nil {
			t.Fatalf("Deploy: %v", err)
		}
		for _, c := range rr.calls {
			if containsStr(c.args, "up") {
				t.Errorf("no consent means no recreate:\n%+v", rr.calls)
			}
		}
		if !strings.Contains(buf.String(), "still uses the previous one") {
			t.Errorf("Deploy should warn the running broker is now stale:\n%s", buf.String())
		}
	})
	t.Run("docker recreates with --restart", func(t *testing.T) {
		cfg, _ := setup(t, config.Docker)
		m, rr, _ := newCapMgr(cfg, config.Docker)
		m.Restart = true
		rr.out = []byte("solace\n")
		if err := m.Deploy(context.Background(), config.Primary); err != nil {
			t.Fatalf("Deploy: %v", err)
		}
		if !hasCall(rr, "docker", []string{"compose", "-f", cfg.Docker.ComposeFile, "up", "-d"}) {
			t.Errorf("--restart should recreate through compose:\n%+v", rr.calls)
		}
	})
}

// --- secret externalization -------------------------------------------------

// TestManagerDeployDockerPassesSecretsAsEnv is the docker half of the secret
// model: nothing is written to this host, the values reach compose only through
// the child process environment, and the artifact carries variable names.
func TestManagerDeployDockerPassesSecretsAsEnv(t *testing.T) {
	dir := t.TempDir()
	cfg := ctrCfg(config.Docker, "yes")
	cfg.Nodes.PSK = "test-psk"
	cfg.Docker.ComposeFile = filepath.Join(dir, "compose.yml")
	m, rr, _ := newCapMgr(cfg, config.Docker)
	if err := m.Deploy(context.Background(), config.Primary); err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	if fileExists(filepath.Join(dir, "solace-secrets")) {
		t.Error("no secret file may be written any more; the values ride the compose process environment")
	}
	var up *capCall
	for i := range rr.calls {
		c := &rr.calls[i]
		if containsStr(c.args, "up") {
			up = c
		}
	}
	if up == nil {
		t.Fatalf("Deploy should run `compose up -d`:\n%+v", rr.calls)
	}
	if up.method != "RunEnv" {
		t.Errorf("compose must run through RunEnv so the values never reach an argv, got %s", up.method)
	}
	for _, want := range []string{"SOLACE_ADMIN_PASSWORD=secret-pass", "SOLACE_REDUNDANCY_PSK=test-psk"} {
		if !containsStr(up.env, want) {
			t.Errorf("compose environment should carry %q, got %v", want, maskedKeys(up.env))
		}
	}
	// The values may reach the environment and nothing else (§3).
	for _, c := range rr.calls {
		for _, a := range c.args {
			for _, leak := range []string{"secret-pass", "test-psk"} {
				if strings.Contains(a, leak) {
					t.Errorf("secret value reached an argv: %s %v", c.name, c.args)
				}
			}
		}
	}
	body, err := os.ReadFile(cfg.Docker.ComposeFile)
	if err != nil {
		t.Fatal(err)
	}
	assertMode(t, cfg.Docker.ComposeFile, 0o600)
	for _, leak := range []string{"secret-pass", "test-psk"} {
		if strings.Contains(string(body), leak) {
			t.Errorf("compose file must reference secrets, not carry %q:\n%s", leak, body)
		}
	}
	for _, want := range []string{
		"source: solace-admin-password",
		"target: username_admin_password",
		"environment: SOLACE_ADMIN_PASSWORD",
		"username_admin_passwordfilepath: \"/run/secrets/username_admin_password\"",
	} {
		if !strings.Contains(string(body), want) {
			t.Errorf("compose file should contain %q:\n%s", want, body)
		}
	}
}

// containsStr reports whether list holds want exactly.
func containsStr(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

// maskedKeys renders an environment list for a failure message without its
// values -- a test diagnostic must not print a secret either.
func maskedKeys(env []string) string { return engine.MaskEnv(env) }

func TestManagerDeployPodmanCreatesSecrets(t *testing.T) {
	cfg := ctrCfg(config.Podman, "yes")
	cfg.Nodes.PSK = "test-psk"
	cfg.Podman.QuadletDir = t.TempDir()
	m, rr, _ := newCapMgr(cfg, config.Podman)
	m.Geteuid = func() int { return -1 }
	if err := m.Deploy(context.Background(), config.Primary); err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	// Names carry the container name, so two brokers on one host cannot overwrite
	// each other's entries in the shared podman store.
	for _, name := range []string{"sol-pod-admin-password", "sol-pod-redundancy-psk"} {
		if !hasCall(rr, "podman", []string{"secret", "create", "--replace", name, "-"}) {
			t.Errorf("Deploy should create podman secret %s:\n%+v", name, rr.calls)
		}
	}
	// The values ride stdin, so they must never appear in an argv (§3).
	for _, c := range rr.calls {
		for _, a := range c.args {
			for _, leak := range []string{"secret-pass", "test-psk"} {
				if strings.Contains(a, leak) {
					t.Errorf("secret value reached an argv: %s %v", c.name, c.args)
				}
			}
		}
	}
	assertMode(t, filepath.Join(cfg.Podman.QuadletDir, "sol-pod.container"), 0o600)
}

func TestManagerDeployRejectsEmptySecret(t *testing.T) {
	cfg := ctrCfg(config.Docker, "yes") // HA -> the PSK secret is required too
	cfg.Nodes.PSK = ""
	cfg.Docker.ComposeFile = filepath.Join(t.TempDir(), "compose.yml")
	m, _, _ := newCapMgr(cfg, config.Docker)
	err := m.Deploy(context.Background(), config.Primary)
	if err == nil {
		t.Fatal("Deploy must fail loud when a required secret is empty")
	}
	if !strings.Contains(err.Error(), "nodes.psk") || !strings.Contains(err.Error(), "prep host") {
		t.Errorf("error should name the field and the fix, got: %v", err)
	}
}

// TestManagerDeployDockerDryRunMasksSecretEnv covers the dry-run contract for the
// one path that carries values: the echo names the variables compose would be
// given and prints none of their values.
func TestManagerDeployDockerDryRunMasksSecretEnv(t *testing.T) {
	dir := t.TempDir()
	cfg := ctrCfg(config.Docker, "yes")
	cfg.Nodes.PSK = "" // empty is fine here: prep host has not generated it yet
	cfg.Docker.ComposeFile = filepath.Join(dir, "compose.yml")
	m, buf := newEchoMgr(cfg, config.Docker)
	if err := m.Deploy(context.Background(), config.Primary); err != nil {
		t.Fatalf("dry-run Deploy must stay previewable before prep host: %v", err)
	}
	if fileExists(filepath.Join(dir, "solace-secrets")) {
		t.Error("dry-run must not create anything, least of all a secrets dir")
	}
	out := buf.String()
	if !strings.Contains(out, "SOLACE_ADMIN_PASSWORD=***") {
		t.Errorf("dry-run should name the secret variables with masked values:\n%s", out)
	}
	if strings.Contains(out, "secret-pass") {
		t.Errorf("dry-run echo leaked the admin password:\n%s", out)
	}
}

func TestManagerDeployPodmanDryRunHidesSecretBytes(t *testing.T) {
	cfg := ctrCfg(config.Podman, "yes")
	cfg.Nodes.PSK = "test-psk"
	cfg.Podman.QuadletDir = t.TempDir()
	m, buf := newEchoMgr(cfg, config.Podman)
	if err := m.Deploy(context.Background(), config.Primary); err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "secret create --replace sol-pod-admin-password -") {
		t.Errorf("dry-run should echo the secret-create command:\n%s", out)
	}
	if !strings.Contains(out, "bytes on stdin") {
		t.Errorf("dry-run should report the value as stdin bytes, not print it:\n%s", out)
	}
	for _, leak := range []string{"secret-pass", "test-psk"} {
		if strings.Contains(out, leak) {
			t.Errorf("dry-run echo leaked %q:\n%s", leak, out)
		}
	}
}

func TestManagerDeployPodmanWritesUnit(t *testing.T) {
	dir := t.TempDir()
	cfg := ctrCfg(config.Podman, "no")
	cfg.Podman.QuadletDir = dir
	// Match this process's euid so the rootless/rootful guard passes on any host
	// (on Windows Geteuid()<0 and the guard is skipped entirely).
	if os.Geteuid() != 0 {
		cfg.Podman.Rootless = true
		cfg.Podman.SystemctlUser = "--user"
	}
	m, rr, _ := newCapMgr(cfg, config.Podman)
	if err := m.Deploy(context.Background(), config.Primary); err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	if !fileExists(filepath.Join(dir, "sol-pod.container")) {
		t.Error("Deploy should write the quadlet unit")
	}
	if !hasCall(rr, "systemctl", withUser(cfg, "daemon-reload")) {
		t.Errorf("Deploy should daemon-reload:\n%+v", rr.calls)
	}
	if !hasCall(rr, "systemctl", withUser(cfg, "start", "sol-pod.service")) {
		t.Errorf("Deploy should start the service:\n%+v", rr.calls)
	}
}

func TestManagerDeployPodmanDryRunSkipsWrite(t *testing.T) {
	dir := t.TempDir()
	cfg := ctrCfg(config.Podman, "no")
	cfg.Podman.QuadletDir = dir
	m, buf := newEchoMgr(cfg, config.Podman)
	if err := m.Deploy(context.Background(), config.Primary); err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	if fileExists(filepath.Join(dir, "sol-pod.container")) {
		t.Error("Deploy must not write the quadlet unit under --dry-run")
	}
	out := buf.String()
	for _, want := range []string{"+ systemctl daemon-reload", "+ systemctl start sol-pod.service"} {
		if !strings.Contains(out, want) {
			t.Errorf("dry-run Deploy should echo %q:\n%s", want, out)
		}
	}
}

func TestManagerPodmanEUIDGuardSkippedOnDryRun(t *testing.T) {
	m, _ := newEchoMgr(ctrCfg(config.Podman, "no"), config.Podman)
	m.Cfg.Podman.Rootless = false // rootful would require root if the guard ran
	if err := m.checkPodmanEUID(); err != nil {
		t.Errorf("euid guard must be skipped under --dry-run, got %v", err)
	}
}

// --- Delete -----------------------------------------------------------------

func TestManagerDeletePodmanRemovesUnit(t *testing.T) {
	dir := t.TempDir()
	cfg := ctrCfg(config.Podman, "no")
	cfg.Podman.QuadletDir = dir
	unit := filepath.Join(dir, "sol-pod.container")
	if err := os.WriteFile(unit, []byte("[Unit]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	m, rr, _ := newCapMgr(cfg, config.Podman)
	if err := m.Delete(context.Background(), false); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if fileExists(unit) {
		t.Error("Delete should remove the quadlet unit")
	}
	if !hasCall(rr, "systemctl", []string{"stop", "sol-pod.service"}) {
		t.Errorf("Delete should stop the service:\n%+v", rr.calls)
	}
	if !hasCall(rr, "systemctl", []string{"daemon-reload"}) {
		t.Errorf("Delete should daemon-reload:\n%+v", rr.calls)
	}
}

func TestManagerDeletePodmanPurgeRootless(t *testing.T) {
	cfg := ctrCfg(config.Podman, "no")
	cfg.Podman.Rootless = true
	cfg.Podman.QuadletDir = t.TempDir()
	m, buf := newEchoMgr(cfg, config.Podman)
	if err := m.Delete(context.Background(), true); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if !strings.Contains(buf.String(), "+ podman unshare rm -rf /opt/solace/data") {
		t.Errorf("rootless purge should rm via `podman unshare`:\n%s", buf.String())
	}
}

func TestManagerDeleteDockerComposeDownWhenFileExists(t *testing.T) {
	dir := t.TempDir()
	cfg := ctrCfg(config.Docker, "no")
	cfg.Docker.ComposeFile = filepath.Join(dir, "compose.yml")
	if err := os.WriteFile(cfg.Docker.ComposeFile, []byte("services:\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	m, rr, _ := newCapMgr(cfg, config.Docker)
	if err := m.Delete(context.Background(), false); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if !hasCall(rr, "docker", []string{"compose", "-f", cfg.Docker.ComposeFile, "down"}) {
		t.Errorf("Delete should compose down when the file exists:\n%+v", rr.calls)
	}
}

func TestManagerDeleteDockerPurgeRemovesDataDir(t *testing.T) {
	dir := t.TempDir()
	cfg := ctrCfg(config.Docker, "no")
	cfg.Docker.ComposeFile = filepath.Join(dir, "compose.yml")
	if err := os.WriteFile(cfg.Docker.ComposeFile, []byte("services:\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	m, rr, _ := newCapMgr(cfg, config.Docker)
	if err := m.Delete(context.Background(), true); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if !hasCall(rr, "docker", []string{"compose", "-f", cfg.Docker.ComposeFile, "down"}) {
		t.Errorf("Delete should compose down:\n%+v", rr.calls)
	}
	if !hasCall(rr, "rm", []string{"-rf", "/opt/solace/data"}) {
		t.Errorf("purge should rm the data dir (rootful/docker):\n%+v", rr.calls)
	}
}

func TestManagerDeleteDockerComposeNoFileFallsBackToStopRm(t *testing.T) {
	dir := t.TempDir()
	cfg := ctrCfg(config.Docker, "no")
	cfg.Docker.ComposeFile = filepath.Join(dir, "absent.yml") // does not exist
	m, rr, _ := newCapMgr(cfg, config.Docker)
	if err := m.Delete(context.Background(), false); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if !hasCall(rr, "docker", []string{"stop", "solace"}) || !hasCall(rr, "docker", []string{"rm", "solace"}) {
		t.Errorf("compose Delete with no file should fall back to stop+rm:\n%+v", rr.calls)
	}
}

// --- Status / Logs / CLI / Shell --------------------------------------------

func TestManagerStatusPodman(t *testing.T) {
	m, rr, _ := newCapMgr(ctrCfg(config.Podman, "no"), config.Podman)
	if err := m.Status(context.Background()); err != nil {
		t.Fatalf("Status: %v", err)
	}
	if !hasCall(rr, "systemctl", []string{"status", "sol-pod.service", "--no-pager"}) {
		t.Errorf("podman Status should show the unit:\n%+v", rr.calls)
	}
	if !hasCall(rr, "podman", []string{"ps", "--all", "--filter", "name=^sol-pod$"}) {
		t.Errorf("podman Status should ps the container:\n%+v", rr.calls)
	}
}

func TestManagerStatusDockerCompose(t *testing.T) {
	m, buf := newEchoMgr(ctrCfg(config.Docker, "no"), config.Docker)
	if err := m.Status(context.Background()); err != nil {
		t.Fatalf("Status: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"+ docker compose -f docker-compose.yml ps",
		"+ docker ps --all --filter 'name=^solace$'",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("docker Status missing %q:\n%s", want, out)
		}
	}
}

func TestManagerLogsCLIShell(t *testing.T) {
	t.Run("logs", func(t *testing.T) {
		m, rr, _ := newCapMgr(ctrCfg(config.Docker, "no"), config.Docker)
		if err := m.Logs(context.Background()); err != nil {
			t.Fatalf("Logs: %v", err)
		}
		got := rr.last()
		if got.method != "Run" || got.name != "docker" || !eqArgs(got.args, []string{"logs", "-f", "solace"}) {
			t.Errorf("Logs argv: %+v", got)
		}
	})
	t.Run("cli", func(t *testing.T) {
		m, rr, _ := newCapMgr(ctrCfg(config.Podman, "no"), config.Podman)
		if err := m.CLI(context.Background()); err != nil {
			t.Fatalf("CLI: %v", err)
		}
		got := rr.last()
		if got.method != "RunInteractive" || got.name != "podman" || !eqArgs(got.args, []string{"exec", "-it", "sol-pod", "cli", "-A"}) {
			t.Errorf("CLI argv: %+v", got)
		}
	})
	t.Run("shell", func(t *testing.T) {
		m, rr, _ := newCapMgr(ctrCfg(config.Docker, "no"), config.Docker)
		if err := m.Shell(context.Background()); err != nil {
			t.Fatalf("Shell: %v", err)
		}
		got := rr.last()
		if got.method != "RunInteractive" || got.name != "docker" || !eqArgs(got.args, []string{"exec", "-it", "solace", "bash"}) {
			t.Errorf("Shell argv: %+v", got)
		}
	})
}

// --- pure helpers -----------------------------------------------------------

func TestReplacePSKLine(t *testing.T) {
	t.Run("replaces the nodes psk, not the replication psk", func(t *testing.T) {
		in := "replication:\n  psk: OLD\nnodes:\n  primary:\n    name: p\n  psk: \n"
		out, ok := replacePSKLine(in, "NEW")
		if !ok {
			t.Fatal("expected a replacement")
		}
		if !strings.Contains(out, `psk: "NEW"`) {
			t.Errorf("nodes psk not replaced:\n%s", out)
		}
		if !strings.Contains(out, "psk: OLD") {
			t.Errorf("replication psk must be untouched:\n%s", out)
		}
	})
	t.Run("no nodes psk line -> not replaced", func(t *testing.T) {
		in := "redundancy: yes\nnodes:\n  primary:\n    name: p\n"
		if _, ok := replacePSKLine(in, "NEW"); ok {
			t.Error("expected no replacement when there is no psk line under nodes")
		}
	})
}

func TestDefaultGenPSK(t *testing.T) {
	psk, err := defaultGenPSK()
	if err != nil {
		t.Fatalf("defaultGenPSK: %v", err)
	}
	raw, err := base64.StdEncoding.DecodeString(psk)
	if err != nil || len(raw) != 60 {
		t.Errorf("defaultGenPSK = %q (decoded %d bytes, err %v); want 60 random bytes", psk, len(raw), err)
	}
}

// --- error paths: the capRunner.fail hook drives each `err != nil` wrap branch ---

func TestManagerCheckReachableError(t *testing.T) {
	m, rr, _ := newCapMgr(ctrCfg(config.Docker, "no"), config.Docker)
	rr.outErr = fmt.Errorf("no engine") // the `version` probe is the only Output call
	if err := m.Check(context.Background()); err == nil {
		t.Fatal("Check should fail when the runtime version probe errors")
	}
}

func TestManagerPrepHostMkdirError(t *testing.T) {
	m, rr, _ := newCapMgr(ctrCfg(config.Docker, "no"), config.Docker)
	rr.fail = failOn("mkdir")
	if err := m.PrepHost(context.Background()); err == nil {
		t.Fatal("PrepHost should propagate a mkdir failure")
	}
}

func TestManagerPrepHostChownError(t *testing.T) {
	m, rr, _ := newCapMgr(ctrCfg(config.Docker, "no"), config.Docker)
	rr.fail = failOn("chown")
	if err := m.PrepHost(context.Background()); err == nil {
		t.Fatal("PrepHost should propagate a chown failure")
	}
}

func TestManagerPrepHostRootlessUnshareChownError(t *testing.T) {
	cfg := ctrCfg(config.Podman, "no")
	cfg.Podman.Rootless = true
	m, rr, _ := newCapMgr(cfg, config.Podman)
	m.Geteuid = func() int { return 1000 } // avoid the rootless-as-root WARN noise
	rr.fail = failOn("unshare")
	if err := m.PrepHost(context.Background()); err == nil {
		t.Fatal("rootless PrepHost should propagate an unshare chown failure")
	}
}

func TestManagerPrepHostGenPSKError(t *testing.T) {
	m, _, _ := newCapMgr(ctrCfg(config.Docker, "yes"), config.Docker)
	m.GenPSK = func() (string, error) { return "", fmt.Errorf("no entropy") }
	if err := m.PrepHost(context.Background()); err == nil {
		t.Fatal("PrepHost should propagate a GenPSK failure")
	}
}

func TestManagerPrepHostWritePSKReadError(t *testing.T) {
	m, _, _ := newCapMgr(ctrCfg(config.Docker, "yes"), config.Docker)
	m.EnvPath = filepath.Join(t.TempDir(), "does-not-exist.yaml")
	m.GenPSK = func() (string, error) { return "PSK", nil }
	if err := m.PrepHost(context.Background()); err == nil {
		t.Fatal("PrepHost should fail when the env file cannot be read to store the PSK")
	}
}

func TestManagerPrepHostWritePSKWriteError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root bypasses file permissions; the write-back-error branch is not reachable")
	}
	dir := t.TempDir()
	envFile := filepath.Join(dir, "env.yaml")
	// A nodes.psk line so replacePSKLine succeeds; 0o400 so the write-back fails.
	body := "redundancy: yes\nnodes:\n  primary:\n    name: pri-host\n  psk:\n"
	if err := os.WriteFile(envFile, []byte(body), 0o400); err != nil {
		t.Fatal(err)
	}
	m, _, _ := newCapMgr(ctrCfg(config.Docker, "yes"), config.Docker)
	m.EnvPath = envFile
	m.GenPSK = func() (string, error) { return "PSK", nil }
	if err := m.PrepHost(context.Background()); err == nil {
		t.Fatal("PrepHost should fail when the PSK cannot be written back")
	}
}

func TestManagerPrepHostPSKAlreadySet(t *testing.T) {
	cfg := ctrCfg(config.Docker, "yes")
	cfg.Nodes.PSK = "EXISTING"
	m, _, buf := newCapMgr(cfg, config.Docker)
	m.GenPSK = func() (string, error) { t.Fatal("GenPSK must not run when nodes.psk is already set"); return "", nil }
	if err := m.PrepHost(context.Background()); err != nil {
		t.Fatalf("PrepHost: %v", err)
	}
	if !strings.Contains(buf.String(), "already set") {
		t.Errorf("PrepHost should note the PSK is already set:\n%s", buf.String())
	}
}

func TestManagerPrepHostNoPSKLinePrintsValue(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, "env.yaml")
	if err := os.WriteFile(envFile, []byte("redundancy: yes\nnodes:\n  primary:\n    name: pri-host\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m, _, buf := newCapMgr(ctrCfg(config.Docker, "yes"), config.Docker)
	m.EnvPath = envFile
	m.GenPSK = func() (string, error) { return "GENPSK", nil }
	if err := m.PrepHost(context.Background()); err != nil {
		t.Fatalf("PrepHost: %v", err)
	}
	if !strings.Contains(buf.String(), "GENPSK") {
		t.Errorf("PrepHost should print the PSK to place when no psk line exists:\n%s", buf.String())
	}
	if got, _ := os.ReadFile(envFile); strings.Contains(string(got), "GENPSK") {
		t.Errorf("PrepHost must not write the PSK when there is no psk line to replace:\n%s", got)
	}
}

func TestManagerDeployPodmanMkdirError(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "not-a-dir")
	if err := os.WriteFile(filePath, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := ctrCfg(config.Podman, "no")
	cfg.Podman.QuadletDir = filePath // MkdirAll on a file -> error
	m, _, _ := newCapMgr(cfg, config.Podman)
	m.Geteuid = func() int { return -1 } // skip the euid guard portably
	if err := m.Deploy(context.Background(), config.Primary); err == nil {
		t.Fatal("Deploy should fail when the quadlet dir cannot be created")
	}
}

func TestManagerDeployPodmanWriteUnitError(t *testing.T) {
	dir := t.TempDir()
	cfg := ctrCfg(config.Podman, "no")
	cfg.Podman.QuadletDir = dir
	if err := os.Mkdir(filepath.Join(dir, "sol-pod.container"), 0o755); err != nil {
		t.Fatal(err) // unit path is a directory -> WriteFile fails
	}
	m, _, _ := newCapMgr(cfg, config.Podman)
	m.Geteuid = func() int { return -1 }
	if err := m.Deploy(context.Background(), config.Primary); err == nil {
		t.Fatal("Deploy should fail when the quadlet unit cannot be written")
	}
}

func TestManagerDeployPodmanDaemonReloadError(t *testing.T) {
	cfg := ctrCfg(config.Podman, "no")
	cfg.Podman.QuadletDir = t.TempDir()
	m, rr, _ := newCapMgr(cfg, config.Podman)
	m.Geteuid = func() int { return -1 }
	rr.fail = failOn("daemon-reload")
	if err := m.Deploy(context.Background(), config.Primary); err == nil {
		t.Fatal("Deploy should propagate a daemon-reload failure")
	}
}

func TestManagerDeployPodmanStartError(t *testing.T) {
	cfg := ctrCfg(config.Podman, "no")
	cfg.Podman.QuadletDir = t.TempDir()
	m, rr, _ := newCapMgr(cfg, config.Podman)
	m.Geteuid = func() int { return -1 }
	rr.fail = failOn("start")
	if err := m.Deploy(context.Background(), config.Primary); err == nil {
		t.Fatal("Deploy should propagate a service-start failure")
	}
}

func TestManagerDeployPodmanEUIDGuardFails(t *testing.T) {
	cfg := ctrCfg(config.Podman, "no")
	cfg.Podman.Rootless = true
	cfg.Podman.QuadletDir = t.TempDir()
	m, _, _ := newCapMgr(cfg, config.Podman)
	m.Geteuid = func() int { return 0 } // rootless as root -> guard rejects
	if err := m.Deploy(context.Background(), config.Primary); err == nil {
		t.Fatal("Deploy should fail when the euid guard rejects the host")
	}
}

func TestManagerDeployDockerComposeWriteError(t *testing.T) {
	dir := t.TempDir()
	cfg := ctrCfg(config.Docker, "no")
	// A directory in the compose file's place makes WriteFile fail.
	cfg.Docker.ComposeFile = filepath.Join(dir, "compose.yml")
	if err := os.Mkdir(cfg.Docker.ComposeFile, 0o755); err != nil {
		t.Fatal(err)
	}
	m, _, _ := newCapMgr(cfg, config.Docker)
	if err := m.Deploy(context.Background(), config.Primary); err == nil {
		t.Fatal("Deploy should fail when the compose file cannot be written")
	}
}

func TestManagerDeployDockerComposeUpError(t *testing.T) {
	dir := t.TempDir()
	cfg := ctrCfg(config.Docker, "no")
	cfg.Docker.ComposeFile = filepath.Join(dir, "compose.yml")
	m, rr, _ := newCapMgr(cfg, config.Docker)
	rr.fail = failOn("compose")
	if err := m.Deploy(context.Background(), config.Primary); err == nil {
		t.Fatal("Deploy should propagate a compose up failure")
	}
}

func TestManagerDeployPodmanSecretCreateError(t *testing.T) {
	cfg := ctrCfg(config.Podman, "no")
	cfg.Podman.QuadletDir = t.TempDir()
	m, rr, _ := newCapMgr(cfg, config.Podman)
	m.Geteuid = func() int { return -1 }
	rr.fail = failOn("secret")
	err := m.Deploy(context.Background(), config.Primary)
	if err == nil {
		t.Fatal("Deploy should propagate a secret-create failure")
	}
	if !strings.Contains(err.Error(), "admin.pass") {
		t.Errorf("error should name the config key behind the secret, got: %v", err)
	}
}

// TestManagerRedeployUnchangedRestartsForRotation covers the one way a rotated
// secret can reach a running broker: the artifact is unchanged (the value lives in
// the config and the environment, so there is nothing to diff), and --restart
// forces the recreate that `compose up -d` would otherwise skip.
func TestManagerRedeployUnchangedRestartsForRotation(t *testing.T) {
	dir := t.TempDir()
	cfg := ctrCfg(config.Docker, "no")
	cfg.Docker.ComposeFile = filepath.Join(dir, "compose.yml")
	if err := os.WriteFile(cfg.Docker.ComposeFile,
		render.Compose(cfg, cfg.ResolveNode(config.Primary)), 0o600); err != nil {
		t.Fatal(err)
	}
	m, rr, buf := newCapMgr(cfg, config.Docker)
	m.Restart = true
	rr.out = []byte("solace\n") // ps lists this container by name -> running
	if err := m.Deploy(context.Background(), config.Primary); err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	want := []string{"compose", "-f", cfg.Docker.ComposeFile, "up", "-d", "--force-recreate"}
	if !hasCall(rr, "docker", want) {
		t.Errorf("--restart on an unchanged compose file should force a recreate:\n%+v", rr.calls)
	}
	if !strings.Contains(buf.String(), "rotated secret") {
		t.Errorf("Deploy should say why it recreated the container:\n%s", buf.String())
	}
}

// TestManagerRedeployPodmanUnchangedRestartsForRotation is the podman half: the
// store secrets were just replaced, but a running container still holds the old
// values and the unit is byte-identical, so --restart is what applies them.
func TestManagerRedeployPodmanUnchangedRestartsForRotation(t *testing.T) {
	dir := t.TempDir()
	cfg := ctrCfg(config.Podman, "no")
	cfg.Podman.QuadletDir = dir
	if err := os.WriteFile(filepath.Join(dir, "sol-pod.container"),
		render.Quadlet(cfg, cfg.ResolveNode(config.Primary)), 0o600); err != nil {
		t.Fatal(err)
	}
	m, rr, buf := newCapMgr(cfg, config.Podman)
	m.Geteuid = func() int { return -1 }
	m.Restart = true
	rr.out = []byte("active\n")
	if err := m.Deploy(context.Background(), config.Primary); err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	if !hasCall(rr, "systemctl", withUser(cfg, "restart", "sol-pod.service")) {
		t.Errorf("--restart on an unchanged unit should restart the service:\n%+v", rr.calls)
	}
	if !strings.Contains(buf.String(), "rotated secret") {
		t.Errorf("Deploy should say why it restarted:\n%s", buf.String())
	}
}

// TestContainerRunningMatchesNameExactly guards the branch selector: `ps --filter
// name=` is an unanchored regex on both engines, so a sibling deployment on the same
// host would otherwise be mistaken for this one. Getting it wrong in either
// direction is expensive -- a false positive skips the deploy, a false negative
// force-recreates a live broker without asking.
func TestContainerRunningMatchesNameExactly(t *testing.T) {
	cases := []struct {
		name    string
		listing string
		want    bool
	}{
		{"exact match", "solace\n", true},
		{"among others", "other\nsolace\nsolace-edge\n", true},
		{"sibling only", "solace-edge\n", false},
		{"prefix only", "sol\n", false},
		{"nothing running", "", false},
		{"padded listing", "  solace  \n", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := ctrCfg(config.Docker, "no") // container.name is "solace"
			m, rr, _ := newCapMgr(cfg, config.Docker)
			rr.out = []byte(tc.listing)
			if got := m.containerRunning(context.Background()); got != tc.want {
				t.Errorf("containerRunning(%q) = %v, want %v", tc.listing, got, tc.want)
			}
		})
	}

	// A probe that cannot run reads as "not running": the deploy then creates or
	// recreates, and if the engine is really unreachable the compose call that
	// follows fails loudly anyway -- so this never silently skips a deploy.
	t.Run("probe fails", func(t *testing.T) {
		m, rr, _ := newCapMgr(ctrCfg(config.Docker, "no"), config.Docker)
		rr.out = []byte("solace\n")
		rr.outErr = fmt.Errorf("engine unreachable")
		if m.containerRunning(context.Background()) {
			t.Error("a failed probe must not report the container as running")
		}
	})
}

// TestManagerRedeployStoppedContainerRecreates covers the arm no consent prompt
// guards: the compose file is unchanged and the container exists but is stopped.
// A plain `up -d` would START it, and a compose secret's value is baked in at
// creation -- so the broker would come back on the credentials it was created with
// and nothing would say so. Recreating is safe here (no traffic to drop) and is
// what makes a deploy's result honest.
func TestManagerRedeployStoppedContainerRecreates(t *testing.T) {
	dir := t.TempDir()
	cfg := ctrCfg(config.Docker, "no")
	cfg.Docker.ComposeFile = filepath.Join(dir, "compose.yml")
	if err := os.WriteFile(cfg.Docker.ComposeFile,
		render.Compose(cfg, cfg.ResolveNode(config.Primary)), 0o600); err != nil {
		t.Fatal(err)
	}
	m, rr, _ := newCapMgr(cfg, config.Docker)
	rr.out = []byte("") // ps lists nothing running
	if err := m.Deploy(context.Background(), config.Primary); err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	want := []string{"compose", "-f", cfg.Docker.ComposeFile, "up", "-d", "--force-recreate"}
	if !hasCall(rr, "docker", want) {
		t.Errorf("a stopped container must be recreated, not started, so a rotated secret applies:\n%+v", rr.calls)
	}
}

// TestManagerRedeployUnchangedHintsRotation is the same state without --restart:
// nothing happens, and the operator is told how to apply a rotation.
func TestManagerRedeployUnchangedHintsRotation(t *testing.T) {
	dir := t.TempDir()
	cfg := ctrCfg(config.Docker, "no")
	cfg.Docker.ComposeFile = filepath.Join(dir, "compose.yml")
	if err := os.WriteFile(cfg.Docker.ComposeFile,
		render.Compose(cfg, cfg.ResolveNode(config.Primary)), 0o600); err != nil {
		t.Fatal(err)
	}
	m, rr, buf := newCapMgr(cfg, config.Docker)
	rr.out = []byte("solace\n")
	if err := m.Deploy(context.Background(), config.Primary); err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	for _, c := range rr.calls {
		if containsStr(c.args, "--force-recreate") {
			t.Errorf("without --restart nothing may be recreated:\n%+v", rr.calls)
		}
	}
	if !strings.Contains(buf.String(), "re-run with --restart") {
		t.Errorf("Deploy should name the way to apply a rotated secret:\n%s", buf.String())
	}
}

func TestManagerDeletePodmanStopTolerated(t *testing.T) {
	cfg := ctrCfg(config.Podman, "no")
	cfg.Podman.QuadletDir = t.TempDir()
	m, rr, buf := newCapMgr(cfg, config.Podman)
	rr.fail = failOn("stop")
	if err := m.Delete(context.Background(), false); err != nil {
		t.Fatalf("Delete should tolerate a failed stop: %v", err)
	}
	if !strings.Contains(buf.String(), "stopping") {
		t.Errorf("Delete should warn about the failed stop:\n%s", buf.String())
	}
}

func TestManagerDeletePodmanDaemonReloadError(t *testing.T) {
	cfg := ctrCfg(config.Podman, "no")
	cfg.Podman.QuadletDir = t.TempDir()
	m, rr, _ := newCapMgr(cfg, config.Podman)
	rr.fail = failOn("daemon-reload")
	if err := m.Delete(context.Background(), false); err == nil {
		t.Fatal("Delete should propagate a daemon-reload failure")
	}
}

func TestManagerDeletePodmanRemoveUnitError(t *testing.T) {
	dir := t.TempDir()
	cfg := ctrCfg(config.Podman, "no")
	cfg.Podman.QuadletDir = dir
	unit := filepath.Join(dir, "sol-pod.container")
	if err := os.Mkdir(unit, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(unit, "keep"), []byte("x"), 0o600); err != nil {
		t.Fatal(err) // non-empty dir -> os.Remove fails with a non-IsNotExist error
	}
	m, _, _ := newCapMgr(cfg, config.Podman)
	if err := m.Delete(context.Background(), false); err == nil {
		t.Fatal("Delete should fail when the quadlet unit cannot be removed")
	}
}

func TestManagerDeleteDockerComposeDownError(t *testing.T) {
	dir := t.TempDir()
	cfg := ctrCfg(config.Docker, "no")
	cfg.Docker.ComposeFile = filepath.Join(dir, "compose.yml")
	if err := os.WriteFile(cfg.Docker.ComposeFile, []byte("services:\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	m, rr, _ := newCapMgr(cfg, config.Docker)
	rr.fail = failOn("down")
	if err := m.Delete(context.Background(), false); err == nil {
		t.Fatal("Delete should propagate a compose down failure")
	}
}

func TestManagerDeleteDockerStopTolerated(t *testing.T) {
	dir := t.TempDir()
	cfg := ctrCfg(config.Docker, "no")
	cfg.Docker.ComposeFile = filepath.Join(dir, "absent.yml") // no file -> stop/rm fallback
	m, rr, buf := newCapMgr(cfg, config.Docker)
	rr.fail = failOn("stop")
	if err := m.Delete(context.Background(), false); err != nil {
		t.Fatalf("Delete should tolerate a failed stop: %v", err)
	}
	if !strings.Contains(buf.String(), "stopping container") {
		t.Errorf("Delete should warn about the failed stop:\n%s", buf.String())
	}
}

func TestManagerDeletePurgeError(t *testing.T) {
	dir := t.TempDir()
	cfg := ctrCfg(config.Docker, "no")
	cfg.Docker.ComposeFile = filepath.Join(dir, "compose.yml")
	if err := os.WriteFile(cfg.Docker.ComposeFile, []byte("services:\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	m, rr, _ := newCapMgr(cfg, config.Docker)
	rr.fail = failOn("-rf") // targets only the purge `rm -rf`, not compose down
	if err := m.Delete(context.Background(), true); err == nil {
		t.Fatal("Delete --purge should propagate a data-dir rm failure")
	}
}

// --- Status permutations / tolerated non-zero -------------------------------

// TestManagerStatusDockerNoComposeFile covers the other branch of Status: with no
// compose file on disk there is nothing for `compose ps` to read, so only the
// container listing runs.
func TestManagerStatusDockerNoComposeFile(t *testing.T) {
	dir := t.TempDir()
	cfg := ctrCfg(config.Docker, "no")
	cfg.Docker.ComposeFile = filepath.Join(dir, "absent.yml")
	m, rr, _ := newCapMgr(cfg, config.Docker)
	if err := m.Status(context.Background()); err != nil {
		t.Fatalf("Status: %v", err)
	}
	if !hasCall(rr, "docker", []string{"ps", "--all", "--filter", "name=^solace$"}) {
		t.Errorf("Status should ps the container:\n%+v", rr.calls)
	}
	for _, c := range rr.calls {
		if len(c.args) > 0 && c.args[0] == "compose" {
			t.Errorf("Status must not call compose with no compose file:\n%+v", rr.calls)
		}
	}
}

func TestManagerStatusPodmanUnitInactiveTolerated(t *testing.T) {
	m, rr, buf := newCapMgr(ctrCfg(config.Podman, "no"), config.Podman)
	rr.fail = failOn("status")
	if err := m.Status(context.Background()); err != nil {
		t.Fatalf("Status should tolerate an inactive unit: %v", err)
	}
	if !strings.Contains(buf.String(), "non-zero") {
		t.Errorf("Status should warn when the unit is not active:\n%s", buf.String())
	}
	if !hasCall(rr, "podman", []string{"ps", "--all", "--filter", "name=^sol-pod$"}) {
		t.Errorf("Status should still ps the container:\n%+v", rr.calls)
	}
}

func TestManagerStatusDockerComposePsTolerated(t *testing.T) {
	dir := t.TempDir()
	cfg := ctrCfg(config.Docker, "no")
	cfg.Docker.ComposeFile = filepath.Join(dir, "compose.yml")
	if err := os.WriteFile(cfg.Docker.ComposeFile, []byte("services:\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	m, rr, buf := newCapMgr(cfg, config.Docker)
	rr.fail = failOn("compose") // fails only `compose ps`, not the plain `ps`
	if err := m.Status(context.Background()); err != nil {
		t.Fatalf("Status should tolerate a compose ps failure: %v", err)
	}
	if !strings.Contains(buf.String(), "compose ps failed") {
		t.Errorf("Status should warn on a compose ps failure:\n%s", buf.String())
	}
	if !hasCall(rr, "docker", []string{"ps", "--all", "--filter", "name=^solace$"}) {
		t.Errorf("Status should still ps the container:\n%+v", rr.calls)
	}
}

// --- euid guard (via the Geteuid seam) --------------------------------------

func TestManagerCheckPodmanEUID(t *testing.T) {
	cases := []struct {
		name     string
		rootless bool
		euid     int
		wantErr  bool
	}{
		{"rootless as root fails", true, 0, true},
		{"rootful as non-root fails", false, 1000, true},
		{"rootless as non-root passes", true, 1000, false},
		{"rootful as root passes", false, 0, false},
		{"non-posix euid skips guard", false, -1, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := ctrCfg(config.Podman, "no")
			cfg.Podman.Rootless = tc.rootless
			m, _, _ := newCapMgr(cfg, config.Podman)
			m.Geteuid = func() int { return tc.euid }
			err := m.checkPodmanEUID()
			if tc.wantErr != (err != nil) {
				t.Errorf("checkPodmanEUID rootless=%t euid=%d: err=%v, wantErr=%t", tc.rootless, tc.euid, err, tc.wantErr)
			}
		})
	}
}

func TestManagerPrepHostRootlessAsRootWarns(t *testing.T) {
	cfg := ctrCfg(config.Podman, "no")
	cfg.Podman.Rootless = true
	m, _, buf := newCapMgr(cfg, config.Podman)
	m.Geteuid = func() int { return 0 }
	if err := m.PrepHost(context.Background()); err != nil {
		t.Fatalf("PrepHost: %v", err)
	}
	if !strings.Contains(buf.String(), "running as root") {
		t.Errorf("PrepHost should warn when rootless runs as root:\n%s", buf.String())
	}
}

// --- nil Log/Out sinks fall back to discard / os.Stdout ----------------------

func TestManagerNilSinks(t *testing.T) {
	m := NewManager(engine.Echo{}, ctrCfg(config.Docker, "no"), config.Docker, nil, nil)
	m.Resolve = func(string) bool { return true }
	if err := m.Check(context.Background()); err != nil {
		t.Fatalf("Check with nil Log/Out should not error: %v", err)
	}
}
