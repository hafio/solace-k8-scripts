package convert

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"solace/internal/config"
)

// Result is a completed conversion: the YAML env file, the platform section it
// was written for, and every assumption or dropped value worth telling the user
// about.
type Result struct {
	YAML     []byte
	Platform config.Platform
	Warnings []string
}

// Detection markers. A legacy env file has no platform field, so the platform is
// inferred from variables only one bootstrap ever defined. Variables both
// bootstraps share (image, admin, tls, cli scripts, diag dir) are deliberately
// absent from both lists -- they say nothing about the target.
var (
	k8sMarkers = []string{
		"KUBE",
		"SOLBK_NAME", "SOLBK_NS", "SOLBK_STORAGE_MSGNODE", "SOLBK_STORAGE_MONNODE",
		"SOLBK_STORAGECLASS", "SOLBK_MSGNODE_CPU", "SOLBK_MSGNODE_MEM",
		"SOLBK_UPDATE_STRATEGY", "SOLBK_ANTIAFFINITY_NS", "SOLBK_SVR_SECRET",
		"SOLOP_IMAGE", "SOLOP_NS", "SOLOP_CPU", "SOLOP_MEM", "SOLOP_WATCH_NS",
	}
	containerMarkers = []string{
		"SOLBK_NODE_PRI_NAME", "SOLBK_NODE_BKP_NAME", "SOLBK_NODE_MON_NAME",
		"SOLBK_DATA_DIR", "SOLBK_NETWORK_MODE", "SOLBK_SPOOL_MAXUSAGE",
		"SOLBK_REDUNDANCY_PSK", "SOLBK_RUN_USER", "SOLBK_TZ", "SOLBK_SHM_SIZE",
		"CONTAINER_NAME", "CONTAINER_RUNTIME",
		"DOCKER_MODE", "DOCKER_COMPOSE_FILE", "PODMAN_ROOTLESS", "QUADLET_DIR",
	}
	dockerMarkers = []string{"DOCKER_MODE", "DOCKER_COMPOSE_FILE"}
	podmanMarkers = []string{"PODMAN_ROOTLESS", "QUADLET_DIR"}
)

// bashOnly are variables the bootstraps used for their own plumbing. They have
// no YAML equivalent by design, so they are consumed silently rather than
// reported as unmapped.
var bashOnly = []string{
	"ENV_FILE", "EXDIR", "PARAMS", "PARM", "GENONLY", "EMPTY_VAR",
	"RUNTIME_DEFAULT", "SOLBK_IMAGE_REF", "SOLOP_DERIVED_NS", "SYSTEMCTL_USER",
	"QUADLET_WANTEDBY", "SELECT_ENV_FILE", "THIS_HOSTNAME", "THIS_NODETYPE",
	"THIS_ACTIVESTANDBY",
}

// Convert parses a legacy bash env file and returns the equivalent YAML.
//
// platform selects which section the platform-specific variables land in; an
// empty platform is detected from the variables present. The conversion never
// fails on an unrecognised variable -- it is reported as a warning so nothing is
// lost silently -- but a malformed array assignment is a hard error, since the
// rest of the file cannot be trusted after one.
func Convert(src []byte, source string, platform config.Platform) (Result, error) {
	v, err := parse(string(src))
	if err != nil {
		return Result{}, fmt.Errorf("parse %s: %w", source, err)
	}
	v.ignore(bashOnly...)

	p, warns := resolvePlatform(v, platform)
	body, mapWarns := emitYAML(v, p, source)
	warns = append(warns, mapWarns...)

	if names := v.unmapped(); len(names) > 0 {
		warns = append(warns, fmt.Sprintf("no YAML equivalent, dropped: %s", strings.Join(names, ", ")))
	}
	if err := validateOutput(body, p); err != nil {
		// "will not load as-is" rather than "is incomplete": the failure is usually a
		// mandatory field the source never set, but it can equally be a legacy
		// KUBE/CONTAINER_RUNTIME the execution guard refuses (a wrapper such as
		// `microk8s kubectl`, which now needs --allow-command). Both are the same
		// thing to the user -- edit the file or approve the binary before running it
		// -- and the wrapped error names which one it was.
		warns = append(warns, fmt.Sprintf("the converted file will not load as-is: %v", err))
	}
	return Result{YAML: []byte(body), Platform: p, Warnings: warns}, nil
}

// resolvePlatform honours an explicit platform, or infers one from the marker
// variables. Every inference that was not clear-cut comes back as a warning, so
// the user can re-run with --platform rather than discover it later.
func resolvePlatform(v *vars, want config.Platform) (config.Platform, []string) {
	if want != "" {
		return want, nil
	}
	k8s, ctr := countMarkers(v, k8sMarkers), countMarkers(v, containerMarkers)
	switch {
	case ctr > 0 && k8s == 0:
		dkr, pod := countMarkers(v, dockerMarkers), countMarkers(v, podmanMarkers)
		switch {
		case pod > 0 && dkr == 0:
			return config.Podman, nil
		case dkr > 0 && pod == 0:
			return config.Docker, nil
		case dkr > 0 && pod > 0:
			return config.Docker, []string{"file carries both docker and podman settings; wrote the docker section (re-run with --platform podman for the other)"}
		}
		return config.Docker, []string{"container env file with no docker- or podman-specific variable; assumed docker (re-run with --platform podman if that is wrong)"}
	case k8s > 0 && ctr == 0:
		return config.K8s, nil
	case k8s > 0 && ctr > 0:
		if ctr > k8s {
			return config.Docker, []string{"file mixes kubernetes and container variables; wrote the docker section (re-run with --platform k8s or podman to pick another)"}
		}
		return config.K8s, []string{"file mixes kubernetes and container variables; wrote the k8s section (re-run with --platform docker or podman to pick another)"}
	}
	return config.K8s, []string{"no platform-specific variable found; assumed k8s (re-run with --platform docker or podman if that is wrong)"}
}

func countMarkers(v *vars, names []string) int {
	n := 0
	for _, name := range names {
		if v.has(name) {
			n++
		}
	}
	return n
}

// validateOutput decodes what was emitted and runs the platform's own defaults
// and validation, so an env file that was already missing mandatory values says
// so at conversion time instead of at the next command.
func validateOutput(body string, p config.Platform) error {
	var c config.Config
	if err := yaml.Unmarshal([]byte(body), &c); err != nil {
		return fmt.Errorf("re-reading the generated YAML failed: %w", err)
	}
	c.ApplyDefaults(p)
	return c.Validate(p)
}

// emitYAML writes the YAML document. It reads the parsed variables directly
// rather than round-tripping a config.Config, so a value the source file never
// set stays absent instead of appearing as a zero.
func emitYAML(v *vars, p config.Platform, source string) (string, []string) {
	var warns []string
	// num and boolean take the destination doc because the value has to be
	// looked up (and a bad one warned about) before anything is written.
	num := func(d *doc, key, name string) {
		raw := v.s(name)
		if raw == "" {
			return
		}
		n, err := strconv.Atoi(raw)
		if err != nil {
			warns = append(warns, fmt.Sprintf("%s=%q is not a number, dropped", name, raw))
			return
		}
		d.raw(key + ": " + strconv.Itoa(n))
	}
	boolean := func(d *doc, key, name string) {
		raw := v.s(name)
		if raw == "" {
			return
		}
		b, ok := boolOf(raw)
		if !ok {
			warns = append(warns, fmt.Sprintf("%s=%q is neither true/yes nor false/no, dropped", name, raw))
			return
		}
		d.raw(key + ": " + strconv.FormatBool(b))
	}

	d := &doc{}
	// The source name is the only free-form text in the document. It lands in a
	// comment, where a newline would end the comment and let the rest become
	// document structure, so strip control characters first (§4).
	d.raw("# Generated by `solace-util convert` from " + commentSafe(source) + ".")
	d.raw("# Review before use: secrets are carried over verbatim, and any value the")
	d.raw("# bash bootstrap defaulted rather than the env file setting is absent here")
	d.raw("# (the Go defaults apply instead). See env/sample.yaml for the full schema.")
	d.b.WriteString("\n")

	// The warning has to be appended outside the emit branch: the case worth
	// telling the user about is precisely the one that emits nothing.
	r, redWarns := redundancy(v, p)
	if r != "" {
		d.kv("redundancy", r)
	}
	warns = append(warns, redWarns...)
	// SOLBK_TZ was the container timezone; the YAML schema has one cross-platform
	// key, so it lands at the top level for every platform.
	d.kv("timezone", v.s("SOLBK_TZ"))

	d.section("image", func(d *doc) {
		d.kv("repo", v.s("SOLBK_IMAGE"))
		d.kv("tag", v.s("SOLBK_IMG_TAG"))
		d.kv("registry", v.s("IMAGEREPO_HOST"))
		d.kv("pullSecret", v.s("IMAGEREPO_SECRET"))
		d.kv("user", v.s("IMAGEREPO_USER"))
		d.kv("pass", v.s("IMAGEREPO_PASS"))
	})

	// SOLBK_USR_PASS was a flat "user=password" list with no access level, so the
	// converted users get the least-privileged one and a warning naming the choice --
	// the schema requires the level explicitly, and guessing "admin" here would hand
	// out rights the source never granted.
	converted := 0
	d.section("admin", func(d *doc) {
		// SOLBK_ADM_USER is read on every platform so it counts as mapped rather than
		// resurfacing in the unmapped list, but it is emitted only for the container
		// platforms: on Kubernetes the operator reads the fixed username_admin_password
		// key, so validateK8s refuses any other admin.user and carrying the value over
		// would convert a working bash file into YAML that will not load.
		if u := v.s("SOLBK_ADM_USER"); p.IsContainer() {
			d.kv("user", u)
		} else if u != "" && u != "admin" {
			warns = append(warns, `SOLBK_ADM_USER="`+u+`" was dropped: on Kubernetes the operator reads the `+
				`fixed username_admin_password key out of the credentials Secret, so the broker admin user `+
				`is always "admin"`)
		}
		d.kv("pass", v.s("SOLBK_ADM_PASS"))
		d.kv("monitorPass", v.s("SOLBK_MON_PASS"))
		d.block("additionalUsers", func(d *doc) {
			for i, entry := range v.l("SOLBK_USR_PASS") {
				user, pass, ok := strings.Cut(entry, "=")
				if !ok || user == "" {
					// The entry is reported by position, never by value: a malformed one
					// is most likely a bare password pasted without its "user=" prefix,
					// so the text itself is the secret (§3, no secret in a log line).
					warns = append(warns, fmt.Sprintf("SOLBK_USR_PASS entry %d is not user=password (value withheld), dropped", i))
					continue
				}
				d.raw("- username: " + scalar(user))
				d.raw("  accessLevel: none")
				d.raw("  password: " + scalar(pass))
				converted++
			}
		})
	})
	if converted > 0 {
		warns = append(warns, "SOLBK_USR_PASS became admin.additionalUsers with accessLevel: none -- "+
			"the bash flow set no access level; give each user the level it actually needs")
	}

	d.section("tls", func(d *doc) {
		d.kv("cert", v.s("SOLBK_TLS_CERT"))
		d.kv("certKey", v.s("SOLBK_TLS_CERTKEY"))
		d.list("cas", v.l("SOLBK_TLS_CERTCAS"))
		d.kv("serverSecret", v.s("SOLBK_SVR_SECRET"))
	})

	d.section("scaling", func(d *doc) {
		num(d, "maxConnections", "SOLBK_SCALING_MAXCONN")
		num(d, "maxQueueMessages", "SOLBK_SCALING_MAXQMSG")
		// One key now, two legacy names for it: the container bootstrap spelled the
		// spool size SOLBK_SPOOL_MAXUSAGE and the k8s one SOLBK_SCALING_MAXPOOL.
		// Read this platform's own first so a file carrying both converts to the
		// value its bootstrap actually used, and say so rather than picking in
		// silence.
		spoolVars := []string{"SOLBK_SPOOL_MAXUSAGE", "SOLBK_SCALING_MAXPOOL"}
		if p == config.K8s {
			spoolVars = []string{"SOLBK_SCALING_MAXPOOL", "SOLBK_SPOOL_MAXUSAGE"}
		}
		pick := spoolVars[1]
		if v.s(spoolVars[0]) != "" {
			pick = spoolVars[0]
			if v.s(spoolVars[1]) != "" {
				warns = append(warns, spoolVars[0]+" and "+spoolVars[1]+" are now the single key "+
					"scaling.maxSpoolUsageMB; "+spoolVars[0]+" was used, the other dropped")
			}
		}
		num(d, "maxSpoolUsageMB", pick)
		num(d, "maxKafkaBridge", "SOLBK_SCALING_MAXKAFKABRIDGE")
		num(d, "maxKafkaConnections", "SOLBK_SCALING_MAXKAFKACONN")
		num(d, "maxBridges", "SOLBK_SCALING_MAXBRIDGE")
		num(d, "maxSubscriptions", "SOLBK_SCALING_MAXSUB")
		num(d, "maxGuaranteedMsgMB", "SOLBK_SCALING_MAXGMSSIZE")
	})

	// The schema keeps this block, but nothing in the binary reads it yet -- so a
	// source that configured replication would otherwise convert into something that
	// looks supported and silently does nothing.
	if v.s("REPL_MATE") != "" || len(v.l("REPL_CONN_SSL")) > 0 || v.s("REPL_PSK") != "" {
		warns = append(warns, "REPL_MATE/REPL_CONN_SSL/REPL_PSK were converted into the replication: block, "+
			"but no command in this binary reads it yet -- configure data replication with "+
			"solace-replication-generator.html or a `config exec-cli` script")
	}
	d.section("replication", func(d *doc) {
		d.kv("mate", v.s("REPL_MATE"))
		d.list("connSsl", v.l("REPL_CONN_SSL"))
		d.kv("psk", v.s("REPL_PSK"))
	})

	kube, kubeWarns := kubeCommand(v)
	warns = append(warns, kubeWarns...)

	// The k8s section carries the operator deployment plus four fields the
	// container platforms reuse (diagDir, cliScriptsFolder, domainCerts,
	// productKeys), so it is written for every platform.
	d.section("k8s", func(d *doc) {
		if p == config.K8s {
			d.kv("runtime", kube)
			d.kv("name", v.s("SOLBK_NAME"))
			d.kv("namespace", v.s("SOLBK_NS"))
			// SOLBK_USR_SECRET named the k8s Secret, so it lands in the k8s section
			// rather than under admin (where the old schema kept it).
			d.kv("adminSecret", v.s("SOLBK_USR_SECRET"))
			d.kv("updateStrategy", v.s("SOLBK_UPDATE_STRATEGY"))
			d.kv("serviceAccount", v.s("SOLBK_SVC_ACCOUNT"))
		}
		d.kv("cliScriptsFolder", v.s("SOLBK_CLISCRIPTS_FOLDER"))
		d.kv("diagDir", v.s("SOLBK_DIAG_DIR"))
		if p == config.K8s {
			d.block("storage", func(d *doc) {
				d.kv("class", v.s("SOLBK_STORAGECLASS"))
				d.kv("msgNode", v.s("SOLBK_STORAGE_MSGNODE"))
				d.kv("monNode", v.s("SOLBK_STORAGE_MONNODE"))
			})
			d.block("msgNode", func(d *doc) {
				// cpu was removed, so carrying the value over would only fail
				// validation later; it is dropped here with the reason named. The
				// read still happens so the variable counts as mapped rather than
				// resurfacing in the unmapped list.
				if v.s("SOLBK_MSGNODE_CPU") != "" {
					warns = append(warns, `SOLBK_MSGNODE_CPU is no longer supported: broker CPU is fixed by the scaling `+
						`tier and derived from scaling.maxConnections, so k8s.msgNode.cpu was omitted. `+
						`Check that SOLBK_SCALING_MAXCONN names the tier you sized for`)
				}
				d.kv("mem", v.s("SOLBK_MSGNODE_MEM"))
			})
			d.block("operator", func(d *doc) {
				d.kv("image", v.s("SOLOP_IMAGE"))
				d.kv("namespace", v.s("SOLOP_NS"))
				d.kv("watchNamespaces", v.s("SOLOP_WATCH_NS"))
				boolean(d, "watchBrokerNs", "SOLOP_WATCH_SOLBK_NS")
				d.kv("cpu", v.s("SOLOP_CPU"))
				d.kv("mem", v.s("SOLOP_MEM"))
			})
			d.block("placement", func(d *doc) {
				d.list("tolerationsPrimary", v.l("SOLBK_NODETOL_PRI"))
				d.list("tolerationsBackup", v.l("SOLBK_NODETOL_BKP"))
				d.list("tolerationsMonitor", v.l("SOLBK_NODETOL_MON"))
				d.list("labelsPrimary", v.l("SOLBK_NODELABEL_PRI"))
				d.list("labelsBackup", v.l("SOLBK_NODELABEL_BKP"))
				d.list("labelsMonitor", v.l("SOLBK_NODELABEL_MON"))
				d.list("antiAffinityNamespaces", v.l("SOLBK_ANTIAFFINITY_NS"))
				num(d, "antiAffinityWeight", "SOLBK_ANTIAFFINITY_WT")
			})
			d.block("loadBalancer", func(d *doc) {
				d.kv("ip", v.s("SOLBK_LOADBALANCER_IP"))
				d.list("annotations", v.l("SOLBK_LOADBALANCER_ANOTN"))
				d.kv("ipPool", v.s("SOLBK_IPPOOL"))
			})
			// SOLBK_PORTS means "name=port[/proto]" for k8s and "host:container"
			// for the container platforms, so it is read under exactly one of them.
			d.list("ports", v.l("SOLBK_PORTS"))
		}
		d.list("productKeys", v.l("SOLBK_PRODUCTKEYS"))
		d.block("domainCerts", func(d *doc) {
			d.kv("folder", v.s("SOLBK_DOMAINCERT_FOLDER"))
			d.pairs("files", v.m("SOLBK_DOMAINCERT_FILES"))
		})
	})

	if p.IsContainer() {
		d.section(string(p), func(d *doc) {
			d.kv("runtime", v.s("CONTAINER_RUNTIME"))
			if p == config.Docker {
				// run mode was removed, so carrying the value over would only fail
				// validation later; it is dropped here with the reason named.
				if strings.EqualFold(strings.TrimSpace(v.s("DOCKER_MODE")), "run") {
					warns = append(warns, `DOCKER_MODE="run" is no longer supported: docker deploys through compose only, `+
						`so docker.mode was omitted (it defaults to compose). Set docker.compose if this host uses the standalone docker-compose binary`)
				} else {
					d.kv("mode", v.s("DOCKER_MODE"))
				}
				d.kv("composeFile", v.s("DOCKER_COMPOSE_FILE"))
			} else {
				boolean(d, "rootless", "PODMAN_ROOTLESS")
				d.kv("quadletDir", v.s("QUADLET_DIR"))
			}
			d.block("network", func(d *doc) {
				d.kv("mode", v.s("SOLBK_NETWORK_MODE"))
				d.list("ports", v.l("SOLBK_PORTS"))
			})
			d.block("container", func(d *doc) {
				d.kv("name", v.s("CONTAINER_NAME"))
				d.kv("runUser", v.s("SOLBK_RUN_USER"))
				d.kv("shmSize", v.s("SOLBK_SHM_SIZE"))
				d.kv("dataDir", v.s("SOLBK_DATA_DIR"))
				d.block("ulimits", func(d *doc) {
					d.kv("nofile", v.s("SOLBK_ULIMIT_NOFILE"))
					d.kv("memlock", v.s("SOLBK_ULIMIT_MEMLOCK"))
					d.kv("core", v.s("SOLBK_ULIMIT_CORE"))
				})
			})
		})

		d.section("nodes", func(d *doc) {
			d.block("primary", func(d *doc) {
				d.kv("name", v.s("SOLBK_NODE_PRI_NAME"))
				d.kv("ip", v.s("SOLBK_NODE_PRI_IP"))
			})
			d.block("backup", func(d *doc) {
				d.kv("name", v.s("SOLBK_NODE_BKP_NAME"))
				d.kv("ip", v.s("SOLBK_NODE_BKP_IP"))
			})
			d.block("monitor", func(d *doc) {
				d.kv("name", v.s("SOLBK_NODE_MON_NAME"))
				d.kv("ip", v.s("SOLBK_NODE_MON_IP"))
			})
			d.kv("psk", v.s("SOLBK_REDUNDANCY_PSK"))
		})
	}
	return d.b.String(), warns
}

// kubeCommand resolves the bash KUBE variable to the k8s.runtime value. KUBE was
// the cluster CLI, expanded unquoted so it could carry a whole profile
// (`kubectl --kubeconfig <file>`, bash/env/customer-sample:7) rather than just a
// binary name -- exactly what k8s.runtime now holds.
//
// It is read unconditionally, even on a container platform, so the variable is
// consumed silently there instead of being reported unmapped.
//
// KUBE="echo" (bash/env/sample:7) was the bash dry-run trick. Carried over
// literally it would turn every cluster call into a no-op whose stdout the
// output-parsing steps then misread -- worse than dropping it -- so it yields a
// warning and no value, the --dry-run flag having replaced it. The match is
// exact: "/bin/echo" or "echo -n" passes through, a deliberate scope limit.
func kubeCommand(v *vars) (string, []string) {
	kube := strings.TrimSpace(v.s("KUBE"))
	if strings.EqualFold(kube, "echo") {
		return "", []string{`KUBE="echo" was the bash dry-run trick, dropped -- use --dry-run instead`}
	}
	return kube, nil
}

// redundancy normalises the two bash spellings: the k8s bootstrap wrote
// true/false, the container bootstrap yes/no. Anything else passes through
// unchanged so Validate rejects it loudly rather than the converter guessing.
//
// An unset value emits no key, so this CLI's own default (standalone) applies.
// That matches the legacy k8s bootstrap, which also defaulted to standalone, but
// it inverts the container bootstrap, which defaulted to HA -- so a container
// source with the variable unset gets that divergence called out rather than
// quietly converting a three-broker group into a single broker.
func redundancy(v *vars, p config.Platform) (string, []string) {
	raw := strings.TrimSpace(v.s("SOLBK_REDUNDANCY"))
	switch strings.ToLower(raw) {
	case "":
		if p.IsContainer() {
			return "", []string{"SOLBK_REDUNDANCY is unset: the container bootstrap defaulted it to yes (HA), " +
				"but this CLI defaults to standalone -- set `redundancy: yes` in the output if this host is part of a redundancy group"}
		}
		return "", nil
	case "yes", "true":
		return "yes", nil
	case "no", "false":
		return "no", nil
	}
	return raw, []string{fmt.Sprintf("SOLBK_REDUNDANCY=%q is neither yes/true nor no/false; copied as-is", raw)}
}

// boolOf parses the true/false spellings the bootstraps used.
func boolOf(s string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true", "yes":
		return true, true
	case "false", "no":
		return false, true
	}
	return false, false
}

// --- YAML emitter -----------------------------------------------------------

// doc builds the YAML document. Every writer skips an empty value, and a block
// whose body came out empty is omitted entirely, so the output carries only what
// the source file actually set.
type doc struct {
	b      strings.Builder
	indent int
}

func (d *doc) pad() string { return strings.Repeat("  ", d.indent) }

func (d *doc) raw(line string) { d.b.WriteString(d.pad() + line + "\n") }

func (d *doc) kv(key, val string) {
	if val == "" {
		return
	}
	d.raw(key + ": " + scalar(val))
}

func (d *doc) list(key string, vals []string) {
	if len(vals) == 0 {
		return
	}
	d.raw(key + ":")
	for _, val := range vals {
		d.raw("  - " + scalar(val))
	}
}

func (d *doc) pairs(key string, m map[string]string) {
	if len(m) == 0 {
		return
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys) // deterministic output from a Go map
	d.raw(key + ":")
	for _, k := range keys {
		d.raw("  " + scalar(k) + ": " + scalar(m[k]))
	}
}

// block writes a nested mapping, or nothing when it has no content.
func (d *doc) block(key string, fill func(*doc)) {
	sub := &doc{indent: d.indent + 1}
	fill(sub)
	if sub.b.Len() == 0 {
		return
	}
	d.raw(key + ":")
	d.b.WriteString(sub.b.String())
}

// section is a top-level block, separated from what precedes it by a blank line.
func (d *doc) section(key string, fill func(*doc)) {
	sub := &doc{indent: d.indent + 1}
	fill(sub)
	if sub.b.Len() == 0 {
		return
	}
	if s := d.b.String(); s != "" && !strings.HasSuffix(s, "\n\n") {
		d.b.WriteString("\n")
	}
	d.raw(key + ":")
	d.b.WriteString(sub.b.String())
}

// bareRE matches the values safe to emit unquoted: a letter, then letters,
// digits, and the separators that carry no YAML meaning.
var bareRE = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_.-]*$`)

// ambiguous are the plain scalars a YAML reader could take for a bool or null.
var ambiguous = map[string]bool{
	"y": true, "n": true, "yes": true, "no": true, "true": true, "false": true,
	"on": true, "off": true, "null": true, "nil": true,
}

// commentSafe makes a value safe to embed in a YAML comment line by replacing
// every control character with a space, so nothing can break out of the comment.
func commentSafe(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return ' '
		}
		return r
	}, s)
}

// scalar renders a string value, quoting anything a YAML reader could misread as
// a number, bool, null, comment, or structure.
func scalar(s string) string {
	if bareRE.MatchString(s) && !ambiguous[strings.ToLower(s)] {
		return s
	}
	esc := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`, "\t", `\t`, "\r", `\r`)
	return `"` + esc.Replace(s) + `"`
}
