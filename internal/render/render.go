// Package render turns a validated config.Config into the deployment artifacts
// each platform applies:
//
//   - BrokerCR  -- the Kubernetes PubSubPlusEventBroker custom resource
//   - EnvPairs  -- the ordered Solace key=value config both container engines share
//   - Quadlet   -- a podman systemd .container unit
//   - Compose   -- a docker compose file
//   - RunArgs   -- docker run arguments
//
// It is the Go port of the bash generators: gen_yaml (020-deploy-broker.sh),
// gen_env_pairs (docker-podman/000-env.sh), gen_quadlet (podman-020) and
// gen_compose/build_run_args (docker-020).
//
// Rendering is done with plain string builders rather than text/template: the
// broker CR is deeply conditional YAML where template whitespace control is
// fragile, and every function here is a pure function of the config (no I/O, no
// exec), so the output is stable and golden-tested in render_test.go.
package render

import (
	"fmt"
	"strconv"
	"strings"

	"solace/internal/config"
)

// k8sTimezone mirrors the timezone hardcoded in gen_yaml (020-deploy-broker.sh).
// Kept for parity; there is no k8s timezone field in the config today.
const k8sTimezone = "Asia/Singapore"

// In-container paths the broker image expects.
const (
	dataMount = "/var/lib/solace"
	certMount = "/run/secrets/tls.crt"
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
	fmt.Fprint(&b, "    pullPolicy: IfNotPresent\n")
	if c.Image.PullSecret != "" {
		fmt.Fprint(&b, "    pullSecrets:\n")
		fmt.Fprintf(&b, "    - name: %s\n", c.Image.PullSecret)
	}
	if c.K8s.ServiceAccount != "" {
		fmt.Fprint(&b, "  serviceAccount:\n")
		fmt.Fprintf(&b, "    name: %s\n", c.K8s.ServiceAccount)
	}
	fmt.Fprintf(&b, "  adminCredentialsSecret: %s\n", c.Admin.UserSecret)
	fmt.Fprintf(&b, "  monitoringCredentialsSecret: %s\n", c.Admin.UserSecret)
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
	fmt.Fprintf(&b, "    maxSpoolUsage: %d\n", s.MaxPool)
	fmt.Fprintf(&b, "    messagingNodeCpu: %q\n", c.K8s.MsgNode.CPU)
	fmt.Fprintf(&b, "    messagingNodeMemory: %s\n", c.K8s.MsgNode.Mem)

	fmt.Fprint(&b, "  storage:\n")
	if c.K8s.Storage.Class != "" {
		fmt.Fprintf(&b, "    useStorageClass: %s\n", c.K8s.Storage.Class)
	}
	fmt.Fprintf(&b, "    messagingNodeStorageSize: %s\n", c.K8s.Storage.MsgNode)
	fmt.Fprintf(&b, "    monitorNodeStorageSize: %s\n", c.K8s.Storage.MonNode)
	fmt.Fprintf(&b, "  timezone: %q\n", k8sTimezone)

	if c.TLS.ServerSecret != "" {
		fmt.Fprint(&b, "  tls:\n")
		fmt.Fprintf(&b, "    serverTlsConfigSecret: %s\n", c.TLS.ServerSecret)
		fmt.Fprint(&b, "    enabled: true\n")
		fmt.Fprint(&b, "    certFilename: tls.crt\n")
		fmt.Fprint(&b, "    certKeyFilename: tls.key\n")
	}

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

// writeNodeAssignment emits the nodeAssignment block when any placement knob is
// set (tolerations, node labels, or anti-affinity namespaces). Backup/Monitor
// nodes are only emitted in HA. Mirrors gen_yaml's nodeAssignment section.
func writeNodeAssignment(b *strings.Builder, c *config.Config) {
	pl := c.K8s.Placement
	hasTol := len(pl.TolerationsPrimary)+len(pl.TolerationsBackup)+len(pl.TolerationsMonitor) > 0
	hasLabels := len(pl.LabelsPrimary)+len(pl.LabelsBackup)+len(pl.LabelsMonitor) > 0
	hasAnti := len(pl.AntiAffinityNS) > 0
	if !hasTol && !hasLabels && !hasAnti {
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
			fmt.Fprintf(b, "        %s\n", l)
		}
	}
	if anti {
		fmt.Fprint(b, "      affinity:\n")
		fmt.Fprint(b, "        podAntiAffinity:\n")
		fmt.Fprint(b, "          preferredDuringSchedulingIgnoredDuringExecution:\n")
		fmt.Fprintf(b, "          - weight: %d\n", pl.AntiAffinityWeight)
		fmt.Fprint(b, "            podAffinityTerm:\n")
		fmt.Fprint(b, "              topologyKey: kubernetes.io/hostname\n")
		fmt.Fprint(b, "              labelSelector:\n")
		fmt.Fprint(b, "                matchLabels:\n")
		fmt.Fprint(b, "                  app.kubernetes.io/name: pubsubpluseventbroker\n")
		fmt.Fprint(b, "              namespaces:\n")
		for _, ns := range pl.AntiAffinityNS {
			fmt.Fprintf(b, "              - %s\n", ns)
		}
	}
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
		fmt.Fprintf(b, "      %s\n", a)
	}
}

// EnvPair is one Solace broker configuration setting. Both container engines
// consume the same ordered list; only the on-disk framing differs.
type EnvPair struct{ Key, Value string }

// Assignment renders the pair as "key=value" (podman Environment=, docker run -e).
func (p EnvPair) Assignment() string { return p.Key + "=" + p.Value }

// EnvPairs ports gen_env_pairs: the ordered broker config for a container host,
// branching on redundancy. id is this host's resolved identity (ResolveNode).
func EnvPairs(c *config.Config, p config.Platform, id config.NodeIdentity) []EnvPair {
	pairs := []EnvPair{
		{"TZ", c.ContainerBlock(p).TZ},
		{"routername", id.Hostname},
		{"nodetype", id.NodeType},
	}

	if c.RedundancyEnabled() {
		if id.ActiveStandby != "" {
			pairs = append(pairs, EnvPair{"redundancy_activestandbyrole", id.ActiveStandby})
		}
		pairs = append(pairs,
			EnvPair{"redundancy_enable", "yes"},
			EnvPair{"redundancy_authentication_presharedkey_key", c.Nodes.PSK},
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

	pairs = append(pairs,
		EnvPair{"system_scaling_maxconnectioncount", itoa(c.Scaling.MaxConnections)},
		EnvPair{"system_scaling_maxqueuemessagecount", itoa(c.Scaling.MaxQueueMessages)},
		EnvPair{"messagespool_maxspoolusage", itoa(c.Scaling.MaxSpoolUsageMB)},
		EnvPair{"username_" + c.Admin.User + "_globalaccesslevel", "admin"},
		EnvPair{"username_" + c.Admin.User + "_password", c.Admin.Pass},
	)
	return pairs
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
	fmt.Fprintf(&b, "ShmSize=%s\n", cb.ShmSize)
	fmt.Fprintf(&b, "Ulimit=nofile=%s\n", cb.Ulimits.NoFile)
	fmt.Fprintf(&b, "Ulimit=memlock=%s\n", cb.Ulimits.MemLock)
	fmt.Fprintf(&b, "Ulimit=core=%s\n", cb.Ulimits.Core)
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
	for _, pair := range EnvPairs(c, p, id) {
		fmt.Fprintf(&b, "Environment=\"%s\"\n", quadletEscape(pair.Assignment()))
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
	fmt.Fprintf(&b, "    shm_size: %s\n", cb.ShmSize)
	fmt.Fprint(&b, "    ulimits:\n")
	fmt.Fprint(&b, "      nofile:\n")
	fmt.Fprintf(&b, "        soft: %s\n", soft)
	fmt.Fprintf(&b, "        hard: %s\n", hard)
	fmt.Fprintf(&b, "      memlock: %s\n", cb.Ulimits.MemLock)
	fmt.Fprintf(&b, "      core: %s\n", cb.Ulimits.Core)
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
	for _, pair := range EnvPairs(c, p, id) {
		fmt.Fprintf(&b, "      %s: %q\n", pair.Key, pair.Value)
	}
	return []byte(b.String())
}

// RunArgs ports build_run_args: the arguments for `<engine> run` in docker
// run-mode. The engine binary itself is prepended by the caller.
func RunArgs(c *config.Config, id config.NodeIdentity) []string {
	const p = config.Docker
	cb := c.ContainerBlock(p)
	net := c.NetworkBlock(p)

	args := []string{
		"run", "-d",
		"--name", cb.Name,
		"--hostname", id.Hostname,
		"-u", cb.RunUser,
		"--restart=always",
		"--shm-size=" + cb.ShmSize,
		"--ulimit", "nofile=" + cb.Ulimits.NoFile,
		"--ulimit", "memlock=" + cb.Ulimits.MemLock,
		"--ulimit", "core=" + cb.Ulimits.Core,
	}
	if net.Mode == "host" {
		args = append(args, "--network=host")
	} else {
		for _, port := range net.Ports {
			args = append(args, "-p", port)
		}
	}
	args = append(args, "--mount",
		"type=bind,source="+cb.DataDir+",destination="+dataMount+",relabel=private,ro=false")
	if c.TLS.Cert != "" {
		args = append(args, "-v", c.TLS.Cert+":"+certMount+":ro")
	}
	for _, pair := range EnvPairs(c, p, id) {
		args = append(args, "-e", pair.Assignment())
	}
	args = append(args, c.Image.Ref())
	return args
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
	s = strings.ReplaceAll(s, "%", "%%")
	return s
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func itoa(i int) string { return strconv.Itoa(i) }
