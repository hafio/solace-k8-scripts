package convert

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"solace/internal/config"
)

// update regenerates the converter goldens, matching the render and k8s packages.
var update = flag.Bool("update", false, "regenerate the goldens in testdata/")

// checkGolden compares got against testdata/<file>, or rewrites it under -update.
func checkGolden(t *testing.T, file string, got []byte) {
	t.Helper()
	path := filepath.Join("testdata", file)
	if *update {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatalf("write golden %s: %v", path, err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s (regenerate: go test ./internal/convert -update): %v", path, err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("%s mismatch\n--- got ---\n%s\n--- want ---\n%s", file, got, want)
	}
}

// legacyK8sEnv is this package's own legacy-format fixture. It deliberately does
// not point at bash/env/sample: the bash/ tree is gitignored, so that path does
// not exist on a fresh checkout and CI could never run this test.
const legacyK8sEnv = "testdata/legacy-k8s.env"

// ctrEnv is a container-flavoured legacy env file. There is no committed one --
// the container bootstrap documented its variables inline -- so the fixture is
// written here from the names 000-env.sh reads.
const ctrEnv = `#!/bin/bash
SOLBK_IMAGE="solace-pubsub-standard"
SOLBK_IMG_TAG="10.10.1.128"
SOLBK_ADM_PASS="secret-pass"
SOLBK_REDUNDANCY="yes"
SOLBK_DATA_DIR="/opt/solace/data"
SOLBK_NETWORK_MODE="host"
SOLBK_RUN_USER="1000:1000"
SOLBK_TZ="UTC"
SOLBK_SHM_SIZE="1g"
SOLBK_ULIMIT_NOFILE="2448:1048576"
SOLBK_ULIMIT_MEMLOCK="-1"
SOLBK_ULIMIT_CORE="-1"
SOLBK_SPOOL_MAXUSAGE="100000"
CONTAINER_NAME="solace"
SOLBK_NODE_PRI_NAME="pri-host"
SOLBK_NODE_PRI_IP="10.0.0.1"
SOLBK_NODE_BKP_NAME="bkp-host"
SOLBK_NODE_BKP_IP="10.0.0.2"
SOLBK_NODE_MON_NAME="mon-host"
SOLBK_NODE_MON_IP="10.0.0.3"
SOLBK_REDUNDANCY_PSK="PSKVALUE"
`

// k8sEnv is a minimal but complete k8s-flavoured legacy env file: it carries
// every mandatory value, so a conversion produces no "incomplete" warning and a
// test asserting on KUBE is reading only its own warnings.
const k8sEnv = `#!/bin/bash
SOLBK_NAME="dev-broker"
SOLBK_NS="solace"
SOLBK_IMAGE="solace-pubsub-standard"
SOLBK_IMG_TAG="10.10.1.128"
SOLBK_STORAGE_MSGNODE="30Gi"
SOLBK_ADM_PASS="secret-pass"
`

// strictDecode re-reads generated YAML with the same strict decoder Load uses,
// so an emitted key that is not in the schema fails the test rather than being
// silently ignored.
func strictDecode(t *testing.T, out []byte) *config.Config {
	t.Helper()
	var c config.Config
	dec := yaml.NewDecoder(bytes.NewReader(out))
	dec.KnownFields(true)
	if err := dec.Decode(&c); err != nil {
		t.Fatalf("generated YAML does not decode against the schema: %v\n%s", err, out)
	}
	return &c
}

func convertOK(t *testing.T, src string, p config.Platform) Result {
	t.Helper()
	res, err := Convert([]byte(src), "fixture", p)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	return res
}

// TestConvertUserPasswordsBecomeAdditionalUsers covers the one legacy variable with
// no like-for-like successor: SOLBK_USR_PASS was a flat "user=password" list, and the
// schema now wants a structured user with an explicit access level. A malformed entry
// is dropped with a warning that must name its POSITION and not its text -- the text
// of a malformed entry is most likely a bare password.
func TestConvertUserPasswordsBecomeAdditionalUsers(t *testing.T) {
	const secret = "UNIQUE-BARE-PASSWORD"
	src := k8sEnv + "declare -a SOLBK_USR_PASS=(\"appuser=app-pass\" \"" + secret + "\" \"=nouser\")\n"
	res := convertOK(t, src, config.K8s)
	c := strictDecode(t, res.YAML)

	if len(c.Admin.AdditionalUsers) != 1 {
		t.Fatalf("additionalUsers = %+v, want only the one well-formed entry", c.Admin.AdditionalUsers)
	}
	u := c.Admin.AdditionalUsers[0]
	if u.Username != "appuser" || u.Password != "app-pass" || u.AccessLevel != "none" {
		t.Errorf("converted user = %+v, want appuser/app-pass with the least-privileged level", u)
	}
	// convertOK succeeding is itself the proof that the emitted level is a valid
	// one: Convert re-reads and validates its own output before returning.
	for _, want := range []string{"entry 1 is not user=password", "entry 2 is not user=password", "accessLevel: none"} {
		if !hasWarning(res.Warnings, want) {
			t.Errorf("warnings %q should mention %q", res.Warnings, want)
		}
	}
	for _, w := range res.Warnings {
		if strings.Contains(w, secret) {
			t.Errorf("a warning carried the malformed entry verbatim, which is the password: %q", w)
		}
	}
}

// TestConvertLegacyK8sEnv converts this package's own legacy fixture end to end
// and pins the result against a committed golden, so the test owns both halves:
// the legacy input and the YAML it must produce.
func TestConvertLegacyK8sEnv(t *testing.T) {
	raw, err := os.ReadFile(legacyK8sEnv)
	if err != nil {
		t.Fatalf("read %s: %v", legacyK8sEnv, err)
	}
	res, err := Convert(raw, legacyK8sEnv, "")
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	checkGolden(t, "legacy-k8s.yaml.golden", res.YAML)
	if res.Platform != config.K8s {
		t.Errorf("detected platform = %q, want k8s", res.Platform)
	}
	c := strictDecode(t, res.YAML)

	// SOLBK_REDUNDANCY="true" is the k8s spelling of yes.
	if c.Redundancy != "yes" {
		t.Errorf("redundancy = %q, want yes", c.Redundancy)
	}
	if c.Image.Repo != "solace-pubsub-standard" || c.Image.Tag != "latest" || c.Image.Registry != "localhost" {
		t.Errorf("image = %+v", c.Image)
	}
	if c.Image.PullSecret != "registry-pull-secret" || c.Image.User != "docker-user" || c.Image.Pass != "docker-pass" {
		t.Errorf("image credentials = %+v", c.Image)
	}
	if c.K8s.Name != "solace-broker" || c.K8s.Namespace != "solace-namespace" {
		t.Errorf("k8s identity = %q/%q", c.K8s.Name, c.K8s.Namespace)
	}
	if c.K8s.Storage.MsgNode != "16100Mi" || c.K8s.Storage.MonNode != "2300Mi" || c.K8s.Storage.Class != "solace-storage-class" {
		t.Errorf("storage = %+v", c.K8s.Storage)
	}
	if c.Admin.Pass != "adminpassword123" || c.Admin.MonitorPass != "monitorpassword123" {
		t.Error("admin credentials did not carry over")
	}
	if c.K8s.Operator.CPU != "500m" || c.K8s.Operator.Mem != "512Mi" {
		t.Errorf("operator resources = %+v", c.K8s.Operator)
	}
	if c.Scaling.MaxConnections != 100 || c.Scaling.MaxSubscriptions != 50000 {
		t.Errorf("scaling = %+v", c.Scaling)
	}
	// A zero in the source is a real value, not an absent one.
	if c.Scaling.MaxKafkaBridge != 0 || !strings.Contains(string(res.YAML), "maxKafkaBridge: 0") {
		t.Error("maxKafkaBridge: 0 should be written explicitly")
	}
	// Indexed array.
	if len(c.TLS.CAs) != 1 || c.TLS.CAs[0] != "/path/to/ca/cert" {
		t.Errorf("tls.cas = %v", c.TLS.CAs)
	}
	// Associative array.
	if c.Broker.DomainCerts.Files["CERT_NAME"] != "cert.crt" {
		t.Errorf("domainCerts.files = %v", c.Broker.DomainCerts.Files)
	}
	// ${SOLBK_NS} expanded from the earlier assignment.
	if len(c.K8s.Placement.AntiAffinityNS) != 1 || c.K8s.Placement.AntiAffinityNS[0] != "solace-namespace" {
		t.Errorf("antiAffinityNamespaces = %v, want [solace-namespace]", c.K8s.Placement.AntiAffinityNS)
	}
	if c.K8s.Placement.AntiAffinityWeight != 100 {
		t.Errorf("antiAffinityWeight = %d, want 100", c.K8s.Placement.AntiAffinityWeight)
	}
	// Trailing comment stripped from an unquoted value.
	if c.Replication.Mate != "mate-virtual-router-name" {
		t.Errorf("replication.mate = %q", c.Replication.Mate)
	}
	if len(c.Replication.ConnSSL) != 3 || c.Replication.ConnSSL[0] != "host:port" {
		t.Errorf("replication.connSsl = %v", c.Replication.ConnSSL)
	}
	// REPL_PSK="" is empty, so it must not appear at all.
	if strings.Contains(string(res.YAML), "psk:") {
		t.Errorf("an empty PSK should be omitted:\n%s", res.YAML)
	}
	// KUBE was expanded unquoted by the bash scripts, so a whole kubectl profile
	// has to survive the conversion as kubernetes.runtime, split into argv.
	wantRuntime := config.Command{"kubectl", "--kubeconfig", "/home/localadmin/.kubeconfig-dev"}
	if c.K8s.Runtime.String() != wantRuntime.String() {
		t.Errorf("kubernetes.runtime = %v, want %v", c.K8s.Runtime, wantRuntime)
	}
	// EXDIR is bash plumbing; every other variable in the file is mapped, so the
	// only warning allowed here is the replication advisory -- this fixture does set
	// REPL_MATE/REPL_CONN_SSL, and they map into a schema block no command reads
	// yet. Anything else is still a failure.
	// SOLBK_MSGNODE_CPU is the second allowed warning: the fixture sets it, as
	// every real legacy file does, and broker CPU is now fixed by the scaling
	// tier -- so it is dropped with a reason rather than carried over.
	for _, w := range res.Warnings {
		if strings.Contains(w, "REPL_MATE") || strings.Contains(w, "SOLBK_MSGNODE_CPU") {
			continue
		}
		t.Errorf("unexpected warning converting the sample: %s", w)
	}
	if !hasWarning(res.Warnings, "REPL_MATE") {
		t.Error("the sample configures replication, so the inert-block advisory should have fired")
	}
	if !hasWarning(res.Warnings, "SOLBK_MSGNODE_CPU") {
		t.Error("the sample sets SOLBK_MSGNODE_CPU, so the removal advisory should have fired")
	}
	// SOLOP_CPU is "500m" in this fixture, so a bare `cpu: "2"` could only be the
	// dropped msgNode one.
	if strings.Contains(string(res.YAML), `cpu: "2"`) {
		t.Errorf("SOLBK_MSGNODE_CPU must not reach the YAML:\n%s", res.YAML)
	}
}

// TestConvertAdminUserIsContainerOnly covers the one admin field that is not portable:
// SOLBK_ADM_USER named the container's admin user, but on Kubernetes the operator reads
// the fixed username_admin_password key out of the credentials Secret, so validateK8s
// refuses any other admin.user. Carrying the value into a k8s document would turn a
// working bash env file into YAML that will not load, so it is dropped with the reason
// named -- the same treatment SOLBK_MSGNODE_CPU gets.
func TestConvertAdminUserIsContainerOnly(t *testing.T) {
	t.Run("k8s drops it and says why", func(t *testing.T) {
		res := convertOK(t, k8sEnv+"SOLBK_ADM_USER=\"ops\"\n", config.K8s)
		if got := strictDecode(t, res.YAML).Admin.User; got != "" {
			t.Errorf("admin.user = %q, want it absent on k8s:\n%s", got, res.YAML)
		}
		if !hasWarning(res.Warnings, "SOLBK_ADM_USER") {
			t.Errorf("dropping the value must be reported; warnings = %v", res.Warnings)
		}
		// The variable is read, not ignored: an unread one would resurface in the generic
		// unmapped list, which says nothing about why the value cannot be carried over.
		if hasWarning(res.Warnings, "no YAML equivalent") {
			t.Errorf("SOLBK_ADM_USER is mapped, not unmapped; warnings = %v", res.Warnings)
		}
		// The whole point of dropping it rather than emitting it: what is written loads.
		if hasWarning(res.Warnings, "will not load as-is") {
			t.Errorf("the converted k8s file must still validate; warnings = %v", res.Warnings)
		}
	})
	t.Run("k8s says nothing when the value was already admin", func(t *testing.T) {
		res := convertOK(t, k8sEnv+"SOLBK_ADM_USER=\"admin\"\n", config.K8s)
		if hasWarning(res.Warnings, "SOLBK_ADM_USER") {
			t.Errorf("admin is what k8s uses regardless, so nothing was lost; warnings = %v", res.Warnings)
		}
	})
	t.Run("container keeps it", func(t *testing.T) {
		res := convertOK(t, ctrEnv+"SOLBK_ADM_USER=\"ops\"\n", config.Docker)
		if got := strictDecode(t, res.YAML).Admin.User; got != "ops" {
			t.Errorf("admin.user = %q, want ops on docker:\n%s", got, res.YAML)
		}
		if hasWarning(res.Warnings, "SOLBK_ADM_USER") {
			t.Errorf("the container platforms honour the variable; warnings = %v", res.Warnings)
		}
	})
}

func TestConvertContainer(t *testing.T) {
	res := convertOK(t, ctrEnv, "")
	if res.Platform != config.Docker {
		t.Errorf("detected platform = %q, want docker (no docker/podman marker)", res.Platform)
	}
	c := strictDecode(t, res.YAML)
	if c.Nodes.Primary.Name != "pri-host" || c.Nodes.Primary.IP != "10.0.0.1" {
		t.Errorf("nodes.primary = %+v", c.Nodes.Primary)
	}
	if c.Nodes.Monitor.Name != "mon-host" || c.Nodes.PSK != "PSKVALUE" {
		t.Errorf("nodes = %+v", c.Nodes)
	}
	if c.Docker.Container.DataDir != "/opt/solace/data" || c.Docker.Container.RunUser != "1000:1000" {
		t.Errorf("docker.container = %+v", c.Docker.Container)
	}
	if c.Docker.Container.Ulimits.MemLock != "-1" || c.Docker.Container.Ulimits.NoFile != "2448:1048576" {
		t.Errorf("ulimits = %+v", c.Docker.Container.Ulimits)
	}
	if c.Docker.Network.Mode != "host" {
		t.Errorf("docker.network.mode = %q", c.Docker.Network.Mode)
	}
	if c.Scaling.MaxSpoolUsageMB != 100000 {
		t.Errorf("maxSpoolUsageMB = %d", c.Scaling.MaxSpoolUsageMB)
	}
	// SOLBK_TZ was container-only in bash; the schema has one cross-platform key.
	if c.Timezone != "UTC" {
		t.Errorf("timezone = %q, want UTC (from SOLBK_TZ)", c.Timezone)
	}
	// An ambiguous container file says which section it picked.
	if !hasWarning(res.Warnings, "assumed docker") {
		t.Errorf("warnings = %v, want the assumed-docker note", res.Warnings)
	}
}

func TestConvertPlatformDetection(t *testing.T) {
	cases := []struct {
		name  string
		extra string
		want  config.Platform
	}{
		{"podman markers", "PODMAN_ROOTLESS=\"true\"\nQUADLET_DIR=\"/etc/containers/systemd\"\n", config.Podman},
		{"docker markers", "DOCKER_MODE=\"run\"\n", config.Docker},
		{"both container families", "DOCKER_MODE=\"run\"\nPODMAN_ROOTLESS=\"true\"\n", config.Docker},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := convertOK(t, ctrEnv+tc.extra, "")
			if res.Platform != tc.want {
				t.Errorf("platform = %q, want %q", res.Platform, tc.want)
			}
			strictDecode(t, res.YAML)
		})
	}
}

func TestConvertPodmanSection(t *testing.T) {
	res := convertOK(t, ctrEnv+"PODMAN_ROOTLESS=\"true\"\nQUADLET_DIR=\"/home/u/.config/containers/systemd\"\n", "")
	c := strictDecode(t, res.YAML)
	if !c.Podman.Rootless {
		t.Error("podman.rootless should be true")
	}
	if c.Podman.QuadletDir != "/home/u/.config/containers/systemd" {
		t.Errorf("podman.quadletDir = %q", c.Podman.QuadletDir)
	}
	if c.Podman.Container.Name != "solace" {
		t.Errorf("podman.container.name = %q", c.Podman.Container.Name)
	}
	// The docker section must stay empty when podman was selected.
	if strings.Contains(string(res.YAML), "\ndocker:") {
		t.Errorf("podman conversion emitted a docker section:\n%s", res.YAML)
	}
}

func TestConvertExplicitPlatformWins(t *testing.T) {
	res := convertOK(t, ctrEnv, config.Podman)
	if res.Platform != config.Podman {
		t.Errorf("platform = %q, want podman", res.Platform)
	}
	if hasWarning(res.Warnings, "assumed docker") {
		t.Error("an explicit platform must not emit a detection warning")
	}
	strictDecode(t, res.YAML)
}

func TestConvertUnmappedVariablesWarn(t *testing.T) {
	res := convertOK(t, ctrEnv+"SOMETHING_CUSTOM=\"x\"\nANOTHER_ONE=\"y\"\n", config.Docker)
	if !hasWarning(res.Warnings, "SOMETHING_CUSTOM") || !hasWarning(res.Warnings, "ANOTHER_ONE") {
		t.Errorf("warnings = %v, want both unmapped names", res.Warnings)
	}
}

func TestConvertBashPlumbingIsSilent(t *testing.T) {
	res := convertOK(t, ctrEnv+"EXDIR=\".\"\nGENONLY=true\n", config.Docker)
	for _, name := range []string{"EXDIR", "GENONLY"} {
		if hasWarning(res.Warnings, name) {
			t.Errorf("%s is bash plumbing and should be dropped silently; warnings = %v", name, res.Warnings)
		}
	}
}

// TestConvertKubeMapsToK8sRuntime covers the shapes KUBE actually carried in the
// field: bash expanded it unquoted, so a wrapper or a whole kubectl profile had
// to survive as argv, not just a binary name.
func TestConvertKubeMapsToK8sRuntime(t *testing.T) {
	cases := []struct {
		name, kube, want string
	}{
		{"drop-in binary", `KUBE="oc"`, "oc"},
		{"wrapper", `KUBE="microk8s kubectl"`, "microk8s kubectl"},
		{"kubeconfig profile", `KUBE="kubectl --kubeconfig /tmp/kc"`, "kubectl --kubeconfig /tmp/kc"},
		{"absolute path", `KUBE="/usr/local/bin/kubectl"`, "/usr/local/bin/kubectl"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := convertOK(t, k8sEnv+tc.kube+"\n", config.K8s)
			if got := strictDecode(t, res.YAML).K8s.Runtime; got.String() != tc.want {
				t.Errorf("kubernetes.runtime = %q, want %q", got, tc.want)
			}
			if hasWarning(res.Warnings, "KUBE") {
				t.Errorf("KUBE maps to kubernetes.runtime now and must not warn; warnings = %v", res.Warnings)
			}
		})
	}
}

// TestConvertKubeEchoIsDropped pins the one KUBE value that must not carry over:
// "echo" was the bash trick for previewing commands, and as a real
// kubernetes.runtime it would turn every cluster call into a no-op whose stdout the
// parsing steps then misread. The warning has to name what replaced it, which is
// `generate` -- rendering the artifact rather than faking the command away.
func TestConvertKubeEchoIsDropped(t *testing.T) {
	res := convertOK(t, k8sEnv+"KUBE=\"echo\"\n", config.K8s)
	if !hasWarning(res.Warnings, "generate") {
		t.Errorf("KUBE=echo should warn and point at generate; warnings = %v", res.Warnings)
	}
	if got := strictDecode(t, res.YAML).K8s.Runtime; len(got) != 0 {
		t.Errorf("kubernetes.runtime = %q, want it omitted so the kubectl default applies", got)
	}
}

// TestConvertKubeSilentOnContainerPlatform: KUBE belongs to the k8s bootstrap, so
// a container conversion consumes it without emitting it and without warning.
func TestConvertKubeSilentOnContainerPlatform(t *testing.T) {
	res := convertOK(t, ctrEnv+"KUBE=\"kubectl\"\n", config.Docker)
	if hasWarning(res.Warnings, "KUBE") {
		t.Errorf("KUBE must be consumed silently on a container platform; warnings = %v", res.Warnings)
	}
	if got := strictDecode(t, res.YAML).K8s.Runtime; len(got) != 0 {
		t.Errorf("a container conversion must not emit kubernetes.runtime, got %q", got)
	}
}

func TestConvertRedundancySpellings(t *testing.T) {
	// want is the emitted line, not just the value: yes/no are YAML-ambiguous and
	// so are quoted, while a pass-through value like "maybe" is a plain scalar.
	cases := []struct {
		in, want string
		warn     bool
	}{
		{"true", `redundancy: "yes"`, false},
		{"yes", `redundancy: "yes"`, false},
		{"false", `redundancy: "no"`, false},
		{"no", `redundancy: "no"`, false},
		{"YES", `redundancy: "yes"`, false},
		{"maybe", "redundancy: maybe", true},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			src := "SOLBK_IMAGE=\"r\"\nSOLBK_IMG_TAG=\"t\"\nSOLBK_ADM_PASS=\"p\"\nSOLBK_NODE_PRI_NAME=\"h\"\nSOLBK_REDUNDANCY=\"" + tc.in + "\"\n"
			res := convertOK(t, src, config.Docker)
			if !strings.Contains(string(res.YAML), tc.want) {
				t.Errorf("missing %q in:\n%s", tc.want, res.YAML)
			}
			if got := hasWarning(res.Warnings, "SOLBK_REDUNDANCY"); got != tc.warn {
				t.Errorf("warned = %v, want %v (warnings %v)", got, tc.warn, res.Warnings)
			}
		})
	}
}

// TestConvertRedundancyOmitted covers the unset case, whose legacy default
// differed by platform: the k8s bootstrap defaulted to standalone -- what this CLI
// now does, so there is nothing to say -- while the container bootstrap defaulted
// to HA. That divergence has to be named rather than quietly turning a
// three-broker group into a single broker. Either way no key is emitted, so the
// converter keeps its "only what the source set" contract.
func TestConvertRedundancyOmitted(t *testing.T) {
	t.Run("container warns about the changed default", func(t *testing.T) {
		src := "SOLBK_IMAGE=\"r\"\nSOLBK_IMG_TAG=\"t\"\nSOLBK_ADM_PASS=\"p\"\nSOLBK_NODE_PRI_NAME=\"h\"\n"
		res := convertOK(t, src, config.Docker)
		if strings.Contains(string(res.YAML), "redundancy:") {
			t.Errorf("an unset value must emit no key so the CLI default applies:\n%s", res.YAML)
		}
		if !hasWarning(res.Warnings, "SOLBK_REDUNDANCY") {
			t.Errorf("container conversion should name the changed default, got %v", res.Warnings)
		}
	})
	t.Run("k8s stays silent", func(t *testing.T) {
		src := "SOLBK_NAME=\"b\"\nSOLBK_NS=\"ns\"\nSOLBK_IMAGE=\"r\"\nSOLBK_IMG_TAG=\"t\"\nSOLBK_ADM_PASS=\"p\"\nSOLBK_STORAGE_MSGNODE=\"30Gi\"\n"
		res := convertOK(t, src, config.K8s)
		if hasWarning(res.Warnings, "SOLBK_REDUNDANCY") {
			t.Errorf("the k8s default already matched, so there is nothing to warn about: %v", res.Warnings)
		}
	})
}

// TestConvertDockerRunModeWarns pins the removed run mode: carrying the value over
// would only fail validation later, so it is dropped and the reason named.
func TestConvertDockerRunModeWarns(t *testing.T) {
	src := "SOLBK_IMAGE=\"r\"\nSOLBK_IMG_TAG=\"t\"\nSOLBK_ADM_PASS=\"p\"\nSOLBK_NODE_PRI_NAME=\"h\"\nDOCKER_MODE=\"run\"\n"
	res := convertOK(t, src, config.Docker)
	if strings.Contains(string(res.YAML), "mode: run") {
		t.Errorf("run mode must not be carried over:\n%s", res.YAML)
	}
	if !hasWarning(res.Warnings, "DOCKER_MODE") {
		t.Errorf("warnings should name DOCKER_MODE, got %v", res.Warnings)
	}
}

// TestConvertOffTierMaxConnWarns covers the legacy value that has no successor:
// SOLBK_SCALING_MAXCONN was any integer, and it is now one of five tiers. The
// value still converts -- rewriting the operator's declared load would be worse
// than reporting it -- and Convert's own re-validation of its output is what
// surfaces the problem, so this needs no dedicated mapping code.
func TestConvertOffTierMaxConnWarns(t *testing.T) {
	src := "SOLBK_IMAGE=\"r\"\nSOLBK_IMG_TAG=\"t\"\nSOLBK_ADM_PASS=\"p\"\nSOLBK_NODE_PRI_NAME=\"h\"\n" +
		"SOLBK_NETWORK_MODE=\"host\"\nSOLBK_DATA_DIR=\"/opt/solace/data\"\nSOLBK_SCALING_MAXCONN=\"5000\"\n"
	res := convertOK(t, src, config.Docker)
	if !strings.Contains(string(res.YAML), "maxConnections: 5000") {
		t.Errorf("the declared load must still be written for the operator to fix:\n%s", res.YAML)
	}
	if !hasWarning(res.Warnings, "will not load as-is") {
		t.Errorf("warnings should flag that the output does not load, got %v", res.Warnings)
	}
	if !hasWarning(res.Warnings, "scaling.maxConnections must be one of") {
		t.Errorf("warnings should carry the tier error, got %v", res.Warnings)
	}
}

// TestConvertSpoolVariablesUnify covers the two legacy names for one key: the k8s
// bootstrap spelled the spool size SOLBK_SCALING_MAXPOOL and the container one
// SOLBK_SPOOL_MAXUSAGE, and scaling.maxSpoolUsageMB is now the only key. Each
// platform's own name wins, and a file carrying both is told which was used
// rather than having one picked in silence.
func TestConvertSpoolVariablesUnify(t *testing.T) {
	const ctrBase = "SOLBK_IMAGE=\"r\"\nSOLBK_IMG_TAG=\"t\"\nSOLBK_ADM_PASS=\"p\"\nSOLBK_NODE_PRI_NAME=\"h\"\n" +
		"SOLBK_NETWORK_MODE=\"host\"\nSOLBK_DATA_DIR=\"/opt/solace/data\"\n"

	// The k8s name alone still converts, on either platform's file.
	res := convertOK(t, ctrBase+"SOLBK_SCALING_MAXPOOL=\"1500\"\n", config.Docker)
	if !strings.Contains(string(res.YAML), "maxSpoolUsageMB: 1500") {
		t.Errorf("the legacy k8s spelling should map to the surviving key:\n%s", res.YAML)
	}
	if strings.Contains(string(res.YAML), "maxPool:") {
		t.Errorf("maxPool was removed and must not be emitted:\n%s", res.YAML)
	}

	// Both set: this platform's own name wins and the other is reported.
	res = convertOK(t, ctrBase+"SOLBK_SPOOL_MAXUSAGE=\"2500\"\nSOLBK_SCALING_MAXPOOL=\"1500\"\n", config.Docker)
	if !strings.Contains(string(res.YAML), "maxSpoolUsageMB: 2500") {
		t.Errorf("the container bootstrap's own variable should win on docker:\n%s", res.YAML)
	}
	if !hasWarning(res.Warnings, "SOLBK_SPOOL_MAXUSAGE was used") {
		t.Errorf("warnings should name which variable won, got %v", res.Warnings)
	}

	// On k8s the preference is the other way round.
	res = convertOK(t, k8sEnv+"SOLBK_SPOOL_MAXUSAGE=\"2500\"\nSOLBK_SCALING_MAXPOOL=\"1500\"\n", config.K8s)
	if !strings.Contains(string(res.YAML), "maxSpoolUsageMB: 1500") {
		t.Errorf("the k8s bootstrap's own variable should win on k8s:\n%s", res.YAML)
	}
	if !hasWarning(res.Warnings, "SOLBK_SCALING_MAXPOOL was used") {
		t.Errorf("warnings should name which variable won, got %v", res.Warnings)
	}
}

func TestConvertBadNumberWarns(t *testing.T) {
	res := convertOK(t, ctrEnv+"SOLBK_SCALING_MAXCONN=\"lots\"\n", config.Docker)
	if !hasWarning(res.Warnings, "is not a number") {
		t.Errorf("warnings = %v, want a not-a-number warning", res.Warnings)
	}
	if strings.Contains(string(res.YAML), "maxConnections") {
		t.Errorf("a non-numeric value must not be written:\n%s", res.YAML)
	}
}

// A bool the bootstraps never enum-checked (SOLOP_WATCH_SOLBK_NS) must not be
// dropped in silence when it carries an unexpected spelling.
func TestConvertBadBooleanWarns(t *testing.T) {
	res := convertOK(t, "SOLBK_NAME=\"b\"\nSOLOP_WATCH_SOLBK_NS=\"sometimes\"\n", config.K8s)
	if !hasWarning(res.Warnings, "SOLOP_WATCH_SOLBK_NS") {
		t.Errorf("warnings = %v, want a bad-boolean warning", res.Warnings)
	}
	if strings.Contains(string(res.YAML), "watchBrokerNs") {
		t.Errorf("an unparseable bool must not be written:\n%s", res.YAML)
	}
}

// The source name is the one free-form string in the document; a control
// character in it must not be able to end the comment line it sits on.
func TestGeneratedHeaderSanitisesSource(t *testing.T) {
	res, err := Convert([]byte("SOLBK_IMAGE=\"r\"\n"), "evil\nadmin:\n  pass: pwned", config.K8s)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	out := string(res.YAML)
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "admin:") {
			t.Errorf("source name broke out of the header comment:\n%s", out)
		}
	}
	strictDecode(t, res.YAML)
}

func TestConvertIncompleteEnvWarns(t *testing.T) {
	// No image or admin password: valid to convert, but not yet usable.
	res := convertOK(t, "SOLBK_NAME=\"b\"\nSOLBK_NS=\"ns\"\n", config.K8s)
	if !hasWarning(res.Warnings, "will not load as-is") {
		t.Errorf("warnings = %v, want a will-not-load warning", res.Warnings)
	}
}

func TestConvertUnterminatedArray(t *testing.T) {
	_, err := Convert([]byte("SOLBK_TLS_CERTCAS=(\n  \"/a\"\n"), "fixture", config.K8s)
	if err == nil || !strings.Contains(err.Error(), "unterminated array") {
		t.Fatalf("err = %v, want an unterminated-array error", err)
	}
}

func TestConvertInvalidPlatformSection(t *testing.T) {
	// An unknown platform still converts the shared sections; validation warns.
	res := convertOK(t, ctrEnv, config.Platform("nope"))
	if !hasWarning(res.Warnings, "will not load as-is") {
		t.Errorf("warnings = %v, want the validation warning", res.Warnings)
	}
}

// --- parser ----------------------------------------------------------------

func TestParseAssignmentForms(t *testing.T) {
	src := `#!/bin/bash
# a comment
BARE=value
QUOTED="a value"
SINGLE='single quoted'
EMPTY=""
TRAILING=word   # explanatory comment
export EXPORTED="e"
declare DECLARED="d"
ARRAY=("one" "two" three)
MULTI=(
  "first"
  # a comment inside the array
  "second"
)
declare -A ASSOC=(
	[key1]="v1"
	[key2]="v2"
)
INLINE_ASSOC=([a]="1" [b]="2")
REF="${QUOTED}/suffix"
SHORTREF=$BARE
MISSING="${NOT_SET}x"
notAnAssignment() {
  echo hi
}
`
	v, err := parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	scalars := map[string]string{
		"BARE":     "value",
		"QUOTED":   "a value",
		"SINGLE":   "single quoted",
		"EMPTY":    "",
		"TRAILING": "word",
		"EXPORTED": "e",
		"DECLARED": "d",
		"REF":      "a value/suffix",
		"SHORTREF": "value",
		"MISSING":  "x",
	}
	for k, want := range scalars {
		if got := v.scalar[k]; got != want {
			t.Errorf("%s = %q, want %q", k, got, want)
		}
	}
	if got := v.array["ARRAY"]; len(got) != 3 || got[0] != "one" || got[2] != "three" {
		t.Errorf("ARRAY = %v", got)
	}
	if got := v.array["MULTI"]; len(got) != 2 || got[0] != "first" || got[1] != "second" {
		t.Errorf("MULTI = %v", got)
	}
	if got := v.assoc["ASSOC"]; got["key1"] != "v1" || got["key2"] != "v2" {
		t.Errorf("ASSOC = %v", got)
	}
	// `[k]=v` entries are an associative array even without `declare -A`.
	if got := v.assoc["INLINE_ASSOC"]; got["a"] != "1" || got["b"] != "2" {
		t.Errorf("INLINE_ASSOC = %v", got)
	}
	if _, ok := v.scalar["notAnAssignment"]; ok {
		t.Error("a function definition must not parse as an assignment")
	}
}

func TestParseScalarListFallback(t *testing.T) {
	// A single-entry list written as a scalar still reads as a one-element list.
	v, err := parse("SOLBK_PRODUCTKEYS=\"only-key\"\n")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := v.l("SOLBK_PRODUCTKEYS"); len(got) != 1 || got[0] != "only-key" {
		t.Errorf("l() = %v, want [only-key]", got)
	}
	if got := v.l("SOLBK_TLS_CERTCAS"); got != nil {
		t.Errorf("absent list = %v, want nil", got)
	}
}

func TestParseCRLF(t *testing.T) {
	v, err := parse("A=\"1\"\r\nB=(\r\n  \"x\"\r\n)\r\n")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if v.scalar["A"] != "1" {
		t.Errorf("A = %q", v.scalar["A"])
	}
	if got := v.array["B"]; len(got) != 1 || got[0] != "x" {
		t.Errorf("B = %v", got)
	}
}

func TestParseEscapedQuote(t *testing.T) {
	v, err := parse(`MSG="say \"hi\""` + "\n")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got, want := v.scalar["MSG"], `say "hi"`; got != want {
		t.Errorf("MSG = %q, want %q", got, want)
	}
}

func TestUnmappedTracksFileOrder(t *testing.T) {
	v, err := parse("ZED=\"1\"\nALPHA=\"2\"\nSOLBK_IMAGE=\"i\"\n")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	_ = v.s("SOLBK_IMAGE")
	got := v.unmapped()
	if len(got) != 2 || got[0] != "ZED" || got[1] != "ALPHA" {
		t.Errorf("unmapped = %v, want [ZED ALPHA] in file order", got)
	}
}

// --- emitter ---------------------------------------------------------------

func TestScalarQuoting(t *testing.T) {
	cases := []struct{ in, want string }{
		{"plain", "plain"},
		{"with-dash.and.dot", "with-dash.and.dot"},
		{"", `""`},
		{"yes", `"yes"`},
		{"no", `"no"`},
		{"true", `"true"`},
		{"null", `"null"`},
		{"10.10.1.128", `"10.10.1.128"`},
		{"100", `"100"`},
		{"/path/to/x", `"/path/to/x"`},
		{"a: b", `"a: b"`},
		{"has # hash", `"has # hash"`},
		{"1000:1048576", `"1000:1048576"`},
		{`quote"inside`, `"quote\"inside"`},
		{`back\slash`, `"back\\slash"`},
	}
	for _, tc := range cases {
		if got := scalar(tc.in); got != tc.want {
			t.Errorf("scalar(%q) = %s, want %s", tc.in, got, tc.want)
		}
	}
}

func TestEmptyBlocksOmitted(t *testing.T) {
	res := convertOK(t, "SOLBK_IMAGE=\"r\"\nSOLBK_IMG_TAG=\"t\"\nSOLBK_ADM_PASS=\"p\"\nSOLBK_NODE_PRI_NAME=\"h\"\n", config.Docker)
	out := string(res.YAML)
	for _, absent := range []string{"tls:", "replication:", "loadBalancer:", "placement:", "storage:"} {
		if strings.Contains(out, absent) {
			t.Errorf("empty block %q should be omitted:\n%s", absent, out)
		}
	}
	if !strings.Contains(out, "image:") || !strings.Contains(out, "nodes:") {
		t.Errorf("populated blocks are missing:\n%s", out)
	}
}

func TestGeneratedHeader(t *testing.T) {
	res, err := Convert([]byte("SOLBK_IMAGE=\"r\"\n"), "bash/env/prod", config.K8s)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if !strings.HasPrefix(string(res.YAML), "# Generated by `solace-util convert` from bash/env/prod.") {
		t.Errorf("missing provenance header:\n%s", res.YAML)
	}
}

func hasWarning(warns []string, substr string) bool {
	for _, w := range warns {
		if strings.Contains(w, substr) {
			return true
		}
	}
	return false
}
