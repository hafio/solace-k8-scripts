package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"solace/internal/config"
	"solace/internal/engine"
)

// App is the shared context threaded through every command: the parsed global
// flags, the resolved config, and the Runner used for all external commands.
// It replaces the bash "source 000-env.sh" bootstrap — one load, reused by the
// whole command tree (§4: explicit context, no globals).
type App struct {
	EnvName  string // --env value
	BaseDir  string // dir that holds env/ (defaults to CWD)
	GenOnly  bool   // --gen: render artifact, don't apply
	DryRun   bool   // --dry-run: echo commands instead of running them
	Yes      bool   // --yes: skip confirmations (never implies data purge)

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
}

// load resolves the env file for the app's platform and builds the config +
// runner. Called from each platform command's PersistentPreRunE so config
// errors surface before any subcommand work, and help still works without a
// valid env (cobra skips PreRun for --help).
func (a *App) load() error {
	path := a.resolveEnvPath()
	a.envPath = path
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

// resolveEnvPath maps --env to a file. A value that looks like a path (has a
// separator or a .yaml/.yml suffix) is used as-is; otherwise it resolves to
// <BaseDir>/env/<name>.yaml.
func (a *App) resolveEnvPath() string {
	name := a.EnvName
	if name == "" {
		name = "default"
	}
	if strings.ContainsAny(name, "/\\") || strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
		return name
	}
	base := a.BaseDir
	if base == "" {
		base = "."
	}
	return filepath.Join(base, "env", name+".yaml")
}

// warn prints a non-fatal warning to stderr in the house [WARN] style.
func warn(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "[WARN] "+format+"\n", args...)
}

// step prints a progress line to stderr so it never pollutes rendered stdout.
func step(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "==> "+format+"\n", args...)
}
