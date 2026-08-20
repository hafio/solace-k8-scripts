package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// writeRuntimeEnv writes a standalone env whose kubernetes.runtime is the given command,
// so a test can drive the whole CLI -- flag parsing, config.Load, Validate, and the
// executors -- against one hostile or wrapped value.
func writeRuntimeEnv(t *testing.T, runtime string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "runtime.yaml")
	doc := "redundancy: no\n" +
		"image:\n  repo: solace/solace-pubsub-standard\n  tag: \"10.10.1.35\"\n" +
		"admin:\n  pass: " + smokeAdminPass + "\n" +
		"kubernetes:\n  name: broker\n  namespace: solace\n  runtime: " + runtime + "\n" +
		"  storage:\n    msgNode: 30Gi\n"
	if err := os.WriteFile(path, []byte(doc), 0o600); err != nil {
		t.Fatalf("writing env: %v", err)
	}
	return path
}

// TestAllowCommandIsRegisteredWhereItExecutes: the escape hatch reaches every
// command that can execute. The platform subtrees that used to each carry it as a
// shared PersistentFlag are gone -- the tree is flat now -- so wireExec puts the
// flag on the command itself (helpers.go addAllowCommandFlag), and this walks the
// whole tree checking it directly rather than checking three subtree attachment
// points.
//
// The flag belongs to each command that executes because that is the unit the
// operator is approving a binary for. Root is the wrong place for the same reason
// it always was: `solace-util convert` hangs off root too, loads no env file, and
// runs no platform CLI, so a root-level flag would offer it somewhere it can never
// apply.
func TestAllowCommandIsRegisteredWhereItExecutes(t *testing.T) {
	root := newRootCmd(&App{})
	exempt := map[*cobra.Command]bool{
		findCmd(t, root, "convert"):    true,
		findCmd(t, root, "version"):    true,
		findCmd(t, root, "auto-complete"): true,
	}
	walkCommands(root, 0, func(c *cobra.Command, _ int) {
		if c.RunE == nil || exempt[c] {
			return
		}
		// Verb groups are runnable only to print help or reject an unknown noun;
		// they execute nothing, so there is no binary for --allow-command to approve.
		if c.Annotations[groupAnnotation] == "true" {
			return
		}
		if p := c.Parent(); p != nil && exempt[p] {
			return // the completion subtree's own shells (bash, zsh, ...)
		}
		if c.Flags().Lookup("allow-command") == nil {
			t.Errorf("%s has RunE but no --allow-command flag", c.CommandPath())
		}
	})
	if root.PersistentFlags().Lookup("allow-command") != nil {
		t.Error("--allow-command is a root persistent flag; it must be scoped to each command that executes")
	}
	if _, err := runRoot(t, []string{"convert", "--allow-command", "lima", "somefile"}); err == nil {
		t.Error("`solace-util convert --allow-command` should be a usage error: convert runs no platform CLI")
	}
}

// TestAllowCommandIsRepeatable: each value extends the same allowlist, so a chain
// needing two approvals does not force a choice between them.
func TestAllowCommandIsRepeatable(t *testing.T) {
	app := &App{}
	root := newRootCmd(app)
	root.SetArgs([]string{"check", "deploy", "--allow-command", "lima", "--allow-command", "microk8s",
		"--env", "does-not-exist.yaml"})
	_ = root.Execute() // fails on the missing env; the flag values are what matter
	if len(app.AllowCommand) != 2 || app.AllowCommand[0] != "lima" || app.AllowCommand[1] != "microk8s" {
		t.Errorf("AllowCommand = %v, want both values collected in order", app.AllowCommand)
	}
}

// TestAllowCommandApprovesAWrappedRuntime is the end-to-end accept case: a chained
// runner the env file could never authorize on its own runs when the operator names
// it, and the same file without the flag does not.
func TestAllowCommandApprovesAWrappedRuntime(t *testing.T) {
	env := writeRuntimeEnv(t, "microk8s kubectl")

	if _, err := runRootWith(t, []string{"check", "deploy", "--platform", "kubernetes", "--env", env}, echoRunner); err == nil {
		t.Fatal("an unapproved `microk8s kubectl` must be refused")
	} else if !strings.Contains(err.Error(), "--allow-command microk8s") {
		t.Errorf("error = %v, want it to name the escape hatch", err)
	}

	out, err := runRootWith(t, []string{"check", "deploy", "--allow-command", "microk8s", "--platform", "kubernetes", "--env", env}, echoRunner)
	if err != nil {
		t.Fatalf("an approved `microk8s kubectl` must run: %v", err)
	}
	if !strings.Contains(out, "microk8s") {
		t.Errorf("the approved wrapper should reach the echoed command:\n%s", out)
	}
}

// TestAllowCommandRejectsBadValues: a bad value is a usage error, and in particular
// the hatch cannot be used to reintroduce the path form the guard refuses.
func TestAllowCommandRejectsBadValues(t *testing.T) {
	cases := []struct {
		name  string
		value string
		want  string
	}{
		{"relative path", "./evil", "must be a bare binary name, not a path"},
		{"absolute path", "/usr/bin/evil", "must be a bare binary name, not a path"},
		{"metacharacter", "lima;rm", "not allowed in a command token"},
		{"empty", "", "empty argument"},
		// Privilege escalation is refused outright, not merely unlisted: granting it
		// once here would elevate every command the tool issues for the life of the
		// env file. `sudo solace-util ...` is the supported way, and the message says so.
		{"sudo", "sudo", "is never permitted"},
		{"doas", "doas", "is never permitted"},
		{"pkexec", "pkexec", "is never permitted"},
		{"sudo.exe", "sudo.exe", "is never permitted"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := runRoot(t, withEnv("check", "deploy", "--allow-command", tc.value, "--platform", "kubernetes"))
			if err == nil {
				t.Fatalf("--allow-command %q must be a usage error", tc.value)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %v, want it to contain %q", err, tc.want)
			}
		})
	}
}

// TestAllowCommandRejectedWhereNothingExecutes: the flag is meaningless on a
// render -- every leaf under `generate` -- and saying so is what keeps it from
// being learned as harmless boilerplate, which is how it ends up pasted into a
// wrapper script that DOES execute.
func TestAllowCommandRejectedWhereNothingExecutes(t *testing.T) {
	cases := [][]string{
		{"generate", "broker", "--allow-command", "lima", "--platform", "kubernetes"},
		{"generate", "operator", "--allow-command", "lima", "--platform", "kubernetes"},
		{"generate", "secrets", "--allow-command", "lima", "--platform", "docker"},
		{"generate", "broker", "--allow-command", "lima", "--platform", "podman"},
	}
	for _, args := range cases {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			_, err := runRoot(t, withEnv(args...))
			if err == nil {
				t.Fatal("--allow-command must be refused where nothing executes")
			}
			if !strings.Contains(err.Error(), "--allow-command is only valid on a command that runs something") {
				t.Errorf("error = %v, want the render-only refusal", err)
			}
		})
	}
}

// TestEscalationIsRefusedEndToEnd: an env file naming a privilege-escalation
// wrapper cannot be rescued by the escape hatch, and the refusal names the supported
// alternative -- elevate the tool itself, at the moment you run it.
func TestEscalationIsRefusedEndToEnd(t *testing.T) {
	env := writeRuntimeEnv(t, "sudo kubectl")

	// Without the hatch: an unlisted binary.
	if _, err := runRootWith(t, []string{"status", "broker", "--platform", "kubernetes", "--env", env}, echoRunner); err == nil {
		t.Fatal("`sudo kubectl` was accepted")
	}
	// With the hatch: refused at the flag, before the config is even read.
	_, err := runRootWith(t, []string{"status", "broker", "--allow-command", "sudo", "--platform", "kubernetes", "--env", env}, echoRunner)
	if err == nil {
		t.Fatal("--allow-command sudo was accepted")
	}
	for _, want := range []string{"is never permitted", "solace ..."} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error = %v, want it to contain %q", err, want)
		}
	}
}

// TestHostileRuntimeIsRefusedByEveryVerb: the guard is not a property of one code
// path. A config naming a binary this tool does not drive must stop every verb that
// could execute, including the read-only ones -- `status` running curl is the same
// arbitrary execution `deploy` running it would be.
func TestHostileRuntimeIsRefusedByEveryVerb(t *testing.T) {
	env := writeRuntimeEnv(t, "curl")
	verbs := [][]string{
		{"check", "deploy"},
		{"status", "broker"},
		{"deploy", "broker"},
		{"remove", "broker", "--no-prompt"},
		{"status", "broker", "--all"},
		{"logs", "broker"},
	}
	for _, verb := range verbs {
		t.Run(strings.Join(verb, " "), func(t *testing.T) {
			_, err := runRootWith(t, append(append([]string{}, verb...), "--platform", "kubernetes", "--env", env), echoRunner)
			if err == nil {
				t.Fatalf("`solace %s` ran with an unlisted binary as kubernetes.runtime", strings.Join(verb, " "))
			}
			if !strings.Contains(err.Error(), "is not a binary this tool runs") {
				t.Errorf("error = %v, want the allowlist refusal", err)
			}
		})
	}
}

// TestSmuggledSubcommandIsRefused: the attack the flag-shape rule exists for. This
// tool appends its own subcommand, so `kubectl delete` in the config would put a
// destructive verb ahead of it.
func TestSmuggledSubcommandIsRefused(t *testing.T) {
	for _, runtime := range []string{"kubectl delete", "kubectl delete ns prod", "kubectl --"} {
		t.Run(runtime, func(t *testing.T) {
			env := writeRuntimeEnv(t, runtime)
			_, err := runRootWith(t, []string{"status", "broker", "--platform", "kubernetes", "--env", env}, echoRunner)
			if err == nil {
				t.Fatalf("kubernetes.runtime %q was accepted", runtime)
			}
			msg := err.Error()
			if !strings.Contains(msg, "subcommand position") && !strings.Contains(msg, `"--" is not allowed`) {
				t.Errorf("error = %v, want the flag-shape refusal", msg)
			}
		})
	}
}

// TestPathRuntimeIsRefused: the bare-name rule, end to end. An env file that ships
// with a kubectl beside it must not be able to point at that copy.
func TestPathRuntimeIsRefused(t *testing.T) {
	for _, runtime := range []string{"./kubectl", "/usr/local/bin/kubectl", "../kubectl"} {
		t.Run(runtime, func(t *testing.T) {
			env := writeRuntimeEnv(t, runtime)
			_, err := runRootWith(t, []string{"status", "broker", "--platform", "kubernetes", "--env", env}, echoRunner)
			if err == nil {
				t.Fatalf("kubernetes.runtime %q was accepted", runtime)
			}
			if !strings.Contains(err.Error(), "must be a bare binary name, not a path") {
				t.Errorf("error = %v, want the bare-name refusal", err)
			}
		})
	}
}

// TestGenPathNeverExecutes backs the trust-model note's promise: rendering an
// artifact from an untrusted env file is safe, because that path runs nothing. The
// hostile command is still refused at load -- the guard runs in Validate -- so the
// promise is tested at the level that matters: no external command is issued.
func TestGenPathNeverExecutes(t *testing.T) {
	app := &App{}
	root := newRootCmd(app)
	root.SetArgs([]string{"generate", "broker", "--platform", "kubernetes", "--env", writeRuntimeEnv(t, "curl")})
	root.SetOut(nil)
	root.SetErr(nil)
	_ = captureStdout(t, func() { _ = root.Execute() })
	// Whatever the outcome, no runner was ever handed a command: App.load builds
	// the Runner, and nothing under `generate` calls it.
	if rr, ok := app.Runner.(*opRunner); ok && len(rr.calls) > 0 {
		t.Errorf("the gen path executed %d command(s): %+v", len(rr.calls), rr.calls)
	}
}
