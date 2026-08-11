package container

import (
	"context"
	"testing"

	"solace/internal/broker"
	"solace/internal/config"
)

// wrappedCtrCfg is the docker ctrCfg with a multi-token runtime. The bash
// container bootstrap expanded ${CONTAINER_RUNTIME} unquoted too, so a wrapper
// such as `sudo -n docker` has to reach exec as argv, not as one impossible
// filename.
func wrappedCtrCfg() *config.Config {
	cfg := ctrCfg(config.Docker, "yes")
	cfg.Docker.Runtime = config.Command{"sudo", "-n", "docker"}
	return cfg
}

// withSudo is the argv prefix wrappedCtrCfg contributes ahead of every subcommand.
func withSudo(rest ...string) []string {
	return append([]string{"-n", "docker"}, rest...)
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
			wantArgs:   withSudo("logs", "-f", "solace"),
		},
		{
			name:       "output",
			call:       func(m *Manager) error { return m.Reachable(context.Background()) },
			wantMethod: "Output",
			wantArgs:   withSudo("version"),
		},
		{
			name:       "cli",
			call:       func(m *Manager) error { return m.CLI(context.Background()) },
			wantMethod: "RunInteractive",
			wantArgs:   withSudo("exec", "-it", "solace", "cli", "-A"),
		},
		{
			name:       "shell",
			call:       func(m *Manager) error { return m.Shell(context.Background()) },
			wantMethod: "RunInteractive",
			wantArgs:   withSudo("exec", "-it", "solace", "bash"),
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
			if got.name != "sudo" {
				t.Errorf("argv[0] = %q, want sudo (from docker.runtime)", got.name)
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
			wantArgs:   withSudo("exec", "solace", "ls"),
		},
		{
			name: "upload rides stdin",
			call: func(tr broker.Transport) error {
				return tr.Upload(context.Background(), config.Primary, []byte("body"), "/tmp/f")
			},
			wantMethod: "RunInput",
			wantArgs:   withSudo("exec", "-i", "solace", "sh", "-c", "cat > '/tmp/f'"),
		},
		{
			name: "cp out",
			call: func(tr broker.Transport) error {
				return tr.Download(context.Background(), config.Primary, "/tmp/remote", "local")
			},
			wantMethod: "Run",
			wantArgs:   withSudo("cp", "solace:/tmp/remote", "local"),
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
			if got.name != "sudo" {
				t.Errorf("argv[0] = %q, want sudo (from docker.runtime)", got.name)
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
