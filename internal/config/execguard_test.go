package config

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// decodeStrict runs a document through the same strict decoder Load uses, so the
// schema tests below exercise the real load path. The sibling of command_test.go's
// decodeRuntime, which returns one field rather than the error.
func decodeStrict(doc string, c *Config) error {
	dec := yaml.NewDecoder(strings.NewReader(doc))
	dec.KnownFields(true)
	return dec.Decode(c)
}

// guardConfig builds a config that passes Validate on p with everything EXCEPT the
// command fields already correct, so a Validate failure in these tests can only
// have come from the execution guard. Defaults are applied first, exactly as
// config.Load does, so the runtime fields hold their real defaults unless a case
// overrides them.
func guardConfig(p Platform) *Config {
	c := &Config{Redundancy: "no"}
	c.Image.Repo = "solace/solace-pubsub-standard"
	c.Image.Tag = "10.10.1.35"
	c.Admin.Pass = "s3cret-not-a-real-password"
	c.K8s.Name = "broker"
	c.K8s.Namespace = "solace"
	c.K8s.Storage.MsgNode = "30Gi"
	c.Nodes.Primary.Name = "primary-host"
	c.ApplyDefaults(p)
	return c
}

// TestGuardConfigIsValid guards the fixture itself: every case below reads a
// Validate failure as "the guard rejected the command", which is only sound while
// the untouched fixture validates cleanly on all three platforms.
func TestGuardConfigIsValid(t *testing.T) {
	for _, p := range []Platform{K8s, Docker, Podman} {
		if err := guardConfig(p).Validate(p); err != nil {
			t.Errorf("guardConfig(%s) must validate cleanly before any command is overridden: %v", p, err)
		}
	}
}

// --- layers 1-3: what a command may be ---------------------------------------

// TestCheckCommandAccepts is the accept half of the guard matrix: the shapes an
// operator legitimately writes. Each case names the field it stands for, because
// the same rules apply to k8s.runtime, docker.runtime, podman.runtime and
// docker.compose from one implementation.
func TestCheckCommandAccepts(t *testing.T) {
	cases := []struct {
		name  string
		rules commandRules
		cmd   Command
		extra map[string]bool
	}{
		// Every allowlisted binary, bare.
		{"kubectl bare", clusterRules(), Command{"kubectl"}, nil},
		{"oc bare", clusterRules(), Command{"oc"}, nil},
		{"docker bare", runtimeRules(Docker), Command{"docker"}, nil},
		{"nerdctl bare", runtimeRules(Docker), Command{"nerdctl"}, nil},
		{"podman bare", runtimeRules(Podman), Command{"podman"}, nil},

		// A profile: flags and their values are the normal way to carry a context.
		{"flag args", clusterRules(), Command{"kubectl", "--context", "prod", "-n", "solace"}, nil},
		{"kubeconfig profile", clusterRules(), Command{"kubectl", "--kubeconfig", "/tmp/kc"}, nil},
		// --flag=value is self-contained, so the NEXT token is not its value and
		// must stand on its own -- here another flag.
		{"joined flag then flag", clusterRules(), Command{"kubectl", "--context=prod", "--namespace=solace"}, nil},
		// A bare "-" is a flag by shape; kubectl and docker both use it for stdin.
		{"lone dash", clusterRules(), Command{"kubectl", "-"}, nil},

		// Windows portability: one .exe suffix is stripped before the allowlist.
		{"exe suffix", clusterRules(), Command{"kubectl.exe"}, nil},
		{"EXE suffix", runtimeRules(Docker), Command{"docker.EXE"}, nil},

		// Chained runners, only WITH the operator's approval. All non-escalating:
		// --allow-command can never approve sudo and its relatives (neverAllowed).
		{"lima podman with hatch", runtimeRules(Podman), Command{"lima", "podman"}, map[string]bool{"lima": true}},
		{"microk8s kubectl with hatch", clusterRules(), Command{"microk8s", "kubectl"}, map[string]bool{"microk8s": true}},
		{"lima flag then nerdctl", runtimeRules(Docker), Command{"lima", "--tty=false", "nerdctl"}, map[string]bool{"lima": true}},

		// docker.compose is the one field allowed a bare subcommand, and only this one.
		{"compose plugin", composeRules(), Command{"docker", "compose"}, nil},
		{"compose standalone", composeRules(), Command{"docker-compose"}, nil},
		// Derived from a wrapped runtime: `compose` still lands last, directly after
		// the allowed binary, which is the position the rule is written for.
		{"compose behind a wrapper", composeRules(), Command{"lima", "nerdctl", "compose"}, map[string]bool{"lima": true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := CheckCommand(tc.rules, tc.cmd, tc.extra); err != nil {
				t.Errorf("CheckCommand(%q) = %v, want accepted", tc.cmd, err)
			}
		})
	}
}

// TestCheckCommandRejects is the reject half. Every case carries the substring the
// message must contain, because an error that does not name the offending token and
// the way out is a failed error, not a passed test (§4a).
func TestCheckCommandRejects(t *testing.T) {
	cases := []struct {
		name  string
		rules commandRules
		cmd   Command
		extra map[string]bool
		want  string
	}{
		// Empty.
		{"empty command", clusterRules(), Command{}, nil, "k8s.runtime is empty"},
		{"nil command", clusterRules(), nil, nil, "k8s.runtime is empty"},
		{"empty argument", clusterRules(), Command{"kubectl", ""}, nil, "k8s.runtime[1] is an empty argument"},

		// Layer 2: an unlisted binary, whatever it is.
		{"curl", clusterRules(), Command{"curl"}, nil, `"curl" is not a binary this tool runs`},
		{"bash", runtimeRules(Docker), Command{"bash"}, nil, `"bash" is not a binary this tool runs`},
		// Right binary, wrong platform: podman is not a Kubernetes CLI.
		{"cross-platform binary", clusterRules(), Command{"podman"}, nil, `"podman" is not a binary this tool runs`},
		// The allowlist and the escape hatch are both named in the message.
		{"names the way out", clusterRules(), Command{"curl"}, nil, "--allow-command curl"},
		{"names the allowlist", clusterRules(), Command{"curl"}, nil, "kubectl, oc"},

		// Layer 2: any path form. This is the attack the bare-name rule exists for --
		// a kubectl shipped in the same archive as the env file.
		{"relative path", clusterRules(), Command{"./kubectl"}, nil, "must be a bare binary name, not a path"},
		{"absolute path", clusterRules(), Command{"/usr/bin/kubectl"}, nil, "must be a bare binary name, not a path"},
		{"windows path", clusterRules(), Command{"C:/tools/kubectl.exe"}, nil, "must be a bare binary name, not a path"},
		{"parent dir", clusterRules(), Command{"../kubectl"}, nil, "must be a bare binary name, not a path"},
		// A backslash path trips the charset first, which is also a refusal.
		{"backslash path", clusterRules(), Command{`.\kubectl.exe`}, nil, `is not allowed in a command token`},

		// Layer 3: a bare word in subcommand position.
		{"bare subcommand", clusterRules(), Command{"kubectl", "delete"}, nil, "not allowed in subcommand position"},
		{"smuggled delete", clusterRules(), Command{"kubectl", "delete", "ns", "prod"}, nil, "not allowed in subcommand position"},
		{"compose smuggling rm", composeRules(), Command{"docker", "rm"}, nil, "not allowed in subcommand position"},
		// After --flag=value the next token is NOT a flag value, so a bare word there
		// is still smuggling.
		{"joined flag then word", clusterRules(), Command{"kubectl", "--context=prod", "delete"}, nil, "not allowed in subcommand position"},
		// The subword exception is field-scoped: `compose` is a bare word anywhere
		// except index 1 of docker.compose.
		{"compose subword on runtime", runtimeRules(Docker), Command{"docker", "compose"}, nil, "not allowed in subcommand position"},
		// ...and only as the LAST token: anything after it is a token this tool did
		// not append, which is the smuggling the rule exists to stop.
		{"compose then extra word", composeRules(), Command{"docker", "compose", "up"}, nil, "not allowed in subcommand position"},

		// Layer 3: a chained runner WITHOUT the escape hatch.
		{"lima no hatch", runtimeRules(Podman), Command{"lima", "podman"}, nil, `"lima" is not a binary this tool runs`},
		{"microk8s no hatch", clusterRules(), Command{"microk8s", "kubectl"}, nil, `"microk8s" is not a binary this tool runs`},
		// The hatch approves the name it was given, not any other.
		{"wrong hatch", runtimeRules(Podman), Command{"lima", "podman"}, map[string]bool{"colima": true}, `"lima" is not a binary this tool runs`},
		// An escalation wrapper is refused even when something managed to put it in
		// the extra-allowed set: allowed() strips the category back out (neverAllowed).
		{"sudo cannot be approved", runtimeRules(Podman), Command{"sudo", "podman"}, map[string]bool{"sudo": true}, `"sudo" is not a binary this tool runs`},
		{"doas cannot be approved", runtimeRules(Podman), Command{"doas", "podman"}, map[string]bool{"doas": true}, `"doas" is not a binary this tool runs`},
		{"pkexec cannot be approved", clusterRules(), Command{"pkexec", "kubectl"}, map[string]bool{"pkexec": true}, `"pkexec" is not a binary this tool runs`},

		// Layer 3: the literal end-of-flags token.
		{"double dash", clusterRules(), Command{"kubectl", "--"}, nil, `"--" is not allowed`},
		{"double dash then word", clusterRules(), Command{"kubectl", "--", "delete"}, nil, `"--" is not allowed`},

		// Layer 1: the charset, one case per class.
		{"semicolon", clusterRules(), Command{"kubectl;rm"}, nil, `contains ";"`},
		{"pipe", clusterRules(), Command{"kubectl|tee"}, nil, `contains "|"`},
		{"ampersand", clusterRules(), Command{"kubectl&"}, nil, `contains "&"`},
		{"redirect out", clusterRules(), Command{"kubectl>out"}, nil, `contains ">"`},
		{"redirect in", clusterRules(), Command{"kubectl<in"}, nil, `contains "<"`},
		{"subshell open", clusterRules(), Command{"kubectl("}, nil, `contains "("`},
		{"subshell close", clusterRules(), Command{"kubectl)"}, nil, `contains ")"`},
		{"glob star", clusterRules(), Command{"kube*"}, nil, `contains "*"`},
		{"glob question", clusterRules(), Command{"kubectl?"}, nil, `contains "?"`},
		{"bracket open", clusterRules(), Command{"kubectl["}, nil, `contains "["`},
		{"bracket close", clusterRules(), Command{"kubectl]"}, nil, `contains "]"`},
		{"brace open", clusterRules(), Command{"kubectl{"}, nil, `contains "{"`},
		{"brace close", clusterRules(), Command{"kubectl}"}, nil, `contains "}"`},
		{"tilde", clusterRules(), Command{"~kubectl"}, nil, `contains "~"`},
		{"hash", clusterRules(), Command{"kubectl#x"}, nil, `contains "#"`},
		{"bang", clusterRules(), Command{"kubectl!"}, nil, `contains "!"`},
		{"dollar", clusterRules(), Command{"$KUBECTL"}, nil, `contains "$"`},
		{"backtick", clusterRules(), Command{"`kubectl`"}, nil, "contains \"`\""},
		{"single quote", clusterRules(), Command{"'kubectl'"}, nil, `contains "'"`},
		{"double quote", clusterRules(), Command{`"kubectl"`}, nil, `contains "\""`},
		{"backslash", clusterRules(), Command{`kube\ctl`}, nil, `contains "\\"`},
		{"control char", clusterRules(), Command{"kubectl\n--all"}, nil, "contains a control character"},
		{"NUL", clusterRules(), Command{"kube\x00ctl"}, nil, "contains a control character"},
		{"DEL", clusterRules(), Command{"kubectl\x7f"}, nil, "contains a control character"},
		// Whitespace inside ONE argument -- only reachable through the list form,
		// since the scalar form splits on whitespace before it gets here.
		{"embedded space", clusterRules(), Command{"C:/Program Files/kubectl.exe"}, nil, "contains whitespace inside one argument"},
		{"space in flag value", clusterRules(), Command{"kubectl", "--kubeconfig", "/tmp/my kc"}, nil, "contains whitespace inside one argument"},
		// ...and whitespace means the whole Unicode property, not just ASCII space.
		// The scalar YAML form splits on all of these (strings.Fields is
		// Unicode-aware), so an ASCII-only check here would accept through the
		// sequence form exactly what the scalar form rejects.
		{"nbsp", clusterRules(), Command{"kubectl", "--context", "pr\u00a0od"}, nil, "contains whitespace inside one argument"},
		{"ideographic space", clusterRules(), Command{"kubectl", "--context", "pr\u3000od"}, nil, "contains whitespace inside one argument"},
		{"figure space", clusterRules(), Command{"kubectl", "--context", "pr\u2007od"}, nil, "contains whitespace inside one argument"},
		{"line separator", clusterRules(), Command{"kubectl", "--context", "pr\u2028od"}, nil, "contains whitespace inside one argument"},
		{"paragraph separator", clusterRules(), Command{"kubectl", "--context", "pr\u2029od"}, nil, "contains whitespace inside one argument"},
		{"ogham space mark", clusterRules(), Command{"kubectl", "--context", "pr\u1680od"}, nil, "contains whitespace inside one argument"},
		{"NEL", clusterRules(), Command{"kubectl", "--context", "pr\u0085od"}, nil, "contains whitespace inside one argument"},
		// Invisible formatting characters get their own refusal: they are not
		// whitespace and split nothing, but a token that renders identically to a
		// legitimate one defeats the review this guard asks an operator to do.
		{"zero-width space", clusterRules(), Command{"kubectl", "--context", "pr\u200bod"}, nil, "invisible formatting character"},
		{"zero-width joiner", clusterRules(), Command{"kubectl", "--context", "pr\u200dod"}, nil, "invisible formatting character"},
		{"RTL override", clusterRules(), Command{"kubectl", "--context", "pr\u202eod"}, nil, "invisible formatting character"},
		{"soft hyphen", clusterRules(), Command{"kubectl", "--context", "pr\u00adod"}, nil, "invisible formatting character"},
		// The message names the code point, since the character cannot be seen.
		{"names the code point", clusterRules(), Command{"kubectl", "--context", "pr\u200bod"}, nil, "U+200B"},
		// The charset applies to every token, not just argv[0].
		{"metachar in later token", clusterRules(), Command{"kubectl", "--context", "prod;rm"}, nil, `k8s.runtime[2]`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckCommand(tc.rules, tc.cmd, tc.extra)
			if err == nil {
				t.Fatalf("CheckCommand(%q) accepted a value it must reject", tc.cmd)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %v\n  want it to contain %q", err, tc.want)
			}
		})
	}
}

// TestFlagValuePositionIsNotGuaranteed documents the guard's one acknowledged
// limit, so it is a stated property rather than a surprise found later. A token
// following a value-shaped flag is accepted as that flag's value, because argument
// arity is unknowable without modelling every flag of every allowed binary -- so
// `docker --tls rm` passes even though --tls takes no value, and the appended
// subcommand would land after `rm`.
//
// This is the reason the hard guarantee is stated as "argv[0] and bare tokens",
// and the reason the trust-model note tells reviewers to read the whole command
// field rather than trusting the parser. If a future change narrows this -- an
// arity table, or refusing any token after a flag -- this test should fail and be
// rewritten as a rejection.
func TestFlagValuePositionIsNotGuaranteed(t *testing.T) {
	cmd := Command{"docker", "--tls", "rm"}
	if err := CheckCommand(runtimeRules(Docker), cmd, nil); err != nil {
		t.Fatalf("the flag-value limit changed: CheckCommand(%q) now returns %v -- "+
			"if that is deliberate, replace this test with a rejection case", cmd, err)
	}
	// What the guard DOES still hold onto in that position: the whole charset --
	// metacharacters, and the invisible characters that would otherwise make a flag
	// value unreviewable. This is the half an adversarial review found incomplete
	// once already (Unicode whitespace beyond ASCII space), so it is asserted here
	// rather than assumed from the accept matrix.
	for _, dirty := range []Command{
		{"docker", "--tls", "rm;curl"},
		{"docker", "--tls", "rm\u3000curl"},
		{"docker", "--tls", "r\u200bm"},
	} {
		if err := CheckCommand(runtimeRules(Docker), dirty, nil); err == nil {
			t.Errorf("CheckCommand(%q) accepted a charset violation in a flag value", dirty)
		}
	}
}

// TestCharsetAgreesAcrossBothYAMLForms is the regression guard for how the gap
// above arose. A Command may be written as a scalar (split with strings.Fields,
// which is Unicode-aware) or as an explicit sequence (preserved token for token).
// If the guard's own whitespace test were narrower than Fields', the sequence form
// would accept exactly what the scalar form splits apart -- two spellings of one
// config with two different verdicts.
func TestCharsetAgreesAcrossBothYAMLForms(t *testing.T) {
	spaces := []rune{' ', '\u00a0', '\u1680', '\u2000', '\u2007', '\u2028', '\u2029', '\u202f', '\u205f', '\u3000'}
	for _, r := range spaces {
		// The scalar form: Fields splits here, so the character never survives into
		// a token. Any rune Fields treats as a separator...
		scalar := decodeRuntime(t, "k8s:\n  runtime: \"kubectl --context a"+string(r)+"b\"\n")
		if len(scalar) != 4 {
			t.Errorf("U+%04X: scalar form produced %d tokens (%q), want 4 -- strings.Fields did not split it", r, len(scalar), scalar)
		}
		// ...the sequence form must refuse to carry embedded.
		seq := Command{"kubectl", "--context", "a" + string(r) + "b"}
		if err := CheckCommand(clusterRules(), seq, nil); err == nil {
			t.Errorf("U+%04X: the sequence form accepted a character the scalar form splits on", r)
		}
	}
}

// TestGuardErrorsAreActionable holds every guard message to the one-line,
// name-the-remedy bar: the offending token quoted, and either the allowlist or the
// escape hatch named. A message that only says "invalid" leaves the operator
// guessing which of four fields to edit.
func TestGuardErrorsAreActionable(t *testing.T) {
	cases := []Command{
		{"curl"},
		{"./kubectl"},
		{"kubectl", "delete"},
		{"kubectl", "--"},
		{"kubectl;rm"},
	}
	for _, cmd := range cases {
		err := CheckCommand(clusterRules(), cmd, nil)
		if err == nil {
			t.Fatalf("CheckCommand(%q) must fail", cmd)
		}
		msg := err.Error()
		if strings.Contains(msg, "\n") {
			t.Errorf("CheckCommand(%q) message spans lines; it must be one line: %q", cmd, msg)
		}
		if !strings.Contains(msg, "k8s.runtime") {
			t.Errorf("CheckCommand(%q) message does not name the field: %q", cmd, msg)
		}
		if !strings.Contains(msg, "--allow-command") && !strings.Contains(msg, "kubectl, oc") &&
			!strings.Contains(msg, "remove it") && !strings.Contains(msg, "let PATH resolve it") &&
			!strings.Contains(msg, "forward slashes") {
			t.Errorf("CheckCommand(%q) message names no remedy: %q", cmd, msg)
		}
	}
}

// --- layer 5: the validator and the executor share one definition -------------

// TestValidatorAndExecutorAgree is the shared-definition test. For every case in
// the matrix it drives BOTH enforcement points -- Validate (which config.Load runs)
// and the accessor every executor calls immediately before building argv -- and
// requires the same verdict. If someone ever forks the check, this fails: that is
// the whole point, since a hostile env file must be inert on a path that never
// reached Validate.
func TestValidatorAndExecutorAgree(t *testing.T) {
	cases := []struct {
		name    string
		p       Platform
		cmd     Command
		allowed []string
		wantOK  bool
	}{
		{"kubectl accepted", K8s, Command{"kubectl"}, nil, true},
		{"oc accepted", K8s, Command{"oc"}, nil, true},
		{"kubectl.exe accepted", K8s, Command{"kubectl.exe"}, nil, true},
		{"kubectl with flags accepted", K8s, Command{"kubectl", "--context", "prod", "-n", "ns"}, nil, true},
		{"docker accepted", Docker, Command{"docker"}, nil, true},
		{"podman accepted", Podman, Command{"podman"}, nil, true},
		{"chained with hatch accepted", Podman, Command{"lima", "podman"}, []string{"lima"}, true},

		{"curl rejected", K8s, Command{"curl"}, nil, false},
		{"relative path rejected", K8s, Command{"./kubectl"}, nil, false},
		{"absolute path rejected", K8s, Command{"/usr/bin/kubectl"}, nil, false},
		{"bare subcommand rejected", K8s, Command{"kubectl", "delete", "ns", "prod"}, nil, false},
		{"chained without hatch rejected", Podman, Command{"lima", "podman"}, nil, false},
		{"escalation rejected even with the hatch", Podman, Command{"sudo", "podman"}, []string{"lima"}, false},
		{"double dash rejected", K8s, Command{"kubectl", "--"}, nil, false},
		{"empty rejected", K8s, Command{""}, nil, false},
		{"metachar rejected", K8s, Command{"kubectl;rm"}, nil, false},
		{"docker unlisted rejected", Docker, Command{"curl"}, nil, false},
		{"podman unlisted rejected", Podman, Command{"curl"}, nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// The validator's verdict.
			vc := guardConfig(tc.p)
			if err := vc.AllowCommands(tc.allowed); err != nil {
				t.Fatalf("AllowCommands(%q) = %v", tc.allowed, err)
			}
			setGuardCommand(vc, tc.p, tc.cmd)
			validatorErr := vc.Validate(tc.p)

			// The executor's verdict, from a config that never saw Validate.
			ec := guardConfig(tc.p)
			if err := ec.AllowCommands(tc.allowed); err != nil {
				t.Fatalf("AllowCommands(%q) = %v", tc.allowed, err)
			}
			setGuardCommand(ec, tc.p, tc.cmd)
			_, executorErr := guardCommandOf(ec, tc.p)

			if (validatorErr == nil) != (executorErr == nil) {
				t.Fatalf("validator and executor disagree on %q: validator=%v executor=%v",
					tc.cmd, validatorErr, executorErr)
			}
			if gotOK := executorErr == nil; gotOK != tc.wantOK {
				t.Errorf("CheckCommand(%q) accepted = %v, want %v (err: %v)", tc.cmd, gotOK, tc.wantOK, executorErr)
			}
		})
	}
}

// setGuardCommand writes cmd into the field the platform p executes.
func setGuardCommand(c *Config, p Platform, cmd Command) {
	switch p {
	case Docker:
		c.Docker.Runtime = cmd
		// Keep compose independently valid, so a docker case tests the runtime
		// field alone rather than tripping over a compose derived from it.
		c.Docker.Compose = Command{"docker", "compose"}
	case Podman:
		c.Podman.Runtime = cmd
	default:
		c.K8s.Runtime = cmd
	}
}

// guardCommandOf reads the same field back through the executor's accessor.
func guardCommandOf(c *Config, p Platform) (Command, error) {
	if p.IsContainer() {
		return c.RuntimeCommand(p)
	}
	return c.ClusterCommand()
}

// TestExecutorRejectsWithoutValidate pins the reason the check runs twice: a
// Config built in code -- which is what every executor is handed, and what a test
// or a future caller may construct without config.Load -- must still be refused.
func TestExecutorRejectsWithoutValidate(t *testing.T) {
	c := &Config{}
	c.K8s.Runtime = Command{"./evil"}
	c.Docker.Runtime = Command{"curl"}
	c.Podman.Runtime = Command{"lima", "podman"}
	c.Docker.Compose = Command{"docker", "rm"}

	if _, err := c.ClusterCommand(); err == nil {
		t.Error("ClusterCommand accepted ./evil on a config that never ran Validate")
	}
	if _, err := c.RuntimeCommand(Docker); err == nil {
		t.Error("RuntimeCommand(Docker) accepted curl on a config that never ran Validate")
	}
	if _, err := c.RuntimeCommand(Podman); err == nil {
		t.Error("RuntimeCommand(Podman) accepted an unapproved lima chain")
	}
	if _, err := c.ComposeCommand(); err == nil {
		t.Error("ComposeCommand accepted a smuggled `docker rm`")
	}
}

// --- layer 4: the escape hatch ------------------------------------------------

func TestAllowCommandsAccepts(t *testing.T) {
	c := &Config{}
	if err := c.AllowCommands([]string{"lima", "microk8s", "colima.exe"}); err != nil {
		t.Fatalf("AllowCommands = %v, want accepted", err)
	}
	// Repeatable: each value extends the same set, and .exe folds to one entry.
	for _, want := range []string{"lima", "microk8s", "colima"} {
		if !c.extraAllowed[want] {
			t.Errorf("AllowCommands did not record %q (got %v)", want, c.extraAllowed)
		}
	}
}

// TestAllowCommandsRejects: a bad value is a usage error, and in particular the
// hatch cannot be used to smuggle back the path form layer 2 refuses.
func TestAllowCommandsRejects(t *testing.T) {
	cases := []struct {
		name  string
		value string
		want  string
	}{
		{"relative path", "./evil", "must be a bare binary name, not a path"},
		{"absolute path", "/usr/bin/evil", "must be a bare binary name, not a path"},
		{"backslash path", `C:\evil.exe`, "not allowed in a command token"},
		{"metacharacter", "lima;rm", "not allowed in a command token"},
		{"whitespace", "lima nerdctl", "whitespace inside one argument"},
		{"control character", "lima\n", "control character"},
		{"empty", "", "empty argument"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := &Config{}
			err := c.AllowCommands([]string{tc.value})
			if err == nil {
				t.Fatalf("AllowCommands(%q) accepted a value it must reject", tc.value)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %v, want it to contain %q", err, tc.want)
			}
			if !strings.Contains(err.Error(), "--allow-command") {
				t.Errorf("error = %v, want it to name the flag the value came from", err)
			}
		})
	}
}

// TestAllowCommandsRejectsEscalation: the escape hatch has a floor. Privilege
// escalation is not something an env file may ask for OR an operator may grant here,
// because granting it once on the command line elevates every command this tool
// issues for the whole life of that env file -- while `sudo solace-util ...` elevates one
// invocation the operator chose, at the moment they chose it. The category is
// refused, not the word: blocking `sudo` while allowing `doas` would be a control in
// name only.
func TestAllowCommandsRejectsEscalation(t *testing.T) {
	for _, name := range []string{"sudo", "doas", "su", "pkexec", "run0", "runas", "gsudo", "sudo.exe", "gsudo.EXE"} {
		t.Run(name, func(t *testing.T) {
			c := &Config{}
			err := c.AllowCommands([]string{name})
			if err == nil {
				t.Fatalf("AllowCommands(%q) approved a privilege-escalation wrapper", name)
			}
			if !strings.Contains(err.Error(), "is never permitted") {
				t.Errorf("error = %v, want it to say the value is never permitted", err)
			}
			// The message must point at the supported alternative, or the rule reads
			// as "you cannot do this" rather than "do it the other way".
			if !strings.Contains(err.Error(), "solace ...") {
				t.Errorf("error = %v, want it to name the `<escalator> solace ...` alternative", err)
			}
			if c.extraAllowed[execBase(name)] {
				t.Errorf("AllowCommands(%q) recorded the name despite failing", name)
			}
		})
	}
}

// TestEscalationCannotBeAllowedByAnyRoute is the structural backstop: even if a
// future edit put an escalation wrapper into execBinaries, or some other caller
// populated extraAllowed directly, allowed() strips the category back out. The rule
// does not depend on AllowCommands being the only door.
func TestEscalationCannotBeAllowedByAnyRoute(t *testing.T) {
	forced := map[string]bool{"sudo": true, "doas": true, "pkexec": true}
	for _, cmd := range []Command{
		{"sudo", "podman"},
		{"doas", "podman"},
		{"podman", "sudo"},
	} {
		if err := CheckCommand(runtimeRules(Podman), cmd, forced); err == nil {
			t.Errorf("CheckCommand(%q) accepted an escalation wrapper from a forced allow-set", cmd)
		}
	}
	// The floor is specific to escalation: a legitimate wrapper in the same forced
	// set still works, so this is a deny-list and not a broken allow-set.
	if err := CheckCommand(runtimeRules(Podman), Command{"lima", "podman"}, map[string]bool{"lima": true}); err != nil {
		t.Errorf("the deny-list broke a legitimate approval: %v", err)
	}
}

// TestAllowedBinaryIsNotGloballyAllowed: an approval widens the allowlist for the
// invocation, not for the process image -- it is stored per-Config, so a second
// config in the same run does not inherit it.
func TestAllowedBinaryIsNotGloballyAllowed(t *testing.T) {
	approved := &Config{}
	if err := approved.AllowCommands([]string{"lima"}); err != nil {
		t.Fatalf("AllowCommands = %v", err)
	}
	approved.Podman.Runtime = Command{"lima", "podman"}
	if _, err := approved.RuntimeCommand(Podman); err != nil {
		t.Fatalf("approved config rejected `lima podman`: %v", err)
	}

	plain := &Config{}
	plain.Podman.Runtime = Command{"lima", "podman"}
	if _, err := plain.RuntimeCommand(Podman); err == nil {
		t.Error("an --allow-command approval leaked into a config that never received it")
	}
}

// --- the compose derivation ---------------------------------------------------

// TestComposeCommandDerivation: docker.compose defaults to the runtime's own
// `compose` subcommand, and ComposeCommand owns that derivation so what Validate
// approved and what Manager.compose runs are the same expression.
func TestComposeCommandDerivation(t *testing.T) {
	c := &Config{}
	c.Docker.Runtime = Command{"docker"}
	got, err := c.ComposeCommand()
	if err != nil {
		t.Fatalf("ComposeCommand = %v", err)
	}
	if got.String() != "docker compose" {
		t.Errorf("derived compose = %q, want %q", got, "docker compose")
	}

	// ApplyDefaults must store the same value the accessor derives.
	d := &Config{}
	d.ApplyDefaults(Docker)
	stored := d.Docker.Compose.String()
	derived, err := d.ComposeCommand()
	if err != nil {
		t.Fatalf("ComposeCommand after defaults = %v", err)
	}
	if stored != derived.String() {
		t.Errorf("ApplyDefaults stored %q but ComposeCommand derives %q -- the two definitions have drifted",
			stored, derived)
	}
}

// TestComposeDerivationInheritsRejection: an unapproved runtime cannot become an
// approved compose command by way of the derivation.
func TestComposeDerivationInheritsRejection(t *testing.T) {
	c := &Config{}
	c.Docker.Runtime = Command{"curl"}
	if _, err := c.ComposeCommand(); err == nil {
		t.Error("a compose command derived from an unlisted runtime was accepted")
	}
}

// --- the schema cannot reach the hatch ----------------------------------------

// TestAllowCommandIsNotASchemaKey is the structural half of "the config author has
// no say": strict decoding (KnownFields) means any key attempting to name the
// escape hatch is a load error, not a silently ignored one. The field backing it is
// unexported, so yaml.v3 could not write it even without strict mode.
func TestAllowCommandIsNotASchemaKey(t *testing.T) {
	for _, key := range []string{"allowCommand", "allow-command", "allowCommands", "extraAllowed"} {
		doc := "redundancy: no\n" + key + ": [lima]\n"
		var c Config
		if err := decodeStrict(doc, &c); err == nil {
			t.Errorf("an env file key %q was accepted; the allowlist must not be settable from config", key)
		}
		if c.extraAllowed != nil {
			t.Errorf("an env file key %q reached the allowlist: %v", key, c.extraAllowed)
		}
	}
}
