package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestPlatformIsContainer(t *testing.T) {
	tests := []struct {
		p    Platform
		want bool
	}{
		{K8s, false},
		{Docker, true},
		{Podman, true},
	}
	for _, tc := range tests {
		if got := tc.p.IsContainer(); got != tc.want {
			t.Errorf("Platform(%q).IsContainer() = %v, want %v", tc.p, got, tc.want)
		}
	}
}

func TestRedundancyEnabled(t *testing.T) {
	tests := []struct {
		redundancy string
		want       bool
	}{
		{"yes", true},
		{"no", false},
		{"", false},
		{"maybe", false},
	}
	for _, tc := range tests {
		c := &Config{Redundancy: tc.redundancy}
		if got := c.RedundancyEnabled(); got != tc.want {
			t.Errorf("Redundancy=%q RedundancyEnabled() = %v, want %v", tc.redundancy, got, tc.want)
		}
	}
}

func TestImageRef(t *testing.T) {
	tests := []struct {
		name string
		img  Image
		want string
	}{
		{"no registry", Image{Repo: "solace/broker", Tag: "latest"}, "solace/broker:latest"},
		{"with registry", Image{Repo: "solace/broker", Tag: "10.0", Registry: "reg.example.com"}, "reg.example.com/solace/broker:10.0"},
	}
	for _, tc := range tests {
		if got := tc.img.Ref(); got != tc.want {
			t.Errorf("%s: Ref() = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestParseRole(t *testing.T) {
	tests := []struct {
		in      string
		want    Role
		wantErr bool
	}{
		{"p", Primary, false},
		{"primary", Primary, false},
		{"b", Backup, false},
		{"backup", Backup, false},
		{"m", Monitor, false},
		{"monitor", Monitor, false},
		{"", Primary, false},
		{"bogus", "", true},
	}
	for _, tc := range tests {
		got, err := ParseRole(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("ParseRole(%q) expected error, got nil", tc.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseRole(%q) unexpected error: %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ParseRole(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestRoleLetter(t *testing.T) {
	tests := []struct {
		r    Role
		want string
	}{
		{Primary, "p"},
		{Backup, "b"},
		{Monitor, "m"},
	}
	for _, tc := range tests {
		if got := tc.r.Letter(); got != tc.want {
			t.Errorf("Role(%q).Letter() = %q, want %q", tc.r, got, tc.want)
		}
	}
}

func haNodesConfig(redundancy string) *Config {
	return &Config{
		Redundancy: redundancy,
		Nodes: Nodes{
			Primary: Node{Name: "solace-p", IP: "10.0.0.1"},
			Backup:  Node{Name: "solace-b", IP: "10.0.0.2"},
			Monitor: Node{Name: "solace-m", IP: "10.0.0.3"},
		},
	}
}

func TestResolveNodeStandalone(t *testing.T) {
	c := haNodesConfig("no")
	// Role is ignored in standalone; always the primary as a message_routing node.
	for _, r := range []Role{Primary, Backup, Monitor} {
		got := c.ResolveNode(r)
		want := NodeIdentity{Hostname: "solace-p", NodeType: "message_routing"}
		if got != want {
			t.Errorf("standalone ResolveNode(%q) = %+v, want %+v", r, got, want)
		}
	}
}

func TestResolveNodeHA(t *testing.T) {
	c := haNodesConfig("yes")
	tests := []struct {
		role Role
		want NodeIdentity
	}{
		{Primary, NodeIdentity{Hostname: "solace-p", NodeType: "message_routing", ActiveStandby: "primary"}},
		{Backup, NodeIdentity{Hostname: "solace-b", NodeType: "message_routing", ActiveStandby: "backup"}},
		{Monitor, NodeIdentity{Hostname: "solace-m", NodeType: "monitoring"}},
	}
	for _, tc := range tests {
		if got := c.ResolveNode(tc.role); got != tc.want {
			t.Errorf("HA ResolveNode(%q) = %+v, want %+v", tc.role, got, tc.want)
		}
	}
}

func TestContainerRuntime(t *testing.T) {
	c := &Config{}
	c.Docker.Runtime = Command{"docker"}
	c.Podman.Runtime = Command{"sudo", "podman"}
	tests := []struct {
		p    Platform
		want string
	}{
		{Docker, "docker"},
		{Podman, "sudo podman"}, // leading args survive the lookup
		{K8s, ""},
	}
	for _, tc := range tests {
		if got := c.ContainerRuntime(tc.p); got.String() != tc.want {
			t.Errorf("ContainerRuntime(%q) = %q, want %q", tc.p, got, tc.want)
		}
	}
}

func TestContainerBlock(t *testing.T) {
	c := &Config{}
	c.Docker.Container.Name = "dkr"
	c.Podman.Container.Name = "pod"
	if got := c.ContainerBlock(Docker).Name; got != "dkr" {
		t.Errorf("ContainerBlock(Docker).Name = %q, want %q", got, "dkr")
	}
	if got := c.ContainerBlock(Podman).Name; got != "pod" {
		t.Errorf("ContainerBlock(Podman).Name = %q, want %q", got, "pod")
	}
	// Non-podman falls through to the docker block.
	if got := c.ContainerBlock(K8s).Name; got != "dkr" {
		t.Errorf("ContainerBlock(K8s).Name = %q, want %q", got, "dkr")
	}
}

func TestNetworkBlock(t *testing.T) {
	c := &Config{}
	c.Docker.Network.Mode = "host"
	c.Podman.Network.Mode = "bridge"
	if got := c.NetworkBlock(Docker).Mode; got != "host" {
		t.Errorf("NetworkBlock(Docker).Mode = %q, want %q", got, "host")
	}
	if got := c.NetworkBlock(Podman).Mode; got != "bridge" {
		t.Errorf("NetworkBlock(Podman).Mode = %q, want %q", got, "bridge")
	}
}

func TestApplyDefaultsK8s(t *testing.T) {
	c := &Config{}
	c.K8s.Namespace = "sol-ns"
	c.ApplyDefaults(K8s)

	// Unset redundancy means standalone: HA provisions three brokers, so it is the
	// choice that must be explicit.
	if c.Redundancy != "no" {
		t.Errorf("Redundancy default = %q, want %q", c.Redundancy, "no")
	}
	if c.K8s.UpdateStrategy != "automatedRolling" {
		t.Errorf("UpdateStrategy = %q, want automatedRolling", c.K8s.UpdateStrategy)
	}
	if c.Admin.UserSecret != "solace-admin-secret" {
		t.Errorf("Admin.UserSecret = %q", c.Admin.UserSecret)
	}
	if c.K8s.DiagDir != "diag-configs" {
		t.Errorf("DiagDir = %q", c.K8s.DiagDir)
	}
	if c.K8s.CLIScriptsFolder != "cli" {
		t.Errorf("CLIScriptsFolder = %q", c.K8s.CLIScriptsFolder)
	}
	if c.K8s.Storage.MonNode != "5Gi" {
		t.Errorf("Storage.MonNode = %q", c.K8s.Storage.MonNode)
	}
	if c.K8s.MsgNode.CPU != "2" || c.K8s.MsgNode.Mem != "3410Mi" {
		t.Errorf("MsgNode = %+v", c.K8s.MsgNode)
	}
	if c.K8s.Operator.Image != "docker.io/solace/pubsubplus-eventbroker-operator:1.4.0" {
		t.Errorf("Operator.Image = %q", c.K8s.Operator.Image)
	}
	if c.K8s.Operator.CPU != "500m" || c.K8s.Operator.Mem != "512Mi" {
		t.Errorf("Operator resources = %+v", c.K8s.Operator)
	}
	if c.Scaling.MaxConnections != 100 || c.Scaling.MaxPool != 10000 ||
		c.Scaling.MaxQueueMessages != 100 || c.Scaling.MaxBridges != 25 ||
		c.Scaling.MaxSubscriptions != 50000 || c.Scaling.MaxGuaranteedMsgMB != 10 {
		t.Errorf("Scaling defaults = %+v", c.Scaling)
	}
	if len(c.K8s.Ports) == 0 {
		t.Errorf("K8s.Ports should be non-empty by default")
	}
	// Anti-affinity defaults to the broker namespace with weight 100.
	if len(c.K8s.Placement.AntiAffinityNS) != 1 || c.K8s.Placement.AntiAffinityNS[0] != "sol-ns" {
		t.Errorf("AntiAffinityNS = %v, want [sol-ns]", c.K8s.Placement.AntiAffinityNS)
	}
	if c.K8s.Placement.AntiAffinityWeight != 100 {
		t.Errorf("AntiAffinityWeight = %d, want 100", c.K8s.Placement.AntiAffinityWeight)
	}
}

func TestApplyDefaultsK8sTLS(t *testing.T) {
	// Without a server secret, cert/key are left empty.
	c := &Config{}
	c.ApplyDefaults(K8s)
	if c.TLS.Cert != "" || c.TLS.CertKey != "" {
		t.Errorf("TLS defaulted without ServerSecret: cert=%q key=%q", c.TLS.Cert, c.TLS.CertKey)
	}

	// With a server secret, cert/key default.
	c2 := &Config{}
	c2.TLS.ServerSecret = "solace-tls"
	c2.ApplyDefaults(K8s)
	if c2.TLS.Cert != "certs/tls.crt" || c2.TLS.CertKey != "certs/tls.key" {
		t.Errorf("TLS defaults with ServerSecret: cert=%q key=%q", c2.TLS.Cert, c2.TLS.CertKey)
	}
}

func assertContainerBlockDefaults(t *testing.T, b Container) {
	t.Helper()
	if b.RunUser != "0:0" {
		t.Errorf("RunUser = %q, want 0:0", b.RunUser)
	}
	if b.ShmSize != "1g" {
		t.Errorf("ShmSize = %q, want 1g", b.ShmSize)
	}
	if b.DataDir != "/opt/solace/data" {
		t.Errorf("DataDir = %q, want /opt/solace/data", b.DataDir)
	}
	if b.Ulimits.NoFile != "2448:1048576" {
		t.Errorf("Ulimits.NoFile = %q", b.Ulimits.NoFile)
	}
	if b.Ulimits.MemLock != "-1" {
		t.Errorf("Ulimits.MemLock = %q", b.Ulimits.MemLock)
	}
	if b.Ulimits.Core != "-1" {
		t.Errorf("Ulimits.Core = %q", b.Ulimits.Core)
	}
	// The health check stays disabled by default, but its timings are filled so the
	// block only needs `enabled` and `cmd` to be useful.
	if b.HealthCheck.Enabled {
		t.Error("HealthCheck must be opt-in")
	}
	if b.HealthCheck.Interval != "5s" || b.HealthCheck.Timeout != "5s" ||
		b.HealthCheck.StartPeriod != "60s" || b.HealthCheck.Retries != 3 {
		t.Errorf("HealthCheck timing defaults = %+v", b.HealthCheck)
	}
}

func assertContainerScaling(t *testing.T, c *Config) {
	t.Helper()
	if c.Scaling.MaxConnections != 1000 || c.Scaling.MaxQueueMessages != 100 || c.Scaling.MaxSpoolUsageMB != 100000 {
		t.Errorf("container Scaling defaults = %+v", c.Scaling)
	}
}

func TestApplyDefaultsDocker(t *testing.T) {
	c := &Config{}
	c.ApplyDefaults(Docker)

	if c.Docker.Runtime.String() != "docker" {
		t.Errorf("Docker.Runtime = %q, want docker", c.Docker.Runtime)
	}
	if c.Docker.Mode != "compose" {
		t.Errorf("Docker.Mode = %q, want compose", c.Docker.Mode)
	}
	// The compose default is derived from the runtime, not hardcoded, so a runtime
	// override carries into it.
	if c.Docker.Compose.String() != "docker compose" {
		t.Errorf("Docker.Compose = %q, want 'docker compose'", c.Docker.Compose)
	}
	if c.Docker.Network.Mode != "host" {
		t.Errorf("Docker.Network.Mode = %q, want host", c.Docker.Network.Mode)
	}
	if c.Admin.User != "admin" {
		t.Errorf("Admin.User = %q, want admin", c.Admin.User)
	}
	if c.Docker.Container.Name != "solace" {
		t.Errorf("Docker.Container.Name = %q, want solace", c.Docker.Container.Name)
	}
	// Container config/verify reuse these k8s.* fields, so they default here too.
	if c.K8s.DiagDir != "diag-configs" {
		t.Errorf("container DiagDir = %q, want diag-configs", c.K8s.DiagDir)
	}
	if c.K8s.CLIScriptsFolder != "cli" {
		t.Errorf("container CLIScriptsFolder = %q, want cli", c.K8s.CLIScriptsFolder)
	}
	assertContainerBlockDefaults(t, c.Docker.Container)
	assertContainerScaling(t, c)
}

func TestApplyDefaultsPodmanRootful(t *testing.T) {
	c := &Config{}
	c.Podman.Rootless = false
	c.ApplyDefaults(Podman)

	if c.Podman.Runtime.String() != "podman" {
		t.Errorf("Podman.Runtime = %q, want podman", c.Podman.Runtime)
	}
	if c.Podman.Network.Mode != "host" {
		t.Errorf("Podman.Network.Mode = %q, want host", c.Podman.Network.Mode)
	}
	if c.Podman.Container.Name != "solace" {
		t.Errorf("Podman.Container.Name = %q, want solace", c.Podman.Container.Name)
	}
	assertContainerBlockDefaults(t, c.Podman.Container)
	assertContainerScaling(t, c)

	if c.Podman.QuadletDir != "/etc/containers/systemd" {
		t.Errorf("rootful QuadletDir = %q, want /etc/containers/systemd", c.Podman.QuadletDir)
	}
	if c.Podman.SystemctlUser != "" {
		t.Errorf("rootful SystemctlUser = %q, want empty", c.Podman.SystemctlUser)
	}
	if c.Podman.WantedBy != "multi-user.target" {
		t.Errorf("rootful WantedBy = %q, want multi-user.target", c.Podman.WantedBy)
	}
}

func TestApplyDefaultsPodmanRootlessXDG(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	c := &Config{}
	c.Podman.Rootless = true
	c.ApplyDefaults(Podman)

	want := filepath.ToSlash(filepath.Join(tmp, "containers", "systemd"))
	if c.Podman.QuadletDir != want {
		t.Errorf("rootless QuadletDir = %q, want %q", c.Podman.QuadletDir, want)
	}
	if c.Podman.SystemctlUser != "--user" {
		t.Errorf("rootless SystemctlUser = %q, want --user", c.Podman.SystemctlUser)
	}
	if c.Podman.WantedBy != "default.target" {
		t.Errorf("rootless WantedBy = %q, want default.target", c.Podman.WantedBy)
	}
}

func TestApplyDefaultsPodmanRootlessHomeDir(t *testing.T) {
	// Empty XDG_CONFIG_HOME exercises the UserHomeDir branch.
	t.Setenv("XDG_CONFIG_HOME", "")

	c := &Config{}
	c.Podman.Rootless = true
	c.ApplyDefaults(Podman)

	if c.Podman.QuadletDir == "" {
		t.Fatal("rootless QuadletDir should be derived, got empty")
	}
	if !strings.HasSuffix(c.Podman.QuadletDir, "containers/systemd") {
		t.Errorf("rootless QuadletDir = %q, want suffix containers/systemd", c.Podman.QuadletDir)
	}
	if c.Podman.SystemctlUser != "--user" {
		t.Errorf("rootless SystemctlUser = %q, want --user", c.Podman.SystemctlUser)
	}
	if c.Podman.WantedBy != "default.target" {
		t.Errorf("rootless WantedBy = %q, want default.target", c.Podman.WantedBy)
	}
}

func validK8sConfig() *Config {
	c := &Config{}
	c.Image.Repo = "solace/broker"
	c.Image.Tag = "latest"
	c.Admin.Pass = "s3cret"
	c.K8s.Name = "mybroker"
	c.K8s.Namespace = "sol-ns"
	c.K8s.Storage.MsgNode = "30Gi"
	c.ApplyDefaults(K8s)
	return c
}

func TestValidateK8sValid(t *testing.T) {
	if err := validK8sConfig().Validate(K8s); err != nil {
		t.Errorf("valid k8s config Validate returned error: %v", err)
	}
}

func TestValidateK8sMissingMandatory(t *testing.T) {
	c := &Config{Redundancy: "yes"} // no defaults applied; all mandatory empty
	err := c.Validate(K8s)
	if err == nil {
		t.Fatal("expected error for missing mandatory k8s fields")
	}
	want := "these fields must not be empty: admin.pass, image.repo, image.tag, k8s.name, k8s.namespace, k8s.storage.msgNode"
	if err.Error() != want {
		t.Errorf("missing-fields message =\n  %q\nwant\n  %q", err.Error(), want)
	}
}

func TestValidateK8sBadUpdateStrategy(t *testing.T) {
	c := validK8sConfig()
	c.K8s.UpdateStrategy = "rollThemAll"
	err := c.Validate(K8s)
	if err == nil || !strings.Contains(err.Error(), "k8s.updateStrategy must be") {
		t.Errorf("expected updateStrategy enum error, got: %v", err)
	}
}

func validContainerConfig(p Platform, redundancy string) *Config {
	c := haNodesConfig(redundancy)
	c.Image.Repo = "solace/broker"
	c.Image.Tag = "latest"
	c.Admin.Pass = "s3cret"
	c.ApplyDefaults(p)
	return c
}

func TestValidateContainerHA(t *testing.T) {
	for _, p := range []Platform{Docker, Podman} {
		c := validContainerConfig(p, "yes")
		if err := c.Validate(p); err != nil {
			t.Errorf("valid HA %q config Validate returned error: %v", p, err)
		}
	}
}

func TestValidateContainerStandalone(t *testing.T) {
	// Standalone (redundancy no): only nodes.primary.name mandatory among nodes.
	c := &Config{Redundancy: "no"}
	c.Image.Repo = "solace/broker"
	c.Image.Tag = "latest"
	c.Admin.Pass = "s3cret"
	c.Nodes.Primary.Name = "solace-only"
	c.ApplyDefaults(Docker)
	if err := c.Validate(Docker); err != nil {
		t.Errorf("valid standalone config Validate returned error: %v", err)
	}
}

func TestValidateContainerMissingMandatory(t *testing.T) {
	c := &Config{Redundancy: "yes"}
	c.ApplyDefaults(Docker) // DataDir defaults, but image/admin/nodes stay empty
	err := c.Validate(Docker)
	if err == nil {
		t.Fatal("expected error for missing mandatory container fields")
	}
	msg := err.Error()
	if !strings.HasPrefix(msg, "these fields must not be empty:") {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"image.repo", "image.tag", "admin.pass", "nodes.primary.name",
		"nodes.primary.ip", "nodes.backup.name", "nodes.backup.ip", "nodes.monitor.name", "nodes.monitor.ip"} {
		if !strings.Contains(msg, want) {
			t.Errorf("missing-fields message %q does not name %q", msg, want)
		}
	}
}

// TestApplyBridgePortDefaults covers the bridge-mode port list: k8s has always
// defaulted its ports, while bridge mode published nothing unless every port was
// listed by hand. Host mode is untouched -- there is nothing to publish there.
func TestApplyBridgePortDefaults(t *testing.T) {
	for _, p := range []Platform{Docker, Podman} {
		c := &Config{}
		if p == Docker {
			c.Docker.Network.Mode = "bridge"
		} else {
			c.Podman.Network.Mode = "bridge"
		}
		c.ApplyDefaults(p)
		got := c.NetworkBlock(p).Ports
		if len(got) != len(defaultK8sPorts()) {
			t.Errorf("%s bridge ports = %d entries, want the k8s port count %d", p, len(got), len(defaultK8sPorts()))
		}
		if len(got) > 0 && got[0] != "2222:2222" {
			t.Errorf("%s bridge ports[0] = %q, want host:container pairs derived from the k8s set", p, got[0])
		}
	}

	// Host mode (the default) stays empty, and an explicit list is never replaced.
	c := &Config{}
	c.ApplyDefaults(Docker)
	if len(c.Docker.Network.Ports) != 0 {
		t.Errorf("host mode should publish nothing, got %v", c.Docker.Network.Ports)
	}
	c2 := &Config{}
	c2.Docker.Network.Mode = "bridge"
	c2.Docker.Network.Ports = []string{"8080:8080"}
	c2.ApplyDefaults(Docker)
	if len(c2.Docker.Network.Ports) != 1 {
		t.Errorf("an explicit port list must be left alone, got %v", c2.Docker.Network.Ports)
	}
}

// TestImageTagVersion covers the tag parsing the health-check gate rests on. The
// unknown case is the important one: a tag with no version must report "unknown",
// never a guess in either direction.
func TestImageTagVersion(t *testing.T) {
	cases := []struct {
		tag          string
		major, minor int
		known        bool
	}{
		{"10.26.1.5", 10, 26, true},
		{"10.26", 10, 26, true},
		{"11.0.0.1", 11, 0, true},
		{"10.26.0-rc1", 10, 26, true},
		{"9.13.1.4", 9, 13, true},
		{"latest", 0, 0, false},
		{"", 0, 0, false},
		{"10", 0, 0, false}, // no minor to compare
		{"stable.build", 0, 0, false},
	}
	for _, tc := range cases {
		major, minor, known := Image{Tag: tc.tag}.TagVersion()
		if known != tc.known || (known && (major != tc.major || minor != tc.minor)) {
			t.Errorf("TagVersion(%q) = (%d, %d, %v), want (%d, %d, %v)",
				tc.tag, major, minor, known, tc.major, tc.minor, tc.known)
		}
	}

	// AtLeast compares major before minor, so a newer major with a lower minor wins.
	for _, tc := range []struct {
		tag       string
		ok, known bool
	}{
		{"10.26.0.1", true, true},
		{"10.27.0.1", true, true},
		{"11.0.0.1", true, true},
		{"10.25.9.9", false, true},
		{"9.99.0.0", false, true},
		{"latest", false, false},
	} {
		ok, known := Image{Tag: tc.tag}.AtLeast(HealthCheckMinMajor, HealthCheckMinMinor)
		if ok != tc.ok || known != tc.known {
			t.Errorf("AtLeast(%q) = (%v, %v), want (%v, %v)", tc.tag, ok, known, tc.ok, tc.known)
		}
	}
}

// TestValidateHealthCheck covers the opt-in probe. With no cmd it uses the built-in
// readiness endpoint, which only exists from 10.26 -- so an older or
// unidentifiable tag is refused rather than deploying a probe that fails forever
// (and, under podman's auto-restart, loops). An explicit cmd is the escape hatch
// and skips the gate.
func TestValidateHealthCheck(t *testing.T) {
	enabled := func(p Platform, tag string) *Config {
		c := validContainerConfig(p, "yes")
		c.Image.Tag = tag
		block := &c.Docker.Container
		if p == Podman {
			block = &c.Podman.Container
		}
		block.HealthCheck.Enabled = true
		return c
	}

	t.Run("built-in probe needs 10.26 or later", func(t *testing.T) {
		c := enabled(Docker, "10.26.0.5")
		if err := c.Validate(Docker); err != nil {
			t.Errorf("10.26 must be accepted: %v", err)
		}
		c = enabled(Podman, "11.1.0.0")
		if err := c.Validate(Podman); err != nil {
			t.Errorf("a later major must be accepted: %v", err)
		}
	})

	t.Run("an older broker is refused", func(t *testing.T) {
		c := enabled(Docker, "10.25.1.4")
		err := c.Validate(Docker)
		if err == nil || !strings.Contains(err.Error(), "older than 10.26") {
			t.Fatalf("expected the too-old error, got: %v", err)
		}
		if !strings.Contains(err.Error(), "healthCheck.cmd") {
			t.Errorf("the error should offer the explicit-cmd escape hatch, got: %v", err)
		}
	})

	t.Run("an unidentifiable tag is refused", func(t *testing.T) {
		c := enabled(Docker, "latest")
		err := c.Validate(Docker)
		if err == nil || !strings.Contains(err.Error(), "cannot be read from image.tag") {
			t.Fatalf("expected the unknown-version error, got: %v", err)
		}
	})

	t.Run("an explicit cmd skips the version gate", func(t *testing.T) {
		c := enabled(Docker, "9.13.1.4") // far older than the built-in probe needs
		c.Docker.Container.HealthCheck.Cmd = []string{"curl", "-fs", "http://localhost:8080/SEMP/v2/monitor"}
		if err := c.Validate(Docker); err != nil {
			t.Errorf("an operator-supplied probe must not be version-gated: %v", err)
		}
		// It still reaches exec, so it keeps the boundary check.
		c.Docker.Container.HealthCheck.Cmd = []string{"curl", "-fs\n"}
		if err := c.Validate(Docker); err == nil || !strings.Contains(err.Error(), "healthCheck.cmd") {
			t.Errorf("expected a control-character error for the probe argv, got: %v", err)
		}
	})

	t.Run("disabled is the default and stays legal on any tag", func(t *testing.T) {
		c := validContainerConfig(Podman, "no") // tag "latest", health check off
		if err := c.Validate(Podman); err != nil {
			t.Errorf("a disabled health check must validate: %v", err)
		}
	})
}

func TestValidateContainerBridge(t *testing.T) {
	// Bridge without ports -> error.
	c := validContainerConfig(Docker, "yes")
	c.Docker.Network.Mode = "bridge"
	c.Docker.Network.Ports = nil
	if err := c.Validate(Docker); err == nil || !strings.Contains(err.Error(), "network.mode=bridge requires") {
		t.Errorf("expected bridge-without-ports error, got: %v", err)
	}

	// Bridge with ports -> ok.
	c2 := validContainerConfig(Docker, "yes")
	c2.Docker.Network.Mode = "bridge"
	c2.Docker.Network.Ports = []string{"55555:55555"}
	if err := c2.Validate(Docker); err != nil {
		t.Errorf("bridge with ports Validate returned error: %v", err)
	}
}

// TestValidateContainerIdentifiers covers the format checks on the values that
// reach the compose/quadlet artifact in structural positions: a container or node
// name carrying a colon, '=' or newline would produce a broken artifact instead of
// an error.
func TestValidateContainerIdentifiers(t *testing.T) {
	bad := []string{"sol ace", "sol:ace", "sol=ace", "sol\nace", "sol/ace"}
	for _, name := range bad {
		t.Run("container.name "+name, func(t *testing.T) {
			c := validContainerConfig(Docker, "yes")
			c.Docker.Container.Name = name
			if err := c.Validate(Docker); err == nil || !strings.Contains(err.Error(), "docker.container.name") {
				t.Errorf("expected a container.name format error for %q, got: %v", name, err)
			}
		})
		t.Run("nodes.backup.name "+name, func(t *testing.T) {
			c := validContainerConfig(Podman, "yes")
			c.Nodes.Backup.Name = name
			if err := c.Validate(Podman); err == nil || !strings.Contains(err.Error(), "nodes.backup.name") {
				t.Errorf("expected a nodes.backup.name format error for %q, got: %v", name, err)
			}
		})
	}
	// Standalone leaves the backup/monitor rows empty, which must stay legal: the
	// format check skips empty values, since emptiness is requireAll's job.
	c := validContainerConfig(Docker, "no")
	c.Nodes.Backup.Name, c.Nodes.Monitor.Name = "", ""
	if err := c.Validate(Docker); err != nil {
		t.Errorf("standalone with empty backup/monitor names must validate: %v", err)
	}
}

// TestValidateContainerRunUser pins the uid[:gid] form: the default "0:0" carries a
// colon, so the identifier check cannot be reused verbatim here.
func TestValidateContainerRunUser(t *testing.T) {
	for _, ok := range []string{"0:0", "1000", "1000:1000", "solace:solace"} {
		c := validContainerConfig(Docker, "yes")
		c.Docker.Container.RunUser = ok
		if err := c.Validate(Docker); err != nil {
			t.Errorf("runUser %q must be accepted: %v", ok, err)
		}
	}
	for _, bad := range []string{"1000 1000", "1000:", "root:root:root", "root\n"} {
		c := validContainerConfig(Docker, "yes")
		c.Docker.Container.RunUser = bad
		if err := c.Validate(Docker); err == nil || !strings.Contains(err.Error(), "runUser") {
			t.Errorf("expected a runUser format error for %q, got: %v", bad, err)
		}
	}
}

// TestValidateK8sKeyValueEntries covers the "key: value" fragments the CR renderer
// emits as mapping entries. The renderer quotes both halves, so the only
// unrecoverable shape is an entry with no key -- which used to be pasted into the
// manifest verbatim and corrupt it.
func TestValidateK8sKeyValueEntries(t *testing.T) {
	cases := []struct {
		name  string
		apply func(*Config)
		field string
	}{
		{"lb annotation without a colon", func(c *Config) {
			c.K8s.LoadBalancer.Annotations = []string{"not-a-pair"}
		}, "k8s.loadBalancer.annotations[0]"},
		{"label with an empty key", func(c *Config) {
			c.K8s.Placement.LabelsPrimary = []string{" : solace"}
		}, "k8s.placement.labelsPrimary[0]"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := validK8sConfig()
			tc.apply(c)
			err := c.Validate(K8s)
			if err == nil || !strings.Contains(err.Error(), tc.field) {
				t.Fatalf("expected an error naming %s, got: %v", tc.field, err)
			}
		})
	}
	// A value carrying a colon is fine: the renderer quotes it.
	c := validK8sConfig()
	c.K8s.LoadBalancer.Annotations = []string{"external-dns.alpha.kubernetes.io/target: https://x:8443"}
	c.K8s.Placement.LabelsPrimary = []string{"nodetype: solace"}
	if err := c.Validate(K8s); err != nil {
		t.Errorf("a quoted-safe value must be accepted: %v", err)
	}
}

// TestValidatePullPolicy covers the enum plus the empty case, which must stay
// legal: unset means the renderer keeps writing the IfNotPresent it always did.
func TestValidatePullPolicy(t *testing.T) {
	for _, ok := range []string{"", "Always", "IfNotPresent", "Never"} {
		c := validK8sConfig()
		c.Image.PullPolicy = ok
		if err := c.Validate(K8s); err != nil {
			t.Errorf("pullPolicy %q must be accepted: %v", ok, err)
		}
	}
	c := validK8sConfig()
	c.Image.PullPolicy = "always" // k8s is case-sensitive here
	if err := c.Validate(K8s); err == nil || !strings.Contains(err.Error(), "image.pullPolicy must be") {
		t.Errorf("expected a pullPolicy enum error, got: %v", err)
	}
}

// TestValidatePlacementAffinity covers the additive affinity blocks: an operator or
// topologyKey the API server would reject should fail here, where the message can
// name the field.
func TestValidatePlacementAffinity(t *testing.T) {
	cases := []struct {
		name  string
		apply func(*Config)
		want  string
	}{
		{"unknown operator", func(c *Config) {
			c.K8s.Placement.NodeAffinity.Required = []NodeMatchExpr{{Key: "k", Operator: "Contains"}}
		}, "operator \"Contains\" is invalid"},
		{"missing key", func(c *Config) {
			c.K8s.Placement.NodeAffinity.Required = []NodeMatchExpr{{Operator: "Exists"}}
		}, "required[0].key must be set"},
		{"In without values", func(c *Config) {
			c.K8s.Placement.NodeAffinity.Required = []NodeMatchExpr{{Key: "k", Operator: "In"}}
		}, "values must not be empty"},
		{"preferred term operator", func(c *Config) {
			c.K8s.Placement.NodeAffinity.Preferred = []WeightedNodeTerm{
				{Weight: 50, Match: []NodeMatchExpr{{Key: "k", Operator: "nope"}}},
			}
		}, "preferred[0].match[0].operator"},
		{"pod affinity without topologyKey", func(c *Config) {
			c.K8s.Placement.PodAffinity = []PodAffinityTerm{{MatchLabels: map[string]string{"a": "b"}}}
		}, "podAffinity[0].topologyKey must be set"},
		{"pod anti-affinity without topologyKey", func(c *Config) {
			c.K8s.Placement.PodAntiAffinity = []PodAffinityTerm{{Weight: 10}}
		}, "podAntiAffinity[0].topologyKey must be set"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := validK8sConfig()
			tc.apply(c)
			err := c.Validate(K8s)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected an error containing %q, got: %v", tc.want, err)
			}
		})
	}

	// A fully populated, valid set must pass, including Exists with no values.
	c := validK8sConfig()
	c.K8s.Placement.NodeAffinity = NodeAffinity{
		Preferred: []WeightedNodeTerm{{Weight: 100, Match: []NodeMatchExpr{{Key: "zone", Operator: "In", Values: []string{"a"}}}}},
		Required:  []NodeMatchExpr{{Key: "solace", Operator: "Exists"}},
	}
	c.K8s.Placement.PodAffinity = []PodAffinityTerm{{TopologyKey: "kubernetes.io/hostname", Weight: 5}}
	c.K8s.Placement.PodAntiAffinity = []PodAffinityTerm{{TopologyKey: "topology.kubernetes.io/zone"}}
	if err := c.Validate(K8s); err != nil {
		t.Errorf("a valid affinity set must be accepted: %v", err)
	}
}

// TestDefaultK8sPortsMatchesOperator pins the built-in port list against the
// operator's own default, whose leading entry (tcp-ssh) this tool used to omit.
func TestDefaultK8sPortsMatchesOperator(t *testing.T) {
	ports := defaultK8sPorts()
	if len(ports) != 17 {
		t.Errorf("defaultK8sPorts has %d entries, want 17 (the operator's own default)", len(ports))
	}
	if ports[0] != "tcp-ssh=2222" {
		t.Errorf("defaultK8sPorts[0] = %q, want tcp-ssh=2222 (the operator lists it first)", ports[0])
	}
	seen := map[string]bool{}
	for _, p := range ports {
		name, _, _ := strings.Cut(p, "=")
		if seen[name] {
			t.Errorf("duplicate port name %q in defaultK8sPorts", name)
		}
		seen[name] = true
	}
}

func TestValidateContainerBadNetworkMode(t *testing.T) {
	c := validContainerConfig(Podman, "yes")
	c.Podman.Network.Mode = "sidecar"
	if err := c.Validate(Podman); err == nil || !strings.Contains(err.Error(), "network.mode must be") {
		t.Errorf("expected bad network mode error, got: %v", err)
	}
}

func TestValidateDockerBadMode(t *testing.T) {
	c := validContainerConfig(Docker, "yes")
	c.Docker.Mode = "swarm"
	if err := c.Validate(Docker); err == nil || !strings.Contains(err.Error(), "docker.mode must be") {
		t.Errorf("expected docker.mode enum error, got: %v", err)
	}
}

// TestValidateDockerRunModeRemoved pins the removed value's own error: an env file
// migrated from run mode has to be told why and what to set, not just that the
// enum is wrong.
func TestValidateDockerRunModeRemoved(t *testing.T) {
	c := validContainerConfig(Docker, "yes")
	c.Docker.Mode = "run"
	err := c.Validate(Docker)
	if err == nil || !strings.Contains(err.Error(), "was removed") {
		t.Fatalf("expected the run-mode removal error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "docker.compose") {
		t.Errorf("removal error should mention docker.compose, got: %v", err)
	}
}

// TestValidateDockerComposeCommand covers the compose command as an exec-bound
// Command, the same boundary check k8s.runtime and docker.runtime get.
func TestValidateDockerComposeCommand(t *testing.T) {
	c := validContainerConfig(Docker, "yes")
	c.Docker.Compose = Command{"docker", ""}
	if err := c.Validate(Docker); err == nil || !strings.Contains(err.Error(), "docker.compose[1]") {
		t.Errorf("expected an empty-argument error for docker.compose, got: %v", err)
	}
}

func TestValidateUnknownPlatform(t *testing.T) {
	c := &Config{Redundancy: "yes"}
	err := c.Validate(Platform("nope"))
	if err == nil || !strings.Contains(err.Error(), "unknown platform") {
		t.Errorf("expected unknown platform error, got: %v", err)
	}
}

func TestValidateBadRedundancy(t *testing.T) {
	c := &Config{Redundancy: "sometimes"}
	err := c.Validate(K8s)
	if err == nil || !strings.Contains(err.Error(), "redundancy must be") {
		t.Errorf("expected redundancy enum error, got: %v", err)
	}
}

// envTree writes the fixture layout the resolver is exercised against and
// returns its root: a base-dir copy that shadows an env/ copy of the same name,
// a file reachable only through the env/ fallback, the default name in env/, a
// nested path outside env/ plus a same-named decoy inside it, and a directory
// occupying a candidate path.
func envTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, rel := range []string{
		"solo.yaml",          // base dir only
		"dev.yaml",           // shadows env/dev.yaml
		"env/dev.yaml",       // shadowed
		"env/only.yaml",      // env/ fallback only
		"env/env.yaml",       // the default name, via the fallback
		"sub/nested.yaml",    // reached as a path
		"env/sub/decoy.yaml", // must stay unreachable from "sub/decoy.yaml"
	} {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatalf("MkdirAll %s: %v", p, err)
		}
		if err := os.WriteFile(p, []byte("redundancy: \"no\"\n"), 0o600); err != nil {
			t.Fatalf("WriteFile %s: %v", p, err)
		}
	}
	if err := os.MkdirAll(filepath.Join(root, "adir"), 0o755); err != nil {
		t.Fatalf("MkdirAll adir: %v", err)
	}
	return root
}

func TestResolveEnvPath(t *testing.T) {
	root := envTree(t)
	at := func(rel string) string { return filepath.Join(root, filepath.FromSlash(rel)) }

	tests := []struct {
		name    string
		base    string
		in      string
		want    string
		wantErr []string
	}{
		{name: "bare name in the base dir", base: root, in: "solo.yaml", want: at("solo.yaml")},
		{name: "base dir shadows env dir", base: root, in: "dev.yaml", want: at("dev.yaml")},
		{name: "falls back to the env dir", base: root, in: "only.yaml", want: at("env/only.yaml")},
		{name: "empty name uses the default", base: root, in: "", want: at("env/env.yaml")},
		{
			name: "no extension is inferred", base: root, in: "dev",
			wantErr: []string{`env file "dev" not found`, strconv.Quote(at("env/dev"))},
		},
		{name: "path is used verbatim", base: root, in: at("sub/nested.yaml"), want: at("sub/nested.yaml")},
		{
			// A path names one file; it must not be retried under env/.
			name: "path does not fall back to the env dir", base: root, in: at("sub/decoy.yaml"),
			wantErr: []string{"not found", strconv.Quote(at("sub/decoy.yaml"))},
		},
		{
			name: "base dir is ignored for a path", base: root, in: filepath.FromSlash("sub/nested.yaml"),
			wantErr: []string{strconv.Quote(filepath.FromSlash("sub/nested.yaml"))},
		},
		{
			name: "a directory is not a match", base: root, in: "adir",
			wantErr: []string{`env file "adir" not found`},
		},
		{
			name: "not found names both candidates", base: root, in: "missing.yaml",
			wantErr: []string{strconv.Quote(at("missing.yaml")), strconv.Quote(at("env/missing.yaml"))},
		},
		{
			name: "control characters are rejected", base: root, in: "dev\n.yaml",
			wantErr: []string{"control characters are not allowed"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveEnvPath(tc.base, tc.in)
			if len(tc.wantErr) > 0 {
				if err == nil {
					t.Fatalf("ResolveEnvPath(%q, %q) = %q, want an error", tc.base, tc.in, got)
				}
				for _, want := range tc.wantErr {
					if !strings.Contains(err.Error(), want) {
						t.Errorf("error %q does not contain %q", err.Error(), want)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolveEnvPath(%q, %q) returned error: %v", tc.base, tc.in, err)
			}
			if got != tc.want {
				t.Errorf("ResolveEnvPath(%q, %q) = %q, want %q", tc.base, tc.in, got, tc.want)
			}
		})
	}
}

// An empty baseDir means the current directory, for both candidates.
func TestResolveEnvPathEmptyBaseDir(t *testing.T) {
	root := envTree(t)
	t.Chdir(root)

	cases := []struct{ in, want string }{
		{"dev.yaml", "dev.yaml"},
		{"only.yaml", filepath.Join("env", "only.yaml")},
		{"", filepath.Join("env", EnvFileDefault)},
		{filepath.FromSlash("sub/nested.yaml"), filepath.FromSlash("sub/nested.yaml")},
	}
	for _, tc := range cases {
		got, err := ResolveEnvPath("", tc.in)
		if err != nil {
			t.Errorf("ResolveEnvPath(\"\", %q) returned error: %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ResolveEnvPath(\"\", %q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// The default name resolves in the base dir before the env/ fallback is tried.
func TestResolveEnvPathDefaultInBaseDir(t *testing.T) {
	root := t.TempDir()
	want := filepath.Join(root, EnvFileDefault)
	if err := os.WriteFile(want, []byte("redundancy: \"no\"\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	got, err := ResolveEnvPath(root, "")
	if err != nil {
		t.Fatalf("ResolveEnvPath returned error: %v", err)
	}
	if got != want {
		t.Errorf("ResolveEnvPath(%q, \"\") = %q, want %q", root, got, want)
	}
}

func writeTempYAML(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "env.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

func TestLoadSuccess(t *testing.T) {
	yaml := `redundancy: "no"
image:
  repo: solace/broker
  tag: latest
admin:
  pass: s3cret
k8s:
  name: mybroker
  namespace: sol-ns
  storage:
    msgNode: 30Gi
`
	path := writeTempYAML(t, yaml)
	c, err := Load(path, K8s)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if c == nil {
		t.Fatal("Load returned nil config")
	}
	if c.K8s.Name != "mybroker" || c.K8s.Namespace != "sol-ns" {
		t.Errorf("loaded config = %+v", c.K8s)
	}
	// Defaults were applied during Load.
	if c.K8s.UpdateStrategy != "automatedRolling" {
		t.Errorf("UpdateStrategy default not applied: %q", c.K8s.UpdateStrategy)
	}
}

func TestLoadReadError(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "does-not-exist.yaml"), K8s)
	if err == nil || !strings.Contains(err.Error(), "read env file") {
		t.Errorf("expected read error, got: %v", err)
	}
}

func TestLoadParseError(t *testing.T) {
	path := writeTempYAML(t, "]")
	_, err := Load(path, K8s)
	if err == nil || !strings.Contains(err.Error(), "parse env file") {
		t.Errorf("expected parse error, got: %v", err)
	}
}

func TestLoadUnknownField(t *testing.T) {
	path := writeTempYAML(t, "bogusTopLevelKey: 123\n")
	_, err := Load(path, K8s)
	if err == nil || !strings.Contains(err.Error(), "parse env file") {
		t.Errorf("expected unknown-field parse error, got: %v", err)
	}
}

// A legacy bash env file handed to -e is the likeliest way to get a file that is
// not YAML, so the error says so and names the converter.
func TestLoadBashEnvFileHint(t *testing.T) {
	path := writeTempYAML(t, "#!/bin/bash\nSOLBK_NAME=\"broker\"\nSOLBK_NS=\"solace\"\n")
	_, err := Load(path, K8s)
	if err == nil {
		t.Fatal("a bash env file should not load")
	}
	for _, want := range []string{"not valid YAML", "this looks like a legacy bash env file", "solace convert"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err.Error(), want)
		}
	}
}

// Any other non-YAML file still says the file must be YAML, and still points at
// the converter as the way to migrate a bash one.
func TestLoadNotYAMLHint(t *testing.T) {
	path := writeTempYAML(t, "this is just a sentence, not a mapping\n")
	_, err := Load(path, K8s)
	if err == nil {
		t.Fatal("a non-YAML file should not load")
	}
	for _, want := range []string{"not valid YAML", "env/sample.yaml", "solace convert"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err.Error(), want)
		}
	}
	// The generic hint mentions converting a bash env file, so the check is for
	// the identification phrase the bash-specific branch adds, not the words.
	if strings.Contains(err.Error(), "this looks like a legacy bash env file") {
		t.Errorf("a plain non-YAML file must not be identified as a bash env file: %v", err)
	}
}

// A file that is valid YAML but carries an unknown key is a schema problem, not
// a format one: it must not gain the convert hint.
func TestLoadUnknownFieldHasNoConvertHint(t *testing.T) {
	path := writeTempYAML(t, "bogusTopLevelKey: 123\n")
	_, err := Load(path, K8s)
	if err == nil {
		t.Fatal("an unknown key should not load")
	}
	if strings.Contains(err.Error(), "solace convert") || strings.Contains(err.Error(), "not valid YAML") {
		t.Errorf("unknown-key error should stay a schema error, got: %v", err)
	}
}

func TestLoadValidationError(t *testing.T) {
	// Parses fine, but mandatory fields are missing -> Validate fails.
	path := writeTempYAML(t, "redundancy: \"no\"\n")
	_, err := Load(path, K8s)
	if err == nil || !strings.Contains(err.Error(), "these fields must not be empty:") {
		t.Errorf("expected validation error, got: %v", err)
	}
}
