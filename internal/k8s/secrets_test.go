package k8s

import (
	"bytes"
	"encoding/base64"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"solace/internal/config"
)

// update regenerates the testdata goldens for the whole k8s package (secrets,
// operator, ...). Run `go test ./internal/k8s -update` only after eyeballing the
// diff; the committed goldens are the reviewed expected output.
var update = flag.Bool("update", false, "regenerate golden files in testdata/")

// sampleFixture is the shared env template, reused as the golden fixture so one file
// drives every renderer (matches internal/render's approach).
const sampleFixture = "../../env/sample.yaml"

func loadK8s(t *testing.T) *config.Config {
	t.Helper()
	c, err := config.Load(sampleFixture, config.K8s)
	if err != nil {
		t.Fatalf("load %s under k8s: %v", sampleFixture, err)
	}
	return c
}

// checkGolden compares got against testdata/<file>, or rewrites it under -update.
func checkGolden(t *testing.T, file string, got []byte) {
	t.Helper()
	golden := filepath.Join("testdata", file)
	if *update {
		if err := os.WriteFile(golden, got, 0o644); err != nil {
			t.Fatalf("write golden %s: %v", golden, err)
		}
		return
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden %s (regenerate: go test ./internal/k8s -update): %v", golden, err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("%s mismatch\n--- got ---\n%s\n--- want ---\n%s", file, got, want)
	}
}

func TestSecretGoldens(t *testing.T) {
	cases := []struct {
		name string
		file string
		gen  func(t *testing.T) []byte
	}{
		{
			name: "admin secret",
			file: "admin_secret.golden",
			gen: func(t *testing.T) []byte {
				cfg := loadK8s(t)
				cfg.Admin.UserPasswords = []string{"appuser=apppass"} // exercise the per-user branch
				b, err := AdminSecret(cfg)
				if err != nil {
					t.Fatalf("AdminSecret: %v", err)
				}
				return b
			},
		},
		{
			name: "tls secret",
			file: "tls_secret.golden",
			gen: func(t *testing.T) []byte {
				cfg := loadK8s(t)
				dir := t.TempDir()
				crt := filepath.Join(dir, "tls.crt")
				ca := filepath.Join(dir, "ca.crt")
				key := filepath.Join(dir, "tls.key")
				writeFile(t, crt, "CERTDATA\n")
				writeFile(t, ca, "CADATA\n")
				writeFile(t, key, "KEYDATA\n")
				cfg.TLS.Cert = crt
				cfg.TLS.CAs = []string{ca}
				cfg.TLS.CertKey = key
				b, err := TLSSecret(cfg)
				if err != nil {
					t.Fatalf("TLSSecret: %v", err)
				}
				return b
			},
		},
		{
			name: "docker registry secret",
			file: "docker_registry_secret.golden",
			gen: func(t *testing.T) []byte {
				b, err := DockerRegistrySecret(loadK8s(t))
				if err != nil {
					t.Fatalf("DockerRegistrySecret: %v", err)
				}
				return b
			},
		},
		{
			name: "operator regcred",
			file: "operator_regcred.golden",
			gen: func(t *testing.T) []byte {
				b, err := operatorRegcred(loadK8s(t), "pubsubplus-operator-system")
				if err != nil {
					t.Fatalf("operatorRegcred: %v", err)
				}
				return b
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			checkGolden(t, tc.file, tc.gen(t))
		})
	}
}

// TestAdminSecretDecodes proves the base64 data round-trips to the plaintext
// secrets (the golden guards format; this guards semantics).
func TestAdminSecretDecodes(t *testing.T) {
	cfg := loadK8s(t)
	got, err := AdminSecret(cfg)
	if err != nil {
		t.Fatalf("AdminSecret: %v", err)
	}
	if dec := decodeDataValue(t, got, "username_admin_password"); dec != "CHANGE-ME-admin" {
		t.Errorf("admin password decodes to %q", dec)
	}
	if dec := decodeDataValue(t, got, "username_monitor_password"); dec != "CHANGE-ME-monitor" {
		t.Errorf("monitor password decodes to %q", dec)
	}
}

func TestAdminSecretErrors(t *testing.T) {
	base := loadK8s(t)
	cases := []struct {
		name   string
		mutate func(c *config.Config)
	}{
		{"empty admin pass", func(c *config.Config) { c.Admin.Pass = "" }},
		{"empty secret name", func(c *config.Config) { c.Admin.UserSecret = "" }},
		{"entry without '='", func(c *config.Config) { c.Admin.UserPasswords = []string{"noequals"} }},
		{"entry with empty user", func(c *config.Config) { c.Admin.UserPasswords = []string{"=nopass"} }},
		{"entry with bad user", func(c *config.Config) { c.Admin.UserPasswords = []string{"bad user=x"} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := *base
			tc.mutate(&cfg)
			if _, err := AdminSecret(&cfg); err == nil {
				t.Errorf("AdminSecret(%s) expected error", tc.name)
			}
		})
	}
}

func TestTLSSecretErrors(t *testing.T) {
	t.Run("missing cert fields", func(t *testing.T) {
		cfg := loadK8s(t)
		cfg.TLS.Cert = ""
		if _, err := TLSSecret(cfg); err == nil {
			t.Error("TLSSecret should fail when tls.cert is unset")
		}
	})
	t.Run("empty secret name", func(t *testing.T) {
		cfg := loadK8s(t)
		cfg.TLS.ServerSecret = ""
		if _, err := TLSSecret(cfg); err == nil {
			t.Error("TLSSecret should fail when tls.serverSecret is unset")
		}
	})
	t.Run("cert file missing", func(t *testing.T) {
		cfg := loadK8s(t)
		cfg.TLS.Cert = filepath.Join(t.TempDir(), "nope.crt")
		if _, err := TLSSecret(cfg); err == nil {
			t.Error("TLSSecret should fail when the cert file does not exist")
		}
	})
	t.Run("CA file missing", func(t *testing.T) {
		cfg := loadK8s(t)
		dir := t.TempDir()
		crt := filepath.Join(dir, "tls.crt")
		key := filepath.Join(dir, "tls.key")
		writeFile(t, crt, "C\n")
		writeFile(t, key, "K\n")
		cfg.TLS.Cert = crt
		cfg.TLS.CertKey = key
		cfg.TLS.CAs = []string{filepath.Join(dir, "missing-ca.crt")}
		if _, err := TLSSecret(cfg); err == nil {
			t.Error("TLSSecret should fail when a CA file does not exist")
		}
	})
	t.Run("key file missing", func(t *testing.T) {
		cfg := loadK8s(t)
		dir := t.TempDir()
		crt := filepath.Join(dir, "tls.crt")
		writeFile(t, crt, "C\n")
		cfg.TLS.Cert = crt
		cfg.TLS.CertKey = filepath.Join(dir, "missing.key")
		cfg.TLS.CAs = nil
		if _, err := TLSSecret(cfg); err == nil {
			t.Error("TLSSecret should fail when the key file does not exist")
		}
	})
}

func TestDockerRegistrySecretEmptyName(t *testing.T) {
	cfg := loadK8s(t)
	cfg.Image.PullSecret = ""
	if _, err := DockerRegistrySecret(cfg); err == nil {
		t.Error("DockerRegistrySecret should fail with an empty pull-secret name")
	}
}

// --- test helpers ----------------------------------------------------------

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// decodeDataValue finds `  <key>: <base64>` in a rendered secret and returns the
// decoded plaintext. Fails the test if the key is absent or not valid base64.
func decodeDataValue(t *testing.T, manifest []byte, key string) string {
	t.Helper()
	prefix := []byte("  " + key + ": ")
	for _, line := range bytes.Split(manifest, []byte("\n")) {
		if bytes.HasPrefix(line, prefix) {
			raw := bytes.TrimPrefix(line, prefix)
			dec, err := base64.StdEncoding.DecodeString(string(raw))
			if err != nil {
				t.Fatalf("data[%s] is not valid base64: %v", key, err)
			}
			return string(dec)
		}
	}
	t.Fatalf("data key %q not found in manifest:\n%s", key, manifest)
	return ""
}
