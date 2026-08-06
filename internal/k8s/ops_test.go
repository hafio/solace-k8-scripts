package k8s

import (
	"bytes"
	"context"
	"errors"
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

func TestShowAll(t *testing.T) {
	pods := "NAMESPACE                    NAME                                   READY   STATUS\n" +
		"solace                       dev-broker-pubsubplus-p-0              1/1     Running\n" +
		"pubsubplus-operator-system   pubsubplus-eventbroker-operator-abc    1/1     Running\n" +
		"kube-system                  coredns-xxxx                           1/1     Running\n"
	svcs := "NAMESPACE   NAME                    TYPE\n" +
		"solace      dev-broker-pubsubplus   LoadBalancer\n" +
		"kube-system kube-dns                ClusterIP\n"
	stss := "NAMESPACE   NAME                     READY\n" +
		"solace      dev-broker-pubsubplus-p  1/1\n"

	rr := &recRunner{outQueue: [][]byte{[]byte(pods), []byte(svcs), []byte(stss)}}
	buf := &bytes.Buffer{}
	c := NewCluster(rr, haCfg(), nil, buf)
	if err := c.ShowAll(context.Background()); err != nil {
		t.Fatalf("ShowAll: %v", err)
	}
	if len(rr.calls) != 3 {
		t.Fatalf("ShowAll made %d calls, want 3 (pods/svc/sts)", len(rr.calls))
	}
	if a := rr.calls[0].args; a[0] != "get" || a[1] != "pods" || a[2] != "--all-namespaces" || a[3] != "-o" || a[4] != "wide" {
		t.Errorf("pods query argv = %v", a)
	}
	out := buf.String()
	// Broker resources kept; operator pod and unrelated cluster resources dropped.
	for _, want := range []string{"### PODS ###", "dev-broker-pubsubplus-p-0", "dev-broker-pubsubplus", "### STATEFULSETS ###"} {
		if !strings.Contains(out, want) {
			t.Errorf("ShowAll output missing %q:\n%s", want, out)
		}
	}
	for _, unwanted := range []string{"coredns", "eventbroker-operator", "kube-dns"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("ShowAll should have filtered out %q:\n%s", unwanted, out)
		}
	}
}

// TestShowAllWrapsGetError: a failing `get` aborts ShowAll loudly with per-resource
// context and preserves the cause, instead of printing a half-filtered table.
func TestShowAllWrapsGetError(t *testing.T) {
	rr := &recRunner{outErr: errFake}
	c := NewCluster(rr, haCfg(), nil, &bytes.Buffer{})
	err := c.ShowAll(context.Background())
	if err == nil || !strings.Contains(err.Error(), "listing pods across namespaces") {
		t.Fatalf("ShowAll should wrap the get error, got %v", err)
	}
	if !errors.Is(err, errFake) {
		t.Errorf("ShowAll should preserve the cause: %v", err)
	}
	if len(rr.calls) != 1 {
		t.Errorf("ShowAll should stop after the first failing section; got %d calls", len(rr.calls))
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
}
