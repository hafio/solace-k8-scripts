package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"solace/internal/broker"
	"solace/internal/config"
	"solace/internal/k8s"
	"solace/internal/render"
)

// The k8s handlers wire the cobra tree to the two Kubernetes entry types: k8s.Cluster
// (operations that talk to the cluster/operator, over engine.Runner) and broker.Ops
// (config/verify operations against a running broker, over the kubectl Transport).
// Under --dry-run both run over engine.Echo (injected in load()), so every handler is
// exercisable without a cluster. Progress goes to stderr via step; rendered artifacts
// are the only stdout (emit).

// k8sCluster builds a Cluster over the app's runner/config, wiring stdout as the report
// sink and stdin as the prompt source (LabelNodes prompts per role).
func k8sCluster(a *App) *k8s.Cluster {
	c := k8s.NewCluster(a.Runner, a.Cfg, step, os.Stdout)
	c.In = os.Stdin
	return c
}

// k8sOps builds a broker.Ops over the kubectl-exec transport for config/verify steps.
func k8sOps(a *App) *broker.Ops {
	return broker.New(k8s.NewTransport(a.Runner, a.Cfg), a.Cfg, step)
}

// bg is the context for CLI-invoked operations. A plain background context matches the
// bash scripts' run-to-completion behavior; cancellation is a later concern.
func bg() context.Context { return context.Background() }

// today / nowStamp stamp certificate filenames and diagnostics archives, porting the
// `date +%F` / `date +%Y%m%d-%H%M%S` of 051/069.
func today() string    { return time.Now().Format("2006-01-02") }
func nowStamp() string { return time.Now().Format("20060102-150405") }

// lifecycle

// opK8sCheck validates everything a deployment needs before it is attempted:
// the config itself, cluster reachability, the StorageClass, and -- since the
// operator is now installed by its own command rather than folded into the
// bring-up -- whether the operator and its CRD are actually present. The operator
// probe only warns: `check deploy` is read-only and a missing operator is a thing
// to be told about, not an error in the checking.
func opK8sCheck(a *App) error {
	c := k8sCluster(a)
	ctx := bg()
	if err := c.Check(ctx); err != nil {
		return err
	}
	if !c.OperatorInstalled(ctx) {
		warn("the EventBroker operator does not look installed in this cluster; " +
			"`solace-util deploy operator` installs it, and `deploy broker` will fail without it")
	}
	return nil
}

// opK8sPrepAll runs the prep steps a broker deployment needs every time: the
// namespace and its secrets. Both are idempotent and need no input, so this stays
// scriptable.
//
// Two things are deliberately NOT here. The operator is cluster-scoped and shared
// between brokers, so installing it is its own command (`deploy operator`). And
// node labelling cannot be automated at all: the env file says WHICH LABEL a
// broker's pods want, but only a human can say which machine should carry it, so
// `prepare labels` is a one-off act of cluster provisioning rather than a step in
// bringing up a broker.
func opK8sPrepAll(a *App) error {
	c := k8sCluster(a)
	ctx := bg()
	if err := c.CreateNamespace(ctx); err != nil {
		return err
	}
	return c.CreateSecrets(ctx)
}

// opK8sPrepLabels stamps the configured placement labels onto nodes the operator
// picks, one per broker role. It refuses without a terminal rather than failing
// deep inside the picker on an unreadable stdin: there is no way to express the
// node choice in the env file or on the command line, so a non-interactive run
// cannot do this at all, and saying so up front is the difference between an
// actionable message and an EOF.
func opK8sPrepLabels(a *App) error {
	// Nothing configured means nothing to ask about, so this stays a no-op even
	// without a terminal -- refusing there would fail a run that had no work to do.
	if !placementConfigured(a.Cfg) {
		step("no placement labels configured (kubernetes.placement.labels*); nothing to label")
		return nil
	}
	if !interactive(a) {
		return fmt.Errorf("labelling nodes needs a terminal: the env file names the label each " +
			"broker role wants, but which machine carries it is chosen interactively, and " +
			"there is no flag for it. Run this once by hand on a cluster you can see -- " +
			"`prepare all` and `deploy all` do not need it")
	}
	return k8sCluster(a).LabelNodes(bg())
}

// placementConfigured reports whether any per-role node placement labels are set.
// It gates prepare labels: with none configured there is no question to ask.
func placementConfigured(cfg *config.Config) bool {
	p := cfg.K8s.Placement
	return len(p.LabelsPrimary) > 0 || len(p.LabelsBackup) > 0 || len(p.LabelsMonitor) > 0
}

// opK8sDeploy renders and applies the broker CR.
func opK8sDeploy(a *App) error {
	return k8sCluster(a).DeployBroker(bg(), false)
}

// prep steps
func opK8sPrepNamespace(a *App) error { return k8sCluster(a).CreateNamespace(bg()) }

// opK8sPrepSecrets applies the broker secrets; a gen flag prints the manifests
// instead. Unlike the other prep steps it is gen-capable, since the secrets are
// its own artifact -- `prep secrets --gen-only` and `--gen-secrets-only` are the
// same rendering here.
func opK8sPrepSecrets(a *App) error {
	return k8sCluster(a).CreateSecrets(bg())
}

// config steps
//
// There is deliberately no run-everything step here. Each of these talks to a live
// broker over its CLI and they are not uniformly re-runnable -- `apply
// additional-users` fails outright on a user that already exists -- so the order
// they should be run in is documented on the `config` command rather than baked
// into a command that would stop halfway on a second run.

func opK8sConfigLeader(a *App) error { return k8sOps(a).Leader(bg()) }

// opK8sConfigServerCert loads/updates the TLS server certificate. On k8s the
// secret-managed path (tls.serverSecret set) updates the Secret so the operator mounts
// it (051 fast path); otherwise the CLI path uploads key+cert+CAs into each broker node.
func opK8sConfigServerCert(a *App) error {
	if a.Cfg.TLS.ServerSecret != "" {
		return k8sCluster(a).UpdateServerCertSecret(bg())
	}
	return k8sOps(a).ServerCert(bg(), today(), k8s.HARoles(a.Cfg)...)
}

func opK8sConfigDomainCerts(a *App) error {
	return k8sOps(a).DomainCerts(bg(), config.Primary, a.Cfg.Broker.DomainCerts.Folder, a.Cfg.Broker.DomainCerts.Files)
}
func opK8sConfigDisableVPN(a *App) error   { return k8sOps(a).DisableDefaultVPN(bg(), config.Primary) }
func opK8sConfigDisableUsers(a *App) error { return k8sOps(a).DisableDefaultUsers(bg(), config.Primary) }
func opK8sConfigProductKeys(a *App) error {
	return k8sOps(a).ProductKeys(bg(), a.Cfg.Broker.ProductKeys, k8s.ProductKeyRoles(a.Cfg)...)
}

// opK8sConfigAdditionalUsers creates the extra CLI users. Primary only: management
// users are router-level config that config-sync replicates to the mates.
// ASSUMED, NOT VERIFIED -- if a failover ever surfaces a mate without these users,
// this is the line to widen to k8s.HARoles(a.Cfg).
func opK8sConfigAdditionalUsers(a *App) error {
	return k8sOps(a).AdditionalUsers(bg(), config.Primary, a.Cfg.Admin.AdditionalUsers)
}

// opK8sExecCLI uploads and runs a local Solace CLI script in the target pod. A bare
// filename (no path separator) is resolved under the configured cliScripts folder; a
// path is used as-is. The interactive file-picker menu of the bash 059 is not ported.
func opK8sExecCLI(a *App, file string) error {
	if file == "" {
		return fmt.Errorf("a CLI script file is required (e.g. `solace-util config exec-cli setup.cli`)")
	}
	role, err := podRole(a)
	if err != nil {
		return err
	}
	localPath := file
	if !strings.ContainsAny(file, `/\`) {
		localPath = filepath.Join(a.Cfg.Broker.CLIScriptsFolder, file)
	}
	return k8sOps(a).ExecCLI(bg(), role, localPath)
}

// check / smoke steps

func opK8sVerifyRedundancy(a *App) error { return k8sOps(a).Redundancy(bg()) }

// opK8sVerifyDiagnostics gathers show-command output and a diagnostics bundle from every
// broker node into the configured diagnostics dir.
func opK8sVerifyDiagnostics(a *App) error {
	return k8sOps(a).Diagnostics(bg(), a.Cfg.Broker.DiagDir, nowStamp(), a.days, k8s.HARoles(a.Cfg)...)
}

func opK8sVerifyLogin(a *App, role config.Role) error {
	return k8sLogin(a, k8sOps(a), role)
}

// k8sLogin tests a SEMP login as the fixed k8s admin user. Login writes the outcome to
// stdout and reports ok=false (not an error) on a failed login, so the handler turns a
// failed login into a non-zero exit.
func k8sLogin(a *App, o *broker.Ops, role config.Role) error {
	ok, err := o.Login(bg(), role, "admin", a.Cfg.Admin.Pass)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("SEMP login failed on the %s node (see reason above)", role)
	}
	return nil
}

// day-2 ops

// opK8sStatusBroker reports on the broker. The two flags widen it along different
// axes and compose: --all trades this env file's one broker for every broker in the
// cluster, and --detail trades the running-artifact summary for the full
// description, load balancer included.
func opK8sStatusBroker(a *App, role config.Role) error {
	c := k8sCluster(a)
	ctx := bg()
	if a.all {
		return c.ShowAll(ctx, a.detail)
	}
	if err := c.Survey(ctx, a.detail); err != nil {
		return err
	}
	if !a.detail {
		return nil
	}
	// Scoped to one broker, --detail can afford to go further than the survey's
	// listing: the full description of the pod and the load balancer is what you
	// actually read when one broker is misbehaving.
	if err := c.DescribeBroker(ctx, role); err != nil {
		return err
	}
	return c.DescribeLB(ctx)
}

// opK8sStatusOperator reports the operator's controller state; --detail adds the
// full description of its Deployment.
func opK8sStatusOperator(a *App) error {
	c := k8sCluster(a)
	ctx := bg()
	if err := c.OperatorStatus(ctx); err != nil {
		return err
	}
	if !a.detail {
		return nil
	}
	return c.OperatorDescribe(ctx)
}

func opK8sLogs(a *App, role config.Role) error           { return k8sCluster(a).Logs(bg(), role, nil) }
func opK8sCLI(a *App, role config.Role) error            { return k8sCluster(a).CLI(bg(), role) }
func opK8sShell(a *App, role config.Role) error          { return k8sCluster(a).Shell(bg(), role) }

func opK8sCopyFrom(a *App, files []string) error {
	role, err := podRole(a)
	if err != nil {
		return err
	}
	return k8sCluster(a).CopyFrom(bg(), role, files)
}

func opK8sCopyInto(a *App, files []string) error {
	role, err := podRole(a)
	if err != nil {
		return err
	}
	return k8sCluster(a).CopyInto(bg(), role, files, a.destDir)
}

// opK8sStartBroker / opK8sStopBroker scale the broker statefulset(s) between 1 and
// 0. This is Kubernetes' version of stopping a deployed broker without deleting it:
// the StatefulSet, its PVCs and the CR all survive, which is exactly what
// `<runtime> stop` leaves behind on a container host.
func opK8sStartBroker(a *App) error { return k8sCluster(a).ReplicasStart(bg()) }
func opK8sStopBroker(a *App) error  { return k8sCluster(a).ReplicasStop(bg()) }

// opK8sRestart bounces broker pods for a manualPodRestart upgrade. Deleting a pod
// interrupts messaging on that node, so it takes the same confirmation a delete
// does: --no-prompt proceeds, an interactive session is asked, and a non-interactive
// one without it refuses rather than bouncing a production broker unattended.
func opK8sRestart(a *App, roleArg string) error {
	c := k8sCluster(a)
	if roleArg == "" {
		if !confirmDelete(a, "every broker pod, one at a time (monitor, backup, primary)") {
			return nil
		}
		return c.RestartRolling(bg())
	}
	role, err := config.ParseRole(roleArg)
	if err != nil {
		return err
	}
	if !confirmDelete(a, "the "+roleWord(role)+" broker pod") {
		return nil
	}
	return c.RestartPod(bg(), role)
}

// roleWord spells a role out for a prompt; config.Role's own form is the single
// letter used in resource names, too terse for a user-facing question.
func roleWord(role config.Role) string {
	switch role {
	case config.Backup:
		return "backup"
	case config.Monitor:
		return "monitor"
	default:
		return "primary"
	}
}

// operator lifecycle

// opK8sOperatorDeploy installs the operator; with a gen flag it prints the
// rendered bundle only. It backs both `operator deploy` and `prep operator`.
func opK8sOperatorDeploy(a *App) error {
	return k8sCluster(a).OperatorApply(bg())
}

// opK8sOperatorRemove removes the operator. Like removing a broker it keeps the
// expensive-to-regret layer by default -- here the CRDs, whose deletion cascades to
// every PubSubPlusEventBroker in the cluster, including ones this env file has never
// heard of. confirmLayer decides, and OperatorDelete reports which way it went.
func opK8sOperatorRemove(a *App) error {
	if !confirmDelete(a, "the EventBroker operator") {
		return nil
	}
	deleteCRDs := confirmLayer(a, layerCRD)
	return k8sCluster(a).OperatorDelete(bg(), deleteCRDs)
}

func opK8sOperatorRestart(a *App) error  { return k8sCluster(a).OperatorRestart(bg()) }
func opK8sOperatorLogs(a *App) error     { return k8sCluster(a).OperatorLogs(bg()) }
func opK8sOperatorDescribe(a *App) error { return k8sCluster(a).OperatorDescribe(bg()) }

// opK8sGenBroker / opK8sGenOperator / opK8sGenSecrets print one manifest and change
// nothing. Rendering is a command with a named target rather than a flag on a
// command that would otherwise deploy: an artifact you meant to inspect and a
// cluster you meant to change should not be one typo apart.
func opK8sGenBroker(a *App) error { return emit(render.BrokerCR(a.Cfg)) }

func opK8sGenOperator(a *App) error {
	b, err := k8s.GenOperator(a.Cfg)
	if err != nil {
		return err
	}
	return emit(b)
}

func opK8sGenSecrets(a *App) error {
	b, err := k8s.GenSecrets(a.Cfg)
	if err != nil {
		return err
	}
	return emit(b)
}

// remove

// opK8sDelete deletes the broker CR, keeping its PVCs unless the layer question
// says otherwise (confirmLayer). Guarded by confirmDelete first: nothing is removed
// without a yes.
func opK8sDelete(a *App) error {
	if !confirmDelete(a, "broker "+a.Cfg.K8s.Name) {
		return nil
	}
	return k8sCluster(a).DeleteBroker(bg(), confirmLayer(a, layerData))
}

// opK8sRemoveSecrets / opK8sRemoveNamespace confirm like every other removal.
// Neither keeps a layer back -- `prepare` recreates both from the env file -- but
// deleting a namespace takes everything else that happens to live in it, which is
// exactly the kind of thing worth being asked about once.
func opK8sRemoveSecrets(a *App) error {
	if !confirmDelete(a, "the secrets for broker "+a.Cfg.K8s.Name) {
		return nil
	}
	return k8sCluster(a).DeleteSecrets(bg())
}

func opK8sRemoveNamespace(a *App) error {
	if !confirmDelete(a, "namespace "+a.Cfg.K8s.Namespace+" and everything in it") {
		return nil
	}
	return k8sCluster(a).DeleteNamespace(bg())
}
func opK8sTeardownDomainCerts(a *App) error {
	return k8sOps(a).RemoveDomainCerts(bg(), config.Primary, domainCANames(a.Cfg))
}

// orchestration

// opK8sDeployAll runs the full bring-up for THIS broker: check -> namespace ->
// secrets -> deploy -> (assert config-sync leader, HA only). Every step aborts loud
// on failure, and every step is scriptable -- nothing here asks a question.
//
// Two things are deliberately absent. The operator is cluster-scoped and outlives
// any one broker, so it belongs to `deploy operator`; this command is therefore
// safe against a cluster whose operator other brokers already share, and `check
// deploy` reports a missing one before the apply trips over it. Node labelling is
// absent because it cannot be scripted at all: the env file names the label a
// role's pods want, but only a person can say which machine carries it, so
// `prepare labels` is a one-off act of cluster provisioning.
func opK8sDeployAll(a *App) error {
	c := k8sCluster(a)
	ctx := bg()
	if err := opK8sCheck(a); err != nil {
		return err
	}
	if err := c.CreateNamespace(ctx); err != nil {
		return err
	}
	if err := c.CreateSecrets(ctx); err != nil {
		return err
	}
	if err := c.DeployBroker(ctx, false); err != nil {
		return err
	}
	if a.Cfg.RedundancyEnabled() {
		return k8sOps(a).Leader(ctx)
	}
	return nil
}

// opK8sRemoveAll tears down the broker and its namespace, leaving the cluster-scoped
// operator installed -- it may be serving brokers this env file knows nothing about,
// so removing it is its own explicit command. Guarded by the same confirm helpers as
// removing the broker alone.
func opK8sRemoveAll(a *App) error {
	if !confirmDelete(a, "broker "+a.Cfg.K8s.Name+" and its namespace") {
		return nil
	}
	deleteData := confirmLayer(a, layerData)
	c := k8sCluster(a)
	ctx := bg()
	if err := c.DeleteBroker(ctx, deleteData); err != nil {
		return err
	}
	if err := c.DeleteSecrets(ctx); err != nil {
		return err
	}
	if err := c.DeleteNamespace(ctx); err != nil {
		return err
	}
	step("operator kept -- it is cluster-scoped and may serve other brokers " +
		"(remove it with `solace-util remove operator`)")
	return nil
}

// podRole resolves the --pod flag to a role, defaulting to the Primary when unset.
func podRole(a *App) (config.Role, error) { return config.ParseRole(a.pod) }

// tlsConfigured reports whether a server certificate is available by either route: a
// managed TLS secret, or a cert+key file pair for the CLI path.
func tlsConfigured(cfg *config.Config) bool {
	return cfg.TLS.ServerSecret != "" || (cfg.TLS.Cert != "" && cfg.TLS.CertKey != "")
}

// domainCANames returns the configured domain CA names (the keys of domainCerts.files),
// unsorted; RemoveDomainCerts sorts and validates them.
func domainCANames(cfg *config.Config) []string {
	names := make([]string, 0, len(cfg.Broker.DomainCerts.Files))
	for ca := range cfg.Broker.DomainCerts.Files {
		names = append(names, ca)
	}
	return names
}
