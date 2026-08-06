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

	switch p {
	case K8s:
		return c.validateK8s()
	case Docker, Podman:
		return c.validateContainer(p)
	default:
		return fmt.Errorf("unknown platform %q", p)
	}
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
