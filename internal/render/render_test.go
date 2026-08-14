package render

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"solace/internal/config"
)

// -update regenerates the testdata goldens from the current renderers. Run it
// only after eyeballing the diff -- the committed goldens are the reviewed
// expected output, so `go test ./internal/render` guards against regressions.
var update = flag.Bool("update", false, "regenerate golden files in testdata/")

// sampleFixture is the shared env file every golden renders from: the user
// template doubles as the golden fixture (one fixture, all three platforms).
const sampleFixture = "../../env/sample.yaml"

func load(t *testing.T, p config.Platform) *config.Config {
	t.Helper()
	c, err := config.Load(sampleFixture, p)
	if err != nil {
		t.Fatalf("load %s under %s: %v", sampleFixture, p, err)
	}
	return c
}

// healthCheckFixture enables the health check with no cmd, so the goldens show the
// built-in readiness probe, and with the timings ApplyDefaults fills. Shared by the
// quadlet and compose cases so the two goldens differ only in framing.
func healthCheckFixture() config.HealthCheck {
	return config.HealthCheck{
		Enabled:     true,
		Interval:    "5s",
		Timeout:     "5s",
		Retries:     3,
		StartPeriod: "60s",
	}
}

// modernTag is a broker release new enough for the built-in readiness probe, which
// config.Validate requires before the probe can be enabled. The health-check cases
// pin it rather than relying on the sample's tag: env/sample.yaml is a template
// whose tag is meant to be edited, and these two cases would render a config
// Validate rejects if it were ever set below 10.26. It is why those goldens carry a
// different image tag from every other one.
const modernTag = "10.26.0.5"

func envLines(pairs []EnvPair) []byte {
	var b strings.Builder
	for _, p := range pairs {
		b.WriteString(p.Assignment())
		b.WriteByte('\n')
	}
	return []byte(b.String())
}

func TestGolden(t *testing.T) {
	cases := []struct {
		name string
		file string
		gen  func(t *testing.T) []byte
	}{
		{
			name: "k8s broker CR",
			file: "k8s_broker_cr.golden",
			gen:  func(t *testing.T) []byte { return BrokerCR(load(t, config.K8s)) },
		},
		{
			// The sample leaves k8s.ports commented, so the case above renders the
			// 16 defaults. An explicit list here covers the other branch of
			// ApplyDefaults' port handling, and with it the two forms only a custom
			// list uses: a container port differing from the service port, and an
			// explicit protocol.
			name: "k8s broker CR with custom ports",
			file: "k8s_broker_cr_ports.golden",
			gen: func(t *testing.T) []byte {
				c := load(t, config.K8s)
				c.K8s.Ports = []string{"tcp-semp=8080", "tcp-smf=55555:55556", "tls-smf=55443/TCP"}
				return BrokerCR(c)
			},
		},
		{
			// Timezone and both security blocks are optional and omitted by
			// default, so the cases above prove they stay out of the CR. This one
			// proves they land correctly when set.
			name: "k8s broker CR with timezone and security context",
			file: "k8s_broker_cr_security.golden",
			gen: func(t *testing.T) []byte {
				c := load(t, config.K8s)
				c.Timezone = "Asia/Singapore"
				c.K8s.SecurityContext = config.PodSecurity{RunAsUser: "1000001", FSGroup: "1000002"}
				readOnly := false
				c.K8s.ContainerSecurity = config.ContainerSecurity{
					RunAsUser:              "1000001",
					RunAsGroup:             "1000002",
					ReadOnlyRootFilesystem: &readOnly,
				}
				return BrokerCR(c)
			},
		},
		{
			// The CR knobs that used to be hardcoded or inexpressible. pullPolicy
			// replaces a literal; podAnnotations/podLabels are new optional blocks,
			// so the cases above prove they stay out when unset. The values here
			// carry a colon and a quote to exercise the escaping.
			name: "k8s broker CR with pull policy and pod metadata",
			file: "k8s_broker_cr_podmeta.golden",
			gen: func(t *testing.T) []byte {
				c := load(t, config.K8s)
				c.Image.PullPolicy = "Always"
				c.K8s.PodAnnotations = map[string]string{
					"prometheus.io/scrape": "true",
					"example.com/note":     `a: "quoted" value`,
				}
				c.K8s.PodLabels = map[string]string{"example.com/tier": "messaging"}
				return BrokerCR(c)
			},
		},
		{
			// The additive affinity blocks alongside the legacy anti-affinity term:
			// the fixed broker-spread term must still come first, unchanged, with the
			// configured terms after it.
			name: "k8s broker CR with node and pod affinity",
			file: "k8s_broker_cr_affinity.golden",
			gen: func(t *testing.T) []byte {
				c := load(t, config.K8s)
				c.K8s.Placement.NodeAffinity = config.NodeAffinity{
					Preferred: []config.WeightedNodeTerm{{
						Weight: 80,
						Match: []config.NodeMatchExpr{
							{Key: "topology.kubernetes.io/zone", Operator: "In", Values: []string{"az-1", "az-2"}},
						},
					}},
					Required: []config.NodeMatchExpr{
						{Key: "solace.com/broker", Operator: "Exists"},
						{Key: "kubernetes.io/arch", Operator: "NotIn", Values: []string{"arm64"}},
					},
				}
				c.K8s.Placement.PodAffinity = []config.PodAffinityTerm{{
					Weight:      20,
					TopologyKey: "topology.kubernetes.io/zone",
					MatchLabels: map[string]string{"app": "gateway"},
					Namespaces:  []string{"edge"},
				}}
				c.K8s.Placement.PodAntiAffinity = []config.PodAffinityTerm{{
					TopologyKey: "topology.kubernetes.io/zone",
					MatchLabels: map[string]string{"app.kubernetes.io/name": "pubsubpluseventbroker"},
				}}
				return BrokerCR(c)
			},
		},
		{
			// loadBalancer.annotations and placement.labels* are user-supplied
			// "key: value" fragments that used to be pasted into the manifest
			// verbatim. The values here carry a colon, a slash and a URL, which only
			// stay intact because both halves are quoted. This is also the only case
			// covering nodeSelector and tolerations at all.
			name: "k8s broker CR with annotations and node labels",
			file: "k8s_broker_cr_labels.golden",
			gen: func(t *testing.T) []byte {
				c := load(t, config.K8s)
				c.K8s.LoadBalancer.Annotations = []string{
					"external-dns.alpha.kubernetes.io/hostname: broker.example.com",
					"service.beta.kubernetes.io/target: https://lb.example.com:8443",
				}
				c.K8s.Placement.LabelsPrimary = []string{"nodetype: solace"}
				c.K8s.Placement.LabelsBackup = []string{"nodetype: solace"}
				c.K8s.Placement.LabelsMonitor = []string{"nodetype: solace"}
				c.K8s.Placement.TolerationsPrimary = []string{"dedicated=solace:NoSchedule"}
				c.K8s.Placement.TolerationsMonitor = []string{"dedicated:NoExecute"}
				return BrokerCR(c)
			},
		},
		{
			name: "podman quadlet primary",
			file: "podman_quadlet_primary.golden",
			gen: func(t *testing.T) []byte {
				c := load(t, config.Podman)
				return Quadlet(c, c.ResolveNode(config.Primary))
			},
		},
		{
			name: "docker compose primary",
			file: "docker_compose_primary.golden",
			gen: func(t *testing.T) []byte {
				c := load(t, config.Docker)
				return Compose(c, c.ResolveNode(config.Primary))
			},
		},
		{
			// The health check is opt-in, so every other container case proves the
			// artifacts stay unchanged without it; this one proves both framings.
			name: "podman quadlet with health check",
			file: "podman_quadlet_healthcheck.golden",
			gen: func(t *testing.T) []byte {
				c := load(t, config.Podman)
				c.Image.Tag = modernTag
				c.Podman.Container.HealthCheck = healthCheckFixture()
				return Quadlet(c, c.ResolveNode(config.Primary))
			},
		},
		{
			name: "docker compose with health check",
			file: "docker_compose_healthcheck.golden",
			gen: func(t *testing.T) []byte {
				c := load(t, config.Docker)
				c.Image.Tag = modernTag
				c.Docker.Container.HealthCheck = healthCheckFixture()
				return Compose(c, c.ResolveNode(config.Primary))
			},
		},
		{
			// Standalone drops the whole redundancy block from the compose file,
			// including its secret reference -- one secret instead of two.
			name: "docker compose standalone",
			file: "docker_compose_standalone.golden",
			gen: func(t *testing.T) []byte {
				c := load(t, config.Docker)
				c.Redundancy = "no"
				return Compose(c, c.ResolveNode(config.Primary))
			},
		},
		{
			// The sample sets no tz, so this covers the omitted-TZ branch; the
			// standalone case below sets one and covers the other.
			name: "container env-pairs primary HA",
			file: "container_envpairs_primary.golden",
			gen: func(t *testing.T) []byte {
				c := load(t, config.Podman)
				return envLines(EnvPairs(c, c.ResolveNode(config.Primary)))
			},
		},
		{
			name: "container env-pairs standalone",
			file: "container_envpairs_standalone.golden",
			gen: func(t *testing.T) []byte {
				c := load(t, config.Podman)
				// Exercise the standalone and TZ-present branches without a second fixture.
				c.Redundancy = "no"
				c.Timezone = "Asia/Singapore"
				return envLines(EnvPairs(c, c.ResolveNode(config.Primary)))
			},
		},
		{
			// The env file is the same pairs in env-file framing, and the whole
			// point is that it is printable -- so the golden also pins the absence
			// of the admin password and the PSK.
			name: "container env file primary HA",
			file: "container_envfile_primary.golden",
			gen: func(t *testing.T) []byte {
				c := load(t, config.Podman)
				return EnvFile(c, c.ResolveNode(config.Primary))
			},
		},
		{
			name: "podman secret script HA",
			file: "podman_secret_script.golden",
			gen:  func(t *testing.T) []byte { return SecretScript(load(t, config.Podman), config.Podman) },
		},
		{
			// Docker's script exports the variables compose reads instead, and a
			// value carrying a quote proves shQuote keeps the line intact.
			name: "docker secret script HA",
			file: "docker_secret_script.golden",
			gen: func(t *testing.T) []byte {
				c := load(t, config.Docker)
				c.Admin.Pass = `pa'ss "w" $ord`
				return SecretScript(c, config.Docker)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.gen(t)
			golden := filepath.Join("testdata", tc.file)
			if *update {
				if err := os.WriteFile(golden, got, 0o644); err != nil {
					t.Fatalf("write golden %s: %v", golden, err)
				}
				return
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("read golden %s (regenerate: go test ./internal/render -update): %v", golden, err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("%s mismatch\n--- got ---\n%s\n--- want ---\n%s", tc.file, got, want)
			}
		})
	}
}

// TestArtifactsCarryNoSecrets is the regression guard behind the secret
// externalization: every deployment artifact must reference the admin password
// and the redundancy pre-shared key by name and never carry their values, so
// `--gen-only` output is safe to share. Distinctive values make a leak
// unmistakable -- the goldens alone would not catch one reintroduced together
// with a regenerated golden.
func TestArtifactsCarryNoSecrets(t *testing.T) {
	const (
		pass     = "UNIQUE-ADMIN-PASSWORD-VALUE"
		psk      = "UNIQUE-PRESHARED-KEY-VALUE"
		userPass = "UNIQUE-EXTRA-USER-PASSWORD"
	)
	secrets := []string{pass, psk, userPass}

	for _, p := range []config.Platform{config.K8s, config.Docker, config.Podman} {
		c := load(t, p)
		c.Admin.Pass = pass
		c.Nodes.PSK = psk
		c.Admin.AdditionalUsers = []config.AdditionalUser{
			{Username: "appuser", AccessLevel: "read-only", Password: userPass},
		}
		id := c.ResolveNode(config.Primary)

		artifacts := map[string][]byte{}
		switch p {
		case config.K8s:
			artifacts["broker CR"] = BrokerCR(c)
		case config.Podman:
			artifacts["quadlet"] = Quadlet(c, id)
			artifacts["env file"] = EnvFile(c, id)
		default:
			artifacts["compose"] = Compose(c, id)
			artifacts["env file"] = EnvFile(c, id)
		}
		for name, body := range artifacts {
			for _, secret := range secrets {
				if bytes.Contains(body, []byte(secret)) {
					t.Errorf("%s %s artifact carries the secret %q; it must reference it by name instead", p, name, secret)
				}
			}
		}
		if p == config.K8s {
			continue
		}
		// SecretScript is the one renderer that must carry the values: it is what
		// creates the secrets, and only --gen-secrets-only prints it.
		script := SecretScript(c, p)
		for _, secret := range secrets {
			if !bytes.Contains(script, []byte(secret)) {
				t.Errorf("%s secret script is missing %q; it is what creates the secrets", p, secret)
			}
		}
	}
}

// TestContainerSecretsRedundancy pins which secrets exist per mode: standalone has
// no mate link, so the PSK secret must not be referenced at all there.
func TestContainerSecretsRedundancy(t *testing.T) {
	c := load(t, config.Podman)

	ha := ContainerSecrets(c, config.Podman)
	if len(ha) != 2 {
		t.Fatalf("HA secrets = %d, want 2 (admin password + PSK)", len(ha))
	}
	if ha[0].EnvKey != "username_admin_password" || ha[1].EnvKey != "redundancy_authentication_presharedkey_key" {
		t.Errorf("HA secret env keys = %q, %q", ha[0].EnvKey, ha[1].EnvKey)
	}
	if got := ha[0].FilePathKey(); got != "username_admin_passwordfilepath" {
		t.Errorf("FilePathKey = %q", got)
	}
	// The mount is named after the setting, not the host-side secret, so the layout
	// inside the container matches the k8s Secret's data keys.
	if got := ha[0].MountPath(); got != "/run/secrets/username_admin_password" {
		t.Errorf("MountPath = %q", got)
	}

	c.Redundancy = "no"
	if standalone := ContainerSecrets(c, config.Podman); len(standalone) != 1 {
		t.Errorf("standalone secrets = %d, want 1 (admin password only)", len(standalone))
	}

	// An encrypted server-certificate key adds a third secret, and only then: an
	// empty passphrase must not produce one the broker would use to unlock a plain
	// key. The broker reads it from the mounted file via the *filepath variant.
	c.TLS.CertPassphrase = "cert-pass"
	withPass := ContainerSecrets(c, config.Podman)
	if len(withPass) != 2 {
		t.Fatalf("standalone + passphrase = %d secrets, want 2", len(withPass))
	}
	pass := withPass[1]
	if pass.EnvKey != "tls_servercertificate_passphrase" {
		t.Errorf("passphrase env key = %q", pass.EnvKey)
	}
	if got := pass.FilePathKey(); got != "tls_servercertificate_passphrasefilepath" {
		t.Errorf("passphrase file-path key = %q", got)
	}
}

// TestContainerSecretNamesAreHostScoped pins the de-confliction: the engine-side
// name carries the container name, so two brokers on one host never share a podman
// store entry or a compose variable -- while the in-container filename stays the
// same everywhere. The default name keeps the historical names byte-for-byte.
func TestContainerSecretNamesAreHostScoped(t *testing.T) {
	c := load(t, config.Docker)
	if got := ContainerSecrets(c, config.Docker)[0].Name; got != "solace-admin-password" {
		t.Errorf("with the default container name the secret must stay %q, got %q", "solace-admin-password", got)
	}

	c.Docker.Container.Name = "edge-2.broker"
	s := ContainerSecrets(c, config.Docker)[0]
	if s.Name != "edge-2.broker-admin-password" {
		t.Errorf("secret name = %q, want it prefixed with the container name", s.Name)
	}
	if s.Target() != "username_admin_password" || s.MountPath() != "/run/secrets/username_admin_password" {
		t.Errorf("the in-container name must not carry the host prefix: target=%q path=%q", s.Target(), s.MountPath())
	}
	// '.' and '-' cannot appear in a variable name; a leading digit cannot start one.
	if got := s.EnvVar(); got != "EDGE_2_BROKER_ADMIN_PASSWORD" {
		t.Errorf("EnvVar = %q", got)
	}
	c.Docker.Container.Name = "9lives"
	if got := ContainerSecrets(c, config.Docker)[0].EnvVar(); got != "_9LIVES_ADMIN_PASSWORD" {
		t.Errorf("a name starting with a digit must be prefixed to stay exportable, got %q", got)
	}
}

// TestAdditionalUsersReachBothHalves pins the extra-user wiring: the password is a
// secret named per host and mounted under the setting it feeds, while the access
// level is not a secret and rides the artifact as an ordinary pair.
func TestAdditionalUsersReachBothHalves(t *testing.T) {
	c := load(t, config.Docker)
	c.Redundancy = "no"
	c.Admin.AdditionalUsers = []config.AdditionalUser{
		{Username: "appuser", AccessLevel: "read-write", Password: "app-secret"},
	}

	secrets := ContainerSecrets(c, config.Docker)
	if len(secrets) != 2 {
		t.Fatalf("secrets = %d, want 2 (admin + appuser)", len(secrets))
	}
	u := secrets[1]
	if u.Name != "solace-user-appuser-password" || u.EnvKey != "username_appuser_password" {
		t.Errorf("additional-user secret = %+v", u)
	}
	if u.ConfigKey != "admin.additionalUsers.appuser.password" {
		t.Errorf("ConfigKey = %q; it must name the env-file key for an actionable error", u.ConfigKey)
	}

	pairs := envLines(EnvPairs(c, c.ResolveNode(config.Primary)))
	for _, want := range []string{
		"username_appuser_globalaccesslevel=read-write",
		"username_appuser_passwordfilepath=/run/secrets/username_appuser_password",
	} {
		if !strings.Contains(string(pairs), want) {
			t.Errorf("env pairs should contain %q:\n%s", want, pairs)
		}
	}
	if strings.Contains(string(pairs), "app-secret") {
		t.Errorf("env pairs must not carry the password:\n%s", pairs)
	}
}

// TestShQuote guards the secret-script quoting: a value with a single quote must
// survive as itself rather than ending the shell string.
func TestShQuote(t *testing.T) {
	if got := shQuote(`pa'ss`); got != `'pa'\''ss'` {
		t.Errorf("shQuote = %q", got)
	}
}

// TestQuadletHealthCmdEscapesPercent covers the specifier trap: systemd expands
// %-specifiers in every assignment in a unit file, not only the quoted
// Environment= ones, so a percent-encoded character in a probe URL has to be
// doubled or systemd fails to resolve it and drops the line -- silently disabling
// the health check. Quotes and backslashes must NOT be escaped here: the value is
// unquoted and podman splits the command line itself.
func TestQuadletHealthCmdEscapesPercent(t *testing.T) {
	c := load(t, config.Podman)
	hc := healthCheckFixture()
	// An explicit probe, which is also what an older broker would have to use.
	hc.Cmd = []string{"curl", "-fs", `http://localhost:5550/health?q=a%20b&s="x"`}
	c.Podman.Container.HealthCheck = hc
	got := string(Quadlet(c, c.ResolveNode(config.Primary)))

	if !strings.Contains(got, `HealthCmd=curl -fs http://localhost:5550/health?q=a%%20b&s="x"`) {
		t.Errorf("HealthCmd must double %% and leave quotes alone:\n%s", got)
	}
	// Compose has no specifier expansion, so the same probe must stay literal there:
	// doubling the percent would change the URL the broker is actually polled with.
	d := load(t, config.Docker)
	d.Docker.Container.HealthCheck = hc
	compose := string(Compose(d, d.ResolveNode(config.Primary)))
	if !strings.Contains(compose, `q=a%20b`) {
		t.Errorf("compose must keep the percent single:\n%s", compose)
	}
	if strings.Contains(compose, `%%`) {
		t.Errorf("compose must not apply systemd's percent doubling:\n%s", compose)
	}
}

// TestHealthCmdDefaultsToReadiness pins the built-in probe: an enabled block with
// no cmd polls the broker's own readiness endpoint, and an explicit cmd wins.
func TestHealthCmdDefaultsToReadiness(t *testing.T) {
	got := healthCmd(config.HealthCheck{Enabled: true})
	want := []string{"curl", "-fs", "http://localhost:5550/health-check/readiness"}
	if len(got) != len(want) {
		t.Fatalf("default probe = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("default probe = %v, want %v", got, want)
			break
		}
	}
	custom := []string{"/opt/probe.sh"}
	if only := healthCmd(config.HealthCheck{Enabled: true, Cmd: custom}); only[0] != custom[0] || len(only) != 1 {
		t.Errorf("an explicit cmd must win, got %v", only)
	}
}

// TestSecretPreflight pins the precondition `deploy` and `--gen-secrets-only`
// share: creating a secret with an empty value leaves the broker with a blank
// password or mate-link key that only fails later, so it is refused up front.
func TestSecretPreflight(t *testing.T) {
	c := load(t, config.Podman) // HA sample: admin password and PSK both set
	if err := SecretPreflight(c, config.Podman); err != nil {
		t.Fatalf("a fully configured deployment must pass preflight: %v", err)
	}

	c.Nodes.PSK = ""
	err := SecretPreflight(c, config.Podman)
	if err == nil {
		t.Fatal("an empty PSK must be refused before a secret is created from it")
	}
	if !strings.Contains(err.Error(), "nodes.psk") || !strings.Contains(err.Error(), "prep host") {
		t.Errorf("the error should name the field and the fix, got: %v", err)
	}

	// Standalone has no mate link, so an empty PSK is not a secret at all there.
	c.Redundancy = "no"
	if err := SecretPreflight(c, config.Podman); err != nil {
		t.Errorf("standalone does not need a PSK: %v", err)
	}

	c.Admin.Pass = ""
	if err := SecretPreflight(c, config.Podman); err == nil || !strings.Contains(err.Error(), "admin.pass") {
		t.Errorf("an empty admin password must be refused, got: %v", err)
	}
}

// TestParsePort covers the port-entry parsing branches independently of the CR.
func TestParsePort(t *testing.T) {
	cases := []struct {
		in                              string
		name, container, service, proto string
	}{
		{"tcp-semp=8080", "tcp-semp", "8080", "8080", "TCP"},
		{"tcp-smf=55555:55556", "tcp-smf", "55555", "55556", "TCP"},
		{"tls-smf=55443/TCP", "tls-smf", "55443", "55443", "TCP"},
		{"udp-x=1234:5678/UDP", "udp-x", "1234", "5678", "UDP"},
	}
	for _, tc := range cases {
		got := parsePort(tc.in)
		if got.name != tc.name || got.container != tc.container || got.service != tc.service || got.proto != tc.proto {
			t.Errorf("parsePort(%q) = %+v, want {%s %s %s %s}",
				tc.in, got, tc.name, tc.container, tc.service, tc.proto)
		}
	}
}

// TestParseToleration covers Equal (key=value) vs Exists (bare key) forms.
func TestParseToleration(t *testing.T) {
	key, val, eff, equal := parseToleration("dedicated=solace:NoSchedule")
	if key != "dedicated" || val != "solace" || eff != "NoSchedule" || !equal {
		t.Errorf("Equal form: got (%q,%q,%q,%v)", key, val, eff, equal)
	}
	key, val, eff, equal = parseToleration("dedicated:NoExecute")
	if key != "dedicated" || val != "" || eff != "NoExecute" || equal {
		t.Errorf("Exists form: got (%q,%q,%q,%v)", key, val, eff, equal)
	}
}

// TestQuadletEscape guards the systemd Environment=" " escaping.
func TestQuadletEscape(t *testing.T) {
	if got := quadletEscape(`a%b"c\d`); got != `a%%b\"c\\d` {
		t.Errorf("quadletEscape = %q", got)
	}
}
