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

// genAnnotation marks a command as able to honor --gen (render an artifact instead
// of applying it). The k8s PersistentPreRunE rejects --gen on any command without it.
const genAnnotation = "solace_gen_capable"

// genCapable tags a command as --gen-aware and returns it, so registration can wrap
// a command inline: genCapable(leaf(...)).
func genCapable(c *cobra.Command) *cobra.Command {
	if c.Annotations == nil {
		c.Annotations = map[string]string{}
	}
	c.Annotations[genAnnotation] = "true"
	return c
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
	if !isTTY(os.Stdin) {
		warn("refusing to delete %s without confirmation; pass --yes to proceed", what)
		return false
	}
	return promptYesNo(os.Stdin, os.Stderr, fmt.Sprintf("Delete %s? [y/N] ", what))
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
	if !isTTY(os.Stdin) {
		return false
	}
	return promptYes(os.Stdin, os.Stderr, "Also clear persistent data (PVCs)? This is irreversible. Type 'yes' to purge: ")
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
