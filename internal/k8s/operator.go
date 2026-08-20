package k8s

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"regexp"
	"strings"
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
	vars := operatorTmplVars{
		Namespace:      opNS,
		WatchNamespace: watchNamespace(cfg),
		Image:          operatorImage(cfg),
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

// operatorImage is the operator image reference a deploy will actually pull:
// Operator.Image with Image.Registry/ prefixed when set (010:2019). RenderOperator
// substitutes it into the bundle and CheckEnv reports it, both from this one definition,
// so the report cannot name an image the apply does not use. Image.Ref() is not the
// helper for this: it composes repo:tag, which Operator.Image already carries.
func operatorImage(cfg *config.Config) string {
	if cfg.Image.Registry != "" {
		return cfg.Image.Registry + "/" + cfg.K8s.Operator.Image
	}
	return cfg.K8s.Operator.Image
}

// watchNamespace builds the operator's WATCH_NAMESPACE value: the configured watch
// list with the broker namespace appended (comma-joined) when broker-ns watching is
// enabled -- the default (000-env.sh:85-89).
//
// Entries are trimmed and de-duplicated, first occurrence winning. The broker namespace
// is very often already in the configured list, and the repeat reached both the `check`
// report and the applied Deployment's WATCH_NAMESPACE; controller-runtime's cache is
// map-keyed, so it collapsed there harmlessly, which is exactly why it went unnoticed.
func watchNamespace(cfg *config.Config) string {
	var out []string
	seen := make(map[string]bool)
	add := func(ns string) {
		ns = strings.TrimSpace(ns)
		if ns == "" || seen[ns] {
			return
		}
		seen[ns] = true
		out = append(out, ns)
	}
	for _, ns := range strings.Split(cfg.K8s.Operator.WatchNamespaces, ",") {
		add(ns)
	}
	if cfg.K8s.Operator.WatchBrokerNSEnabled() {
		add(cfg.K8s.Namespace)
	}
	return strings.Join(out, ",")
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

// crdKindRE matches a document whose own `kind:` is CustomResourceDefinition. It is
// anchored to column 0 on purpose: a CRD carries an OpenAPI schema that contains
// nested `kind:` keys of its own, and only the document's top-level mapping keys sit
// unindented.
var crdKindRE = regexp.MustCompile(`(?m)^kind:[ \t]*CustomResourceDefinition[ \t]*$`)

// yamlDocSepRE matches the `---` line separating documents in the rendered bundle.
var yamlDocSepRE = regexp.MustCompile(`(?m)^---[ \t]*$`)

// splitOperatorBundle separates the CustomResourceDefinition documents from
// everything else in the rendered bundle. The two halves are deleted separately
// because they have very different blast radii: the Deployment and RBAC belong to
// this operator install, while the CRDs are the cluster-wide type definitions --
// removing them cascade-deletes every PubSubPlusEventBroker resource in the cluster,
// including brokers this env file has never heard of.
func splitOperatorBundle(manifest []byte) (crds, rest []byte) {
	var crdDocs, restDocs []string
	for _, doc := range yamlDocSepRE.Split(string(manifest), -1) {
		if strings.TrimSpace(doc) == "" {
			continue
		}
		if crdKindRE.MatchString(doc) {
			crdDocs = append(crdDocs, doc)
			continue
		}
		restDocs = append(restDocs, doc)
	}
	return joinYAMLDocs(crdDocs), joinYAMLDocs(restDocs)
}

// joinYAMLDocs reassembles documents into one multi-document manifest, or returns
// nil when there are none -- so a caller can test the result for emptiness rather
// than piping a lone separator into kubectl.
func joinYAMLDocs(docs []string) []byte {
	if len(docs) == 0 {
		return nil
	}
	var b strings.Builder
	for i, doc := range docs {
		if i > 0 {
			b.WriteString("\n---\n")
		}
		b.WriteString(strings.Trim(doc, "\n"))
		b.WriteString("\n")
	}
	return []byte(b.String())
}

// OperatorDelete removes the operator by deleting the rendered bundle on stdin with
// --ignore-not-found (110:2057) -- one mirrored path with OperatorApply, replacing the
// separately-maintained delete manifest the legacy 110 shipped.
//
// deleteCRDs is the same retained-layer decision `remove broker` makes about PVCs:
// the expensive-to-recreate, easy-to-regret part is kept unless it is asked for by
// name. Here that is the CRDs, whose removal takes every broker in the cluster with
// them. Either outcome is stated rather than left to be inferred.
func (c *Cluster) OperatorDelete(ctx context.Context, deleteCRDs bool) error {
	if err := c.Preflight(ctx, "delete", "customresourcedefinitions"); err != nil {
		return err
	}
	opNS := c.operatorNS(ctx)
	c.logf("deleting operator from namespace %s", opNS)
	manifest, err := RenderOperator(c.Cfg, opNS)
	if err != nil {
		return err
	}
	crds, rest := splitOperatorBundle(manifest)
	if len(rest) > 0 {
		if err := c.deleteStdin(ctx, rest); err != nil {
			return fmt.Errorf("delete operator bundle: %w", err)
		}
	}
	if !deleteCRDs {
		c.logf("operator CRDs kept -- existing broker resources are untouched " +
			"(pass --delete-crd to remove them)")
		return nil
	}
	if len(crds) == 0 {
		c.logf("operator CRDs: none in the bundle, nothing to delete")
		return nil
	}
	if err := c.deleteStdin(ctx, crds); err != nil {
		return fmt.Errorf("delete operator CRDs: %w", err)
	}
	c.logf("operator CRDs deleted -- every PubSubPlusEventBroker resource in the cluster went with them")
	return nil
}

// OperatorRestart bounces the controller without changing what is installed, for the
// case where the operator is wedged rather than out of date -- `deploy operator` is
// what re-applies a changed bundle.
func (c *Cluster) OperatorRestart(ctx context.Context) error {
	opNS := c.operatorNS(ctx)
	if err := c.Preflight(ctx, "patch", "deployments"); err != nil {
		return err
	}
	c.logf("restarting operator deployment %s in %s", operatorDeployment, opNS)
	return c.kubectl(ctx, "rollout", "restart", "deployment", operatorDeployment, "-n", opNS)
}

// OperatorInstalled reports whether the operator's CRD and controller Deployment are
// both present. It returns a plain bool rather than an error: every way of failing to
// find them -- absent, wrong namespace, cluster unreachable -- leads to the same
// advice, and the only caller uses this to decide whether to warn before a deploy
// that would otherwise fail confusingly. A false here is never fatal on its own.
func (c *Cluster) OperatorInstalled(ctx context.Context) bool {
	if _, err := c.output(ctx, "get", "crd", brokerResource); err != nil {
		return false
	}
	opNS := c.operatorNS(ctx)
	_, err := c.output(ctx, "get", "deployment", operatorDeployment, "-n", opNS)
	return err == nil
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
