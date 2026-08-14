package k8s

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"text/template"

	"solace/internal/config"
)

// operatorBundle is the full operator install manifest (CRDs, RBAC, controller
// Deployment) as a Go text/template with six substitution points. It is the Go port
// of the ~119 KB heredoc embedded in 010-deploy-operator.sh, applied on stdin.
//
//go:embed assets/operator-1.4.0.yaml.tmpl
var operatorBundle string

// operatorTmplVars are the six substitution points in operatorBundle. See
// RenderOperator for how each is derived from config.
type operatorTmplVars struct {
	Namespace      string // operator namespace (appears 6x in the bundle)
	WatchNamespace string // WATCH_NAMESPACE env value (watch list, broker ns appended)
	Image          string // operator image, registry-prefixed when Image.Registry is set
	CPU            string // manager container cpu limit
	Mem            string // manager container memory limit
	PullSecret     bool   // true -> emit the imagePullSecrets: regcred block
}

// RenderOperator renders the operator bundle for namespace opNS, porting the heredoc
// substitutions of 010-deploy-operator.sh: the operator image is prefixed with
// Image.Registry/ when set (010:2019); WATCH_NAMESPACE is Operator.WatchNamespaces with
// the broker namespace appended when broker-ns watching is enabled (000-env.sh:85-89);
// the imagePullSecrets block is emitted only when an image-pull secret is configured.
func RenderOperator(cfg *config.Config, opNS string) ([]byte, error) {
	op := cfg.K8s.Operator
	image := op.Image
	if cfg.Image.Registry != "" {
		image = cfg.Image.Registry + "/" + image
	}
	vars := operatorTmplVars{
		Namespace:      opNS,
		WatchNamespace: watchNamespace(cfg),
		Image:          image,
		CPU:            op.CPU,
		Mem:            op.Mem,
		PullSecret:     cfg.Image.PullSecret != "",
	}
	t, err := template.New("operator").Parse(operatorBundle)
	if err != nil {
		return nil, fmt.Errorf("parse operator bundle template: %w", err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, vars); err != nil {
		return nil, fmt.Errorf("render operator bundle: %w", err)
	}
	return buf.Bytes(), nil
}

// GenOperator renders the operator bundle for `gen operator` / `operator deploy --gen`
// without contacting the cluster: it uses the configured Operator.Namespace, falling
// back to the fixed default when unset (the running-deployment discovery of operatorNS
// needs a live cluster, so render-only cannot use it). It is the artifact-only
// counterpart to OperatorApply.
func GenOperator(cfg *config.Config) ([]byte, error) {
	opNS := cfg.K8s.Operator.Namespace
	if opNS == "" {
		opNS = defaultOperatorNS
	}
	return RenderOperator(cfg, opNS)
}

// watchNamespace builds the operator's WATCH_NAMESPACE value: the configured watch
// list with the broker namespace appended (comma-joined) when broker-ns watching is
// enabled -- the default (000-env.sh:85-89).
func watchNamespace(cfg *config.Config) string {
	watch := cfg.K8s.Operator.WatchNamespaces
	if cfg.K8s.Operator.WatchBrokerNSEnabled() {
		if watch != "" {
			watch += ","
		}
		watch += cfg.K8s.Namespace
	}
	return watch
}

// OperatorApply installs the operator: it resolves the operator namespace, applies the
// image-pull secret (regcred) into that namespace first when pull creds are configured
// (010:29), then applies the rendered bundle on stdin (010:2063).
func (c *Cluster) OperatorApply(ctx context.Context) error {
	// The bundle is cluster-scoped (CRDs, ClusterRoles), so the permission that
	// matters is the one an under-privileged context most often lacks -- and
	// discovering that halfway through a multi-document apply is the worst case.
	if err := c.Preflight(ctx, "create", "customresourcedefinitions"); err != nil {
		return err
	}
	opNS := c.operatorNS(ctx)
	if c.Cfg.Image.PullSecret != "" {
		c.logf("applying operator image-pull secret regcred in %s", opNS)
		regcred, err := operatorRegcred(c.Cfg, opNS)
		if err != nil {
			return fmt.Errorf("build operator regcred: %w", err)
		}
		if err := c.apply(ctx, regcred); err != nil {
			return fmt.Errorf("apply operator regcred: %w", err)
		}
	}
	c.logf("deploying operator to namespace %s", opNS)
	manifest, err := RenderOperator(c.Cfg, opNS)
	if err != nil {
		return err
	}
	if err := c.apply(ctx, manifest); err != nil {
		return fmt.Errorf("apply operator bundle: %w", err)
	}
	return nil
}

// OperatorDelete removes the operator by deleting the rendered bundle on stdin with
// --ignore-not-found (110:2057) -- one mirrored path with OperatorApply, replacing the
// separately-maintained delete manifest the legacy 110 shipped.
func (c *Cluster) OperatorDelete(ctx context.Context) error {
	if err := c.Preflight(ctx, "delete", "customresourcedefinitions"); err != nil {
		return err
	}
	opNS := c.operatorNS(ctx)
	c.logf("deleting operator from namespace %s", opNS)
	manifest, err := RenderOperator(c.Cfg, opNS)
	if err != nil {
		return err
	}
	if err := c.deleteStdin(ctx, manifest); err != nil {
		return fmt.Errorf("delete operator bundle: %w", err)
	}
	return nil
}

// OperatorStatus prints the operator Deployment and its controller pods in the
// operator namespace.
func (c *Cluster) OperatorStatus(ctx context.Context) error {
	opNS := c.operatorNS(ctx)
	if err := c.kubectl(ctx, "get", "deployment", operatorDeployment, "-n", opNS, "-o", "wide"); err != nil {
		return err
	}
	return c.kubectl(ctx, "get", "pods", "-n", opNS, "-l", "control-plane=controller-manager", "-o", "wide")
}

// OperatorLogs streams the operator manager logs; passthrough args (e.g. -f, --tail)
// are forwarded verbatim.
func (c *Cluster) OperatorLogs(ctx context.Context, passthrough ...string) error {
	opNS := c.operatorNS(ctx)
	args := append([]string{"logs", "-n", opNS, "deployment/" + operatorDeployment}, passthrough...)
	return c.kubectl(ctx, args...)
}

// OperatorDescribe describes the operator Deployment in the operator namespace.
func (c *Cluster) OperatorDescribe(ctx context.Context) error {
	opNS := c.operatorNS(ctx)
	return c.kubectl(ctx, "describe", "deployment/"+operatorDeployment, "-n", opNS)
}
