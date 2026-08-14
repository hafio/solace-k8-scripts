package container

import (
	"context"
	"io"
	"strings"
	"testing"

	"solace/internal/broker"
	"solace/internal/config"
)

// wrappedCtrCfg is the docker ctrCfg with a multi-token runtime. The bash
// container bootstrap expanded ${CONTAINER_RUNTIME} unquoted too, so a wrapper
// such as `lima nerdctl` (the container CLI inside a Lima VM) has to reach exec as
// argv, not as one impossible filename.
// `lima` is not a CLI this tool drives, so the execution guard refuses it from
// config alone; the operator approves it with --allow-command, which is what
// AllowCommands models here. These tests are therefore also the container-side
// proof that an approved chained runner reaches argv intact. Deliberately NOT a
// privilege-escalation wrapper -- --allow-command can never approve one of those
// (config.neverAllowed), which TestAllowCommandsRejectsEscalation covers.
func wrappedCtrCfg() *config.Config {
	cfg := ctrCfg(config.Docker, "yes")
	cfg.Docker.Runtime = config.Command{"lima", "nerdctl"}
	cfg.Docker.Compose = nil // let compose derive from the wrapped runtime
	if err := cfg.AllowCommands([]string{"lima"}); err != nil {
		panic("wrappedCtrCfg: " + err.Error()) // a fixture bug, not a test failure
	}
	return cfg
}

// unapprovedCtrCfg is wrappedCtrCfg WITHOUT the operator's approval -- an env file
// that tried to elevate on its own.
func unapprovedCtrCfg(p config.Platform) *config.Config {
	cfg := ctrCfg(p, "yes")
	cmd := config.Command{"lima", string(p)}
	if p == config.Podman {
		cfg.Podman.Runtime = cmd
	} else {
		cfg.Docker.Runtime = cmd
		cfg.Docker.Compose = nil
	}
	return cfg
}

// TestCtrExecutorRefusesUnapprovedRuntime is the container half of layer 5: the
// Manager and the transport are built straight from a *config.Config that never
// went through config.Load, so nothing validated it -- and every path that could
// reach exec must still refuse, handing nothing to the runner.
func TestCtrExecutorRefusesUnapprovedRuntime(t *testing.T) {
	calls := []struct {
		name string
		call func(*config.Config, *capRunner) error
	}{
		{"run", func(cfg *config.Config, rr *capRunner) error {
			return mgrOver(rr, cfg).Logs(context.Background())
		}},
		{"output", func(cfg *config.Config, rr *capRunner) error {
			return mgrOver(rr, cfg).Reachable(context.Background())
		}},
		{"compose", func(cfg *config.Config, rr *capRunner) error {
			return mgrOver(rr, cfg).compose(context.Background(), "ps")
		}},
		{"interactive", func(cfg *config.Config, rr *capRunner) error {
			return mgrOver(rr, cfg).CLI(context.Background())
		}},
		{"preflight", func(cfg *config.Config, rr *capRunner) error {
			return mgrOver(rr, cfg).Preflight(context.Background())
		}},
		{"transport exec", func(cfg *config.Config, rr *capRunner) error {
			return NewTransport(rr, cfg, config.Docker).Run(context.Background(), config.Primary, "ls")
		}},
		{"transport output", func(cfg *config.Config, rr *capRunner) error {
			_, err := NewTransport(rr, cfg, config.Docker).Output(context.Background(), config.Primary, "ls")
			return err
		}},
		{"transport outputInput", func(cfg *config.Config, rr *capRunner) error {
			_, err := NewTransport(rr, cfg, config.Docker).OutputInput(context.Background(), config.Primary, []byte("i"), "cat")
			return err
		}},
		{"transport upload", func(cfg *config.Config, rr *capRunner) error {
			return NewTransport(rr, cfg, config.Docker).Upload(context.Background(), config.Primary, []byte("b"), "/tmp/f")
		}},
		{"transport uploadFile", func(cfg *config.Config, rr *capRunner) error {
			return NewTransport(rr, cfg, config.Docker).UploadFile(context.Background(), config.Primary, "l", "/tmp/d")
		}},
		{"transport download", func(cfg *config.Config, rr *capRunner) error {
			return NewTransport(rr, cfg, config.Docker).Download(context.Background(), config.Primary, "/tmp/r", "l")
		}},
	}
	for _, tc := range calls {
		t.Run(tc.name, func(t *testing.T) {
			rr := &capRunner{}
			err := tc.call(unapprovedCtrCfg(config.Docker), rr)
			if err == nil {
				t.Fatal("an unapproved runtime reached the runner")
			}
			if !strings.Contains(err.Error(), "--allow-command lima") {
				t.Errorf("error = %v, want it to name the escape hatch", err)
			}
			if len(rr.calls) != 0 {
				t.Errorf("%d call(s) reached the runner before the refusal: %+v", len(rr.calls), rr.calls)
			}
		})
	}
}

// mgrOver builds a Manager over an arbitrary runner and config for the guard tests,
// bypassing newCapMgr's fixed fixture.
func mgrOver(rr *capRunner, cfg *config.Config) *Manager {
	m := NewManager(rr, cfg, config.Docker, nil, io.Discard)
	m.Resolve = func(string) bool { return true }
	return m
}

// withWrapper is the argv prefix wrappedCtrCfg contributes ahead of every subcommand.
func withWrapper(rest ...string) []string {
	return append([]string{"nerdctl"}, rest...)
}

// TestManagerReachableProbesRuntimeThenCompose: on docker both the engine and the
// compose command are probed, and the compose default derives from the wrapped
// runtime -- a `lima nerdctl` runtime must yield `lima nerdctl compose`, not a
// bare `docker compose` that would bypass the wrapper.
func TestManagerReachableProbesRuntimeThenCompose(t *testing.T) {
	m, rr, _ := newCapMgr(wrappedCtrCfg(), config.Docker)
	if err := m.Reachable(context.Background()); err != nil {
		t.Fatalf("Reachable: %v", err)
	}
	if !hasCall(rr, "lima", withWrapper("version")) {
		t.Errorf("Reachable should probe the engine through the wrapper:\n%+v", rr.calls)
	}
	if !hasCall(rr, "lima", withWrapper("compose", "version")) {
		t.Errorf("Reachable should probe compose through the wrapper:\n%+v", rr.calls)
	}
}

// TestManagerHonoursRuntime: the manager's own shell-outs run argv[0] from the
// configured runtime with its leading arguments ahead of the subcommand.
func TestManagerHonoursRuntime(t *testing.T) {
	cases := []struct {
		name       string
		call       func(*Manager) error
		wantMethod string
		wantArgs   []string
	}{
		{
			name:       "run",
			call:       func(m *Manager) error { return m.Logs(context.Background()) },
			wantMethod: "Run",
			wantArgs:   withWrapper("logs", "-f", "solace"),
		},
		{
			// Reachable probes the engine then the compose command, so the compose
			// probe is the last call; the engine probe is asserted separately in
			// TestManagerReachableProbesRuntimeThenCompose.
			name:       "output",
			call:       func(m *Manager) error { return m.Reachable(context.Background()) },
			wantMethod: "Output",
			wantArgs:   withWrapper("compose", "version"),
		},
		{
			name:       "cli",
			call:       func(m *Manager) error { return m.CLI(context.Background()) },
			wantMethod: "RunInteractive",
			wantArgs:   withWrapper("exec", "-it", "solace", "cli", "-A"),
		},
		{
			name:       "shell",
			call:       func(m *Manager) error { return m.Shell(context.Background()) },
			wantMethod: "RunInteractive",
			wantArgs:   withWrapper("exec", "-it", "solace", "bash"),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m, rr, _ := newCapMgr(wrappedCtrCfg(), config.Docker)
			if err := tc.call(m); err != nil {
				t.Fatalf("call: %v", err)
			}
			got := rr.last()
			if got.method != tc.wantMethod {
				t.Errorf("method = %q, want %q", got.method, tc.wantMethod)
			}
			if got.name != "lima" {
				t.Errorf("argv[0] = %q, want lima (from docker.runtime)", got.name)
			}
			if !eqArgs(got.args, tc.wantArgs) {
				t.Errorf("args = %v, want %v", got.args, tc.wantArgs)
			}
		})
	}
}

// TestCtrTransportHonoursRuntime: the node-local broker transport uses the same
// configured command for exec and cp.
func TestCtrTransportHonoursRuntime(t *testing.T) {
	cases := []struct {
		name       string
		call       func(broker.Transport) error
		wantMethod string
		wantArgs   []string
	}{
		{
			name: "exec",
			call: func(tr broker.Transport) error {
				return tr.Run(context.Background(), config.Primary, "ls")
			},
			wantMethod: "Run",
			wantArgs:   withWrapper("exec", "solace", "ls"),
		},
		{
			name: "upload rides stdin",
			call: func(tr broker.Transport) error {
				return tr.Upload(context.Background(), config.Primary, []byte("body"), "/tmp/f")
			},
			wantMethod: "RunInput",
			wantArgs:   withWrapper("exec", "-i", "solace", "sh", "-c", "cat > '/tmp/f'"),
		},
		{
			name: "cp out",
			call: func(tr broker.Transport) error {
				return tr.Download(context.Background(), config.Primary, "/tmp/remote", "local")
			},
			wantMethod: "Run",
			wantArgs:   withWrapper("cp", "solace:/tmp/remote", "local"),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := &capRunner{}
			tr := NewTransport(rr, wrappedCtrCfg(), config.Docker)
			if err := tc.call(tr); err != nil {
				t.Fatalf("call: %v", err)
			}
			got := rr.last()
			if got.method != tc.wantMethod {
				t.Errorf("method = %q, want %q", got.method, tc.wantMethod)
			}
			if got.name != "lima" {
				t.Errorf("argv[0] = %q, want lima (from docker.runtime)", got.name)
			}
			if !eqArgs(got.args, tc.wantArgs) {
				t.Errorf("args = %v, want %v", got.args, tc.wantArgs)
			}
		})
	}
}

// TestCtrRuntimeDefaultArgvUnchanged: with a single-token runtime the argv is
// exactly what it was before runtime became a command, so no existing
// `+ docker ...` / `+ podman ...` assertion shifts.
func TestCtrRuntimeDefaultArgvUnchanged(t *testing.T) {
	for _, p := range []config.Platform{config.Docker, config.Podman} {
		m, rr, _ := newCapMgr(ctrCfg(p, "yes"), p)
		if err := m.Logs(context.Background()); err != nil {
			t.Fatalf("%s Logs: %v", p, err)
		}
		got := rr.last()
		if got.name != string(p) {
			t.Errorf("argv[0] = %q, want %q", got.name, p)
		}
		wantName := "solace"
		if p == config.Podman {
			wantName = "sol-pod"
		}
		if !eqArgs(got.args, []string{"logs", "-f", wantName}) {
			t.Errorf("%s args = %v, want [logs -f %s] with no prefix", p, got.args, wantName)
		}
	}
}
