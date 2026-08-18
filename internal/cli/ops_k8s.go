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

func opK8sCheck(a *App) error { return k8sCluster(a).Check(bg()) }

// opK8sPrepAll runs every prep step in order (operator, namespace, secrets, labels).
// Each step self-skips when its config is absent (LabelNodes early-exits with no custom
// labels), so a standalone/minimal env runs the applicable subset.
func opK8sPrepAll(a *App) error {
	c := k8sCluster(a)
	ctx := bg()
	if err := c.OperatorApply(ctx); err != nil {
		return err
	}
	if err := c.CreateNamespace(ctx); err != nil {
		return err
	}
	if err := c.CreateSecrets(ctx); err != nil {
		return err
	}
	return c.LabelNodes(ctx)
}

// opK8sDeploy renders and applies the broker CR; with a --gen-*-only flag it
// prints the requested manifest only.
func opK8sDeploy(a *App) error {
	if a.anyGen() {
		return emitK8sArtifact(a, "broker")
	}
	return k8sCluster(a).DeployBroker(bg(), a.keepYAML)
}

// prep steps
func opK8sPrepNamespace(a *App) error { return k8sCluster(a).CreateNamespace(bg()) }
func opK8sPrepLabels(a *App) error    { return k8sCluster(a).LabelNodes(bg()) }

// opK8sPrepSecrets applies the broker secrets; a gen flag prints the manifests
// instead. Unlike the other prep steps it is gen-capable, since the secrets are
// its own artifact -- `prep secrets --gen-only` and `--gen-secrets-only` are the
// same rendering here.
func opK8sPrepSecrets(a *App) error {
	if a.anyGen() {
		return emitK8sArtifact(a, "secrets")
	}
	return k8sCluster(a).CreateSecrets(bg())
}

// config steps

// opK8sConfigAll runs every applicable post-deploy config step in order, skipping the
// ones whose config is absent: leader (HA only), server-cert (when TLS is configured),
// domain-certs (when any are listed), the VPN/user hardening (always), and product-keys
// (when any are listed).
func opK8sConfigAll(a *App) error {
	ctx := bg()
	o := k8sOps(a)
	if a.Cfg.RedundancyEnabled() {
		if err := o.Leader(ctx); err != nil {
			return err
		}
	}
	if tlsConfigured(a.Cfg) {
		if err := opK8sConfigServerCert(a); err != nil {
			return err
		}
	}
	if len(a.Cfg.K8s.DomainCerts.Files) > 0 {
		if err := o.DomainCerts(ctx, config.Primary, a.Cfg.K8s.DomainCerts.Folder, a.Cfg.K8s.DomainCerts.Files); err != nil {
			return err
		}
	}
	if err := o.DisableDefaultVPN(ctx, config.Primary); err != nil {
		return err
	}
	if err := o.DisableDefaultUsers(ctx, config.Primary); err != nil {
		return err
	}
	// After the default-user hardening, so the sequence reads harden-then-provision.
	// NOTE: `create username` fails on a user that already exists, so with
	// additionalUsers configured a second `config` run stops here -- unlike every other
	// step, this one is not re-runnable. Run `config additional-users` deliberately, or
	// the individual steps, once the users exist.
	if len(a.Cfg.Admin.AdditionalUsers) > 0 {
		if err := o.AdditionalUsers(ctx, config.Primary, a.Cfg.Admin.AdditionalUsers); err != nil {
			return err
		}
	}
	if len(a.Cfg.K8s.ProductKeys) > 0 {
		return o.ProductKeys(ctx, a.Cfg.K8s.ProductKeys, k8s.ProductKeyRoles(a.Cfg)...)
	}
	return nil
}

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
	return k8sOps(a).DomainCerts(bg(), config.Primary, a.Cfg.K8s.DomainCerts.Folder, a.Cfg.K8s.DomainCerts.Files)
}
func opK8sConfigDisableVPN(a *App) error   { return k8sOps(a).DisableDefaultVPN(bg(), config.Primary) }
func opK8sConfigDisableUsers(a *App) error { return k8sOps(a).DisableDefaultUsers(bg(), config.Primary) }
func opK8sConfigProductKeys(a *App) error {
	return k8sOps(a).ProductKeys(bg(), a.Cfg.K8s.ProductKeys, k8s.ProductKeyRoles(a.Cfg)...)
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
		return fmt.Errorf("a CLI script file is required (e.g. `solace-util k8s config exec-cli setup.cli`)")
	}
	role, err := podRole(a)
	if err != nil {
		return err
	}
	localPath := file
	if !strings.ContainsAny(file, `/\`) {
		localPath = filepath.Join(a.Cfg.K8s.CLIScriptsFolder, file)
	}
	return k8sOps(a).ExecCLI(bg(), role, localPath)
}

// verify steps

// opK8sVerifyAll runs the health checks: the redundancy failover exercise (HA only;
// standalone self-skips) followed by a SEMP login.
func opK8sVerifyAll(a *App) error {
	o := k8sOps(a)
	if err := o.Redundancy(bg()); err != nil {
		return err
	}
	return k8sLogin(a, o, config.Primary)
}

func opK8sVerifyRedundancy(a *App) error { return k8sOps(a).Redundancy(bg()) }

// opK8sVerifyDiagnostics gathers show-command output and a diagnostics bundle from every
// broker node into the configured diagnostics dir.
func opK8sVerifyDiagnostics(a *App) error {
	return k8sOps(a).Diagnostics(bg(), a.Cfg.K8s.DiagDir, nowStamp(), a.days, k8s.HARoles(a.Cfg)...)
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
func opK8sStatus(a *App) error    { return k8sCluster(a).Status(bg()) }
func opK8sShowAll(a *App) error   { return k8sCluster(a).ShowAll(bg()) }
func opK8sDescribeLB(a *App) error { return k8sCluster(a).DescribeLB(bg()) }

func opK8sLogs(a *App, role config.Role) error           { return k8sCluster(a).Logs(bg(), role, nil) }
func opK8sCLI(a *App, role config.Role) error            { return k8sCluster(a).CLI(bg(), role) }
func opK8sShell(a *App, role config.Role) error          { return k8sCluster(a).Shell(bg(), role) }
func opK8sDescribeBroker(a *App, role config.Role) error { return k8sCluster(a).DescribeBroker(bg(), role) }

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

func opK8sReplicasStart(a *App) error { return k8sCluster(a).ReplicasStart(bg()) }
func opK8sReplicasStop(a *App) error  { return k8sCluster(a).ReplicasStop(bg()) }

// opK8sRestart bounces broker pods for a manualPodRestart upgrade. Deleting a pod
// interrupts messaging on that node, so it takes the same confirmation a delete
// does: --yes proceeds, an interactive session is asked, and a non-interactive one
// without --yes refuses rather than bouncing a production broker unattended.
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
	if a.anyGen() {
		return emitK8sArtifact(a, "operator")
	}
	return k8sCluster(a).OperatorApply(bg())
}

func opK8sOperatorDelete(a *App) error   { return k8sCluster(a).OperatorDelete(bg()) }
func opK8sOperatorStatus(a *App) error   { return k8sCluster(a).OperatorStatus(bg()) }
func opK8sOperatorLogs(a *App) error     { return k8sCluster(a).OperatorLogs(bg()) }
func opK8sOperatorDescribe(a *App) error { return k8sCluster(a).OperatorDescribe(bg()) }

// opK8sGen renders a manifest to stdout without applying.
func opK8sGen(a *App, target string) error { return emitK8sArtifact(a, target) }

// emitK8sArtifact prints one k8s manifest and changes nothing. target is what the
// calling command renders by default; --gen-secrets-only overrides it, so the flag
// selects the artifact uniformly across the tree the way it does for containers.
// --gen-env-only has no k8s meaning: broker settings live in the custom resource,
// not an env file, so it is rejected rather than silently treated as --gen-only.
func emitK8sArtifact(a *App, target string) error {
	if a.GenEnvOnly {
		return fmt.Errorf("--gen-env-only renders a container env file and has no Kubernetes equivalent " +
			"(the broker settings are part of the custom resource); use --gen-only")
	}
	if a.GenSecretsOnly {
		target = "secrets"
	}
	switch target {
	case "", "broker":
		return emit(render.BrokerCR(a.Cfg))
	case "operator":
		b, err := k8s.GenOperator(a.Cfg)
		if err != nil {
			return err
		}
		return emit(b)
	case "secrets":
		b, err := k8s.GenSecrets(a.Cfg)
		if err != nil {
			return err
		}
		return emit(b)
	default:
		return fmt.Errorf("unknown gen target %q (expected broker|operator|secrets)", target)
	}
}

// teardown

// opK8sDelete deletes the broker CR. It is guarded by confirmDelete (keep-by-default:
// no delete without confirmation) and confirmPurge (PVCs kept unless --purge/confirmed).
func opK8sDelete(a *App) error {
	if !confirmDelete(a, "broker "+a.Cfg.K8s.Name) {
		return nil
	}
	return k8sCluster(a).DeleteBroker(bg(), confirmPurge(a))
}

func opK8sTeardownSecrets(a *App) error   { return k8sCluster(a).DeleteSecrets(bg()) }
func opK8sTeardownNamespace(a *App) error { return k8sCluster(a).DeleteNamespace(bg()) }
func opK8sTeardownDomainCerts(a *App) error {
	return k8sOps(a).RemoveDomainCerts(bg(), config.Primary, domainCANames(a.Cfg))
}

// orchestration

// opK8sUp runs the full bring-up: check -> operator -> namespace -> secrets ->
// (labels, only when placement is configured and stdin is a TTY, so `up` stays
// scriptable) -> deploy -> (assert config-sync leader, HA only). Every step aborts
// loud on failure.
func opK8sUp(a *App) error {
	c := k8sCluster(a)
	ctx := bg()
	if err := c.Check(ctx); err != nil {
		return err
	}
	if err := c.OperatorApply(ctx); err != nil {
		return err
	}
	if err := c.CreateNamespace(ctx); err != nil {
		return err
	}
	if err := c.CreateSecrets(ctx); err != nil {
		return err
	}
	if placementConfigured(a.Cfg) && interactive(a) {
		if err := c.LabelNodes(ctx); err != nil {
			return err
		}
	}
	if err := c.DeployBroker(ctx, a.keepYAML); err != nil {
		return err
	}
	if a.Cfg.RedundancyEnabled() {
		return k8sOps(a).Leader(ctx)
	}
	return nil
}

// opK8sDown tears down the broker and its namespace, leaving the cluster-scoped operator
// installed. Guarded by the same confirm helpers as delete.
func opK8sDown(a *App) error {
	if !confirmDelete(a, "broker "+a.Cfg.K8s.Name+" and its namespace") {
		return nil
	}
	purge := confirmPurge(a)
	c := k8sCluster(a)
	ctx := bg()
	if err := c.DeleteBroker(ctx, purge); err != nil {
		return err
	}
	if err := c.DeleteSecrets(ctx); err != nil {
		return err
	}
	return c.DeleteNamespace(ctx)
}

// podRole resolves the --pod flag to a role, defaulting to the Primary when unset.
func podRole(a *App) (config.Role, error) { return config.ParseRole(a.pod) }

// tlsConfigured reports whether a server certificate is available by either route: a
// managed TLS secret, or a cert+key file pair for the CLI path.
func tlsConfigured(cfg *config.Config) bool {
	return cfg.TLS.ServerSecret != "" || (cfg.TLS.Cert != "" && cfg.TLS.CertKey != "")
}

// placementConfigured reports whether any per-role node placement labels are set, gating
// the interactive LabelNodes step in `up`.
func placementConfigured(cfg *config.Config) bool {
	p := cfg.K8s.Placement
	return len(p.LabelsPrimary) > 0 || len(p.LabelsBackup) > 0 || len(p.LabelsMonitor) > 0
}

// domainCANames returns the configured domain CA names (the keys of domainCerts.files),
// unsorted; RemoveDomainCerts sorts and validates them.
func domainCANames(cfg *config.Config) []string {
	names := make([]string, 0, len(cfg.K8s.DomainCerts.Files))
	for ca := range cfg.K8s.DomainCerts.Files {
		names = append(names, ca)
	}
	return names
}
