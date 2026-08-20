// Package config defines the single unified YAML schema for every platform
// (kubernetes, docker, podman), plus loading, defaulting, and validation. It
// replaces the two bash bootstraps (000-env.sh and docker-podman/000-env.sh):
// shared identity/image/admin/tls/redundancy live at the top level;
// platform-specific knobs live under kubernetes/docker/podman/nodes.
package config

import (
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Platform selects which platform's defaults, validation, and renderers apply.
type Platform string

const (
	K8s    Platform = "kubernetes"
	Docker Platform = "docker"
	Podman Platform = "podman"
)

// IsContainer reports whether p is a host-container platform (docker or podman),
// which share the container config, host-prep, and env-pair generation.
func (p Platform) IsContainer() bool { return p == Docker || p == Podman }

// Config is the whole deserialized env file. Fields shared by all platforms sit
// at the top; platform sections hold the rest. Unknown-to-a-platform fields are
// simply ignored by that platform's renderer.
type Config struct {
	// Redundancy is unified across platforms as "yes"|"no". The k8s renderer
	// emits it as a bool into the CR; the container renderer emits the
	// redundancy_*/configsync_* key=value set (or just redundancy_enable=no).
	Redundancy string `yaml:"redundancy"`

	// Timezone applies to every platform -- the k8s CR's timezone field and the
	// containers' TZ setting are the same knob. Unset on purpose: an omitted
	// value leaves the broker on the image default rather than pinning a region.
	Timezone string `yaml:"timezone"`

	Image       Image       `yaml:"image"`
	Admin       Admin       `yaml:"admin"`
	TLS         TLS         `yaml:"tls"`
	Scaling     Scaling     `yaml:"scaling"`
	Replication Replication `yaml:"replication"`
	Broker      Broker      `yaml:"broker"`

	K8s    K8sConfig    `yaml:"kubernetes"`
	Docker DockerConfig `yaml:"docker"`
	Podman PodmanConfig `yaml:"podman"`
	Nodes  Nodes        `yaml:"nodes"`

	// extraAllowed holds the binaries the operator approved for this invocation
	// with --allow-command. Unexported on purpose: yaml.v3 cannot decode into an
	// unexported field, so no env file can widen its own allowlist -- that
	// authority stays with the person at the keyboard (see AllowCommands).
	extraAllowed map[string]bool
}

// RedundancyEnabled reports HA mode (redundancy: yes). Container HA and k8s HA
// both key off this; HA-only steps (leader, redundancy verify) no-op otherwise.
func (c *Config) RedundancyEnabled() bool { return c.Redundancy == "yes" }

// Command is an external command line: argv[0] plus any leading arguments that
// precede every call's own. It exists so the platform CLI can be more than a
// binary name -- `oc`, `microk8s kubectl`, or a profile like
// `kubectl --kubeconfig /path/.kubeconfig-cluster` (bash/env/customer-sample:7).
//
// The bash bootstraps got this for free by expanding ${KUBE} and
// ${CONTAINER_RUNTIME} unquoted, which word-splits. Go's exec never splits and
// never involves a shell, so the split happens here instead.
type Command []string

// UnmarshalYAML accepts either a scalar, split on whitespace exactly as the
// bash bootstraps' unquoted expansion did, or an explicit sequence, which is the
// only way to express a token that itself contains a space (a binary path under
// "C:\Program Files\..."). Whitespace splitting deliberately does not honour
// embedded quotes -- neither did bash's word splitting.
func (c *Command) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		var s string
		if err := value.Decode(&s); err != nil {
			return err
		}
		*c = strings.Fields(s)
	case yaml.SequenceNode:
		var parts []string
		if err := value.Decode(&parts); err != nil {
			return err
		}
		*c = parts
	default:
		return fmt.Errorf("line %d: a command must be a string (split on whitespace) "+
			"or a list of exact arguments", value.Line)
	}
	return nil
}

// Name is argv[0] -- the executable actually run.
func (c Command) Name() string {
	if len(c) == 0 {
		return ""
	}
	return c[0]
}

// Args prepends the configured leading arguments to a single call's own args. It
// always builds a new slice, so a caller's backing array is never aliased or
// appended into by a later call.
func (c Command) Args(extra ...string) []string {
	if len(c) <= 1 {
		return extra
	}
	out := make([]string, 0, len(c)-1+len(extra))
	out = append(out, c[1:]...)
	return append(out, extra...)
}

// String renders the command for reports and error messages.
func (c Command) String() string { return strings.Join(c, " ") }

// Image is the broker image reference and (k8s) pull-secret material.
type Image struct {
	Repo       string `yaml:"repo"`       // SOLBK_IMAGE
	Tag        string `yaml:"tag"`        // SOLBK_IMG_TAG
	Registry   string `yaml:"registry"`   // IMAGEREPO_HOST (optional prefix)
	PullSecret string `yaml:"pullSecret"` // IMAGEREPO_SECRET (k8s; enables imagePullSecrets)
	User       string `yaml:"user"`       // IMAGEREPO_USER
	Pass       string `yaml:"pass"`       // IMAGEREPO_PASS (secret)
	PassEnv    string `yaml:"passEnv"`    // env var holding pass instead
	// PullPolicy is the k8s image pull policy: Always for a moving tag, Never for
	// an air-gapped cluster with the image preloaded. Empty keeps the CR's own
	// IfNotPresent, so an unset value renders exactly as before.
	PullPolicy string `yaml:"pullPolicy"`
}

// Ref is the fully-qualified image reference, with the optional registry prefix.
func (i Image) Ref() string {
	if i.Registry != "" {
		return fmt.Sprintf("%s/%s:%s", i.Registry, i.Repo, i.Tag)
	}
	return fmt.Sprintf("%s:%s", i.Repo, i.Tag)
}

// TagVersion parses the leading major.minor of the image tag ("10.26.1.5" -> 10,
// 26). ok is false when the tag carries no version at all -- "latest", a digest,
// a bare codename -- and callers must treat that as *unknown*, never as old or
// new: a feature gate that guessed either way would be wrong half the time.
func (i Image) TagVersion() (major, minor int, ok bool) {
	parts := strings.Split(strings.TrimSpace(i.Tag), ".")
	if len(parts) < 2 {
		return 0, 0, false
	}
	major, ok = atoiPrefix(parts[0])
	if !ok {
		return 0, 0, false
	}
	minor, ok = atoiPrefix(parts[1])
	if !ok {
		return 0, 0, false
	}
	return major, minor, true
}

// AtLeast reports whether the image tag names a broker release at or above
// major.minor. known is false when the tag carries no version to compare.
func (i Image) AtLeast(major, minor int) (ok, known bool) {
	haveMajor, haveMinor, known := i.TagVersion()
	if !known {
		return false, false
	}
	if haveMajor != major {
		return haveMajor > major, true
	}
	return haveMinor >= minor, true
}

// atoiPrefix parses the digits at the start of s, so a tag component like "0-rc1"
// still yields 0. It fails when there are no leading digits at all.
func atoiPrefix(s string) (int, bool) {
	end := 0
	for end < len(s) && s[end] >= '0' && s[end] <= '9' {
		end++
	}
	if end == 0 {
		return 0, false
	}
	n, err := strconv.Atoi(s[:end])
	if err != nil {
		return 0, false
	}
	return n, true
}

// Admin holds broker credentials. Passwords are secrets: never logged/echoed.
// Every secret field has a sibling *Env key naming an environment variable to
// read the value from instead, so an env file can be reviewed and shared without
// carrying a single secret (resolveSecretRefs).
type Admin struct {
	User            string           `yaml:"user"`            // SOLBK_ADM_USER (container); k8s admin is "admin"
	Pass            string           `yaml:"pass"`            // SOLBK_ADM_PASS (secret, mandatory)
	PassEnv         string           `yaml:"passEnv"`         // env var holding pass instead
	MonitorPass     string           `yaml:"monitorPass"`     // SOLBK_MON_PASS (k8s, secret)
	MonitorPassEnv  string           `yaml:"monitorPassEnv"`  // env var holding monitorPass instead
	AdditionalUsers []AdditionalUser `yaml:"additionalUsers"` // extra CLI users beyond admin/monitor
}

// AdditionalUser is one extra broker CLI user. It replaces the old
// admin.userPasswords "user=password" list: the access level was not expressible
// there, and a password is now referable through the environment like every other
// secret. Exactly one of Password/PasswordEnv must be set.
type AdditionalUser struct {
	Username    string `yaml:"username"`    // becomes username_<username>_password
	AccessLevel string `yaml:"accessLevel"` // admin|read-write|read-only|none
	Password    string `yaml:"password"`    // secret
	PasswordEnv string `yaml:"passwordEnv"` // env var holding password instead
}

// TLS is the broker server certificate + trusted CAs.
type TLS struct {
	Cert    string   `yaml:"cert"`    // SOLBK_TLS_CERT
	CertKey string   `yaml:"certKey"` // SOLBK_TLS_CERTKEY
	CAs     []string `yaml:"cas"`     // SOLBK_TLS_CERTCAS
	// CertPassphrase unlocks an encrypted server-certificate key. It is a secret,
	// so on containers it is externalized like the admin password rather than
	// written into the deploy artifact. Empty means the key is not encrypted.
	CertPassphrase    string `yaml:"certPassphrase"`
	CertPassphraseEnv string `yaml:"certPassphraseEnv"` // env var holding certPassphrase instead
	ServerSecret      string `yaml:"serverSecret"`      // SOLBK_SVR_SECRET (k8s; enables the TLS secret)
}

// Scaling is the broker's sizing, and every knob applies to every platform. They
// differ only in delivery: k8s writes them into the CR's spec.systemScaling,
// while docker and podman pass them to the container as environment variables
// under the same broker setting names (render.EnvPairs).
type Scaling struct {
	MaxConnections      int `yaml:"maxConnections"`      // system_scaling_maxconnectioncount
	MaxQueueMessages    int `yaml:"maxQueueMessages"`    // system_scaling_maxqueuemessagecount
	MaxSpoolUsageMB     int `yaml:"maxSpoolUsageMB"`     // messagespool_maxspoolusage / CR maxSpoolUsage
	MaxKafkaBridge      int `yaml:"maxKafkaBridge"`      // system_scaling_maxkafkabridgecount
	MaxKafkaConnections int `yaml:"maxKafkaConnections"` // system_scaling_maxkafkabrokerconnectioncount
	MaxBridges          int `yaml:"maxBridges"`          // system_scaling_maxbridgecount
	MaxSubscriptions    int `yaml:"maxSubscriptions"`    // system_scaling_maxsubscriptioncount
	MaxGuaranteedMsgMB  int `yaml:"maxGuaranteedMsgMB"`  // system_scaling_maxguaranteedmessagesize

	// MaxPool is retained so an env file carrying the removed maxPool fails with
	// an actionable error instead of a bare unknown-field decode error. It named
	// the same broker setting as MaxSpoolUsageMB -- one concept under two keys,
	// one per platform -- which is exactly what this block no longer has.
	MaxPool int `yaml:"maxPool"`

	// CPU is the broker CPU the MaxConnections tier fixes, derived in
	// ApplyDefaults once MaxConnections resolves (scaling.go). Not read from
	// YAML, so no env file can set it: k8s renders it as messagingNodeCpu,
	// docker and podman as their own CPU cap. It replaces the independently
	// settable kubernetes.msgNode.cpu, which could contradict the tier.
	CPU string `yaml:"-"`
}

// Replication holds the data-replication generator inputs (k8s repl tooling).
type Replication struct {
	Mate    string   `yaml:"mate"`    // REPL_MATE
	ConnSSL []string `yaml:"connSsl"` // REPL_CONN_SSL host:port entries
	PSK     string   `yaml:"psk"`     // REPL_PSK (secret; generated if blank)
}

// Broker holds the post-deployment broker configuration every platform applies
// over the broker CLI, plus the local folders those operations read and write.
// It is platform-neutral on purpose: the container ops apply exactly the same
// domain certificates, product keys, and .cli scripts as the kubernetes ops do,
// and while these lived under kubernetes.* a container env file had to carry a
// kubernetes: section to reach them -- which made "which platform is this file
// for?" unanswerable from the file itself (DetectPlatforms).
type Broker struct {
	CLIScriptsFolder string      `yaml:"cliScriptsFolder"` // SOLBK_CLISCRIPTS_FOLDER
	DiagDir          string      `yaml:"diagDir"`          // SOLBK_DIAG_DIR
	ProductKeys      []string    `yaml:"productKeys"`      // SOLBK_PRODUCTKEYS
	DomainCerts      DomainCerts `yaml:"domainCerts"`
}

// K8sConfig holds everything specific to the operator-based Kubernetes deployment.
type K8sConfig struct {
	Runtime           Command           `yaml:"runtime"`           // KUBE (default: kubectl)
	Name              string            `yaml:"name"`              // SOLBK_NAME
	Namespace         string            `yaml:"namespace"`         // SOLBK_NS
	AdminSecret       string            `yaml:"adminSecret"`       // SOLBK_USR_SECRET: Secret holding the admin/monitor creds
	UpdateStrategy    string            `yaml:"updateStrategy"`    // automatedRolling|manualPodRestart
	ServiceAccount    string            `yaml:"serviceAccount"`    // SOLBK_SVC_ACCOUNT (optional)
	Storage           Storage           `yaml:"storage"`
	MsgNode           Resources         `yaml:"msgNode"` // SOLBK_MSGNODE_CPU/MEM
	Operator          Operator          `yaml:"operator"`
	SecurityContext   PodSecurity       `yaml:"securityContext"`   // -> spec.securityContext
	ContainerSecurity ContainerSecurity `yaml:"containerSecurity"` // -> spec.brokerContainerSecurity
	PodAnnotations    map[string]string `yaml:"podAnnotations"`    // -> spec.podAnnotations
	PodLabels         map[string]string `yaml:"podLabels"`         // -> spec.podLabels
	Placement         Placement         `yaml:"placement"`
	LoadBalancer      LoadBalancer      `yaml:"loadBalancer"`
	Ports             []string          `yaml:"ports"` // SOLBK_PORTS "name=port[/proto]"
}

// Storage is the k8s PVC sizing / storage class.
type Storage struct {
	Class   string `yaml:"class"`   // SOLBK_STORAGECLASS
	MsgNode string `yaml:"msgNode"` // SOLBK_STORAGE_MSGNODE (mandatory)
	MonNode string `yaml:"monNode"` // SOLBK_STORAGE_MONNODE
}

// Resources is the message-node resource block.
type Resources struct {
	// CPU is retained so an env file carrying the removed kubernetes.msgNode.cpu fails
	// with an actionable error instead of a bare unknown-field decode error. It
	// is never defaulted and never rendered: broker CPU is fixed by the scaling
	// tier (scaling.go, Scaling.CPU). validateK8s rejects any value here.
	CPU string `yaml:"cpu"`
	Mem string `yaml:"mem"` // SOLBK_MSGNODE_MEM (defaults to the scaling tier's memory)
}

// Operator is the cluster-scoped EventBroker Operator configuration.
type Operator struct {
	Image           string `yaml:"image"`           // SOLOP_IMAGE
	Namespace       string `yaml:"namespace"`       // SOLOP_NS (blank -> derive at runtime)
	WatchNamespaces string `yaml:"watchNamespaces"` // SOLOP_WATCH_NS
	WatchBrokerNS   *bool  `yaml:"watchBrokerNs"`   // SOLOP_WATCH_SOLBK_NS (nil -> default true)
	CPU             string `yaml:"cpu"`             // SOLOP_CPU
	Mem             string `yaml:"mem"`             // SOLOP_MEM
}

// WatchBrokerNSEnabled reports whether the operator should also watch the
// broker namespace. Unset (nil) defaults to true, matching the bash default
// SOLOP_WATCH_SOLBK_NS=true; set it to false in YAML to opt out.
func (o Operator) WatchBrokerNSEnabled() bool {
	return o.WatchBrokerNS == nil || *o.WatchBrokerNS
}

// PodSecurity is the broker pod's securityContext. The ids are strings, not
// ints, so an explicit "0" (which asks OpenShift to auto-assign) stays
// distinguishable from an unset field -- the whole block is optional and is
// omitted from the CR when nothing is set.
type PodSecurity struct {
	RunAsUser string `yaml:"runAsUser"`
	FSGroup   string `yaml:"fsGroup"`
}

// Configured reports whether any field was set, which is what decides if the
// block reaches the CR at all.
func (s PodSecurity) Configured() bool { return s.RunAsUser != "" || s.FSGroup != "" }

// ContainerSecurity is the broker container's own security settings. Same
// optional-block rule as PodSecurity; ReadOnlyRootFilesystem is a pointer so an
// explicit false is not mistaken for "not configured".
type ContainerSecurity struct {
	RunAsUser              string `yaml:"runAsUser"`
	RunAsGroup             string `yaml:"runAsGroup"`
	ReadOnlyRootFilesystem *bool  `yaml:"readOnlyRootFilesystem"`
}

// Configured reports whether any field was set.
func (s ContainerSecurity) Configured() bool {
	return s.RunAsUser != "" || s.RunAsGroup != "" || s.ReadOnlyRootFilesystem != nil
}

// Placement controls broker pod scheduling (tolerations, node labels, anti-affinity).
type Placement struct {
	TolerationsPrimary []string `yaml:"tolerationsPrimary"` // SOLBK_NODETOL_PRI
	TolerationsBackup  []string `yaml:"tolerationsBackup"`  // SOLBK_NODETOL_BKP
	TolerationsMonitor []string `yaml:"tolerationsMonitor"` // SOLBK_NODETOL_MON
	LabelsPrimary      []string `yaml:"labelsPrimary"`      // SOLBK_NODELABEL_PRI
	LabelsBackup       []string `yaml:"labelsBackup"`       // SOLBK_NODELABEL_BKP
	LabelsMonitor      []string `yaml:"labelsMonitor"`      // SOLBK_NODELABEL_MON
	AntiAffinityNS     []string `yaml:"antiAffinityNamespaces"` // SOLBK_ANTIAFFINITY_NS
	AntiAffinityWeight int      `yaml:"antiAffinityWeight"`     // SOLBK_ANTIAFFINITY_WT

	// The blocks below are additive: unset, the rendered CR is exactly what
	// AntiAffinityNS/Weight and the label/toleration lists above produce, so an
	// existing env file is unaffected. They apply to every broker role -- per-role
	// affinity is not modelled, matching how AntiAffinityNS already behaves.
	NodeAffinity    NodeAffinity      `yaml:"nodeAffinity"`
	PodAffinity     []PodAffinityTerm `yaml:"podAffinity"`
	PodAntiAffinity []PodAffinityTerm `yaml:"podAntiAffinity"`
}

// NodeAffinity mirrors the Kubernetes nodeAffinity subset the operator passes
// through, rather than inventing a parallel spelling. Deliberately not modelled:
// matchFields, and OR-ed nodeSelectorTerms -- Required is one ANDed term.
type NodeAffinity struct {
	Preferred []WeightedNodeTerm `yaml:"preferred"`
	Required  []NodeMatchExpr    `yaml:"required"`
}

// Configured reports whether either list was set, which is what decides if a
// nodeAffinity block reaches the CR at all.
func (n NodeAffinity) Configured() bool { return len(n.Preferred) > 0 || len(n.Required) > 0 }

// WeightedNodeTerm is one weighted preference: every expression in Match must hold
// for the weight to apply.
type WeightedNodeTerm struct {
	Weight int             `yaml:"weight"` // 1-100
	Match  []NodeMatchExpr `yaml:"match"`
}

// NodeMatchExpr is a node-label match expression.
type NodeMatchExpr struct {
	Key      string   `yaml:"key"`
	Operator string   `yaml:"operator"` // In|NotIn|Exists|DoesNotExist|Gt|Lt
	Values   []string `yaml:"values"`   // required by In/NotIn/Gt/Lt
}

// PodAffinityTerm is one pod (anti-)affinity rule. Weight 0 makes it a required
// term; 1-100 makes it a preference with that weight. The pod selector is
// matchLabels only -- matchExpressions there is not modelled.
type PodAffinityTerm struct {
	Weight      int               `yaml:"weight"`
	TopologyKey string            `yaml:"topologyKey"`
	MatchLabels map[string]string `yaml:"matchLabels"`
	Namespaces  []string          `yaml:"namespaces"`
}

// LoadBalancer holds MetalLB / service-LB options.
type LoadBalancer struct {
	IP          string   `yaml:"ip"`          // SOLBK_LOADBALANCER_IP
	Annotations []string `yaml:"annotations"` // SOLBK_LOADBALANCER_ANOTN "key: value"
	IPPool      string   `yaml:"ipPool"`      // SOLBK_IPPOOL
}

// DomainCerts are trusted domain CA certificates loaded post-deploy.
type DomainCerts struct {
	Folder string            `yaml:"folder"` // SOLBK_DOMAINCERT_FOLDER
	Files  map[string]string `yaml:"files"`  // SOLBK_DOMAINCERT_FILES [CA-NAME]=filename
}

// DockerConfig holds docker-only deployment options plus the shared container block.
type DockerConfig struct {
	Runtime Command `yaml:"runtime"` // CONTAINER_RUNTIME override (default: docker)
	// Compose is the compose invocation, whose form differs per host: the modern
	// plugin is a runtime subcommand (`docker compose`), the standalone v1 binary
	// is its own executable (`docker-compose`). Unset defaults to the configured
	// runtime plus `compose`; this tool appends every argument after it.
	Compose     Command   `yaml:"compose"`
	ComposeFile string    `yaml:"composeFile"` // DOCKER_COMPOSE_FILE
	Network     Network   `yaml:"network"`
	Container   Container `yaml:"container"`
}

// PodmanConfig holds podman-only deployment options plus the shared container block.
type PodmanConfig struct {
	Runtime   Command   `yaml:"runtime"`   // CONTAINER_RUNTIME override (default: podman)
	Rootless  bool      `yaml:"rootless"`  // PODMAN_ROOTLESS
	QuadletDir string   `yaml:"quadletDir"` // QUADLET_DIR override
	Network   Network   `yaml:"network"`
	Container Container `yaml:"container"`

	// Derived from Rootless in ApplyDefaults (not read from YAML).
	SystemctlUser string `yaml:"-"`
	WantedBy      string `yaml:"-"`
}

// Network is the container networking mode + published ports.
type Network struct {
	Mode  string   `yaml:"mode"`  // host|bridge (SOLBK_NETWORK_MODE)
	Ports []string `yaml:"ports"` // SOLBK_PORTS host:container (required for bridge)
}

// Container is the shared docker/podman container runtime settings.
type Container struct {
	Name    string `yaml:"name"`    // CONTAINER_NAME
	RunUser string `yaml:"runUser"` // SOLBK_RUN_USER uid:gid
	ShmSize string `yaml:"shmSize"` // SOLBK_SHM_SIZE
	// Mem is the container memory limit in docker's and podman's own b|k|m|g
	// suffix, NOT the Mi/Gi Kubernetes quantity kubernetes.msgNode.mem takes -- the
	// engines reject that spelling, so validateContainer catches it here rather
	// than letting compose fail at deploy. Defaults to the scaling tier's memory
	// (scaling.go). CPU has no counterpart: it is fixed by the tier, so there is
	// nothing here to override.
	Mem         string      `yaml:"mem"`
	DataDir     string      `yaml:"dataDir"` // SOLBK_DATA_DIR (host bind mount)
	Ulimits     Ulimits     `yaml:"ulimits"`
	HealthCheck HealthCheck `yaml:"healthCheck"`
}

// HealthCheck is the container engine's own probe against the broker, which
// upgrades `docker ps`/`compose ps` and podman's auto-restart from "the process is
// up" to "the broker is ready". It is opt-in; leaving the whole block out renders
// the artifacts unchanged.
//
// Enabled with no Cmd uses the built-in readiness probe (the broker's own
// /health-check/readiness endpoint). That endpoint only exists from
// HealthCheckMinMajor.HealthCheckMinMinor onward, so Validate refuses to enable it
// against an older -- or an unidentifiable -- image tag: an always-failing probe
// would mark the container permanently unhealthy, which under podman's
// auto-restart becomes a restart loop. Setting Cmd explicitly is the escape hatch
// and skips the version gate, since the probe is then the operator's own choice.
type HealthCheck struct {
	Enabled bool `yaml:"enabled"`
	// Cmd is the probe argv, run inside the container. Empty means the built-in
	// readiness probe. The quadlet form is a command line rather than an argv, so a
	// token containing a space is not representable there -- wrap such a probe in a
	// script instead.
	Cmd         []string `yaml:"cmd"`
	Interval    string   `yaml:"interval"`
	Timeout     string   `yaml:"timeout"`
	Retries     int      `yaml:"retries"`
	StartPeriod string   `yaml:"startPeriod"`
}

// The first broker release exposing /health-check/readiness, which the built-in
// probe depends on.
const (
	HealthCheckMinMajor = 10
	HealthCheckMinMinor = 26
)

// Ulimits are the container resource limits (soft:hard where applicable).
type Ulimits struct {
	NoFile  string `yaml:"nofile"`  // SOLBK_ULIMIT_NOFILE
	MemLock string `yaml:"memlock"` // SOLBK_ULIMIT_MEMLOCK
	Core    string `yaml:"core"`    // SOLBK_ULIMIT_CORE
}

// Nodes is the redundancy-group node table (identical on all container hosts).
type Nodes struct {
	Primary Node   `yaml:"primary"` // message_routing (also the standalone broker)
	Backup  Node   `yaml:"backup"`  // message_routing (redundancy only)
	Monitor Node   `yaml:"monitor"` // monitoring (redundancy only)
	PSK     string `yaml:"psk"`     // SOLBK_REDUNDANCY_PSK (secret; generated by host-prep)
	PSKEnv  string `yaml:"pskEnv"`  // env var holding psk instead (host-prep then generates nothing)
}

// Node is one redundancy-group member.
type Node struct {
	Name string `yaml:"name"`
	IP   string `yaml:"ip"`
}
