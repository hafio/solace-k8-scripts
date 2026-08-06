package k8s

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"solace/internal/config"
)

// secretManifest is a core/v1 Secret rendered to YAML with base64-encoded data,
// matching the shape of `kubectl create secret ... -o yaml`. Building the manifest
// in Go and applying it on stdin (`apply -f -`) is behavior-equivalent to the bash
// `create secret --from-literal=...` form (012) but keeps every secret value off
// the argv and out of the --dry-run echo, which the bash form leaked (012:26,36,39,
// 43). §3 hardening.
type secretManifest struct {
	name      string
	namespace string
	typ       string
	data      map[string][]byte
}

// render emits the manifest. Data keys are sorted so the output is deterministic
// and golden-testable; kubectl is order-insensitive.
func (s secretManifest) render() []byte {
	var b strings.Builder
	b.WriteString("apiVersion: v1\n")
	b.WriteString("kind: Secret\n")
	b.WriteString("metadata:\n")
	b.WriteString("  name: " + s.name + "\n")
	b.WriteString("  namespace: " + s.namespace + "\n")
	b.WriteString("type: " + s.typ + "\n")
	b.WriteString("data:\n")
	keys := make([]string, 0, len(s.data))
	for k := range s.data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		b.WriteString("  " + k + ": " + base64.StdEncoding.EncodeToString(s.data[k]) + "\n")
	}
	return []byte(b.String())
}

// secretKeyUserRE constrains the user portion of an admin.userPasswords entry: it
// lands in a Secret data key (username_<user>_password), a structured position, so a
// space or colon would corrupt the YAML. §4 escape-what-lands-in-structured-output.
var secretKeyUserRE = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// AdminSecret builds the Opaque secret holding broker credentials, porting the
// user-secret of 012:26-32: username_admin_password (mandatory), an optional
// username_monitor_password, and one username_<user>_password per admin.userPasswords
// "user=password" entry. Fails loud on an empty admin password or a malformed entry.
func AdminSecret(cfg *config.Config) ([]byte, error) {
	if cfg.Admin.Pass == "" {
		return nil, fmt.Errorf("admin.pass must be set to build the admin secret")
	}
	if cfg.Admin.UserSecret == "" {
		return nil, fmt.Errorf("admin.userSecret (the secret name) must be set")
	}
	data := map[string][]byte{
		"username_admin_password": []byte(cfg.Admin.Pass),
	}
	if cfg.Admin.MonitorPass != "" {
		data["username_monitor_password"] = []byte(cfg.Admin.MonitorPass)
	}
	for _, up := range cfg.Admin.UserPasswords {
		user, pass, ok := strings.Cut(up, "=")
		if !ok || user == "" {
			return nil, fmt.Errorf("invalid admin.userPasswords entry %q: expected user=password", up)
		}
		if !secretKeyUserRE.MatchString(user) {
			return nil, fmt.Errorf("invalid admin.userPasswords user %q: only letters, digits, '.', '_' and '-' are allowed", user)
		}
		data["username_"+user+"_password"] = []byte(pass)
	}
	return secretManifest{
		name:      cfg.Admin.UserSecret,
		namespace: cfg.K8s.Namespace,
		typ:       "Opaque",
		data:      data,
	}.render(), nil
}

// TLSSecret builds the kubernetes.io/tls secret from the configured server
// certificate, porting 012:39 / 051:32: tls.crt is the certificate followed by any
// trusted CAs (the bash `--cert <(cat cert cas)`), tls.key is the private key. Both
// files are read from disk here; the manifest is applied on stdin so the key never
// reaches an argv or the --dry-run echo (§3).
func TLSSecret(cfg *config.Config) ([]byte, error) {
	if cfg.TLS.Cert == "" || cfg.TLS.CertKey == "" {
		return nil, fmt.Errorf("tls.cert and tls.certKey must both be set to build the TLS secret")
	}
	if cfg.TLS.ServerSecret == "" {
		return nil, fmt.Errorf("tls.serverSecret (the secret name) must be set")
	}
	crt, err := os.ReadFile(cfg.TLS.Cert)
	if err != nil {
		return nil, fmt.Errorf("read tls.cert %q: %w", cfg.TLS.Cert, err)
	}
	for _, ca := range cfg.TLS.CAs {
		caBytes, err := os.ReadFile(ca)
		if err != nil {
			return nil, fmt.Errorf("read tls CA %q: %w", ca, err)
		}
		crt = append(crt, caBytes...)
	}
	key, err := os.ReadFile(cfg.TLS.CertKey)
	if err != nil {
		return nil, fmt.Errorf("read tls.certKey %q: %w", cfg.TLS.CertKey, err)
	}
	return secretManifest{
		name:      cfg.TLS.ServerSecret,
		namespace: cfg.K8s.Namespace,
		typ:       "kubernetes.io/tls",
		data: map[string][]byte{
			"tls.crt": crt,
			"tls.key": key,
		},
	}.render(), nil
}

// dockerAuthEntry is one registry credential in a .dockerconfigjson payload. The
// field order (username, password, auth) is fixed by struct order so the rendered
// secret is deterministic and golden-testable.
type dockerAuthEntry struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Auth     string `json:"auth"`
}

// dockerConfigJSON is the .dockerconfigjson document of an image-pull secret.
type dockerConfigJSON struct {
	Auths map[string]dockerAuthEntry `json:"auths"`
}

// dockerRegistrySecret builds a kubernetes.io/dockerconfigjson pull secret named
// name in namespace ns for cfg.Image's registry/user/pass, porting 012:43. It backs
// both the broker image-pull secret (DockerRegistrySecret) and the operator's fixed
// "regcred" (operatorRegcred, 010:29), which live in different namespaces.
func dockerRegistrySecret(name, ns string, cfg *config.Config) ([]byte, error) {
	if name == "" {
		return nil, fmt.Errorf("image-pull secret name must not be empty")
	}
	auth := base64.StdEncoding.EncodeToString([]byte(cfg.Image.User + ":" + cfg.Image.Pass))
	payload, err := json.Marshal(dockerConfigJSON{
		Auths: map[string]dockerAuthEntry{
			cfg.Image.Registry: {
				Username: cfg.Image.User,
				Password: cfg.Image.Pass,
				Auth:     auth,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("marshal dockerconfigjson: %w", err)
	}
	return secretManifest{
		name:      name,
		namespace: ns,
		typ:       "kubernetes.io/dockerconfigjson",
		data:      map[string][]byte{".dockerconfigjson": payload},
	}.render(), nil
}

// DockerRegistrySecret builds the broker image-pull secret (name = image.pullSecret)
// in the broker namespace.
func DockerRegistrySecret(cfg *config.Config) ([]byte, error) {
	return dockerRegistrySecret(cfg.Image.PullSecret, cfg.K8s.Namespace, cfg)
}

// operatorRegcred builds the operator's image-pull secret under the fixed name
// "regcred" in the operator namespace opNS (010:29).
func operatorRegcred(cfg *config.Config, opNS string) ([]byte, error) {
	return dockerRegistrySecret("regcred", opNS, cfg)
}
