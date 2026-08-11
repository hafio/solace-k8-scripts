package k8s

import (
	"context"
	"io"
	"strings"

	"solace/internal/config"
	"solace/internal/engine"
)

// defaultOperatorNS is where the operator lands when no namespace is configured and
// none is discovered running on the cluster (000-env.sh:83).
const defaultOperatorNS = "pubsubplus-operator-system"

// operatorDeployment is the fixed name of the operator's controller Deployment and
// ServiceAccount (assets/operator-1.4.0.yaml.tmpl:1971). Its substring is also what
// the namespace-discovery grep matches (000-env.sh:76).
const operatorDeployment = "pubsubplus-eventbroker-operator"

// Cluster performs Kubernetes operations that talk to the cluster or the operator --
// as opposed to a running broker, which goes through internal/broker over the
// transport. Every command routes through R, so --dry-run echoes it and tests capture
// the exact argv. Out is the report sink; In is the prompt source for the few
// interactive operations (node labelling).
type Cluster struct {
	R   engine.Runner
	Cfg *config.Config
	Log func(string, ...any)
	Out io.Writer
	In  io.Reader
}

// NewCluster builds a Cluster over the given runner, config, step logger and output
// sink. In is left nil; callers that prompt (LabelNodes) set it explicitly.
func NewCluster(r engine.Runner, cfg *config.Config, log func(string, ...any), out io.Writer) *Cluster {
	return &Cluster{R: r, Cfg: cfg, Log: log, Out: out}
}

// logf emits a progress line via the injected step logger, if any.
func (c *Cluster) logf(format string, a ...any) {
	if c.Log != nil {
		c.Log(format, a...)
	}
}

// ns is the broker namespace.
func (c *Cluster) ns() string { return c.Cfg.K8s.Namespace }

// cmd is the configured cluster CLI (k8s.runtime, default `kubectl`): argv[0]
// plus any leading arguments that precede every call's own. Ported from the bash
// KUBE variable, which the scripts expanded unquoted so it could carry a whole
// profile (`kubectl --kubeconfig <file>`), not just a binary name.
func (c *Cluster) cmd() config.Command { return c.Cfg.K8s.Runtime }

// kubectl runs `kubectl args...`, streaming stdout/stderr.
func (c *Cluster) kubectl(ctx context.Context, args ...string) error {
	k := c.cmd()
	return c.R.Run(ctx, k.Name(), k.Args(args...)...)
}

// apply pipes a rendered manifest to `kubectl apply -f -` on stdin (never a temp
// file, so secret-bearing manifests stay off disk -- §3).
func (c *Cluster) apply(ctx context.Context, manifest []byte) error {
	k := c.cmd()
	return c.R.RunInput(ctx, manifest, k.Name(), k.Args("apply", "-f", "-")...)
}

// deleteStdin pipes a rendered manifest to `kubectl delete -f - --ignore-not-found`,
// so teardown mirrors apply through one code path and is idempotent.
func (c *Cluster) deleteStdin(ctx context.Context, manifest []byte) error {
	k := c.cmd()
	return c.R.RunInput(ctx, manifest, k.Name(), k.Args("delete", "-f", "-", "--ignore-not-found")...)
}

// output runs `kubectl args...` and returns captured stdout.
func (c *Cluster) output(ctx context.Context, args ...string) ([]byte, error) {
	k := c.cmd()
	return c.R.Output(ctx, k.Name(), k.Args(args...)...)
}

// operatorNS resolves the namespace the operator runs in: the configured
// Operator.Namespace if set; otherwise the namespace of an operator Deployment
// discovered on the cluster; otherwise the fixed default. Mirrors 000-env.sh:73-83.
// Under --dry-run the discovery output is empty, so it falls through to the default --
// safe, because the rendered manifests already carry the namespace.
func (c *Cluster) operatorNS(ctx context.Context) string {
	if c.Cfg.K8s.Operator.Namespace != "" {
		return c.Cfg.K8s.Operator.Namespace
	}
	if ns := c.discoverOperatorNS(ctx); ns != "" {
		return ns
	}
	return defaultOperatorNS
}

// discoverOperatorNS greps `get deployment --all-namespaces` for the operator's
// deployment and returns its namespace (the first custom-column). Returns "" if the
// lookup fails or the operator is not found -- both mean "not installed", which the
// caller's default covers, so the error is deliberately swallowed (000-env.sh:76).
func (c *Cluster) discoverOperatorNS(ctx context.Context) string {
	out, err := c.output(ctx, "get", "deployment", "--all-namespaces",
		"-o", "custom-columns=NS:.metadata.namespace,NAME:.metadata.name")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, operatorDeployment) {
			continue
		}
		if f := strings.Fields(line); len(f) > 0 {
			return f[0]
		}
	}
	return ""
}
