package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"solace/internal/config"
)

// newCompletionCmd builds `solace-util auto-complete <shell>`. Cobra adds a command by
// this name on its own, but only from inside Execute -- and the command reference
// is rendered straight from newRootCmd, so a command that shipped and worked was
// missing from the generated docs. Registering it here makes it a command like any
// other: it lands in docs/commands.md, it is testable through the tree, and the
// help text is ours. That last part matters because writeCommand copies Long into
// the reference verbatim, and cobra's stock text carries markdown headings.
func newCompletionCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "auto-complete",
		Short: "Print the shell auto-completion script for solace-util",
		Long: "Print a shell's completion script on stdout. Load it to complete commands and\n" +
			"flags, plus the values they take: env files for -e/--env, primary|backup|monitor\n" +
			"for the [role] positionals and --pod, and directories for --base-dir and --dir.\n\n" +
			"Completion never reads the env file, so it stays inert -- a TAB press cannot\n" +
			"parse config or run anything. See each shell's help for how to load it.",
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		// A parent with no RunE is not Runnable, and cobra answers a non-runnable
		// command by printing help to STDOUT and exiting 0 -- so `solace-util completion
		// tcsh > solace-util.ps1` would write the help text into the profile script and
		// report success. Being runnable is what lets NoArgs reject the unknown shell
		// instead, and makes the bare command say what to pass. `--help` is unaffected:
		// the help flag short-circuits ahead of both.
		RunE: func(cmd *cobra.Command, _ []string) error {
			return fmt.Errorf("%s needs a shell: bash, zsh, fish, or powershell", cmd.CommandPath())
		},
	}
	c.AddCommand(
		completionShell("bash",
			"Load into the current shell:\n\n"+
				"  source <(solace-util auto-complete bash)\n\n"+
				"Load for every session (needs the bash-completion package):\n\n"+
				"  solace-util auto-complete bash > /etc/bash_completion.d/solace-util",
			func(root *cobra.Command, w io.Writer, desc bool) error {
				return root.GenBashCompletionV2(w, desc)
			}),
		completionShell("zsh",
			"Load into the current shell:\n\n"+
				"  source <(solace-util auto-complete zsh)\n\n"+
				"Load for every session (compinit must be enabled in ~/.zshrc):\n\n"+
				"  solace-util auto-complete zsh > \"${fpath[1]}/_solace-util\"",
			func(root *cobra.Command, w io.Writer, desc bool) error {
				if !desc {
					return root.GenZshCompletionNoDesc(w)
				}
				return root.GenZshCompletion(w)
			}),
		completionShell("fish",
			"Load into the current shell:\n\n"+
				"  solace-util auto-complete fish | source\n\n"+
				"Load for every session:\n\n"+
				"  solace-util auto-complete fish > ~/.config/fish/completions/solace-util.fish",
			func(root *cobra.Command, w io.Writer, desc bool) error {
				return root.GenFishCompletion(w, desc)
			}),
		completionShell("powershell",
			"Load into the current shell:\n\n"+
				"  solace-util auto-complete powershell | Out-String | Invoke-Expression\n\n"+
				"Load for every session, by writing the script once and sourcing it from\n"+
				"your profile:\n\n"+
				"  solace-util auto-complete powershell > solace-util.ps1",
			func(root *cobra.Command, w io.Writer, desc bool) error {
				if !desc {
					return root.GenPowerShellCompletion(w)
				}
				return root.GenPowerShellCompletionWithDesc(w)
			}),
	)
	return c
}

// completionShell builds one shell's generator. gen is handed the include-descriptions
// decision rather than reading the flag itself, which keeps the flag variable scoped
// to the command that declares it.
func completionShell(shell, long string, gen func(root *cobra.Command, w io.Writer, desc bool) error) *cobra.Command {
	var noDesc bool
	c := &cobra.Command{
		Use:               shell,
		Short:             "Print the " + shell + " completion script",
		Long:              long,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// os.Stdout, like emit: the script is this command's only stdout.
			return gen(cmd.Root(), os.Stdout, !noDesc)
		},
	}
	c.Flags().BoolVar(&noDesc, "no-descriptions", false, "omit the descriptions shown beside each completion")
	return c
}

// completeEnvFiles offers the env files -e/--env can name. It mirrors
// config.ResolveEnvPath: the base dir first, then <base-dir>/env, bare names only,
// first one wins -- so a base-dir file shadowing the env/ copy of the same name is
// offered once, the way it resolves. A value that already carries a directory is
// used verbatim by ResolveEnvPath, so that case hands back to the shell's own file
// completion instead of guessing. The .yaml/.yml filter is a suggestion filter and
// not validation: ResolveEnvPath still accepts any regular file typed out in full.
func completeEnvFiles(app *App) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if strings.ContainsAny(toComplete, `/\`) {
			return nil, cobra.ShellCompDirectiveDefault
		}
		base := app.BaseDir
		if base == "" {
			base = "."
		}
		var names []string
		seen := map[string]bool{}
		for _, dir := range []string{base, filepath.Join(base, "env")} {
			entries, err := os.ReadDir(dir)
			if err != nil {
				continue // a missing env/ is the normal case, and a TAB press has nowhere to report it
			}
			for _, e := range entries {
				name := e.Name()
				if e.IsDir() || seen[name] || !isEnvFileName(name) || !strings.HasPrefix(name, toComplete) {
					continue
				}
				seen[name] = true
				names = append(names, name)
			}
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	}
}

// isEnvFileName reports whether name looks like a YAML env file.
func isEnvFileName(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".yaml", ".yml":
		return true
	}
	return false
}

// completeRoles completes a role value -- a [role] positional or --pod -- from the
// long names ParseRole accepts.
func completeRoles(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return matching(config.RoleNames(), toComplete), cobra.ShellCompDirectiveNoFileComp
}

// completePlatforms completes --platform. Only the canonical names are offered:
// the kube/dk/pm abbreviations exist to save typing something you already know,
// which is precisely what a completion removes the need for, and suggesting both
// spellings would put two names for one platform in front of the user. The empty
// value is left out too -- "detect it from the env file" is what omitting the
// flag already gives you.
func completePlatforms(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	all := make([]string, 0, len(config.Platforms()))
	for _, p := range config.Platforms() {
		all = append(all, string(p))
	}
	return matching(all, toComplete), cobra.ShellCompDirectiveNoFileComp
}

// completeDirs completes a flag whose value is a directory.
func completeDirs(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveFilterDirs
}

// matching keeps the candidates the partial word could still become. Cobra filters
// ValidArgs this way but leaves a completion function to filter its own.
func matching(candidates []string, toComplete string) []string {
	var out []string
	for _, c := range candidates {
		if strings.HasPrefix(c, toComplete) {
			out = append(out, c)
		}
	}
	return out
}

// registerFlagCompletion wires a completion function onto a flag. The only way
// RegisterFlagCompletionFunc fails is a flag name that was never declared -- a
// wiring bug, not a runtime condition -- and the tree builders have no error to
// return, so swallowing it would ship a flag that silently completes nothing.
// Every test that builds the tree runs this, so the panic cannot reach a user.
func registerFlagCompletion(c *cobra.Command, name string,
	fn func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective)) {
	if err := c.RegisterFlagCompletionFunc(name, fn); err != nil {
		panic(fmt.Sprintf("completion for --%s on %q: %v", name, c.CommandPath(), err))
	}
}
