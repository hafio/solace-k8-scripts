package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// commandAliases maps a command's NAME to its approved short forms.
//
// Keyed by name, not by path, and applied by a tree walk: a word means the same
// thing wherever it appears, so `broker` is `br` under every verb that takes it and
// `operator` is `op` under every verb that takes it. That is the property being
// bought -- one abbreviation per concept, not one per command path.
//
// Three deliberate absences:
//
//   - `start` and `stop` have none. Any two-letter form is ambiguous between them
//     and `status`, and the one place a slip costs an outage is not where to save a
//     keystroke.
//   - `rollout`/`rollback` were considered as names and rejected; the verbs that
//     replaced them (`deploy all`, `remove all`) inherit `dp`/`rm` from their verb,
//     so `dp all` and `rm all` already work.
//   - `remove` is `rm` even though it is the most destructive verb. It takes a noun
//     before it does anything, so `rm` on its own prints help rather than removing a
//     broker -- which is exactly what makes the short form safe to hand out.
//
// `delete` is deliberately NOT an alias for `remove`: one removal word, everywhere.
var commandAliases = map[string][]string{
	// verbs
	"check":       {"ck"},
	"config":      {"cfg"},
	"convert":     {"cv"},
	"copy":        {"cp"},
	"deploy":      {"dp"},
	"diagnostics": {"diag"},
	"generate":    {"gen"},
	"logs":        {"lg"},
	"prepare":     {"pre"},
	"remove":      {"rm"},
	"restart":     {"rs"},
	"shell":       {"sh"},
	"status":      {"sts"},
	"version":     {"ver"},

	// nouns -- these ride along under every verb that takes them
	"broker":   {"br"},
	"operator": {"op"},
}

// applyAliases attaches every approved short form to the tree, and panics if one
// would collide with a sibling's name or with another sibling's alias.
//
// The panic is the point: a collision is a wiring bug in this package, not a
// runtime condition, and cobra resolves a duplicate silently by taking whichever
// command it reaches first. Failing at construction means every test that builds
// the tree is also a collision test. It matches how registerFlagCompletion and
// flagOnlyOn already treat their own wiring bugs.
func applyAliases(root *cobra.Command) {
	var walk func(*cobra.Command)
	walk = func(parent *cobra.Command) {
		// taken maps every word already claimed among these siblings to the command
		// that claimed it, so the panic can name both sides of a collision.
		taken := map[string]string{}
		claim := func(c *cobra.Command, word string) {
			if owner, dup := taken[word]; dup {
				panic(fmt.Sprintf("alias collision under %q: %q is claimed by both %q and %q",
					parent.CommandPath(), word, owner, c.Name()))
			}
			taken[word] = c.Name()
		}
		for _, c := range parent.Commands() {
			claim(c, c.Name())
			for _, a := range c.Aliases {
				claim(c, a)
			}
		}
		for _, c := range parent.Commands() {
			for _, a := range commandAliases[c.Name()] {
				claim(c, a)
				// Append rather than replace: a command may already carry an alias of
				// its own for a reason this table knows nothing about.
				c.Aliases = append(c.Aliases, a)
			}
			walk(c)
		}
	}
	walk(root)
}
