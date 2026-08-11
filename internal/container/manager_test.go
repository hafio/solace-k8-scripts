package container

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"solace/internal/config"
	"solace/internal/engine"
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
	switch p {
	case config.Podman:
		c.Podman.Runtime = config.Command{"podman"}
		c.Podman.Container.Name = "sol-pod"
		c.Podman.Container.RunUser = "1000:1000"
		c.Podman.Container.DataDir = "/opt/solace/data"
		c.Podman.QuadletDir = "/etc/containers/systemd"
		c.Podman.Network.Mode = "host"
	default:
		c.Docker.Runtime = config.Command{"docker"}
		c.Docker.Mode = "compose"
		c.Docker.Container.Name = "solace"
		c.Docker.Container.RunUser = "0:0"
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
			"+ " + m.runtime().String() + " version",
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
	if !hasCall(rr, "docker", []string{"compose", "-f", cfg.Docker.ComposeFile, "up", "-d"}) {
		t.Errorf("Deploy should compose up:\n%+v", rr.calls)
	}
}

func TestManagerDeployDockerRun(t *testing.T) {
	cfg := ctrCfg(config.Docker, "no")
	cfg.Docker.Mode = "run"
	m, buf := newEchoMgr(cfg, config.Docker)
	if err := m.Deploy(context.Background(), config.Primary); err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	if !strings.Contains(buf.String(), "+ docker run") {
		t.Errorf("run-mode Deploy should `docker run`:\n%s", buf.String())
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

func TestManagerDeleteDockerRunStopsAndRemoves(t *testing.T) {
	cfg := ctrCfg(config.Docker, "no")
	cfg.Docker.Mode = "run"
	m, rr, _ := newCapMgr(cfg, config.Docker)
	if err := m.Delete(context.Background(), true); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if !hasCall(rr, "docker", []string{"stop", "solace"}) || !hasCall(rr, "docker", []string{"rm", "solace"}) {
		t.Errorf("run-mode Delete should stop+rm the container:\n%+v", rr.calls)
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
	if !hasCall(rr, "podman", []string{"ps", "--all", "--filter", "name=sol-pod"}) {
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
		"+ docker ps --all --filter name=solace",
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
	cfg := ctrCfg(config.Docker, "no")
	cfg.Docker.ComposeFile = t.TempDir() // a directory -> WriteFile fails
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

func TestManagerDeployDockerRunError(t *testing.T) {
	cfg := ctrCfg(config.Docker, "no")
	cfg.Docker.Mode = "run"
	m, rr, _ := newCapMgr(cfg, config.Docker)
	rr.fail = failOn("run")
	if err := m.Deploy(context.Background(), config.Primary); err == nil {
		t.Fatal("Deploy should propagate a docker run failure")
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

func TestManagerDeleteDockerRunStopTolerated(t *testing.T) {
	cfg := ctrCfg(config.Docker, "no")
	cfg.Docker.Mode = "run"
	m, rr, buf := newCapMgr(cfg, config.Docker)
	rr.fail = failOn("stop")
	if err := m.Delete(context.Background(), false); err != nil {
		t.Fatalf("run-mode Delete should tolerate a failed stop: %v", err)
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

func TestManagerStatusDockerRun(t *testing.T) {
	cfg := ctrCfg(config.Docker, "no")
	cfg.Docker.Mode = "run"
	m, rr, _ := newCapMgr(cfg, config.Docker)
	if err := m.Status(context.Background()); err != nil {
		t.Fatalf("Status: %v", err)
	}
	if !hasCall(rr, "docker", []string{"ps", "--all", "--filter", "name=solace"}) {
		t.Errorf("run-mode Status should ps the container:\n%+v", rr.calls)
	}
	for _, c := range rr.calls {
		if len(c.args) > 0 && c.args[0] == "compose" {
			t.Errorf("run-mode Status must not call compose:\n%+v", rr.calls)
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
	if !hasCall(rr, "podman", []string{"ps", "--all", "--filter", "name=sol-pod"}) {
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
	if !hasCall(rr, "docker", []string{"ps", "--all", "--filter", "name=solace"}) {
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
