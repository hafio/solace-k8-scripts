package config

import (
	"fmt"
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

	net := c.NetworkBlock(p)
	switch net.Mode {
	case "host":
	case "bridge":
		if len(net.Ports) == 0 {
			return fmt.Errorf("%s.network.mode=bridge requires a non-empty ports list (host:container entries)",
				platformKey(p))
		}
	default:
		return fmt.Errorf("%s.network.mode must be 'host' or 'bridge' (got: %q)", platformKey(p), net.Mode)
	}

	if p == Docker {
		switch c.Docker.Mode {
		case "compose", "run":
		default:
			return fmt.Errorf("docker.mode must be 'compose' or 'run' (got: %q)", c.Docker.Mode)
		}
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
