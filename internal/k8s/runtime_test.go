package k8s

import (
	"context"
	"testing"

	"solace/internal/broker"
	"solace/internal/config"
)

// wrappedCfg is haCfg with a multi-token k8s.runtime -- the shape the bash KUBE
// variable carried (bash/env/customer-sample:7 set a whole kubectl profile).
func wrappedCfg() *config.Config {
	cfg := haCfg()
	cfg.K8s.Runtime = config.Command{"microk8s", "kubectl", "--kubeconfig", "/tmp/kc"}
	return cfg
}

// withLeading is the argv prefix wrappedCfg contributes ahead of every subcommand:
// everything after argv[0].
func withLeading(rest ...string) []string {
	return append([]string{"kubectl", "--kubeconfig", "/tmp/kc"}, rest...)
}

// TestClusterHonoursRuntime: every Cluster helper must run argv[0] from
// k8s.runtime and place its leading arguments ahead of the subcommand.
func TestClusterHonoursRuntime(t *testing.T) {
	cases := []struct {
		name       string
		call       func(*Cluster) error
		wantMethod string
		wantArgs   []string
	}{
		{
			name:       "kubectl",
			call:       func(c *Cluster) error { return c.kubectl(context.Background(), "get", "pods") },
			wantMethod: "Run",
			wantArgs:   withLeading("get", "pods"),
		},
		{
			name:       "apply",
			call:       func(c *Cluster) error { return c.apply(context.Background(), []byte("manifest")) },
			wantMethod: "RunInput",
			wantArgs:   withLeading("apply", "-f", "-"),
		},
		{
			name:       "deleteStdin",
			call:       func(c *Cluster) error { return c.deleteStdin(context.Background(), []byte("manifest")) },
			wantMethod: "RunInput",
			wantArgs:   withLeading("delete", "-f", "-", "--ignore-not-found"),
		},
		{
			name: "output",
			call: func(c *Cluster) error {
				_, err := c.output(context.Background(), "get", "sc")
				return err
			},
			wantMethod: "Output",
			wantArgs:   withLeading("get", "sc"),
		},
		{
			name:       "interactiveExec",
			call:       func(c *Cluster) error { return c.CLI(context.Background(), config.Primary) },
			wantMethod: "RunInteractive",
			wantArgs: withLeading("exec", "-it", "-n", "solace",
				"dev-broker-pubsubplus-p-0", "--", "cli", "-A"),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := &recRunner{}
			c := NewCluster(rr, wrappedCfg(), nil, nil)
			if err := tc.call(c); err != nil {
				t.Fatalf("call: %v", err)
			}
			got := rr.last()
			if got.method != tc.wantMethod {
				t.Errorf("method = %q, want %q", got.method, tc.wantMethod)
			}
			if got.name != "microk8s" {
				t.Errorf("argv[0] = %q, want microk8s (from k8s.runtime)", got.name)
			}
			if !eqArgs(got.args, tc.wantArgs) {
				t.Errorf("args = %v, want %v", got.args, tc.wantArgs)
			}
		})
	}
}

// TestTransportHonoursRuntime: the broker transport shells out through the same
// configured command, including the `cp` paths that build argv themselves.
func TestTransportHonoursRuntime(t *testing.T) {
	const pod = "dev-broker-pubsubplus-p-0"
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
			wantArgs:   withLeading("exec", "-n", "solace", pod, "--", "ls"),
		},
		{
			name: "exec with stdin",
			call: func(tr broker.Transport) error {
				_, err := tr.OutputInput(context.Background(), config.Primary, []byte("in"), "cat")
				return err
			},
			wantMethod: "OutputInput",
			wantArgs:   withLeading("exec", "-i", "-n", "solace", pod, "--", "cat"),
		},
		{
			name: "upload rides stdin",
			call: func(tr broker.Transport) error {
				return tr.Upload(context.Background(), config.Primary, []byte("body"), "/tmp/f")
			},
			wantMethod: "RunInput",
			wantArgs:   withLeading("exec", "-i", "-n", "solace", pod, "--", "sh", "-c", "cat > '/tmp/f'"),
		},
		{
			name: "cp in",
			call: func(tr broker.Transport) error {
				return tr.UploadFile(context.Background(), config.Primary, "local", "/tmp/dest")
			},
			wantMethod: "Run",
			wantArgs:   withLeading("cp", "-n", "solace", "local", pod+":/tmp/dest"),
		},
		{
			name: "cp out",
			call: func(tr broker.Transport) error {
				return tr.Download(context.Background(), config.Primary, "/tmp/remote", "local")
			},
			wantMethod: "Run",
			wantArgs:   withLeading("cp", "-n", "solace", pod+":/tmp/remote", "local"),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := &recRunner{}
			if err := tc.call(NewTransport(rr, wrappedCfg())); err != nil {
				t.Fatalf("call: %v", err)
			}
			got := rr.last()
			if got.method != tc.wantMethod {
				t.Errorf("method = %q, want %q", got.method, tc.wantMethod)
			}
			if got.name != "microk8s" {
				t.Errorf("argv[0] = %q, want microk8s (from k8s.runtime)", got.name)
			}
			if !eqArgs(got.args, tc.wantArgs) {
				t.Errorf("args = %v, want %v", got.args, tc.wantArgs)
			}
		})
	}
}

// TestRuntimeDefaultArgvUnchanged is the regression guard for the whole change:
// with the default runtime the argv must be byte-identical to what the hardcoded
// kubectl constant produced, so no existing dry-run assertion shifts.
func TestRuntimeDefaultArgvUnchanged(t *testing.T) {
	cfg := haCfg() // Runtime is config.Command{"kubectl"} -- no leading args
	rr := &recRunner{}
	c := NewCluster(rr, cfg, nil, nil)
	if err := c.kubectl(context.Background(), "get", "pods"); err != nil {
		t.Fatalf("kubectl: %v", err)
	}
	got := rr.last()
	if got.name != "kubectl" {
		t.Errorf("argv[0] = %q, want kubectl", got.name)
	}
	if !eqArgs(got.args, []string{"get", "pods"}) {
		t.Errorf("args = %v, want [get pods] with no prefix", got.args)
	}
}
