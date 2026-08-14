package container

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"solace/internal/config"
	"solace/internal/engine"
	"solace/internal/render"
)

// TestPreflightRunsBeforeAnything is the layer-7 ordering guarantee on the
// container side: `<runtime> info` is the FIRST thing every mutating operation
// does. Anything that ran before it -- a compose file written, a secret loaded into
// podman's store -- would be host state left behind by a deploy that then failed on
// a stopped daemon.
func TestPreflightRunsBeforeAnything(t *testing.T) {
	cases := []struct {
		name string
		p    config.Platform
		call func(*Manager) error
	}{
		{"docker deploy", config.Docker, func(m *Manager) error { return m.Deploy(context.Background(), config.Primary) }},
		{"podman deploy", config.Podman, func(m *Manager) error { return m.Deploy(context.Background(), config.Primary) }},
		{"docker delete", config.Docker, func(m *Manager) error { return m.Delete(context.Background(), false) }},
		{"podman delete", config.Podman, func(m *Manager) error { return m.Delete(context.Background(), false) }},
		{"prep host", config.Docker, func(m *Manager) error { return m.PrepHost(context.Background()) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Chdir(t.TempDir()) // keep any written artifact out of the repo
			cfg := ctrCfg(tc.p, "no")
			cfg.Podman.QuadletDir = t.TempDir()
			m, rr, _ := newCapMgr(cfg, tc.p)
			m.Geteuid = func() int { return 0 }
			_ = tc.call(m) // the operation's own outcome is asserted elsewhere

			if len(rr.calls) == 0 {
				t.Fatal("no calls recorded: the read-only `info` probe must run first")
			}
			first := rr.calls[0]
			if first.method != "Output" || !eqArgs(first.args, []string{"info"}) {
				t.Fatalf("first call = %+v, want the Output `info` probe", first)
			}
			if first.name != string(tc.p) {
				t.Errorf("probe ran %q, want the configured runtime %q", first.name, tc.p)
			}
		})
	}
}

// TestPreflightFailureStopsTheDeploy: when the engine cannot be reached, Deploy
// stops nonzero with the engine's own error passed through and one actionable hint
// -- and writes nothing. Without the "writes nothing" half the probe would be
// decoration.
func TestPreflightFailureStopsTheDeploy(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	cfg := ctrCfg(config.Docker, "no")
	m, rr, _ := newCapMgr(cfg, config.Docker)
	rr.outFail = failOn("info")

	err := m.Deploy(context.Background(), config.Primary)
	if err == nil {
		t.Fatal("Deploy must fail when the engine cannot be reached")
	}
	msg := err.Error()
	for _, want := range []string{"cannot talk to the Docker engine", "start the daemon", "docker` group"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error = %v, want it to contain %q", msg, want)
		}
	}
	// The engine's own failure is carried, not replaced.
	if !strings.Contains(msg, "injected failure") {
		t.Errorf("error = %v, want it to carry the engine's own error", msg)
	}
	// Nothing written, nothing else run.
	if _, statErr := os.Stat(filepath.Join(dir, "docker-compose.yml")); statErr == nil {
		t.Error("a compose file was written despite a failed preflight")
	}
	if len(rr.calls) != 1 {
		t.Errorf("%d calls made after a failed preflight, want only the probe: %+v", len(rr.calls), rr.calls)
	}
}

// TestPreflightHintIsPlatformShaped: rootless podman's usual failure is a missing
// user session, and telling that operator to `sudo systemctl start` would start the
// engine their deploy is not using. The tool never starts anything itself.
func TestPreflightHintIsPlatformShaped(t *testing.T) {
	cases := []struct {
		name     string
		p        config.Platform
		rootless bool
		want     []string
		reject   string
	}{
		{"docker", config.Docker, false, []string{"sudo systemctl start docker", "newgrp docker"}, ""},
		{"rootful podman", config.Podman, false, []string{"sudo systemctl start podman.socket"}, ""},
		{"rootless podman", config.Podman, true,
			[]string{"systemctl --user start podman.socket", "enable-linger", "do NOT use sudo"}, "sudo systemctl"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := ctrCfg(tc.p, "no")
			cfg.Podman.Rootless = tc.rootless
			m, rr, _ := newCapMgr(cfg, tc.p)
			rr.outFail = failOn("info")

			err := m.Preflight(context.Background())
			if err == nil {
				t.Fatal("Preflight must fail when the engine cannot be reached")
			}
			for _, want := range tc.want {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("hint = %v, want it to contain %q", err, want)
				}
			}
			if tc.reject != "" && strings.Contains(err.Error(), tc.reject) {
				t.Errorf("hint = %v, must not suggest %q for a rootless deploy", err, tc.reject)
			}
			// Never an offer to do it for them.
			for _, forbidden := range []string{"starting it for you", "logging in"} {
				if strings.Contains(err.Error(), forbidden) {
					t.Errorf("hint = %v, must not offer to act on the operator's behalf", err)
				}
			}
		})
	}
}

// TestPreflightIsPreviewableUnderDryRun: --dry-run must stay usable with no engine
// installed at all, so the probe is echoed and its assertion skipped -- the same
// arrangement checkDNS and checkPodmanEUID already use. This is why there is no
// skip flag: the one legitimate reason to skip already has one.
func TestPreflightIsPreviewableUnderDryRun(t *testing.T) {
	m, buf := newEchoMgr(ctrCfg(config.Docker, "no"), config.Docker)
	if err := m.Preflight(context.Background()); err != nil {
		t.Fatalf("Preflight under --dry-run must not fail: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "+ docker info") {
		t.Errorf("dry-run should still show the probe it would run:\n%s", out)
	}
	if !strings.Contains(out, "skipped (dry-run)") {
		t.Errorf("dry-run should say the assertion was skipped:\n%s", out)
	}
}

// TestComposeSecretEnvNamesCannotBeSystemVars is the config-side half of the
// child-environment rule (engine's TestChildEnvNamesAreNotSystemVariables is the
// other). docker's compose secrets are the ONE place this tool adds variables to a
// child process, and their names are built from config text -- the container name
// and each additional user's name. This proves the fixed literal suffix always
// survives that folding, so no config value can produce a bare PATH, LD_PRELOAD or
// any other name the child's loader reads.
func TestComposeSecretEnvNamesCannotBeSystemVars(t *testing.T) {
	// Names chosen to attack the folding: the exact target, its case variants, and
	// separators that fold to '_'.
	hostile := []string{"PATH", "path", "LD_PRELOAD", "ld.preload", "ld-preload", "IFS", "BASH_ENV"}

	for _, name := range hostile {
		cfg := ctrCfg(config.Docker, "yes")
		cfg.Docker.Container.Name = name
		cfg.Admin.AdditionalUsers = []config.AdditionalUser{
			{Username: name + "-x", Password: "p", AccessLevel: "read-only"},
		}
		m, _, _ := newCapMgr(cfg, config.Docker)

		for _, pair := range m.composeSecretEnv() {
			varName, _, _ := strings.Cut(pair, "=")
			for _, bad := range hostile {
				if strings.EqualFold(varName, bad) {
					t.Errorf("container.name %q produced the child variable %q, which the loader reads", name, varName)
				}
			}
			// The structural reason it cannot: a fixed literal suffix always
			// remains, so the name is never only config text.
			if !strings.Contains(varName, "_PASSWORD") && !strings.Contains(varName, "_PSK") &&
				!strings.Contains(varName, "_PASSPHRASE") {
				t.Errorf("child variable %q carries no fixed suffix; the collision guarantee rests on that suffix", varName)
			}
		}
	}
}

// TestComposeSecretEnvIsTheOnlyChildEnvironment guards the assumption the two tests
// above rest on: if a second code path ever starts adding variables to a child, it
// must be audited the same way. render.ContainerSecrets is that single source.
func TestComposeSecretEnvIsTheOnlyChildEnvironment(t *testing.T) {
	cfg := ctrCfg(config.Docker, "yes")
	m, _, _ := newCapMgr(cfg, config.Docker)
	if got, want := len(m.composeSecretEnv()), len(render.ContainerSecrets(cfg, config.Docker)); got != want {
		t.Errorf("composeSecretEnv built %d variables from %d secrets; it must pass through exactly the "+
			"secrets render declares, adding none of its own", got, want)
	}
	// And every value is masked in any display path (§3).
	masked := engine.MaskEnv(m.composeSecretEnv())
	if strings.Contains(masked, "secret-pass") || strings.Contains(masked, "test-psk") {
		t.Errorf("MaskEnv leaked a secret value: %q", masked)
	}
}
