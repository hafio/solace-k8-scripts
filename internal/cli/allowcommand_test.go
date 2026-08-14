package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeRuntimeEnv writes a standalone env whose k8s.runtime is the given command,
// so a test can drive the whole CLI -- flag parsing, config.Load, Validate, and the
// executors -- against one hostile or wrapped value.
func writeRuntimeEnv(t *testing.T, runtime string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "runtime.yaml")
	doc := "redundancy: no\n" +
		"image:\n  repo: solace/solace-pubsub-standard\n  tag: \"10.10.1.35\"\n" +
		"admin:\n  pass: " + smokeAdminPass + "\n" +
		"k8s:\n  name: broker\n  namespace: solace\n  runtime: " + runtime + "\n" +
		"  storage:\n    msgNode: 30Gi\n"
	if err := os.WriteFile(path, []byte(doc), 0o600); err != nil {
		t.Fatalf("writing env: %v", err)
	}
	return path
}

// TestAllowCommandIsRegisteredOnPlatformTrees: the escape hatch reaches every verb
// that can execute, and only through the platform subtrees. Registering it on root
// would put it on `solace convert`, which loads no config and runs no platform CLI.
func TestAllowCommandIsRegisteredOnPlatformTrees(t *testing.T) {
	root := newRootCmd(&App{})
	for _, platform := range []string{"k8s", "docker", "podman"} {
		cmd := findCmd(t, root, platform)
		if cmd.PersistentFlags().Lookup("allow-command") == nil {
			t.Errorf("%s subtree does not declare --allow-command", platform)
		}
		// Inherited by the leaves, which is where execution happens.
		if findCmd(t, root, platform, "deploy").InheritedFlags().Lookup("allow-command") == nil {
			t.Errorf("%s deploy does not inherit --allow-command", platform)
		}
	}
	if root.PersistentFlags().Lookup("allow-command") != nil {
		t.Error("--allow-command is a root persistent flag; it must be scoped to the platform trees")
	}
	if _, err := runRoot(t, []string{"convert", "--allow-command", "lima", "somefile"}); err == nil {
		t.Error("`solace convert --allow-command` should be a usage error: convert runs no platform CLI")
	}
}

// TestAllowCommandIsRepeatable: each value extends the same allowlist, so a chain
// needing two approvals does not force a choice between them.
func TestAllowCommandIsRepeatable(t *testing.T) {
	app := &App{}
	root := newRootCmd(app)
	root.SetArgs([]string{"k8s", "check", "--allow-command", "lima", "--allow-command", "microk8s",
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

	if _, err := runRoot(t, []string{"k8s", "check", "--dry-run", "--env", env}); err == nil {
		t.Fatal("an unapproved `microk8s kubectl` must be refused")
	} else if !strings.Contains(err.Error(), "--allow-command microk8s") {
		t.Errorf("error = %v, want it to name the escape hatch", err)
	}

	out, err := runRoot(t, []string{"k8s", "check", "--dry-run", "--allow-command", "microk8s", "--env", env})
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
		// env file. `sudo solace ...` is the supported way, and the message says so.
		{"sudo", "sudo", "is never permitted"},
		{"doas", "doas", "is never permitted"},
		{"pkexec", "pkexec", "is never permitted"},
		{"sudo.exe", "sudo.exe", "is never permitted"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := runRoot(t, withEnv("k8s", "check", "--dry-run", "--allow-command", tc.value))
			if err == nil {
				t.Fatalf("--allow-command %q must be a usage error", tc.value)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %v, want it to contain %q", err, tc.want)
			}
		})
	}
}

// TestAllowCommandRejectedWhereNothingExecutes: the flag is meaningless on a render
// -- `gen`, or any verb under a --gen-*-only flag -- and saying so is what keeps it
// from being learned as harmless boilerplate, which is how it ends up pasted into a
// wrapper script that DOES execute.
func TestAllowCommandRejectedWhereNothingExecutes(t *testing.T) {
	cases := [][]string{
		{"k8s", "gen", "broker", "--allow-command", "lima"},
		{"docker", "gen", "--allow-command", "lima"},
		{"k8s", "deploy", "--gen-only", "--allow-command", "lima"},
		{"docker", "deploy", "--gen-only", "--allow-command", "lima"},
	}
	for _, args := range cases {
		t.Run(strings.Join(args[:2], " "), func(t *testing.T) {
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
	if _, err := runRoot(t, []string{"k8s", "status", "--dry-run", "--env", env}); err == nil {
		t.Fatal("`sudo kubectl` was accepted")
	}
	// With the hatch: refused at the flag, before the config is even read.
	_, err := runRoot(t, []string{"k8s", "status", "--dry-run", "--allow-command", "sudo", "--env", env})
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
		{"k8s", "check"},
		{"k8s", "status"},
		{"k8s", "deploy"},
		{"k8s", "delete", "--yes"},
		{"k8s", "show-all"},
		{"k8s", "logs"},
	}
	for _, verb := range verbs {
		t.Run(strings.Join(verb, " "), func(t *testing.T) {
			_, err := runRoot(t, append(append([]string{}, verb...), "--dry-run", "--env", env))
			if err == nil {
				t.Fatalf("`solace %s` ran with an unlisted binary as k8s.runtime", strings.Join(verb, " "))
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
			_, err := runRoot(t, []string{"k8s", "status", "--dry-run", "--env", env})
			if err == nil {
				t.Fatalf("k8s.runtime %q was accepted", runtime)
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
			_, err := runRoot(t, []string{"k8s", "status", "--dry-run", "--env", env})
			if err == nil {
				t.Fatalf("k8s.runtime %q was accepted", runtime)
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
	root.SetArgs([]string{"k8s", "gen", "broker", "--env", writeRuntimeEnv(t, "curl")})
	root.SetOut(nil)
	root.SetErr(nil)
	_ = captureStdout(t, func() { _ = root.Execute() })
	// Whatever the outcome, no runner was ever handed a command: App.load builds
	// the Runner, and nothing under `gen` calls it.
	if rr, ok := app.Runner.(*opRunner); ok && len(rr.calls) > 0 {
		t.Errorf("the gen path executed %d command(s): %+v", len(rr.calls), rr.calls)
	}
}
