package cli

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// The abbreviations are a convenience layer over a tree that is already settled, so
// what these tests protect is not the individual words but the two properties that
// make the words safe to hand out: an alias reaches exactly the command its long
// form does, and no alias can ever be ambiguous with a sibling.

// TestAliasesResolveToTheCanonicalCommand pins equivalence by resolution rather than
// by running anything: cobra's Find is what dispatch itself uses, so proving both
// paths land on the same *cobra.Command proves they behave identically without
// needing a cluster, an env file, or a runner.
func TestAliasesResolveToTheCanonicalCommand(t *testing.T) {
	for _, tc := range []struct{ alias, canonical []string }{
		{[]string{"rm", "br"}, []string{"remove", "broker"}},
		{[]string{"rm", "op"}, []string{"remove", "operator"}},
		{[]string{"rm", "all"}, []string{"remove", "all"}},
		{[]string{"dp", "br"}, []string{"deploy", "broker"}},
		{[]string{"dp", "op"}, []string{"deploy", "operator"}},
		{[]string{"sts", "br"}, []string{"status", "broker"}},
		{[]string{"sts", "op"}, []string{"status", "operator"}},
		{[]string{"lg", "br"}, []string{"logs", "broker"}},
		{[]string{"cfg", "leader"}, []string{"config", "leader"}},
		{[]string{"cfg", "apply", "server-cert"}, []string{"config", "apply", "server-cert"}},
		{[]string{"ck", "deploy"}, []string{"check", "deploy"}},
		{[]string{"pre", "all"}, []string{"prepare", "all"}},
		{[]string{"rs", "br"}, []string{"restart", "broker"}},
		{[]string{"gen", "br"}, []string{"generate", "broker"}},
		{[]string{"diag"}, []string{"diagnostics"}},
		{[]string{"cv"}, []string{"convert"}},
		{[]string{"ver"}, []string{"version"}},
		{[]string{"sh"}, []string{"shell"}},
		{[]string{"cp", "from"}, []string{"copy", "from"}},
	} {
		t.Run(strings.Join(tc.alias, " "), func(t *testing.T) {
			root := newRootCmd(&App{})
			viaAlias, _, err := root.Find(tc.alias)
			if err != nil {
				t.Fatalf("resolving %v: %v", tc.alias, err)
			}
			root2 := newRootCmd(&App{})
			viaName, _, err := root2.Find(tc.canonical)
			if err != nil {
				t.Fatalf("resolving %v: %v", tc.canonical, err)
			}
			if viaAlias.CommandPath() != viaName.CommandPath() {
				t.Errorf("%v resolved to %q, want the same command as %v (%q)",
					tc.alias, viaAlias.CommandPath(), tc.canonical, viaName.CommandPath())
			}
		})
	}
}

// TestAliasesDoNotCollide walks the real tree and proves no two siblings answer to
// the same word. applyAliases panics on a collision at construction time, so this
// is the second line of defence -- and the one that would still catch a collision
// introduced by a command's own hand-written Aliases rather than by the table.
func TestAliasesDoNotCollide(t *testing.T) {
	root := newRootCmd(&App{})
	var walk func(*cobra.Command)
	walk = func(parent *cobra.Command) {
		seen := map[string]string{}
		for _, c := range parent.Commands() {
			for _, word := range append([]string{c.Name()}, c.Aliases...) {
				if owner, dup := seen[word]; dup {
					t.Errorf("under %q, %q is claimed by both %q and %q",
						parent.CommandPath(), word, owner, c.Name())
				}
				seen[word] = c.Name()
			}
			walk(c)
		}
	}
	walk(root)
}

// TestEveryAliasEntryIsLive catches the quiet failure mode of a name-keyed table:
// a command gets renamed, its entry stops matching anything, and the abbreviation
// silently disappears while the table still claims to provide it.
func TestEveryAliasEntryIsLive(t *testing.T) {
	root := newRootCmd(&App{})
	live := map[string]bool{}
	var walk func(*cobra.Command)
	walk = func(parent *cobra.Command) {
		for _, c := range parent.Commands() {
			live[c.Name()] = true
			walk(c)
		}
	}
	walk(root)
	for name := range commandAliases {
		if !live[name] {
			t.Errorf("commandAliases has an entry for %q, which is not a command in the tree", name)
		}
	}
}

// TestDangerousVerbsHaveNoBareAlias is the safety property behind giving the
// removal verb a two-letter form at all: `rm` reaches a verb that acts on nothing
// until it is given a noun. The check is that these verbs are GROUPS -- their RunE
// exists only to print help or reject an unknown noun (group(), commands.go), so
// no argument-less invocation can destroy anything. If one ever became a real
// command, this fails, which is the moment to reconsider the abbreviation rather
// than after someone loses a broker to a typo.
func TestDangerousVerbsHaveNoBareAlias(t *testing.T) {
	root := newRootCmd(&App{})
	for _, name := range []string{"remove", "deploy", "config", "start", "stop", "restart"} {
		c, _, err := root.Find([]string{name})
		if err != nil {
			t.Fatalf("finding %q: %v", name, err)
		}
		if c.Annotations[groupAnnotation] != "true" {
			t.Errorf("%q is not a verb group; a verb that owns objects must not act on its own", name)
		}
		if !c.HasSubCommands() {
			t.Errorf("%q has no subcommands, so there is no noun to name", name)
		}
	}
}

// TestGroupsRejectAnUnknownNoun pins the reason groups are runnable at all. Cobra
// answers a NON-runnable command by printing help and exiting 0 whatever arguments
// it got, so a mistyped noun on a destructive verb would report success having done
// nothing. Bare still prints help and succeeds; a word the verb does not know fails.
func TestGroupsRejectAnUnknownNoun(t *testing.T) {
	for _, verb := range []string{"remove", "deploy", "generate", "config", "status"} {
		t.Run(verb, func(t *testing.T) {
			if _, err := runRoot(t, []string{verb}); err != nil {
				t.Errorf("bare %q err = %v, want help and success", verb, err)
			}
			if _, err := runRoot(t, []string{verb, "bogus"}); err == nil {
				t.Errorf("%q bogus err = nil, want a loud refusal naming the unknown word", verb)
			}
		})
	}
}

// TestStartStopHaveNoAlias pins the one deliberate omission. `st` could mean start,
// stop or status, and the cost of guessing wrong between the first two is an
// outage, so none of them gets a two-letter form that could be confused.
func TestStartStopHaveNoAlias(t *testing.T) {
	for _, name := range []string{"start", "stop"} {
		if got, ok := commandAliases[name]; ok {
			t.Errorf("commandAliases[%q] = %v, want no alias: it would be ambiguous with the other", name, got)
		}
	}
	root := newRootCmd(&App{})
	for _, name := range []string{"start", "stop"} {
		c, _, err := root.Find([]string{name})
		if err != nil {
			t.Fatalf("finding %q: %v", name, err)
		}
		if len(c.Aliases) > 0 {
			t.Errorf("%q has aliases %v, want none", name, c.Aliases)
		}
	}
}
