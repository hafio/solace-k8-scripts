package k8s

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"solace/internal/config"
)

// out returns the report/prompt sink, defaulting to stdout when unset.
func (c *Cluster) out() io.Writer {
	if c.Out != nil {
		return c.Out
	}
	return os.Stdout
}

// in returns the prompt source, defaulting to stdin when unset. Only the
// interactive operations (LabelNodes) read it.
func (c *Cluster) in() io.Reader {
	if c.In != nil {
		return c.In
	}
	return os.Stdin
}

// namespaceManifest is a minimal core/v1 Namespace. Applying it on stdin is the
// idempotent equivalent of the bash `create ns --dry-run=client -o yaml | apply -f -`
// (011:15): apply creates it if absent and no-ops if it already exists.
func namespaceManifest(ns string) []byte {
	return []byte("apiVersion: v1\nkind: Namespace\nmetadata:\n  name: " + ns + "\n")
}

// CreateNamespace applies the broker namespace (011). Idempotent via `apply`.
func (c *Cluster) CreateNamespace(ctx context.Context) error {
	c.logf("creating namespace %s", c.ns())
	return c.apply(ctx, namespaceManifest(c.ns()))
}

// DeleteNamespace removes the broker namespace (111). --ignore-not-found makes a
// repeat teardown a no-op rather than an error.
func (c *Cluster) DeleteNamespace(ctx context.Context) error {
	c.logf("deleting namespace %s", c.ns())
	return c.kubectl(ctx, "delete", "namespace", c.ns(), "--ignore-not-found")
}

// secretPreflight fails loud before any manifest is built when the TLS server
// secret is requested but its cert/key inputs are missing, porting the guard of
// 012:19-24 so the operator does not later fail to mount a half-built secret. The
// admin secret's own guards live in AdminSecret.
func (c *Cluster) secretPreflight() error {
	if c.Cfg.TLS.ServerSecret == "" {
		return nil
	}
	if c.Cfg.TLS.Cert == "" || c.Cfg.TLS.CertKey == "" {
		return fmt.Errorf("tls.serverSecret %q is set but tls.cert and tls.certKey are not both configured", c.Cfg.TLS.ServerSecret)
	}
	for _, f := range []string{c.Cfg.TLS.Cert, c.Cfg.TLS.CertKey} {
		if _, err := os.Stat(f); err != nil {
			return fmt.Errorf("tls certificate input %q is not readable: %w", f, err)
		}
	}
	return nil
}

// CreateSecrets builds every applicable secret (admin always; TLS when
// tls.serverSecret is set; the image-pull secret when image.pullSecret is set),
// joins them into one multi-doc manifest, and applies it on stdin -- porting 012
// while keeping every secret value off the argv and out of the --dry-run echo (§3).
// The whole manifest is built before the first apply, so a builder error aborts
// cleanly without leaving a partially-applied secret set.
func (c *Cluster) CreateSecrets(ctx context.Context) error {
	if err := c.secretPreflight(); err != nil {
		return err
	}
	docs := make([][]byte, 0, 3)

	admin, err := AdminSecret(c.Cfg)
	if err != nil {
		return err
	}
	docs = append(docs, admin)

	if c.Cfg.TLS.ServerSecret != "" {
		tls, err := TLSSecret(c.Cfg)
		if err != nil {
			return err
		}
		docs = append(docs, tls)
	}

	if c.Cfg.Image.PullSecret != "" {
		pull, err := DockerRegistrySecret(c.Cfg)
		if err != nil {
			return err
		}
		docs = append(docs, pull)
	}

	c.logf("creating %d secret(s) in %s", len(docs), c.ns())
	return c.apply(ctx, joinManifests(docs))
}

// DeleteSecrets removes the secrets CreateSecrets created (112): the admin secret
// always, the TLS and image-pull secrets only when their names are configured. All
// use --ignore-not-found so a partial or repeat teardown is not an error.
func (c *Cluster) DeleteSecrets(ctx context.Context) error {
	names := []string{c.Cfg.Admin.UserSecret}
	if c.Cfg.TLS.ServerSecret != "" {
		names = append(names, c.Cfg.TLS.ServerSecret)
	}
	if c.Cfg.Image.PullSecret != "" {
		names = append(names, c.Cfg.Image.PullSecret)
	}
	for _, name := range names {
		if name == "" {
			continue
		}
		c.logf("deleting secret %s", name)
		if err := c.kubectl(ctx, "delete", "secret", name, "-n", c.ns(), "--ignore-not-found"); err != nil {
			return err
		}
	}
	return nil
}

// UpdateServerCertSecret rebuilds the kubernetes.io/tls secret from the current
// certificate files and applies it on stdin, porting the secret-managed path of
// 051-load-server-cert.sh (051:28-38). Applying on stdin replaces the bash
// `create secret tls --dry-run|apply`, so the private key never reaches an argv or
// the --dry-run echo (§3). The broker re-reads the secret; no pod restart here.
func (c *Cluster) UpdateServerCertSecret(ctx context.Context) error {
	if c.Cfg.TLS.ServerSecret == "" {
		return fmt.Errorf("tls.serverSecret must be set to update the server-certificate secret")
	}
	manifest, err := TLSSecret(c.Cfg)
	if err != nil {
		return err
	}
	c.logf("updating server-certificate secret %s", c.Cfg.TLS.ServerSecret)
	return c.apply(ctx, manifest)
}

// joinManifests concatenates rendered YAML documents with a `---` separator so
// they apply as one multi-doc stream.
func joinManifests(docs [][]byte) []byte {
	parts := make([]string, len(docs))
	for i, d := range docs {
		parts[i] = strings.TrimRight(string(d), "\n")
	}
	return []byte(strings.Join(parts, "\n---\n") + "\n")
}

// --- node labelling (013) --------------------------------------------------

// builtinLabelPrefixes are Kubernetes-managed label namespaces that the operator
// never asks the user to set on a node; entries under them are silently skipped so
// the prompt only offers labels the user actually configured (013).
var builtinLabelPrefixes = []string{
	"kubernetes.io/",
	"k8s.io/",
	"node.kubernetes.io/",
	"beta.kubernetes.io/",
}

// isBuiltinLabel reports whether a label key sits under a Kubernetes-managed
// prefix and should not be applied by hand.
func isBuiltinLabel(key string) bool {
	for _, p := range builtinLabelPrefixes {
		if strings.HasPrefix(key, p) {
			return true
		}
	}
	return false
}

// splitLabel parses a configured node-label entry into a key and value. It accepts
// both the kubectl form `key=value` and the YAML/bash form `key: value`, trimming
// surrounding space; ok is false when either side is empty or no separator is
// present. The bash port only handled `key: value` (013), so `=` support is a
// deliberate convenience.
func splitLabel(entry string) (key, val string, ok bool) {
	var i int
	if i = strings.IndexByte(entry, '='); i < 0 {
		i = strings.IndexByte(entry, ':')
	}
	if i < 0 {
		return "", "", false
	}
	key = strings.TrimSpace(entry[:i])
	val = strings.TrimSpace(entry[i+1:])
	if key == "" || val == "" {
		return "", "", false
	}
	return key, val, true
}

// labelTokenRE constrains a label key/value to Kubernetes' own charset. It is a
// defensive check on config that reaches the kubectl argv: it rejects whitespace,
// shell metacharacters and a leading '-' (which kubectl would read as a flag). §3.
var labelTokenRE = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9._/-]*[A-Za-z0-9])?$`)

func validLabelToken(s string) bool { return labelTokenRE.MatchString(s) }

// labelKV is one validated custom label destined for a node.
type labelKV struct{ key, val string }

// rolePlacementLabels returns the configured node labels for a role.
func rolePlacementLabels(cfg *config.Config, role config.Role) []string {
	switch role {
	case config.Backup:
		return cfg.K8s.Placement.LabelsBackup
	case config.Monitor:
		return cfg.K8s.Placement.LabelsMonitor
	default:
		return cfg.K8s.Placement.LabelsPrimary
	}
}

// roleName is the human label used in the node-selection prompt.
func roleName(role config.Role) string {
	switch role {
	case config.Backup:
		return "backup"
	case config.Monitor:
		return "monitor"
	default:
		return "primary"
	}
}

// customLabels returns, per broker role present in this deployment, the user labels
// that are safe to apply: malformed entries and Kubernetes-managed prefixes are
// dropped with a warning so they neither prompt nor reach the argv.
func (c *Cluster) customLabels() map[config.Role][]labelKV {
	out := map[config.Role][]labelKV{}
	for _, role := range HARoles(c.Cfg) {
		for _, entry := range rolePlacementLabels(c.Cfg, role) {
			key, val, ok := splitLabel(entry)
			if !ok {
				fmt.Fprintf(c.out(), "  [WARN] skipping malformed node label %q for %s\n", entry, roleName(role))
				continue
			}
			if isBuiltinLabel(key) {
				continue // managed by Kubernetes; not user-applied
			}
			if !validLabelToken(key) || !validLabelToken(val) {
				fmt.Fprintf(c.out(), "  [WARN] skipping node label with unsupported characters %q for %s\n", entry, roleName(role))
				continue
			}
			out[role] = append(out[role], labelKV{key, val})
		}
	}
	return out
}

// nodeNames lists cluster node names via a name-only custom-columns query.
func (c *Cluster) nodeNames(ctx context.Context) ([]string, error) {
	raw, err := c.output(ctx, "get", "nodes", "-o", "custom-columns=NAME:.metadata.name", "--no-headers")
	if err != nil {
		return nil, fmt.Errorf("listing cluster nodes: %w", err)
	}
	var names []string
	for _, line := range strings.Split(string(raw), "\n") {
		if n := strings.TrimSpace(line); n != "" {
			names = append(names, n)
		}
	}
	return names, nil
}

// LabelNodes interactively applies the configured custom node labels, porting 013.
// It early-exits when nothing is configured (so `up` can call it unconditionally),
// prechecks the RBAC to update nodes before prompting, then for each role prompts
// the operator to pick a node from the cluster's node list and runs
// `kubectl label node <node> <key>=<val> --overwrite`. A single label failure is
// reported and skipped, not fatal, matching the bash loop.
func (c *Cluster) LabelNodes(ctx context.Context) error {
	custom := c.customLabels()
	if len(custom) == 0 {
		fmt.Fprintln(c.out(), "No custom node labels configured; nothing to label.")
		return nil
	}
	if _, err := c.output(ctx, "auth", "can-i", "update", "nodes"); err != nil {
		return fmt.Errorf("current context cannot label nodes (kubectl auth can-i update nodes): %w", err)
	}
	nodes, err := c.nodeNames(ctx)
	if err != nil {
		return err
	}
	if len(nodes) == 0 {
		return fmt.Errorf("no cluster nodes found to label")
	}

	reader := bufio.NewReader(c.in())
	for _, role := range HARoles(c.Cfg) {
		labels := custom[role]
		if len(labels) == 0 {
			continue
		}
		node, err := c.promptNode(reader, role, nodes)
		if err != nil {
			return err
		}
		for _, kv := range labels {
			arg := kv.key + "=" + kv.val
			if err := c.kubectl(ctx, "label", "node", node, arg, "--overwrite"); err != nil {
				fmt.Fprintf(c.out(), "  [ERROR] failed to apply %s to %s: %v\n", arg, node, err)
				continue
			}
			fmt.Fprintf(c.out(), "  [ OK ] labelled %s with %s\n", node, arg)
		}
	}
	return nil
}

// promptNode asks the operator to choose a node for a role from nodes, re-prompting
// on invalid input. EOF with no valid selection is a hard error rather than a
// silent default, so a mis-piped `up` cannot label the wrong node.
func (c *Cluster) promptNode(r *bufio.Reader, role config.Role, nodes []string) (string, error) {
	for {
		fmt.Fprintf(c.out(), "Select the node for the %s broker:\n", roleName(role))
		for i, n := range nodes {
			fmt.Fprintf(c.out(), "  %d) %s\n", i+1, n)
		}
		fmt.Fprint(c.out(), "> ")

		line, rerr := r.ReadString('\n')
		choice, cerr := strconv.Atoi(strings.TrimSpace(line))
		if cerr == nil && choice >= 1 && choice <= len(nodes) {
			return nodes[choice-1], nil
		}
		if rerr != nil {
			return "", fmt.Errorf("no valid node selection for the %s role", roleName(role))
		}
		fmt.Fprintln(c.out(), "Invalid selection; enter the number of a listed node.")
	}
}
