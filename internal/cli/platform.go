package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"solace/internal/config"
)

// platformAnnotation lists the platforms a command (or, as a flag annotation, a
// flag) applies to, comma-joined. Absent means every platform, so only the
// exceptions carry it and a plain shared command needs no ceremony.
//
// The tree is one static shape on every platform: help and completion render it
// without an env file (completion must never read one), so applicability is
// something the help text SAYS and the pre-run check ENFORCES, never something
// the tree hides. A command that quietly vanished depending on the env file
// would make `--help` output unquotable in a runbook.
const platformAnnotation = "solace_platforms"

// onlyOn tags c as applicable only on the given platforms and says so in its
// Short text, then returns it so registration can wrap inline.
func onlyOn(c *cobra.Command, platforms ...config.Platform) *cobra.Command {
	if c.Annotations == nil {
		c.Annotations = map[string]string{}
	}
	c.Annotations[platformAnnotation] = config.JoinPlatforms(platforms)
	if s := platformSuffix(platforms); s != "" {
		c.Short += s
	}
	return c
}

// commandPlatforms reports the platforms c applies to. An untagged command
// applies everywhere.
func commandPlatforms(c *cobra.Command) []config.Platform {
	return parsePlatformList(c.Annotations[platformAnnotation])
}

// parsePlatformList reads a comma-joined annotation value. An empty value means
// every platform. Values are written by onlyOn from real constants, so an
// unparseable name is a wiring bug, not user input: it is dropped rather than
// widening the set, which fails closed.
func parsePlatformList(s string) []config.Platform {
	if s == "" {
		return config.Platforms()
	}
	var out []config.Platform
	for _, name := range strings.Split(s, ",") {
		if p, err := config.ParsePlatform(strings.TrimSpace(name)); err == nil && p != "" {
			out = append(out, p)
		}
	}
	return out
}

func supportsPlatform(ps []config.Platform, p config.Platform) bool {
	for _, q := range ps {
		if q == p {
			return true
		}
	}
	return false
}

// platformSuffix renders the " (kubernetes only)" tail appended to a scoped
// command's Short text. A command applying everywhere gets nothing.
func platformSuffix(ps []config.Platform) string {
	if len(ps) == 0 || len(ps) == len(config.Platforms()) {
		return ""
	}
	names := make([]string, len(ps))
	for i, p := range ps {
		names[i] = string(p)
	}
	return " (" + strings.Join(names, "/") + " only)"
}

// unsupportedErr is the loud refusal when a command that exists in the tree does
// not apply to the platform this env file selected. It names the platforms that
// do have it, so the message is a pointer rather than a dead end.
func unsupportedErr(cmd *cobra.Command, p config.Platform) error {
	return fmt.Errorf("%q is not supported on %s: it applies to %s",
		cmd.CommandPath(), p, config.JoinPlatforms(commandPlatforms(cmd)))
}

// flagOnlyOn scopes an already-declared flag to a subset of platforms: the flag
// is registered on every platform so the tree and its generated reference stay
// one static shape, and passing it where it means nothing is refused at pre-run.
// The scope is recorded on the flag itself (pflag carries per-flag annotations),
// which keeps it beside the declaration instead of in a table to be kept in sync.
func flagOnlyOn(c *cobra.Command, name string, platforms ...config.Platform) {
	if err := c.Flags().SetAnnotation(name, platformAnnotation, []string{config.JoinPlatforms(platforms)}); err != nil {
		// Only reachable if the flag was not declared first, which is a wiring
		// bug in this package rather than anything a user can cause.
		panic(fmt.Sprintf("flagOnlyOn %s --%s: %v", c.Name(), name, err))
	}
	if f := c.Flags().Lookup(name); f != nil {
		if s := platformSuffix(platforms); s != "" {
			f.Usage += s
		}
	}
}

// checkFlagPlatforms refuses a platform-scoped flag that was actually passed on
// a platform it means nothing to. Visit walks only the flags the user set, so an
// untouched flag never trips this.
func checkFlagPlatforms(cmd *cobra.Command, p config.Platform) error {
	var bad error
	cmd.Flags().Visit(func(f *pflag.Flag) {
		if bad != nil {
			return
		}
		vals := f.Annotations[platformAnnotation]
		if len(vals) == 0 {
			return
		}
		allowed := parsePlatformList(vals[0])
		if !supportsPlatform(allowed, p) {
			bad = fmt.Errorf("--%s is not supported on %s: it applies to %s",
				f.Name, p, config.JoinPlatforms(allowed))
		}
	})
	return bad
}

// rejectRole refuses a [role] positional on a platform that does not act on it.
// Accepting and ignoring it is the dangerous alternative: on a container host
// there is one broker per machine and the transport ignores the role entirely,
// so `logs backup` would print the local broker's logs and look like it had
// reached the backup.
func rejectRole(p config.Platform, arg string, actOn ...config.Platform) error {
	if arg == "" || supportsPlatform(actOn, p) {
		return nil
	}
	return fmt.Errorf("a [role] argument is not supported on %s: it applies to %s",
		p, config.JoinPlatforms(actOn))
}

// wireExec installs the shared pre-run on a runnable command and gives it the
// --allow-command escape hatch.
//
// It is per-command rather than inherited from a parent because cobra runs the
// nearest ancestor's PersistentPreRunE, and the nearest ancestor of the hidden
// __complete command is root: a hook on root would parse the env file on every
// TAB press. A non-inherited PreRunE on the leaf itself is never in __complete's
// ancestry, so completion stays inert (completion.go).
func wireExec(app *App, c *cobra.Command) *cobra.Command {
	addAllowCommandFlag(c, app)
	c.PreRunE = func(cmd *cobra.Command, _ []string) error { return prepare(app, cmd) }
	return c
}

// prepare is everything that must hold before a command touches the world:
// the flags make sense on their own, a platform is settled, the command and its
// flags apply to that platform, and only then is the env file loaded.
//
// The order is deliberate. The check that needs nothing but the command line runs
// first, so a plain misuse is reported as itself rather than behind whatever the
// env file happens to be wrong about -- `--allow-command` on a render-only command
// is a usage error whether or not an env file even exists.
func prepare(app *App, cmd *cobra.Command) error {
	if err := checkAllowCommand(cmd, app); err != nil {
		return err
	}
	p, err := resolvePlatform(app)
	if err != nil {
		return err
	}
	app.Platform = p
	if !supportsPlatform(commandPlatforms(cmd), p) {
		return unsupportedErr(cmd, p)
	}
	if err := checkFlagPlatforms(cmd, p); err != nil {
		return err
	}
	return app.load(cmd)
}

// resolvePlatform settles which platform this invocation drives. --platform wins
// and must name a section the file actually declares; otherwise the env file's
// own sections decide, and only a genuine ambiguity asks the operator.
//
// The env file is the source of truth on purpose. It already describes one
// deployment, so making the platform a thing you repeat on the command line
// would be a second place for the same fact to be wrong.
func resolvePlatform(a *App) (config.Platform, error) {
	path, err := config.ResolveEnvPath(a.BaseDir, a.EnvName)
	if err != nil {
		return "", err
	}
	found, err := config.DetectPlatforms(path)
	if err != nil {
		return "", err
	}
	if a.PlatformFlag != "" {
		p, err := config.ParsePlatform(a.PlatformFlag)
		if err != nil {
			return "", err
		}
		if !supportsPlatform(found, p) {
			return "", fmt.Errorf("--platform %s: env file %q declares no %s: section (it declares %s)",
				p, path, p, declaredList(found))
		}
		step("platform: %s (--platform)", p)
		return p, nil
	}
	switch len(found) {
	case 0:
		return "", fmt.Errorf("env file %q declares no platform section: add kubernetes:, docker: or podman: "+
			"(an empty section is enough, e.g. `docker: {}`) -- see env/sample.yaml", path)
	case 1:
		step("platform: %s (from %s)", found[0], path)
		return found[0], nil
	}
	if !interactive(a) {
		return "", fmt.Errorf("env file %q declares more than one platform section (%s): "+
			"pass --platform kubernetes|docker|podman (kube|dk|pm) to choose", path, config.JoinPlatforms(found))
	}
	p, err := promptPlatform(a, found)
	if err != nil {
		return "", err
	}
	step("platform: %s (selected)", p)
	return p, nil
}

// declaredList renders what a file did declare, for the error raised when
// --platform named something else. "nothing" rather than an empty string keeps
// the sentence readable in the case where the file declares no platform at all.
func declaredList(found []config.Platform) string {
	if len(found) == 0 {
		return "nothing"
	}
	return config.JoinPlatforms(found)
}

// promptPlatform asks which of the declared platforms to drive. It is reached
// only from an interactive terminal: a piped or scripted run fails loudly
// instead, because a deployment target guessed on a machine's behalf is exactly
// the decision that should not be silent.
func promptPlatform(a *App, found []config.Platform) (config.Platform, error) {
	fmt.Fprintf(os.Stderr, "This env file describes more than one platform:\n")
	for i, p := range found {
		fmt.Fprintf(os.Stderr, "  %d) %s\n", i+1, p)
	}
	answer := promptLine(promptSource(a), os.Stderr,
		fmt.Sprintf("Which platform? [1-%d] ", len(found)))
	n, err := strconv.Atoi(strings.TrimSpace(answer))
	if err != nil || n < 1 || n > len(found) {
		return "", fmt.Errorf("no platform selected (got %q): pass --platform %s to choose without being asked",
			answer, config.JoinPlatforms(found))
	}
	return found[n-1], nil
}

// platformOps pairs a kubernetes implementation with the container one, which
// docker and podman always share -- they are one host-local platform with one
// set of ops (internal/container). A nil half means the operation does not exist
// there, which is what makes the map the single source of truth: the same
// entries that dispatch the call also tag the command, so help text and the
// pre-run refusal can never disagree with what would actually run.
func platformOps(k8s, container opFunc) map[config.Platform]opFunc {
	m := map[config.Platform]opFunc{}
	if k8s != nil {
		m[config.K8s] = k8s
	}
	if container != nil {
		m[config.Docker] = container
		m[config.Podman] = container
	}
	return m
}

// supported lists the platforms an ops map implements, in Platforms() order.
func supported(impls map[config.Platform]opFunc) []config.Platform {
	var out []config.Platform
	for _, p := range config.Platforms() {
		if impls[p] != nil {
			out = append(out, p)
		}
	}
	return out
}

// dispatchLeaf builds a no-arg leaf whose implementation differs by platform.
func dispatchLeaf(app *App, use, short string, impls map[config.Platform]opFunc) *cobra.Command {
	c := leaf(app, use, short, func(a *App) error { return dispatch(impls, a) })
	return onlyOn(c, supported(impls)...)
}

// dispatch runs the implementation for the resolved platform. Reaching the nil
// case would mean the command's own annotation disagreed with this map, which
// prepare already refuses -- so it fails loudly rather than silently doing
// nothing if the two ever drift.
func dispatch(impls map[config.Platform]opFunc, a *App) error {
	fn := impls[a.Platform]
	if fn == nil {
		return fmt.Errorf("no implementation for platform %s", a.Platform)
	}
	return fn(a)
}
