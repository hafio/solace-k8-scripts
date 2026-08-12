package config

import (
	"fmt"
	"regexp"
	"strings"
)

// Validate checks mandatory and enum fields for the given platform, mirroring
// the mandatory-vars and enum checks in the two bash bootstraps. It fails loud
// with an actionable message listing every offending field at once.
func (c *Config) Validate(p Platform) error {
	// Redundancy is a shared enum on every platform.
	switch c.Redundancy {
	case "yes", "no":
	default:
		return fmt.Errorf("redundancy must be 'yes' or 'no' (got: %q)", c.Redundancy)
	}

	// The platform CLI is user-supplied and reaches os/exec, so it is checked on
	// every platform before anything can shell out.
	if err := validateCommand("k8s.runtime", c.K8s.Runtime); err != nil {
		return err
	}
	if p.IsContainer() {
		if err := validateCommand(platformKey(p)+".runtime", c.ContainerRuntime(p)); err != nil {
			return err
		}
	}

	switch p {
	case K8s:
		return c.validateK8s()
	case Docker, Podman:
		return c.validateContainer(p)
	default:
		return fmt.Errorf("unknown platform %q", p)
	}
}

// validateCommand checks a Command's tokens before it can reach os/exec. exec
// never goes through a shell, so a metacharacter here is an ordinary filename
// character rather than an injection; what this catches is a token that could
// only ever fail obscurely at exec time -- an empty argument, or a control
// character carried in from a converted bash file (§4a).
//
// An empty Command is not an error: ApplyDefaults runs before Validate on every
// path and fills the platform default, so "empty" means "unset" exactly as it
// does for every setDefault field in this schema.
func validateCommand(field string, cmd Command) error {
	for i, tok := range cmd {
		if tok == "" {
			return fmt.Errorf("%s[%d] is an empty argument; remove it or quote the intended value", field, i)
		}
		if j := strings.IndexFunc(tok, isCtrl); j >= 0 {
			return fmt.Errorf("%s[%d] contains a control character (0x%02x) at offset %d: %q",
				field, i, tok[j], j, tok)
		}
	}
	return nil
}

// isCtrl reports the ASCII control characters (including NUL and DEL), which can
// never legitimately appear in a command name or argument here.
func isCtrl(r rune) bool { return r < 0x20 || r == 0x7f }

// identRE constrains the identifiers that reach a rendered artifact in a
// structural position: a compose service key, a `container_name`, a systemd
// ContainerName=/HostName=, and the systemd Environment= keys built from node
// names. A colon, '=' or newline there produces a broken artifact instead of an
// error, so the check belongs here where it can name the field (§4a).
//
// It is a package-local copy of broker.nameRE / k8s.secretKeyUserRE, which are
// already the same expression: config sits below both in the import graph, and
// the house convention is one small copy per package over a shared micro-package.
var identRE = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// runUserRE allows the container runtime's `uid[:gid]` form, which identRE alone
// would reject -- the default "0:0" contains a colon.
var runUserRE = regexp.MustCompile(`^[A-Za-z0-9._-]+(:[A-Za-z0-9._-]+)?$`)

// validIdent checks one identifier, ignoring an empty value: emptiness is
// requireAll's job, and several of these fields are legitimately empty (the
// backup/monitor rows in standalone).
func validIdent(field, value string) error {
	if value == "" {
		return nil
	}
	if !identRE.MatchString(value) {
		return fmt.Errorf("%s %q is invalid: only letters, digits, '.', '_' and '-' are allowed "+
			"(it becomes a container name, host name and systemd/compose key)", field, value)
	}
	return nil
}

// keyValueEntries is a named list of user-supplied "key: value" fragments, kept as
// a slice rather than a map so a failure message is deterministic.
type keyValueEntries struct {
	field   string
	entries []string
}

// requireKeyValue checks that each entry carries the "key: value" shape the
// renderer emits as a YAML mapping entry. The renderer quotes both halves, so any
// character is safe once the shape holds -- what cannot be recovered is an entry
// with no key at all.
func requireKeyValue(groups []keyValueEntries) error {
	for _, g := range groups {
		for i, entry := range g.entries {
			key, _, ok := strings.Cut(entry, ":")
			if !ok || strings.TrimSpace(key) == "" {
				return fmt.Errorf("%s[%d] = %q is not a \"key: value\" entry; write it as `key: value`", g.field, i, entry)
			}
		}
	}
	return nil
}

func (c *Config) validateK8s() error {
	missing := requireAll(map[string]string{
		"k8s.name":            c.K8s.Name,
		"k8s.namespace":       c.K8s.Namespace,
		"image.repo":          c.Image.Repo,
		"image.tag":           c.Image.Tag,
		"k8s.storage.msgNode": c.K8s.Storage.MsgNode,
		"admin.pass":          c.Admin.Pass, // hardening: no hardcoded default password
	})
	if len(missing) > 0 {
		return missingErr(missing)
	}
	switch c.K8s.UpdateStrategy {
	case "automatedRolling", "manualPodRestart":
	default:
		return fmt.Errorf("k8s.updateStrategy must be 'automatedRolling' or 'manualPodRestart' (got: %q)", c.K8s.UpdateStrategy)
	}
	switch c.Image.PullPolicy {
	case "", "Always", "IfNotPresent", "Never":
	default:
		return fmt.Errorf("image.pullPolicy must be 'Always', 'IfNotPresent' or 'Never' (got: %q)", c.Image.PullPolicy)
	}
	pl := c.K8s.Placement
	if err := requireKeyValue([]keyValueEntries{
		{"k8s.loadBalancer.annotations", c.K8s.LoadBalancer.Annotations},
		{"k8s.placement.labelsPrimary", pl.LabelsPrimary},
		{"k8s.placement.labelsBackup", pl.LabelsBackup},
		{"k8s.placement.labelsMonitor", pl.LabelsMonitor},
	}); err != nil {
		return err
	}
	return validatePlacementAffinity(pl)
}

// nodeMatchOperators are the node-label match operators Kubernetes accepts. The
// value reaches the manifest unquoted as a bare enum, so it is checked here.
var nodeMatchOperators = map[string]bool{
	"In": true, "NotIn": true, "Exists": true, "DoesNotExist": true, "Gt": true, "Lt": true,
}

// validatePlacementAffinity checks the additive affinity blocks. It deliberately
// does not police weight bounds, matching the existing laxness on
// antiAffinityWeight -- what it catches is a value the API server would reject
// with a far less obvious message, or a term with no topology to spread over.
func validatePlacementAffinity(pl Placement) error {
	for i, term := range pl.NodeAffinity.Preferred {
		if err := validateMatchExprs(fmt.Sprintf("k8s.placement.nodeAffinity.preferred[%d].match", i), term.Match); err != nil {
			return err
		}
	}
	if err := validateMatchExprs("k8s.placement.nodeAffinity.required", pl.NodeAffinity.Required); err != nil {
		return err
	}
	for _, group := range []struct {
		field string
		terms []PodAffinityTerm
	}{
		{"k8s.placement.podAffinity", pl.PodAffinity},
		{"k8s.placement.podAntiAffinity", pl.PodAntiAffinity},
	} {
		for i, term := range group.terms {
			if strings.TrimSpace(term.TopologyKey) == "" {
				return fmt.Errorf("%s[%d].topologyKey must be set (e.g. kubernetes.io/hostname)", group.field, i)
			}
		}
	}
	return nil
}

func validateMatchExprs(field string, exprs []NodeMatchExpr) error {
	for i, e := range exprs {
		if strings.TrimSpace(e.Key) == "" {
			return fmt.Errorf("%s[%d].key must be set", field, i)
		}
		if !nodeMatchOperators[e.Operator] {
			return fmt.Errorf("%s[%d].operator %q is invalid: expected In, NotIn, Exists, DoesNotExist, Gt or Lt",
				field, i, e.Operator)
		}
		needsValues := e.Operator == "In" || e.Operator == "NotIn" || e.Operator == "Gt" || e.Operator == "Lt"
		if needsValues && len(e.Values) == 0 {
			return fmt.Errorf("%s[%d].values must not be empty for operator %s", field, i, e.Operator)
		}
	}
	return nil
}

func (c *Config) validateContainer(p Platform) error {
	req := map[string]string{
		"image.repo":         c.Image.Repo,
		"image.tag":          c.Image.Tag,
		"admin.pass":         c.Admin.Pass,
		"nodes.primary.name": c.Nodes.Primary.Name,
	}
	// The backup/monitor rows + primary IP are required only for the HA group.
	if c.RedundancyEnabled() {
		req["nodes.primary.ip"] = c.Nodes.Primary.IP
		req["nodes.backup.name"] = c.Nodes.Backup.Name
		req["nodes.backup.ip"] = c.Nodes.Backup.IP
		req["nodes.monitor.name"] = c.Nodes.Monitor.Name
		req["nodes.monitor.ip"] = c.Nodes.Monitor.IP
	}
	// Data dir lives in the platform's container block.
	dataKey := "docker.container.dataDir"
	if p == Podman {
		dataKey = "podman.container.dataDir"
	}
	req[dataKey] = c.ContainerBlock(p).DataDir

	if missing := requireAll(req); len(missing) > 0 {
		return missingErr(missing)
	}

	// These reach the compose/quadlet artifact in structural positions, so they are
	// format-checked here rather than being allowed to produce a broken artifact.
	cb := c.ContainerBlock(p)
	for _, f := range []struct{ field, value string }{
		{platformKey(p) + ".container.name", cb.Name},
		{"nodes.primary.name", c.Nodes.Primary.Name},
		{"nodes.backup.name", c.Nodes.Backup.Name},
		{"nodes.monitor.name", c.Nodes.Monitor.Name},
	} {
		if err := validIdent(f.field, f.value); err != nil {
			return err
		}
	}
	if u := cb.RunUser; u != "" && !runUserRE.MatchString(u) {
		return fmt.Errorf("%s.container.runUser %q is invalid: expected uid[:gid] using only letters, digits, '.', '_' and '-'",
			platformKey(p), u)
	}

	if hc := cb.HealthCheck; hc.Enabled {
		if len(hc.Cmd) > 0 {
			// An explicit probe is the operator's own; it only gets the exec-boundary
			// check, not the version gate.
			if err := validateCommand(platformKey(p)+".container.healthCheck.cmd", Command(hc.Cmd)); err != nil {
				return err
			}
		} else if err := c.checkHealthCheckVersion(p); err != nil {
			return err
		}
	}

	net := c.NetworkBlock(p)
	switch net.Mode {
	case "host":
	case "bridge":
		if len(net.Ports) == 0 {
			// ApplyDefaults fills this list, so reaching here means Validate ran on
			// its own (a hand-built config) -- say so rather than implying the user
			// must always list ports.
			return fmt.Errorf("%s.network.mode=bridge requires a non-empty ports list (host:container entries); "+
				"config.Load fills the default set automatically", platformKey(p))
		}
	default:
		return fmt.Errorf("%s.network.mode must be 'host' or 'bridge' (got: %q)", platformKey(p), net.Mode)
	}

	if p == Docker {
		switch c.Docker.Mode {
		case "compose":
		case "run":
			// run mode was removed: a bare `docker run` cannot recreate an existing
			// container, so redeploying after an image.tag bump hard-failed on a
			// name conflict where compose recreates cleanly.
			return fmt.Errorf("docker.mode 'run' was removed; docker deploys through compose only -- " +
				"set docker.mode: compose (or drop the key), and set docker.compose if this host " +
				"uses the standalone 'docker-compose' binary")
		default:
			return fmt.Errorf("docker.mode must be 'compose' (got: %q)", c.Docker.Mode)
		}
		if err := validateCommand("docker.compose", c.Docker.Compose); err != nil {
			return err
		}
	}
	return nil
}

// checkHealthCheckVersion gates the built-in readiness probe on the broker
// release that first serves /health-check/readiness. An older broker has no such
// endpoint, so the probe would fail forever and report a healthy broker as
// unhealthy -- under podman's auto-restart, a restart loop. An unidentifiable tag
// is refused for the same reason: this cannot be verified, and guessing "new
// enough" is the dangerous direction. Both errors name the explicit-cmd escape
// hatch, which is the supported way to probe an older or custom-tagged broker.
func (c *Config) checkHealthCheckVersion(p Platform) error {
	const field = ".container.healthCheck"
	ok, known := c.Image.AtLeast(HealthCheckMinMajor, HealthCheckMinMinor)
	switch {
	case !known:
		return fmt.Errorf("%s%s is enabled with no cmd, which uses the built-in readiness probe, "+
			"but the broker version cannot be read from image.tag %q; the probe needs %d.%d or later -- "+
			"use a version-numbered tag, or set %s%s.cmd to your own probe",
			platformKey(p), field, c.Image.Tag, HealthCheckMinMajor, HealthCheckMinMinor, platformKey(p), field)
	case !ok:
		return fmt.Errorf("%s%s is enabled with no cmd, which uses the built-in readiness probe, "+
			"but image.tag %q is older than %d.%d, where /health-check/readiness was introduced -- "+
			"upgrade the broker, or set %s%s.cmd to a probe that image supports",
			platformKey(p), field, c.Image.Tag, HealthCheckMinMajor, HealthCheckMinMinor, platformKey(p), field)
	}
	return nil
}

func platformKey(p Platform) string {
	if p == Podman {
		return "podman"
	}
	return "docker"
}

func requireAll(fields map[string]string) []string {
	var missing []string
	for name, val := range fields {
		if strings.TrimSpace(val) == "" {
			missing = append(missing, name)
		}
	}
	return missing
}

func missingErr(missing []string) error {
	sortStrings(missing)
	return fmt.Errorf("these fields must not be empty: %s", strings.Join(missing, ", "))
}

// sortStrings is a tiny insertion sort to keep the missing-fields message stable
// without pulling in the sort package for a handful of items.
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}
