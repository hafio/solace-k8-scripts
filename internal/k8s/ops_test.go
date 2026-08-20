package k8s

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"solace/internal/config"
)

func TestStatus(t *testing.T) {
	rr := &recRunner{}
	c := newCluster(rr)
	if err := c.Status(context.Background()); err != nil {
		t.Fatalf("Status: %v", err)
	}
	want := [][]string{
		{"get", "pods", "-n", "solace", "-o", "wide"},
		{"get", "svc", "-n", "solace"},
		{"get", "statefulset", "-n", "solace"},
	}
	if len(rr.calls) != len(want) {
		t.Fatalf("Status made %d calls, want %d", len(rr.calls), len(want))
	}
	for i, w := range want {
		if got := rr.calls[i]; got.method != "Run" || got.name != "kubectl" || !eqArgs(got.args, w) {
			t.Errorf("Status call[%d] = %+v, want Run kubectl %v", i, got, w)
		}
	}
}

// TestStatusFailureStopsEarly proves a failing intermediate `get` aborts Status
// before the remaining queries, matching ShowAll's identical guard (already covered
// by TestShowAllWrapsGetError); TestStatus itself only exercises the all-succeed
// sequence.
func TestStatusFailureStopsEarly(t *testing.T) {
	cases := []struct {
		name      string
		rr        *recRunner
		wantCalls int
	}{
		{"pods get fails", &recRunner{runErr: errFake}, 1},
		{"svc get fails", &recRunner{runErrQueue: []error{nil, errFake}}, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := newCluster(tc.rr)
			if err := c.Status(context.Background()); err == nil {
				t.Error("Status should fail loud when an intermediate get fails")
			}
			if len(tc.rr.calls) != tc.wantCalls {
				t.Errorf("Status made %d calls, want %d", len(tc.rr.calls), tc.wantCalls)
			}
		})
	}
}

// showAllOutputs feeds one canned listing per section, in order.
func showAllOutputs() [][]byte {
	deploys := "NAMESPACE                    NAME                              READY\n" +
		"pubsubplus-operator-system   pubsubplus-eventbroker-operator   1/1\n" +
		"kube-system                  coredns                           2/2\n"
	brokers := "NAMESPACE   NAME         AGE\n" +
		"solace      dev-broker   3d\n"
	pods := "NAMESPACE                    NAME                                   READY   STATUS\n" +
		"solace                       dev-broker-pubsubplus-p-0              1/1     Running\n" +
		"pubsubplus-operator-system   pubsubplus-eventbroker-operator-abc    1/1     Running\n" +
		"kube-system                  coredns-xxxx                           1/1     Running\n"
	svcs := "NAMESPACE   NAME                    TYPE\n" +
		"solace      dev-broker-pubsubplus   LoadBalancer\n" +
		"kube-system kube-dns                ClusterIP\n"
	stss := "NAMESPACE   NAME                     READY\n" +
		"solace      dev-broker-pubsubplus-p  1/1\n"
	return [][]byte{[]byte(deploys), []byte(brokers), []byte(pods), []byte(svcs), []byte(stss)}
}

// TestShowAll covers the running picture `--all` reports: the operator that
// everything depends on, the broker resources themselves, and the pods, services
// and statefulsets behind them -- across every namespace, with unrelated cluster
// workloads filtered out.
func TestShowAll(t *testing.T) {
	rr := &recRunner{outQueue: showAllOutputs()}
	buf := &bytes.Buffer{}
	c := NewCluster(rr, haCfg(), nil, buf)
	if err := c.ShowAll(context.Background(), false); err != nil {
		t.Fatalf("ShowAll: %v", err)
	}
	if len(rr.calls) != len(showAllSections) {
		t.Fatalf("ShowAll made %d calls, want %d (one per section)", len(rr.calls), len(showAllSections))
	}
	if a := rr.calls[0].args; a[0] != "get" || a[1] != "deployments" || a[2] != "--all-namespaces" {
		t.Errorf("operator query argv = %v, want a cluster-wide deployments get", a)
	}
	out := buf.String()
	for _, want := range []string{
		"### OPERATOR ###", "pubsubplus-eventbroker-operator",
		"### BROKERS ###", "dev-broker",
		"### PODS ###", "dev-broker-pubsubplus-p-0",
		"### SERVICES ###", "### STATEFULSETS ###",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("ShowAll output missing %q:\n%s", want, out)
		}
	}
	// Unrelated cluster workloads are dropped. The operator's own POD is too: it is
	// reported once, as a deployment, and does not belong in the broker pod list.
	for _, unwanted := range []string{"coredns", "kube-dns", "eventbroker-operator-abc"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("ShowAll should have filtered out %q:\n%s", unwanted, out)
		}
	}
}

// TestShowAllDetailAddsStaticArtifacts: --detail appends the artifacts a broker is
// built from, which the running picture never shows -- and which is where a PVC
// left behind by a removed broker turns up.
func TestShowAllDetailAddsStaticArtifacts(t *testing.T) {
	outs := showAllOutputs()
	for range showDetailSections {
		outs = append(outs, []byte("NAMESPACE   NAME                    AGE\nsolace      dev-broker-pubsubplus   3d\n"))
	}
	rr := &recRunner{outQueue: outs}
	buf := &bytes.Buffer{}
	c := NewCluster(rr, haCfg(), nil, buf)
	if err := c.ShowAll(context.Background(), true); err != nil {
		t.Fatalf("ShowAll --detail: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"### SECRETS ###", "### CONFIGMAPS ###", "### PERSISTENT VOLUME CLAIMS ###"} {
		if !strings.Contains(out, want) {
			t.Errorf("ShowAll --detail missing %q:\n%s", want, out)
		}
	}
}

// TestSurveyScopesToTheBrokerNamespace: without --all the same sections are read,
// but only in the namespace this env file names -- that is the whole difference
// between the two, so it is asserted on the argv rather than the output.
func TestSurveyScopesToTheBrokerNamespace(t *testing.T) {
	rr := &recRunner{outQueue: showAllOutputs()}
	c := NewCluster(rr, haCfg(), nil, &bytes.Buffer{})
	if err := c.Survey(context.Background(), false); err != nil {
		t.Fatalf("Survey: %v", err)
	}
	for _, call := range rr.calls {
		joined := strings.Join(call.args, " ")
		if strings.Contains(joined, "--all-namespaces") {
			t.Errorf("Survey went cluster-wide: %v", call.args)
		}
		if !strings.Contains(joined, "-n "+haCfg().K8s.Namespace) {
			t.Errorf("Survey did not scope to the broker namespace: %v", call.args)
		}
	}
}

// TestShowAllReportsAndContinuesOnGetError: one resource kind that cannot be listed
// -- an RBAC-restricted secrets read, or a CRD that is not installed -- must not
// hide the kinds that can. The failure is named in place and the survey goes on;
// aborting would make `--detail` unusable for anyone without cluster-admin.
func TestShowAllReportsAndContinuesOnGetError(t *testing.T) {
	rr := &recRunner{outErr: errFake}
	buf := &bytes.Buffer{}
	c := NewCluster(rr, haCfg(), nil, buf)
	if err := c.ShowAll(context.Background(), false); err != nil {
		t.Fatalf("ShowAll should not fail on one unreadable section, got %v", err)
	}
	if len(rr.calls) != len(showAllSections) {
		t.Errorf("ShowAll stopped early: %d calls, want %d", len(rr.calls), len(showAllSections))
	}
	out := buf.String()
	if !strings.Contains(out, "could not list") || !strings.Contains(out, errFake.Error()) {
		t.Errorf("ShowAll should name the section it could not read and why:\n%s", out)
	}
	if !strings.Contains(out, "### STATEFULSETS ###") {
		t.Errorf("ShowAll should still reach the later sections:\n%s", out)
	}
}

func TestFilterLines(t *testing.T) {
	if got := filterLines("", "x"); got != "  (none)" {
		t.Errorf("filterLines(empty) = %q, want (none)", got)
	}
	raw := "HEADER\nkeep-pubsubplus-line\ndrop-me\n"
	got := filterLines(raw, "pubsubplus")
	if !strings.Contains(got, "HEADER") || !strings.Contains(got, "keep-pubsubplus-line") || strings.Contains(got, "drop-me") {
		t.Errorf("filterLines kept the wrong lines: %q", got)
	}
	if got := filterLines("HEADER\ndrop\n", "pubsubplus"); !strings.Contains(got, "(none matched)") {
		t.Errorf("filterLines with no matches should note it: %q", got)
	}
}

func TestDescribeBroker(t *testing.T) {
	rr := &recRunner{}
	c := newCluster(rr)
	if err := c.DescribeBroker(context.Background(), config.Backup); err != nil {
		t.Fatalf("DescribeBroker: %v", err)
	}
	got := rr.last()
	want := []string{"describe", "pod", "-n", "solace", "dev-broker-pubsubplus-b-0"}
	if got.method != "Run" || !eqArgs(got.args, want) {
		t.Errorf("DescribeBroker argv = %+v, want Run kubectl %v", got, want)
	}
}

func TestDescribeLB(t *testing.T) {
	rr := &recRunner{}
	c := newCluster(rr)
	if err := c.DescribeLB(context.Background()); err != nil {
		t.Fatalf("DescribeLB: %v", err)
	}
	got := rr.last()
	want := []string{"describe", "service/dev-broker-pubsubplus", "-n", "solace"}
	if got.method != "Run" || !eqArgs(got.args, want) {
		t.Errorf("DescribeLB argv = %+v, want Run kubectl %v", got, want)
	}
}

func TestLogsPassthrough(t *testing.T) {
	rr := &recRunner{}
	c := newCluster(rr)
	if err := c.Logs(context.Background(), config.Primary, []string{"-f", "--tail=100"}); err != nil {
		t.Fatalf("Logs: %v", err)
	}
	got := rr.last()
	want := []string{"logs", "-n", "solace", "pod/dev-broker-pubsubplus-p-0", "-f", "--tail=100"}
	if got.method != "Run" || !eqArgs(got.args, want) {
		t.Errorf("Logs argv = %+v, want Run kubectl %v", got, want)
	}
}

func TestCLIAndShellAreInteractive(t *testing.T) {
	t.Run("cli", func(t *testing.T) {
		rr := &recRunner{}
		c := newCluster(rr)
		if err := c.CLI(context.Background(), config.Monitor); err != nil {
			t.Fatalf("CLI: %v", err)
		}
		got := rr.last()
		want := []string{"exec", "-it", "-n", "solace", "dev-broker-pubsubplus-m-0", "--", "cli", "-A"}
		if got.method != "RunInteractive" || !eqArgs(got.args, want) {
			t.Errorf("CLI argv = %+v, want RunInteractive kubectl %v", got, want)
		}
	})
	t.Run("shell", func(t *testing.T) {
		rr := &recRunner{}
		c := newCluster(rr)
		if err := c.Shell(context.Background(), config.Primary); err != nil {
			t.Fatalf("Shell: %v", err)
		}
		got := rr.last()
		want := []string{"exec", "-it", "-n", "solace", "dev-broker-pubsubplus-p-0", "--", "bash"}
		if got.method != "RunInteractive" || !eqArgs(got.args, want) {
			t.Errorf("Shell argv = %+v, want RunInteractive kubectl %v", got, want)
		}
	})
}

func TestCopyFrom(t *testing.T) {
	t.Run("downloads each file under its basename", func(t *testing.T) {
		rr := &recRunner{}
		buf := &bytes.Buffer{}
		c := NewCluster(rr, haCfg(), nil, buf)
		files := []string{"/usr/sw/jail/logs/a.log", "/tmp/b.tgz"}
		if err := c.CopyFrom(context.Background(), config.Primary, files); err != nil {
			t.Fatalf("CopyFrom: %v", err)
		}
		if len(rr.calls) != 2 {
			t.Fatalf("CopyFrom made %d calls, want 2", len(rr.calls))
		}
		want := []string{"cp", "-n", "solace", "dev-broker-pubsubplus-p-0:/usr/sw/jail/logs/a.log", "a.log"}
		if got := rr.calls[0]; got.method != "Run" || !eqArgs(got.args, want) {
			t.Errorf("CopyFrom call[0] = %+v, want Run kubectl %v", got, want)
		}
	})
	t.Run("empty file list is an error", func(t *testing.T) {
		c := newCluster(&recRunner{})
		if err := c.CopyFrom(context.Background(), config.Primary, nil); err == nil {
			t.Error("CopyFrom with no files should error")
		}
	})
	t.Run("aggregates copy failures", func(t *testing.T) {
		rr := &recRunner{runErr: errFake}
		buf := &bytes.Buffer{}
		c := NewCluster(rr, haCfg(), nil, buf)
		err := c.CopyFrom(context.Background(), config.Primary, []string{"x", "y"})
		if err == nil {
			t.Fatal("CopyFrom should fail loud when a copy fails")
		}
		if len(rr.calls) != 2 {
			t.Errorf("all files should be attempted; got %d calls", len(rr.calls))
		}
	})
}

func TestCopyInto(t *testing.T) {
	t.Run("uploads into the target dir", func(t *testing.T) {
		rr := &recRunner{}
		buf := &bytes.Buffer{}
		c := NewCluster(rr, haCfg(), nil, buf)
		if err := c.CopyInto(context.Background(), config.Backup, []string{"local.cli"}, "/usr/sw/jail"); err != nil {
			t.Fatalf("CopyInto: %v", err)
		}
		got := rr.last()
		want := []string{"cp", "-n", "solace", "local.cli", "dev-broker-pubsubplus-b-0:/usr/sw/jail"}
		if got.method != "Run" || !eqArgs(got.args, want) {
			t.Errorf("CopyInto argv = %+v, want Run kubectl %v", got, want)
		}
	})
	t.Run("defaults the target dir to .", func(t *testing.T) {
		rr := &recRunner{}
		buf := &bytes.Buffer{}
		c := NewCluster(rr, haCfg(), nil, buf)
		if err := c.CopyInto(context.Background(), config.Primary, []string{"local.cli"}, ""); err != nil {
			t.Fatalf("CopyInto: %v", err)
		}
		got := rr.last()
		want := []string{"cp", "-n", "solace", "local.cli", "dev-broker-pubsubplus-p-0:."}
		if !eqArgs(got.args, want) {
			t.Errorf("CopyInto default-dir argv = %v, want %v", got.args, want)
		}
	})
	t.Run("empty file list is an error", func(t *testing.T) {
		c := newCluster(&recRunner{})
		if err := c.CopyInto(context.Background(), config.Primary, nil, ""); err == nil {
			t.Error("CopyInto with no files should error")
		}
	})
	t.Run("aggregates copy failures", func(t *testing.T) {
		rr := &recRunner{runErr: errFake}
		buf := &bytes.Buffer{}
		c := NewCluster(rr, haCfg(), nil, buf)
		err := c.CopyInto(context.Background(), config.Backup, []string{"x", "y"}, "/tmp")
		if err == nil {
			t.Fatal("CopyInto should fail loud when a copy fails")
		}
		if len(rr.calls) != 2 {
			t.Errorf("all files should be attempted; got %d calls", len(rr.calls))
		}
	})
}

// TestRestartPod covers the manualPodRestart step: delete the pod, then wait for
// the statefulset to bring it back, bounded like every other rollout wait.
func TestRestartPod(t *testing.T) {
	rr := &recRunner{}
	c := newCluster(rr)
	if err := c.RestartPod(context.Background(), config.Backup); err != nil {
		t.Fatalf("RestartPod: %v", err)
	}
	if len(rr.calls) != 2 {
		t.Fatalf("RestartPod made %d calls, want 2 (delete + rollout status)", len(rr.calls))
	}
	wantDelete := []string{"delete", "pod", "-n", "solace", "dev-broker-pubsubplus-b-0", "--ignore-not-found"}
	if !eqArgs(rr.calls[0].args, wantDelete) {
		t.Errorf("delete argv = %v, want %v", rr.calls[0].args, wantDelete)
	}
	wantWait := []string{"rollout", "status", "statefulset/dev-broker-pubsubplus-b", "-n", "solace", "--timeout=300s"}
	if !eqArgs(rr.calls[1].args, wantWait) {
		t.Errorf("rollout argv = %v, want %v", rr.calls[1].args, wantWait)
	}
}

// TestRestartPodDeleteFails proves the pod-delete failing outright (RBAC denied,
// etc.) surfaces its own distinct, actionable message and never reaches the
// rollout-status wait -- only the rollout-wait failure was tested before, via
// RestartRolling's runErrQueue.
func TestRestartPodDeleteFails(t *testing.T) {
	rr := &recRunner{runErr: errFake}
	c := newCluster(rr)
	err := c.RestartPod(context.Background(), config.Backup)
	if err == nil || !strings.Contains(err.Error(), "deleting pod") {
		t.Fatalf("RestartPod error = %v, want it to wrap \"deleting pod\"", err)
	}
	if len(rr.calls) != 1 {
		t.Errorf("RestartPod should stop before the rollout wait; got %d calls", len(rr.calls))
	}
}

func TestRestartRolling(t *testing.T) {
	t.Run("HA bounces all three in order", func(t *testing.T) {
		rr := &recRunner{}
		c := newCluster(rr)
		if err := c.RestartRolling(context.Background()); err != nil {
			t.Fatalf("RestartRolling: %v", err)
		}
		if len(rr.calls) != 6 {
			t.Fatalf("RestartRolling(HA) made %d calls, want 6 (delete+wait x3)", len(rr.calls))
		}
		// Monitor first, primary last: the delete calls carry the pod names.
		for i, want := range []string{"dev-broker-pubsubplus-m-0", "dev-broker-pubsubplus-b-0", "dev-broker-pubsubplus-p-0"} {
			if got := rr.calls[i*2].args[4]; got != want {
				t.Errorf("delete %d targeted %q, want %q", i, got, want)
			}
		}
	})
	t.Run("standalone bounces only the primary", func(t *testing.T) {
		rr := &recRunner{}
		c := NewCluster(rr, saCfg(), nil, nil)
		if err := c.RestartRolling(context.Background()); err != nil {
			t.Fatalf("RestartRolling: %v", err)
		}
		if len(rr.calls) != 2 {
			t.Fatalf("RestartRolling(standalone) made %d calls, want 2", len(rr.calls))
		}
	})
	t.Run("a failed restart stops before the next role", func(t *testing.T) {
		// Let the monitor's delete succeed, then fail the readiness wait: a pod that
		// never comes back must not be followed by bouncing the next one.
		rr := &recRunner{runErrQueue: []error{nil, errFake}}
		c := newCluster(rr)
		err := c.RestartRolling(context.Background())
		if err == nil || !strings.Contains(err.Error(), "did not become ready") {
			t.Fatalf("RestartRolling should fail loud on a pod that stays down, got %v", err)
		}
		if len(rr.calls) != 2 {
			t.Errorf("RestartRolling made %d calls, want 2 -- it must not continue to the next role", len(rr.calls))
		}
	})
}

func TestReplicasStart(t *testing.T) {
	t.Run("HA scales and waits for all three roles", func(t *testing.T) {
		rr := &recRunner{}
		c := newCluster(rr) // haCfg
		if err := c.ReplicasStart(context.Background()); err != nil {
			t.Fatalf("ReplicasStart: %v", err)
		}
		if len(rr.calls) != 6 {
			t.Fatalf("ReplicasStart(HA) made %d calls, want 6 (scale+rollout x3)", len(rr.calls))
		}
		wantScale := []string{"scale", "statefulset", "dev-broker-pubsubplus-p", "-n", "solace", "--replicas=1"}
		wantRollout := []string{"rollout", "status", "statefulset/dev-broker-pubsubplus-p", "-n", "solace", "--timeout=300s"}
		if !eqArgs(rr.calls[0].args, wantScale) {
			t.Errorf("scale argv = %v, want %v", rr.calls[0].args, wantScale)
		}
		if !eqArgs(rr.calls[1].args, wantRollout) {
			t.Errorf("rollout argv = %v, want %v", rr.calls[1].args, wantRollout)
		}
	})
	t.Run("standalone scales only the primary", func(t *testing.T) {
		rr := &recRunner{}
		c := NewCluster(rr, saCfg(), nil, nil)
		if err := c.ReplicasStart(context.Background()); err != nil {
			t.Fatalf("ReplicasStart: %v", err)
		}
		if len(rr.calls) != 2 {
			t.Fatalf("ReplicasStart(standalone) made %d calls, want 2", len(rr.calls))
		}
	})
	t.Run("fails loud when a rollout does not become ready", func(t *testing.T) {
		// Let the primary scale succeed, then fail the rollout-status wait that
		// follows it -- the bounded wait's whole purpose is to fail loud here.
		rr := &recRunner{runErrQueue: []error{nil, errFake}}
		c := newCluster(rr) // haCfg
		err := c.ReplicasStart(context.Background())
		if err == nil || !strings.Contains(err.Error(), "did not become ready") {
			t.Fatalf("ReplicasStart should fail loud on a stuck rollout, got %v", err)
		}
		if len(rr.calls) != 2 {
			t.Errorf("ReplicasStart should stop at the first stuck role; got %d calls", len(rr.calls))
		}
	})
	t.Run("fails loud when the scale itself fails", func(t *testing.T) {
		// The existing failure test only exercises the rollout-wait failing after a
		// successful scale; the scale command's own failure is a distinct, equally
		// real condition with its own message.
		rr := &recRunner{runErr: errFake}
		c := newCluster(rr) // haCfg
		err := c.ReplicasStart(context.Background())
		if err == nil || !strings.Contains(err.Error(), "scaling") || !strings.Contains(err.Error(), "up") {
			t.Fatalf("ReplicasStart error = %v, want it to wrap \"scaling ... up\"", err)
		}
		if len(rr.calls) != 1 {
			t.Errorf("ReplicasStart should stop before the rollout wait; got %d calls", len(rr.calls))
		}
	})
}

func TestReplicasStop(t *testing.T) {
	t.Run("HA scales all three to zero", func(t *testing.T) {
		rr := &recRunner{}
		c := newCluster(rr)
		if err := c.ReplicasStop(context.Background()); err != nil {
			t.Fatalf("ReplicasStop: %v", err)
		}
		if len(rr.calls) != 3 {
			t.Fatalf("ReplicasStop(HA) made %d calls, want 3", len(rr.calls))
		}
		want := []string{"scale", "statefulset", "dev-broker-pubsubplus-p", "-n", "solace", "--replicas=0"}
		if !eqArgs(rr.calls[0].args, want) {
			t.Errorf("scale-down argv = %v, want %v", rr.calls[0].args, want)
		}
	})
	t.Run("standalone scales only the primary", func(t *testing.T) {
		rr := &recRunner{}
		c := NewCluster(rr, saCfg(), nil, nil)
		if err := c.ReplicasStop(context.Background()); err != nil {
			t.Fatalf("ReplicasStop: %v", err)
		}
		if len(rr.calls) != 1 {
			t.Fatalf("ReplicasStop(standalone) made %d calls, want 1", len(rr.calls))
		}
	})
	t.Run("fails loud when the scale-down itself fails", func(t *testing.T) {
		// ReplicasStop had no failure test at all before this: the scale command
		// failing outright is its own real condition with its own message.
		rr := &recRunner{runErr: errFake}
		c := newCluster(rr)
		err := c.ReplicasStop(context.Background())
		if err == nil || !strings.Contains(err.Error(), "scaling") || !strings.Contains(err.Error(), "down") {
			t.Fatalf("ReplicasStop error = %v, want it to wrap \"scaling ... down\"", err)
		}
		if len(rr.calls) != 1 {
			t.Errorf("ReplicasStop should stop at the first failing role; got %d calls", len(rr.calls))
		}
	})
}
