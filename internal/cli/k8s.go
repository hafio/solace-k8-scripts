package cli

import (
	"github.com/spf13/cobra"

	"solace/internal/config"
)

// newK8sCmd builds the `solace k8s` subtree: the operator-based Kubernetes
// deployment. Config is loaded once in PersistentPreRunE for every k8s verb.
func newK8sCmd(app *App) *cobra.Command {
	k := &cobra.Command{
		Use:   "k8s",
		Short: "Deploy/operate the broker on Kubernetes via the EventBroker Operator",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			app.Platform = config.K8s
			if err := checkGenFlags(cmd, app); err != nil {
				return err
			}
			if err := checkAllowCommand(cmd, app); err != nil {
				return err
			}
			return app.load()
		},
	}
	addAllowCommandFlag(k, app)

	// --- lifecycle: check -> prep -> deploy -> config -> verify ---
	k.AddCommand(newK8sCheckCmd(app))
	k.AddCommand(newK8sPrepCmd(app))
	k.AddCommand(newK8sDeployCmd(app))
	k.AddCommand(newK8sConfigCmd(app))
	k.AddCommand(newK8sVerifyCmd(app))

	// --- day-2 ops ---
	k.AddCommand(newK8sStatusCmd(app))
	k.AddCommand(roleLeaf(app, "logs", "Tail broker pod logs", opK8sLogs))
	k.AddCommand(roleLeaf(app, "cli", "Open an interactive Solace CLI in a pod", opK8sCLI))
	k.AddCommand(roleLeaf(app, "shell", "Open an interactive shell in a pod", opK8sShell))
	k.AddCommand(newK8sDescribeCmd(app))
	k.AddCommand(newK8sCopyCmd(app))
	k.AddCommand(newK8sReplicasCmd(app))
	k.AddCommand(newK8sRestartCmd(app))
	k.AddCommand(newK8sOperatorCmd(app))
	k.AddCommand(newK8sGenCmd(app))
	k.AddCommand(newK8sShowAllCmd(app))

	// --- teardown: delete -> teardown ---
	k.AddCommand(newK8sDeleteCmd(app))
	k.AddCommand(newK8sTeardownCmd(app))

	// --- orchestration ---
	k.AddCommand(newK8sUpCmd(app))
	k.AddCommand(newK8sDownCmd(app))
	return k
}

func newK8sCheckCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "Validate config, cluster reachability, and StorageClass",
		Args:  cobra.NoArgs,
		RunE:  func(*cobra.Command, []string) error { return opK8sCheck(app) },
	}
}

func newK8sPrepCmd(app *App) *cobra.Command {
	prep := &cobra.Command{
		Use:   "prep",
		Short: "Prepare cluster prerequisites (operator, namespace, secrets, labels)",
		Long:  "With no subcommand, runs all prep steps in order, skipping ones whose config is absent.",
		Args:  cobra.NoArgs,
		RunE:  func(*cobra.Command, []string) error { return opK8sPrepAll(app) },
	}
	prep.AddCommand(
		genCapable(leaf(app, "operator", "Install the EventBroker Operator", opK8sOperatorDeploy)),
		leaf(app, "namespace", "Create the broker namespace", opK8sPrepNamespace),
		genCapable(leaf(app, "secrets", "Create admin/monitor, TLS, and image-pull secrets", opK8sPrepSecrets)),
		leaf(app, "labels", "Label nodes for primary/backup/monitor placement", opK8sPrepLabels),
	)
	return prep
}

func newK8sDeployCmd(app *App) *cobra.Command {
	c := &cobra.Command{
		Use:   "deploy",
		Short: "Render and apply the PubSubPlusEventBroker custom resource",
		Args:  cobra.NoArgs,
		RunE:  func(*cobra.Command, []string) error { return opK8sDeploy(app) },
	}
	c.Flags().BoolVar(&app.keepYAML, "keep-yaml", false, "keep the rendered manifest on disk after applying")
	return genCapable(c)
}

func newK8sConfigCmd(app *App) *cobra.Command {
	cfg := &cobra.Command{
		Use:   "config",
		Short: "Configure a DEPLOYED broker (certs, hardening, product keys, CLI)",
		Long: "Post-deployment configuration: every step here talks to a broker that is already " +
			"deployed and running, over the Solace CLI. Nothing under `config` is part of `deploy`, " +
			"and none of it is applied by `up` -- run it after the pods are ready.\n\n" +
			"With no subcommand, runs all applicable config steps in order (HA-only steps skipped " +
			"in standalone).",
		Args:  cobra.NoArgs,
		RunE:  func(*cobra.Command, []string) error { return opK8sConfigAll(app) },
	}
	execCli := &cobra.Command{
		Use:   "exec-cli [file]",
		Short: "Run a Solace CLI script inside a pod",
		Args:  cobra.MaximumNArgs(1),
		RunE:  func(_ *cobra.Command, args []string) error { return opK8sExecCLI(app, firstArg(args)) },
	}
	execCli.Flags().StringVar(&app.pod, "pod", "", "pod role to target (p|b|m)")
	cfg.AddCommand(
		leaf(app, "leader", "Assert the config-sync leader (HA only)", opK8sConfigLeader),
		leaf(app, "server-cert", "Load/update the TLS server certificate", opK8sConfigServerCert),
		leaf(app, "domain-certs", "Load domain CA certificates", opK8sConfigDomainCerts),
		leaf(app, "disable-default-vpn", "Shut down the default message-VPN", opK8sConfigDisableVPN),
		leaf(app, "disable-default-users", "Shut down default client-usernames in all VPNs", opK8sConfigDisableUsers),
		leaf(app, "additional-users", "Create the admin.additionalUsers CLI users", opK8sConfigAdditionalUsers),
		leaf(app, "product-keys", "Apply product keys", opK8sConfigProductKeys),
		execCli,
	)
	return cfg
}

func newK8sVerifyCmd(app *App) *cobra.Command {
	v := &cobra.Command{
		Use:   "verify",
		Short: "Verify broker health: redundancy failover (HA) then a SEMP login",
		Args:  cobra.NoArgs,
		RunE:  func(*cobra.Command, []string) error { return opK8sVerifyAll(app) },
	}
	diag := &cobra.Command{
		Use:   "diagnostics",
		Short: "Gather show-command output and a diagnostics bundle",
		Args:  cobra.NoArgs,
		RunE:  func(*cobra.Command, []string) error { return opK8sVerifyDiagnostics(app) },
	}
	diag.Flags().IntVar(&app.days, "days", 1, "days of logs/diagnostics to gather")
	v.AddCommand(
		roleLeaf(app, "login", "Test SEMP login", opK8sVerifyLogin),
		leaf(app, "redundancy", "Exercise failover (HA only)", opK8sVerifyRedundancy),
		diag,
	)
	return v
}

func newK8sStatusCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show pods, services, and statefulset for the broker",
		Args:  cobra.NoArgs,
		RunE:  func(*cobra.Command, []string) error { return opK8sStatus(app) },
	}
}

// newK8sDescribeCmd exposes the describe verbs. "inspect" is an alias so the same
// word works on every platform -- on docker/podman the operation really is
// `<runtime> inspect`, and an operator should not have to remember which tree
// calls it what.
func newK8sDescribeCmd(app *App) *cobra.Command {
	d := &cobra.Command{
		Use:     "describe",
		Aliases: []string{"inspect"},
		Short:   "Describe broker/load-balancer resources",
	}
	d.AddCommand(
		roleLeaf(app, "broker", "Describe a broker pod", opK8sDescribeBroker),
		leaf(app, "lb", "Describe the load-balancer service", opK8sDescribeLB),
	)
	return d
}

func newK8sCopyCmd(app *App) *cobra.Command {
	c := &cobra.Command{Use: "copy", Short: "Copy files to/from a broker pod"}
	from := &cobra.Command{
		Use:   "from files...",
		Short: "Copy files from a broker pod to the host",
		Args:  cobra.MinimumNArgs(1),
		RunE:  func(_ *cobra.Command, args []string) error { return opK8sCopyFrom(app, args) },
	}
	from.Flags().StringVar(&app.pod, "pod", "", "pod role to target (p|b|m)")
	into := &cobra.Command{
		Use:   "into files...",
		Short: "Copy files from the host into a broker pod",
		Args:  cobra.MinimumNArgs(1),
		RunE:  func(_ *cobra.Command, args []string) error { return opK8sCopyInto(app, args) },
	}
	into.Flags().StringVar(&app.pod, "pod", "", "pod role to target (p|b|m)")
	into.Flags().StringVar(&app.destDir, "dir", "", "destination directory inside the pod")
	c.AddCommand(from, into)
	return c
}

func newK8sReplicasCmd(app *App) *cobra.Command {
	r := &cobra.Command{Use: "replicas", Short: "Scale the broker statefulset(s)"}
	r.AddCommand(
		leaf(app, "start", "Scale broker statefulset(s) to 1", opK8sReplicasStart),
		leaf(app, "stop", "Scale broker statefulset(s) to 0", opK8sReplicasStop),
	)
	return r
}

// newK8sRestartCmd finishes a manualPodRestart upgrade: the operator updates the
// StatefulSet's pod template on deploy and then waits for the pods to be deleted,
// which nothing in this CLI could do -- `replicas start/stop` only scales a whole
// StatefulSet to 1 or 0.
func newK8sRestartCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "restart [primary|backup|monitor]",
		Short: "Delete a broker pod so the statefulset recreates it (manualPodRestart upgrades)",
		Long: "For k8s.updateStrategy=manualPodRestart: `deploy` updates the statefulset's pod\n" +
			"template but the operator waits for a pod to be deleted before applying it.\n" +
			"With no role, restarts every pod in the safe order (monitor, backup, primary;\n" +
			"standalone: just the primary), waiting for each to become ready before the next.\n\n" +
			"The order is by configured role, not by which node is currently active -- after a\n" +
			"failover they differ. Check `solace k8s verify redundancy` first, or pass a role\n" +
			"and restart them one at a time in the order you want.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error { return opK8sRestart(app, firstArg(args)) },
	}
}

func newK8sOperatorCmd(app *App) *cobra.Command {
	o := &cobra.Command{Use: "operator", Short: "Manage the cluster-scoped EventBroker Operator"}
	o.AddCommand(
		genCapable(leaf(app, "deploy", "Install the operator (embedded bundle)", opK8sOperatorDeploy)),
		leaf(app, "delete", "Remove the operator (embedded bundle)", opK8sOperatorDelete),
		leaf(app, "status", "Show operator deployment/pod status", opK8sOperatorStatus),
		leaf(app, "logs", "Tail operator logs", opK8sOperatorLogs),
		leaf(app, "describe", "Describe the operator deployment", opK8sOperatorDescribe),
	)
	return o
}

func newK8sGenCmd(app *App) *cobra.Command {
	return renderOnly(genCapable(&cobra.Command{
		Use:       "gen [broker|operator|secrets]",
		Short:     "Render a manifest to stdout without applying (like --gen-only)",
		ValidArgs: []string{"broker", "operator", "secrets"},
		Args:      cobra.MaximumNArgs(1),
		RunE:      func(_ *cobra.Command, args []string) error { return opK8sGen(app, firstArgOr(args, "broker")) },
	}))
}

func newK8sShowAllCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "show-all",
		Short: "List all brokers across namespaces",
		Args:  cobra.NoArgs,
		RunE:  func(*cobra.Command, []string) error { return opK8sShowAll(app) },
	}
}

func newK8sDeleteCmd(app *App) *cobra.Command {
	c := &cobra.Command{
		Use:   "delete",
		Short: "Delete the broker CR (PVCs kept by default)",
		Args:  cobra.NoArgs,
		RunE:  func(*cobra.Command, []string) error { return opK8sDelete(app) },
	}
	addDataFlags(c, app)
	return c
}

func newK8sTeardownCmd(app *App) *cobra.Command {
	t := &cobra.Command{Use: "teardown", Short: "Remove broker-scoped prerequisites (operator kept)"}
	t.AddCommand(
		leaf(app, "secrets", "Delete broker secrets", opK8sTeardownSecrets),
		leaf(app, "namespace", "Delete the broker namespace", opK8sTeardownNamespace),
		leaf(app, "domain-certs", "Remove domain CA certificates", opK8sTeardownDomainCerts),
	)
	return t
}

func newK8sUpCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "up",
		Short: "Orchestrate check -> prep -> deploy -> config leader (if HA)",
		Args:  cobra.NoArgs,
		RunE:  func(*cobra.Command, []string) error { return opK8sUp(app) },
	}
}

func newK8sDownCmd(app *App) *cobra.Command {
	c := &cobra.Command{
		Use:   "down",
		Short: "Orchestrate delete -> teardown secrets -> teardown namespace",
		Args:  cobra.NoArgs,
		RunE:  func(*cobra.Command, []string) error { return opK8sDown(app) },
	}
	addDataFlags(c, app)
	return c
}
