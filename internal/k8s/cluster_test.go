package k8s

import (
	"context"
	"errors"
	"testing"
)

// newCluster builds a Cluster over a fresh recRunner for argv/behaviour assertions.
func newCluster(rr *recRunner) *Cluster {
	return NewCluster(rr, haCfg(), nil, nil)
}

func TestOperatorNSExplicit(t *testing.T) {
	rr := &recRunner{}
	c := newCluster(rr)
	c.Cfg.K8s.Operator.Namespace = "chosen-ns"
	if got := c.operatorNS(context.Background()); got != "chosen-ns" {
		t.Errorf("operatorNS = %q, want chosen-ns", got)
	}
	if len(rr.calls) != 0 {
		t.Errorf("explicit namespace must not probe the cluster; got %d calls", len(rr.calls))
	}
}

func TestOperatorNSDerived(t *testing.T) {
	rr := &recRunner{out: []byte(
		"NS            NAME\n" +
			"kube-system   coredns\n" +
			"my-op-ns      pubsubplus-eventbroker-operator\n" +
			"solace        dev-broker-pubsubplus-p\n")}
	c := newCluster(rr) // haCfg has no Operator.Namespace -> discovery
	if got := c.operatorNS(context.Background()); got != "my-op-ns" {
		t.Errorf("operatorNS = %q, want my-op-ns (first column of the operator row)", got)
	}
	got := rr.last()
	if got.method != "Output" || got.name != "kubectl" {
		t.Fatalf("discovery should use kubectl Output; got %+v", got)
	}
	if got.args[0] != "get" || got.args[1] != "deployment" || got.args[2] != "--all-namespaces" {
		t.Errorf("discovery argv = %v", got.args)
	}
}

func TestOperatorNSDefaultWhenAbsent(t *testing.T) {
	rr := &recRunner{out: []byte("NS   NAME\nkube-system   coredns\n")} // no operator row
	c := newCluster(rr)
	if got := c.operatorNS(context.Background()); got != defaultOperatorNS {
		t.Errorf("operatorNS = %q, want default %q", got, defaultOperatorNS)
	}
}

func TestOperatorNSDefaultOnError(t *testing.T) {
	rr := &recRunner{outErr: errors.New("connection refused")} // fresh/unreachable cluster
	c := newCluster(rr)
	if got := c.operatorNS(context.Background()); got != defaultOperatorNS {
		t.Errorf("operatorNS on lookup error = %q, want default %q", got, defaultOperatorNS)
	}
}

func TestApplyOnStdin(t *testing.T) {
	rr := &recRunner{}
	c := newCluster(rr)
	manifest := []byte("kind: Namespace\n")
	if err := c.apply(context.Background(), manifest); err != nil {
		t.Fatalf("apply: %v", err)
	}
	got := rr.last()
	if got.method != "RunInput" || got.name != "kubectl" ||
		!eqArgs(got.args, []string{"apply", "-f", "-"}) || got.stdin != string(manifest) {
		t.Errorf("apply\n got: %+v\nwant RunInput kubectl [apply -f -] with manifest on stdin", got)
	}
}

func TestDeleteStdin(t *testing.T) {
	rr := &recRunner{}
	c := newCluster(rr)
	manifest := []byte("kind: Namespace\n")
	if err := c.deleteStdin(context.Background(), manifest); err != nil {
		t.Fatalf("deleteStdin: %v", err)
	}
	got := rr.last()
	if got.method != "RunInput" || got.name != "kubectl" ||
		!eqArgs(got.args, []string{"delete", "-f", "-", "--ignore-not-found"}) || got.stdin != string(manifest) {
		t.Errorf("deleteStdin\n got: %+v\nwant RunInput kubectl [delete -f - --ignore-not-found]", got)
	}
}
