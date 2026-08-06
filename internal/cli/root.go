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
		Use:   "solace",
		Short: "Deploy and operate Solace PubSub+ brokers on Kubernetes, Docker, or Podman",
		Long: "solace is a single CLI for deploying and operating Solace PubSub+ Event Brokers.\n" +
			"It presents the same lifecycle verbs on every platform:\n\n" +
			"  check -> prep -> deploy -> config -> verify   (up)\n" +
			"  delete -> teardown                            (down)\n\n" +
			"Pick a platform (k8s, docker, podman), then a verb. Every command takes\n" +
			"--env <name> (env/<name>.yaml). See 'solace <platform> --help'.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().StringVar(&app.EnvName, "env", "default", "env file name (env/<name>.yaml) or path")
	root.PersistentFlags().StringVar(&app.BaseDir, "base-dir", "", "directory containing env/ (default: current directory)")
	root.PersistentFlags().BoolVar(&app.GenOnly, "gen", false, "render the artifact this command would apply and print it; change nothing")
	root.PersistentFlags().BoolVar(&app.DryRun, "dry-run", false, "print the external commands instead of running them")
	root.PersistentFlags().BoolVarP(&app.Yes, "yes", "y", false, "skip confirmation prompts (does NOT imply --purge)")

	root.AddCommand(
		newK8sCmd(app),
		newContainerCmd(app, config.Docker),
		newContainerCmd(app, config.Podman),
	)
	return root
}

// notImplemented is a placeholder for handlers whose downstream package is not
// wired yet. It fails loud so a half-built command can never appear to succeed.
func notImplemented(name string) error {
	return fmt.Errorf("%s: not implemented yet", name)
}
