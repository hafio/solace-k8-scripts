package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Load reads the YAML env file at path, applies platform defaults, and validates
// it. Defaults and validation are platform-scoped, mirroring the two separate
// bash bootstraps (root 000-env.sh for k8s, docker-podman/000-env.sh for containers).
func Load(path string, p Platform) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read env file %q: %w", path, err)
	}
	var c Config
	dec := yaml.NewDecoder(bytes.NewReader(raw))
	dec.KnownFields(true) // fail loud on typo'd keys
	if err := dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("parse env file %q: %w", path, err)
	}
	c.ApplyDefaults(p)
	if err := c.Validate(p); err != nil {
		return nil, err
	}
	return &c, nil
}

// ResolveEnvPath turns an --env name into env/<name>.yaml under baseDir.
func ResolveEnvPath(baseDir, name string) string {
	if name == "" {
		name = "default"
	}
	return filepath.Join(baseDir, "env", name+".yaml")
}

// ApplyDefaults fills unset optional fields, applying the platform's defaults.
// Only fields that are safe to default are touched; mandatory fields are left
// empty so Validate can flag them. Secrets are never defaulted to a literal.
func (c *Config) ApplyDefaults(p Platform) {
	if c.Redundancy == "" {
		c.Redundancy = "yes"
	}

	if p == K8s {
		c.applyK8sDefaults()
	}
	if p.IsContainer() {
		c.applyContainerDefaults(p)
	}
}

func (c *Config) applyK8sDefaults() {
	setDefault(&c.K8s.UpdateStrategy, "automatedRolling")
	setDefault(&c.Admin.UserSecret, "solace-admin-secret")
	setDefault(&c.K8s.DiagDir, "diag-configs")
	setDefault(&c.K8s.CLIScriptsFolder, "cli")
	setDefault(&c.K8s.Storage.MonNode, "5Gi")
	setDefault(&c.K8s.MsgNode.CPU, "2")
	setDefault(&c.K8s.MsgNode.Mem, "3410Mi")

	setDefault(&c.K8s.Operator.Image, "docker.io/solace/pubsubplus-eventbroker-operator:1.4.0")
	setDefault(&c.K8s.Operator.CPU, "500m")
	setDefault(&c.K8s.Operator.Mem, "512Mi")

	// Scaling (k8s defaults differ from container).
	setDefaultInt(&c.Scaling.MaxConnections, 100)
	setDefaultInt(&c.Scaling.MaxPool, 10000)
	setDefaultInt(&c.Scaling.MaxQueueMessages, 100)
	setDefaultInt(&c.Scaling.MaxBridges, 25)
	setDefaultInt(&c.Scaling.MaxSubscriptions, 50000)
	setDefaultInt(&c.Scaling.MaxGuaranteedMsgMB, 10)

	if len(c.K8s.Placement.AntiAffinityNS) > 0 {
		setDefaultInt(&c.K8s.Placement.AntiAffinityWeight, 100)
	}
	// If no anti-affinity namespace is given, default to the broker's own.
	if len(c.K8s.Placement.AntiAffinityNS) == 0 && c.K8s.Namespace != "" {
		c.K8s.Placement.AntiAffinityNS = []string{c.K8s.Namespace}
		setDefaultInt(&c.K8s.Placement.AntiAffinityWeight, 100)
	}

	if len(c.K8s.Ports) == 0 {
		c.K8s.Ports = defaultK8sPorts()
	}

	// TLS cert/key only default when a server secret is requested.
	if c.TLS.ServerSecret != "" {
		setDefault(&c.TLS.Cert, "certs/tls.crt")
		setDefault(&c.TLS.CertKey, "certs/tls.key")
	}
}

func (c *Config) applyContainerDefaults(p Platform) {
	setDefault(&c.Docker.Mode, "compose")

	setDefault(&c.Docker.Container.Name, "solace")
	setDefault(&c.Podman.Container.Name, "solace")
	applyContainerBlockDefaults(&c.Docker.Container)
	applyContainerBlockDefaults(&c.Podman.Container)

	setDefault(&c.Docker.Network.Mode, "host")
	setDefault(&c.Podman.Network.Mode, "host")

	setDefault(&c.Admin.User, "admin")

	// Container config/verify reuse these k8s.* fields as the broker-ops source
	// (diagnostics dir, CLI-scripts folder), so default them here too -- otherwise
	// `verify diagnostics` would MkdirAll("") and exec-cli would lose its folder.
	setDefault(&c.K8s.DiagDir, "diag-configs")
	setDefault(&c.K8s.CLIScriptsFolder, "cli")

	// Scaling (container defaults differ from k8s).
	setDefaultInt(&c.Scaling.MaxConnections, 1000)
	setDefaultInt(&c.Scaling.MaxQueueMessages, 100)
	setDefaultInt(&c.Scaling.MaxSpoolUsageMB, 100000)

	if p == Docker {
		setDefault(&c.Docker.Runtime, "docker")
	}
	if p == Podman {
		setDefault(&c.Podman.Runtime, "podman")
		// Derive the three rootless-dependent knobs in one place.
		if c.Podman.Rootless {
			if c.Podman.QuadletDir == "" {
				c.Podman.QuadletDir = filepath.ToSlash(filepath.Join(xdgConfigHome(), "containers", "systemd"))
			}
			c.Podman.SystemctlUser = "--user"
			c.Podman.WantedBy = "default.target"
		} else {
			setDefault(&c.Podman.QuadletDir, "/etc/containers/systemd")
			c.Podman.SystemctlUser = ""
			c.Podman.WantedBy = "multi-user.target"
		}
	}
}

func applyContainerBlockDefaults(b *Container) {
	setDefault(&b.RunUser, "0:0")
	setDefault(&b.TZ, "UTC")
	setDefault(&b.ShmSize, "1g")
	setDefault(&b.DataDir, "/opt/solace/data")
	setDefault(&b.Ulimits.NoFile, "2448:1048576")
	setDefault(&b.Ulimits.MemLock, "-1")
	setDefault(&b.Ulimits.Core, "-1")
}

// ContainerRuntime returns the runtime binary for the container platform p.
func (c *Config) ContainerRuntime(p Platform) string {
	switch p {
	case Docker:
		return c.Docker.Runtime
	case Podman:
		return c.Podman.Runtime
	default:
		return ""
	}
}

// ContainerBlock returns the shared container settings for the platform p.
func (c *Config) ContainerBlock(p Platform) Container {
	if p == Podman {
		return c.Podman.Container
	}
	return c.Docker.Container
}

// NetworkBlock returns the network settings for the platform p.
func (c *Config) NetworkBlock(p Platform) Network {
	if p == Podman {
		return c.Podman.Network
	}
	return c.Docker.Network
}

func xdgConfigHome() string {
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		home = "~"
	}
	return filepath.Join(home, ".config")
}

func defaultK8sPorts() []string {
	return []string{
		"tcp-semp=8080", "tls-semp=1943",
		"tcp-smf=55555", "tcp-smfcomp=55003", "tls-smf=55443", "tcp-smfroute=55556",
		"tcp-web=8008", "tls-web=1443",
		"tcp-rest=9000", "tls-rest=9443",
		"tcp-amqp=5672", "tls-amqp=5671",
		"tcp-mqtt=1883", "tls-mqtt=8883",
		"tcp-mqttweb=8000", "tls-mqttweb=8443",
	}
}

func setDefault(p *string, v string) {
	if *p == "" {
		*p = v
	}
}

func setDefaultInt(p *int, v int) {
	if *p == 0 {
		*p = v
	}
}
