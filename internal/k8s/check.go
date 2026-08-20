package k8s

import (
	"context"
	"fmt"
	"strings"

	"solace/internal/engine"
)

// defaultSCJSONPath selects the name of the StorageClass annotated as cluster
// default, porting 009:18. The dots in the annotation key are backslash-escaped as
// jsonpath requires; the whole value is one kubectl argv token (no shell involved).
const defaultSCJSONPath = `jsonpath={.items[?(@.metadata.annotations.storageclass\.kubernetes\.io/is-default-class=="true")].metadata.name}`

// isDryRun reports whether commands are being echoed rather than run (the Echo
// runner). The cluster-touching checks use this to skip assertions they cannot make
// without a live cluster, while still echoing the commands they would run. A real
// cluster never hits this path, so `no default StorageClass` still fails loud there.
func (c *Cluster) isDryRun() bool {
	_, ok := c.R.(engine.Echo)
	return ok
}

// Check is the preflight run before a deploy and standalone as `k8s check`: it
// prints the resolved configuration, confirms the API server is reachable, and
// validates the StorageClass the broker will bind. Ported from 001 (env report,
// minus the interactive "Press any key") and 009 (storage-class checks).
func (c *Cluster) Check(ctx context.Context) error {
	c.CheckEnv()
	if err := c.Reachable(ctx); err != nil {
		return err
	}
	c.checkOperatorNS(ctx)
	return c.CheckStorageClass(ctx)
}

// checkOperatorNS reports the namespace the operator actually runs in, which CheckEnv can
// only print as "(derived at runtime)": the discovery needs a live cluster, so it happens
// here, after Reachable. It never fails -- an operator that cannot be found means "not
// installed yet", which the default covers and `prep operator` fixes. A configured
// kubernetes.operator.namespace short-circuits in operatorNSOrigin, so the line costs a cluster
// round-trip only when the value has to be discovered.
func (c *Cluster) checkOperatorNS(ctx context.Context) {
	w := c.out()
	if c.isDryRun() {
		fmt.Fprintln(w, "  operator ns    : skipped (dry-run)")
		return
	}
	ns, origin := c.operatorNSOrigin(ctx)
	fmt.Fprintf(w, "  operator ns    : %s (%s)\n", ns, origin)
}

// Reachable probes the API server (001's kubectl availability check, strengthened
// to an actual server round-trip). Under --dry-run the Echo runner returns no error,
// so this passes and the command is shown.
func (c *Cluster) Reachable(ctx context.Context) error {
	if _, err := c.output(ctx, "version", "-o", "json"); err != nil {
		return fmt.Errorf("cannot reach the Kubernetes API server (check kubeconfig/context): %w", err)
	}
	return nil
}

// CheckEnv writes a human-readable summary of the resolved configuration to Out.
// Secrets are reported as set/MISSING only -- their values are never printed
// (§3). It never fails; it is a report.
func (c *Cluster) CheckEnv() {
	w := c.out()
	cfg := c.Cfg

	mode := "standalone (single broker)"
	if cfg.RedundancyEnabled() {
		mode = "HA redundancy group (primary + backup + monitor)"
	}

	fmt.Fprintln(w, "Solace broker deployment (Kubernetes):")
	fmt.Fprintf(w, "  name/namespace : %s / %s\n", cfg.K8s.Name, cfg.K8s.Namespace)
	// The cluster CLI is configurable (kubernetes.runtime), so report which one is in
	// play -- 001-check-env.sh:23 printed the resolved KUBE for the same reason.
	fmt.Fprintf(w, "  cluster cmd    : %s\n", cfg.K8s.Runtime)
	fmt.Fprintf(w, "  redundancy     : %s\n", mode)
	fmt.Fprintf(w, "  update strategy: %s\n", orNone(cfg.K8s.UpdateStrategy))
	fmt.Fprintf(w, "  image          : %s\n", cfg.Image.Ref())
	fmt.Fprintf(w, "  image pull     : secret=%s creds=%s\n",
		orNone(cfg.Image.PullSecret), setOrNone(cfg.Image.User != "" && cfg.Image.Pass != ""))

	opNS := cfg.K8s.Operator.Namespace
	if opNS == "" {
		opNS = "(derived at runtime)"
	}
	// operatorImage, not Operator.Image: the apply prefixes image.registry, and a report
	// naming an image the deploy will not pull is worse than no report. Same reason the
	// broker line above goes through cfg.Image.Ref().
	fmt.Fprintf(w, "  operator       : image=%s ns=%s cpu=%s mem=%s\n",
		operatorImage(cfg), opNS, orNone(cfg.K8s.Operator.CPU), orNone(cfg.K8s.Operator.Mem))
	// An empty WATCH_NAMESPACE does not mean "the broker namespace" -- it means the
	// operator watches every namespace in the cluster, which is the widest scope it has
	// and the one an operator is least likely to have chosen on purpose. It is only
	// reachable by setting watchBrokerNs: false with no watchNamespaces list, so the
	// report says what that produces instead of the reassuring opposite.
	fmt.Fprintf(w, "  operator watch : %s\n",
		orValue(watchNamespace(cfg), "(empty -- the operator watches ALL namespaces)"))

	fmt.Fprintf(w, "  storage        : class=%s msgNode=%s monNode=%s\n",
		orValue(cfg.K8s.Storage.Class, "(cluster default)"), cfg.K8s.Storage.MsgNode, orNone(cfg.K8s.Storage.MonNode))

	// user= is the literal "admin", not cfg.Admin.User: the operator reads the fixed
	// username_admin_password key out of the credentials Secret (AdminSecret), so that is
	// the broker's admin user whatever an env file says -- and validateK8s now refuses
	// any other value rather than letting it look effective here.
	fmt.Fprintf(w, "  admin          : user=admin secret=%s password=%s monitorPassword=%s extraUsers=%d\n",
		orNone(cfg.K8s.AdminSecret), setOrMissing(cfg.Admin.Pass), setOrNone(cfg.Admin.MonitorPass != ""), len(cfg.Admin.AdditionalUsers))

	if cfg.TLS.ServerSecret != "" {
		fmt.Fprintf(w, "  tls            : secret=%s cert=%s key=%s cas=%d\n",
			cfg.TLS.ServerSecret, orNone(cfg.TLS.Cert), setOrMissing(cfg.TLS.CertKey), len(cfg.TLS.CAs))
	} else {
		fmt.Fprintln(w, "  tls            : (not configured)")
	}

	fmt.Fprintf(w, "  loadBalancer   : ip=%s ipPool=%s\n", orNone(cfg.K8s.LoadBalancer.IP), orNone(cfg.K8s.LoadBalancer.IPPool))
	fmt.Fprintf(w, "  placement      : labels p=%d b=%d m=%d antiAffinityNs=%d\n",
		len(cfg.K8s.Placement.LabelsPrimary), len(cfg.K8s.Placement.LabelsBackup),
		len(cfg.K8s.Placement.LabelsMonitor), len(cfg.K8s.Placement.AntiAffinityNS))
}

// CheckStorageClass validates the StorageClass the broker PVCs will bind, porting
// 009:26-41: it must use volumeBindingMode=WaitForFirstConsumer (so the PV lands in
// the pod's zone) and allowVolumeExpansion=true (so storage can grow). The class is
// the configured kubernetes.storage.class, or the cluster default resolved via 009:18.
func (c *Cluster) CheckStorageClass(ctx context.Context) error {
	name, err := c.resolveStorageClass(ctx)
	if err != nil {
		return err
	}
	if c.isDryRun() {
		fmt.Fprintln(c.out(), "  storage class  : skipped (dry-run)")
		return nil
	}
	if name == "" {
		return fmt.Errorf("no default StorageClass found and kubernetes.storage.class is not set (009); set kubernetes.storage.class or mark a StorageClass default")
	}

	binding, err := c.scColumn(ctx, name, ".volumeBindingMode")
	if err != nil {
		return fmt.Errorf("reading StorageClass %q: %w", name, err)
	}
	expansion, err := c.scColumn(ctx, name, ".allowVolumeExpansion")
	if err != nil {
		return fmt.Errorf("reading StorageClass %q: %w", name, err)
	}

	w := c.out()
	fmt.Fprintf(w, "  storage class  : %s (volumeBindingMode=%s allowVolumeExpansion=%s)\n", name, binding, expansion)
	if binding != "WaitForFirstConsumer" || expansion != "true" {
		return fmt.Errorf("StorageClass %q is unsuitable: need volumeBindingMode=WaitForFirstConsumer (got %q) and allowVolumeExpansion=true (got %q)",
			name, binding, expansion)
	}
	return nil
}

// resolveStorageClass returns the configured class if set, else the cluster default
// resolved by annotation (009:18). Multiple defaults are an error rather than a
// silent pick, since the choice would be non-deterministic.
func (c *Cluster) resolveStorageClass(ctx context.Context) (string, error) {
	if c.Cfg.K8s.Storage.Class != "" {
		return c.Cfg.K8s.Storage.Class, nil
	}
	out, err := c.output(ctx, "get", "sc", "-o", defaultSCJSONPath)
	if err != nil {
		return "", fmt.Errorf("resolving default StorageClass: %w", err)
	}
	fields := strings.Fields(string(out))
	if len(fields) > 1 {
		return "", fmt.Errorf("multiple default StorageClasses found (%s); set kubernetes.storage.class explicitly", strings.Join(fields, ", "))
	}
	if len(fields) == 1 {
		return fields[0], nil
	}
	return "", nil
}

// scColumn reads one field of a StorageClass via custom-columns. kubectl prints
// "<none>" (not empty) for an absent field, which is preserved so the caller can
// report it rather than mistaking it for a missing class.
func (c *Cluster) scColumn(ctx context.Context, name, field string) (string, error) {
	out, err := c.output(ctx, "get", "sc", name, "-o", "custom-columns=V:"+field, "--no-headers")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// --- small report formatters ------------------------------------------------

func orNone(s string) string {
	if s == "" {
		return "(none)"
	}
	return s
}

func orValue(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func setOrMissing(s string) string {
	if s == "" {
		return "MISSING"
	}
	return "set"
}

func setOrNone(present bool) string {
	if present {
		return "set"
	}
	return "(none)"
}
