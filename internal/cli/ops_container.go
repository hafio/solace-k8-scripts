package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"solace/internal/broker"
	"solace/internal/config"
	"solace/internal/container"
	"solace/internal/render"
)

// The container (docker/podman) handlers wire the cobra tree to the two entry
// types, mirroring the k8s handlers: container.Manager (host operations --
// prep/deploy/delete/status, over engine.Runner) and broker.Ops (config/verify
// against the running broker, over the node-local container Transport). a.Platform
// selects docker vs podman downstream. Under --dry-run both run over engine.Echo
// (injected in load()), so every handler is exercisable without a real engine.
//
// The container transport is node-local (one broker per host), so every broker.Ops
// call targets a single nominal role (config.Primary); the HA coordination that k8s
// drives cross-pod is instead a per-host handshake (LeaderLocal/RedundancyLocal).
// Container config/verify reuse the shared kubernetes.* fields (DomainCerts, ProductKeys,
// DiagDir, CLIScriptsFolder) as the broker-ops config source -- no schema change.

// ctrOps builds a broker.Ops over the node-local container exec transport.
func ctrOps(a *App) *broker.Ops {
	return broker.New(container.NewTransport(a.Runner, a.Cfg, a.Platform), a.Cfg, step)
}

// ctrManager builds a container host Manager, wiring stdout as the report sink,
// stdin as the prompt source, and the resolved env path so PrepHost can write a
// generated PSK back into it.
func ctrManager(a *App) *container.Manager {
	m := container.NewManager(a.Runner, a.Cfg, a.Platform, step, os.Stdout)
	m.In = os.Stdin
	m.EnvPath = a.envPath
	m.RestartApproved = a.restart
	m.Confirm = func(question string) bool { return confirmRestart(a, question) }
	return m
}

// lifecycle

func opCtrCheck(a *App) error    { return ctrManager(a).Check(bg()) }
func opCtrPrepAll(a *App) error  { return ctrManager(a).PrepHost(bg()) }
func opCtrPrepHost(a *App) error { return ctrManager(a).PrepHost(bg()) }

// opCtrDeploy renders the deploy artifact and starts the container.
func opCtrDeploy(a *App, role config.Role) error {
	return ctrManager(a).Deploy(bg(), role)
}

// opCtrStartBroker / opCtrStopBroker / opCtrRestartBroker act on a container that
// is already deployed: the compose file or quadlet unit and the data directory all
// survive, so a stopped broker starts again without redeploying. They are the
// container half of what scaling a StatefulSet to 1 or 0 does on Kubernetes.
func opCtrStartBroker(a *App) error   { return ctrManager(a).Start(bg()) }
func opCtrStopBroker(a *App) error    { return ctrManager(a).Stop(bg()) }
func opCtrRestartBroker(a *App) error { return ctrManager(a).Restart(bg()) }

// config steps
//
// There is no run-everything step: these talk to a live broker over its CLI and are
// not uniformly re-runnable, so `config` documents the order rather than executing
// it. `leader` was always separate anyway -- it is primary-only and part of a
// cross-host handshake, so it fails loud on a backup or monitor host.

// opCtrConfigLeader asserts the config-sync leader from this host. It is
// primary-only and HA-only; the role arg (empty -> detect from hostname) lets an
// operator override detection. LeaderLocal fails loud on a backup/monitor host.
func opCtrConfigLeader(a *App, roleArg string) error {
	return ctrOps(a).LeaderLocal(bg(), roleArg)
}

// opCtrConfigServerCert loads the TLS server certificate via the CLI path (there
// is no secret-managed route for containers). Node-local: the single broker.
func opCtrConfigServerCert(a *App) error {
	return ctrOps(a).ServerCert(bg(), today(), config.Primary)
}

func opCtrConfigDomainCerts(a *App) error {
	return ctrOps(a).DomainCerts(bg(), config.Primary, a.Cfg.Broker.DomainCerts.Folder, a.Cfg.Broker.DomainCerts.Files)
}
func opCtrConfigDisableVPN(a *App) error   { return ctrOps(a).DisableDefaultVPN(bg(), config.Primary) }
func opCtrConfigDisableUsers(a *App) error { return ctrOps(a).DisableDefaultUsers(bg(), config.Primary) }
func opCtrConfigProductKeys(a *App) error {
	return ctrOps(a).ProductKeys(bg(), a.Cfg.Broker.ProductKeys, config.Primary)
}

// opCtrExecCLI uploads and runs a local Solace CLI script in the broker container.
// A bare filename (no path separator) resolves under the configured cliScripts
// folder; a path is used as-is -- the same rule as the k8s handler.
func opCtrExecCLI(a *App, file string) error {
	if file == "" {
		return fmt.Errorf("a CLI script file is required (e.g. `solace-util cli --input setup.cli`)")
	}
	localPath := file
	if !strings.ContainsAny(file, `/\`) {
		localPath = filepath.Join(a.Cfg.Broker.CLIScriptsFolder, file)
	}
	return ctrOps(a).ExecCLI(bg(), config.Primary, localPath)
}

// check / smoke steps

func opCtrVerifyLogin(a *App) error { return ctrLogin(a, ctrOps(a)) }

// opCtrVerifyRedundancy exercises failover from this host. The role arg (empty ->
// detect) picks the primary or backup half of the handshake; the monitor is
// rejected loud.
//
// PLACEHOLDER, NOT IMPLEMENTED: unlike Kubernetes -- where one kubectl context
// drives the whole redundancy group -- a container host can only run its own half,
// so the operator must start this on the primary and the backup themselves and the
// two rendezvous through the broker. Driving both from one point would need a
// control channel this tool does not have (SSH to the mate, or a coordinating
// broker session). Until then the two-host handshake is the documented procedure.
func opCtrVerifyRedundancy(a *App, roleArg string) error {
	return ctrOps(a).RedundancyLocal(bg(), roleArg)
}

// opCtrVerifyDiagnostics gathers show-command output and a diagnostics bundle from
// this host's broker into the configured diagnostics dir.
func opCtrVerifyDiagnostics(a *App) error {
	return ctrOps(a).Diagnostics(bg(), a.Cfg.Broker.DiagDir, nowStamp(), a.days, config.Primary)
}

// ctrLogin tests a SEMP login as the configured admin user against this host's
// broker. Login reports ok=false (not an error) on a failed login, so the handler
// turns a failed login into a non-zero exit.
func ctrLogin(a *App, o *broker.Ops) error {
	ok, err := o.Login(bg(), config.Primary, a.Cfg.Admin.User, a.Cfg.Admin.Pass)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("SEMP login failed on this host's broker (see reason above)")
	}
	return nil
}

// day-2 ops (node-local)
func opCtrStatus(a *App) error   { return ctrManager(a).Status(bg()) }
func opCtrLogs(a *App) error     { return ctrManager(a).Logs(bg()) }
func opCtrCLI(a *App) error      { return ctrManager(a).CLI(bg()) }
func opCtrShell(a *App) error    { return ctrManager(a).Shell(bg()) }
func opCtrDescribe(a *App) error { return ctrManager(a).Describe(bg()) }

// opCtrStatusBroker reports on this host's broker: the running-artifact summary,
// plus the full inspection (and, on podman, the installed unit) under --detail.
//
// --all widens it from the container this env file names to every Solace broker
// container on the host, found by image rather than by name -- the host-local
// answer to the cluster survey `--all` gives on Kubernetes. One deployed by hand,
// or under a name this config never mentions, still shows up.
func opCtrStatusBroker(a *App) error {
	if a.all {
		return ctrManager(a).StatusAll(bg(), a.detail)
	}
	if err := opCtrStatus(a); err != nil {
		return err
	}
	if !a.detail {
		return nil
	}
	return opCtrDescribe(a)
}

func opCtrCopyFrom(a *App, files []string) error { return ctrManager(a).CopyFrom(bg(), files) }
func opCtrCopyInto(a *App, files []string) error {
	return ctrManager(a).CopyInto(bg(), files, a.destDir)
}

// opCtrTeardownDomainCerts removes the configured domain CAs from this host's
// broker. The operation is platform-agnostic and already ran over this transport
// for the loading half; domainCANames is shared with the k8s handler.
func opCtrTeardownDomainCerts(a *App) error {
	return ctrOps(a).RemoveDomainCerts(bg(), config.Primary, domainCANames(a.Cfg))
}
// opCtrGenArtifact renders this host's deploy artifact (podman quadlet / docker
// compose file) without applying it.
func opCtrGenArtifact(a *App, role config.Role) error {
	id := a.Cfg.ResolveNode(role)
	if a.Platform == config.Podman {
		return emit(render.Quadlet(a.Cfg, id))
	}
	return emit(render.Compose(a.Cfg, id))
}

// opCtrGenSecrets renders the shell that supplies this host's secrets (podman:
// create them in its store; docker: export the variables compose reads). It gets
// the same preflight the real deploy does -- printing a script that would create an
// empty secret is worse than refusing.
func opCtrGenSecrets(a *App) error {
	if err := render.SecretPreflight(a.Cfg, a.Platform); err != nil {
		return err
	}
	return emit(render.SecretScript(a.Cfg, a.Platform))
}


// remove + orchestration

// opCtrDelete stops and removes the broker container, guarded by confirmDelete
// (nothing goes without a yes) and confirmLayer (the data directory is kept unless
// asked for by name).
func opCtrDelete(a *App) error {
	if !confirmDelete(a, containerWhat(a)) {
		return nil
	}
	return ctrManager(a).Delete(bg(), confirmLayer(a, layerData))
}

// opCtrDeployAll runs the full node-local bring-up: check -> prep host -> deploy.
// The cross-host config-sync leader (HA, primary-only) is a separate explicit step.
func opCtrDeployAll(a *App, role config.Role) error {
	m := ctrManager(a)
	ctx := bg()
	if err := m.Check(ctx); err != nil {
		return err
	}
	if err := m.PrepHost(ctx); err != nil {
		return err
	}
	return m.Deploy(ctx, role)
}

// opCtrRemoveAll tears down this host's broker. Unlike Kubernetes -- where removing
// everything also takes the secrets and the namespace -- a container host has no
// layer above the broker, so removing all of it is exactly removing the broker.
func opCtrRemoveAll(a *App) error { return opCtrDelete(a) }

// containerWhat labels the delete/down confirmation target.
func containerWhat(a *App) string {
	return fmt.Sprintf("%s broker container %s", a.Platform, a.Cfg.ContainerBlock(a.Platform).Name)
}
