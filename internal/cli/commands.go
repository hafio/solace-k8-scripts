package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"solace/internal/config"
)

// This file builds the whole command tree. Two rules shape it.
//
// ONE TREE, NOT ONE PER PLATFORM. The platform is a property of the deployment the
// env file already describes, so it is resolved from that file (or named with
// --platform) rather than typed again as the first word of every command
// (platform.go). The tree is the union of what the platforms can do and is the SAME
// shape on all of them, because help and completion render it without reading an
// env file. A command that does not apply says so in its help text and refuses at
// pre-run instead of disappearing.
//
// VERB THEN NOUN, AND NOTHING IMPLICIT. Every verb that acts on more than one kind
// of thing names the thing: `deploy broker`, `remove operator`, `status broker`. A
// verb with object children never acts on its own -- running it bare prints what it
// can act on. That costs one word and buys the property that no command does
// something you did not name, which matters most exactly where it is cheapest to
// get wrong: `remove` on its own removes nothing.
//
// The op bodies stay split across ops_k8s.go and ops_container.go, which is where
// the two platforms really are different things.

// addCommands hangs the lifecycle tree off root.
func addCommands(root *cobra.Command, app *App) {
	root.AddCommand(
		newCheckCmd(app),
		newSmokeCmd(app),
		newPrepareCmd(app),
		newDeployCmd(app),
		newConfigCmd(app),
		newStartCmd(app),
		newStopCmd(app),
		newRestartCmd(app),
		newStatusCmd(app),
		newLogsCmd(app),
		newCLICmd(app),
		newShellCmd(app),
		newCopyCmd(app),
		newGenerateCmd(app),
		newDiagnosticsCmd(app),
		newRemoveCmd(app),
	)
}

// groupAnnotation marks a verb that owns objects and executes nothing itself. It
// exists so the wiring tests can tell a group apart from a command that runs
// something: a group carries a RunE (see below) but must NOT carry the pre-run or
// --allow-command, because it never reaches an external command.
const groupAnnotation = "solace_group"

// group builds a verb that owns objects. Run bare it prints its own help -- the
// list of things the verb can act on -- which is the whole no-implicit-actions
// rule. Given a word it does not know, it fails loudly.
//
// That second half is why a group carries a RunE at all. Cobra answers a
// NON-runnable command by printing help to stdout and exiting 0 whatever arguments
// it was given, so `solace-util remove bogus` would report success having removed
// nothing, and a script would never notice. It is the same trap the completion
// command documents (completion.go), and it matters more here: these are the verbs
// that destroy things, so a mistyped noun must not look like it worked.
func group(use, short, long string) *cobra.Command {
	c := &cobra.Command{
		Use:         use,
		Short:       short,
		Long:        long,
		Annotations: map[string]string{groupAnnotation: "true"},
	}
	c.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		return fmt.Errorf("%q is not something %s can act on; run `%s` to see what it can",
			args[0], cmd.CommandPath(), cmd.CommandPath())
	}
	return c
}

// --- check / smoke ----------------------------------------------------------

// newCheckCmd owns the read-only questions. Everything here can be run against a
// system you are not willing to disturb -- which is exactly why the failover
// exercise is NOT here but under `smoke`.
func newCheckCmd(app *App) *cobra.Command {
	c := group("check", "Run read-only checks",
		"Every check here is read-only: it reports and changes nothing.\n\n"+
			"  check deploy      before deploying -- config, cluster/engine reachability,\n"+
			"                    storage or DNS, and whether the operator is installed\n"+
			"  check semp-login  after deploying -- the broker answers an authenticated\n"+
			"                    SEMP request\n\n"+
			"The failover exercise is deliberately not here: it moves live traffic, so it\n"+
			"lives under `smoke` with the other invasive checks.")
	c.AddCommand(
		dispatchLeaf(app, "deploy", "Validate config and platform prerequisites before deploying",
			platformOps(opK8sCheck, opCtrCheck)),
		roleOnK8sLeaf(app, "semp-login", "Test an authenticated SEMP request against a running broker",
			opK8sVerifyLogin, opCtrVerifyLogin),
	)
	return c
}

// newSmokeCmd is where checks that DISTURB the broker live. The separation is the
// point: an operator scanning `check` should never find something that moves
// messaging traffic, and someone reaching for `smoke` has been told by the word
// itself that this is not a passive question.
func newSmokeCmd(app *App) *cobra.Command {
	c := group("smoke", "Run invasive checks that exercise the broker",
		"These checks prove the broker works by making it work, so they disturb it.\n"+
			"Read-only questions live under `check`.")
	c.AddCommand(roleOnContainerLeaf(app, "redundancy",
		"Exercise a real failover and fail back (HA only)",
		opK8sVerifyRedundancy, opCtrVerifyRedundancy))
	return c
}

// --- prepare ----------------------------------------------------------------

func newPrepareCmd(app *App) *cobra.Command {
	c := group("prepare", "Prepare the prerequisites a broker deployment needs",
		"Everything a broker needs to exist before it is deployed.\n\n"+
			"`prepare all` runs the steps that are needed every time and need no input --\n"+
			"the namespace and its secrets on Kubernetes, the host on docker and podman --\n"+
			"so it is safe to script. `deploy all` runs the same steps for you.\n\n"+
			"Two things are deliberately outside it. The operator is cluster-scoped and\n"+
			"shared between brokers, so it is installed and removed on its own\n"+
			"(`deploy operator`). And `prepare labels` cannot be scripted at all: the env\n"+
			"file names the label each broker role wants, but only you can say which\n"+
			"machine should carry it, so it prompts -- run it once when provisioning the\n"+
			"cluster, not on every deployment.")
	c.AddCommand(
		onlyOn(leaf(app, "namespace", "Create the broker namespace", opK8sPrepNamespace), config.K8s),
		onlyOn(leaf(app, "secrets", "Create admin/monitor, TLS, and image-pull secrets", opK8sPrepSecrets), config.K8s),
		onlyOn(leaf(app, "labels",
			"Label cluster nodes for primary/backup/monitor placement (interactive, one-off)",
			opK8sPrepLabels), config.K8s),
		onlyOn(leaf(app, "host", "Create/own the data dir, verify DNS, generate the redundancy PSK", opCtrPrepHost),
			config.Docker, config.Podman),
		dispatchLeaf(app, "all", "Run every applicable prepare step, in order",
			platformOps(opK8sPrepAll, opCtrPrepAll)),
	)
	return c
}

// --- deploy -----------------------------------------------------------------

func newDeployCmd(app *App) *cobra.Command {
	c := group("deploy", "Deploy the broker, the operator, or the whole broker stack",
		"`deploy broker` applies just the broker. `deploy all` runs the whole bring-up\n"+
			"for it: check -> prepare -> deploy -> assert the config-sync leader (HA).\n\n"+
			"Neither installs the operator. It is cluster-scoped and may already be serving\n"+
			"other brokers, so `deploy operator` is its own command -- run it once per\n"+
			"cluster. `check deploy` reports when it is missing.")

	brokerCmd := wireExec(app, &cobra.Command{
		Use:       "broker [role]",
		Short:     "Deploy the broker (containers: this host's container, role required in HA)",
		ValidArgs: config.RoleNames(),
		Args:      cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return deployBroker(app, firstArg(args))
		},
	})
	addRestartFlag(brokerCmd, app)
	flagOnlyOn(brokerCmd, "restart", config.Docker, config.Podman)

	allCmd := wireExec(app, &cobra.Command{
		Use:       "all [role]",
		Short:     "Orchestrate the whole bring-up for this broker",
		ValidArgs: config.RoleNames(),
		Args:      cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			arg := firstArg(args)
			if err := rejectRole(app.Platform, arg, config.Docker, config.Podman); err != nil {
				return err
			}
			if app.Platform == config.K8s {
				return opK8sDeployAll(app)
			}
			role, err := config.ParseRole(arg)
			if err != nil {
				return err
			}
			return opCtrDeployAll(app, role)
		},
	})
	addRestartFlag(allCmd, app)
	flagOnlyOn(allCmd, "restart", config.Docker, config.Podman)

	c.AddCommand(
		brokerCmd,
		onlyOn(leaf(app, "operator", "Install the cluster-scoped EventBroker Operator", opK8sOperatorDeploy), config.K8s),
		allCmd,
	)
	return c
}

// deployBroker is shared by `deploy broker` and the container half of `deploy all`.
func deployBroker(app *App, arg string) error {
	if err := rejectRole(app.Platform, arg, config.Docker, config.Podman); err != nil {
		return err
	}
	if app.Platform == config.K8s {
		return opK8sDeploy(app)
	}
	role, err := config.ParseRole(arg)
	if err != nil {
		return err
	}
	return opCtrDeploy(app, role)
}

// --- config -----------------------------------------------------------------

// newConfigCmd owns everything applied to a broker that is already running, over
// its own CLI. There is deliberately no run-everything step: these are not
// uniformly re-runnable -- `apply additional-users` fails outright on a user that
// already exists -- so the order is documented here rather than baked into a
// command that would stop halfway through on a second run.
func newConfigCmd(app *App) *cobra.Command {
	c := group("config", "Configure a DEPLOYED broker (certs, hardening, product keys)",
		"Post-deployment configuration: every step here talks to a broker that is already\n"+
			"deployed and running, over the Solace CLI. None of it is part of `deploy`.\n\n"+
			"There is no run-everything command, because these steps are not uniformly\n"+
			"re-runnable. The order that works on a fresh broker is:\n\n"+
			"  1. config leader                        (HA only; on containers, the primary)\n"+
			"  2. config apply server-cert             (when TLS is configured)\n"+
			"  3. config apply domain-certs            (when any are listed)\n"+
			"  4. config disable default-vpn\n"+
			"  5. config disable default-users\n"+
			"  6. config apply additional-users        (Kubernetes; after the hardening, so\n"+
			"                                           the sequence reads harden-then-provision.\n"+
			"                                           NOT re-runnable: the broker refuses to\n"+
			"                                           create a user that already exists)\n"+
			"  7. config apply product-keys            (when any are listed)\n\n"+
			"Only domain-certs can be undone from here (`config delete domain-certs`).\n"+
			"There is no un-harden, and no way to withdraw a server certificate or a\n"+
			"product key through this tool.")

	apply := group("apply", "Apply configuration to the running broker", "")
	apply.AddCommand(
		dispatchLeaf(app, "server-cert", "Load/update the TLS server certificate",
			platformOps(opK8sConfigServerCert, opCtrConfigServerCert)),
		dispatchLeaf(app, "domain-certs", "Load the configured domain CA certificates",
			platformOps(opK8sConfigDomainCerts, opCtrConfigDomainCerts)),
		dispatchLeaf(app, "product-keys", "Apply the configured product keys",
			platformOps(opK8sConfigProductKeys, opCtrConfigProductKeys)),
		// Containers create these at boot from the mounted password file, so there is
		// nothing to apply afterwards; on Kubernetes the operator ignores extra keys in
		// the credentials Secret, so they are created here over the broker CLI.
		onlyOn(leaf(app, "additional-users", "Create the admin.additionalUsers CLI users (not re-runnable)",
			opK8sConfigAdditionalUsers), config.K8s),
	)

	del := group("delete", "Remove configuration from the running broker",
		"Only domain certificates can be withdrawn this way. A server certificate, the\n"+
			"default-VPN hardening and an applied product key all stay applied.")
	del.AddCommand(dispatchLeaf(app, "domain-certs", "Remove the configured domain CA certificates",
		platformOps(opK8sTeardownDomainCerts, opCtrTeardownDomainCerts)))

	disable := group("disable", "Shut down the broker's built-in defaults (hardening)",
		"Both steps are one-way: this tool has no command to re-enable what they shut down.")
	disable.AddCommand(
		dispatchLeaf(app, "default-vpn", "Shut down the default message-VPN",
			platformOps(opK8sConfigDisableVPN, opCtrConfigDisableVPN)),
		dispatchLeaf(app, "default-users", "Shut down the default client-usernames in all VPNs",
			platformOps(opK8sConfigDisableUsers, opCtrConfigDisableUsers)),
	)

	c.AddCommand(apply, del, disable,
		roleOnContainerLeaf(app, "leader", "Assert the config-sync leader (HA only)",
			opK8sConfigLeader, opCtrConfigLeader),
	)
	return c
}

// --- start / stop / restart -------------------------------------------------

// A deployed broker can be stopped without being removed. Kubernetes expresses that
// by scaling the StatefulSet to zero and containers by stopping the container; both
// leave the deployment, its data and its configuration in place, so one pair of
// verbs covers them.

func newStartCmd(app *App) *cobra.Command {
	c := group("start", "Start a broker that is deployed but not running", "")
	c.AddCommand(dispatchLeaf(app, "broker",
		"Start the broker (Kubernetes: scale the statefulset(s) to 1; containers: start the container)",
		platformOps(opK8sStartBroker, opCtrStartBroker)))
	return c
}

func newStopCmd(app *App) *cobra.Command {
	c := group("stop", "Stop a running broker without removing it",
		"The deployment, its persistent data and its configuration all survive --\n"+
			"`start broker` brings it back. Use `remove broker` to delete it.")
	c.AddCommand(dispatchLeaf(app, "broker",
		"Stop the broker (Kubernetes: scale the statefulset(s) to 0; containers: stop the container)",
		platformOps(opK8sStopBroker, opCtrStopBroker)))
	return c
}

func newRestartCmd(app *App) *cobra.Command {
	c := group("restart", "Bounce a running broker or the operator",
		"Restarting applies nothing new. A changed deploy artifact needs\n"+
			"`deploy broker` (containers: with --restart), which rewrites it first.")
	brokerCmd := wireExec(app, &cobra.Command{
		Use:       "broker [role]",
		Short:     "Restart the broker (Kubernetes: delete pods so the statefulset recreates them)",
		Long:      restartBrokerLong,
		ValidArgs: config.RoleNames(),
		Args:      cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			arg := firstArg(args)
			if err := rejectRole(app.Platform, arg, config.K8s); err != nil {
				return err
			}
			if app.Platform == config.K8s {
				return opK8sRestart(app, arg)
			}
			return opCtrRestartBroker(app)
		},
	})
	// Restarting confirms too: on Kubernetes it deletes pods, which drops messaging
	// traffic. It carries no --delete-* because it destroys nothing that survives
	// the bounce.
	addRemoveFlags(brokerCmd, app, nil)

	c.AddCommand(
		brokerCmd,
		onlyOn(leaf(app, "operator", "Restart the operator's controller deployment", opK8sOperatorRestart), config.K8s),
	)
	return c
}

const restartBrokerLong = "For kubernetes.updateStrategy=manualPodRestart: `deploy broker` updates the\n" +
	"statefulset's pod template but the operator waits for a pod to be deleted before\n" +
	"applying it.\n\n" +
	"With no role, every pod is restarted in the safe order (monitor, backup, primary;\n" +
	"standalone: just the primary), waiting for each to become ready before the next.\n" +
	"The order is by configured role, not by which node is currently active -- after a\n" +
	"failover they differ. Check `solace-util smoke redundancy` first, or pass a role\n" +
	"and restart them one at a time.\n\n" +
	"On docker and podman there is one broker per host and no role to pick: the\n" +
	"container is restarted in place."

// --- status / logs / cli / shell / copy --------------------------------------

func newStatusCmd(app *App) *cobra.Command {
	c := group("status", "Report on the broker or the operator",
		"By default this reports the RUNNING artifacts. --detail adds the static ones --\n"+
			"the full description of what is deployed, load balancer included.")

	brokerCmd := wireExec(app, &cobra.Command{
		Use:       "broker [role]",
		Short:     "Show the broker's deployment status",
		ValidArgs: config.RoleNames(),
		Args:      cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			arg := firstArg(args)
			if err := rejectRole(app.Platform, arg, config.K8s); err != nil {
				return err
			}
			role, err := config.ParseRole(arg)
			if err != nil {
				return err
			}
			if app.Platform == config.K8s {
				return opK8sStatusBroker(app, role)
			}
			return opCtrStatusBroker(app)
		},
	})
	brokerCmd.Flags().BoolVar(&app.all, "all", false,
		"report every Solace broker found, not just the one this env file describes "+
			"(Kubernetes: across all namespaces; docker/podman: every Solace container on this host)")
	brokerCmd.Flags().BoolVar(&app.detail, "detail", false,
		"include the static artifacts, not just the running ones (Kubernetes: secrets, "+
			"configmaps and PVCs; docker/podman: mounts, which is also where secrets appear)")

	operatorCmd := onlyOn(leaf(app, "operator", "Show the operator's controller status", opK8sStatusOperator), config.K8s)
	operatorCmd.Flags().BoolVar(&app.detail, "detail", false,
		"include the full description of the operator deployment")

	c.AddCommand(brokerCmd, operatorCmd)
	return c
}

func newLogsCmd(app *App) *cobra.Command {
	c := group("logs", "Tail broker or operator logs", "")
	c.AddCommand(
		roleOnK8sLeaf(app, "broker", "Tail the broker's logs", opK8sLogs, opCtrLogs),
		onlyOn(leaf(app, "operator", "Tail the operator's controller logs", opK8sOperatorLogs), config.K8s),
	)
	return c
}

// newCLICmd opens a Solace CLI session in the broker, or -- with --input -- runs a
// script through one instead. The script form used to be its own command; it is a
// flag because it answers the same question ("give me the broker's CLI") with the
// session automated rather than interactive.
func newCLICmd(app *App) *cobra.Command {
	c := wireExec(app, &cobra.Command{
		Use:       "cli [role]",
		Short:     "Open an interactive Solace CLI in the broker (Kubernetes: [role] picks the pod)",
		Long: "With no flags this opens an interactive Solace CLI session.\n\n" +
			"--input runs a script through that CLI instead of opening a session: a bare\n" +
			"filename is resolved under broker.cliScriptsFolder, a path is used as typed,\n" +
			"and the file is uploaded to the broker and run there. Errors reported by the\n" +
			"broker are surfaced as warnings, not failures -- a CLI script is a sequence of\n" +
			"independent commands, and one refused line does not invalidate the rest.",
		ValidArgs: config.RoleNames(),
		Args:      cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			arg := firstArg(args)
			if err := rejectRole(app.Platform, arg, config.K8s); err != nil {
				return err
			}
			if app.inputFile != "" {
				if app.Platform == config.K8s {
					return opK8sExecCLI(app, app.inputFile)
				}
				return opCtrExecCLI(app, app.inputFile)
			}
			role, err := config.ParseRole(arg)
			if err != nil {
				return err
			}
			if app.Platform == config.K8s {
				return opK8sCLI(app, role)
			}
			return opCtrCLI(app)
		},
	})
	c.Flags().StringVarP(&app.inputFile, "input", "i", "",
		"run this Solace CLI script instead of opening an interactive session")
	c.Flags().StringVar(&app.pod, "pod", "", podFlagUsage)
	flagOnlyOn(c, "pod", config.K8s)
	registerFlagCompletion(c, "pod", completeRoles)
	return c
}

func newShellCmd(app *App) *cobra.Command {
	return roleOnK8sLeaf(app, "shell", "Open an interactive shell in the broker", opK8sShell, opCtrShell)
}

// newCopyCmd mirrors the same verbs on every platform. On a container host the
// transport is node-local, so the files are already on this machine -- the verbs
// exist so a script does not have to know which platform it is driving.
func newCopyCmd(app *App) *cobra.Command {
	c := group("copy", "Copy files to/from the broker", "")

	from := wireExec(app, &cobra.Command{
		Use:   "from files...",
		Short: "Copy files from the broker to the host",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if app.Platform == config.K8s {
				return opK8sCopyFrom(app, args)
			}
			return opCtrCopyFrom(app, args)
		},
	})
	from.Flags().StringVar(&app.pod, "pod", "", podFlagUsage)
	flagOnlyOn(from, "pod", config.K8s)
	registerFlagCompletion(from, "pod", completeRoles)

	into := wireExec(app, &cobra.Command{
		Use:   "into files...",
		Short: "Copy files from the host into the broker",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if app.Platform == config.K8s {
				return opK8sCopyInto(app, args)
			}
			return opCtrCopyInto(app, args)
		},
	})
	into.Flags().StringVar(&app.pod, "pod", "", podFlagUsage)
	flagOnlyOn(into, "pod", config.K8s)
	into.Flags().StringVar(&app.destDir, "dir", "", "destination directory inside the broker")
	registerFlagCompletion(into, "pod", completeRoles)
	registerFlagCompletion(into, "dir", completeDirs)

	c.AddCommand(from, into)
	return c
}

// podFlagUsage is shared by the commands that can target a specific pod.
const podFlagUsage = "pod role to target (p|b|m)"

// --- generate ---------------------------------------------------------------

// newGenerateCmd renders artifacts and applies nothing. It is the only way to see
// what a command WOULD send, which is why it is a command with a named target
// rather than a flag on the commands that deploy: an artifact you meant to read
// and a cluster you meant to change should not be one typo apart.
func newGenerateCmd(app *App) *cobra.Command {
	c := group("generate", "Render a deployment artifact to stdout without applying it",
		"Nothing here contacts the cluster or the container engine, so it is safe to run\n"+
			"against an env file you have not vetted.\n\n"+
			"The nouns are the same ones the acting verbs use: `generate broker` renders what\n"+
			"`deploy broker` would apply, whichever platform that is -- a custom resource on\n"+
			"Kubernetes, a compose file or systemd quadlet on a container host (which is\n"+
			"per-host, so it takes a [role] there).\n\n"+
			"Only `operator` is platform-scoped, and because the thing does not exist\n"+
			"elsewhere rather than because it goes by another name: there is no container\n"+
			"operator to install.")

	brokerCmd := wireExec(app, renderOnly(&cobra.Command{
		Use:   "broker [role]",
		Short: "Render what `deploy broker` would apply",
		Long: "Kubernetes: the PubSubPlusEventBroker custom resource. Docker and podman: this\n" +
			"host's deploy artifact -- a compose file or a systemd quadlet unit -- which is\n" +
			"per-host, so [role] selects which node's artifact to render.",
		ValidArgs: config.RoleNames(),
		Args:      cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			arg := firstArg(args)
			if err := rejectRole(app.Platform, arg, config.Docker, config.Podman); err != nil {
				return err
			}
			if app.Platform == config.K8s {
				return opK8sGenBroker(app)
			}
			role, err := config.ParseRole(arg)
			if err != nil {
				return err
			}
			return opCtrGenArtifact(app, role)
		},
	}))

	c.AddCommand(
		brokerCmd,
		renderOnly(dispatchLeaf(app, "secrets",
			"Render the secret-creation artifact (Kubernetes: Secret manifests; containers: a shell script)",
			platformOps(opK8sGenSecrets, opCtrGenSecrets))),
		onlyOn(renderOnly(leaf(app, "operator", "Render the operator install bundle", opK8sGenOperator)), config.K8s),
	)
	return c
}

// --- diagnostics ------------------------------------------------------------

// newDiagnosticsCmd gathers a support bundle. It is not a check: it collects a
// large `show` sweep plus the broker's own diagnostics archive and downloads them,
// which is what Solace support asks for rather than something you read yourself.
func newDiagnosticsCmd(app *App) *cobra.Command {
	c := dispatchLeaf(app, "diagnostics",
		"Gather a support bundle from the broker into broker.diagDir",
		platformOps(opK8sVerifyDiagnostics, opCtrVerifyDiagnostics))
	c.Flags().IntVar(&app.days, "days", 1, "days of logs/diagnostics to gather")
	registerFlagCompletion(c, "days", cobra.NoFileCompletions)
	return c
}

// --- remove -----------------------------------------------------------------

// newRemoveCmd owns every deletion. Each object is named: `remove` on its own
// removes nothing, which is the whole reason the verb takes a noun.
//
// Both `remove broker` and `remove operator` keep their expensive layer by default
// -- persistent data and the CRDs respectively -- and ask about it the same way
// (addLayerFlags/confirmLayer in helpers.go), so learning the contract on one
// teaches the other. Either outcome is reported rather than left to be inferred.
func newRemoveCmd(app *App) *cobra.Command {
	c := group("remove", "Remove the broker, the operator, or the whole broker stack",
		"Every command here asks before it removes anything, and --no-prompt is the one\n"+
			"flag that makes it silent -- a script switches off one thing, not one per\n"+
			"question.\n\n"+
			"Nothing here removes the layer that is expensive to get back unless you say so:\n"+
			"the broker's persistent data and the operator's CRDs are kept by default, you\n"+
			"are asked about them separately, and what happened is printed either way. The\n"+
			"two flags compose, so an unattended removal that also drops the data is\n"+
			"`--delete-data --no-prompt`: naming the data you are willing to lose is not the\n"+
			"same as confirming the removal, so neither flag implies the other.\n\n"+
			"`remove all` takes this broker and its namespace. It leaves the operator, which\n"+
			"is cluster-scoped and may be serving brokers this env file does not describe.")

	brokerCmd := dispatchLeaf(app, "broker", "Remove the deployed broker",
		platformOps(opK8sDelete, opCtrDelete))
	addRemoveFlags(brokerCmd, app, &layerData)

	operatorCmd := onlyOn(leaf(app, "operator", "Remove the cluster-scoped EventBroker Operator",
		opK8sOperatorRemove), config.K8s)
	addRemoveFlags(operatorCmd, app, &layerCRD)

	allCmd := dispatchLeaf(app, "all", "Remove the broker, its secrets and its namespace (the operator is kept)",
		platformOps(opK8sRemoveAll, opCtrRemoveAll))
	addRemoveFlags(allCmd, app, &layerData)

	// secrets and namespace have no retained layer -- `prepare` recreates both from
	// the env file -- but they still confirm, because deleting a namespace takes
	// whatever else happens to be in it.
	secretsCmd := onlyOn(leaf(app, "secrets", "Delete the broker's secrets", opK8sRemoveSecrets), config.K8s)
	addRemoveFlags(secretsCmd, app, nil)
	namespaceCmd := onlyOn(leaf(app, "namespace", "Delete the broker's namespace", opK8sRemoveNamespace), config.K8s)
	addRemoveFlags(namespaceCmd, app, nil)

	c.AddCommand(brokerCmd, operatorCmd, secretsCmd, namespaceCmd, allCmd)
	return c
}

// --- shared leaf shapes ------------------------------------------------------

// roleOnK8sLeaf builds a leaf whose [role] picks a pod on Kubernetes and means
// nothing on a container host, where there is one broker per machine and the
// transport ignores the role entirely. The role is refused there rather than
// accepted and dropped (rejectRole).
func roleOnK8sLeaf(app *App, use, short string, k8sFn roleOpFunc, ctrFn opFunc) *cobra.Command {
	return wireExec(app, &cobra.Command{
		Use:       use + " [role]",
		Short:     short,
		ValidArgs: config.RoleNames(),
		Args:      cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			arg := firstArg(args)
			if err := rejectRole(app.Platform, arg, config.K8s); err != nil {
				return err
			}
			role, err := config.ParseRole(arg)
			if err != nil {
				return err
			}
			if app.Platform == config.K8s {
				return k8sFn(app, role)
			}
			return ctrFn(app)
		},
	})
}

// roleOnContainerLeaf is the mirror image: the [role] names which half of a
// cross-host operation THIS machine is, which only a container host needs --
// Kubernetes drives the whole redundancy group from one context, so a role there
// would be answering a question the cluster already knows.
func roleOnContainerLeaf(app *App, use, short string, k8sFn opFunc, ctrFn func(*App, string) error) *cobra.Command {
	return wireExec(app, &cobra.Command{
		Use:       use + " [role]",
		Short:     short + " (containers: [role] is this host, detected from its name when omitted)",
		ValidArgs: config.RoleNames(),
		Args:      cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			arg := firstArg(args)
			if err := rejectRole(app.Platform, arg, config.Docker, config.Podman); err != nil {
				return err
			}
			if app.Platform == config.K8s {
				return k8sFn(app)
			}
			return ctrFn(app, arg)
		},
	})
}
