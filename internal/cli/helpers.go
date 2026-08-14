package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"solace/internal/config"
)

// emit writes rendered bytes to stdout. Used by `gen` and `--gen` paths so the
// artifact is the command's only stdout (progress/warnings go to stderr).
func emit(b []byte) error {
	_, err := os.Stdout.Write(b)
	return err
}

// genAnnotation marks a command as able to honor the --gen-*-only flags (render
// an artifact instead of applying it). Every platform's PersistentPreRunE rejects
// them on a command without it.
const genAnnotation = "solace_gen_capable"

// genCapable tags a command as gen-aware and returns it, so registration can wrap
// a command inline: genCapable(leaf(...)).
func genCapable(c *cobra.Command) *cobra.Command {
	if c.Annotations == nil {
		c.Annotations = map[string]string{}
	}
	c.Annotations[genAnnotation] = "true"
	return c
}

// renderAnnotation marks a command that ONLY renders -- it never executes an
// external command, whatever flags it is given. `gen` is the whole set today. It
// exists so checkAllowCommand can refuse --allow-command where there is nothing to
// allow, the mirror of what genAnnotation does for the --gen-*-only trio.
const renderAnnotation = "solace_render_only"

// renderOnly tags a command as never executing, and returns it so registration can
// wrap inline: renderOnly(genCapable(...)).
func renderOnly(c *cobra.Command) *cobra.Command {
	if c.Annotations == nil {
		c.Annotations = map[string]string{}
	}
	c.Annotations[renderAnnotation] = "true"
	return c
}

// anyGen reports whether a gen flag asked for a rendering instead of the real
// work. Handlers branch on it before doing anything that changes state.
func (a *App) anyGen() bool { return a.GenOnly || a.GenSecretsOnly || a.GenEnvOnly }

// willExecute reports whether this invocation can reach an external command. A
// --gen-*-only run renders and changes nothing, and a render-only command never
// executes at all.
func (a *App) willExecute(cmd *cobra.Command) bool {
	return !a.anyGen() && cmd.Annotations[renderAnnotation] != "true"
}

// addAllowCommandFlag wires --allow-command onto a platform subtree. It is the
// operator's escape hatch for the execution-guard allowlist (config/execguard.go):
// a binary this tool does not drive by default -- a `microk8s kubectl`, a site
// wrapper -- runs only when the person at the keyboard names it, for that one
// invocation. It cannot approve a privilege-escalation wrapper at all: elevate this
// tool when you run it (`sudo solace ...`), never through an env file.
//
// It is a CLI flag and NOTHING else on purpose. There is no config key for it, no
// environment variable, and no binding layer that could give an env file a way to
// set it: an env file that could approve its own binary would make the allowlist
// decorative. It is registered on the platform commands rather than on root so
// `solace convert --allow-command ...` is a usage error too.
func addAllowCommandFlag(c *cobra.Command, app *App) {
	c.PersistentFlags().StringArrayVar(&app.AllowCommand, "allow-command", nil,
		"approve one extra binary for the config's platform command, for this run only "+
			"(repeatable; a bare name, never a path). The env file cannot grant this")
}

// checkAllowCommand rejects --allow-command on an invocation that cannot execute
// anything. Silently accepting it there would teach the flag as harmless boilerplate
// -- exactly the habit that gets it pasted into a wrapper script, where it then
// applies to runs that DO execute. Hand-rolled rather than cobra's flag groups for
// the same reason checkGenFlags is: the flag is declared on the platform command and
// validated against the leaf that inherited it, which lets the error name the leaf.
func checkAllowCommand(cmd *cobra.Command, app *App) error {
	if len(app.AllowCommand) == 0 || app.willExecute(cmd) {
		return nil
	}
	return fmt.Errorf("--allow-command is only valid on a command that runs something, and %q renders "+
		"without executing; drop the flag", cmd.CommandPath())
}

// checkGenFlags validates the --gen-*-only trio for the command about to run.
// They are root persistent flags, so they parse on every command on every
// platform, but only a command tagged genCapable renders an artifact: silently
// ignoring one elsewhere would mask a user mistake -- and could let someone think
// a destructive command was a dry render -- so we fail loud (§4). Combining them
// is rejected too: each selects a different artifact, so a pair has no meaning.
//
// This is a hand-rolled check rather than cobra's MarkFlagsMutuallyExclusive
// because the flags are declared on root and validated against the leaf command
// that inherited them, which also lets the error name the offending command.
func checkGenFlags(cmd *cobra.Command, app *App) error {
	var set []string
	if app.GenOnly {
		set = append(set, "--gen-only")
	}
	if app.GenSecretsOnly {
		set = append(set, "--gen-secrets-only")
	}
	if app.GenEnvOnly {
		set = append(set, "--gen-env-only")
	}
	switch {
	case len(set) == 0:
		return nil
	case len(set) > 1:
		return fmt.Errorf("%s cannot be combined: each renders a different artifact, so pass exactly one",
			strings.Join(set, " and "))
	case cmd.Annotations[genAnnotation] != "true":
		return fmt.Errorf("%s is only valid on artifact commands (deploy, gen -- plus prep secrets, prep operator and operator deploy on k8s), not %q",
			set[0], cmd.CommandPath())
	}
	return nil
}

// opFunc is a leaf handler that needs only the app context.
type opFunc func(*App) error

// roleOpFunc is a leaf handler parameterized by a broker node role (p|b|m).
type roleOpFunc func(*App, config.Role) error

// leaf builds a no-arg subcommand that dispatches straight to fn.
func leaf(app *App, use, short string, fn opFunc) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.NoArgs,
		RunE:  func(*cobra.Command, []string) error { return fn(app) },
	}
}

// roleLeaf builds a subcommand taking an optional [role] positional (p|b|m,
// default primary), normalizing it via config.ParseRole before dispatching.
func roleLeaf(app *App, use, short string, fn roleOpFunc) *cobra.Command {
	return &cobra.Command{
		Use:   use + " [role]",
		Short: short,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			role, err := config.ParseRole(firstArg(args))
			if err != nil {
				return err
			}
			return fn(app, role)
		},
	}
}

// addDataFlags wires the unified data-retention flags onto a delete/down command.
// Data is kept by default; --purge (alias --clear-data) clears it; --keep-data
// keeps it and skips the interactive prompt. Keeping and clearing are mutually
// exclusive so an ambiguous `--purge --keep-data` fails at parse time.
func addDataFlags(c *cobra.Command, app *App) {
	c.Flags().BoolVar(&app.purge, "purge", false, "clear persistent data (k8s PVCs / container data folder)")
	c.Flags().BoolVar(&app.purge, "clear-data", false, "alias for --purge")
	c.Flags().BoolVar(&app.keepData, "keep-data", false, "keep persistent data and skip the confirmation prompt")
	c.MarkFlagsMutuallyExclusive("purge", "keep-data")
	c.MarkFlagsMutuallyExclusive("clear-data", "keep-data")
}

// isTTY reports whether f is an interactive terminal (a character device). Prompts
// are gated on it so a piped/CI invocation never blocks reading stdin.
func isTTY(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// interactive reports whether this run may prompt. It routes through the App's
// seam so a test can exercise the prompt branches that gate every destructive
// action; unset (the production case) it is exactly isTTY(os.Stdin).
func interactive(a *App) bool {
	if a != nil && a.Interactive != nil {
		return a.Interactive()
	}
	return isTTY(os.Stdin)
}

// promptSource is where a confirmation answer is read from -- the App's seam, or
// os.Stdin in production.
func promptSource(a *App) io.Reader {
	if a != nil && a.PromptIn != nil {
		return a.PromptIn
	}
	return os.Stdin
}

// promptLine writes prompt to out and returns one trimmed line read from in.
func promptLine(in io.Reader, out io.Writer, prompt string) string {
	fmt.Fprint(out, prompt)
	line, _ := bufio.NewReader(in).ReadString('\n')
	return strings.TrimSpace(line)
}

// promptYesNo returns true for y/yes (case-insensitive) -- the lenient form used to
// confirm a reversible delete.
func promptYesNo(in io.Reader, out io.Writer, prompt string) bool {
	switch strings.ToLower(promptLine(in, out, prompt)) {
	case "y", "yes":
		return true
	}
	return false
}

// promptYes returns true only for an exact "yes" (case-insensitive) -- the strict
// form required before an irreversible data purge.
func promptYes(in io.Reader, out io.Writer, prompt string) bool {
	return strings.ToLower(promptLine(in, out, prompt)) == "yes"
}

// confirmDelete gates a delete/teardown. --yes confirms non-interactively; an
// interactive session is prompted [y/N]; a non-TTY without --yes declines loudly
// (never delete unattended without an explicit --yes).
func confirmDelete(a *App, what string) bool {
	if a.Yes {
		return true
	}
	if !interactive(a) {
		warn("refusing to delete %s without confirmation; pass --yes to proceed", what)
		return false
	}
	return promptYesNo(promptSource(a), os.Stderr, fmt.Sprintf("Delete %s? [y/N] ", what))
}

// addRestartFlag wires --restart onto a container deploy/up command. Deliberately
// separate from --yes: bouncing a live broker to apply a changed artifact is its
// own explicit decision, the same way clearing data is.
func addRestartFlag(c *cobra.Command, app *App) {
	c.Flags().BoolVar(&app.restart, "restart", false,
		"restart an already-running broker when the deploy artifact changed (otherwise you are asked, and a non-interactive run leaves it running)")
}

// confirmRestart asks whether a running broker may be bounced to apply a changed
// deploy artifact. A non-interactive session declines: the caller then leaves the
// new artifact in place and warns, so a scripted deploy never drops messaging
// traffic unattended.
// It takes the App so the prompt goes through the same seams as the other confirm
// helpers; ops_container wires it to Manager.Confirm as a closure.
func confirmRestart(a *App, question string) bool {
	if !interactive(a) {
		return false
	}
	return promptYesNo(promptSource(a), os.Stderr, question+" [y/N] ")
}

// confirmPurge decides whether persistent data (PVCs) is cleared alongside a delete.
// Data is kept by default: --keep-data keeps, --purge clears, a non-TTY keeps, and an
// interactive session clears only on an exact "yes". --yes never implies purge --
// clearing data is always its own explicit decision (safer than legacy 120).
func confirmPurge(a *App) bool {
	if a.keepData {
		return false
	}
	if a.purge {
		return true
	}
	if !interactive(a) {
		return false
	}
	return promptYes(promptSource(a), os.Stderr, "Also clear persistent data (PVCs)? This is irreversible. Type 'yes' to purge: ")
}

func firstArg(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	return ""
}

func firstArgOr(args []string, def string) string {
	if len(args) > 0 && args[0] != "" {
		return args[0]
	}
	return def
}
