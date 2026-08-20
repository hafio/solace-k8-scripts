# Command reference

Every command `solace-util` exposes, with its arguments and flags.

**Generated from the command tree -- do not edit by hand.** Regenerate after any
command, flag, or description change:

```
go test ./internal/cli -update
```

The `test` task fails while this file is stale, so it cannot drift from the code.

## Tree

```
solace-util
  auto-complete
    bash
    fish
    powershell
    zsh
  check
    deploy
    semp-login [role]
  cli [role]
  config
    apply
      additional-users
      domain-certs
      product-keys
      server-cert
    delete
      domain-certs
    disable
      default-users
      default-vpn
    leader [role]
  convert <bash-env-file>
  copy
    from files...
    into files...
  deploy
    all [role]
    broker [role]
    operator
  diagnostics
  generate
    broker [role]
    operator
    secrets
  logs
    broker [role]
    operator
  prepare
    all
    host
    labels
    namespace
    secrets
  remove
    all
    broker
    namespace
    operator
    secrets
  restart
    broker [role]
    operator
  shell [role]
  smoke
    redundancy [role]
  start
    broker
  status
    broker [role]
    operator
  stop
    broker
  version
```

## Global flags

Inherited by every command.

| Flag | Default | Meaning |
| --- | --- | --- |
| `--base-dir` | (none) | directory searched for the env file, and holding env/ (default: current directory) |
| `-e`, `--env` | `env.yaml` | env file name, searched in the base dir then &lt;base-dir&gt;/env; a value with a directory is used as-is |
| `--platform` | (none) | platform to drive: kubernetes (kube), docker (dk) or podman (pm). Default: the one the env file declares, or a prompt if it declares several |
| `-v`, `--verbose` | `false` | announce every external command as it runs; by default the binaries this env file names are resolved and listed once, up front |

## Commands

### solace-util

Deploy and operate Solace PubSub+ brokers on Kubernetes, Docker, or Podman

solace-util is a single CLI for deploying and operating Solace PubSub+ Event Brokers.
It presents the same lifecycle verbs on every platform, and every verb names
what it acts on -- run a verb on its own to see what it can act on:

  check deploy -> prepare all -> deploy all     build it
  config ...                                    POST-DEPLOYMENT, over the broker CLI
  check semp-login / smoke redundancy           prove it works
  stop broker / start broker                    pause it without removing it
  remove all                                    tear it down

The operator is cluster-scoped and shared, so it is installed and removed on
its own: `deploy operator`, `remove operator`.

`generate` renders any artifact to stdout without applying it -- that is how
you see what a command would send before you send it.

Every command takes -e/--env <file>, searched in the current directory then
./env. The platform comes from that file: whichever of kubernetes:, docker:
or podman: it declares is the one driven. A file declaring more than one asks
which to use, and --platform kubernetes|docker|podman (kube|dk|pm) answers that
up front. A few commands apply to only one platform; their help says so.

Coming from the bash scripts? 'solace-util convert <bash-env-file>' turns an old
env file into the YAML this reads.

```
solace-util
```

Subcommands: `auto-complete`, `check`, `cli`, `config`, `convert`, `copy`, `deploy`, `diagnostics`, `generate`, `logs`, `prepare`, `remove`, `restart`, `shell`, `smoke`, `start`, `status`, `stop`, `version`


### solace-util auto-complete

Print the shell auto-completion script for solace-util

Print a shell's completion script on stdout. Load it to complete commands and
flags, plus the values they take: env files for -e/--env, primary|backup|monitor
for the [role] positionals and --pod, and directories for --base-dir and --dir.

Completion never reads the env file, so it stays inert -- a TAB press cannot
parse config or run anything. See each shell's help for how to load it.

```
solace-util auto-complete
```

Subcommands: `bash`, `fish`, `powershell`, `zsh`


### solace-util auto-complete bash

Print the bash completion script

Load into the current shell:

  source <(solace-util auto-complete bash)

Load for every session (needs the bash-completion package):

  solace-util auto-complete bash > /etc/bash_completion.d/solace-util

```
solace-util auto-complete bash [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--no-descriptions` | `false` | omit the descriptions shown beside each completion |


### solace-util auto-complete fish

Print the fish completion script

Load into the current shell:

  solace-util auto-complete fish | source

Load for every session:

  solace-util auto-complete fish > ~/.config/fish/completions/solace-util.fish

```
solace-util auto-complete fish [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--no-descriptions` | `false` | omit the descriptions shown beside each completion |


### solace-util auto-complete powershell

Print the powershell completion script

Load into the current shell:

  solace-util auto-complete powershell | Out-String | Invoke-Expression

Load for every session, by writing the script once and sourcing it from
your profile:

  solace-util auto-complete powershell > solace-util.ps1

```
solace-util auto-complete powershell [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--no-descriptions` | `false` | omit the descriptions shown beside each completion |


### solace-util auto-complete zsh

Print the zsh completion script

Load into the current shell:

  source <(solace-util auto-complete zsh)

Load for every session (compinit must be enabled in ~/.zshrc):

  solace-util auto-complete zsh > "${fpath[1]}/_solace-util"

```
solace-util auto-complete zsh [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--no-descriptions` | `false` | omit the descriptions shown beside each completion |


### solace-util check

Run read-only checks

Every check here is read-only: it reports and changes nothing.

  check deploy      before deploying -- config, cluster/engine reachability,
                    storage or DNS, and whether the operator is installed
  check semp-login  after deploying -- the broker answers an authenticated
                    SEMP request

The failover exercise is deliberately not here: it moves live traffic, so it
lives under `smoke` with the other invasive checks.

```
solace-util check
```

Subcommands: `deploy`, `semp-login`

Also available as: ck


### solace-util check deploy

Validate config and platform prerequisites before deploying

```
solace-util check deploy [flags]
```

Also available as: dp

| Flag | Default | Meaning |
| --- | --- | --- |
| `--allow-command` | `[]` | approve one extra binary for the config's platform command, for this run only (repeatable; a bare name, never a path). The env file cannot grant this |


### solace-util check semp-login

Test an authenticated SEMP request against a running broker

```
solace-util check semp-login [role] [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--allow-command` | `[]` | approve one extra binary for the config's platform command, for this run only (repeatable; a bare name, never a path). The env file cannot grant this |


### solace-util cli

Open an interactive Solace CLI in the broker (Kubernetes: [role] picks the pod)

With no flags this opens an interactive Solace CLI session.

--input runs a script through that CLI instead of opening a session: a bare
filename is resolved under broker.cliScriptsFolder, a path is used as typed,
and the file is uploaded to the broker and run there. Errors reported by the
broker are surfaced as warnings, not failures -- a CLI script is a sequence of
independent commands, and one refused line does not invalidate the rest.

```
solace-util cli [role] [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--allow-command` | `[]` | approve one extra binary for the config's platform command, for this run only (repeatable; a bare name, never a path). The env file cannot grant this |
| `-i`, `--input` | (none) | run this Solace CLI script instead of opening an interactive session |
| `--pod` | (none) | pod role to target (p\|b\|m) (kubernetes only) |


### solace-util config

Configure a DEPLOYED broker (certs, hardening, product keys)

Post-deployment configuration: every step here talks to a broker that is already
deployed and running, over the Solace CLI. None of it is part of `deploy`.

There is no run-everything command, because these steps are not uniformly
re-runnable. The order that works on a fresh broker is:

  1. config leader                        (HA only; on containers, the primary)
  2. config apply server-cert             (when TLS is configured)
  3. config apply domain-certs            (when any are listed)
  4. config disable default-vpn
  5. config disable default-users
  6. config apply additional-users        (Kubernetes; after the hardening, so
                                           the sequence reads harden-then-provision.
                                           NOT re-runnable: the broker refuses to
                                           create a user that already exists)
  7. config apply product-keys            (when any are listed)

Only domain-certs can be undone from here (`config delete domain-certs`).
There is no un-harden, and no way to withdraw a server certificate or a
product key through this tool.

```
solace-util config
```

Subcommands: `apply`, `delete`, `disable`, `leader`

Also available as: cfg


### solace-util config apply

Apply configuration to the running broker

```
solace-util config apply
```

Subcommands: `additional-users`, `domain-certs`, `product-keys`, `server-cert`


### solace-util config apply additional-users

Create the admin.additionalUsers CLI users (not re-runnable) (kubernetes only)

```
solace-util config apply additional-users [flags]
```

Applies to: kubernetes. On any other platform this command fails rather than doing nothing.

| Flag | Default | Meaning |
| --- | --- | --- |
| `--allow-command` | `[]` | approve one extra binary for the config's platform command, for this run only (repeatable; a bare name, never a path). The env file cannot grant this |


### solace-util config apply domain-certs

Load the configured domain CA certificates

```
solace-util config apply domain-certs [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--allow-command` | `[]` | approve one extra binary for the config's platform command, for this run only (repeatable; a bare name, never a path). The env file cannot grant this |


### solace-util config apply product-keys

Apply the configured product keys

```
solace-util config apply product-keys [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--allow-command` | `[]` | approve one extra binary for the config's platform command, for this run only (repeatable; a bare name, never a path). The env file cannot grant this |


### solace-util config apply server-cert

Load/update the TLS server certificate

```
solace-util config apply server-cert [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--allow-command` | `[]` | approve one extra binary for the config's platform command, for this run only (repeatable; a bare name, never a path). The env file cannot grant this |


### solace-util config delete

Remove configuration from the running broker

Only domain certificates can be withdrawn this way. A server certificate, the
default-VPN hardening and an applied product key all stay applied.

```
solace-util config delete
```

Subcommands: `domain-certs`


### solace-util config delete domain-certs

Remove the configured domain CA certificates

```
solace-util config delete domain-certs [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--allow-command` | `[]` | approve one extra binary for the config's platform command, for this run only (repeatable; a bare name, never a path). The env file cannot grant this |


### solace-util config disable

Shut down the broker's built-in defaults (hardening)

Both steps are one-way: this tool has no command to re-enable what they shut down.

```
solace-util config disable
```

Subcommands: `default-users`, `default-vpn`


### solace-util config disable default-users

Shut down the default client-usernames in all VPNs

```
solace-util config disable default-users [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--allow-command` | `[]` | approve one extra binary for the config's platform command, for this run only (repeatable; a bare name, never a path). The env file cannot grant this |


### solace-util config disable default-vpn

Shut down the default message-VPN

```
solace-util config disable default-vpn [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--allow-command` | `[]` | approve one extra binary for the config's platform command, for this run only (repeatable; a bare name, never a path). The env file cannot grant this |


### solace-util config leader

Assert the config-sync leader (HA only) (containers: [role] is this host, detected from its name when omitted)

```
solace-util config leader [role] [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--allow-command` | `[]` | approve one extra binary for the config's platform command, for this run only (repeatable; a bare name, never a path). The env file cannot grant this |


### solace-util convert

Convert a legacy bash env file into a YAML env file

Convert a legacy bash env file -- the pre-Go format sourced by bash/000-env.sh --
into the YAML env file this CLI reads.

The target platform section is detected from the variables present; pass
--platform to choose it yourself. Variables with no YAML equivalent are
reported on stderr rather than dropped silently.

The output carries every secret from the source file verbatim, so treat it
like the source: write it with -o rather than through a shared terminal, and
never commit it.

  solace-util convert bash/env/prod -o prod.yaml
  solace-util convert bash/env/prod --platform podman -o prod.yaml
  solace-util check deploy -e prod.yaml

```
solace-util convert <bash-env-file> [flags]
```

Also available as: cv

| Flag | Default | Meaning |
| --- | --- | --- |
| `--force` | `false` | overwrite the --out file if it already exists |
| `-o`, `--out` | (none) | write the YAML here instead of stdout |


### solace-util copy

Copy files to/from the broker

```
solace-util copy
```

Subcommands: `from`, `into`

Also available as: cp


### solace-util copy from

Copy files from the broker to the host

```
solace-util copy from files... [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--allow-command` | `[]` | approve one extra binary for the config's platform command, for this run only (repeatable; a bare name, never a path). The env file cannot grant this |
| `--pod` | (none) | pod role to target (p\|b\|m) (kubernetes only) |


### solace-util copy into

Copy files from the host into the broker

```
solace-util copy into files... [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--allow-command` | `[]` | approve one extra binary for the config's platform command, for this run only (repeatable; a bare name, never a path). The env file cannot grant this |
| `--dir` | (none) | destination directory inside the broker |
| `--pod` | (none) | pod role to target (p\|b\|m) (kubernetes only) |


### solace-util deploy

Deploy the broker, the operator, or the whole broker stack

`deploy broker` applies just the broker. `deploy all` runs the whole bring-up
for it: check -> prepare -> deploy -> assert the config-sync leader (HA).

Neither installs the operator. It is cluster-scoped and may already be serving
other brokers, so `deploy operator` is its own command -- run it once per
cluster. `check deploy` reports when it is missing.

```
solace-util deploy
```

Subcommands: `all`, `broker`, `operator`

Also available as: dp


### solace-util deploy all

Orchestrate the whole bring-up for this broker

```
solace-util deploy all [role] [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--allow-command` | `[]` | approve one extra binary for the config's platform command, for this run only (repeatable; a bare name, never a path). The env file cannot grant this |
| `--restart` | `false` | restart an already-running broker when the deploy artifact changed (otherwise you are asked, and a non-interactive run leaves it running) (docker/podman only) |


### solace-util deploy broker

Deploy the broker (containers: this host's container, role required in HA)

```
solace-util deploy broker [role] [flags]
```

Also available as: br

| Flag | Default | Meaning |
| --- | --- | --- |
| `--allow-command` | `[]` | approve one extra binary for the config's platform command, for this run only (repeatable; a bare name, never a path). The env file cannot grant this |
| `--restart` | `false` | restart an already-running broker when the deploy artifact changed (otherwise you are asked, and a non-interactive run leaves it running) (docker/podman only) |


### solace-util deploy operator

Install the cluster-scoped EventBroker Operator (kubernetes only)

```
solace-util deploy operator [flags]
```

Also available as: op

Applies to: kubernetes. On any other platform this command fails rather than doing nothing.

| Flag | Default | Meaning |
| --- | --- | --- |
| `--allow-command` | `[]` | approve one extra binary for the config's platform command, for this run only (repeatable; a bare name, never a path). The env file cannot grant this |


### solace-util diagnostics

Gather a support bundle from the broker into broker.diagDir

```
solace-util diagnostics [flags]
```

Also available as: diag

| Flag | Default | Meaning |
| --- | --- | --- |
| `--allow-command` | `[]` | approve one extra binary for the config's platform command, for this run only (repeatable; a bare name, never a path). The env file cannot grant this |
| `--days` | `1` | days of logs/diagnostics to gather |


### solace-util generate

Render a deployment artifact to stdout without applying it

Nothing here contacts the cluster or the container engine, so it is safe to run
against an env file you have not vetted.

The nouns are the same ones the acting verbs use: `generate broker` renders what
`deploy broker` would apply, whichever platform that is -- a custom resource on
Kubernetes, a compose file or systemd quadlet on a container host (which is
per-host, so it takes a [role] there).

Only `operator` is platform-scoped, and because the thing does not exist
elsewhere rather than because it goes by another name: there is no container
operator to install.

```
solace-util generate
```

Subcommands: `broker`, `operator`, `secrets`

Also available as: gen


### solace-util generate broker

Render what `deploy broker` would apply

Kubernetes: the PubSubPlusEventBroker custom resource. Docker and podman: this
host's deploy artifact -- a compose file or a systemd quadlet unit -- which is
per-host, so [role] selects which node's artifact to render.

```
solace-util generate broker [role] [flags]
```

Also available as: br

| Flag | Default | Meaning |
| --- | --- | --- |
| `--allow-command` | `[]` | approve one extra binary for the config's platform command, for this run only (repeatable; a bare name, never a path). The env file cannot grant this |


### solace-util generate operator

Render the operator install bundle (kubernetes only)

```
solace-util generate operator [flags]
```

Also available as: op

Applies to: kubernetes. On any other platform this command fails rather than doing nothing.

| Flag | Default | Meaning |
| --- | --- | --- |
| `--allow-command` | `[]` | approve one extra binary for the config's platform command, for this run only (repeatable; a bare name, never a path). The env file cannot grant this |


### solace-util generate secrets

Render the secret-creation artifact (Kubernetes: Secret manifests; containers: a shell script)

```
solace-util generate secrets [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--allow-command` | `[]` | approve one extra binary for the config's platform command, for this run only (repeatable; a bare name, never a path). The env file cannot grant this |


### solace-util logs

Tail broker or operator logs

```
solace-util logs
```

Subcommands: `broker`, `operator`

Also available as: lg


### solace-util logs broker

Tail the broker's logs

```
solace-util logs broker [role] [flags]
```

Also available as: br

| Flag | Default | Meaning |
| --- | --- | --- |
| `--allow-command` | `[]` | approve one extra binary for the config's platform command, for this run only (repeatable; a bare name, never a path). The env file cannot grant this |


### solace-util logs operator

Tail the operator's controller logs (kubernetes only)

```
solace-util logs operator [flags]
```

Also available as: op

Applies to: kubernetes. On any other platform this command fails rather than doing nothing.

| Flag | Default | Meaning |
| --- | --- | --- |
| `--allow-command` | `[]` | approve one extra binary for the config's platform command, for this run only (repeatable; a bare name, never a path). The env file cannot grant this |


### solace-util prepare

Prepare the prerequisites a broker deployment needs

Everything a broker needs to exist before it is deployed.

`prepare all` runs the steps that are needed every time and need no input --
the namespace and its secrets on Kubernetes, the host on docker and podman --
so it is safe to script. `deploy all` runs the same steps for you.

Two things are deliberately outside it. The operator is cluster-scoped and
shared between brokers, so it is installed and removed on its own
(`deploy operator`). And `prepare labels` cannot be scripted at all: the env
file names the label each broker role wants, but only you can say which
machine should carry it, so it prompts -- run it once when provisioning the
cluster, not on every deployment.

```
solace-util prepare
```

Subcommands: `all`, `host`, `labels`, `namespace`, `secrets`

Also available as: pre


### solace-util prepare all

Run every applicable prepare step, in order

```
solace-util prepare all [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--allow-command` | `[]` | approve one extra binary for the config's platform command, for this run only (repeatable; a bare name, never a path). The env file cannot grant this |


### solace-util prepare host

Create/own the data dir, verify DNS, generate the redundancy PSK (docker/podman only)

```
solace-util prepare host [flags]
```

Applies to: docker, podman. On any other platform this command fails rather than doing nothing.

| Flag | Default | Meaning |
| --- | --- | --- |
| `--allow-command` | `[]` | approve one extra binary for the config's platform command, for this run only (repeatable; a bare name, never a path). The env file cannot grant this |


### solace-util prepare labels

Label cluster nodes for primary/backup/monitor placement (interactive, one-off) (kubernetes only)

```
solace-util prepare labels [flags]
```

Applies to: kubernetes. On any other platform this command fails rather than doing nothing.

| Flag | Default | Meaning |
| --- | --- | --- |
| `--allow-command` | `[]` | approve one extra binary for the config's platform command, for this run only (repeatable; a bare name, never a path). The env file cannot grant this |


### solace-util prepare namespace

Create the broker namespace (kubernetes only)

```
solace-util prepare namespace [flags]
```

Applies to: kubernetes. On any other platform this command fails rather than doing nothing.

| Flag | Default | Meaning |
| --- | --- | --- |
| `--allow-command` | `[]` | approve one extra binary for the config's platform command, for this run only (repeatable; a bare name, never a path). The env file cannot grant this |


### solace-util prepare secrets

Create admin/monitor, TLS, and image-pull secrets (kubernetes only)

```
solace-util prepare secrets [flags]
```

Applies to: kubernetes. On any other platform this command fails rather than doing nothing.

| Flag | Default | Meaning |
| --- | --- | --- |
| `--allow-command` | `[]` | approve one extra binary for the config's platform command, for this run only (repeatable; a bare name, never a path). The env file cannot grant this |


### solace-util remove

Remove the broker, the operator, or the whole broker stack

Every command here asks before it removes anything, and --no-prompt is the one
flag that makes it silent -- a script switches off one thing, not one per
question.

Nothing here removes the layer that is expensive to get back unless you say so:
the broker's persistent data and the operator's CRDs are kept by default, you
are asked about them separately, and what happened is printed either way. The
two flags compose, so an unattended removal that also drops the data is
`--delete-data --no-prompt`: naming the data you are willing to lose is not the
same as confirming the removal, so neither flag implies the other.

`remove all` takes this broker and its namespace. It leaves the operator, which
is cluster-scoped and may be serving brokers this env file does not describe.

```
solace-util remove
```

Subcommands: `all`, `broker`, `namespace`, `operator`, `secrets`

Also available as: rm


### solace-util remove all

Remove the broker, its secrets and its namespace (the operator is kept)

```
solace-util remove all [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--allow-command` | `[]` | approve one extra binary for the config's platform command, for this run only (repeatable; a bare name, never a path). The env file cannot grant this |
| `--delete-data` | `false` | delete the broker's persistent data too (Kubernetes PVCs / the container data directory). Without it the data is kept |
| `--no-prompt` | `false` | do not ask anything: proceed with the removal, and keep whatever is kept by default unless a --delete-* flag says otherwise |


### solace-util remove broker

Remove the deployed broker

```
solace-util remove broker [flags]
```

Also available as: br

| Flag | Default | Meaning |
| --- | --- | --- |
| `--allow-command` | `[]` | approve one extra binary for the config's platform command, for this run only (repeatable; a bare name, never a path). The env file cannot grant this |
| `--delete-data` | `false` | delete the broker's persistent data too (Kubernetes PVCs / the container data directory). Without it the data is kept |
| `--no-prompt` | `false` | do not ask anything: proceed with the removal, and keep whatever is kept by default unless a --delete-* flag says otherwise |


### solace-util remove namespace

Delete the broker's namespace (kubernetes only)

```
solace-util remove namespace [flags]
```

Applies to: kubernetes. On any other platform this command fails rather than doing nothing.

| Flag | Default | Meaning |
| --- | --- | --- |
| `--allow-command` | `[]` | approve one extra binary for the config's platform command, for this run only (repeatable; a bare name, never a path). The env file cannot grant this |
| `--no-prompt` | `false` | do not ask anything: proceed with the removal, and keep whatever is kept by default unless a --delete-* flag says otherwise |


### solace-util remove operator

Remove the cluster-scoped EventBroker Operator (kubernetes only)

```
solace-util remove operator [flags]
```

Also available as: op

Applies to: kubernetes. On any other platform this command fails rather than doing nothing.

| Flag | Default | Meaning |
| --- | --- | --- |
| `--allow-command` | `[]` | approve one extra binary for the config's platform command, for this run only (repeatable; a bare name, never a path). The env file cannot grant this |
| `--delete-crd` | `false` | delete the operator's CustomResourceDefinitions too. Without it they are kept, so existing brokers survive |
| `--no-prompt` | `false` | do not ask anything: proceed with the removal, and keep whatever is kept by default unless a --delete-* flag says otherwise |


### solace-util remove secrets

Delete the broker's secrets (kubernetes only)

```
solace-util remove secrets [flags]
```

Applies to: kubernetes. On any other platform this command fails rather than doing nothing.

| Flag | Default | Meaning |
| --- | --- | --- |
| `--allow-command` | `[]` | approve one extra binary for the config's platform command, for this run only (repeatable; a bare name, never a path). The env file cannot grant this |
| `--no-prompt` | `false` | do not ask anything: proceed with the removal, and keep whatever is kept by default unless a --delete-* flag says otherwise |


### solace-util restart

Bounce a running broker or the operator

Restarting applies nothing new. A changed deploy artifact needs
`deploy broker` (containers: with --restart), which rewrites it first.

```
solace-util restart
```

Subcommands: `broker`, `operator`

Also available as: rs


### solace-util restart broker

Restart the broker (Kubernetes: delete pods so the statefulset recreates them)

For kubernetes.updateStrategy=manualPodRestart: `deploy broker` updates the
statefulset's pod template but the operator waits for a pod to be deleted before
applying it.

With no role, every pod is restarted in the safe order (monitor, backup, primary;
standalone: just the primary), waiting for each to become ready before the next.
The order is by configured role, not by which node is currently active -- after a
failover they differ. Check `solace-util smoke redundancy` first, or pass a role
and restart them one at a time.

On docker and podman there is one broker per host and no role to pick: the
container is restarted in place.

```
solace-util restart broker [role] [flags]
```

Also available as: br

| Flag | Default | Meaning |
| --- | --- | --- |
| `--allow-command` | `[]` | approve one extra binary for the config's platform command, for this run only (repeatable; a bare name, never a path). The env file cannot grant this |
| `--no-prompt` | `false` | do not ask anything: proceed with the removal, and keep whatever is kept by default unless a --delete-* flag says otherwise |


### solace-util restart operator

Restart the operator's controller deployment (kubernetes only)

```
solace-util restart operator [flags]
```

Also available as: op

Applies to: kubernetes. On any other platform this command fails rather than doing nothing.

| Flag | Default | Meaning |
| --- | --- | --- |
| `--allow-command` | `[]` | approve one extra binary for the config's platform command, for this run only (repeatable; a bare name, never a path). The env file cannot grant this |


### solace-util shell

Open an interactive shell in the broker

```
solace-util shell [role] [flags]
```

Also available as: sh

| Flag | Default | Meaning |
| --- | --- | --- |
| `--allow-command` | `[]` | approve one extra binary for the config's platform command, for this run only (repeatable; a bare name, never a path). The env file cannot grant this |


### solace-util smoke

Run invasive checks that exercise the broker

These checks prove the broker works by making it work, so they disturb it.
Read-only questions live under `check`.

```
solace-util smoke
```

Subcommands: `redundancy`


### solace-util smoke redundancy

Exercise a real failover and fail back (HA only) (containers: [role] is this host, detected from its name when omitted)

```
solace-util smoke redundancy [role] [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--allow-command` | `[]` | approve one extra binary for the config's platform command, for this run only (repeatable; a bare name, never a path). The env file cannot grant this |


### solace-util start

Start a broker that is deployed but not running

```
solace-util start
```

Subcommands: `broker`


### solace-util start broker

Start the broker (Kubernetes: scale the statefulset(s) to 1; containers: start the container)

```
solace-util start broker [flags]
```

Also available as: br

| Flag | Default | Meaning |
| --- | --- | --- |
| `--allow-command` | `[]` | approve one extra binary for the config's platform command, for this run only (repeatable; a bare name, never a path). The env file cannot grant this |


### solace-util status

Report on the broker or the operator

By default this reports the RUNNING artifacts. --detail adds the static ones --
the full description of what is deployed, load balancer included.

```
solace-util status
```

Subcommands: `broker`, `operator`

Also available as: sts


### solace-util status broker

Show the broker's deployment status

```
solace-util status broker [role] [flags]
```

Also available as: br

| Flag | Default | Meaning |
| --- | --- | --- |
| `--all` | `false` | report every Solace broker found, not just the one this env file describes (Kubernetes: across all namespaces; docker/podman: every Solace container on this host) |
| `--allow-command` | `[]` | approve one extra binary for the config's platform command, for this run only (repeatable; a bare name, never a path). The env file cannot grant this |
| `--detail` | `false` | include the static artifacts, not just the running ones (Kubernetes: secrets, configmaps and PVCs; docker/podman: mounts, which is also where secrets appear) |


### solace-util status operator

Show the operator's controller status (kubernetes only)

```
solace-util status operator [flags]
```

Also available as: op

Applies to: kubernetes. On any other platform this command fails rather than doing nothing.

| Flag | Default | Meaning |
| --- | --- | --- |
| `--allow-command` | `[]` | approve one extra binary for the config's platform command, for this run only (repeatable; a bare name, never a path). The env file cannot grant this |
| `--detail` | `false` | include the full description of the operator deployment |


### solace-util stop

Stop a running broker without removing it

The deployment, its persistent data and its configuration all survive --
`start broker` brings it back. Use `remove broker` to delete it.

```
solace-util stop
```

Subcommands: `broker`


### solace-util stop broker

Stop the broker (Kubernetes: scale the statefulset(s) to 0; containers: stop the container)

```
solace-util stop broker [flags]
```

Also available as: br

| Flag | Default | Meaning |
| --- | --- | --- |
| `--allow-command` | `[]` | approve one extra binary for the config's platform command, for this run only (repeatable; a bare name, never a path). The env file cannot grant this |


### solace-util version

Print the solace-util version

Print the version this binary was built at, plus the Go toolchain and
platform that built it -- useful to paste alongside a support request.

A release binary (built by scripts/dev.sh or dev.ps1) reports the git tag
it shipped as, e.g. v1.2.3 -- matching the GitHub release exactly. A plain
`go build .` with no version stamped reports "dev".

```
solace-util version
```

Also available as: ver

