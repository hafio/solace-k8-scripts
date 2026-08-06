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
			name: "docker run args primary",
			file: "docker_run_args_primary.golden",
			gen: func(t *testing.T) []byte {
				c := load(t, config.Docker)
				return []byte(strings.Join(RunArgs(c, c.ResolveNode(config.Primary)), "\n") + "\n")
			},
		},
		{
			name: "container env-pairs primary HA",
			file: "container_envpairs_primary.golden",
			gen: func(t *testing.T) []byte {
				c := load(t, config.Podman)
				return envLines(EnvPairs(c, config.Podman, c.ResolveNode(config.Primary)))
			},
		},
		{
			name: "container env-pairs standalone",
			file: "container_envpairs_standalone.golden",
			gen: func(t *testing.T) []byte {
				c := load(t, config.Podman)
				c.Redundancy = "no" // exercise the standalone branch without a second fixture
				return envLines(EnvPairs(c, config.Podman, c.ResolveNode(config.Primary)))
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
