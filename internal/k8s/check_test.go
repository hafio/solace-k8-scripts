package k8s

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"solace/internal/engine"
)

// errFake is a stand-in cluster error shared by the check/prep tests.
var errFake = errors.New("connection refused")

// TestCheckEnvNoSecretLeak proves the config report shows secrets as set/MISSING
// and never prints their values (§3). env/sample.yaml carries CHANGE-ME-* secrets.
func TestCheckEnvNoSecretLeak(t *testing.T) {
	cfg := loadK8s(t)
	buf := &bytes.Buffer{}
	c := NewCluster(&recRunner{}, cfg, nil, buf)
	c.CheckEnv()
	out := buf.String()

	for _, secret := range []string{"CHANGE-ME-admin", "CHANGE-ME-monitor", "CHANGE-ME-registry"} {
		if strings.Contains(out, secret) {
			t.Errorf("CheckEnv leaked secret value %q:\n%s", secret, out)
		}
	}
	// "cluster cmd" reports the resolved k8s.runtime, as 001-check-env.sh:23 did.
	for _, want := range []string{"dev-broker", "solace", "HA redundancy", "password=set", "cluster cmd    : kubectl"} {
		if !strings.Contains(out, want) {
			t.Errorf("CheckEnv missing %q in:\n%s", want, out)
		}
	}
}

// TestCheckEnvSparseConfig proves the negative half of every CheckEnv formatter:
// env/sample.yaml (the only fixture every other CheckEnv test uses) has every field
// set, so MISSING admin password, "(not configured)" TLS, "(cluster default)"
// storage class and "(none)" monitor password have never actually printed. A broken
// fallback here would silently mislead an operator running `k8s check`.
func TestCheckEnvSparseConfig(t *testing.T) {
	cfg := loadK8s(t)
	cfg.Admin.Pass = ""
	cfg.Admin.MonitorPass = ""
	cfg.TLS.ServerSecret = ""
	cfg.K8s.Storage.Class = ""
	cfg.K8s.Operator.WatchBrokerNS = boolPtr(false)
	cfg.K8s.Operator.WatchNamespaces = ""
	buf := &bytes.Buffer{}
	c := NewCluster(&recRunner{}, cfg, nil, buf)
	c.CheckEnv()
	out := buf.String()
	for _, want := range []string{"password=MISSING", "(not configured)", "(cluster default)", "monitorPassword=(none)"} {
		if !strings.Contains(out, want) {
			t.Errorf("CheckEnv sparse config missing %q in:\n%s", want, out)
		}
	}
}

func TestReachable(t *testing.T) {
	t.Run("reachable", func(t *testing.T) {
		rr := &recRunner{}
		c := newCluster(rr)
		if err := c.Reachable(context.Background()); err != nil {
			t.Fatalf("Reachable: %v", err)
		}
		got := rr.last()
		if got.method != "Output" || !eqArgs(got.args, []string{"version", "-o", "json"}) {
			t.Errorf("Reachable argv = %+v, want Output kubectl [version -o json]", got)
		}
	})
	t.Run("unreachable", func(t *testing.T) {
		c := newCluster(&recRunner{outErr: errFake})
		if err := c.Reachable(context.Background()); err == nil {
			t.Error("Reachable should fail when the API server errors")
		}
	})
}

// TestCheckAbortsWhenUnreachable pins Check's early return: an unreachable API
// server must abort before CheckStorageClass wastes a round-trip resolving the
// default StorageClass.
func TestCheckAbortsWhenUnreachable(t *testing.T) {
	rr := &recRunner{outErr: errFake}
	c := NewCluster(rr, loadK8s(t), nil, &bytes.Buffer{})
	err := c.Check(context.Background())
	if err == nil || !strings.Contains(err.Error(), "cannot reach") {
		t.Fatalf("Check error = %v, want it to wrap \"cannot reach\"", err)
	}
	if len(rr.calls) != 1 {
		t.Errorf("Check should stop after the Reachable probe; got %d calls", len(rr.calls))
	}
}

func TestCheckStorageClass(t *testing.T) {
	t.Run("suitable configured class", func(t *testing.T) {
		cfg := haCfg()
		cfg.K8s.Storage.Class = "fast"
		rr := &recRunner{outQueue: [][]byte{[]byte("WaitForFirstConsumer\n"), []byte("true\n")}}
		c := NewCluster(rr, cfg, nil, &bytes.Buffer{})
		if err := c.CheckStorageClass(context.Background()); err != nil {
			t.Fatalf("CheckStorageClass: %v", err)
		}
		// Configured class -> no default-resolution query; just the two attribute reads.
		if len(rr.calls) != 2 {
			t.Fatalf("made %d calls, want 2 attribute reads", len(rr.calls))
		}
		want := []string{"get", "sc", "fast", "-o", "custom-columns=V:.volumeBindingMode", "--no-headers"}
		if !eqArgs(rr.calls[0].args, want) {
			t.Errorf("first read argv = %v, want %v", rr.calls[0].args, want)
		}
	})
	t.Run("unsuitable binding/expansion", func(t *testing.T) {
		cfg := haCfg()
		cfg.K8s.Storage.Class = "slow"
		rr := &recRunner{outQueue: [][]byte{[]byte("Immediate\n"), []byte("false\n")}}
		c := NewCluster(rr, cfg, nil, &bytes.Buffer{})
		if err := c.CheckStorageClass(context.Background()); err == nil {
			t.Error("CheckStorageClass should reject Immediate binding / no expansion")
		}
	})
	t.Run("missing fields report as <none>", func(t *testing.T) {
		cfg := haCfg()
		cfg.K8s.Storage.Class = "x"
		rr := &recRunner{outQueue: [][]byte{[]byte("<none>\n"), []byte("<none>\n")}}
		c := NewCluster(rr, cfg, nil, &bytes.Buffer{})
		if err := c.CheckStorageClass(context.Background()); err == nil {
			t.Error("CheckStorageClass should fail when the class lacks the required attributes")
		}
	})
	t.Run("dry-run skips assertions", func(t *testing.T) {
		cfg := haCfg() // no storage class -> would resolve default, but Echo returns nothing
		buf := &bytes.Buffer{}
		c := NewCluster(engine.Echo{W: buf}, cfg, nil, buf)
		if err := c.CheckStorageClass(context.Background()); err != nil {
			t.Fatalf("CheckStorageClass under dry-run: %v", err)
		}
		if !strings.Contains(buf.String(), "skipped (dry-run)") {
			t.Errorf("expected a dry-run skip note; got %q", buf.String())
		}
	})
	// TestCheckStorageClass/default-resolution query fails proves a genuine query
	// failure (RBAC, connection refused) propagates resolveStorageClass's own wrap,
	// not just a bare error -- previously only exercised the successful/empty/
	// multi-value shapes of the lookup, never a real failure end-to-end.
	t.Run("default-resolution query fails", func(t *testing.T) {
		cfg := haCfg() // no Storage.Class -> resolves the cluster default
		rr := &recRunner{outErr: errFake}
		c := NewCluster(rr, cfg, nil, &bytes.Buffer{})
		err := c.CheckStorageClass(context.Background())
		if err == nil || !strings.Contains(err.Error(), "resolving default StorageClass") {
			t.Fatalf("CheckStorageClass error = %v, want it to wrap \"resolving default StorageClass\"", err)
		}
		if len(rr.calls) != 1 {
			t.Errorf("CheckStorageClass should stop at the failing default-resolution query; got %d calls", len(rr.calls))
		}
	})
	// TestCheckStorageClass/no default and none configured proves the 009-ported
	// actionable error fires when the class resolves empty (no default, none
	// configured) -- previously only checked via resolveStorageClass in isolation,
	// never through the caller that turns "" into a fail-loud error.
	t.Run("no default and none configured", func(t *testing.T) {
		cfg := haCfg() // no Storage.Class, no default StorageClass on the cluster
		rr := &recRunner{out: []byte("")}
		c := NewCluster(rr, cfg, nil, &bytes.Buffer{})
		err := c.CheckStorageClass(context.Background())
		if err == nil || !strings.Contains(err.Error(), "no default StorageClass found") {
			t.Fatalf("CheckStorageClass error = %v, want \"no default StorageClass found\"", err)
		}
	})
	// TestCheckStorageClass/scColumn read fails proves scColumn's own error return
	// and the caller's "reading StorageClass %q" wrap (naming the class) both fire --
	// every other subtest here scripts a successful column read.
	t.Run("scColumn read fails", func(t *testing.T) {
		cfg := haCfg()
		cfg.K8s.Storage.Class = "fast"
		rr := &recRunner{outErr: errFake}
		c := NewCluster(rr, cfg, nil, &bytes.Buffer{})
		err := c.CheckStorageClass(context.Background())
		if err == nil || !strings.Contains(err.Error(), `reading StorageClass "fast"`) {
			t.Fatalf(`CheckStorageClass error = %v, want it to wrap reading StorageClass "fast"`, err)
		}
		if len(rr.calls) != 1 {
			t.Errorf("CheckStorageClass should stop at the first failing column read; got %d calls", len(rr.calls))
		}
	})
	// TestCheckStorageClass/second attribute read fails after the first succeeds is
	// the other real, reachable half of the two-read sequence: volumeBindingMode and
	// allowVolumeExpansion are two separate kubectl calls, so a transient failure on
	// just the second is just as real as on the first. Needs outErrQueue to script
	// differing per-call results -- previously impossible with a single outErr.
	t.Run("second attribute read fails after the first succeeds", func(t *testing.T) {
		cfg := haCfg()
		cfg.K8s.Storage.Class = "fast"
		rr := &recRunner{
			outQueue:    [][]byte{[]byte("WaitForFirstConsumer\n")},
			outErrQueue: []error{nil, errFake},
		}
		c := NewCluster(rr, cfg, nil, &bytes.Buffer{})
		err := c.CheckStorageClass(context.Background())
		if err == nil || !strings.Contains(err.Error(), `reading StorageClass "fast"`) {
			t.Fatalf(`CheckStorageClass error = %v, want it to wrap reading StorageClass "fast"`, err)
		}
		if len(rr.calls) != 2 {
			t.Errorf("CheckStorageClass should have attempted both attribute reads; got %d calls", len(rr.calls))
		}
	})
}

func TestResolveStorageClass(t *testing.T) {
	t.Run("configured", func(t *testing.T) {
		cfg := haCfg()
		cfg.K8s.Storage.Class = "chosen"
		rr := &recRunner{}
		c := NewCluster(rr, cfg, nil, nil)
		got, err := c.resolveStorageClass(context.Background())
		if err != nil || got != "chosen" {
			t.Fatalf("resolveStorageClass = (%q,%v), want (chosen,nil)", got, err)
		}
		if len(rr.calls) != 0 {
			t.Errorf("a configured class must not query the cluster; got %d calls", len(rr.calls))
		}
	})
	t.Run("single default", func(t *testing.T) {
		rr := &recRunner{out: []byte("standard\n")}
		c := NewCluster(rr, haCfg(), nil, nil)
		got, err := c.resolveStorageClass(context.Background())
		if err != nil || got != "standard" {
			t.Fatalf("resolveStorageClass = (%q,%v), want (standard,nil)", got, err)
		}
		if a := rr.last().args; a[0] != "get" || a[1] != "sc" || a[2] != "-o" || !strings.HasPrefix(a[3], "jsonpath=") {
			t.Errorf("default-resolution argv = %v", a)
		}
	})
	t.Run("multiple defaults is an error", func(t *testing.T) {
		c := NewCluster(&recRunner{out: []byte("a b\n")}, haCfg(), nil, nil)
		if _, err := c.resolveStorageClass(context.Background()); err == nil {
			t.Error("resolveStorageClass should reject multiple default classes")
		}
	})
	t.Run("no default returns empty", func(t *testing.T) {
		c := NewCluster(&recRunner{out: []byte("")}, haCfg(), nil, nil)
		got, err := c.resolveStorageClass(context.Background())
		if err != nil || got != "" {
			t.Fatalf("resolveStorageClass = (%q,%v), want (\"\",nil)", got, err)
		}
	})
}

// TestCheckDryRun runs the whole preflight under the Echo runner: it must not error
// and must produce both the config report and the storage-class skip note.
func TestCheckDryRun(t *testing.T) {
	buf := &bytes.Buffer{}
	c := NewCluster(engine.Echo{W: buf}, loadK8s(t), nil, buf)
	if err := c.Check(context.Background()); err != nil {
		t.Fatalf("Check under dry-run: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Solace broker deployment") || !strings.Contains(out, "skipped (dry-run)") {
		t.Errorf("Check dry-run output incomplete:\n%s", out)
	}
}
