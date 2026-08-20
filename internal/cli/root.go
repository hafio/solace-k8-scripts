package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"solace/internal/config"
)

// Execute builds the command tree and runs it. main() calls this.
func Execute() error {
	app := &App{}
	root := newRootCmd(app)
	return root.Execute()
}

func newRootCmd(app *App) *cobra.Command {
	root := &cobra.Command{
		Use:   "solace-util",
		Short: "Deploy and operate Solace PubSub+ brokers on Kubernetes, Docker, or Podman",
		Long: "solace-util is a single CLI for deploying and operating Solace PubSub+ Event Brokers.\n" +
			"It presents the same lifecycle verbs on every platform, and every verb names\n" +
			"what it acts on -- run a verb on its own to see what it can act on:\n\n" +
			"  check deploy -> prepare all -> deploy all     build it\n" +
			"  config ...                                    POST-DEPLOYMENT, over the broker CLI\n" +
			"  check semp-login / smoke redundancy           prove it works\n" +
			"  stop broker / start broker                    pause it without removing it\n" +
			"  remove all                                    tear it down\n\n" +
			"The operator is cluster-scoped and shared, so it is installed and removed on\n" +
			"its own: `deploy operator`, `remove operator`.\n\n" +
			"`generate` renders any artifact to stdout without applying it -- that is how\n" +
			"you see what a command would send before you send it.\n\n" +
			"Every command takes -e/--env <file>, searched in the current directory then\n" +
			"./env. The platform comes from that file: whichever of kubernetes:, docker:\n" +
			"or podman: it declares is the one driven. A file declaring more than one asks\n" +
			"which to use, and --platform kubernetes|docker|podman (kube|dk|pm) answers that\n" +
			"up front. A few commands apply to only one platform; their help says so.\n\n" +
			"Coming from the bash scripts? 'solace-util convert <bash-env-file>' turns an old\n" +
			"env file into the YAML this reads.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	// newCompletionCmd below replaces the one cobra would add during Execute, which
	// never appeared in the generated reference because that renders this tree
	// without executing it. Disabling cobra's is explicit rather than relying on it
	// standing down for a same-named command of ours.
	root.CompletionOptions.DisableDefaultCmd = true

	root.PersistentFlags().StringVarP(&app.EnvName, "env", "e", config.EnvFileDefault, "env file name, searched in the base dir then <base-dir>/env; a value with a directory is used as-is")
	root.PersistentFlags().StringVar(&app.BaseDir, "base-dir", "", "directory searched for the env file, and holding env/ (default: current directory)")
	root.PersistentFlags().BoolVarP(&app.Verbose, "verbose", "v", false, "announce every external command as it runs; by default the binaries this env file names are resolved and listed once, up front")
	root.PersistentFlags().StringVar(&app.PlatformFlag, "platform", "",
		"platform to drive: kubernetes (kube), docker (dk) or podman (pm). Default: the one the env file declares, "+
			"or a prompt if it declares several")

	// The persistent flags that take a value worth suggesting. An inherited
	// flag is the same *pflag.Flag in every subcommand, and cobra keys completion
	// functions by that pointer, so registering here covers the whole tree.
	registerFlagCompletion(root, "env", completeEnvFiles(app))
	registerFlagCompletion(root, "base-dir", completeDirs)
	registerFlagCompletion(root, "platform", completePlatforms)

	addCommands(root, app)
	root.AddCommand(
		newConvertCmd(app),
		newCompletionCmd(),
		newVersionCmd(),
	)
	// Last, so it sees the whole tree: the abbreviations are attached by walking it,
	// and the collision check that comes with them can only be complete once every
	// command is in place.
	applyAliases(root)
	return root
}

// notImplemented is a placeholder for handlers whose downstream package is not
// wired yet. It fails loud so a half-built command can never appear to succeed.
func notImplemented(name string) error {
	return fmt.Errorf("%s: not implemented yet", name)
}
