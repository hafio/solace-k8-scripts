package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// runComplete drives cobra's hidden __complete endpoint -- the same request a
// loaded completion script makes on every TAB press. runRoot cannot be reused:
// it points cobra's own writer at io.Discard, and __complete writes the
// candidates there rather than to os.Stdout. The last line is always the
// ":<directive>" marker.
func runComplete(t *testing.T, args ...string) (candidates []string, directive string) {
	t.Helper()
	var buf bytes.Buffer
	root := newRootCmd(&App{})
	root.SetArgs(append([]string{cobra.ShellCompRequestCmd}, args...))
	root.SetOut(&buf)
	root.SetErr(io.Discard)
	if err := root.Execute(); err != nil {
		t.Fatalf("__complete %v: %v", args, err)
	}
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) == 0 {
		t.Fatalf("__complete %v produced no output, want at least a directive line", args)
	}
	return lines[:len(lines)-1], lines[len(lines)-1]
}

// wantDirective renders the ":<n>" marker cobra prints last, so the tests name the
// directive instead of a magic number.
func wantDirective(d cobra.ShellCompDirective) string { return fmt.Sprintf(":%d", d) }

// TestCompletionScriptsGenerate: every advertised shell emits its own script. The
// marker is the line that actually binds the completer to the `solace-util` command, so
// a script that generated but wired up nothing would still fail.
func TestCompletionScriptsGenerate(t *testing.T) {
	for _, tc := range []struct{ shell, marker string }{
		{"bash", "__start_solace-util"},
		{"zsh", "#compdef solace-util"},
		{"fish", "complete -c solace-util"},
		{"powershell", "Register-ArgumentCompleter -CommandName 'solace-util'"},
	} {
		t.Run(tc.shell, func(t *testing.T) {
			out, err := runRoot(t, []string{"auto-complete", tc.shell})
			if err != nil {
				t.Fatalf("completion %s: %v", tc.shell, err)
			}
			if !strings.Contains(out, tc.marker) {
				t.Errorf("completion %s output missing %q", tc.shell, tc.marker)
			}
		})
	}
}

// TestCompletionNoDescriptions: --no-descriptions is honoured on every shell. It
// switches the request the generated script makes from __complete to
// __completeNoDesc, which is the only externally visible difference.
func TestCompletionNoDescriptions(t *testing.T) {
	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
		t.Run(shell, func(t *testing.T) {
			with, err := runRoot(t, []string{"auto-complete", shell, "--no-descriptions"})
			if err != nil {
				t.Fatalf("completion %s --no-descriptions: %v", shell, err)
			}
			if !strings.Contains(with, cobra.ShellCompNoDescRequestCmd) {
				t.Errorf("completion %s --no-descriptions does not request %s", shell, cobra.ShellCompNoDescRequestCmd)
			}
			without, err := runRoot(t, []string{"auto-complete", shell})
			if err != nil {
				t.Fatalf("completion %s: %v", shell, err)
			}
			if strings.Contains(without, cobra.ShellCompNoDescRequestCmd) {
				t.Errorf("completion %s requests %s without the flag", shell, cobra.ShellCompNoDescRequestCmd)
			}
		})
	}
}

// TestCompletionNeedsAShell: an unsupported shell, or none at all, fails loud with
// nothing on stdout. This is why the parent carries a RunE: cobra answers a
// non-runnable command by printing help to stdout and exiting 0, which would put
// the help text into `solace-util completion tcsh > solace-util.ps1` and call it a success.
func TestCompletionNeedsAShell(t *testing.T) {
	for _, args := range [][]string{{"auto-complete", "tcsh"}, {"auto-complete"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			out, err := runRoot(t, args)
			if err == nil {
				t.Fatalf("%v err = nil, want a loud failure naming the shells", args)
			}
			if out != "" {
				t.Errorf("%v wrote %q to stdout, want nothing", args, out)
			}
		})
	}
}

// TestCompletionHelpStillWorks: --help short-circuits ahead of the RunE above, so
// asking how to use the command is not itself an error.
func TestCompletionHelpStillWorks(t *testing.T) {
	if _, err := runRoot(t, []string{"auto-complete", "--help"}); err != nil {
		t.Errorf("completion --help err = %v, want nil", err)
	}
}

// TestEnvFlagCompletesEnvFiles: -e is completed from the two directories
// config.ResolveEnvPath searches, by bare name. The base-dir copy of a name that
// also exists under env/ is offered once, matching the shadowing the resolver
// applies, and a non-YAML file is not suggested.
func TestEnvFlagCompletesEnvFiles(t *testing.T) {
	base := t.TempDir()
	write := func(rel string) {
		t.Helper()
		p := filepath.Join(base, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatalf("mkdir for %s: %v", rel, err)
		}
		if err := os.WriteFile(p, []byte("redundancy: no\n"), 0o600); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	write("prod.yaml")
	write("env/prod.yaml") // shadowed by the base-dir copy above
	write("env/dev.yml")
	write("notes.txt")

	got, directive := runComplete(t, "status", "--base-dir", base, "-e", "")
	want := []string{"prod.yaml", "dev.yml"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("-e completions = %v, want %v (base dir first, deduped, YAML only)", got, want)
	}
	if directive != wantDirective(cobra.ShellCompDirectiveNoFileComp) {
		t.Errorf("-e directive = %s, want no-file-completion", directive)
	}
}

// TestEnvFlagPrefixFilters: a partial name narrows the suggestions rather than
// dumping every env file back at the shell.
func TestEnvFlagPrefixFilters(t *testing.T) {
	base := t.TempDir()
	for _, name := range []string{"prod.yaml", "preprod.yaml", "dev.yaml"} {
		if err := os.WriteFile(filepath.Join(base, name), []byte("redundancy: no\n"), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	got, _ := runComplete(t, "status", "--base-dir", base, "-e", "pr")
	want := []string{"preprod.yaml", "prod.yaml"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("-e pr completions = %v, want %v", got, want)
	}
}

// TestEnvFlagWithPathDefersToShell: a value carrying a directory is used verbatim
// by ResolveEnvPath, so completion hands back to the shell's own file completion
// instead of offering bare names that would resolve somewhere else.
func TestEnvFlagWithPathDefersToShell(t *testing.T) {
	base := t.TempDir()
	if err := os.WriteFile(filepath.Join(base, "prod.yaml"), []byte("redundancy: no\n"), 0o600); err != nil {
		t.Fatalf("write prod.yaml: %v", err)
	}
	got, directive := runComplete(t, "status", "--base-dir", base, "-e", "env/")
	if len(got) != 0 {
		t.Errorf("-e env/ completions = %v, want none", got)
	}
	if directive != wantDirective(cobra.ShellCompDirectiveDefault) {
		t.Errorf("-e env/ directive = %s, want the shell's default file completion", directive)
	}
}

// TestRoleArgsComplete: every command taking a [role] positional offers the three
// role names -- the ones built through roleOnK8sLeaf/roleOnContainerLeaf/roleLeaf
// and the ones assembled inline (cli, deploy broker/all). The tree is flat, so one
// case per verb covers every platform it applies to; completion never reads the
// env file, so it cannot tell here whether a given run will land on Kubernetes
// (which rejects the role) or a container host (which uses it) -- see
// TestUnusableRoleFailsLoud (platform_test.go) for that refusal.
func TestRoleArgsComplete(t *testing.T) {
	for _, path := range [][]string{
		{"logs", "broker"},
		{"cli"},
		{"shell"},
		{"check", "semp-login"},
		{"restart", "broker"},
		{"deploy", "broker"},
		{"deploy", "all"},
		{"config", "leader"},
		{"smoke", "redundancy"},
		{"generate", "broker"},
	} {
		t.Run(strings.Join(path, " "), func(t *testing.T) {
			got, directive := runComplete(t, append(path, "")...)
			want := []string{"primary", "backup", "monitor"}
			if strings.Join(got, ",") != strings.Join(want, ",") {
				t.Errorf("completions = %v, want %v", got, want)
			}
			if directive != wantDirective(cobra.ShellCompDirectiveNoFileComp) {
				t.Errorf("directive = %s, want no-file-completion", directive)
			}
		})
	}
}

// TestPodFlagCompletesRoles: --pod names the same roles as the positionals, so it
// completes to the same set rather than to filenames.
func TestPodFlagCompletesRoles(t *testing.T) {
	got, directive := runComplete(t, "copy", "into", "--pod", "")
	want := []string{"primary", "backup", "monitor"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("--pod completions = %v, want %v", got, want)
	}
	if directive != wantDirective(cobra.ShellCompDirectiveNoFileComp) {
		t.Errorf("--pod directive = %s, want no-file-completion", directive)
	}
}

// TestPlatformFlagCompletes: --platform completes to the three canonical platform
// names, on root and on every command that inherits it -- convert included, since
// it no longer declares its own copy of the flag and reads the root one instead.
// The empty "detect" value is left out -- omitting the flag is how you ask for it.
//
// Neither the retired k8s spelling nor the kube/dk/pm abbreviations are offered: the
// abbreviations exist to save typing something you already know, which is what a
// completion already does -- offering both would put two names for one platform
// in front of the user.
func TestPlatformFlagCompletes(t *testing.T) {
	for _, path := range [][]string{
		{"status"},
		{"convert", "old-env"},
	} {
		t.Run(strings.Join(path, " "), func(t *testing.T) {
			got, directive := runComplete(t, append(path, "--platform", "")...)
			want := []string{"kubernetes", "docker", "podman"}
			if strings.Join(got, ",") != strings.Join(want, ",") {
				t.Errorf("--platform completions = %v, want %v", got, want)
			}
			if directive != wantDirective(cobra.ShellCompDirectiveNoFileComp) {
				t.Errorf("--platform directive = %s, want no-file-completion", directive)
			}
			for _, retired := range []string{"k8s", "k8", "kube", "dk", "pm"} {
				for _, c := range got {
					if c == retired {
						t.Errorf("--platform completions = %v, must not offer %q", got, retired)
					}
				}
			}
		})
	}
}

// TestDirFlagCompletesDirectories: --dir takes a directory, so it asks the shell
// to filter to directories rather than offering every file. The tree is flat now,
// so `copy into` is one command shared by every platform rather than a separate
// copy per platform subtree.
func TestDirFlagCompletesDirectories(t *testing.T) {
	_, directive := runComplete(t, "copy", "into", "--dir", "")
	if directive != wantDirective(cobra.ShellCompDirectiveFilterDirs) {
		t.Errorf("--dir directive = %s, want directory filtering", directive)
	}
}

// TestNoArgsLeafOffersNoFiles: a command built by leaf takes no arguments, so it
// offers nothing. Cobra's fallback is filename completion, which is what this
// stops -- the wrong suggestion on the majority of commands in the tree.
//
// The cases below deliberately include every no-arg command that needs a flag on
// top (remove broker/all, diagnostics): those used to hand-roll the literal leaf
// already builds, which is how they lost NoFileCompletions while the plain leaves
// kept it. They now go through leaf and attach the extra afterwards, so there is
// one definition of "takes no arguments" for the whole tree.
//
// The verb groups themselves (check, status, remove, ...) are deliberately
// absent: they carry no RunE and no ValidArgsFunction of their own, so completing
// after one offers its object subcommands' names -- the group's own semantics,
// not "takes no arguments". `deploy broker`/`deploy all` are absent for the same
// reason as TestRoleArgsComplete's comment: their [role] positional is offered by
// ValidArgs unconditionally, since completion cannot tell here whether this run
// will land on Kubernetes (which rejects the role) or a container host (which
// uses it).
func TestNoArgsLeafOffersNoFiles(t *testing.T) {
	for _, path := range [][]string{
		{"check", "deploy"},
		{"status", "operator"},
		{"remove", "broker"},
		{"remove", "all"},
		{"diagnostics"},
		{"prepare", "namespace"},
		{"logs", "operator"},
		{"config", "delete", "domain-certs"},
		{"version"},
	} {
		t.Run(strings.Join(path, " "), func(t *testing.T) {
			got, directive := runComplete(t, append(path, "")...)
			if len(got) != 0 {
				t.Errorf("completions = %v, want none", got)
			}
			if directive != wantDirective(cobra.ShellCompDirectiveNoFileComp) {
				t.Errorf("directive = %s, want no-file-completion", directive)
			}
		})
	}
}

// TestAllowCommandOffersNoFiles: --allow-command takes a bare binary name, never a
// path. Offering files would coach the mistake its own help text warns against.
// wireExec wires the flag the same way on every command that carries it, so one
// case built through leaf (check) and one built inline (deploy) are enough to
// prove the registration, rather than enumerating every runnable command.
func TestAllowCommandOffersNoFiles(t *testing.T) {
	for _, path := range [][]string{{"check", "deploy"}, {"deploy", "broker"}} {
		t.Run(strings.Join(path, " "), func(t *testing.T) {
			got, directive := runComplete(t, append(path, "--allow-command", "")...)
			if len(got) != 0 {
				t.Errorf("completions = %v, want none", got)
			}
			if directive != wantDirective(cobra.ShellCompDirectiveNoFileComp) {
				t.Errorf("directive = %s, want no-file-completion", directive)
			}
		})
	}
}

// TestFlagCompletionsRegistered is the drift gate: renaming a flag without moving
// its completion registration leaves the flag completing filenames again, which no
// other test here would notice because it fails silently at a TAB press.
func TestFlagCompletionsRegistered(t *testing.T) {
	root := newRootCmd(&App{})
	cases := []struct {
		path []string
		flag string
	}{
		{nil, "env"},
		{nil, "base-dir"},
		{nil, "platform"},
		{[]string{"status", "broker"}, "allow-command"},
		{[]string{"deploy", "broker"}, "allow-command"},
		{[]string{"cli"}, "pod"},
		{[]string{"copy", "from"}, "pod"},
		{[]string{"copy", "into"}, "pod"},
		{[]string{"copy", "into"}, "dir"},
		{[]string{"diagnostics"}, "days"},
	}
	for _, tc := range cases {
		cmd := findCmd(t, root, tc.path...)
		if _, ok := cmd.GetFlagCompletionFunc(tc.flag); !ok {
			t.Errorf("%s: --%s has no completion function registered", cmd.CommandPath(), tc.flag)
		}
	}
}
