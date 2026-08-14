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

// TestOperatorDelete asserts teardown deletes the bundle on stdin with
// --ignore-not-found.
func TestOperatorDelete(t *testing.T) {
	cfg := loadK8s(t)
	cfg.K8s.Operator.Namespace = "op-ns"
	rr := &recRunner{}
	c := NewCluster(rr, cfg, nil, nil)
	if err := c.OperatorDelete(context.Background()); err != nil {
		t.Fatalf("OperatorDelete: %v", err)
	}
	got := rr.last()
	if got.method != "RunInput" || got.name != "kubectl" ||
		!eqArgs(got.args, []string{"delete", "-f", "-", "--ignore-not-found"}) {
		t.Errorf("OperatorDelete\n got: %+v\nwant RunInput kubectl [delete -f - --ignore-not-found]", got)
	}
	if !strings.Contains(got.stdin, "kind: Deployment") {
		t.Error("delete should pipe the rendered bundle")
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

func mustContain(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("rendered output missing %q", needle)
	}
}
