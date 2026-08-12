package cli

import (
	"fmt"
	"os"

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
	Yes     bool   // --yes: skip confirmations (never implies data purge)

	// The three --gen-*-only flags each replace a command's real work with a
	// different rendering. Exactly one may be set (checkGenFlags enforces it).
	GenOnly        bool // --gen-only: the deployment artifact (CR / compose / quadlet)
	GenSecretsOnly bool // --gen-secrets-only: the secret-creation artifact
	GenEnvOnly     bool // --gen-env-only: the container env file (container-only)

	Platform config.Platform
	Cfg      *config.Config
	Runner   engine.Runner
	envPath  string // resolved env-file path (container PrepHost writes the PSK back here)

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
// valid env (cobra skips PreRun for --help).
func (a *App) load() error {
	path, err := config.ResolveEnvPath(a.BaseDir, a.EnvName)
	if err != nil {
		return err
	}
	a.envPath = path
	// Echo the winner: a file in the base dir shadows the env/ copy of the same
	// name, and that has to be visible rather than silent.
	step("env file: %s", path)
	cfg, err := config.Load(path, a.Platform)
	if err != nil {
		return err
	}
	a.Cfg = cfg
	if a.DryRun {
		a.Runner = engine.Echo{W: os.Stdout}
	} else {
		a.Runner = engine.Exec{}
	}
	return nil
}

// warn prints a non-fatal warning to stderr in the house [WARN] style.
func warn(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "[WARN] "+format+"\n", args...)
}

// step prints a progress line to stderr so it never pollutes rendered stdout.
func step(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "==> "+format+"\n", args...)
}
