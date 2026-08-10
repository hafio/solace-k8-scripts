package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

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
		return nil, parseError(path, raw, err)
	}
	c.ApplyDefaults(p)
	if err := c.Validate(p); err != nil {
		return nil, err
	}
	return &c, nil
}

// bashAssignRE matches a shell variable assignment at the start of a line -- the
// shape of the legacy bash env files, which are the one thing users are likely
// to hand to -e by mistake.
var bashAssignRE = regexp.MustCompile(`(?m)^\s*(declare\s+(-[Aa]\s+)?|export\s+)?[A-Z][A-Z0-9_]*=`)

// parseError turns a decode failure into an actionable one. A file that is not
// YAML at all is the interesting case: say so plainly, and point at `solace
// convert` when it looks like a legacy bash env file. A file that *is* valid
// YAML but carries an unknown key keeps the plain strict-decoding error, since
// the schema, not the format, is the problem.
func parseError(path string, raw []byte, err error) error {
	var probe map[string]any
	if yaml.Unmarshal(raw, &probe) == nil {
		return fmt.Errorf("parse env file %q: %w", path, err)
	}
	if bashAssignRE.Match(raw) || bytes.HasPrefix(raw, []byte("#!")) {
		return fmt.Errorf("parse env file %q: not valid YAML -- this looks like a legacy bash env file: %w\n"+
			"  convert it first:  solace convert %s -o <name>.yaml", path, err, path)
	}
	return fmt.Errorf("parse env file %q: not valid YAML: %w\n"+
		"  the env file must be YAML -- see env/sample.yaml for the schema.\n"+
		"  converting a legacy bash env file:  solace convert <bash-env-file> -o <name>.yaml", path, err)
}

// EnvFileDefault is the env file name used when -e/--env is omitted.
const EnvFileDefault = "env.yaml"

// ResolveEnvPath finds the env file named by -e/--env and returns its path. The
// name is taken literally -- no extension is ever inferred, so "dev" and
// "dev.yaml" name different files. A bare file name is searched in baseDir, then
// in baseDir/env (the layout the bash scripts hard-wired). A value carrying a
// directory component is used exactly as typed and never falls back to env/, so
// `-e env/dev.yaml` reads that one file and `-e ../shared/prod.yaml` leaves the
// project tree deliberately.
func ResolveEnvPath(baseDir, name string) (string, error) {
	if name == "" {
		name = EnvFileDefault
	}
	// The resolved path is echoed to the terminal, so reject control characters
	// at the boundary rather than printing them (§3).
	if strings.ContainsFunc(name, func(r rune) bool { return r < 0x20 || r == 0x7f }) {
		return "", fmt.Errorf("invalid env file name %q: control characters are not allowed", name)
	}
	var candidates []string
	if name != filepath.Base(name) {
		candidates = []string{name} // has a directory component -> verbatim, no env/ fallback
	} else {
		base := baseDir
		if base == "" {
			base = "."
		}
		candidates = []string{filepath.Join(base, name), filepath.Join(base, "env", name)}
	}
	for _, p := range candidates {
		if st, err := os.Stat(p); err == nil && st.Mode().IsRegular() {
			return p, nil
		}
	}
	quoted := make([]string, len(candidates))
	for i, p := range candidates {
		quoted[i] = fmt.Sprintf("%q", p)
	}
	// Actionable: name every place that was tried (§4).
	return "", fmt.Errorf("env file %q not found: looked for %s", name, strings.Join(quoted, " then "))
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

// applyContainerBlockDefaults fills the container runtime knobs. TZ is
// deliberately absent: the timezone is optional on every platform, so an unset
// value emits no TZ setting at all rather than silently pinning one.
func applyContainerBlockDefaults(b *Container) {
	setDefault(&b.RunUser, "0:0")
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
