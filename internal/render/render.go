// Package render turns a validated config.Config into the deployment artifacts
// each platform applies:
//
//   - BrokerCR         -- the Kubernetes PubSubPlusEventBroker custom resource
//   - EnvPairs -- the ordered Solace key=value config both container engines share
//   - Quadlet          -- a podman systemd .container unit
//   - Compose          -- a docker compose file
//   - ContainerSecrets -- the secret values both engines externalize, and
//     SecretScript, the shell commands that create them
//
// It is the Go port of the bash generators: gen_yaml (020-deploy-broker.sh),
// gen_env_pairs (docker-podman/000-env.sh), gen_quadlet (podman-020) and
// gen_compose (docker-020).
//
// No deployment artifact here ever carries a secret value: every secret is
// externalized (podman's secret store, docker's environment-sourced compose
// secrets) and referenced by name, so a rendered artifact is safe to print, diff,
// and commit. SecretScript is the one function that emits secret values, and only
// a `--gen-secrets-only` run calls it.
//
// Both engines deliver secrets to the broker the same way k8s does: as files the
// broker reads through the setting's *filepath variant, mounted under
// /run/secrets under the name of the setting they feed.
//
// Rendering is done with plain string builders rather than text/template: the
// broker CR is deeply conditional YAML where template whitespace control is
// fragile, and every function here is a pure function of the config (no I/O, no
// exec), so the output is stable and golden-tested in render_test.go.
package render

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"solace/internal/config"
)

// In-container paths the broker image expects.
const (
	dataMount = "/var/lib/solace"
	certMount = "/run/secrets/tls.crt"
	// secretMount is where both engines expose a file-backed secret.
	secretMount = "/run/secrets"
)

// BrokerCR renders the PubSubPlusEventBroker manifest for the k8s platform,
// porting gen_yaml. Redundancy is emitted as the CR's bool (yes -> true).
func BrokerCR(c *config.Config) []byte {
	var b strings.Builder

	repo := c.Image.Repo
	if c.Image.Registry != "" {
		repo = c.Image.Registry + "/" + repo
	}

	fmt.Fprint(&b, "apiVersion: pubsubplus.solace.com/v1beta1\n")
	fmt.Fprint(&b, "kind: PubSubPlusEventBroker\n")
	fmt.Fprint(&b, "metadata:\n")
	fmt.Fprintf(&b, "  namespace: %s\n", c.K8s.Namespace)
	fmt.Fprintf(&b, "  name: %s\n", c.K8s.Name)
	fmt.Fprint(&b, "spec:\n")
	fmt.Fprint(&b, "  image:\n")
	fmt.Fprintf(&b, "    repository: %s\n", repo)
	fmt.Fprintf(&b, "    tag: %s\n", c.Image.Tag)
	// A closed, already-validated enum, so it is emitted bare like updateStrategy.
	// Unset keeps the value this renderer has always written.
	pullPolicy := c.Image.PullPolicy
	if pullPolicy == "" {
		pullPolicy = "IfNotPresent"
	}
	fmt.Fprintf(&b, "    pullPolicy: %s\n", pullPolicy)
	if c.Image.PullSecret != "" {
		fmt.Fprint(&b, "    pullSecrets:\n")
		fmt.Fprintf(&b, "    - name: %s\n", c.Image.PullSecret)
	}
	if c.K8s.ServiceAccount != "" {
		fmt.Fprint(&b, "  serviceAccount:\n")
		fmt.Fprintf(&b, "    name: %s\n", c.K8s.ServiceAccount)
	}
	fmt.Fprintf(&b, "  adminCredentialsSecret: %s\n", c.K8s.AdminSecret)
	fmt.Fprintf(&b, "  monitoringCredentialsSecret: %s\n", c.K8s.AdminSecret)
	fmt.Fprintf(&b, "  redundancy: %s\n", boolStr(c.RedundancyEnabled()))
	fmt.Fprintf(&b, "  updateStrategy: %s\n", c.K8s.UpdateStrategy)
	fmt.Fprint(&b, "  podDisruptionBudgetForHA: true\n")

	s := c.Scaling
	fmt.Fprint(&b, "  systemScaling:\n")
	fmt.Fprintf(&b, "    system_scaling_maxconnectioncount: %d\n", s.MaxConnections)
	fmt.Fprintf(&b, "    system_scaling_maxqueuemessagecount: %d\n", s.MaxQueueMessages)
	fmt.Fprintf(&b, "    system_scaling_maxkafkabridgecount: %d\n", s.MaxKafkaBridge)
	fmt.Fprintf(&b, "    system_scaling_maxkafkabrokerconnectioncount: %d\n", s.MaxKafkaConnections)
	fmt.Fprintf(&b, "    system_scaling_maxbridgecount: %d\n", s.MaxBridges)
	fmt.Fprintf(&b, "    system_scaling_maxsubscriptioncount: %d\n", s.MaxSubscriptions)
	fmt.Fprintf(&b, "    system_scaling_maxguaranteedmessagesize: %d\n", s.MaxGuaranteedMsgMB)
	fmt.Fprintf(&b, "    maxSpoolUsage: %d\n", s.MaxSpoolUsageMB)
	// CPU is derived from the tier above rather than read from the env file
	// (config/scaling.go). It is empty only for a Config built in code that never
	// ran ApplyDefaults; emitting nothing then leaves the operator's own default
	// in place, which beats writing an empty quantity the CRD would reject.
	if s.CPU != "" {
		fmt.Fprintf(&b, "    messagingNodeCpu: %q\n", s.CPU)
	}
	fmt.Fprintf(&b, "    messagingNodeMemory: %s\n", c.K8s.MsgNode.Mem)

	fmt.Fprint(&b, "  storage:\n")
	if c.K8s.Storage.Class != "" {
		fmt.Fprintf(&b, "    useStorageClass: %s\n", c.K8s.Storage.Class)
	}
	fmt.Fprintf(&b, "    messagingNodeStorageSize: %s\n", c.K8s.Storage.MsgNode)
	fmt.Fprintf(&b, "    monitorNodeStorageSize: %s\n", c.K8s.Storage.MonNode)
	// The bash gen_yaml hardcoded a timezone; here it is config, and an unset
	// one leaves the broker on the image default rather than pinning a region.
	if c.Timezone != "" {
		fmt.Fprintf(&b, "  timezone: %q\n", c.Timezone)
	}

	if c.TLS.ServerSecret != "" {
		fmt.Fprint(&b, "  tls:\n")
		fmt.Fprintf(&b, "    serverTlsConfigSecret: %s\n", c.TLS.ServerSecret)
		fmt.Fprint(&b, "    enabled: true\n")
		fmt.Fprint(&b, "    certFilename: tls.crt\n")
		fmt.Fprint(&b, "    certKeyFilename: tls.key\n")
	}

	writeSecurity(&b, c.K8s)
	writeStringMap(&b, "  podAnnotations:\n", c.K8s.PodAnnotations)
	writeStringMap(&b, "  podLabels:\n", c.K8s.PodLabels)
	writeNodeAssignment(&b, c)

	fmt.Fprint(&b, "  service:\n")
	fmt.Fprint(&b, "    type: LoadBalancer\n")
	writeLBAnnotations(&b, c.K8s.LoadBalancer)
	fmt.Fprint(&b, "    ports:\n")
	for _, entry := range c.K8s.Ports {
		p := parsePort(entry)
		fmt.Fprintf(&b, "    - containerPort: %s\n", p.container)
		fmt.Fprintf(&b, "      name: %s\n", p.name)
		fmt.Fprintf(&b, "      protocol: %s\n", p.proto)
		fmt.Fprintf(&b, "      servicePort: %s\n", p.service)
	}

	return []byte(b.String())
}

// writeSecurity emits the pod and container security blocks, each only when
// something was configured. Both are optional: the operator applies the image's
// own defaults when they are absent, so an unset block must stay out of the CR
// rather than be rendered with zeros.
func writeSecurity(b *strings.Builder, k config.K8sConfig) {
	if sc := k.SecurityContext; sc.Configured() {
		fmt.Fprint(b, "  securityContext:\n")
		if sc.RunAsUser != "" {
			fmt.Fprintf(b, "    runAsUser: %s\n", sc.RunAsUser)
		}
		if sc.FSGroup != "" {
			fmt.Fprintf(b, "    fsGroup: %s\n", sc.FSGroup)
		}
	}
	if cs := k.ContainerSecurity; cs.Configured() {
		fmt.Fprint(b, "  brokerContainerSecurity:\n")
		if cs.RunAsUser != "" {
			fmt.Fprintf(b, "    runAsUser: %s\n", cs.RunAsUser)
		}
		if cs.RunAsGroup != "" {
			fmt.Fprintf(b, "    runAsGroup: %s\n", cs.RunAsGroup)
		}
		if cs.ReadOnlyRootFilesystem != nil {
			fmt.Fprintf(b, "    readOnlyRootFilesystem: %s\n", boolStr(*cs.ReadOnlyRootFilesystem))
		}
	}
}

// writeNodeAssignment emits the nodeAssignment block when any placement knob is
// set (tolerations, node labels, or anti-affinity namespaces). Backup/Monitor
// nodes are only emitted in HA. Mirrors gen_yaml's nodeAssignment section.
func writeNodeAssignment(b *strings.Builder, c *config.Config) {
	pl := c.K8s.Placement
	hasTol := len(pl.TolerationsPrimary)+len(pl.TolerationsBackup)+len(pl.TolerationsMonitor) > 0
	hasLabels := len(pl.LabelsPrimary)+len(pl.LabelsBackup)+len(pl.LabelsMonitor) > 0
	hasAnti := len(pl.AntiAffinityNS) > 0
	hasAffinity := pl.NodeAffinity.Configured() || len(pl.PodAffinity) > 0 || len(pl.PodAntiAffinity) > 0
	if !hasTol && !hasLabels && !hasAnti && !hasAffinity {
		return
	}

	fmt.Fprint(b, "  nodeAssignment:\n")
	writeNode(b, "Primary", pl.TolerationsPrimary, pl.LabelsPrimary, pl, hasAnti)
	if c.RedundancyEnabled() {
		writeNode(b, "Backup", pl.TolerationsBackup, pl.LabelsBackup, pl, hasAnti)
		writeNode(b, "Monitor", pl.TolerationsMonitor, pl.LabelsMonitor, pl, hasAnti)
	}
}

func writeNode(b *strings.Builder, name string, tols, labels []string, pl config.Placement, anti bool) {
	fmt.Fprintf(b, "  - name: %s\n", name)
	fmt.Fprint(b, "    spec:\n")
	if len(tols) > 0 {
		fmt.Fprint(b, "      tolerations:\n")
		for _, t := range tols {
			key, val, eff, equal := parseToleration(t)
			fmt.Fprintf(b, "      - key: %q\n", key)
			if equal {
				fmt.Fprint(b, "        operator: \"Equal\"\n")
				fmt.Fprintf(b, "        value: %q\n", val)
			} else {
				fmt.Fprint(b, "        operator: \"Exists\"\n")
			}
			fmt.Fprintf(b, "        effect: %q\n", eff)
		}
	}
	if len(labels) > 0 {
		fmt.Fprint(b, "      nodeSelector:\n")
		for _, l := range labels {
			writeKeyValueEntry(b, "        ", l)
		}
	}
	// The three affinity kinds share one `affinity:` header and are each emitted
	// only when configured, so a config using nothing but antiAffinityNamespaces
	// renders exactly the block it always has.
	legacyNS := pl.AntiAffinityNS
	if !anti {
		legacyNS = nil
	}
	hasNodeAff := pl.NodeAffinity.Configured()
	hasPodAff := len(pl.PodAffinity) > 0
	hasAntiAff := len(legacyNS) > 0 || len(pl.PodAntiAffinity) > 0
	if !hasNodeAff && !hasPodAff && !hasAntiAff {
		return
	}
	fmt.Fprint(b, "      affinity:\n")
	if hasNodeAff {
		writeNodeAffinity(b, pl.NodeAffinity)
	}
	if hasPodAff {
		writePodTerms(b, "podAffinity", pl.PodAffinity, nil, 0)
	}
	if hasAntiAff {
		writePodTerms(b, "podAntiAffinity", pl.PodAntiAffinity, legacyNS, pl.AntiAffinityWeight)
	}
}

// writeNodeAffinity emits the nodeAffinity block. Required is rendered as a single
// ANDed nodeSelectorTerm, which is the subset the schema models.
func writeNodeAffinity(b *strings.Builder, na config.NodeAffinity) {
	fmt.Fprint(b, "        nodeAffinity:\n")
	if len(na.Preferred) > 0 {
		fmt.Fprint(b, "          preferredDuringSchedulingIgnoredDuringExecution:\n")
		for _, term := range na.Preferred {
			fmt.Fprintf(b, "          - weight: %d\n", term.Weight)
			fmt.Fprint(b, "            preference:\n")
			fmt.Fprint(b, "              matchExpressions:\n")
			writeMatchExprs(b, "              ", term.Match)
		}
	}
	if len(na.Required) > 0 {
		fmt.Fprint(b, "          requiredDuringSchedulingIgnoredDuringExecution:\n")
		fmt.Fprint(b, "            nodeSelectorTerms:\n")
		fmt.Fprint(b, "            - matchExpressions:\n")
		writeMatchExprs(b, "              ", na.Required)
	}
}

func writeMatchExprs(b *strings.Builder, indent string, exprs []config.NodeMatchExpr) {
	for _, e := range exprs {
		fmt.Fprintf(b, "%s- key: %q\n", indent, e.Key)
		// operator is a validated enum, so it needs no quoting.
		fmt.Fprintf(b, "%s  operator: %s\n", indent, e.Operator)
		if len(e.Values) > 0 {
			fmt.Fprintf(b, "%s  values:\n", indent)
			for _, v := range e.Values {
				fmt.Fprintf(b, "%s  - %q\n", indent, v)
			}
		}
	}
}

// writePodTerms emits a podAffinity or podAntiAffinity block. legacyNS, when set,
// first emits the fixed broker-spread term antiAffinityNamespaces has always
// produced -- byte for byte, so existing manifests do not move -- and terms adds
// any explicitly configured ones after it. Weight 0 means a required term.
func writePodTerms(b *strings.Builder, kind string, terms []config.PodAffinityTerm, legacyNS []string, legacyWeight int) {
	fmt.Fprintf(b, "        %s:\n", kind)
	var preferred, required []config.PodAffinityTerm
	for _, t := range terms {
		if t.Weight > 0 {
			preferred = append(preferred, t)
		} else {
			required = append(required, t)
		}
	}
	if len(legacyNS) > 0 || len(preferred) > 0 {
		fmt.Fprint(b, "          preferredDuringSchedulingIgnoredDuringExecution:\n")
		if len(legacyNS) > 0 {
			fmt.Fprintf(b, "          - weight: %d\n", legacyWeight)
			fmt.Fprint(b, "            podAffinityTerm:\n")
			fmt.Fprint(b, "              topologyKey: kubernetes.io/hostname\n")
			fmt.Fprint(b, "              labelSelector:\n")
			fmt.Fprint(b, "                matchLabels:\n")
			fmt.Fprint(b, "                  app.kubernetes.io/name: pubsubpluseventbroker\n")
			fmt.Fprint(b, "              namespaces:\n")
			for _, ns := range legacyNS {
				fmt.Fprintf(b, "              - %s\n", ns)
			}
		}
		for _, t := range preferred {
			fmt.Fprintf(b, "          - weight: %d\n", t.Weight)
			fmt.Fprint(b, "            podAffinityTerm:\n")
			fmt.Fprintf(b, "              topologyKey: %q\n", t.TopologyKey)
			writePodTermSelector(b, "              ", t)
		}
	}
	if len(required) > 0 {
		fmt.Fprint(b, "          requiredDuringSchedulingIgnoredDuringExecution:\n")
		for _, t := range required {
			fmt.Fprintf(b, "          - topologyKey: %q\n", t.TopologyKey)
			writePodTermSelector(b, "            ", t)
		}
	}
}

// writePodTermSelector emits a pod-affinity term's selector and namespaces, the
// part shared by the preferred and required forms.
func writePodTermSelector(b *strings.Builder, indent string, t config.PodAffinityTerm) {
	if len(t.MatchLabels) > 0 {
		fmt.Fprintf(b, "%slabelSelector:\n", indent)
		fmt.Fprintf(b, "%s  matchLabels:\n", indent)
		for _, k := range sortedKeys(t.MatchLabels) {
			fmt.Fprintf(b, "%s    %q: %q\n", indent, k, t.MatchLabels[k])
		}
	}
	if len(t.Namespaces) > 0 {
		fmt.Fprintf(b, "%snamespaces:\n", indent)
		for _, ns := range t.Namespaces {
			fmt.Fprintf(b, "%s- %q\n", indent, ns)
		}
	}
}

// writeStringMap emits an optional map as a YAML mapping under header, quoting
// both key and value so an annotation carrying a colon or a quote cannot corrupt
// the manifest. Keys are sorted: Go map order is random, and the output has to be
// deterministic to be diffable and golden-tested.
func writeStringMap(b *strings.Builder, header string, m map[string]string) {
	if len(m) == 0 {
		return
	}
	fmt.Fprint(b, header)
	for _, k := range sortedKeys(m) {
		fmt.Fprintf(b, "    %q: %q\n", k, m[k])
	}
}

// sortedKeys returns a map's keys in a stable order.
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func writeLBAnnotations(b *strings.Builder, lb config.LoadBalancer) {
	if lb.IP == "" && lb.IPPool == "" && len(lb.Annotations) == 0 {
		return
	}
	fmt.Fprint(b, "    annotations:\n")
	if lb.IP != "" {
		fmt.Fprintf(b, "      metallb.universe.tf/loadBalancerIPs: %s\n", lb.IP)
		fmt.Fprintf(b, "      metallb.io/loadBalancerIPs: %s\n", lb.IP)
	}
	if lb.IPPool != "" {
		fmt.Fprintf(b, "      metallb.universe.tf/address-pool: %s\n", lb.IPPool)
		fmt.Fprintf(b, "      metallb.io/address-pool: %s\n", lb.IPPool)
	}
	for _, a := range lb.Annotations {
		writeKeyValueEntry(b, "      ", a)
	}
}

// writeKeyValueEntry emits a user-supplied "key: value" fragment as a quoted YAML
// mapping entry. These configured forms (loadBalancer.annotations,
// placement.labels*) are free text that used to be pasted into the manifest
// verbatim, so a value carrying a colon, a quote or a leading '@' silently
// corrupted the document; quoting both halves keeps the structure intact whatever
// the value is (§4a: escape anything landing in a structured format). Validate has
// already rejected an entry with no key at all.
func writeKeyValueEntry(b *strings.Builder, indent, entry string) {
	key, value := cut(entry, ":")
	fmt.Fprintf(b, "%s%q: %q\n", indent, strings.TrimSpace(key), strings.TrimSpace(value))
}

// EnvPair is one Solace broker configuration setting. Both container engines
// consume the same ordered list; only the on-disk framing differs.
type EnvPair struct{ Key, Value string }

// Assignment renders the pair as "key=value" (podman Environment=, env-file line).
func (p EnvPair) Assignment() string { return p.Key + "=" + p.Value }

// Suffixes for the engine-side secret names. The name is the container name plus
// one of these: it is a host-wide identifier (a podman secret in the host's store,
// a host environment variable feeding compose), so two brokers on one host must
// not collide -- while the default container name "solace" keeps the names the
// tool has always used. The deploy artifact and the secret-creation script derive
// them the same way, and nothing else consumes them.
const (
	adminPassSuffix = "-admin-password"
	pskSuffix       = "-redundancy-psk"
	certPassSuffix  = "-tls-passphrase"

	// filePathSuffix turns a broker setting into its "read the value from this
	// file" variant (username_admin_password ->
	// username_admin_passwordfilepath). Every platform uses this form: secrets
	// reach the broker as files, never as environment values.
	filePathSuffix = "filepath"
)

// secretSpec is one secret of this deployment before the engine-side name is
// known: the broker setting it feeds, the env-file key it came from, its value,
// and the suffix that names it on the host. ContainerSecrets adds the container
// name; EnvPairs needs only the setting, since the in-container path is derived
// from it and is therefore identical on both engines.
type secretSpec struct {
	envKey     string
	configKey  string
	value      string
	nameSuffix string
}

// containerSecretSpecs lists the values that must never appear in a deploy
// artifact: the admin password (always), one password per additional CLI user, the
// redundancy pre-shared key in HA, and the server-certificate passphrase when the
// key is encrypted. The order is fixed -- config order for the users -- so the
// rendered references and the secret-creation script are deterministic and
// golden-testable.
func containerSecretSpecs(c *config.Config) []secretSpec {
	specs := []secretSpec{{
		envKey:     "username_" + c.Admin.User + "_password",
		configKey:  "admin.pass",
		value:      c.Admin.Pass,
		nameSuffix: adminPassSuffix,
	}}
	// Usernames are unique and identifier-checked by config.Validate, so both the
	// host-side name and the in-container filename are unique without further work.
	for _, u := range c.Admin.AdditionalUsers {
		specs = append(specs, secretSpec{
			envKey:     "username_" + u.Username + "_password",
			configKey:  "admin.additionalUsers." + u.Username + ".password",
			value:      u.Password,
			nameSuffix: "-user-" + u.Username + "-password",
		})
	}
	if c.RedundancyEnabled() {
		specs = append(specs, secretSpec{
			envKey:     "redundancy_authentication_presharedkey_key",
			configKey:  "nodes.psk",
			value:      c.Nodes.PSK,
			nameSuffix: pskSuffix,
		})
	}
	// Only when the key is actually encrypted: an empty passphrase must not
	// produce a secret the broker would then try to unlock a plain key with.
	if c.TLS.CertPassphrase != "" {
		specs = append(specs, secretSpec{
			envKey:     "tls_servercertificate_passphrase",
			configKey:  "tls.certPassphrase",
			value:      c.TLS.CertPassphrase,
			nameSuffix: certPassSuffix,
		})
	}
	return specs
}

// ContainerSecret is one broker setting whose value is a secret, kept out of
// every deployment artifact. Both engines mount Value as a file at MountPath and
// point FilePathKey at it; they differ only in where the value comes from --
// podman's own secret store, or a host environment variable compose reads. Value
// is only ever handed to the engine and printed by SecretScript -- never rendered
// into an artifact.
type ContainerSecret struct {
	Name      string // engine-side secret name (container name + suffix)
	EnvKey    string // the broker setting this secret feeds
	Value     string // the secret itself
	ConfigKey string // the env-file key it came from, for actionable errors
}

// FilePathKey is the broker setting that reads the value from the mounted secret
// file instead of taking it directly.
func (s ContainerSecret) FilePathKey() string { return s.EnvKey + filePathSuffix }

// MountPath is where the secret's file appears inside the container. It is named
// after the broker setting rather than the host-side secret name, so the layout
// inside the container is the same on every host and matches the data keys of the
// equivalent Kubernetes Secret.
func (s ContainerSecret) MountPath() string { return secretFilePath(s.EnvKey) }

// Target is the in-container filename the engine mounts this secret as -- the
// last element of MountPath, which is what compose's `target:` and podman's
// `target=` take.
func (s ContainerSecret) Target() string { return s.EnvKey }

// EnvVar is the host environment variable docker's compose file reads this
// secret's value from: the uppercased secret name with every character a variable
// name cannot carry replaced by '_'. A name starting with a digit gets a leading
// '_', since a shell cannot export the former.
func (s ContainerSecret) EnvVar() string {
	var b strings.Builder
	for i, r := range strings.ToUpper(s.Name) {
		switch {
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			if i == 0 {
				b.WriteByte('_')
			}
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}

// secretFilePath is the in-container path of the secret feeding setting envKey.
func secretFilePath(envKey string) string { return secretMount + "/" + envKey }

// ContainerSecrets is the deployment's secrets with their engine-side names for
// platform p, which is what the artifacts reference and the Manager creates.
func ContainerSecrets(c *config.Config, p config.Platform) []ContainerSecret {
	prefix := c.ContainerBlock(p).Name
	specs := containerSecretSpecs(c)
	secrets := make([]ContainerSecret, 0, len(specs))
	for _, s := range specs {
		secrets = append(secrets, ContainerSecret{
			Name:      prefix + s.nameSuffix,
			EnvKey:    s.envKey,
			Value:     s.value,
			ConfigKey: s.configKey,
		})
	}
	return secrets
}

// EnvPairs ports gen_env_pairs: the ordered broker config for a container host,
// branching on redundancy. id is this host's resolved identity (ResolveNode).
// It takes no platform: every value it reads is shared, since the docker and
// podman broker configuration is identical and only the framing differs.
//
// No secret value is here: each one is externalized (ContainerSecrets) and this
// list only points the broker at the file it will be mounted as, so it is safe to
// print in full (§3).
func EnvPairs(c *config.Config, id config.NodeIdentity) []EnvPair {
	// TZ is the same cross-platform timezone the k8s CR uses, and it is optional:
	// an unset one leaves the container on the image default, so the pair is
	// omitted entirely rather than emitted empty.
	var pairs []EnvPair
	if c.Timezone != "" {
		pairs = append(pairs, EnvPair{"TZ", c.Timezone})
	}
	pairs = append(pairs,
		EnvPair{"routername", id.Hostname},
		EnvPair{"nodetype", id.NodeType},
	)

	if c.RedundancyEnabled() {
		if id.ActiveStandby != "" {
			pairs = append(pairs, EnvPair{"redundancy_activestandbyrole", id.ActiveStandby})
		}
		pairs = append(pairs,
			EnvPair{"redundancy_enable", "yes"},
			EnvPair{"configsync_enable", "yes"},
		)
		n := c.Nodes
		pairs = append(pairs,
			EnvPair{groupKey(n.Primary.Name, "connectvia"), n.Primary.IP},
			EnvPair{groupKey(n.Primary.Name, "nodetype"), "message_routing"},
			EnvPair{groupKey(n.Backup.Name, "connectvia"), n.Backup.IP},
			EnvPair{groupKey(n.Backup.Name, "nodetype"), "message_routing"},
			EnvPair{groupKey(n.Monitor.Name, "connectvia"), n.Monitor.IP},
			EnvPair{groupKey(n.Monitor.Name, "nodetype"), "monitoring"},
		)
		if c.TLS.Cert != "" {
			pairs = append(pairs,
				EnvPair{"configsync_tls_enable", "yes"},
				EnvPair{"redundancy_matelink_tls_enable", "yes"},
			)
		}
	} else {
		pairs = append(pairs, EnvPair{"redundancy_enable", "no"})
	}

	if c.TLS.Cert != "" {
		pairs = append(pairs, EnvPair{"tls_servercertificate_filepath", certMount})
	}

	// Every scaling knob reaches the container, in the same order and under the
	// same broker setting names the k8s CR writes into spec.systemScaling -- one
	// schema block, one set of settings, two ways of delivering them. They are
	// emitted unconditionally so the artifact states the whole sizing rather than
	// leaving the broker's internal defaults to fill the gaps invisibly.
	pairs = append(pairs,
		EnvPair{"system_scaling_maxconnectioncount", itoa(c.Scaling.MaxConnections)},
		EnvPair{"system_scaling_maxqueuemessagecount", itoa(c.Scaling.MaxQueueMessages)},
		EnvPair{"system_scaling_maxkafkabridgecount", itoa(c.Scaling.MaxKafkaBridge)},
		EnvPair{"system_scaling_maxkafkabrokerconnectioncount", itoa(c.Scaling.MaxKafkaConnections)},
		EnvPair{"system_scaling_maxbridgecount", itoa(c.Scaling.MaxBridges)},
		EnvPair{"system_scaling_maxsubscriptioncount", itoa(c.Scaling.MaxSubscriptions)},
		EnvPair{"system_scaling_maxguaranteedmessagesize", itoa(c.Scaling.MaxGuaranteedMsgMB)},
		EnvPair{"messagespool_maxspoolusage", itoa(c.Scaling.MaxSpoolUsageMB)},
		EnvPair{"username_" + c.Admin.User + "_globalaccesslevel", "admin"},
	)

	// An additional user's access level is not a secret, so it rides the artifact
	// like any other setting; only the password is externalized (below).
	for _, u := range c.Admin.AdditionalUsers {
		pairs = append(pairs, EnvPair{"username_" + u.Username + "_globalaccesslevel", u.AccessLevel})
	}

	// Point each secret-bearing setting at the file the engine mounts it as. The
	// path derives from the setting, not from the per-host secret name, so both
	// engines emit exactly these lines and the value stays out of the artifact (§3).
	for _, s := range containerSecretSpecs(c) {
		pairs = append(pairs, EnvPair{s.envKey + filePathSuffix, secretFilePath(s.envKey)})
	}
	return pairs
}

// healthPort is where the broker serves its own health-check endpoints.
const healthPort = "5550"

// healthCmd is the probe argv for a container health check: the configured one, or
// the built-in readiness probe. /health-check/readiness reports whether the broker
// is ready to carry traffic rather than merely running, which is the distinction
// that makes `docker ps`/`compose ps` and podman's auto-restart meaningful. It
// exists from broker 10.26 onward, and config.Validate refuses to enable the
// built-in probe against an older or unidentifiable image, so by the time this
// renders the version has been checked.
func healthCmd(hc config.HealthCheck) []string {
	if len(hc.Cmd) > 0 {
		return hc.Cmd
	}
	return []string{"curl", "-fs", "http://localhost:" + healthPort + "/health-check/readiness"}
}


func groupKey(node, suffix string) string {
	return "redundancy_group_node_" + node + "_" + suffix
}

// Quadlet ports gen_quadlet: a podman systemd .container unit for this host.
func Quadlet(c *config.Config, id config.NodeIdentity) []byte {
	const p = config.Podman
	cb := c.ContainerBlock(p)
	net := c.NetworkBlock(p)
	uid, gid := splitUser(cb.RunUser)

	var b strings.Builder
	fmt.Fprint(&b, "[Unit]\n")
	fmt.Fprintf(&b, "Description=Solace PubSub+ Event Broker (%s, nodetype=%s)\n", id.Hostname, id.NodeType)
	fmt.Fprint(&b, "Wants=network-online.target\n")
	fmt.Fprint(&b, "After=network-online.target\n")
	fmt.Fprint(&b, "\n")
	fmt.Fprint(&b, "[Container]\n")
	fmt.Fprintf(&b, "Image=%s\n", c.Image.Ref())
	fmt.Fprintf(&b, "ContainerName=%s\n", cb.Name)
	fmt.Fprintf(&b, "HostName=%s\n", id.Hostname)
	fmt.Fprintf(&b, "User=%s\n", uid)
	if gid != "" {
		fmt.Fprintf(&b, "Group=%s\n", gid)
	}
	// Memory has a first-class quadlet key; CPU has none, so the tier's core cap
	// rides PodmanArgs -- the documented escape hatch for a podman run flag
	// quadlet does not map. ASSUMED, NOT VERIFIED: podman was not testable here,
	// so confirm both against the target podman before relying on them; if
	// Memory= predates that version, fold it into the same PodmanArgs line.
	// Both are skipped when unset, which is what a Config built in code without
	// ApplyDefaults carries -- an empty --cpus= would fail the unit at start.
	if c.Scaling.CPU != "" {
		fmt.Fprintf(&b, "PodmanArgs=--cpus=%s\n", c.Scaling.CPU)
	}
	if cb.Mem != "" {
		fmt.Fprintf(&b, "Memory=%s\n", cb.Mem)
	}
	fmt.Fprintf(&b, "ShmSize=%s\n", cb.ShmSize)
	fmt.Fprintf(&b, "Ulimit=nofile=%s\n", cb.Ulimits.NoFile)
	fmt.Fprintf(&b, "Ulimit=memlock=%s\n", cb.Ulimits.MemLock)
	fmt.Fprintf(&b, "Ulimit=core=%s\n", cb.Ulimits.Core)
	if hc := cb.HealthCheck; hc.Enabled {
		// Quadlet takes a command line rather than an argv, so the probe is joined;
		// a token containing a space is not representable here (documented in the
		// schema: wrap such a probe in a script). Only the percent is escaped:
		// systemd expands specifiers in every assignment in the unit, so a
		// percent-encoded character in a probe URL would otherwise be read as one
		// and the line dropped. Quotes and backslashes are deliberately left alone,
		// unlike Environment= -- these values are unquoted and podman splits the
		// command line itself, so escaping them here would corrupt the probe.
		fmt.Fprintf(&b, "HealthCmd=%s\n", escapePercent(strings.Join(healthCmd(hc), " ")))
		fmt.Fprintf(&b, "HealthInterval=%s\n", escapePercent(hc.Interval))
		fmt.Fprintf(&b, "HealthTimeout=%s\n", escapePercent(hc.Timeout))
		fmt.Fprintf(&b, "HealthRetries=%d\n", hc.Retries)
		fmt.Fprintf(&b, "HealthStartPeriod=%s\n", escapePercent(hc.StartPeriod))
	}
	if net.Mode == "host" {
		fmt.Fprint(&b, "Network=host\n")
	} else {
		for _, port := range net.Ports {
			fmt.Fprintf(&b, "PublishPort=%s\n", port)
		}
	}
	fmt.Fprintf(&b, "Volume=%s:%s:Z\n", cb.DataDir, dataMount)
	if c.TLS.Cert != "" {
		fmt.Fprintf(&b, "Volume=%s:%s:ro\n", c.TLS.Cert, certMount)
	}
	for _, pair := range EnvPairs(c, id) {
		fmt.Fprintf(&b, "Environment=\"%s\"\n", quadletEscape(pair.Assignment()))
	}
	// Secrets ride podman's own secret store: the unit names them, podman mounts
	// each one at /run/secrets/<target> at start (the EnvPairs above already point
	// the broker there), and no secret lands in the unit. A relative target is
	// resolved under /run/secrets by every podman that supports mount secrets.
	for _, s := range ContainerSecrets(c, p) {
		fmt.Fprintf(&b, "Secret=%s,type=mount,target=%s\n", s.Name, s.Target())
	}
	fmt.Fprint(&b, "\n")
	fmt.Fprint(&b, "[Service]\n")
	fmt.Fprint(&b, "Restart=always\n")
	fmt.Fprint(&b, "\n")
	fmt.Fprint(&b, "[Install]\n")
	fmt.Fprintf(&b, "WantedBy=%s\n", c.Podman.WantedBy)
	return []byte(b.String())
}

// Compose ports gen_compose: a docker compose file for this host's broker.
func Compose(c *config.Config, id config.NodeIdentity) []byte {
	const p = config.Docker
	cb := c.ContainerBlock(p)
	net := c.NetworkBlock(p)
	soft, hard := splitPair(cb.Ulimits.NoFile)

	var b strings.Builder
	fmt.Fprint(&b, "services:\n")
	fmt.Fprintf(&b, "  %s:\n", cb.Name)
	fmt.Fprintf(&b, "    image: %s\n", c.Image.Ref())
	fmt.Fprintf(&b, "    container_name: %s\n", cb.Name)
	fmt.Fprintf(&b, "    hostname: %s\n", id.Hostname)
	fmt.Fprintf(&b, "    user: %q\n", cb.RunUser)
	fmt.Fprint(&b, "    restart: always\n")
	// Service-level cpus:/mem_limit: rather than deploy.resources.limits, which
	// the standalone v1 docker-compose binary -- the documented fallback behind
	// docker.compose -- ignores without --compatibility, silently dropping the
	// cap. These two are honoured by both v1 and the v2 plugin. Skipped when
	// unset, which is what a Config built in code without ApplyDefaults carries.
	if c.Scaling.CPU != "" {
		fmt.Fprintf(&b, "    cpus: %q\n", c.Scaling.CPU)
	}
	if cb.Mem != "" {
		fmt.Fprintf(&b, "    mem_limit: %s\n", cb.Mem)
	}
	fmt.Fprintf(&b, "    shm_size: %s\n", cb.ShmSize)
	fmt.Fprint(&b, "    ulimits:\n")
	fmt.Fprint(&b, "      nofile:\n")
	fmt.Fprintf(&b, "        soft: %s\n", soft)
	fmt.Fprintf(&b, "        hard: %s\n", hard)
	fmt.Fprintf(&b, "      memlock: %s\n", cb.Ulimits.MemLock)
	fmt.Fprintf(&b, "      core: %s\n", cb.Ulimits.Core)
	if hc := cb.HealthCheck; hc.Enabled {
		fmt.Fprint(&b, "    healthcheck:\n")
		fmt.Fprint(&b, "      test: [\"CMD\"")
		for _, tok := range healthCmd(hc) {
			fmt.Fprintf(&b, ", %q", tok)
		}
		fmt.Fprint(&b, "]\n")
		fmt.Fprintf(&b, "      interval: %s\n", hc.Interval)
		fmt.Fprintf(&b, "      timeout: %s\n", hc.Timeout)
		fmt.Fprintf(&b, "      retries: %d\n", hc.Retries)
		fmt.Fprintf(&b, "      start_period: %s\n", hc.StartPeriod)
	}
	if net.Mode == "host" {
		fmt.Fprint(&b, "    network_mode: host\n")
	} else {
		fmt.Fprint(&b, "    ports:\n")
		for _, port := range net.Ports {
			fmt.Fprintf(&b, "      - %q\n", port)
		}
	}
	fmt.Fprint(&b, "    volumes:\n")
	fmt.Fprintf(&b, "      - %q\n", cb.DataDir+":"+dataMount+":Z")
	if c.TLS.Cert != "" {
		fmt.Fprintf(&b, "      - %q\n", c.TLS.Cert+":"+certMount+":ro")
	}
	fmt.Fprint(&b, "    environment:\n")
	for _, pair := range EnvPairs(c, id) {
		fmt.Fprintf(&b, "      %s: %q\n", pair.Key, pair.Value)
	}
	// Secrets are mounted as files under the name of the setting they feed (the
	// *filepath pointers are already among the EnvPairs above), and their values
	// come from this host's environment -- which `deploy` sets for the compose
	// process -- so nothing secret is written beside this file.
	secrets := ContainerSecrets(c, p)
	fmt.Fprint(&b, "    secrets:\n")
	for _, s := range secrets {
		fmt.Fprintf(&b, "      - source: %s\n", s.Name)
		fmt.Fprintf(&b, "        target: %s\n", s.Target())
	}
	fmt.Fprint(&b, "secrets:\n")
	for _, s := range secrets {
		fmt.Fprintf(&b, "  %s:\n", s.Name)
		fmt.Fprintf(&b, "    environment: %s\n", s.EnvVar())
	}
	return []byte(b.String())
}

// SecretPreflight reports the first secret this deployment cannot create because
// its value is unset. Both the real deploy and `--gen-secrets-only` call it, so
// the two refuse on the same precondition: a script that creates an EMPTY secret
// is worse than no script, because the broker then starts with a blank password
// or mate-link key and only fails later, obscurely. nodes.psk is legitimately
// empty until `prep host` generates it, which is exactly the case the hint names.
func SecretPreflight(c *config.Config, p config.Platform) error {
	for _, s := range ContainerSecrets(c, p) {
		if s.Value != "" {
			continue
		}
		hint := ""
		if s.ConfigKey == "nodes.psk" {
			hint = fmt.Sprintf(" or run `solace %s prep host` to generate it", p)
		}
		return fmt.Errorf("%s is empty but the deploy needs it as secret %q; set it in the env file%s",
			s.ConfigKey, s.Name, hint)
	}
	return nil
}

// SecretScript renders the shell commands that supply this deployment's
// externalized secrets, one line per secret -- what `--gen-secrets-only` prints.
// Podman loads them into its secret store, so the script is run once. Docker's
// compose secrets read host environment variables, so the script is *sourced* in
// the shell that runs compose; `deploy` sets the same variables for its own
// compose process and needs no script at all. This is the only renderer that
// emits secret values, so its output must be handled exactly like the env file it
// came from.
func SecretScript(c *config.Config, p config.Platform) []byte {
	secrets := ContainerSecrets(c, p)
	var b strings.Builder
	fmt.Fprintf(&b, "# %s secrets for container %s -- CONTAINS SECRET VALUES.\n", p, c.ContainerBlock(p).Name)
	if p == config.Podman {
		fmt.Fprint(&b, "# Run on this host to create them; `deploy` creates the same ones itself.\n")
		rt := c.ContainerRuntime(p).String()
		for _, s := range secrets {
			fmt.Fprintf(&b, "printf '%%s' %s | %s secret create --replace %s -\n", shQuote(s.Value), rt, s.Name)
		}
		return []byte(b.String())
	}
	fmt.Fprint(&b, "# Source this in the shell you run `docker compose` from; `deploy` sets the\n")
	fmt.Fprint(&b, "# same variables for its own compose process, so it needs nothing from here.\n")
	for _, s := range secrets {
		fmt.Fprintf(&b, "export %s=%s\n", s.EnvVar(), shQuote(s.Value))
	}
	return []byte(b.String())
}

// --- small helpers -----------------------------------------------------------

type portSpec struct{ name, container, service, proto string }

// parsePort splits a "name=containerPort[:servicePort][/proto]" entry, porting
// the parsing in gen_yaml's ports loop. Defaults: proto TCP, servicePort =
// containerPort.
func parsePort(entry string) portSpec {
	name, rest := cut(entry, "=")
	proto := "TCP"
	if i := strings.LastIndex(rest, "/"); i >= 0 {
		proto = rest[i+1:]
		rest = rest[:i]
	}
	container := rest
	service := rest
	if i := strings.Index(rest, ":"); i >= 0 {
		container = rest[:i]
		service = rest[i+1:]
	}
	return portSpec{name: name, container: container, service: service, proto: proto}
}

// parseToleration splits "key[=value]:effect", porting gen_yaml's toleration
// parsing (key before the first ':', effect after the last ':').
func parseToleration(s string) (key, value, effect string, equal bool) {
	keyPart := s
	if i := strings.Index(s, ":"); i >= 0 {
		keyPart = s[:i]
	}
	if i := strings.LastIndex(s, ":"); i >= 0 {
		effect = s[i+1:]
	}
	if i := strings.Index(keyPart, "="); i >= 0 {
		return keyPart[:i], keyPart[i+1:], effect, true
	}
	return keyPart, "", effect, false
}

// splitUser splits "uid:gid" into its parts; a bare "uid" yields an empty gid.
func splitUser(u string) (uid, gid string) { return cut(u, ":") }

// splitPair splits "soft:hard"; a bare value yields it for both.
func splitPair(v string) (a, b string) {
	if i := strings.Index(v, ":"); i >= 0 {
		return v[:i], v[i+1:]
	}
	return v, v
}

// cut splits s on the first sep; if sep is absent, before=s and after="".
func cut(s, sep string) (before, after string) {
	if i := strings.Index(s, sep); i >= 0 {
		return s[:i], s[i+len(sep):]
	}
	return s, ""
}

// quadletEscape escapes a value for a systemd Environment="..." assignment:
// backslash, double-quote, and '%' (systemd expands %-specifiers). Ports
// gen_quadlet's inline escaping.
func quadletEscape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return escapePercent(s)
}

// escapePercent doubles '%' so systemd does not read it as a specifier. It applies
// to every assignment in a unit file, not just the quoted ones, so the unquoted
// Health* keys need it too even though they must not have quotes escaped.
func escapePercent(s string) string { return strings.ReplaceAll(s, "%", "%%") }

// shQuote single-quotes a value for the generated secret script, so a password
// holding shell metacharacters is created verbatim rather than interpreted.
func shQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func itoa(i int) string { return strconv.Itoa(i) }
