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
