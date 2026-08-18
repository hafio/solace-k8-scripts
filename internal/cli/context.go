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
	DryRun  bool   // --dry-run: echo commands instead of running them
	Verbose bool   // -v/--verbose: announce every external command as it runs
	Yes     bool   // --yes: skip confirmations (never implies data purge)

	// The three --gen-*-only flags each replace a command's real work with a
	// different rendering. Exactly one may be set (checkGenFlags enforces it).
	GenOnly        bool // --gen-only: the deployment artifact (CR / compose / quadlet)
	GenSecretsOnly bool // --gen-secrets-only: the secret-creation artifact
	GenEnvOnly     bool // --gen-env-only: the container env file (container-only)

	// AllowCommand is the repeatable --allow-command escape hatch: binaries the
	// OPERATOR approved for the config's platform command, for this invocation
	// only. It reaches config through Load's argument list, never through the
	// schema, so an env file has no way to extend its own allowlist.
	AllowCommand []string

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

	// Command-local flag scratch space. Only one command runs per invocation,
	// so sharing these on the app context is safe.
	keepYAML bool   // k8s deploy --keep-yaml
	purge    bool   // delete/down --purge (alias --clear-data)
	keepData bool   // delete/down --keep-data
	pod      string // --pod role override for exec-cli/copy
	destDir  string // copy into --dir
	days     int    // verify diagnostics --days
	restart  bool   // container deploy/up --restart (bounce a running broker)
}

// load resolves the env file for the app's platform and builds the config +
// runner. Called from each platform command's PersistentPreRunE so config
// errors surface before any subcommand work, and help still works without a
// valid env (cobra skips PreRun for --help). cmd is the command about to run:
// announceCommands needs it to stay quiet where nothing will execute.
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
	if a.DryRun {
		// No announcement here on purpose: --dry-run is documented to need no
		// kubectl/docker/podman binary installed at all, and Echo prints every command
		// it would run anyway.
		a.Runner = engine.Echo{W: os.Stdout}
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
// The set is exactly the four fields the execution guard exists for (k8s.runtime,
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
