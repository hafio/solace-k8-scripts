package cli

import (
	"github.com/spf13/cobra"

	"solace/internal/config"
)

// newContainerCmd builds the `solace docker` or `solace podman` subtree. Both
// engines share one tree shape; only the deploy artifact (compose/run vs systemd
// quadlet) and rootless handling differ, resolved downstream by app.Platform.
func newContainerCmd(app *App, p config.Platform) *cobra.Command {
	var short string
	switch p {
	case config.Docker:
		short = "Deploy/operate the broker directly on a host with Docker"
	default:
		short = "Deploy/operate the broker directly on a host with Podman (systemd quadlet)"
	}

	c := &cobra.Command{
		Use:   string(p),
		Short: short,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			app.Platform = p
			if err := checkGenFlags(cmd, app); err != nil {
				return err
			}
			if err := checkAllowCommand(cmd, app); err != nil {
				return err
			}
			return app.load()
		},
	}
	addAllowCommandFlag(c, app)

	// --- lifecycle: check -> prep -> deploy -> config -> verify ---
	c.AddCommand(leaf(app, "check", "Validate config, DNS, and container runtime", opCtrCheck))
	c.AddCommand(newCtrPrepCmd(app))
	c.AddCommand(newCtrDeployCmd(app))
	c.AddCommand(newCtrConfigCmd(app))
	c.AddCommand(newCtrVerifyCmd(app))

	// --- day-2 ops (node-local: no role) ---
	c.AddCommand(leaf(app, "status", "Show the local broker container/service status", opCtrStatus))
	c.AddCommand(leaf(app, "logs", "Tail the local broker container logs", opCtrLogs))
	c.AddCommand(leaf(app, "cli", "Open an interactive Solace CLI in the container", opCtrCLI))
	c.AddCommand(leaf(app, "shell", "Open an interactive shell in the container", opCtrShell))
	c.AddCommand(newCtrDescribeCmd(app))
	c.AddCommand(newCtrCopyCmd(app))
	c.AddCommand(newCtrGenCmd(app))

	// --- teardown + orchestration ---
	c.AddCommand(newCtrTeardownCmd(app))
	c.AddCommand(newCtrDeleteCmd(app))
	c.AddCommand(newCtrUpCmd(app))
	c.AddCommand(newCtrDownCmd(app))
	return c
}

func newCtrPrepCmd(app *App) *cobra.Command {
	prep := &cobra.Command{
		Use:   "prep",
		Short: "Prepare the host (data dir + ownership, DNS, PSK generation)",
		Args:  cobra.NoArgs,
		RunE:  func(*cobra.Command, []string) error { return opCtrPrepAll(app) },
	}
	prep.AddCommand(leaf(app, "host", "Create/own the data dir, verify DNS, generate the redundancy PSK", opCtrPrepHost))
	return prep
}

// deploy honors the --gen-*-only flags: opCtrDeploy renders the requested
// artifact instead of applying it. The annotation is both what the generated
// command reference reads and what checkGenFlags gates on, so a gen flag on any
// other container command fails loud rather than being ignored.
func newCtrDeployCmd(app *App) *cobra.Command {
	c := genCapable(&cobra.Command{
		Use:   "deploy [primary|backup|monitor]",
		Short: "Deploy the broker on this host (role required in HA, ignored in standalone)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			role, err := config.ParseRole(firstArg(args))
			if err != nil {
				return err
			}
			return opCtrDeploy(app, role)
		},
	})
	addRestartFlag(c, app)
	return c
}

func newCtrConfigCmd(app *App) *cobra.Command {
	cfg := &cobra.Command{
		Use:   "config",
		Short: "Configure a DEPLOYED broker (certs, hardening, product keys, CLI)",
		Long: "Post-deployment configuration: every step here talks to a broker that is already " +
			"running in this host's container, over the Solace CLI. Nothing under `config` is part " +
			"of `deploy`, and none of it is applied by `up` -- run it once the container is up.\n\n" +
			"With no subcommand, runs all applicable config steps (HA-only steps skipped in " +
			"standalone), except `leader`: that one is cross-host and primary-only, so run it " +
			"explicitly on the primary.",
		Args:  cobra.NoArgs,
		RunE:  func(*cobra.Command, []string) error { return opCtrConfigAll(app) },
	}
	execCli := &cobra.Command{
		Use:   "exec-cli [file]",
		Short: "Run a Solace CLI script inside the container (menu if no file given)",
		Args:  cobra.MaximumNArgs(1),
		RunE:  func(_ *cobra.Command, args []string) error { return opCtrExecCLI(app, firstArg(args)) },
	}
	// leader takes an optional role (empty -> detect from hostname). It is
	// primary-only and part of the cross-host handshake, so it fails loud on a
	// backup/monitor host.
	leader := &cobra.Command{
		Use:   "leader [primary|backup|monitor]",
		Short: "Assert the config-sync leader on the primary (HA only)",
		Args:  cobra.MaximumNArgs(1),
		RunE:  func(_ *cobra.Command, args []string) error { return opCtrConfigLeader(app, firstArg(args)) },
	}
	cfg.AddCommand(
		leader,
		leaf(app, "server-cert", "Load/update the TLS server certificate", opCtrConfigServerCert),
		leaf(app, "domain-certs", "Load domain CA certificates", opCtrConfigDomainCerts),
		leaf(app, "disable-default-vpn", "Shut down the default message-VPN", opCtrConfigDisableVPN),
		leaf(app, "disable-default-users", "Shut down default client-usernames in all VPNs", opCtrConfigDisableUsers),
		leaf(app, "product-keys", "Apply product keys", opCtrConfigProductKeys),
		execCli,
	)
	return cfg
}

func newCtrVerifyCmd(app *App) *cobra.Command {
	v := &cobra.Command{
		Use:   "verify",
		Short: "Verify broker health (login, redundancy, diagnostics)",
		Args:  cobra.NoArgs,
		RunE:  func(*cobra.Command, []string) error { return opCtrVerifyAll(app) },
	}
	diag := &cobra.Command{
		Use:   "diagnostics",
		Short: "Gather show-command output and a diagnostics bundle",
		Args:  cobra.NoArgs,
		RunE:  func(*cobra.Command, []string) error { return opCtrVerifyDiagnostics(app) },
	}
	diag.Flags().IntVar(&app.days, "days", 1, "days of logs/diagnostics to gather")
	// redundancy takes an optional role (empty -> detect). Run it on the primary
	// and backup concurrently; the monitor is rejected loud.
	redundancy := &cobra.Command{
		Use:   "redundancy [primary|backup|monitor]",
		Short: "Exercise failover on this node (HA only; run on primary and backup)",
		Args:  cobra.MaximumNArgs(1),
		RunE:  func(_ *cobra.Command, args []string) error { return opCtrVerifyRedundancy(app, firstArg(args)) },
	}
	v.AddCommand(
		leaf(app, "login", "Test SEMP login", opCtrVerifyLogin),
		redundancy,
		diag,
	)
	return v
}

func newCtrGenCmd(app *App) *cobra.Command {
	return renderOnly(genCapable(&cobra.Command{
		Use:   "gen [primary|backup|monitor]",
		Short: "Render the deploy artifact (quadlet/compose/run) to stdout without applying",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			role, err := config.ParseRole(firstArg(args))
			if err != nil {
				return err
			}
			return opCtrGen(app, role)
		},
	}))
}

// newCtrDescribeCmd is the container analog of `k8s describe`. "inspect" is an
// alias because that is the container-world name for the same question, and the
// underlying call is literally `<runtime> inspect`.
func newCtrDescribeCmd(app *App) *cobra.Command {
	c := leaf(app, "describe", "Show detailed inspection of the broker container (podman: also the installed unit)", opCtrDescribe)
	c.Aliases = []string{"inspect"}
	return c
}

// newCtrCopyCmd mirrors `k8s copy`. The transport is node-local, so the files are
// already on this host -- the verbs exist so a script does not have to know which
// platform it is driving.
func newCtrCopyCmd(app *App) *cobra.Command {
	c := &cobra.Command{Use: "copy", Short: "Copy files to/from the broker container"}
	from := &cobra.Command{
		Use:   "from files...",
		Short: "Copy files from the broker container to the host",
		Args:  cobra.MinimumNArgs(1),
		RunE:  func(_ *cobra.Command, args []string) error { return opCtrCopyFrom(app, args) },
	}
	into := &cobra.Command{
		Use:   "into files...",
		Short: "Copy files from the host into the broker container",
		Args:  cobra.MinimumNArgs(1),
		RunE:  func(_ *cobra.Command, args []string) error { return opCtrCopyInto(app, args) },
	}
	into.Flags().StringVar(&app.destDir, "dir", "", "destination directory inside the container")
	c.AddCommand(from, into)
	return c
}

// newCtrTeardownCmd removes broker-scoped prerequisites the container itself
// outlives, mirroring the k8s verb: `config X` applies, `teardown X` removes.
func newCtrTeardownCmd(app *App) *cobra.Command {
	t := &cobra.Command{Use: "teardown", Short: "Remove broker-scoped prerequisites (the container itself: see delete)"}
	t.AddCommand(leaf(app, "domain-certs", "Remove domain CA certificates", opCtrTeardownDomainCerts))
	return t
}

func newCtrDeleteCmd(app *App) *cobra.Command {
	c := &cobra.Command{
		Use:   "delete",
		Short: "Remove the broker container/unit (data folder kept by default)",
		Args:  cobra.NoArgs,
		RunE:  func(*cobra.Command, []string) error { return opCtrDelete(app) },
	}
	addDataFlags(c, app)
	return c
}

func newCtrUpCmd(app *App) *cobra.Command {
	c := &cobra.Command{
		Use:   "up [primary|backup|monitor]",
		Short: "Orchestrate check -> prep host -> deploy <role>",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			role, err := config.ParseRole(firstArg(args))
			if err != nil {
				return err
			}
			return opCtrUp(app, role)
		},
	}
	addRestartFlag(c, app)
	return c
}

func newCtrDownCmd(app *App) *cobra.Command {
	c := &cobra.Command{
		Use:   "down",
		Short: "Orchestrate delete (data folder kept unless --purge)",
		Args:  cobra.NoArgs,
		RunE:  func(*cobra.Command, []string) error { return opCtrDown(app) },
	}
	addDataFlags(c, app)
	return c
}
