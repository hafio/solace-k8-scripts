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
	calls := rr.afterPreflight(t, "create", brokerResource)
	if len(calls) != 1 {
		t.Fatalf("DeployBroker(keepYAML=false) made %d calls after the probe, want 1 apply", len(calls))
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

// TestDeployBrokerKeepYAMLWriteError proves a failed .broker.yaml write (disk full,
// permission denied, path collision) fails loud and never applies -- otherwise a
// user would believe the manifest was saved for review/VCS when it was not.
func TestDeployBrokerKeepYAMLWriteError(t *testing.T) {
	cfg := loadK8s(t) // load before chdir: sampleFixture is a relative path
	t.Chdir(t.TempDir())
	// Occupy the path with a directory so os.WriteFile fails cross-platform.
	if err := os.Mkdir(brokerYAMLFile, 0o755); err != nil {
		t.Fatalf("Mkdir %s: %v", brokerYAMLFile, err)
	}
	rr := &recRunner{}
	c := NewCluster(rr, cfg, nil, nil)
	err := c.DeployBroker(context.Background(), true)
	if err == nil || !strings.Contains(err.Error(), brokerYAMLFile) {
		t.Fatalf("DeployBroker error = %v, want it to name %s", err, brokerYAMLFile)
	}
	// The probe ran (it precedes the write); nothing after it may have.
	if calls := rr.afterPreflight(t, "create", brokerResource); len(calls) != 0 {
		t.Errorf("DeployBroker should abort before applying when the write fails; got %d calls after the probe", len(calls))
	}
}

// TestDeployBrokerStopsOnPreflightFailure is the layer-7 ordering guarantee: when
// the probe says no, nothing is written and nothing is applied. Without this the
// preflight would be decoration -- a check whose failure still let the work proceed.
func TestDeployBrokerStopsOnPreflightFailure(t *testing.T) {
	cfg := loadK8s(t) // load before chdir: sampleFixture is a relative path
	dir := t.TempDir()
	t.Chdir(dir)
	rr := &recRunner{canI: "no"}
	c := NewCluster(rr, cfg, nil, nil)

	err := c.DeployBroker(context.Background(), true)
	if err == nil {
		t.Fatal("DeployBroker must fail when the permission probe answers no")
	}
	if !strings.Contains(err.Error(), "not allowed to create") {
		t.Errorf("error = %v, want it to say the permission was refused", err)
	}
	// Nonzero, and nothing done: no manifest on disk...
	if _, statErr := os.Stat(brokerYAMLFile); statErr == nil {
		t.Errorf("%s was written despite a failed preflight", brokerYAMLFile)
	}
	// ...and no call beyond the probe itself.
	if len(rr.calls) != 1 {
		t.Errorf("%d calls made after a failed preflight, want only the probe: %+v", len(rr.calls), rr.calls)
	}
}

// TestPreflightUnreachableClusterHints: an expired token or missing context is a
// different failure from an RBAC refusal, and gets the hint that actually helps.
// The tool never offers to log in on the operator's behalf.
func TestPreflightUnreachableClusterHints(t *testing.T) {
	cfg := loadK8s(t)
	rr := &recRunner{canIErr: errFake}
	c := NewCluster(rr, cfg, nil, nil)

	err := c.DeployBroker(context.Background(), false)
	if err == nil {
		t.Fatal("DeployBroker must fail when the cluster cannot be reached")
	}
	msg := err.Error()
	for _, want := range []string{"cannot check permission", "log in first", "oc login"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error = %v, want it to contain %q", msg, want)
		}
	}
	// The CLI's own error is passed through rather than replaced.
	if !strings.Contains(msg, errFake.Error()) {
		t.Errorf("error = %v, want it to carry the CLI's own failure", msg)
	}
	if len(rr.calls) != 1 {
		t.Errorf("%d calls made after an unreachable cluster, want only the probe", len(rr.calls))
	}
}

func TestDeleteBrokerNoPurge(t *testing.T) {
	cfg := loadK8s(t)
	rr := &recRunner{}
	c := NewCluster(rr, cfg, nil, nil)
	if err := c.DeleteBroker(context.Background(), false); err != nil {
		t.Fatalf("DeleteBroker: %v", err)
	}
	if calls := rr.afterPreflight(t, "delete", brokerResource); len(calls) != 1 {
		t.Fatalf("DeleteBroker(purge=false) made %d calls after the probe, want 1 (CR delete, no PVCs)", len(calls))
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
	calls := rr.afterPreflight(t, "delete", brokerResource)
	if len(calls) != 4 {
		t.Fatalf("DeleteBroker(purge, HA) made %d calls after the probe, want 4 (CR + 3 PVCs)", len(calls))
	}
	wantPVCs := []string{
		"data-dev-broker-pubsubplus-p-0",
		"data-dev-broker-pubsubplus-b-0",
		"data-dev-broker-pubsubplus-m-0",
	}
	for i, pvc := range wantPVCs {
		got := calls[i+1]
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
	calls := rr.afterPreflight(t, "delete", brokerResource)
	if len(calls) != 2 {
		t.Fatalf("DeleteBroker(purge, standalone) made %d calls after the probe, want 2 (CR + 1 PVC)", len(calls))
	}
	got := calls[1]
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
	if calls := rr.afterPreflight(t, "delete", brokerResource); len(calls) != 4 {
		t.Fatalf("all PVC deletes should still be attempted; got %d calls after the probe, want 4", len(calls))
	}
}
