package k8s

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"solace/internal/config"
)

// adminCfg is haCfg with the minimum admin fields AdminSecret requires, so
// CreateSecrets/DeleteSecrets tests are hermetic (no dependency on env/sample.yaml).
func adminCfg() *config.Config {
	c := haCfg()
	c.Admin.Pass = "pw"
	c.Admin.UserSecret = "solace-admin-secret"
	return c
}

func TestCreateNamespace(t *testing.T) {
	rr := &recRunner{}
	c := newCluster(rr)
	if err := c.CreateNamespace(context.Background()); err != nil {
		t.Fatalf("CreateNamespace: %v", err)
	}
	got := rr.last()
	if got.method != "RunInput" || got.name != "kubectl" || !eqArgs(got.args, []string{"apply", "-f", "-"}) {
		t.Fatalf("CreateNamespace argv = %+v, want RunInput kubectl [apply -f -]", got)
	}
	if !strings.Contains(got.stdin, "kind: Namespace") || !strings.Contains(got.stdin, "name: solace") {
		t.Errorf("CreateNamespace manifest on stdin =\n%s", got.stdin)
	}
}

func TestDeleteNamespace(t *testing.T) {
	rr := &recRunner{}
	c := newCluster(rr)
	if err := c.DeleteNamespace(context.Background()); err != nil {
		t.Fatalf("DeleteNamespace: %v", err)
	}
	got := rr.last()
	want := []string{"delete", "namespace", "solace", "--ignore-not-found"}
	if got.method != "Run" || got.name != "kubectl" || !eqArgs(got.args, want) {
		t.Errorf("DeleteNamespace argv = %+v, want Run kubectl %v", got, want)
	}
}

// TestCreateSecretsAdminOnly: with no TLS and no pull secret, only the admin secret
// is applied, as a single (unseparated) manifest.
func TestCreateSecretsAdminOnly(t *testing.T) {
	rr := &recRunner{}
	c := NewCluster(rr, adminCfg(), nil, nil)
	if err := c.CreateSecrets(context.Background()); err != nil {
		t.Fatalf("CreateSecrets: %v", err)
	}
	if len(rr.calls) != 1 {
		t.Fatalf("CreateSecrets made %d calls, want 1 apply", len(rr.calls))
	}
	got := rr.last()
	if got.method != "RunInput" || !eqArgs(got.args, []string{"apply", "-f", "-"}) {
		t.Fatalf("CreateSecrets argv = %+v, want RunInput kubectl [apply -f -]", got)
	}
	if !strings.Contains(got.stdin, "name: solace-admin-secret") || !strings.Contains(got.stdin, "type: Opaque") {
		t.Errorf("admin secret missing from stdin:\n%s", got.stdin)
	}
	if strings.Contains(got.stdin, "kubernetes.io/tls") || strings.Contains(got.stdin, ".dockerconfigjson") {
		t.Error("admin-only CreateSecrets should not emit a TLS or pull secret")
	}
	if strings.Contains(got.stdin, "---") {
		t.Error("a single secret must not carry a document separator")
	}
}

// TestCreateSecretsAllThree: admin + TLS + pull secret join into one multi-doc
// manifest applied on stdin, and no secret value ever reaches the argv (§3).
func TestCreateSecretsAllThree(t *testing.T) {
	dir := t.TempDir()
	crt := filepath.Join(dir, "tls.crt")
	key := filepath.Join(dir, "tls.key")
	writeFile(t, crt, "CERT\n")
	writeFile(t, key, "KEY\n")

	cfg := adminCfg()
	cfg.TLS.ServerSecret = "solace-tls-secret"
	cfg.TLS.Cert = crt
	cfg.TLS.CertKey = key
	cfg.Image.PullSecret = "solace-image-pull"
	cfg.Image.User = "u"
	cfg.Image.Pass = "SECRET-REG-PASS"
	cfg.Image.Registry = "registry.example.com"

	rr := &recRunner{}
	c := NewCluster(rr, cfg, nil, nil)
	if err := c.CreateSecrets(context.Background()); err != nil {
		t.Fatalf("CreateSecrets: %v", err)
	}
	if len(rr.calls) != 1 {
		t.Fatalf("CreateSecrets made %d calls, want 1 apply", len(rr.calls))
	}
	got := rr.last()
	if got.method != "RunInput" || !eqArgs(got.args, []string{"apply", "-f", "-"}) {
		t.Fatalf("CreateSecrets argv = %+v", got)
	}
	if strings.Count(got.stdin, "---") != 2 {
		t.Errorf("expected 3 joined docs (2 separators):\n%s", got.stdin)
	}
	for _, want := range []string{"name: solace-admin-secret", "name: solace-tls-secret", "name: solace-image-pull"} {
		if !strings.Contains(got.stdin, want) {
			t.Errorf("stdin missing %q", want)
		}
	}
	if strings.Contains(strings.Join(got.args, " "), "SECRET-REG-PASS") {
		t.Error("registry password leaked into the argv")
	}
	if strings.Contains(got.stdin, "SECRET-REG-PASS") {
		t.Error("registry password appears in plaintext on stdin (must be base64 in the secret data)")
	}
}

// TestCreateSecretsPreflight fails loud when the TLS secret is requested but its
// inputs are missing -- before any apply runs (012:19-24).
func TestCreateSecretsPreflight(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(c *config.Config)
	}{
		{"cert/key fields unset", func(c *config.Config) { c.TLS.Cert = ""; c.TLS.CertKey = "" }},
		{"cert file missing", func(c *config.Config) {
			c.TLS.Cert = filepath.Join(t.TempDir(), "nope.crt")
			c.TLS.CertKey = filepath.Join(t.TempDir(), "nope.key")
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := adminCfg()
			cfg.TLS.ServerSecret = "solace-tls-secret"
			tc.mutate(cfg)
			rr := &recRunner{}
			c := NewCluster(rr, cfg, nil, nil)
			if err := c.CreateSecrets(context.Background()); err == nil {
				t.Error("CreateSecrets should fail the preflight")
			}
			if len(rr.calls) != 0 {
				t.Errorf("no apply must run when the preflight fails; got %d calls", len(rr.calls))
			}
		})
	}
}

func TestDeleteSecrets(t *testing.T) {
	t.Run("all three", func(t *testing.T) {
		cfg := adminCfg()
		cfg.TLS.ServerSecret = "solace-tls-secret"
		cfg.Image.PullSecret = "solace-image-pull"
		rr := &recRunner{}
		c := NewCluster(rr, cfg, nil, nil)
		if err := c.DeleteSecrets(context.Background()); err != nil {
			t.Fatalf("DeleteSecrets: %v", err)
		}
		wantNames := []string{"solace-admin-secret", "solace-tls-secret", "solace-image-pull"}
		if len(rr.calls) != len(wantNames) {
			t.Fatalf("DeleteSecrets made %d calls, want %d", len(rr.calls), len(wantNames))
		}
		for i, name := range wantNames {
			want := []string{"delete", "secret", name, "-n", "solace", "--ignore-not-found"}
			if got := rr.calls[i]; got.method != "Run" || !eqArgs(got.args, want) {
				t.Errorf("delete[%d] = %+v, want Run kubectl %v", i, got, want)
			}
		}
	})
	t.Run("admin only", func(t *testing.T) {
		rr := &recRunner{}
		c := NewCluster(rr, adminCfg(), nil, nil)
		if err := c.DeleteSecrets(context.Background()); err != nil {
			t.Fatalf("DeleteSecrets: %v", err)
		}
		if len(rr.calls) != 1 {
			t.Fatalf("DeleteSecrets (admin only) made %d calls, want 1", len(rr.calls))
		}
	})
}

func TestUpdateServerCertSecret(t *testing.T) {
	t.Run("applies TLS secret on stdin", func(t *testing.T) {
		dir := t.TempDir()
		crt := filepath.Join(dir, "tls.crt")
		key := filepath.Join(dir, "tls.key")
		writeFile(t, crt, "CERT\n")
		writeFile(t, key, "KEY\n")
		cfg := haCfg()
		cfg.TLS.ServerSecret = "solace-tls-secret"
		cfg.TLS.Cert = crt
		cfg.TLS.CertKey = key
		rr := &recRunner{}
		c := NewCluster(rr, cfg, nil, nil)
		if err := c.UpdateServerCertSecret(context.Background()); err != nil {
			t.Fatalf("UpdateServerCertSecret: %v", err)
		}
		got := rr.last()
		if got.method != "RunInput" || !eqArgs(got.args, []string{"apply", "-f", "-"}) {
			t.Fatalf("UpdateServerCertSecret argv = %+v", got)
		}
		if !strings.Contains(got.stdin, "kubernetes.io/tls") || !strings.Contains(got.stdin, "name: solace-tls-secret") {
			t.Errorf("stdin is not the TLS secret:\n%s", got.stdin)
		}
	})
	t.Run("errors without a secret name", func(t *testing.T) {
		rr := &recRunner{}
		c := NewCluster(rr, haCfg(), nil, nil) // no TLS.ServerSecret
		if err := c.UpdateServerCertSecret(context.Background()); err == nil {
			t.Error("UpdateServerCertSecret should fail when tls.serverSecret is unset")
		}
	})
}

// --- pure label helpers ----------------------------------------------------

func TestSplitLabel(t *testing.T) {
	cases := []struct {
		in            string
		key, val      string
		ok            bool
	}{
		{"topology.kubernetes.io/zone=z1", "topology.kubernetes.io/zone", "z1", true},
		{"zone: z1", "zone", "z1", true},
		{"zone:z1", "zone", "z1", true},
		{"  a = b ", "a", "b", true},
		{"nosep", "", "", false},
		{"=z1", "", "", false},
		{"zone=", "", "", false},
		{": v", "", "", false},
	}
	for _, tc := range cases {
		key, val, ok := splitLabel(tc.in)
		if ok != tc.ok || key != tc.key || val != tc.val {
			t.Errorf("splitLabel(%q) = (%q,%q,%v), want (%q,%q,%v)", tc.in, key, val, ok, tc.key, tc.val, tc.ok)
		}
	}
}

func TestIsBuiltinLabel(t *testing.T) {
	builtin := []string{"kubernetes.io/arch", "beta.kubernetes.io/os", "node.kubernetes.io/instance-type", "k8s.io/foo"}
	custom := []string{"topology.kubernetes.io/zone", "acme.com/tier", "zone"}
	for _, k := range builtin {
		if !isBuiltinLabel(k) {
			t.Errorf("isBuiltinLabel(%q) = false, want true", k)
		}
	}
	for _, k := range custom {
		if isBuiltinLabel(k) {
			t.Errorf("isBuiltinLabel(%q) = true, want false", k)
		}
	}
}

// --- LabelNodes ------------------------------------------------------------

// labelCluster builds a Cluster wired for interactive labelling: rr as runner, buf
// as the report sink, and stdin as the selection source.
func labelCluster(rr *recRunner, cfg *config.Config, stdin string) (*Cluster, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	return &Cluster{R: rr, Cfg: cfg, Out: buf, In: strings.NewReader(stdin)}, buf
}

func TestLabelNodesNoCustomLabels(t *testing.T) {
	rr := &recRunner{}
	c, buf := labelCluster(rr, saCfg(), "")
	if err := c.LabelNodes(context.Background()); err != nil {
		t.Fatalf("LabelNodes: %v", err)
	}
	if len(rr.calls) != 0 {
		t.Errorf("no cluster calls expected with no labels; got %d", len(rr.calls))
	}
	if !strings.Contains(buf.String(), "nothing to label") {
		t.Errorf("expected an early-exit message; got %q", buf.String())
	}
}

func TestLabelNodesBuiltinOnly(t *testing.T) {
	cfg := saCfg()
	cfg.K8s.Placement.LabelsPrimary = []string{"kubernetes.io/arch: amd64"}
	rr := &recRunner{}
	c, _ := labelCluster(rr, cfg, "")
	if err := c.LabelNodes(context.Background()); err != nil {
		t.Fatalf("LabelNodes: %v", err)
	}
	if len(rr.calls) != 0 {
		t.Errorf("built-in labels must not be applied; got %d calls", len(rr.calls))
	}
}

func TestLabelNodesMalformedAndUnsafe(t *testing.T) {
	cfg := saCfg()
	cfg.K8s.Placement.LabelsPrimary = []string{"nosep", "bad key=v"}
	rr := &recRunner{}
	c, buf := labelCluster(rr, cfg, "")
	if err := c.LabelNodes(context.Background()); err != nil {
		t.Fatalf("LabelNodes: %v", err)
	}
	if len(rr.calls) != 0 {
		t.Errorf("dropped labels must not reach the cluster; got %d calls", len(rr.calls))
	}
	out := buf.String()
	if !strings.Contains(out, "malformed") || !strings.Contains(out, "unsupported characters") {
		t.Errorf("expected warnings for both dropped labels; got %q", out)
	}
}

func TestLabelNodesHappyPath(t *testing.T) {
	cfg := saCfg()
	cfg.K8s.Placement.LabelsPrimary = []string{"topology.kubernetes.io/zone: z1"}
	rr := &recRunner{outQueue: [][]byte{
		[]byte("yes\n"),            // auth can-i update nodes
		[]byte("node-a\nnode-b\n"), // node list
	}}
	c, _ := labelCluster(rr, cfg, "1\n") // pick node-a
	if err := c.LabelNodes(context.Background()); err != nil {
		t.Fatalf("LabelNodes: %v", err)
	}
	// calls: Output(auth can-i), Output(get nodes), Run(label ...)
	if len(rr.calls) != 3 {
		t.Fatalf("LabelNodes made %d calls, want 3", len(rr.calls))
	}
	if a := rr.calls[0].args; a[0] != "auth" || a[1] != "can-i" || a[2] != "update" || a[3] != "nodes" {
		t.Errorf("first call should be the RBAC precheck; got %v", a)
	}
	got := rr.calls[2]
	want := []string{"label", "node", "node-a", "topology.kubernetes.io/zone=z1", "--overwrite"}
	if got.method != "Run" || !eqArgs(got.args, want) {
		t.Errorf("label call = %+v, want Run kubectl %v", got, want)
	}
}

func TestLabelNodesReprompt(t *testing.T) {
	cfg := saCfg()
	cfg.K8s.Placement.LabelsPrimary = []string{"zone: z2"}
	rr := &recRunner{outQueue: [][]byte{
		[]byte("yes\n"),
		[]byte("node-a\nnode-b\n"),
	}}
	c, buf := labelCluster(rr, cfg, "9\nx\n2\n") // out-of-range, non-numeric, then node-b
	if err := c.LabelNodes(context.Background()); err != nil {
		t.Fatalf("LabelNodes: %v", err)
	}
	got := rr.last()
	want := []string{"label", "node", "node-b", "zone=z2", "--overwrite"}
	if !eqArgs(got.args, want) {
		t.Errorf("after reprompt, label call = %v, want %v", got.args, want)
	}
	if strings.Count(buf.String(), "Invalid selection") != 2 {
		t.Errorf("expected two re-prompts; got %q", buf.String())
	}
}

func TestLabelNodesRBACDenied(t *testing.T) {
	cfg := saCfg()
	cfg.K8s.Placement.LabelsPrimary = []string{"zone: z1"}
	rr := &recRunner{outErr: errFake}
	c, _ := labelCluster(rr, cfg, "")
	if err := c.LabelNodes(context.Background()); err == nil {
		t.Fatal("LabelNodes should fail when the RBAC precheck fails")
	}
	for _, call := range rr.calls {
		if len(call.args) > 0 && call.args[0] == "label" {
			t.Error("no node must be labelled after an RBAC denial")
		}
	}
}

func TestLabelNodesEOFNoSelection(t *testing.T) {
	cfg := saCfg()
	cfg.K8s.Placement.LabelsPrimary = []string{"zone: z1"}
	rr := &recRunner{outQueue: [][]byte{
		[]byte("yes\n"),
		[]byte("node-a\n"),
	}}
	c, _ := labelCluster(rr, cfg, "") // EOF before any selection
	if err := c.LabelNodes(context.Background()); err == nil {
		t.Error("LabelNodes should fail when no node selection is provided")
	}
}
