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

	// The platform CLI is user-supplied and reaches os/exec, so it goes through the
	// full execution guard (execguard.go) on every platform before anything can
	// run. This is the first of the two enforcement points; every executor re-runs
	// the same CheckCommand immediately before it builds argv, so a hostile env
	// file is inert even on a path that never reached Validate.
	if err := c.validateExecCommands(p); err != nil {
		return err
	}

	// Scaling applies to every platform -- k8s through the CR, containers through
	// the environment -- so it is checked once here (scaling.go).
	if err := c.validateScaling(); err != nil {
		return err
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

// validateProbeCommand checks the tokens of a command this tool does NOT execute:
// the container health-check probe, which is rendered into the compose/quadlet
// artifact and run by the container engine INSIDE the broker container. It never
// becomes argv on the operator's machine, so the execution guard's allowlist and
// subcommand rules would be meaningless here -- a probe is legitimately
// `sh -c 'curl ... || exit 1'`, and the engine, not this process, decides what it
// means. What still applies is the exec-boundary check the field always had: an
// empty argument, or a control character carried in from a converted bash file,
// can only ever fail obscurely (§4a).
//
// An empty Command is not an error: ApplyDefaults runs before Validate on every
// path and fills the platform default, so "empty" means "unset" exactly as it
// does for every setDefault field in this schema.
func validateProbeCommand(field string, cmd Command) error {
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
	if c.K8s.MsgNode.CPU != "" {
		// Removed rather than ignored: a stale cpu: in an env file is a sizing
		// decision the operator believes is in effect, so it has to be seen.
		return fmt.Errorf("k8s.msgNode.cpu was removed; broker CPU is fixed by the scaling tier and "+
			"derived from scaling.maxConnections (one of %s) -- drop the key. "+
			"k8s.msgNode.mem is unaffected: it still overrides the tier's default memory", scalingTierList)
	}
	if u := c.Admin.User; u != "" && u != "admin" {
		// Rejected rather than ignored, for the same reason as msgNode.cpu above: this is
		// the login an operator believes is in effect. The operator reads the fixed
		// username_admin_password key out of the credentials Secret (k8s/secrets.go,
		// verified against a live cluster), and creates the admin user itself, so nothing
		// on this platform can honour another name. An unset value is skipped: ApplyDefaults
		// fills "admin", so empty means "will be defaulted" as it does for every other
		// setDefault field.
		return fmt.Errorf("admin.user %q is not supported on Kubernetes: the operator reads the fixed "+
			"username_admin_password key out of k8s.adminSecret, so the broker admin user is always "+
			"'admin' -- drop the key (it applies to docker and podman, where the username is yours to choose)", u)
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
	if err := c.validateAdditionalUsers(K8s); err != nil {
		return err
	}
	return validatePlacementAffinity(pl)
}

// accessLevels are the broker's global access levels, in increasing order of
// privilege. The value reaches the broker as a username_<user>_globalaccesslevel
// setting (containers) or a `global-access-level` CLI attribute (k8s), so an invalid
// one is a user the broker declines to create -- checked here where the field can be
// named.
var accessLevels = map[string]bool{
	"none": true, "read-only": true, "mesh-manager": true, "read-write": true, "admin": true,
}

// accessLevelList is the enum for error messages, in the same order.
const accessLevelList = "'none', 'read-only', 'mesh-manager', 'read-write' or 'admin'"

// cliForbiddenPassword are the characters the broker's own CLI rejects in a
// `create username ... password ...` value. They are refused here rather than at
// config time on the cluster, so an env file that cannot be applied fails at load.
// Only k8s delivers a password through the CLI: on containers it is written to a
// mounted file, which has no such restriction, so this is checked per platform.
const cliForbiddenPassword = ":()\";'<>,`\\*&|"

// foldToEnvVar upper-cases a username and folds every character an environment
// variable name cannot carry to '_' -- the same mapping render's
// ContainerSecret.EnvVar applies to the whole secret name when docker sources a
// compose secret from the host environment. It exists here only to detect the
// collision that mapping can create; the rendering itself stays in render, and
// config must not import it (render depends on config). A small package-local copy
// in the spirit of identRE, which likewise mirrors a regexp two other packages own.
func foldToEnvVar(name string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(name) {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			continue
		}
		b.WriteByte('_')
	}
	return b.String()
}

// validateAdditionalUsers checks the extra CLI users, which every platform carries
// but delivers differently: containers create them at boot from a mounted secret
// file plus an access-level setting, while k8s creates them post-deployment through
// the broker CLI (`config additional-users`). Access level is required rather than
// defaulted -- silently choosing someone's permissions is not a default worth
// having. p selects the platform-specific rules; only the k8s path constrains the
// password, since only it puts the value on a CLI line.
func (c *Config) validateAdditionalUsers(p Platform) error {
	seen := make(map[string]bool, len(c.Admin.AdditionalUsers))
	folded := make(map[string]string, len(c.Admin.AdditionalUsers))
	for i, u := range c.Admin.AdditionalUsers {
		field := fmt.Sprintf("admin.additionalUsers[%d]", i)
		if strings.TrimSpace(u.Username) == "" {
			return fmt.Errorf("%s.username must be set", field)
		}
		if !identRE.MatchString(u.Username) {
			return fmt.Errorf("%s.username %q is invalid: only letters, digits, '.', '_' and '-' are allowed "+
				"(it becomes the secret name username_%s_password)", field, u.Username, u.Username)
		}
		if u.Username == "admin" || u.Username == "monitor" || u.Username == c.Admin.User {
			return fmt.Errorf("%s.username %q is a built-in user: admin has admin.pass and monitor has "+
				"admin.monitorPass -- additionalUsers is for users beyond those", field, u.Username)
		}
		if seen[u.Username] {
			return fmt.Errorf("%s.username %q is listed twice; each user appears once", field, u.Username)
		}
		seen[u.Username] = true
		// Two users differing only in separator style are distinct to the broker but
		// fold to ONE docker host variable name (render's ContainerSecret.EnvVar maps
		// every non-alphanumeric to '_'), which would feed one user's password to
		// both. Caught here, where both offending fields can be named, rather than
		// silently at deploy time.
		key := foldToEnvVar(u.Username)
		if other := folded[key]; other != "" {
			return fmt.Errorf("%s.username %q collides with %q: they differ only in '.', '_' or '-', "+
				"which become the same host environment variable (...%s...) for docker's compose secrets -- "+
				"rename one", field, u.Username, other, key)
		}
		folded[key] = u.Username
		if !accessLevels[u.AccessLevel] {
			return fmt.Errorf("%s.accessLevel must be %s (got: %q)", field, accessLevelList, u.AccessLevel)
		}
		if u.Password == "" {
			return fmt.Errorf("%s.password must not be empty; set it, or point %s.passwordEnv at an "+
				"environment variable holding it", field, field)
		}
		// k8s creates the user with `create username "<u>" password "<p>"`, and the
		// broker CLI rejects these characters in the value. The message names the
		// offending character but never the password (§3).
		if p == K8s {
			if i := strings.IndexAny(u.Password, cliForbiddenPassword); i >= 0 {
				return fmt.Errorf("%s.password contains %q, which the broker CLI rejects in a password; "+
					"on Kubernetes the user is created over the CLI, so none of %s may appear "+
					"(the value itself is not shown)", field, string(u.Password[i]), cliForbiddenPassword)
			}
		}
	}
	return nil
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
	if m := cb.Mem; m != "" && !containerMemRE.MatchString(m) {
		// The likely mistake is copying k8s.msgNode.mem's Kubernetes quantity
		// across; the engines reject "Mi"/"Gi", and catching it here beats a
		// compose parse error at deploy time.
		return fmt.Errorf("%s.container.mem %q is invalid: docker and podman take an integer followed by "+
			"b, k, m or g (e.g. 6898m), not the Mi/Gi suffix k8s.msgNode.mem uses", platformKey(p), m)
	}
	if err := c.validateAdditionalUsers(p); err != nil {
		return err
	}

	if hc := cb.HealthCheck; hc.Enabled {
		if len(hc.Cmd) > 0 {
			// An explicit probe is the operator's own; it only gets the exec-boundary
			// check, not the version gate -- and not the execution guard, since it
			// runs inside the container rather than here (validateProbeCommand).
			if err := validateProbeCommand(platformKey(p)+".container.healthCheck.cmd", Command(hc.Cmd)); err != nil {
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
