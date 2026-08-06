package config

import (
	"os"
	"path/filepath"
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
	c.Docker.Runtime = "docker"
	c.Podman.Runtime = "podman"
	tests := []struct {
		p    Platform
		want string
	}{
		{Docker, "docker"},
		{Podman, "podman"},
		{K8s, ""},
	}
	for _, tc := range tests {
		if got := c.ContainerRuntime(tc.p); got != tc.want {
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

	if c.Redundancy != "yes" {
		t.Errorf("Redundancy default = %q, want %q", c.Redundancy, "yes")
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
	if b.TZ != "UTC" {
		t.Errorf("TZ = %q, want UTC", b.TZ)
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

	if c.Docker.Runtime != "docker" {
		t.Errorf("Docker.Runtime = %q, want docker", c.Docker.Runtime)
	}
	if c.Docker.Mode != "compose" {
		t.Errorf("Docker.Mode = %q, want compose", c.Docker.Mode)
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

	if c.Podman.Runtime != "podman" {
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

func TestResolveEnvPath(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty -> default", "", filepath.Join("/base", "env", "default.yaml")},
		{"named", "dev", filepath.Join("/base", "env", "dev.yaml")},
	}
	for _, tc := range tests {
		if got := ResolveEnvPath("/base", tc.in); got != tc.want {
			t.Errorf("%s: ResolveEnvPath(/base, %q) = %q, want %q", tc.name, tc.in, got, tc.want)
		}
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

func TestLoadValidationError(t *testing.T) {
	// Parses fine, but mandatory fields are missing -> Validate fails.
	path := writeTempYAML(t, "redundancy: \"no\"\n")
	_, err := Load(path, K8s)
	if err == nil || !strings.Contains(err.Error(), "these fields must not be empty:") {
		t.Errorf("expected validation error, got: %v", err)
	}
}
