// Package config defines the single unified YAML schema for every platform
// (k8s, docker, podman), plus loading, defaulting, and validation. It replaces
// the two bash bootstraps (000-env.sh and docker-podman/000-env.sh): shared
// identity/image/admin/tls/redundancy live at the top level; platform-specific
// knobs live under k8s/docker/podman/nodes.
package config

import "fmt"

// Platform selects which platform's defaults, validation, and renderers apply.
type Platform string

const (
	K8s    Platform = "k8s"
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

	K8s    K8sConfig    `yaml:"k8s"`
	Docker DockerConfig `yaml:"docker"`
	Podman PodmanConfig `yaml:"podman"`
	Nodes  Nodes        `yaml:"nodes"`
}

// RedundancyEnabled reports HA mode (redundancy: yes). Container HA and k8s HA
// both key off this; HA-only steps (leader, redundancy verify) no-op otherwise.
func (c *Config) RedundancyEnabled() bool { return c.Redundancy == "yes" }

// Image is the broker image reference and (k8s) pull-secret material.
type Image struct {
	Repo       string `yaml:"repo"`       // SOLBK_IMAGE
	Tag        string `yaml:"tag"`        // SOLBK_IMG_TAG
	Registry   string `yaml:"registry"`   // IMAGEREPO_HOST (optional prefix)
	PullSecret string `yaml:"pullSecret"` // IMAGEREPO_SECRET (k8s; enables imagePullSecrets)
	User       string `yaml:"user"`       // IMAGEREPO_USER
	Pass       string `yaml:"pass"`       // IMAGEREPO_PASS (secret)
}

// Ref is the fully-qualified image reference, with the optional registry prefix.
func (i Image) Ref() string {
	if i.Registry != "" {
		return fmt.Sprintf("%s/%s:%s", i.Registry, i.Repo, i.Tag)
	}
	return fmt.Sprintf("%s:%s", i.Repo, i.Tag)
}

// Admin holds broker credentials. Passwords are secrets: never logged/echoed.
type Admin struct {
	User          string   `yaml:"user"`          // SOLBK_ADM_USER (container); k8s admin is "admin"
	Pass          string   `yaml:"pass"`          // SOLBK_ADM_PASS (secret, mandatory)
	MonitorPass   string   `yaml:"monitorPass"`   // SOLBK_MON_PASS (k8s, secret)
	UserSecret    string   `yaml:"userSecret"`    // SOLBK_USR_SECRET (k8s secret name)
	UserPasswords []string `yaml:"userPasswords"` // SOLBK_USR_PASS entries "user=password" (secrets)
}

// TLS is the broker server certificate + trusted CAs.
type TLS struct {
	Cert         string   `yaml:"cert"`         // SOLBK_TLS_CERT
	CertKey      string   `yaml:"certKey"`      // SOLBK_TLS_CERTKEY
	CAs          []string `yaml:"cas"`          // SOLBK_TLS_CERTCAS
	ServerSecret string   `yaml:"serverSecret"` // SOLBK_SVR_SECRET (k8s; enables the TLS secret)
}

// Scaling is the superset of broker scaling knobs; each platform reads the
// subset it supports (container uses MaxConnections/MaxQueueMessages/SpoolMaxUsageMB).
type Scaling struct {
	MaxConnections      int `yaml:"maxConnections"`      // system_scaling_maxconnectioncount
	MaxQueueMessages    int `yaml:"maxQueueMessages"`    // system_scaling_maxqueuemessagecount
	MaxSpoolUsageMB     int `yaml:"maxSpoolUsageMB"`     // messagespool_maxspoolusage (container)
	MaxPool             int `yaml:"maxPool"`             // k8s maxSpoolUsage / connection pool
	MaxKafkaBridge      int `yaml:"maxKafkaBridge"`      // k8s
	MaxKafkaConnections int `yaml:"maxKafkaConnections"` // k8s
	MaxBridges          int `yaml:"maxBridges"`          // k8s
	MaxSubscriptions    int `yaml:"maxSubscriptions"`    // k8s
	MaxGuaranteedMsgMB  int `yaml:"maxGuaranteedMsgMB"`  // k8s
}

// Replication holds the data-replication generator inputs (k8s repl tooling).
type Replication struct {
	Mate    string   `yaml:"mate"`    // REPL_MATE
	ConnSSL []string `yaml:"connSsl"` // REPL_CONN_SSL host:port entries
	PSK     string   `yaml:"psk"`     // REPL_PSK (secret; generated if blank)
}

// K8sConfig holds everything specific to the operator-based Kubernetes deployment.
type K8sConfig struct {
	Name              string            `yaml:"name"`              // SOLBK_NAME
	Namespace         string            `yaml:"namespace"`         // SOLBK_NS
	UpdateStrategy    string            `yaml:"updateStrategy"`    // automatedRolling|manualPodRestart
	ServiceAccount    string            `yaml:"serviceAccount"`    // SOLBK_SVC_ACCOUNT (optional)
	CLIScriptsFolder  string            `yaml:"cliScriptsFolder"`  // SOLBK_CLISCRIPTS_FOLDER
	DiagDir           string            `yaml:"diagDir"`           // SOLBK_DIAG_DIR
	Storage           Storage           `yaml:"storage"`
	MsgNode           Resources         `yaml:"msgNode"` // SOLBK_MSGNODE_CPU/MEM
	Operator          Operator          `yaml:"operator"`
	SecurityContext   PodSecurity       `yaml:"securityContext"`   // -> spec.securityContext
	ContainerSecurity ContainerSecurity `yaml:"containerSecurity"` // -> spec.brokerContainerSecurity
	Placement         Placement         `yaml:"placement"`
	LoadBalancer      LoadBalancer      `yaml:"loadBalancer"`
	Ports             []string          `yaml:"ports"`       // SOLBK_PORTS "name=port[/proto]"
	ProductKeys       []string          `yaml:"productKeys"` // SOLBK_PRODUCTKEYS
	DomainCerts       DomainCerts       `yaml:"domainCerts"`
}

// Storage is the k8s PVC sizing / storage class.
type Storage struct {
	Class   string `yaml:"class"`   // SOLBK_STORAGECLASS
	MsgNode string `yaml:"msgNode"` // SOLBK_STORAGE_MSGNODE (mandatory)
	MonNode string `yaml:"monNode"` // SOLBK_STORAGE_MONNODE
}

// Resources is a cpu/memory pair.
type Resources struct {
	CPU string `yaml:"cpu"`
	Mem string `yaml:"mem"`
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
	Runtime     string    `yaml:"runtime"`     // CONTAINER_RUNTIME override (default: docker)
	Mode        string    `yaml:"mode"`        // compose|run
	ComposeFile string    `yaml:"composeFile"` // DOCKER_COMPOSE_FILE
	Network     Network   `yaml:"network"`
	Container   Container `yaml:"container"`
}

// PodmanConfig holds podman-only deployment options plus the shared container block.
type PodmanConfig struct {
	Runtime   string    `yaml:"runtime"`   // CONTAINER_RUNTIME override (default: podman)
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
	Name    string  `yaml:"name"`    // CONTAINER_NAME
	RunUser string  `yaml:"runUser"` // SOLBK_RUN_USER uid:gid
	ShmSize string  `yaml:"shmSize"` // SOLBK_SHM_SIZE
	DataDir string  `yaml:"dataDir"` // SOLBK_DATA_DIR (host bind mount)
	Ulimits Ulimits `yaml:"ulimits"`
}

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
}

// Node is one redundancy-group member.
type Node struct {
	Name string `yaml:"name"`
	IP   string `yaml:"ip"`
}
