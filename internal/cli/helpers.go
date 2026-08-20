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

// willExecute reports whether this invocation can reach an external command. Only
// the render-only commands cannot: they build an artifact from the env file and
// print it, touching nothing.
func (a *App) willExecute(cmd *cobra.Command) bool {
	return cmd.Annotations[renderAnnotation] != "true"
}

// addAllowCommandFlag wires --allow-command onto one command that executes. It is
// the operator's escape hatch for the execution-guard allowlist (config/execguard.go):
// a binary this tool does not drive by default -- a `microk8s kubectl`, a site
// wrapper -- runs only when the person at the keyboard names it, for that one
// invocation. It cannot approve a privilege-escalation wrapper at all: elevate this
// tool when you run it (`sudo solace-util ...`), never through an env file.
//
// It is a CLI flag and NOTHING else on purpose. There is no config key for it, no
// environment variable, and no binding layer that could give an env file a way to
// set it: an env file that could approve its own binary would make the allowlist
// decorative. wireExec adds it to each command that runs something rather than to
// root, so `solace-util convert --allow-command ...` is a usage error too.
func addAllowCommandFlag(c *cobra.Command, app *App) {
	c.Flags().StringArrayVar(&app.AllowCommand, "allow-command", nil,
		"approve one extra binary for the config's platform command, for this run only "+
			"(repeatable; a bare name, never a path). The env file cannot grant this")
	// No file completion: the value is a bare binary name, and offering paths would
	// coach exactly the mistake the help text above warns against.
	registerFlagCompletion(c, "allow-command", cobra.NoFileCompletions)
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

// opFunc is a leaf handler that needs only the app context.
type opFunc func(*App) error

// roleOpFunc is a leaf handler parameterized by a broker node role (p|b|m).
type roleOpFunc func(*App, config.Role) error

// leaf builds a no-arg subcommand that dispatches straight to fn. It takes no
// arguments, so NoFileCompletions is what it should offer -- cobra's default is
// to fall back to filenames, which would be wrong for every command built here.
func leaf(app *App, use, short string, fn opFunc) *cobra.Command {
	return wireExec(app, &cobra.Command{
		Use:               use,
		Short:             short,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE:              func(*cobra.Command, []string) error { return fn(app) },
	})
}

// roleLeaf builds a subcommand taking an optional [role] positional (p|b|m,
// default primary), normalizing it via config.ParseRole before dispatching.
func roleLeaf(app *App, use, short string, fn roleOpFunc) *cobra.Command {
	return wireExec(app, &cobra.Command{
		Use:       use + " [role]",
		Short:     short,
		ValidArgs: config.RoleNames(),
		Args:      cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			role, err := config.ParseRole(firstArg(args))
			if err != nil {
				return err
			}
			return fn(app, role)
		},
	})
}

// layer describes the one thing a removal keeps by default: the part that is
// expensive to recreate and impossible to get back. Both removals ask about theirs
// the same way, through addLayerFlags/confirmLayer, so learning the contract on one
// teaches the other.
type layer struct {
	flag  string // the flag that deletes it without asking
	what  string // what is kept, for the flag help and the report
	why   string // what deleting it costs, said at the moment of asking
	usage string // the flag's own help text
}

var (
	layerData = layer{
		flag: "delete-data",
		what: "persistent data",
		why:  "Kubernetes PVCs / the container data directory -- the broker's messages and configuration",
		usage: "delete the broker's persistent data too (Kubernetes PVCs / the container data " +
			"directory). Without it the data is kept",
	}
	layerCRD = layer{
		flag: "delete-crd",
		what: "the operator CRDs",
		why: "the CRDs are cluster-wide: deleting them cascade-deletes EVERY PubSubPlusEventBroker " +
			"in this cluster, including brokers this env file does not describe",
		usage: "delete the operator's CustomResourceDefinitions too. Without it they are kept, " +
			"so existing brokers survive",
	}
)

// addRemoveFlags wires the confirmation contract onto a command that destroys
// something. Every such command asks before it acts; --no-prompt is the ONE flag
// that makes it silent, so a script needs exactly one thing switched off rather
// than one per question.
//
// l is the retained layer, or nil for a command that has none (secrets and the
// namespace are recreated by `prepare`, so there is nothing worth keeping back).
// Where there is one, the two flags COMPOSE and are deliberately not exclusive:
// --delete-data says what to do with the layer, --no-prompt says not to ask about
// any of it, and a fully unattended removal wants both. Passing --delete-data
// alone still stops to confirm the removal itself, which is the point -- naming
// the data you are willing to lose is not the same as confirming the deletion.
func addRemoveFlags(c *cobra.Command, app *App, l *layer) {
	c.Flags().BoolVar(&app.noPrompt, "no-prompt", false,
		"do not ask anything: proceed with the removal, and keep whatever is kept by default "+
			"unless a --delete-* flag says otherwise")
	if l != nil {
		c.Flags().BoolVar(&app.deleteLayer, l.flag, false, l.usage)
	}
}

// confirmLayer decides whether the retained layer goes with the removal. Keeping it
// is the default in every direction: the --delete-* flag is the only way to a yes
// without being asked, a non-interactive or --no-prompt run keeps it, and a prompt
// answered with anything but an exact "yes" keeps it.
//
// So --no-prompt on its own is safe by construction: it silences the questions and
// takes the conservative answer to each. Losing the data always takes an explicit
// --delete-*, whatever else is on the command line.
func confirmLayer(a *App, l layer) bool {
	if a.deleteLayer {
		return true
	}
	if a.noPrompt || !interactive(a) {
		return false
	}
	return promptYes(promptSource(a), os.Stderr, fmt.Sprintf(
		"Also delete %s? %s.\nThis cannot be undone. Type 'yes' to delete, anything else keeps it: ",
		l.what, l.why))
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

// confirmDelete gates every command that destroys something. --no-prompt confirms
// without asking; an interactive session is prompted [y/N]; a non-TTY without
// --no-prompt declines loudly, because nothing should be destroyed unattended
// without someone having said so on the command line.
func confirmDelete(a *App, what string) bool {
	if a.noPrompt {
		return true
	}
	if !interactive(a) {
		warn("refusing to delete %s without confirmation; pass --no-prompt to proceed", what)
		return false
	}
	return promptYesNo(promptSource(a), os.Stderr, fmt.Sprintf("Delete %s? [y/N] ", what))
}

// addRestartFlag wires --restart onto the deploy command. Deliberately separate
// from --no-prompt: bouncing a live broker to apply a changed artifact is its own
// explicit decision, the same way deleting its data is.
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
