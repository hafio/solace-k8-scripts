package config

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// decodeRuntime runs a document through the same strict decoder Load uses, so
// these cases exercise the real schema path rather than a bare Command.
func decodeRuntime(t *testing.T, doc string) Command {
	t.Helper()
	var c Config
	dec := yaml.NewDecoder(strings.NewReader(doc))
	dec.KnownFields(true)
	if err := dec.Decode(&c); err != nil {
		t.Fatalf("decode %q: %v", doc, err)
	}
	return c.K8s.Runtime
}

// TestCommandUnmarshal pins both accepted forms. The scalar form reproduces the
// bash bootstraps' unquoted word splitting; the sequence form is the only way to
// express a token that itself contains a space.
func TestCommandUnmarshal(t *testing.T) {
	cases := []struct {
		name string
		doc  string
		want []string
	}{
		{"scalar binary", "k8s:\n  runtime: kubectl\n", []string{"kubectl"}},
		{"scalar drop-in", "k8s:\n  runtime: oc\n", []string{"oc"}},
		{"scalar wrapper", "k8s:\n  runtime: microk8s kubectl\n", []string{"microk8s", "kubectl"}},
		{"scalar profile", "k8s:\n  runtime: kubectl --kubeconfig /tmp/kc\n",
			[]string{"kubectl", "--kubeconfig", "/tmp/kc"}},
		// Quoting the scalar groups it for YAML, not for the split: bash did not
		// honour embedded quotes when word splitting either.
		{"quoted scalar still splits", "k8s:\n  runtime: \"kubectl --context=dev\"\n",
			[]string{"kubectl", "--context=dev"}},
		{"whitespace runs collapse", "k8s:\n  runtime: \"  oc   version  \"\n", []string{"oc", "version"}},
		{"empty scalar", "k8s:\n  runtime: \"\"\n", nil},
		{"omitted", "k8s:\n  name: dev-broker\n", nil},
		{"flow sequence", "k8s:\n  runtime: [microk8s, kubectl]\n", []string{"microk8s", "kubectl"}},
		{"block sequence", "k8s:\n  runtime:\n    - kubectl\n    - --context=dev\n",
			[]string{"kubectl", "--context=dev"}},
		// The escape hatch the scalar form cannot express.
		{"sequence keeps embedded spaces",
			"k8s:\n  runtime:\n    - 'C:\\Program Files\\bin\\kubectl.exe'\n    - --context=dev\n",
			[]string{`C:\Program Files\bin\kubectl.exe`, "--context=dev"}},
		{"empty sequence", "k8s:\n  runtime: []\n", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := decodeRuntime(t, tc.doc)
			if len(got) != len(tc.want) {
				t.Fatalf("runtime = %#v, want %#v", []string(got), tc.want)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Errorf("runtime[%d] = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

// TestCommandUnmarshalRejectsOtherKinds: a mapping is neither a command line nor
// an argv, so it fails loud at decode rather than exec'ing an empty binary.
func TestCommandUnmarshalRejectsOtherKinds(t *testing.T) {
	var c Config
	err := yaml.Unmarshal([]byte("k8s:\n  runtime:\n    bin: kubectl\n"), &c)
	if err == nil {
		t.Fatal("a mapping should not decode as a command")
	}
	if !strings.Contains(err.Error(), "must be a string") {
		t.Errorf("error = %v, want it to name the accepted forms", err)
	}
}

// TestCommandUnmarshalPropagatesDecodeErrors: a node of an accepted kind whose
// contents still will not decode must surface yaml's error rather than fall
// through to an empty command that later exec's nothing.
func TestCommandUnmarshalPropagatesDecodeErrors(t *testing.T) {
	cases := []struct {
		name, doc, want string
	}{
		// Any plain scalar decodes into a string, so the one failing case is a
		// tag that carries its own decoding: !!binary with invalid base64.
		{"scalar", "k8s:\n  runtime: !!binary \"*not base64*\"\n", "base64"},
		// A sequence element that is not a scalar cannot become an argument.
		{"sequence element", "k8s:\n  runtime:\n    - [nested]\n", "cannot unmarshal"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var c Config
			err := yaml.Unmarshal([]byte(tc.doc), &c)
			if err == nil {
				t.Fatalf("%s should not decode", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %v, want it to mention %q", err, tc.want)
			}
		})
	}
}

// TestValidateRejectsBadRuntime: a malformed command fails the load rather than
// reaching os/exec. validateCommand runs ahead of the platform checks, so the
// message names the runtime field, not the mandatory fields also missing here.
func TestValidateRejectsBadRuntime(t *testing.T) {
	cases := []struct {
		name  string
		p     Platform
		setup func(*Config)
		want  string
	}{
		{
			name:  "k8s runtime with a newline",
			p:     K8s,
			setup: func(c *Config) { c.K8s.Runtime = Command{"kubectl\n--all-namespaces"} },
			want:  "k8s.runtime[0] contains a control character",
		},
		{
			name:  "docker runtime with an empty argument",
			p:     Docker,
			setup: func(c *Config) { c.Docker.Runtime = Command{"docker", ""} },
			want:  "docker.runtime[1] is an empty argument",
		},
		{
			name:  "podman runtime with a NUL",
			p:     Podman,
			setup: func(c *Config) { c.Podman.Runtime = Command{"pod\x00man"} },
			want:  "podman.runtime[0] contains a control character",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := &Config{Redundancy: "yes"}
			c.ApplyDefaults(tc.p) // defaults first, so the override is what is tested
			tc.setup(c)
			err := c.Validate(tc.p)
			if err == nil {
				t.Fatal("a malformed runtime must fail validation")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %v, want it to contain %q", err, tc.want)
			}
		})
	}
}

func TestCommandNameAndArgs(t *testing.T) {
	cases := []struct {
		name     string
		cmd      Command
		wantName string
		wantArgs string
	}{
		{"unset", nil, "", "get pods"},
		{"bare binary", Command{"kubectl"}, "kubectl", "get pods"},
		{"wrapper", Command{"microk8s", "kubectl"}, "microk8s", "kubectl get pods"},
		{"profile", Command{"kubectl", "--kubeconfig", "/tmp/kc"}, "kubectl", "--kubeconfig /tmp/kc get pods"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.cmd.Name(); got != tc.wantName {
				t.Errorf("Name() = %q, want %q", got, tc.wantName)
			}
			if got := strings.Join(tc.cmd.Args("get", "pods"), " "); got != tc.wantArgs {
				t.Errorf("Args() = %q, want %q", got, tc.wantArgs)
			}
		})
	}
}

// TestCommandArgsDoesNotAliasCommand guards the aliasing trap: a Command is a
// long-lived config value reused by every call, so building argv with
// append(cmd[1:], extra...) would write into its backing array whenever there is
// spare capacity -- corrupting the previous call's argv, silently and only
// sometimes. Args must allocate.
func TestCommandArgsDoesNotAliasCommand(t *testing.T) {
	// Spare capacity is the shape that makes a naive append destructive.
	base := make([]string, 0, 8)
	base = append(base, "microk8s", "kubectl")
	cmd := Command(base)

	first := cmd.Args("get", "pods")
	second := cmd.Args("delete", "ns")

	if got := strings.Join(first, " "); got != "kubectl get pods" {
		t.Errorf("the second call overwrote the first: %q", got)
	}
	if got := strings.Join(second, " "); got != "kubectl delete ns" {
		t.Errorf("second Args() = %q", got)
	}
	if got := cmd.String(); got != "microk8s kubectl" {
		t.Errorf("Args() mutated the Command itself: %q", got)
	}
}

func TestCommandString(t *testing.T) {
	cases := []struct {
		cmd  Command
		want string
	}{
		{nil, ""},
		{Command{"kubectl"}, "kubectl"},
		{Command{"microk8s", "kubectl"}, "microk8s kubectl"},
	}
	for _, tc := range cases {
		if got := tc.cmd.String(); got != tc.want {
			t.Errorf("String() = %q, want %q", got, tc.want)
		}
	}
}

func TestValidateCommandAccepts(t *testing.T) {
	cases := []struct {
		name string
		cmd  Command
	}{
		{"binary", Command{"kubectl"}},
		{"wrapper", Command{"microk8s", "kubectl"}},
		{"path with spaces", Command{`C:\Program Files\bin\kubectl.exe`}},
		// Metacharacters are inert: exec never goes through a shell, so these are
		// ordinary filename characters, not an injection.
		{"metacharacters", Command{"kubectl;rm -rf /"}},
		// Unset is not an error -- ApplyDefaults fills it before Validate runs.
		{"unset", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateCommand("k8s.runtime", tc.cmd); err != nil {
				t.Errorf("validateCommand(%q) = %v, want accepted", tc.cmd, err)
			}
		})
	}
}

func TestValidateCommandRejects(t *testing.T) {
	cases := []struct {
		name string
		cmd  Command
		want string
	}{
		{"empty argument", Command{"kubectl", ""}, "k8s.runtime[1] is an empty argument"},
		{"newline", Command{"kubectl\n--all"}, "k8s.runtime[0] contains a control character"},
		{"NUL", Command{"kube\x00ctl"}, "k8s.runtime[0] contains a control character"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateCommand("k8s.runtime", tc.cmd)
			if err == nil {
				t.Fatalf("validateCommand(%q) accepted a value it must reject", tc.cmd)
			}
			// The message names the field and the offending index, so the user
			// can find it in the env file (§4a).
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %v, want it to contain %q", err, tc.want)
			}
		})
	}
}

// TestRuntimeDefaults: the defaults must reproduce today's argv exactly -- one
// token, no leading arguments -- so every existing `+ kubectl ...` / `+ docker`
// dry-run assertion keeps passing unchanged.
func TestRuntimeDefaults(t *testing.T) {
	cases := []struct {
		p    Platform
		get  func(*Config) Command
		want string
	}{
		{K8s, func(c *Config) Command { return c.K8s.Runtime }, "kubectl"},
		{Docker, func(c *Config) Command { return c.Docker.Runtime }, "docker"},
		{Podman, func(c *Config) Command { return c.Podman.Runtime }, "podman"},
		// k8s.runtime is defaulted on every platform, not just k8s.
		{Docker, func(c *Config) Command { return c.K8s.Runtime }, "kubectl"},
	}
	for _, tc := range cases {
		c := &Config{}
		c.ApplyDefaults(tc.p)
		if got := tc.get(c); got.String() != tc.want {
			t.Errorf("on platform %q: got %q, want %q", tc.p, got, tc.want)
		}
	}
}

// TestRuntimeExplicitValueSurvivesDefaults: an override must not be overwritten.
func TestRuntimeExplicitValueSurvivesDefaults(t *testing.T) {
	c := &Config{}
	c.K8s.Runtime = Command{"microk8s", "kubectl"}
	c.Docker.Runtime = Command{"sudo", "docker"}
	c.ApplyDefaults(Docker)

	if got := c.K8s.Runtime.String(); got != "microk8s kubectl" {
		t.Errorf("k8s.runtime = %q, want the configured value", got)
	}
	if got := c.Docker.Runtime.String(); got != "sudo docker" {
		t.Errorf("docker.runtime = %q, want the configured value", got)
	}
}
