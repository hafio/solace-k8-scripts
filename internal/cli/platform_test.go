package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"solace/internal/config"
)

// The platform used to be the first word of every command. It is now resolved --
// from --platform, or from the platform sections the env file declares -- so the
// resolution itself is the thing that has to be pinned: it decides what every
// later step in the run talks to, and getting it wrong silently would point a
// deploy at the wrong system.

// writePlatformEnv writes a minimal but VALID env carrying exactly the given
// platform sections, so a test can drive resolution without also tripping over
// schema validation. The kubernetes and container schemas need different
// mandatory fields, hence the two bodies.
func writePlatformEnv(t *testing.T, platforms ...config.Platform) string {
	t.Helper()
	body := "redundancy: no\n" +
		"image:\n  repo: solace-pubsub-standard\n  tag: \"10.10.1.128\"\n" +
		"admin:\n  pass: " + smokeAdminPass + "\n"
	for _, p := range platforms {
		switch p {
		case config.K8s:
			body += "kubernetes:\n  name: dev-broker\n  namespace: solace\n" +
				"  storage:\n    class: standard\n    msgNode: 30Gi\n"
		default:
			body += string(p) + ": {}\n"
		}
	}
	for _, p := range platforms {
		if p.IsContainer() {
			body += "nodes:\n  primary:\n    name: pri-host\n"
			break
		}
	}
	path := filepath.Join(t.TempDir(), "platforms.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write env: %v", err)
	}
	return path
}

// runPlatform runs `status broker` over the echo seam against path, which is the
// cheapest command that still goes all the way through resolution and config
// loading.
func runPlatform(t *testing.T, path string, configure func(*App), args ...string) (string, error) {
	t.Helper()
	full := append(append([]string{"status", "broker"}, args...), "--env", path)
	return runRootWith(t, full, func(a *App) {
		if configure != nil {
			configure(a)
		}
		echoRunner(a)
	})
}

// TestResolvesSinglePlatformSilently is the everyday case: an env file describes
// one deployment, so nothing needs to be said on the command line.
func TestResolvesSinglePlatformSilently(t *testing.T) {
	for _, p := range config.Platforms() {
		t.Run(string(p), func(t *testing.T) {
			out, err := runPlatform(t, writePlatformEnv(t, p), nil)
			if err != nil {
				t.Fatalf("status on a %s-only env: %v", p, err)
			}
			// The dry-run echo names the binary the platform drives, which is the
			// observable proof the right one was chosen.
			want := "kubectl"
			if p.IsContainer() {
				want = string(p)
			}
			if !strings.Contains(out, want) {
				t.Errorf("status on a %s-only env should drive %s, got:\n%s", p, want, out)
			}
		})
	}
}

// TestNoPlatformSectionIsRefused pins the marker requirement. A container env
// file needs no docker: keys at all -- every one of them defaults -- so without
// this rule such a file would be indistinguishable from a kubernetes one, and
// the tool would have to guess which system to deploy to.
func TestNoPlatformSectionIsRefused(t *testing.T) {
	path := filepath.Join(t.TempDir(), "no-platform.yaml")
	body := "redundancy: no\nimage:\n  repo: solace-pubsub-standard\n  tag: \"10.10.1.128\"\n" +
		"admin:\n  pass: " + smokeAdminPass + "\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write env: %v", err)
	}
	_, err := runPlatform(t, path, nil)
	if err == nil {
		t.Fatal("an env file declaring no platform section should be refused")
	}
	for _, want := range []string{"kubernetes", "docker", "podman"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q should name %q as a section to add", err, want)
		}
	}
}

// TestMultiPlatformNonInteractiveIsRefused: with no terminal to ask, guessing is
// the one thing that must not happen -- a scripted run that picked a platform on
// its own could deploy to the wrong one and look like it worked.
func TestMultiPlatformNonInteractiveIsRefused(t *testing.T) {
	path := writePlatformEnv(t, config.K8s, config.Docker)
	_, err := runPlatform(t, path, func(a *App) { a.Interactive = func() bool { return false } })
	if err == nil {
		t.Fatal("an ambiguous env file should be refused when nothing can be asked")
	}
	if !strings.Contains(err.Error(), "--platform") {
		t.Errorf("error %q should point at --platform", err)
	}
	for _, want := range []string{"kubernetes", "docker"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q should name the declared platform %q", err, want)
		}
	}
}

// TestMultiPlatformPromptSelects drives the interactive branch through the App's
// prompt seams, the same way the destructive-confirmation tests do.
func TestMultiPlatformPromptSelects(t *testing.T) {
	path := writePlatformEnv(t, config.K8s, config.Docker)
	out, err := runPlatform(t, path, func(a *App) {
		a.Interactive = func() bool { return true }
		a.PromptIn = strings.NewReader("2\n")
	})
	if err != nil {
		t.Fatalf("selecting docker at the prompt: %v", err)
	}
	if !strings.Contains(out, "docker") {
		t.Errorf("selecting entry 2 should drive docker, got:\n%s", out)
	}
}

// TestMultiPlatformPromptRejectsBadAnswer: an unusable answer stops the run. It
// must not fall through to a default, for the same reason the non-interactive
// case refuses to guess.
func TestMultiPlatformPromptRejectsBadAnswer(t *testing.T) {
	path := writePlatformEnv(t, config.K8s, config.Docker)
	for _, answer := range []string{"\n", "9\n", "docker\n", "0\n"} {
		_, err := runPlatform(t, path, func(a *App) {
			a.Interactive = func() bool { return true }
			a.PromptIn = strings.NewReader(answer)
		})
		if err == nil {
			t.Errorf("answer %q should not select a platform", strings.TrimSpace(answer))
		}
	}
}

// TestPlatformFlagSilencesThePrompt: naming the platform is what makes an
// ambiguous file usable from a script.
func TestPlatformFlagSilencesThePrompt(t *testing.T) {
	path := writePlatformEnv(t, config.K8s, config.Docker)
	out, err := runPlatform(t, path, func(a *App) { a.Interactive = func() bool { return false } },
		"--platform", "docker")
	if err != nil {
		t.Fatalf("--platform docker on an ambiguous env: %v", err)
	}
	if !strings.Contains(out, "docker") {
		t.Errorf("--platform docker should drive docker, got:\n%s", out)
	}
}

// TestPlatformFlagAcceptsAbbreviations pins the short spellings end to end, not
// just in the parser: they have to survive the whole resolution path.
func TestPlatformFlagAcceptsAbbreviations(t *testing.T) {
	for abbrev, want := range map[string]config.Platform{"kube": config.K8s, "dk": config.Docker, "pm": config.Podman} {
		path := writePlatformEnv(t, config.K8s, config.Docker, config.Podman)
		out, err := runPlatform(t, path, func(a *App) { a.Interactive = func() bool { return false } },
			"--platform", abbrev)
		if err != nil {
			t.Errorf("--platform %s: %v", abbrev, err)
			continue
		}
		marker := "kubectl"
		if want.IsContainer() {
			marker = string(want)
		}
		if !strings.Contains(out, marker) {
			t.Errorf("--platform %s should drive %s, got:\n%s", abbrev, want, out)
		}
	}
}

// TestPlatformFlagRejectsUndeclaredSection: --platform names which of the file's
// deployments to drive, so naming one the file does not describe is a mistake
// worth stopping for -- the alternative is deploying from defaults the operator
// never wrote down.
func TestPlatformFlagRejectsUndeclaredSection(t *testing.T) {
	path := writePlatformEnv(t, config.K8s)
	_, err := runPlatform(t, path, nil, "--platform", "podman")
	if err == nil {
		t.Fatal("--platform podman against a kubernetes-only env should be refused")
	}
	if !strings.Contains(err.Error(), "podman") || !strings.Contains(err.Error(), "kubernetes") {
		t.Errorf("error %q should name both what was asked for and what the file declares", err)
	}
}

// TestPlatformFlagRejectsUnknownValue keeps the retired spellings out: `k8s`
// never came back, and `k8` was the abbreviation only until `kube` replaced it.
func TestPlatformFlagRejectsUnknownValue(t *testing.T) {
	path := writePlatformEnv(t, config.K8s)
	for _, bad := range []string{"k8s", "k8", "swarm"} {
		if _, err := runPlatform(t, path, nil, "--platform", bad); err == nil {
			t.Errorf("--platform %s should be refused", bad)
		}
	}
}

// TestUnsupportedCommandFailsLoud is the other half of the one-tree decision: the
// tree shows every command on every platform, so the refusal has to be the thing
// that tells an operator a command does not apply here -- and it has to name
// where it does apply, or the message is a dead end.
func TestUnsupportedCommandFailsLoud(t *testing.T) {
	for _, tc := range []struct {
		platform config.Platform
		args     []string
		applies  string
	}{
		{config.Docker, []string{"status", "operator"}, "kubernetes"},
		{config.Docker, []string{"restart", "operator"}, "kubernetes"},
		{config.Podman, []string{"prepare", "namespace"}, "kubernetes"},
		{config.Docker, []string{"deploy", "operator"}, "kubernetes"},
		{config.Docker, []string{"config", "apply", "additional-users"}, "kubernetes"},
		{config.Docker, []string{"remove", "namespace"}, "kubernetes"},
		{config.Docker, []string{"remove", "operator"}, "kubernetes"},
		{config.Docker, []string{"generate", "operator"}, "kubernetes"},
		{config.K8s, []string{"prepare", "host"}, "docker"},
		// `generate broker` is deliberately absent: it is the one noun that means
		// the same thing on both families -- what `deploy broker` would apply -- so
		// it is refused nowhere. TestGenerateWired covers both renderings.
	} {
		t.Run(string(tc.platform)+" "+strings.Join(tc.args, " "), func(t *testing.T) {
			path := writePlatformEnv(t, config.K8s, config.Docker, config.Podman)
			args := append(append([]string{}, tc.args...), "--platform", string(tc.platform))
			_, err := runRootWith(t, append(args, "--env", path),
				func(a *App) { a.Interactive = func() bool { return false }; echoRunner(a) })
			if err == nil {
				t.Fatalf("%v should be refused on %s", tc.args, tc.platform)
			}
			if !strings.Contains(err.Error(), "not supported on "+string(tc.platform)) {
				t.Errorf("error %q should say it is not supported on %s", err, tc.platform)
			}
			if !strings.Contains(err.Error(), tc.applies) {
				t.Errorf("error %q should name %s, where it does apply", err, tc.applies)
			}
		})
	}
}

// TestScopedFlagFailsLoud: a flag that exists on every platform but means
// something on only one is refused where it means nothing, rather than accepted
// and ignored -- a --restart that did nothing would read as "already restarted".
func TestScopedFlagFailsLoud(t *testing.T) {
	for _, tc := range []struct {
		platform config.Platform
		args     []string
		flag     string
	}{
		{config.K8s, []string{"deploy", "broker", "--restart"}, "restart"},
		{config.K8s, []string{"deploy", "all", "--restart"}, "restart"},
		{config.Docker, []string{"cli", "--pod", "primary"}, "pod"},
		// --all is deliberately NOT here: it applies on every platform. On
		// Kubernetes it surveys the cluster, on a container host every Solace
		// container found by image -- the same question, asked of what that
		// platform has. TestStatusAllFindsBrokersByImage covers the container half.
	} {
		t.Run(string(tc.platform)+" --"+tc.flag, func(t *testing.T) {
			path := writePlatformEnv(t, config.K8s, config.Docker, config.Podman)
			args := append(append([]string{}, tc.args...), "--platform", string(tc.platform))
			_, err := runRootWith(t, append(args, "--env", path),
				func(a *App) { a.Interactive = func() bool { return false }; echoRunner(a) })
			if err == nil {
				t.Fatalf("--%s should be refused on %s", tc.flag, tc.platform)
			}
			if !strings.Contains(err.Error(), "--"+tc.flag) {
				t.Errorf("error %q should name --%s", err, tc.flag)
			}
		})
	}
}

// TestUnusableRoleFailsLoud: the [role] positional means opposite things on the
// two platform families, and on the family that ignores it a role that was typed
// must not be silently dropped -- `logs backup` reading the local broker's logs
// on a container host is the exact mistake this prevents.
func TestUnusableRoleFailsLoud(t *testing.T) {
	for _, tc := range []struct {
		platform config.Platform
		args     []string
	}{
		{config.Docker, []string{"logs", "broker", "backup"}},
		{config.Docker, []string{"cli", "backup"}},
		{config.Docker, []string{"shell", "monitor"}},
		{config.Docker, []string{"check", "semp-login", "backup"}},
		{config.Docker, []string{"status", "broker", "backup"}},
		{config.Docker, []string{"restart", "broker", "backup"}},
		{config.K8s, []string{"deploy", "broker", "backup"}},
		{config.K8s, []string{"deploy", "all", "backup"}},
		{config.K8s, []string{"config", "leader", "backup"}},
		{config.K8s, []string{"smoke", "redundancy", "backup"}},
	} {
		t.Run(string(tc.platform)+" "+strings.Join(tc.args, " "), func(t *testing.T) {
			path := writePlatformEnv(t, config.K8s, config.Docker, config.Podman)
			args := append(append([]string{}, tc.args...), "--platform", string(tc.platform))
			_, err := runRootWith(t, append(args, "--env", path),
				func(a *App) { a.Interactive = func() bool { return false }; echoRunner(a) })
			if err == nil {
				t.Fatalf("%v should be refused on %s", tc.args, tc.platform)
			}
			if !strings.Contains(err.Error(), "role") {
				t.Errorf("error %q should explain the role argument", err)
			}
		})
	}
}

// TestPlatformIsAnnouncedInThePreamble: which system a command is about to talk
// to is now inferred rather than typed, so it has to be stated -- otherwise the
// one fact the operator no longer supplies is also the one they cannot see.
func TestPlatformIsAnnouncedInThePreamble(t *testing.T) {
	path := writePlatformEnv(t, config.Docker)
	stderr := captureStderr(t, func() {
		if _, err := runPlatform(t, path, nil); err != nil {
			t.Fatalf("status: %v", err)
		}
	})
	if !strings.Contains(stderr, "platform: docker") {
		t.Errorf("the preamble should announce the resolved platform, got:\n%s", stderr)
	}
}

// TestCompletionNeverReadsTheEnvFile is the invariant that decided where the
// pre-run hook lives. Cobra runs the NEAREST ancestor's PersistentPreRunE, and
// __complete is a child of root -- so a hook on root would parse an untrusted
// env file on every TAB press. Keeping the hook on each command instead is what
// prevents that, and this proves it: completion still works when the env file
// named on the command line does not even exist.
func TestCompletionNeverReadsTheEnvFile(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist.yaml")
	for _, args := range [][]string{
		{"--env", missing, "logs", "broker", ""},
		{"--env", missing, "status", "broker", "--allow-command", ""},
		{"--env", missing, ""},
	} {
		if _, directive := runComplete(t, args...); directive == "" {
			t.Errorf("completion for %v should work without an env file", args)
		}
	}
}

// TestPlatformFlagIsOnRoot pins the flag's placement: it is inherited by every
// command, including convert, which is what lets one word mean one thing across
// the whole CLI.
func TestPlatformFlagIsOnRoot(t *testing.T) {
	root := newRootCmd(&App{})
	if root.PersistentFlags().Lookup("platform") == nil {
		t.Fatal("--platform should be a root persistent flag")
	}
	convert := findCmd(t, root, "convert")
	if convert.Flags().Lookup("platform") != nil {
		t.Error("convert should inherit --platform, not declare its own")
	}
	if convert.InheritedFlags().Lookup("platform") == nil {
		t.Error("convert should see the inherited --platform")
	}
}

// TestScopedCommandsSaySoInHelp: the tree is one static shape, so the help text
// is the only place an operator can learn that a command does not apply before
// running it.
func TestScopedCommandsSaySoInHelp(t *testing.T) {
	root := newRootCmd(&App{})
	for _, tc := range []struct {
		path []string
		want string
	}{
		{[]string{"status", "operator"}, "kubernetes only"},
		{[]string{"restart", "operator"}, "kubernetes only"},
		{[]string{"prepare", "host"}, "docker/podman only"},
		{[]string{"generate", "operator"}, "kubernetes only"},
	} {
		c := findCmd(t, root, tc.path...)
		if !strings.Contains(c.Short, tc.want) {
			t.Errorf("%q Short = %q, want it to mention %q", c.CommandPath(), c.Short, tc.want)
		}
	}
	// A command that applies everywhere carries no such tail.
	if c := findCmd(t, root, "status"); strings.Contains(c.Short, "only)") {
		t.Errorf("status applies everywhere, so its Short should carry no scope: %q", c.Short)
	}
}

// TestPlatformAnnotationsMatchDispatch guards the one place this design could rot
// silently: a command's applicability annotation is what help and the pre-run
// check read, while the ops map is what actually runs. If a command were tagged
// for a platform it has no implementation for, the refusal would come from the
// wrong place -- as an internal error at dispatch instead of an actionable one
// before anything loaded.
func TestPlatformAnnotationsMatchDispatch(t *testing.T) {
	root := newRootCmd(&App{})
	var walk func(*cobra.Command)
	walk = func(c *cobra.Command) {
		if v, ok := c.Annotations[platformAnnotation]; ok {
			if got := parsePlatformList(v); len(got) == 0 {
				t.Errorf("%q carries an unparseable platform annotation %q", c.CommandPath(), v)
			}
		}
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(root)
}
