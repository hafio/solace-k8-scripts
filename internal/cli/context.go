package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"solace/internal/config"
	"solace/internal/engine"
)

// App is the shared context threaded through every command: the parsed global
// flags, the resolved config, and the Runner used for all external commands.
// It replaces the bash "source 000-env.sh" bootstrap — one load, reused by the
// whole command tree (§4: explicit context, no globals).
type App struct {
	EnvName string // -e/--env value: an env file name, or a path
	BaseDir string // dir searched for the env file, and holding env/ (defaults to CWD)
	Verbose bool   // -v/--verbose: announce every external command as it runs

	// AllowCommand is the repeatable --allow-command escape hatch: binaries the
	// OPERATOR approved for the config's platform command, for this invocation
	// only. It reaches config through Load's argument list, never through the
	// schema, so an env file has no way to extend its own allowlist.
	AllowCommand []string

	// PlatformFlag is the raw --platform value, still in whatever spelling was
	// typed (canonical or the kube/dk/pm abbreviations). Platform below is the
	// resolved one, settled by resolvePlatform from this flag or, when it is
	// empty, from the platform sections the env file declares.
	PlatformFlag string

	Platform config.Platform
	Cfg      *config.Config
	Runner   engine.Runner
	envPath  string // resolved env-file path (container PrepHost writes the PSK back here)

	// Prompt seams, in the spirit of Manager.Resolve/GenPSK/Geteuid: the confirm
	// helpers gate destructive actions on an interactive terminal, and a test cannot
	// supply one. Interactive nil means "ask isTTY(os.Stdin)" and PromptIn nil means
	// os.Stdin, so production behaviour is unchanged and only tests set them.
	Interactive func() bool
	PromptIn    io.Reader

	// NewRunner builds the Runner every command executes through. It exists as a
	// seam for the same reason Interactive/PromptIn do: a test cannot supply a
	// cluster or a container engine, and the property worth asserting is which
	// argv a command would issue. Nil means the production runner, so nothing but
	// a test ever substitutes one -- there is no user-facing way to make this tool
	// print commands instead of running them, because `generate` is how you look
	// at an artifact before applying it.
	NewRunner func(a *App) engine.Runner

	// Command-local flag scratch space. Only one command runs per invocation,
	// so sharing these on the app context is safe.
	deleteLayer bool   // remove --delete-data / --delete-crd (take the retained layer too)
	noPrompt    bool   // --no-prompt: ask nothing, and take the safe answer to each question
	all         bool   // status broker --all (every broker in the cluster)
	detail      bool   // status --detail (static artifacts, not just running ones)
	pod         string // --pod role override for cli --input / copy
	destDir     string // copy into --dir
	inputFile   string // cli --input/-i: run this CLI script instead of an interactive session
	days        int    // diagnostics --days
	restart     bool   // deploy broker --restart (bounce a running broker)
}

// load resolves the env file for the app's platform and builds the config +
// runner. Called last in every runnable command's PreRunE (prepare, platform.go)
// so config errors surface before any work, and help still works without a valid
// env (cobra skips PreRun for --help). By then a.Platform is settled, which is
// what config.Load needs to scope its defaults and validation. cmd is the command
// about to run: announceCommands needs it to stay quiet where nothing executes.
func (a *App) load(cmd *cobra.Command) error {
	path, err := config.ResolveEnvPath(a.BaseDir, a.EnvName)
	if err != nil {
		return err
	}
	a.envPath = path
	// Echo the winner: a file in the base dir shadows the env/ copy of the same
	// name, and that has to be visible rather than silent.
	step("env file: %s", path)
	// The operator's --allow-command approvals go in here, before Validate runs
	// inside Load, so the execution guard sees the same allowlist the executors
	// will. Passing them as an argument is what keeps them out of the schema.
	cfg, err := config.Load(path, a.Platform, a.AllowCommand...)
	if err != nil {
		return err
	}
	a.Cfg = cfg
	if a.NewRunner != nil {
		// A test-supplied runner: it does not execute, so there are no binaries
		// worth resolving and announcing.
		a.Runner = a.NewRunner(a)
		return nil
	}
	a.Runner = engine.NewExec(os.Stderr, a.Verbose)
	if a.willExecute(cmd) {
		a.announceCommands()
	}
	return nil
}

// announceCommands names the binaries this env file chose, resolved, before any of them
// runs -- so the information sits with `==> env file:` in the preamble instead of
// repeating itself between report lines on every call.
//
// The set is exactly the four fields the execution guard exists for (kubernetes.runtime,
// docker.runtime, podman.runtime, docker.compose), read through their guarded accessors:
// those are the binaries CONFIG TEXT chose, which is the whole reason their location is
// worth showing. The tool's own fixed helpers (mkdir, chown, rm, sh, systemctl) were
// chosen here, not by an env file, so they are announced only under --verbose.
//
// It never fails. A name that resolves nowhere is skipped silently: this is a report,
// and the first real execution already fails with engine.Resolve's own actionable
// message (not found on PATH, or the current-directory refusal).
func (a *App) announceCommands() {
	var cmds []config.Command
	switch a.Platform {
	case config.K8s:
		if c, err := a.Cfg.ClusterCommand(); err == nil {
			cmds = append(cmds, c)
		}
	case config.Docker, config.Podman:
		if c, err := a.Cfg.RuntimeCommand(a.Platform); err == nil {
			cmds = append(cmds, c)
		}
		if a.Platform == config.Docker {
			if c, err := a.Cfg.ComposeCommand(); err == nil {
				cmds = append(cmds, c)
			}
		}
	}
	// Keyed by resolved path, not by name: `docker` and a derived `docker compose` are
	// one binary and deserve one line, while a standalone docker-compose gets its own.
	seen := make(map[string]bool, len(cmds))
	for _, c := range cmds {
		path, err := engine.Resolve(c.Name())
		if err != nil || seen[path] {
			continue
		}
		seen[path] = true
		step("using %s: %s", c.Name(), path)
	}
}

// warn prints a non-fatal warning to stderr in the house [WARN] style.
func warn(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "[WARN] "+format+"\n", args...)
}

// step prints a progress line to stderr so it never pollutes rendered stdout.
func step(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "==> "+format+"\n", args...)
}
