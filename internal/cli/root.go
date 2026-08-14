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
			"  check -> prep -> deploy       build the deployment   (up)\n" +
			"  config -> verify              POST-DEPLOYMENT, over the broker CLI\n" +
			"  delete -> teardown            tear it down           (down)\n\n" +
			"'up' covers only the first line. config and verify drive the Solace CLI\n" +
			"inside a broker that is already running, so run them once it is ready.\n\n" +
			"Pick a platform (k8s, docker, podman), then a verb. Every command takes\n" +
			"-e/--env <file>, searched in the current directory then ./env.\n" +
			"Coming from the bash scripts? 'solace convert <bash-env-file>' turns an old\n" +
			"env file into the YAML this reads. See 'solace <platform> --help'.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().StringVarP(&app.EnvName, "env", "e", config.EnvFileDefault, "env file name, searched in the base dir then <base-dir>/env; a value with a directory is used as-is")
	root.PersistentFlags().StringVar(&app.BaseDir, "base-dir", "", "directory searched for the env file, and holding env/ (default: current directory)")
	root.PersistentFlags().BoolVar(&app.GenOnly, "gen-only", false, "render the deployment artifact this command would apply and print it; change nothing")
	root.PersistentFlags().BoolVar(&app.GenSecretsOnly, "gen-secrets-only", false, "render this deployment's secrets (k8s Secret manifests; podman secret-create commands; docker export lines to source) and print them; change nothing")
	root.PersistentFlags().BoolVar(&app.GenEnvOnly, "gen-env-only", false, "render the container broker settings as an env file and print them; change nothing (docker/podman only)")
	root.PersistentFlags().BoolVar(&app.DryRun, "dry-run", false, "print the external commands instead of running them")
	root.PersistentFlags().BoolVarP(&app.Yes, "yes", "y", false, "skip confirmation prompts (does NOT imply --purge)")

	root.AddCommand(
		newK8sCmd(app),
		newContainerCmd(app, config.Docker),
		newContainerCmd(app, config.Podman),
		newConvertCmd(app),
	)
	return root
}

// notImplemented is a placeholder for handlers whose downstream package is not
// wired yet. It fails loud so a half-built command can never appear to succeed.
func notImplemented(name string) error {
	return fmt.Errorf("%s: not implemented yet", name)
}
