package cli

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"solace/internal/config"
)

// update regenerates docs/commands.md from the live command tree. The doc is a
// golden, so this test is also the generator -- the same arrangement the render
// and k8s packages use for their manifest goldens.
var update = flag.Bool("update", false, "regenerate docs/commands.md from the command tree")

// commandDocPath is anchored on the package dir, which is `go test`'s cwd.
const commandDocPath = "../../docs/commands.md"

// TestCommandDocs is the drift gate: a new command, a renamed flag, or an edited
// Short string fails `test` until the reference is regenerated. Nothing else in
// the suite covers Short text or the whole tree.
func TestCommandDocs(t *testing.T) {
	got := renderCommandDocs(newRootCmd(&App{}))

	if *update {
		if err := os.WriteFile(commandDocPath, got, 0o644); err != nil {
			t.Fatalf("write %s: %v", commandDocPath, err)
		}
		return
	}
	want, err := os.ReadFile(commandDocPath)
	if err != nil {
		t.Fatalf("read %s (regenerate: go test ./internal/cli -update): %v", commandDocPath, err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("%s is stale -- regenerate with: go test ./internal/cli -update\n%s",
			commandDocPath, firstDiff(got, want))
	}
}

// firstDiff names the first line that differs, so the failure says what changed
// instead of dumping two whole documents.
func firstDiff(got, want []byte) string {
	g, w := strings.Split(string(got), "\n"), strings.Split(string(want), "\n")
	for i := 0; i < len(g) || i < len(w); i++ {
		gl, wl := "", ""
		if i < len(g) {
			gl = g[i]
		}
		if i < len(w) {
			wl = w[i]
		}
		if gl != wl {
			return fmt.Sprintf("first difference at line %d:\n  generated: %q\n  committed: %q", i+1, gl, wl)
		}
	}
	return ""
}

// renderCommandDocs renders the whole command tree as markdown. It is the single
// source of truth for docs/commands.md: every fact in the reference is read off
// the cobra tree, so the doc cannot describe a command that does not exist.
func renderCommandDocs(root *cobra.Command) []byte {
	var b strings.Builder

	b.WriteString("# Command reference\n\n")
	b.WriteString("Every command `solace-util` exposes, with its arguments and flags.\n\n")
	b.WriteString("**Generated from the command tree -- do not edit by hand.** Regenerate after any\n")
	b.WriteString("command, flag, or description change:\n\n")
	b.WriteString("```\ngo test ./internal/cli -update\n```\n\n")
	b.WriteString("The `test` task fails while this file is stale, so it cannot drift from the code.\n\n")

	b.WriteString("## Tree\n\n```\n")
	walkCommands(root, 0, func(c *cobra.Command, depth int) {
		fmt.Fprintf(&b, "%s%s\n", strings.Repeat("  ", depth), c.Use)
	})
	b.WriteString("```\n\n")

	b.WriteString("## Global flags\n\n")
	b.WriteString("Inherited by every command.\n\n")
	writeFlagTable(&b, root.PersistentFlags())

	b.WriteString("## Commands\n")
	walkCommands(root, 0, func(c *cobra.Command, _ int) {
		writeCommand(&b, c, c == root)
	})
	return []byte(b.String())
}

// walkCommands visits the tree depth-first. cobra sorts Commands() by name, so
// the order is stable without sorting here.
func walkCommands(c *cobra.Command, depth int, fn func(*cobra.Command, int)) {
	fn(c, depth)
	for _, sub := range c.Commands() {
		if !sub.IsAvailableCommand() {
			continue
		}
		walkCommands(sub, depth+1, fn)
	}
}

// writeCommand renders one command. isRoot suppresses the flag table, because
// the root's local flags are the global ones already listed above.
func writeCommand(b *strings.Builder, c *cobra.Command, isRoot bool) {
	fmt.Fprintf(b, "\n### %s\n\n", c.CommandPath())
	if c.Short != "" {
		fmt.Fprintf(b, "%s\n\n", c.Short)
	}
	if c.Long != "" && c.Long != c.Short {
		fmt.Fprintf(b, "%s\n\n", c.Long)
	}
	fmt.Fprintf(b, "```\n%s\n```\n\n", c.UseLine())

	if subs := availableSubs(c); len(subs) > 0 {
		fmt.Fprintf(b, "Subcommands: %s\n\n", strings.Join(subs, ", "))
	}
	if len(c.Aliases) > 0 {
		fmt.Fprintf(b, "Also available as: %s\n\n", strings.Join(c.Aliases, ", "))
	}
	// The tree is one shape on every platform, so applicability is a fact about a
	// command rather than something the reader can infer from where it sits. It is
	// read off the annotation that also drives the refusal, so the reference cannot
	// promise a platform the command would reject.
	if v, ok := c.Annotations[platformAnnotation]; ok && v != config.JoinPlatforms(config.Platforms()) {
		fmt.Fprintf(b, "Applies to: %s. On any other platform this command fails rather than doing nothing.\n\n", v)
	}
	if !isRoot {
		writeFlagTable(b, c.NonInheritedFlags())
	}
}

func availableSubs(c *cobra.Command) []string {
	var out []string
	for _, sub := range c.Commands() {
		if sub.IsAvailableCommand() {
			out = append(out, "`"+sub.Name()+"`")
		}
	}
	return out
}

// writeFlagTable renders a flag set, or nothing when it is empty.
func writeFlagTable(b *strings.Builder, fs *pflag.FlagSet) {
	type row struct{ name, def, usage string }
	var rows []row
	fs.VisitAll(func(f *pflag.Flag) {
		if f.Hidden || f.Name == "help" {
			return
		}
		name := "`--" + f.Name + "`"
		if f.Shorthand != "" {
			name = "`-" + f.Shorthand + "`, " + name
		}
		def := "(none)"
		if f.DefValue != "" {
			def = "`" + f.DefValue + "`"
		}
		rows = append(rows, row{name, def, mdCell(f.Usage)})
	})
	if len(rows) == 0 {
		return
	}
	b.WriteString("| Flag | Default | Meaning |\n| --- | --- | --- |\n")
	for _, r := range rows {
		fmt.Fprintf(b, "| %s | %s | %s |\n", r.name, r.def, r.usage)
	}
	b.WriteString("\n")
}

// mdCell escapes the characters that would break a markdown table cell: a pipe
// ends the cell, and angle brackets are stripped as unknown HTML by renderers.
func mdCell(s string) string {
	return strings.NewReplacer("|", "\\|", "<", "&lt;", ">", "&gt;").Replace(s)
}
