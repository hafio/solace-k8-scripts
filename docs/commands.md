# Command reference

Every command `solace` exposes, with its arguments and flags.

**Generated from the command tree -- do not edit by hand.** Regenerate after any
command, flag, or description change:

```
go test ./internal/cli -update
```

The `test` task fails while this file is stale, so it cannot drift from the code.

## Tree

```
solace
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
    delete
    deploy [primary|backup|monitor]
    down
    gen [primary|backup|monitor]
    logs
    prep
      host
    shell
    status
    up [primary|backup|monitor]
    verify
      diagnostics
      login
      redundancy [primary|backup|monitor]
  k8s
    check
    cli [role]
    config
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
    gen [broker|operator]
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
    delete
    deploy [primary|backup|monitor]
    down
    gen [primary|backup|monitor]
    logs
    prep
      host
    shell
    status
    up [primary|backup|monitor]
    verify
      diagnostics
      login
      redundancy [primary|backup|monitor]
```

## Global flags

Inherited by every command.

| Flag | Default | Meaning |
| --- | --- | --- |
| `--base-dir` | (none) | directory searched for the env file, and holding env/ (default: current directory) |
| `--dry-run` | `false` | print the external commands instead of running them |
| `-e`, `--env` | `env.yaml` | env file name, searched in the base dir then &lt;base-dir&gt;/env; a value with a directory is used as-is |
| `--gen` | `false` | render the artifact this command would apply and print it; change nothing |
| `-y`, `--yes` | `false` | skip confirmation prompts (does NOT imply --purge) |

## Commands

### solace

Deploy and operate Solace PubSub+ brokers on Kubernetes, Docker, or Podman

solace is a single CLI for deploying and operating Solace PubSub+ Event Brokers.
It presents the same lifecycle verbs on every platform:

  check -> prep -> deploy -> config -> verify   (up)
  delete -> teardown                            (down)

Pick a platform (k8s, docker, podman), then a verb. Every command takes
-e/--env <file>, searched in the current directory then ./env.
Coming from the bash scripts? 'solace convert <bash-env-file>' turns an old
env file into the YAML this reads. See 'solace <platform> --help'.

```
solace
```

Subcommands: `convert`, `docker`, `k8s`, `podman`


### solace convert

Convert a legacy bash env file into a YAML env file

Convert a legacy bash env file -- the pre-Go format sourced by bash/000-env.sh --
into the YAML env file this CLI reads.

The target platform section is detected from the variables present; pass
--platform to choose it yourself. Variables with no YAML equivalent are
reported on stderr rather than dropped silently.

The output carries every secret from the source file verbatim, so treat it
like the source: write it with -o rather than through a shared terminal, and
never commit it.

  solace convert bash/env/prod -o prod.yaml
  solace convert bash/env/prod --platform podman -o prod.yaml
  solace k8s check -e prod.yaml --dry-run

```
solace convert <bash-env-file> [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--force` | `false` | overwrite the --out file if it already exists |
| `-o`, `--out` | (none) | write the YAML here instead of stdout |
| `--platform` | (none) | platform section to write: k8s, docker, or podman (default: detect) |


### solace docker

Deploy/operate the broker directly on a host with Docker

```
solace docker
```

Subcommands: `check`, `cli`, `config`, `delete`, `deploy`, `down`, `gen`, `logs`, `prep`, `shell`, `status`, `up`, `verify`


### solace docker check

Validate config, DNS, and container runtime

```
solace docker check
```


### solace docker cli

Open an interactive Solace CLI in the container

```
solace docker cli
```


### solace docker config

Post-deploy configuration (certs, hardening, product keys, CLI)

With no subcommand, runs all applicable config steps (HA-only steps skipped in standalone).

```
solace docker config
```

Subcommands: `disable-default-users`, `disable-default-vpn`, `domain-certs`, `exec-cli`, `leader`, `product-keys`, `server-cert`


### solace docker config disable-default-users

Shut down default client-usernames in all VPNs

```
solace docker config disable-default-users
```


### solace docker config disable-default-vpn

Shut down the default message-VPN

```
solace docker config disable-default-vpn
```


### solace docker config domain-certs

Load domain CA certificates

```
solace docker config domain-certs
```


### solace docker config exec-cli

Run a Solace CLI script inside the container (menu if no file given)

```
solace docker config exec-cli [file]
```


### solace docker config leader

Assert the config-sync leader on the primary (HA only)

```
solace docker config leader [primary|backup|monitor]
```


### solace docker config product-keys

Apply product keys

```
solace docker config product-keys
```


### solace docker config server-cert

Load/update the TLS server certificate

```
solace docker config server-cert
```


### solace docker delete

Remove the broker container/unit (data folder kept by default)

```
solace docker delete [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--clear-data` | `false` | alias for --purge |
| `--keep-data` | `false` | keep persistent data and skip the confirmation prompt |
| `--purge` | `false` | clear persistent data (k8s PVCs / container data folder) |


### solace docker deploy

Deploy the broker on this host (role required in HA, ignored in standalone)

```
solace docker deploy [primary|backup|monitor]
```

Honors `--gen`: renders the artifact this command would apply, and changes nothing.


### solace docker down

Orchestrate delete (data folder kept unless --purge)

```
solace docker down [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--clear-data` | `false` | alias for --purge |
| `--keep-data` | `false` | keep persistent data and skip the confirmation prompt |
| `--purge` | `false` | clear persistent data (k8s PVCs / container data folder) |


### solace docker gen

Render the deploy artifact (quadlet/compose/run) to stdout without applying

```
solace docker gen [primary|backup|monitor]
```

Honors `--gen`: renders the artifact this command would apply, and changes nothing.


### solace docker logs

Tail the local broker container logs

```
solace docker logs
```


### solace docker prep

Prepare the host (data dir + ownership, DNS, PSK generation)

```
solace docker prep
```

Subcommands: `host`


### solace docker prep host

Create/own the data dir, verify DNS, generate the redundancy PSK

```
solace docker prep host
```


### solace docker shell

Open an interactive shell in the container

```
solace docker shell
```


### solace docker status

Show the local broker container/service status

```
solace docker status
```


### solace docker up

Orchestrate check -> prep host -> deploy <role>

```
solace docker up [primary|backup|monitor]
```


### solace docker verify

Verify broker health (login, redundancy, diagnostics)

```
solace docker verify
```

Subcommands: `diagnostics`, `login`, `redundancy`


### solace docker verify diagnostics

Gather show-command output and a diagnostics bundle

```
solace docker verify diagnostics [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--days` | `1` | days of logs/diagnostics to gather |


### solace docker verify login

Test SEMP login

```
solace docker verify login
```


### solace docker verify redundancy

Exercise failover on this node (HA only; run on primary and backup)

```
solace docker verify redundancy [primary|backup|monitor]
```


### solace k8s

Deploy/operate the broker on Kubernetes via the EventBroker Operator

```
solace k8s
```

Subcommands: `check`, `cli`, `config`, `copy`, `delete`, `deploy`, `describe`, `down`, `gen`, `logs`, `operator`, `prep`, `replicas`, `shell`, `show-all`, `status`, `teardown`, `up`, `verify`


### solace k8s check

Validate config, cluster reachability, and StorageClass

```
solace k8s check
```


### solace k8s cli

Open an interactive Solace CLI in a pod

```
solace k8s cli [role]
```


### solace k8s config

Post-deploy configuration (certs, hardening, product keys, CLI)

With no subcommand, runs all applicable config steps in order (HA-only steps skipped in standalone).

```
solace k8s config
```

Subcommands: `disable-default-users`, `disable-default-vpn`, `domain-certs`, `exec-cli`, `leader`, `product-keys`, `server-cert`


### solace k8s config disable-default-users

Shut down default client-usernames in all VPNs

```
solace k8s config disable-default-users
```


### solace k8s config disable-default-vpn

Shut down the default message-VPN

```
solace k8s config disable-default-vpn
```


### solace k8s config domain-certs

Load domain CA certificates

```
solace k8s config domain-certs
```


### solace k8s config exec-cli

Run a Solace CLI script inside a pod

```
solace k8s config exec-cli [file] [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--pod` | (none) | pod role to target (p\|b\|m) |


### solace k8s config leader

Assert the config-sync leader (HA only)

```
solace k8s config leader
```


### solace k8s config product-keys

Apply product keys

```
solace k8s config product-keys
```


### solace k8s config server-cert

Load/update the TLS server certificate

```
solace k8s config server-cert
```


### solace k8s copy

Copy files to/from a broker pod

```
solace k8s copy
```

Subcommands: `from`, `into`


### solace k8s copy from

Copy files from a broker pod to the host

```
solace k8s copy from files... [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--pod` | (none) | pod role to target (p\|b\|m) |


### solace k8s copy into

Copy files from the host into a broker pod

```
solace k8s copy into files... [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--dir` | (none) | destination directory inside the pod |
| `--pod` | (none) | pod role to target (p\|b\|m) |


### solace k8s delete

Delete the broker CR (PVCs kept by default)

```
solace k8s delete [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--clear-data` | `false` | alias for --purge |
| `--keep-data` | `false` | keep persistent data and skip the confirmation prompt |
| `--purge` | `false` | clear persistent data (k8s PVCs / container data folder) |


### solace k8s deploy

Render and apply the PubSubPlusEventBroker custom resource

```
solace k8s deploy [flags]
```

Honors `--gen`: renders the artifact this command would apply, and changes nothing.

| Flag | Default | Meaning |
| --- | --- | --- |
| `--keep-yaml` | `false` | keep the rendered manifest on disk after applying |


### solace k8s describe

Describe broker/load-balancer resources

```
solace k8s describe
```

Subcommands: `broker`, `lb`


### solace k8s describe broker

Describe a broker pod

```
solace k8s describe broker [role]
```


### solace k8s describe lb

Describe the load-balancer service

```
solace k8s describe lb
```


### solace k8s down

Orchestrate delete -> teardown secrets -> teardown namespace

```
solace k8s down [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--clear-data` | `false` | alias for --purge |
| `--keep-data` | `false` | keep persistent data and skip the confirmation prompt |
| `--purge` | `false` | clear persistent data (k8s PVCs / container data folder) |


### solace k8s gen

Render a manifest to stdout without applying (like --gen)

```
solace k8s gen [broker|operator]
```

Honors `--gen`: renders the artifact this command would apply, and changes nothing.


### solace k8s logs

Tail broker pod logs

```
solace k8s logs [role]
```


### solace k8s operator

Manage the cluster-scoped EventBroker Operator

```
solace k8s operator
```

Subcommands: `delete`, `deploy`, `describe`, `logs`, `status`


### solace k8s operator delete

Remove the operator (embedded bundle)

```
solace k8s operator delete
```


### solace k8s operator deploy

Install the operator (embedded bundle)

```
solace k8s operator deploy
```

Honors `--gen`: renders the artifact this command would apply, and changes nothing.


### solace k8s operator describe

Describe the operator deployment

```
solace k8s operator describe
```


### solace k8s operator logs

Tail operator logs

```
solace k8s operator logs
```


### solace k8s operator status

Show operator deployment/pod status

```
solace k8s operator status
```


### solace k8s prep

Prepare cluster prerequisites (operator, namespace, secrets, labels)

With no subcommand, runs all prep steps in order, skipping ones whose config is absent.

```
solace k8s prep
```

Subcommands: `labels`, `namespace`, `operator`, `secrets`


### solace k8s prep labels

Label nodes for primary/backup/monitor placement

```
solace k8s prep labels
```


### solace k8s prep namespace

Create the broker namespace

```
solace k8s prep namespace
```


### solace k8s prep operator

Install the EventBroker Operator

```
solace k8s prep operator
```

Honors `--gen`: renders the artifact this command would apply, and changes nothing.


### solace k8s prep secrets

Create admin/monitor, TLS, and image-pull secrets

```
solace k8s prep secrets
```


### solace k8s replicas

Scale the broker statefulset(s)

```
solace k8s replicas
```

Subcommands: `start`, `stop`


### solace k8s replicas start

Scale broker statefulset(s) to 1

```
solace k8s replicas start
```


### solace k8s replicas stop

Scale broker statefulset(s) to 0

```
solace k8s replicas stop
```


### solace k8s shell

Open an interactive shell in a pod

```
solace k8s shell [role]
```


### solace k8s show-all

List all brokers across namespaces

```
solace k8s show-all
```


### solace k8s status

Show pods, services, and statefulset for the broker

```
solace k8s status
```


### solace k8s teardown

Remove broker-scoped prerequisites (operator kept)

```
solace k8s teardown
```

Subcommands: `domain-certs`, `namespace`, `secrets`


### solace k8s teardown domain-certs

Remove domain CA certificates

```
solace k8s teardown domain-certs
```


### solace k8s teardown namespace

Delete the broker namespace

```
solace k8s teardown namespace
```


### solace k8s teardown secrets

Delete broker secrets

```
solace k8s teardown secrets
```


### solace k8s up

Orchestrate check -> prep -> deploy -> config leader (if HA)

```
solace k8s up
```


### solace k8s verify

Verify broker health: redundancy failover (HA) then a SEMP login

```
solace k8s verify
```

Subcommands: `diagnostics`, `login`, `redundancy`


### solace k8s verify diagnostics

Gather show-command output and a diagnostics bundle

```
solace k8s verify diagnostics [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--days` | `1` | days of logs/diagnostics to gather |


### solace k8s verify login

Test SEMP login

```
solace k8s verify login [role]
```


### solace k8s verify redundancy

Exercise failover (HA only)

```
solace k8s verify redundancy
```


### solace podman

Deploy/operate the broker directly on a host with Podman (systemd quadlet)

```
solace podman
```

Subcommands: `check`, `cli`, `config`, `delete`, `deploy`, `down`, `gen`, `logs`, `prep`, `shell`, `status`, `up`, `verify`


### solace podman check

Validate config, DNS, and container runtime

```
solace podman check
```


### solace podman cli

Open an interactive Solace CLI in the container

```
solace podman cli
```


### solace podman config

Post-deploy configuration (certs, hardening, product keys, CLI)

With no subcommand, runs all applicable config steps (HA-only steps skipped in standalone).

```
solace podman config
```

Subcommands: `disable-default-users`, `disable-default-vpn`, `domain-certs`, `exec-cli`, `leader`, `product-keys`, `server-cert`


### solace podman config disable-default-users

Shut down default client-usernames in all VPNs

```
solace podman config disable-default-users
```


### solace podman config disable-default-vpn

Shut down the default message-VPN

```
solace podman config disable-default-vpn
```


### solace podman config domain-certs

Load domain CA certificates

```
solace podman config domain-certs
```


### solace podman config exec-cli

Run a Solace CLI script inside the container (menu if no file given)

```
solace podman config exec-cli [file]
```


### solace podman config leader

Assert the config-sync leader on the primary (HA only)

```
solace podman config leader [primary|backup|monitor]
```


### solace podman config product-keys

Apply product keys

```
solace podman config product-keys
```


### solace podman config server-cert

Load/update the TLS server certificate

```
solace podman config server-cert
```


### solace podman delete

Remove the broker container/unit (data folder kept by default)

```
solace podman delete [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--clear-data` | `false` | alias for --purge |
| `--keep-data` | `false` | keep persistent data and skip the confirmation prompt |
| `--purge` | `false` | clear persistent data (k8s PVCs / container data folder) |


### solace podman deploy

Deploy the broker on this host (role required in HA, ignored in standalone)

```
solace podman deploy [primary|backup|monitor]
```

Honors `--gen`: renders the artifact this command would apply, and changes nothing.


### solace podman down

Orchestrate delete (data folder kept unless --purge)

```
solace podman down [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--clear-data` | `false` | alias for --purge |
| `--keep-data` | `false` | keep persistent data and skip the confirmation prompt |
| `--purge` | `false` | clear persistent data (k8s PVCs / container data folder) |


### solace podman gen

Render the deploy artifact (quadlet/compose/run) to stdout without applying

```
solace podman gen [primary|backup|monitor]
```

Honors `--gen`: renders the artifact this command would apply, and changes nothing.


### solace podman logs

Tail the local broker container logs

```
solace podman logs
```


### solace podman prep

Prepare the host (data dir + ownership, DNS, PSK generation)

```
solace podman prep
```

Subcommands: `host`


### solace podman prep host

Create/own the data dir, verify DNS, generate the redundancy PSK

```
solace podman prep host
```


### solace podman shell

Open an interactive shell in the container

```
solace podman shell
```


### solace podman status

Show the local broker container/service status

```
solace podman status
```


### solace podman up

Orchestrate check -> prep host -> deploy <role>

```
solace podman up [primary|backup|monitor]
```


### solace podman verify

Verify broker health (login, redundancy, diagnostics)

```
solace podman verify
```

Subcommands: `diagnostics`, `login`, `redundancy`


### solace podman verify diagnostics

Gather show-command output and a diagnostics bundle

```
solace podman verify diagnostics [flags]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--days` | `1` | days of logs/diagnostics to gather |


### solace podman verify login

Test SEMP login

```
solace podman verify login
```


### solace podman verify redundancy

Exercise failover on this node (HA only; run on primary and backup)

```
solace podman verify redundancy [primary|backup|monitor]
```

