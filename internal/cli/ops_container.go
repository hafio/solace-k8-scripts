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
// Container config/verify reuse the shared k8s.* fields (DomainCerts, ProductKeys,
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
	m.Restart = a.restart
	m.Confirm = func(question string) bool { return confirmRestart(a, question) }
	return m
}

// lifecycle

func opCtrCheck(a *App) error    { return ctrManager(a).Check(bg()) }
func opCtrPrepAll(a *App) error  { return ctrManager(a).PrepHost(bg()) }
func opCtrPrepHost(a *App) error { return ctrManager(a).PrepHost(bg()) }

// opCtrDeploy renders the deploy artifact and starts the container; with a
// --gen-*-only flag it only prints the requested artifact.
func opCtrDeploy(a *App, role config.Role) error {
	if a.anyGen() {
		return emitCtrArtifact(a, role)
	}
	return ctrManager(a).Deploy(bg(), role)
}

// config steps

// opCtrConfigAll runs every applicable post-deploy config step locally on this
// host's broker: server-cert (when a cert+key pair is configured), domain-certs
// (when any are listed), the VPN/user hardening (always), and product-keys (when
// any are listed). Unlike k8s ConfigAll it does NOT run leader: assert-leader is
// primary-only and part of the cross-host handshake, so it is a separate explicit
// `config leader` step -- bundling it would fail loud on a backup/monitor host.
func opCtrConfigAll(a *App) error {
	ctx := bg()
	o := ctrOps(a)
	if a.Cfg.TLS.Cert != "" && a.Cfg.TLS.CertKey != "" {
		if err := o.ServerCert(ctx, today(), config.Primary); err != nil {
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
	if len(a.Cfg.K8s.ProductKeys) > 0 {
		return o.ProductKeys(ctx, a.Cfg.K8s.ProductKeys, config.Primary)
	}
	return nil
}

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
	return ctrOps(a).DomainCerts(bg(), config.Primary, a.Cfg.K8s.DomainCerts.Folder, a.Cfg.K8s.DomainCerts.Files)
}
func opCtrConfigDisableVPN(a *App) error   { return ctrOps(a).DisableDefaultVPN(bg(), config.Primary) }
func opCtrConfigDisableUsers(a *App) error { return ctrOps(a).DisableDefaultUsers(bg(), config.Primary) }
func opCtrConfigProductKeys(a *App) error {
	return ctrOps(a).ProductKeys(bg(), a.Cfg.K8s.ProductKeys, config.Primary)
}

// opCtrExecCLI uploads and runs a local Solace CLI script in the broker container.
// A bare filename (no path separator) resolves under the configured cliScripts
// folder; a path is used as-is -- the same rule as the k8s handler.
func opCtrExecCLI(a *App, file string) error {
	if file == "" {
		return fmt.Errorf("a CLI script file is required (e.g. `solace %s config exec-cli setup.cli`)", a.Platform)
	}
	localPath := file
	if !strings.ContainsAny(file, `/\`) {
		localPath = filepath.Join(a.Cfg.K8s.CLIScriptsFolder, file)
	}
	return ctrOps(a).ExecCLI(bg(), config.Primary, localPath)
}

// verify steps

// opCtrVerifyAll runs the health checks on this host: the redundancy failover
// exercise (HA only; skipped on the monitor, which cannot run it) followed by a
// SEMP login. Standalone skips the redundancy exercise entirely.
func opCtrVerifyAll(a *App) error {
	ctx := bg()
	o := ctrOps(a)
	if a.Cfg.RedundancyEnabled() {
		role, err := o.LocalRole("")
		if err != nil {
			return err
		}
		if role == config.Monitor {
			step("verify redundancy skipped on the monitor node (run it on the primary and backup)")
		} else if err := o.RedundancyLocal(ctx, ""); err != nil {
			return err
		}
	}
	return ctrLogin(a, o)
}

func opCtrVerifyLogin(a *App) error { return ctrLogin(a, ctrOps(a)) }

// opCtrVerifyRedundancy exercises failover from this host. The role arg (empty ->
// detect) picks the primary or backup half of the handshake; the monitor is
// rejected loud. Run it on the primary and backup concurrently.
func opCtrVerifyRedundancy(a *App, roleArg string) error {
	return ctrOps(a).RedundancyLocal(bg(), roleArg)
}

// opCtrVerifyDiagnostics gathers show-command output and a diagnostics bundle from
// this host's broker into the configured diagnostics dir.
func opCtrVerifyDiagnostics(a *App) error {
	return ctrOps(a).Diagnostics(bg(), a.Cfg.K8s.DiagDir, nowStamp(), a.days, config.Primary)
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
func opCtrGen(a *App, role config.Role) error {
	return emitCtrArtifact(a, role)
}

// emitCtrArtifact prints the artifact the gen flag asked for, changing nothing.
// The flag picks the artifact, not the command: --gen-only renders the deploy
// artifact (podman quadlet / docker compose file), --gen-secrets-only the shell
// that supplies this host's secrets (podman: create them in its store; docker:
// export the variables compose reads), --gen-env-only the broker settings as an
// env file. Ports the gen_*() + --only-gen contract of the bash deploy scripts.
func emitCtrArtifact(a *App, role config.Role) error {
	id := a.Cfg.ResolveNode(role)
	switch {
	case a.GenSecretsOnly:
		// The script's own header tells the operator to run it, so it gets the same
		// preflight the real deploy does -- printing one that would create an empty
		// secret is worse than refusing.
		if err := render.SecretPreflight(a.Cfg, a.Platform); err != nil {
			return err
		}
		return emit(render.SecretScript(a.Cfg, a.Platform))
	case a.GenEnvOnly:
		return emit(render.EnvFile(a.Cfg, id))
	case a.Platform == config.Podman:
		return emit(render.Quadlet(a.Cfg, id))
	default: // docker
		return emit(render.Compose(a.Cfg, id))
	}
}

// teardown + orchestration

// opCtrDelete stops and removes the broker container, guarded by confirmDelete
// (keep-by-default) and confirmPurge (data dir kept unless --purge/confirmed).
func opCtrDelete(a *App) error {
	if !confirmDelete(a, containerWhat(a)) {
		return nil
	}
	return ctrManager(a).Delete(bg(), confirmPurge(a))
}

// opCtrUp runs the full node-local bring-up: check -> prep host -> deploy. The
// cross-host config-sync leader (HA, primary-only) is a separate explicit step.
func opCtrUp(a *App, role config.Role) error {
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

// opCtrDown tears down the broker container. Unlike k8s (where down also removes
// secrets and the namespace), a container host has no layer above the broker, so
// down is exactly delete.
func opCtrDown(a *App) error { return opCtrDelete(a) }

// containerWhat labels the delete/down confirmation target.
func containerWhat(a *App) string {
	return fmt.Sprintf("%s broker container %s", a.Platform, a.Cfg.ContainerBlock(a.Platform).Name)
}
