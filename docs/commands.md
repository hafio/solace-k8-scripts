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
  completion
    bash
    fish
    powershell
    zsh
  convert <bash-env-file>
  docker
    check
    cli
    config
      disable-default-users
      disable-default-vpn
      domain-certs
      exec-cli [file]
      leader [primary|backup|monitor]
      product-keys
      server-cert
    copy
      from files...
      into files...
    delete
    deploy [primary|backup|monitor]
    describe
    down
    gen [primary|backup|monitor]
    logs
    prep
      host
    shell
    status
    teardown
      domain-certs
    up [primary|backup|monitor]
    verify
      diagnostics
      login
      redundancy [primary|backup|monitor]
  k8s
    check
    cli [role]
    config
      additional-users
      disable-default-users
      disable-default-vpn
      domain-certs
      exec-cli [file]
      leader
      product-keys
      server-cert
    copy
      from files...
      into files...
    delete
    deploy
    describe
      broker [role]
      lb
    down
    gen [broker|operator|secrets]
    logs [role]
    operator
      delete
      deploy
      describe
      logs
      status
    prep
      labels
      namespace
      operator
      secrets
    replicas
      start
      stop
    restart [primary|backup|monitor]
    shell [role]
    show-all
    status
    teardown
      domain-certs
      namespace
      secrets
    up
    verify
      diagnostics
      login [role]
      redundancy
  podman
    check
    cli
    config
      disable-default-users
      disable-default-vpn
      domain-certs
      exec-cli [file]
      leader [primary|backup|monitor]
      product-keys
      server-cert
    copy
      from files...
      into files...
    delete
    deploy [primary|backup|monitor]
    describe
    down
    gen [primary|backup|monitor]
    logs
    prep
      host
    shell
    status
    teardown
      domain-certs
    up [primary|backup|monitor]
    verify
      diagnostics
      login
      redundancy [primary|backup|monitor]
  version
```

## Global flags

Inherited by every command.

| Flag | Default | Meaning |
| --- | --- | --- |
| `--base-dir` | (none) | directory searched for the env file, and holding env/ (default: current directory) |
| `--dry-run` | `false` | print the external commands instead of running them |
| `-e`, `--env` | `env.yaml` | env file name, searched in the base dir then &lt;base-dir&gt;/env; a value with a directory is used as-is |
| `--gen-env-only` | `false` | render the container broker settings as an env file and print them; change nothing (docker/podman only) |
| `--gen-only` | `false` | render the deployment artifact this command would apply and print it; change nothing |
| `--gen-secrets-only` | `false` | render this deployment's secrets (k8s Secret manifests; podman secret-create commands; docker export lines to source) and print them; change nothing |
| `-v`, `--verbose` | `false` | announce every external command as it runs; by default the binaries this env file names are resolved and listed once, up front |
| `-y`, `--yes` | `false` | skip confirmation prompts (does NOT imply --purge) |

## Commands

### solace-util

Deploy and operate Solace PubSub+ brokers on Kubernetes, Docker, or Podman

solace-util is a single CLI for deploying and operating Solace PubSub+ Event Brokers.
It presents the same lifecycle verbs on every platform:

  check -> prep -> deploy       build the deployment   (up)
  config -> verify              POST-DEPLOYMENT, over the broker CLI
  delete -> teardown            tear it down           (down)

'up' covers only the first line. config and verify drive the Solace CLI
inside a broker that is already running, so run them once it is ready.

Pick a platform (k8s, docker, podman), then a verb. Every command takes
-e/--env <file>, searched in the current directory then ./env.
Coming from the bash scripts? 'solace-util convert <bash-env-file>' turns an old
env file into the YAML this reads. See 'solace-util <platform> --help'.

```
solace-util
```

Subcommands: `completion`, `convert`, `docker`, `k8s`, `podman`, `version`


### solace-util completion

Print the shell completion script for solace-util

Print a shell's completion script on stdout. Load it to complete commands and
flags, plus the values they take: env files for -e/--env, primary|backup|monitor
for the [role] positionals and --pod, and directories for --base-dir and --dir.

Completion never reads the env file, so it stays inert -- a TAB press cannot
parse config or run anything. See each shell's help for how to load it.

```
solace-util completion
```

Subcommands: `bash`, `fish`, `powershell`, `zsh`


### solace-util completion bash

Print the bash completion script

Load into the current shell:

  source <(solace-util completion bash)

Load for every session (needs the bash-completion package):

  solace-util completion bash > /etc/bash_completion.d/solace-util

```
solace-util completion bash [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--no-descriptions` | `false` | omit the descriptions shown beside each completion |


### solace-util completion fish

Print the fish completion script

Load into the current shell:

  solace-util completion fish | source

Load for every session:

  solace-util completion fish > ~/.config/fish/completions/solace-util.fish

```
solace-util completion fish [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--no-descriptions` | `false` | omit the descriptions shown beside each completion |


### solace-util completion powershell

Print the powershell completion script

Load into the current shell:

  solace-util completion powershell | Out-String | Invoke-Expression

Load for every session, by writing the script once and sourcing it from
your profile:

  solace-util completion powershell > solace-util.ps1

```
solace-util completion powershell [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--no-descriptions` | `false` | omit the descriptions shown beside each completion |


### solace-util completion zsh

Print the zsh completion script

Load into the current shell:

  source <(solace-util completion zsh)

Load for every session (compinit must be enabled in ~/.zshrc):

  solace-util completion zsh > "${fpath[1]}/_solace-util"

```
solace-util completion zsh [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--no-descriptions` | `false` | omit the descriptions shown beside each completion |


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
  solace-util k8s check -e prod.yaml --dry-run

```
solace-util convert <bash-env-file> [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--force` | `false` | overwrite the --out file if it already exists |
| `-o`, `--out` | (none) | write the YAML here instead of stdout |
| `--platform` | (none) | platform section to write: k8s, docker, or podman (default: detect) |


### solace-util docker

Deploy/operate the broker directly on a host with Docker

```
solace-util docker
```

Subcommands: `check`, `cli`, `config`, `copy`, `delete`, `deploy`, `describe`, `down`, `gen`, `logs`, `prep`, `shell`, `status`, `teardown`, `up`, `verify`

| Flag | Default | Meaning |
| --- | --- | --- |
| `--allow-command` | `[]` | approve one extra binary for the config's platform command, for this run only (repeatable; a bare name, never a path). The env file cannot grant this |


### solace-util docker check

Validate config, DNS, and container runtime

```
solace-util docker check
```


### solace-util docker cli

Open an interactive Solace CLI in the container

```
solace-util docker cli
```


### solace-util docker config

Configure a DEPLOYED broker (certs, hardening, product keys, CLI)

Post-deployment configuration: every step here talks to a broker that is already running in this host's container, over the Solace CLI. Nothing under `config` is part of `deploy`, and none of it is applied by `up` -- run it once the container is up.

With no subcommand, runs all applicable config steps (HA-only steps skipped in standalone), except `leader`: that one is cross-host and primary-only, so run it explicitly on the primary.

```
solace-util docker config
```

Subcommands: `disable-default-users`, `disable-default-vpn`, `domain-certs`, `exec-cli`, `leader`, `product-keys`, `server-cert`


### solace-util docker config disable-default-users

Shut down default client-usernames in all VPNs

```
solace-util docker config disable-default-users
```


### solace-util docker config disable-default-vpn

Shut down the default message-VPN

```
solace-util docker config disable-default-vpn
```


### solace-util docker config domain-certs

Load domain CA certificates

```
solace-util docker config domain-certs
```


### solace-util docker config exec-cli

Run a Solace CLI script inside the container (menu if no file given)

```
solace-util docker config exec-cli [file]
```


### solace-util docker config leader

Assert the config-sync leader on the primary (HA only)

```
solace-util docker config leader [primary|backup|monitor]
```


### solace-util docker config product-keys

Apply product keys

```
solace-util docker config product-keys
```


### solace-util docker config server-cert

Load/update the TLS server certificate

```
solace-util docker config server-cert
```


### solace-util docker copy

Copy files to/from the broker container

```
solace-util docker copy
```

Subcommands: `from`, `into`


### solace-util docker copy from

Copy files from the broker container to the host

```
solace-util docker copy from files...
```


### solace-util docker copy into

Copy files from the host into the broker container

```
solace-util docker copy into files... [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--dir` | (none) | destination directory inside the container |


### solace-util docker delete

Remove the broker container/unit (data folder kept by default)

```
solace-util docker delete [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--clear-data` | `false` | alias for --purge |
| `--keep-data` | `false` | keep persistent data and skip the confirmation prompt |
| `--purge` | `false` | clear persistent data (k8s PVCs / container data folder) |


### solace-util docker deploy

Deploy the broker on this host (role required in HA, ignored in standalone)

```
solace-util docker deploy [primary|backup|monitor] [flags]
```

Honors `--gen-only`, `--gen-secrets-only` and `--gen-env-only`: renders the requested artifact instead of applying it, and changes nothing.

| Flag | Default | Meaning |
| --- | --- | --- |
| `--restart` | `false` | restart an already-running broker when the deploy artifact changed (otherwise you are asked, and a non-interactive run leaves it running) |


### solace-util docker describe

Show detailed inspection of the broker container (podman: also the installed unit)

```
solace-util docker describe
```

Also available as: inspect


### solace-util docker down

Orchestrate delete (data folder kept unless --purge)

```
solace-util docker down [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--clear-data` | `false` | alias for --purge |
| `--keep-data` | `false` | keep persistent data and skip the confirmation prompt |
| `--purge` | `false` | clear persistent data (k8s PVCs / container data folder) |


### solace-util docker gen

Render the deploy artifact (quadlet/compose/run) to stdout without applying

```
solace-util docker gen [primary|backup|monitor]
```

Honors `--gen-only`, `--gen-secrets-only` and `--gen-env-only`: renders the requested artifact instead of applying it, and changes nothing.


### solace-util docker logs

Tail the local broker container logs

```
solace-util docker logs
```


### solace-util docker prep

Prepare the host (data dir + ownership, DNS, PSK generation)

```
solace-util docker prep
```

Subcommands: `host`


### solace-util docker prep host

Create/own the data dir, verify DNS, generate the redundancy PSK

```
solace-util docker prep host
```


### solace-util docker shell

Open an interactive shell in the container

```
solace-util docker shell
```


### solace-util docker status

Show the local broker container/service status

```
solace-util docker status
```


### solace-util docker teardown

Remove broker-scoped prerequisites (the container itself: see delete)

```
solace-util docker teardown
```

Subcommands: `domain-certs`


### solace-util docker teardown domain-certs

Remove domain CA certificates

```
solace-util docker teardown domain-certs
```


### solace-util docker up

Orchestrate check -> prep host -> deploy <role>

```
solace-util docker up [primary|backup|monitor] [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--restart` | `false` | restart an already-running broker when the deploy artifact changed (otherwise you are asked, and a non-interactive run leaves it running) |


### solace-util docker verify

Verify broker health (login, redundancy, diagnostics)

```
solace-util docker verify
```

Subcommands: `diagnostics`, `login`, `redundancy`


### solace-util docker verify diagnostics

Gather show-command output and a diagnostics bundle

```
solace-util docker verify diagnostics [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--days` | `1` | days of logs/diagnostics to gather |


### solace-util docker verify login

Test SEMP login

```
solace-util docker verify login
```


### solace-util docker verify redundancy

Exercise failover on this node (HA only; run on primary and backup)

```
solace-util docker verify redundancy [primary|backup|monitor]
```


### solace-util k8s

Deploy/operate the broker on Kubernetes via the EventBroker Operator

```
solace-util k8s
```

Subcommands: `check`, `cli`, `config`, `copy`, `delete`, `deploy`, `describe`, `down`, `gen`, `logs`, `operator`, `prep`, `replicas`, `restart`, `shell`, `show-all`, `status`, `teardown`, `up`, `verify`

| Flag | Default | Meaning |
| --- | --- | --- |
| `--allow-command` | `[]` | approve one extra binary for the config's platform command, for this run only (repeatable; a bare name, never a path). The env file cannot grant this |


### solace-util k8s check

Validate config, cluster reachability, and StorageClass

```
solace-util k8s check
```


### solace-util k8s cli

Open an interactive Solace CLI in a pod

```
solace-util k8s cli [role]
```


### solace-util k8s config

Configure a DEPLOYED broker (certs, hardening, product keys, CLI)

Post-deployment configuration: every step here talks to a broker that is already deployed and running, over the Solace CLI. Nothing under `config` is part of `deploy`, and none of it is applied by `up` -- run it after the pods are ready.

With no subcommand, runs all applicable config steps in order (HA-only steps skipped in standalone).

```
solace-util k8s config
```

Subcommands: `additional-users`, `disable-default-users`, `disable-default-vpn`, `domain-certs`, `exec-cli`, `leader`, `product-keys`, `server-cert`


### solace-util k8s config additional-users

Create the admin.additionalUsers CLI users

```
solace-util k8s config additional-users
```


### solace-util k8s config disable-default-users

Shut down default client-usernames in all VPNs

```
solace-util k8s config disable-default-users
```


### solace-util k8s config disable-default-vpn

Shut down the default message-VPN

```
solace-util k8s config disable-default-vpn
```


### solace-util k8s config domain-certs

Load domain CA certificates

```
solace-util k8s config domain-certs
```


### solace-util k8s config exec-cli

Run a Solace CLI script inside a pod

```
solace-util k8s config exec-cli [file] [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--pod` | (none) | pod role to target (p\|b\|m) |


### solace-util k8s config leader

Assert the config-sync leader (HA only)

```
solace-util k8s config leader
```


### solace-util k8s config product-keys

Apply product keys

```
solace-util k8s config product-keys
```


### solace-util k8s config server-cert

Load/update the TLS server certificate

```
solace-util k8s config server-cert
```


### solace-util k8s copy

Copy files to/from a broker pod

```
solace-util k8s copy
```

Subcommands: `from`, `into`


### solace-util k8s copy from

Copy files from a broker pod to the host

```
solace-util k8s copy from files... [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--pod` | (none) | pod role to target (p\|b\|m) |


### solace-util k8s copy into

Copy files from the host into a broker pod

```
solace-util k8s copy into files... [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--dir` | (none) | destination directory inside the pod |
| `--pod` | (none) | pod role to target (p\|b\|m) |


### solace-util k8s delete

Delete the broker CR (PVCs kept by default)

```
solace-util k8s delete [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--clear-data` | `false` | alias for --purge |
| `--keep-data` | `false` | keep persistent data and skip the confirmation prompt |
| `--purge` | `false` | clear persistent data (k8s PVCs / container data folder) |


### solace-util k8s deploy

Render and apply the PubSubPlusEventBroker custom resource

```
solace-util k8s deploy [flags]
```

Honors `--gen-only`, `--gen-secrets-only` and `--gen-env-only`: renders the requested artifact instead of applying it, and changes nothing.

| Flag | Default | Meaning |
| --- | --- | --- |
| `--keep-yaml` | `false` | keep the rendered manifest on disk after applying |


### solace-util k8s describe

Describe broker/load-balancer resources

```
solace-util k8s describe
```

Subcommands: `broker`, `lb`

Also available as: inspect


### solace-util k8s describe broker

Describe a broker pod

```
solace-util k8s describe broker [role]
```


### solace-util k8s describe lb

Describe the load-balancer service

```
solace-util k8s describe lb
```


### solace-util k8s down

Orchestrate delete -> teardown secrets -> teardown namespace

```
solace-util k8s down [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--clear-data` | `false` | alias for --purge |
| `--keep-data` | `false` | keep persistent data and skip the confirmation prompt |
| `--purge` | `false` | clear persistent data (k8s PVCs / container data folder) |


### solace-util k8s gen

Render a manifest to stdout without applying (like --gen-only)

```
solace-util k8s gen [broker|operator|secrets]
```

Honors `--gen-only`, `--gen-secrets-only` and `--gen-env-only`: renders the requested artifact instead of applying it, and changes nothing.


### solace-util k8s logs

Tail broker pod logs

```
solace-util k8s logs [role]
```


### solace-util k8s operator

Manage the cluster-scoped EventBroker Operator

```
solace-util k8s operator
```

Subcommands: `delete`, `deploy`, `describe`, `logs`, `status`


### solace-util k8s operator delete

Remove the operator (embedded bundle)

```
solace-util k8s operator delete
```


### solace-util k8s operator deploy

Install the operator (embedded bundle)

```
solace-util k8s operator deploy
```

Honors `--gen-only`, `--gen-secrets-only` and `--gen-env-only`: renders the requested artifact instead of applying it, and changes nothing.


### solace-util k8s operator describe

Describe the operator deployment

```
solace-util k8s operator describe
```


### solace-util k8s operator logs

Tail operator logs

```
solace-util k8s operator logs
```


### solace-util k8s operator status

Show operator deployment/pod status

```
solace-util k8s operator status
```


### solace-util k8s prep

Prepare cluster prerequisites (operator, namespace, secrets, labels)

With no subcommand, runs all prep steps in order, skipping ones whose config is absent.

```
solace-util k8s prep
```

Subcommands: `labels`, `namespace`, `operator`, `secrets`


### solace-util k8s prep labels

Label nodes for primary/backup/monitor placement

```
solace-util k8s prep labels
```


### solace-util k8s prep namespace

Create the broker namespace

```
solace-util k8s prep namespace
```


### solace-util k8s prep operator

Install the EventBroker Operator

```
solace-util k8s prep operator
```

Honors `--gen-only`, `--gen-secrets-only` and `--gen-env-only`: renders the requested artifact instead of applying it, and changes nothing.


### solace-util k8s prep secrets

Create admin/monitor, TLS, and image-pull secrets

```
solace-util k8s prep secrets
```

Honors `--gen-only`, `--gen-secrets-only` and `--gen-env-only`: renders the requested artifact instead of applying it, and changes nothing.


### solace-util k8s replicas

Scale the broker statefulset(s)

```
solace-util k8s replicas
```

Subcommands: `start`, `stop`


### solace-util k8s replicas start

Scale broker statefulset(s) to 1

```
solace-util k8s replicas start
```


### solace-util k8s replicas stop

Scale broker statefulset(s) to 0

```
solace-util k8s replicas stop
```


### solace-util k8s restart

Delete a broker pod so the statefulset recreates it (manualPodRestart upgrades)

For k8s.updateStrategy=manualPodRestart: `deploy` updates the statefulset's pod
template but the operator waits for a pod to be deleted before applying it.
With no role, restarts every pod in the safe order (monitor, backup, primary;
standalone: just the primary), waiting for each to become ready before the next.

The order is by configured role, not by which node is currently active -- after a
failover they differ. Check `solace-util k8s verify redundancy` first, or pass a role
and restart them one at a time in the order you want.

```
solace-util k8s restart [primary|backup|monitor]
```


### solace-util k8s shell

Open an interactive shell in a pod

```
solace-util k8s shell [role]
```


### solace-util k8s show-all

List all brokers across namespaces

```
solace-util k8s show-all
```


### solace-util k8s status

Show pods, services, and statefulset for the broker

```
solace-util k8s status
```


### solace-util k8s teardown

Remove broker-scoped prerequisites (operator kept)

```
solace-util k8s teardown
```

Subcommands: `domain-certs`, `namespace`, `secrets`


### solace-util k8s teardown domain-certs

Remove domain CA certificates

```
solace-util k8s teardown domain-certs
```


### solace-util k8s teardown namespace

Delete the broker namespace

```
solace-util k8s teardown namespace
```


### solace-util k8s teardown secrets

Delete broker secrets

```
solace-util k8s teardown secrets
```


### solace-util k8s up

Orchestrate check -> prep -> deploy -> config leader (if HA)

```
solace-util k8s up
```


### solace-util k8s verify

Verify broker health: redundancy failover (HA) then a SEMP login

```
solace-util k8s verify
```

Subcommands: `diagnostics`, `login`, `redundancy`


### solace-util k8s verify diagnostics

Gather show-command output and a diagnostics bundle

```
solace-util k8s verify diagnostics [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--days` | `1` | days of logs/diagnostics to gather |


### solace-util k8s verify login

Test SEMP login

```
solace-util k8s verify login [role]
```


### solace-util k8s verify redundancy

Exercise failover (HA only)

```
solace-util k8s verify redundancy
```


### solace-util podman

Deploy/operate the broker directly on a host with Podman (systemd quadlet)

```
solace-util podman
```

Subcommands: `check`, `cli`, `config`, `copy`, `delete`, `deploy`, `describe`, `down`, `gen`, `logs`, `prep`, `shell`, `status`, `teardown`, `up`, `verify`

| Flag | Default | Meaning |
| --- | --- | --- |
| `--allow-command` | `[]` | approve one extra binary for the config's platform command, for this run only (repeatable; a bare name, never a path). The env file cannot grant this |


### solace-util podman check

Validate config, DNS, and container runtime

```
solace-util podman check
```


### solace-util podman cli

Open an interactive Solace CLI in the container

```
solace-util podman cli
```


### solace-util podman config

Configure a DEPLOYED broker (certs, hardening, product keys, CLI)

Post-deployment configuration: every step here talks to a broker that is already running in this host's container, over the Solace CLI. Nothing under `config` is part of `deploy`, and none of it is applied by `up` -- run it once the container is up.

With no subcommand, runs all applicable config steps (HA-only steps skipped in standalone), except `leader`: that one is cross-host and primary-only, so run it explicitly on the primary.

```
solace-util podman config
```

Subcommands: `disable-default-users`, `disable-default-vpn`, `domain-certs`, `exec-cli`, `leader`, `product-keys`, `server-cert`


### solace-util podman config disable-default-users

Shut down default client-usernames in all VPNs

```
solace-util podman config disable-default-users
```


### solace-util podman config disable-default-vpn

Shut down the default message-VPN

```
solace-util podman config disable-default-vpn
```


### solace-util podman config domain-certs

Load domain CA certificates

```
solace-util podman config domain-certs
```


### solace-util podman config exec-cli

Run a Solace CLI script inside the container (menu if no file given)

```
solace-util podman config exec-cli [file]
```


### solace-util podman config leader

Assert the config-sync leader on the primary (HA only)

```
solace-util podman config leader [primary|backup|monitor]
```


### solace-util podman config product-keys

Apply product keys

```
solace-util podman config product-keys
```


### solace-util podman config server-cert

Load/update the TLS server certificate

```
solace-util podman config server-cert
```


### solace-util podman copy

Copy files to/from the broker container

```
solace-util podman copy
```

Subcommands: `from`, `into`


### solace-util podman copy from

Copy files from the broker container to the host

```
solace-util podman copy from files...
```


### solace-util podman copy into

Copy files from the host into the broker container

```
solace-util podman copy into files... [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--dir` | (none) | destination directory inside the container |


### solace-util podman delete

Remove the broker container/unit (data folder kept by default)

```
solace-util podman delete [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--clear-data` | `false` | alias for --purge |
| `--keep-data` | `false` | keep persistent data and skip the confirmation prompt |
| `--purge` | `false` | clear persistent data (k8s PVCs / container data folder) |


### solace-util podman deploy

Deploy the broker on this host (role required in HA, ignored in standalone)

```
solace-util podman deploy [primary|backup|monitor] [flags]
```

Honors `--gen-only`, `--gen-secrets-only` and `--gen-env-only`: renders the requested artifact instead of applying it, and changes nothing.

| Flag | Default | Meaning |
| --- | --- | --- |
| `--restart` | `false` | restart an already-running broker when the deploy artifact changed (otherwise you are asked, and a non-interactive run leaves it running) |


### solace-util podman describe

Show detailed inspection of the broker container (podman: also the installed unit)

```
solace-util podman describe
```

Also available as: inspect


### solace-util podman down

Orchestrate delete (data folder kept unless --purge)

```
solace-util podman down [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--clear-data` | `false` | alias for --purge |
| `--keep-data` | `false` | keep persistent data and skip the confirmation prompt |
| `--purge` | `false` | clear persistent data (k8s PVCs / container data folder) |


### solace-util podman gen

Render the deploy artifact (quadlet/compose/run) to stdout without applying

```
solace-util podman gen [primary|backup|monitor]
```

Honors `--gen-only`, `--gen-secrets-only` and `--gen-env-only`: renders the requested artifact instead of applying it, and changes nothing.


### solace-util podman logs

Tail the local broker container logs

```
solace-util podman logs
```


### solace-util podman prep

Prepare the host (data dir + ownership, DNS, PSK generation)

```
solace-util podman prep
```

Subcommands: `host`


### solace-util podman prep host

Create/own the data dir, verify DNS, generate the redundancy PSK

```
solace-util podman prep host
```


### solace-util podman shell

Open an interactive shell in the container

```
solace-util podman shell
```


### solace-util podman status

Show the local broker container/service status

```
solace-util podman status
```


### solace-util podman teardown

Remove broker-scoped prerequisites (the container itself: see delete)

```
solace-util podman teardown
```

Subcommands: `domain-certs`


### solace-util podman teardown domain-certs

Remove domain CA certificates

```
solace-util podman teardown domain-certs
```


### solace-util podman up

Orchestrate check -> prep host -> deploy <role>

```
solace-util podman up [primary|backup|monitor] [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--restart` | `false` | restart an already-running broker when the deploy artifact changed (otherwise you are asked, and a non-interactive run leaves it running) |


### solace-util podman verify

Verify broker health (login, redundancy, diagnostics)

```
solace-util podman verify
```

Subcommands: `diagnostics`, `login`, `redundancy`


### solace-util podman verify diagnostics

Gather show-command output and a diagnostics bundle

```
solace-util podman verify diagnostics [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--days` | `1` | days of logs/diagnostics to gather |


### solace-util podman verify login

Test SEMP login

```
solace-util podman verify login
```


### solace-util podman verify redundancy

Exercise failover on this node (HA only; run on primary and backup)

```
solace-util podman verify redundancy [primary|backup|monitor]
```


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

