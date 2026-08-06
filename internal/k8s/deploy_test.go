package k8s

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestDeployBrokerApply(t *testing.T) {
	cfg := loadK8s(t)
	rr := &recRunner{}
	c := NewCluster(rr, cfg, nil, nil)
	if err := c.DeployBroker(context.Background(), false); err != nil {
		t.Fatalf("DeployBroker: %v", err)
	}
	if len(rr.calls) != 1 {
		t.Fatalf("DeployBroker(keepYAML=false) made %d calls, want 1 apply", len(rr.calls))
	}
	got := rr.last()
	if got.method != "RunInput" || got.name != "kubectl" || !eqArgs(got.args, []string{"apply", "-f", "-"}) {
		t.Fatalf("DeployBroker argv = %+v, want RunInput kubectl [apply -f -]", got)
	}
	if !strings.Contains(got.stdin, "dev-broker") {
		t.Errorf("rendered CR missing the broker name:\n%s", got.stdin)
	}
}

// TestDeployBrokerKeepYAML: with keepYAML set, the rendered manifest is written to
// .broker.yaml in the working directory and is byte-identical to what was applied.
func TestDeployBrokerKeepYAML(t *testing.T) {
	cfg := loadK8s(t) // load before chdir: sampleFixture is a relative path
	t.Chdir(t.TempDir())
	rr := &recRunner{}
	c := NewCluster(rr, cfg, nil, nil)
	if err := c.DeployBroker(context.Background(), true); err != nil {
		t.Fatalf("DeployBroker: %v", err)
	}
	data, err := os.ReadFile(brokerYAMLFile)
	if err != nil {
		t.Fatalf("reading %s: %v", brokerYAMLFile, err)
	}
	if string(data) != rr.last().stdin {
		t.Errorf("%s differs from the applied manifest", brokerYAMLFile)
	}
}

func TestDeleteBrokerNoPurge(t *testing.T) {
	cfg := loadK8s(t)
	rr := &recRunner{}
	c := NewCluster(rr, cfg, nil, nil)
	if err := c.DeleteBroker(context.Background(), false); err != nil {
		t.Fatalf("DeleteBroker: %v", err)
	}
	if len(rr.calls) != 1 {
		t.Fatalf("DeleteBroker(purge=false) made %d calls, want 1 (CR delete, no PVCs)", len(rr.calls))
	}
	got := rr.last()
	if got.method != "RunInput" || !eqArgs(got.args, []string{"delete", "-f", "-", "--ignore-not-found"}) {
		t.Errorf("DeleteBroker argv = %+v, want RunInput kubectl [delete -f - --ignore-not-found]", got)
	}
}

func TestDeleteBrokerPurgeHA(t *testing.T) {
	cfg := loadK8s(t) // redundancy: yes
	rr := &recRunner{}
	c := NewCluster(rr, cfg, nil, nil)
	if err := c.DeleteBroker(context.Background(), true); err != nil {
		t.Fatalf("DeleteBroker: %v", err)
	}
	if len(rr.calls) != 4 {
		t.Fatalf("DeleteBroker(purge, HA) made %d calls, want 4 (CR + 3 PVCs)", len(rr.calls))
	}
	wantPVCs := []string{
		"data-dev-broker-pubsubplus-p-0",
		"data-dev-broker-pubsubplus-b-0",
		"data-dev-broker-pubsubplus-m-0",
	}
	for i, pvc := range wantPVCs {
		got := rr.calls[i+1]
		want := []string{"delete", "pvc", pvc, "-n", "solace", "--ignore-not-found"}
		if got.method != "Run" || !eqArgs(got.args, want) {
			t.Errorf("PVC delete[%d] = %+v, want Run kubectl %v", i, got, want)
		}
	}
}

func TestDeleteBrokerPurgeStandalone(t *testing.T) {
	cfg := loadK8s(t)
	cfg.Redundancy = "no"
	rr := &recRunner{}
	c := NewCluster(rr, cfg, nil, nil)
	if err := c.DeleteBroker(context.Background(), true); err != nil {
		t.Fatalf("DeleteBroker: %v", err)
	}
	if len(rr.calls) != 2 {
		t.Fatalf("DeleteBroker(purge, standalone) made %d calls, want 2 (CR + 1 PVC)", len(rr.calls))
	}
	got := rr.calls[1]
	want := []string{"delete", "pvc", "data-dev-broker-pubsubplus-p-0", "-n", "solace", "--ignore-not-found"}
	if !eqArgs(got.args, want) {
		t.Errorf("PVC delete = %v, want %v", got.args, want)
	}
}

// TestDeleteBrokerPurgeSwallowsPVCError: a failing PVC delete is best-effort -- it is
// logged as a WARN but must not abort teardown (deploy.go). runErr hits the Run-backed
// PVC deletes; the CR delete rides RunInput and still succeeds.
func TestDeleteBrokerPurgeSwallowsPVCError(t *testing.T) {
	cfg := loadK8s(t) // redundancy: yes
	rr := &recRunner{runErr: errFake}
	c := NewCluster(rr, cfg, nil, nil)
	if err := c.DeleteBroker(context.Background(), true); err != nil {
		t.Fatalf("DeleteBroker must swallow PVC-delete failures, got: %v", err)
	}
	if len(rr.calls) != 4 {
		t.Fatalf("all PVC deletes should still be attempted; got %d calls, want 4", len(rr.calls))
	}
}
