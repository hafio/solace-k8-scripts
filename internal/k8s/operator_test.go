package k8s

import (
	"context"
	"strings"
	"testing"
)

func boolPtr(b bool) *bool { return &b }

// TestWatchNamespace pins the WATCH_NAMESPACE join logic (000-env.sh:85-89): the
// broker namespace is appended by default, comma-joined onto any configured list, and
// omitted when broker-ns watching is explicitly disabled.
//
// The dedupe cases are the fix for a report and a manifest that both listed the broker
// namespace twice whenever the configured list already named it (the common case, since
// watchBrokerNs defaults on). controller-runtime's map-keyed cache collapsed the repeat,
// so only the printed and applied text was ever wrong -- which is precisely what makes a
// regression here invisible without these cases.
func TestWatchNamespace(t *testing.T) {
	cases := []struct {
		name     string
		list     string
		watchBrk *bool
		want     string
	}{
		{"default appends broker ns", "", nil, "solace"},
		{"appends onto configured list", "ns-a,ns-b", nil, "ns-a,ns-b,solace"},
		{"disabled keeps only the list", "ns-a,ns-b", boolPtr(false), "ns-a,ns-b"},
		{"disabled with empty list is empty", "", boolPtr(false), ""},
		{"broker ns already listed is not repeated", "ns-a,solace", nil, "ns-a,solace"},
		{"repeat inside the list is dropped", "ns-a,ns-b,ns-a", nil, "ns-a,ns-b,solace"},
		{"entries are trimmed, empties dropped", " ns-a , ,ns-b,", nil, "ns-a,ns-b,solace"},
		{"disabled dedupes the list too", "solace,ns-b,solace", boolPtr(false), "solace,ns-b"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := haCfg() // Namespace = solace
			cfg.K8s.Operator.WatchNamespaces = tc.list
			cfg.K8s.Operator.WatchBrokerNS = tc.watchBrk
			if got := watchNamespace(cfg); got != tc.want {
				t.Errorf("watchNamespace = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestOperatorImage pins the registry-prefix rule now that RenderOperator and CheckEnv
// share it: the report drifted from the apply for exactly as long as each had its own
// idea of the reference, so the helper's two branches are worth their own test rather
// than only being reached through the 119 KB bundle render.
func TestOperatorImage(t *testing.T) {
	cfg := haCfg()
	cfg.K8s.Operator.Image = "solace/pubsubplus-eventbroker-operator:1.4.0"
	cfg.Image.Registry = "registry.example.com"
	if got, want := operatorImage(cfg), "registry.example.com/solace/pubsubplus-eventbroker-operator:1.4.0"; got != want {
		t.Errorf("operatorImage with a registry = %q, want %q", got, want)
	}
	cfg.Image.Registry = ""
	if got, want := operatorImage(cfg), "solace/pubsubplus-eventbroker-operator:1.4.0"; got != want {
		t.Errorf("operatorImage without a registry = %q, want %q", got, want)
	}
}

// TestRenderOperatorSubstitutions renders the bundle for known inputs and asserts each
// of the six substitution points landed, and that no template marker survives. The
// bundle is ~119 KB of otherwise-static CRD/RBAC YAML, so pinning the varying points is
// the reviewable check (vs. a full-bundle golden).
func TestRenderOperatorSubstitutions(t *testing.T) {
	t.Run("registry prefix, pull secret, broker-ns watch", func(t *testing.T) {
		cfg := loadK8s(t) // sample: registry set, pullSecret set, watch defaults on
		out, err := RenderOperator(cfg, "op-ns")
		if err != nil {
			t.Fatalf("RenderOperator: %v", err)
		}
		s := string(out)
		mustContain(t, s, "  name: op-ns\n")
		mustContain(t, s, "  namespace: op-ns\n")
		mustContain(t, s, `value: "solace"`) // WATCH_NAMESPACE = broker ns
		mustContain(t, s, "image: registry.example.com/docker.io/solace/pubsubplus-eventbroker-operator:1.4.0")
		mustContain(t, s, "cpu: 500m")
		mustContain(t, s, "memory: 512Mi")
		mustContain(t, s, "imagePullSecrets:")
		mustContain(t, s, "- name: regcred")
		if strings.Contains(s, "{{") {
			t.Error("rendered bundle still contains an unresolved template marker {{")
		}
	})

	t.Run("no registry, no pull secret, watch list without broker ns", func(t *testing.T) {
		cfg := loadK8s(t)
		cfg.Image.Registry = ""
		cfg.Image.PullSecret = ""
		cfg.K8s.Operator.WatchNamespaces = "team-a"
		cfg.K8s.Operator.WatchBrokerNS = boolPtr(false)
		out, err := RenderOperator(cfg, "op-ns")
		if err != nil {
			t.Fatalf("RenderOperator: %v", err)
		}
		s := string(out)
		mustContain(t, s, "image: docker.io/solace/pubsubplus-eventbroker-operator:1.4.0")
		mustContain(t, s, `value: "team-a"`)
		if strings.Contains(s, "imagePullSecrets:") {
			t.Error("imagePullSecrets block must be omitted when no pull secret is configured")
		}
		if strings.Contains(s, "- name: regcred") {
			t.Error("regcred reference must be omitted when no pull secret is configured")
		}
	})
}

// TestGenOperator covers the render-only path (`gen operator` / `operator deploy
// --gen`): it uses the configured operator namespace when set, and falls back to the
// fixed default when unset (render-only cannot discover the running deployment). It is
// otherwise RenderOperator, so a spot-check of the namespace substitution suffices.
func TestGenOperator(t *testing.T) {
	t.Run("uses configured operator namespace", func(t *testing.T) {
		cfg := loadK8s(t)
		cfg.K8s.Operator.Namespace = "my-op-ns"
		out, err := GenOperator(cfg)
		if err != nil {
			t.Fatalf("GenOperator: %v", err)
		}
		mustContain(t, string(out), "  namespace: my-op-ns\n")
	})
	t.Run("falls back to the default when unset", func(t *testing.T) {
		cfg := loadK8s(t)
		cfg.K8s.Operator.Namespace = ""
		out, err := GenOperator(cfg)
		if err != nil {
			t.Fatalf("GenOperator: %v", err)
		}
		mustContain(t, string(out), "  namespace: "+defaultOperatorNS+"\n")
	})
}

// TestOperatorApply asserts the apply sequence: regcred first (into the operator ns),
// then the bundle, both on stdin via `apply -f -`.
func TestOperatorApply(t *testing.T) {
	cfg := loadK8s(t) // pull secret set -> regcred applied
	cfg.K8s.Operator.Namespace = "op-ns"
	rr := &recRunner{}
	c := NewCluster(rr, cfg, nil, nil)
	if err := c.OperatorApply(context.Background()); err != nil {
		t.Fatalf("OperatorApply: %v", err)
	}
	calls := rr.afterPreflight(t, "create", "customresourcedefinitions")
	if len(calls) != 2 {
		t.Fatalf("OperatorApply made %d calls after the probe, want 2 (regcred + bundle)", len(calls))
	}
	reg, bundle := calls[0], calls[1]
	for _, call := range []rrCall{reg, bundle} {
		if call.method != "RunInput" || call.name != "kubectl" || !eqArgs(call.args, []string{"apply", "-f", "-"}) {
			t.Errorf("apply call = %+v, want RunInput kubectl [apply -f -]", call)
		}
	}
	if !strings.Contains(reg.stdin, ".dockerconfigjson") || !strings.Contains(reg.stdin, "name: regcred") ||
		!strings.Contains(reg.stdin, "namespace: op-ns") {
		t.Errorf("first apply should be the regcred secret in op-ns:\n%s", reg.stdin)
	}
	if !strings.Contains(bundle.stdin, "kind: Deployment") || !strings.Contains(bundle.stdin, "name: pubsubplus-eventbroker-operator") {
		t.Error("second apply should be the operator bundle")
	}
	if strings.Contains(bundle.stdin, "{{") {
		t.Error("applied bundle still contains an unresolved template marker")
	}
}

// TestOperatorApplyNoPullSecret covers the branch where no image-pull secret is
// configured: only the bundle is applied, no regcred.
func TestOperatorApplyNoPullSecret(t *testing.T) {
	cfg := loadK8s(t)
	cfg.K8s.Operator.Namespace = "op-ns"
	cfg.Image.PullSecret = ""
	rr := &recRunner{}
	c := NewCluster(rr, cfg, nil, nil)
	if err := c.OperatorApply(context.Background()); err != nil {
		t.Fatalf("OperatorApply: %v", err)
	}
	if calls := rr.afterPreflight(t, "create", "customresourcedefinitions"); len(calls) != 1 {
		t.Fatalf("OperatorApply without pull secret made %d calls after the probe, want 1 (bundle only)", len(calls))
	}
}

// TestOperatorDelete asserts teardown with deleteCRDs=false deletes only the
// non-CRD documents (Deployment, RBAC, ...) on stdin with --ignore-not-found, and
// leaves the CRDs out of the piped manifest entirely -- deleting them would
// cascade-delete every PubSubPlusEventBroker resource in the cluster, including
// brokers this env file has never heard of.
func TestOperatorDelete(t *testing.T) {
	cfg := loadK8s(t)
	cfg.K8s.Operator.Namespace = "op-ns"
	rr := &recRunner{}
	c := NewCluster(rr, cfg, nil, nil)
	if err := c.OperatorDelete(context.Background(), false); err != nil {
		t.Fatalf("OperatorDelete: %v", err)
	}
	calls := rr.afterPreflight(t, "delete", "customresourcedefinitions")
	if len(calls) != 1 {
		t.Fatalf("OperatorDelete(deleteCRDs=false) made %d call(s) after the probe, want 1 (the non-CRD documents)", len(calls))
	}
	got := calls[0]
	if got.method != "RunInput" || got.name != "kubectl" ||
		!eqArgs(got.args, []string{"delete", "-f", "-", "--ignore-not-found"}) {
		t.Errorf("OperatorDelete\n got: %+v\nwant RunInput kubectl [delete -f - --ignore-not-found]", got)
	}
	if !strings.Contains(got.stdin, "kind: Deployment") {
		t.Error("delete should still pipe the operator Deployment/RBAC documents")
	}
	if strings.Contains(got.stdin, "kind: CustomResourceDefinition") {
		t.Error("deleteCRDs=false must not include any CustomResourceDefinition document in the piped manifest")
	}
}

// TestOperatorDeleteWithCRDs is the deleteCRDs=true half: the CRD document is
// deleted in a SECOND, separate `kubectl delete -f -` call, never folded into the
// first -- so a caller inspecting just the first call (as most teardowns do)
// cannot mistake a CRD-carrying delete for the safe default.
func TestOperatorDeleteWithCRDs(t *testing.T) {
	cfg := loadK8s(t)
	cfg.K8s.Operator.Namespace = "op-ns"
	rr := &recRunner{}
	c := NewCluster(rr, cfg, nil, nil)
	if err := c.OperatorDelete(context.Background(), true); err != nil {
		t.Fatalf("OperatorDelete: %v", err)
	}
	calls := rr.afterPreflight(t, "delete", "customresourcedefinitions")
	if len(calls) != 2 {
		t.Fatalf("OperatorDelete(deleteCRDs=true) made %d call(s) after the probe, want 2 (non-CRD documents, then CRDs)", len(calls))
	}
	rest, crds := calls[0], calls[1]
	for _, call := range []rrCall{rest, crds} {
		if call.method != "RunInput" || call.name != "kubectl" ||
			!eqArgs(call.args, []string{"delete", "-f", "-", "--ignore-not-found"}) {
			t.Errorf("delete call = %+v, want RunInput kubectl [delete -f - --ignore-not-found]", call)
		}
	}
	if !strings.Contains(rest.stdin, "kind: Deployment") || strings.Contains(rest.stdin, "kind: CustomResourceDefinition") {
		t.Errorf("first delete should carry the Deployment/RBAC documents and no CRD:\n%s", rest.stdin)
	}
	if !strings.Contains(crds.stdin, "kind: CustomResourceDefinition") || strings.Contains(crds.stdin, "kind: Deployment") {
		t.Errorf("second delete should carry only the CRD document, on its own:\n%s", crds.stdin)
	}
}

// TestOperatorLogsArgs checks the log passthrough argv against the operator deployment.
func TestOperatorLogsArgs(t *testing.T) {
	cfg := loadK8s(t)
	cfg.K8s.Operator.Namespace = "op-ns"
	rr := &recRunner{}
	c := NewCluster(rr, cfg, nil, nil)
	if err := c.OperatorLogs(context.Background(), "-f", "--tail=10"); err != nil {
		t.Fatalf("OperatorLogs: %v", err)
	}
	got := rr.last()
	want := []string{"logs", "-n", "op-ns", "deployment/pubsubplus-eventbroker-operator", "-f", "--tail=10"}
	if got.method != "Run" || got.name != "kubectl" || !eqArgs(got.args, want) {
		t.Errorf("OperatorLogs\n got: %+v\nwant Run kubectl %v", got, want)
	}
}

// TestOperatorStatus covers a completely untested function (0.0%): the two kubectl
// gets (deployment wide, then controller pods by label) plus the early return when
// the first fails. Same one-liner-wrapper shape the suite already covers for every
// sibling (TestOperatorLogsArgs, TestOperatorDescribe); a typo in the label selector
// or resource name would otherwise ship silently.
func TestOperatorStatus(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cfg := loadK8s(t)
		cfg.K8s.Operator.Namespace = "op-ns"
		rr := &recRunner{}
		c := NewCluster(rr, cfg, nil, nil)
		if err := c.OperatorStatus(context.Background()); err != nil {
			t.Fatalf("OperatorStatus: %v", err)
		}
		if len(rr.calls) != 2 {
			t.Fatalf("OperatorStatus made %d calls, want 2 (deployment + pods)", len(rr.calls))
		}
		wantDeploy := []string{"get", "deployment", "pubsubplus-eventbroker-operator", "-n", "op-ns", "-o", "wide"}
		if got := rr.calls[0]; got.method != "Run" || !eqArgs(got.args, wantDeploy) {
			t.Errorf("deployment get argv = %+v, want Run kubectl %v", got, wantDeploy)
		}
		wantPods := []string{"get", "pods", "-n", "op-ns", "-l", "control-plane=controller-manager", "-o", "wide"}
		if got := rr.calls[1]; got.method != "Run" || !eqArgs(got.args, wantPods) {
			t.Errorf("pods get argv = %+v, want Run kubectl %v", got, wantPods)
		}
	})
	t.Run("stops after the deployment get fails", func(t *testing.T) {
		cfg := loadK8s(t)
		cfg.K8s.Operator.Namespace = "op-ns"
		rr := &recRunner{runErr: errFake}
		c := NewCluster(rr, cfg, nil, nil)
		if err := c.OperatorStatus(context.Background()); err == nil {
			t.Error("OperatorStatus should fail when the deployment get fails")
		}
		if len(rr.calls) != 1 {
			t.Errorf("OperatorStatus should stop after the first failing get; got %d calls", len(rr.calls))
		}
	})
}

// TestOperatorDescribe covers another completely untested function (0.0%);
// identical shape to TestOperatorLogsArgs already in the suite.
func TestOperatorDescribe(t *testing.T) {
	cfg := loadK8s(t)
	cfg.K8s.Operator.Namespace = "op-ns"
	rr := &recRunner{}
	c := NewCluster(rr, cfg, nil, nil)
	if err := c.OperatorDescribe(context.Background()); err != nil {
		t.Fatalf("OperatorDescribe: %v", err)
	}
	got := rr.last()
	want := []string{"describe", "deployment/pubsubplus-eventbroker-operator", "-n", "op-ns"}
	if got.method != "Run" || got.name != "kubectl" || !eqArgs(got.args, want) {
		t.Errorf("OperatorDescribe\n got: %+v\nwant Run kubectl %v", got, want)
	}
}

// TestSplitOperatorBundle pins the column-0 anchoring OperatorDelete's safety
// depends on, plus the edge shapes joinYAMLDocs must hand back as nil rather than
// a lone "---" separator.
func TestSplitOperatorBundle(t *testing.T) {
	t.Run("indented kind inside a nested block is not misclassified", func(t *testing.T) {
		// The ConfigMap embeds an indented "kind: CustomResourceDefinition" line
		// inside a literal block (an example snippet in its data), which must not
		// make this document count as a CRD -- only a document's OWN top-level
		// kind may. The second document is a real, unindented CRD kind, to prove
		// the anchor still catches the case it exists for.
		manifest := []byte(
			"apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: cm\n" +
				"data:\n" +
				"  example.yaml: |\n" +
				"    kind: CustomResourceDefinition\n" +
				"    metadata:\n" +
				"      name: widgets.example.com\n" +
				"---\n" +
				"apiVersion: apiextensions.k8s.io/v1\n" +
				"kind: CustomResourceDefinition\n" +
				"metadata:\n  name: real-crd.example.com\n")
		crds, rest := splitOperatorBundle(manifest)
		if !strings.Contains(string(rest), "kind: ConfigMap") {
			t.Errorf("the ConfigMap document should land in rest:\n%s", rest)
		}
		if strings.Contains(string(rest), "real-crd.example.com") {
			t.Errorf("the real CRD document must not also land in rest:\n%s", rest)
		}
		if !strings.Contains(string(crds), "real-crd.example.com") {
			t.Errorf("the real (unindented) CRD document should land in crds:\n%s", crds)
		}
		if strings.Contains(string(crds), "example.yaml") {
			t.Errorf("a ConfigMap with an indented kind: CustomResourceDefinition text must not be misclassified as a CRD:\n%s", crds)
		}
	})
	t.Run("empty input", func(t *testing.T) {
		crds, rest := splitOperatorBundle([]byte(""))
		if crds != nil || rest != nil {
			t.Errorf("splitOperatorBundle(\"\") = (%q, %q), want (nil, nil)", crds, rest)
		}
	})
	t.Run("no CRD documents", func(t *testing.T) {
		manifest := []byte("kind: ConfigMap\n---\nkind: Secret\n")
		crds, rest := splitOperatorBundle(manifest)
		if crds != nil {
			t.Errorf("crds = %q, want nil when the bundle has no CustomResourceDefinition", crds)
		}
		if !strings.Contains(string(rest), "kind: ConfigMap") || !strings.Contains(string(rest), "kind: Secret") {
			t.Errorf("rest should carry both documents:\n%s", rest)
		}
	})
	t.Run("CRD-only input", func(t *testing.T) {
		manifest := []byte("kind: CustomResourceDefinition\nmetadata:\n  name: widgets.example.com\n")
		crds, rest := splitOperatorBundle(manifest)
		if rest != nil {
			t.Errorf("rest = %q, want nil when every document is a CRD", rest)
		}
		if !strings.Contains(string(crds), "widgets.example.com") {
			t.Errorf("crds should carry the CRD document:\n%s", crds)
		}
	})
}

// TestOperatorRestart pins the rollout-restart argv against the resolved operator
// namespace, behind the same permission probe every mutating operator command runs.
func TestOperatorRestart(t *testing.T) {
	cfg := loadK8s(t)
	cfg.K8s.Operator.Namespace = "op-ns"
	rr := &recRunner{}
	c := NewCluster(rr, cfg, nil, nil)
	if err := c.OperatorRestart(context.Background()); err != nil {
		t.Fatalf("OperatorRestart: %v", err)
	}
	calls := rr.afterPreflight(t, "patch", "deployments")
	if len(calls) != 1 {
		t.Fatalf("OperatorRestart made %d call(s) after the probe, want 1", len(calls))
	}
	got := calls[0]
	want := []string{"rollout", "restart", "deployment", operatorDeployment, "-n", "op-ns"}
	if got.method != "Run" || got.name != "kubectl" || !eqArgs(got.args, want) {
		t.Errorf("OperatorRestart\n got: %+v\nwant Run kubectl %v", got, want)
	}
}

// TestOperatorInstalled covers the read-only probe: it reports true only when
// BOTH the CRD and the controller Deployment are visible, and it stops after the
// first failing get rather than probing the second for nothing -- and, per its
// signature, never returns an error: an absent operator and an unreachable
// cluster are both just "false" to the caller.
func TestOperatorInstalled(t *testing.T) {
	t.Run("both gets succeed", func(t *testing.T) {
		cfg := loadK8s(t)
		cfg.K8s.Operator.Namespace = "op-ns"
		rr := &recRunner{}
		c := NewCluster(rr, cfg, nil, nil)
		if !c.OperatorInstalled(context.Background()) {
			t.Error("OperatorInstalled should report true when both gets succeed")
		}
		if len(rr.calls) != 2 {
			t.Fatalf("OperatorInstalled made %d calls, want 2 (crd, then deployment)", len(rr.calls))
		}
		wantCRD := []string{"get", "crd", brokerResource}
		if got := rr.calls[0]; got.method != "Output" || !eqArgs(got.args, wantCRD) {
			t.Errorf("crd get argv = %+v, want Output kubectl %v", got, wantCRD)
		}
		wantDeploy := []string{"get", "deployment", operatorDeployment, "-n", "op-ns"}
		if got := rr.calls[1]; got.method != "Output" || !eqArgs(got.args, wantDeploy) {
			t.Errorf("deployment get argv = %+v, want Output kubectl %v", got, wantDeploy)
		}
	})
	t.Run("CRD missing", func(t *testing.T) {
		cfg := loadK8s(t)
		cfg.K8s.Operator.Namespace = "op-ns"
		rr := &recRunner{outErr: errFake}
		c := NewCluster(rr, cfg, nil, nil)
		if c.OperatorInstalled(context.Background()) {
			t.Error("OperatorInstalled should report false when the crd get fails")
		}
		if len(rr.calls) != 1 {
			t.Errorf("OperatorInstalled should stop after the failing crd get; got %d calls", len(rr.calls))
		}
	})
	t.Run("deployment missing", func(t *testing.T) {
		cfg := loadK8s(t)
		cfg.K8s.Operator.Namespace = "op-ns"
		rr := &recRunner{outErrQueue: []error{nil, errFake}}
		c := NewCluster(rr, cfg, nil, nil)
		if c.OperatorInstalled(context.Background()) {
			t.Error("OperatorInstalled should report false when the deployment get fails")
		}
		if len(rr.calls) != 2 {
			t.Errorf("OperatorInstalled should still probe the deployment after a successful crd get; got %d calls", len(rr.calls))
		}
	})
}

func mustContain(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("rendered output missing %q", needle)
	}
}
