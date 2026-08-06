# solace

A single Go binary that deploys and operates Solace PubSub+ Event Brokers on
Kubernetes, Docker, or Podman. You describe the broker once in a YAML env file and drive
the whole lifecycle -- `check -> prep -> deploy -> config -> verify` and back down again
-- through one standardized command tree.

**Platform status**

| Platform | State |
| --- | --- |
| Kubernetes (via the Solace EventBroker Operator) | Fully supported |
| Docker / Podman (host-local containers, no operator) | Fully supported |

> Unsupported -- this is not a Solace product. Use at your own risk.

## Requirements

- **Build:** Go 1.26+.
- **Run (Kubernetes):** `kubectl` on your `PATH` and a reachable cluster/context. The
  binary shells out to `kubectl`; it does not embed a Kubernetes client.
- **Run (Docker/Podman):** the `docker` or `podman` binary on your `PATH`, on the host that
  runs the broker. Podman deploys a systemd **quadlet** unit (its host also needs systemd);
  Docker uses `compose` (default) or `run`. The binary shells out to the runtime; it embeds
  no container client.
- `--dry-run` needs no cluster, runtime, or `kubectl`/`docker`/`podman` binary -- it prints
  the commands instead of running them.

## Build

Quick local build:

```
go build -o solace .
```

Release binaries (cross-compiled, stripped) via the dev scripts:

```
scripts/dev.ps1 dist      # Windows
./scripts/dev.sh dist     # Linux/macOS
```

This emits `dist/solace-<os>-<arch>` for linux/amd64, linux/arm64, darwin/arm64,
and windows/amd64 (Windows gets a `.exe` suffix), built with
`CGO_ENABLED=0 -trimpath -ldflags '-s -w'`. `scripts/dev.sh build` compiles a
single target instead -- the host's by default, or `TARGET_OS`/`TARGET_ARCH`
when set, which is how the release pipeline drives it.

## Quick start (Kubernetes)

1. Copy the annotated sample to your own env file and edit it:

   ```
   cp env/sample.yaml env/dev.yaml
   ```

   At minimum set `image.repo`, `image.tag`, `admin.pass`, `k8s.name`,
   `k8s.namespace`, and `k8s.storage.msgNode`.

2. Dry-run first to see exactly what would happen (no cluster needed):

   ```
   solace k8s check --env dev --dry-run
   ```

3. Bring the broker up (check -> prep -> deploy -> config leader if HA):

   ```
   solace k8s up --env dev
   ```

4. Verify and inspect:

   ```
   solace k8s verify --env dev
   solace k8s status --env dev
   ```

5. Tear it down (persistent volumes are **kept** by default):

   ```
   solace k8s down --env dev            # keeps PVCs
   solace k8s down --env dev --purge    # also clears PVCs (irreversible)
   ```

## Configuration (`--env`)

Every command reads one YAML env file, selected with `--env`:

- A **bare name** resolves to `<base-dir>/env/<name>.yaml` -- e.g. `--env dev` loads
  `env/dev.yaml`. `--base-dir` sets the directory containing `env/` (default: current
  directory).
- A value containing a path separator or ending in `.yaml`/`.yml` is used **as a path**
  as-is -- e.g. `--env ./configs/prod.yaml`.
- The default name is `default` (`env/default.yaml`). That file is not shipped; copy
  `env/sample.yaml` to create your own.

Decoding is **strict**: an unknown or misspelled key is a hard error, so typos fail loud
instead of being silently ignored.

**`env/sample.yaml` is the authoritative, fully annotated schema** -- start there rather
than from this README. The most-used keys:

Minimum required (Kubernetes):

| Key | Purpose |
| --- | --- |
| `image.repo` | Broker image repository |
| `image.tag` | Image tag |
| `admin.pass` | Broker admin password (never defaulted) |
| `k8s.name` | Broker / custom-resource name |
| `k8s.namespace` | Target namespace |
| `k8s.storage.msgNode` | Message-node PVC size (e.g. `30Gi`) |

Common optional knobs:

| Key | Default | Purpose |
| --- | --- | --- |
| `redundancy` | `yes` | `yes` = HA group (primary+backup+monitor); `no` = single standalone broker |
| `image.registry` | docker.io | Registry prefix for the image reference |
| `k8s.storage.class` | cluster default | StorageClass for the broker PVCs |
| `k8s.updateStrategy` | `automatedRolling` | `automatedRolling` or `manualPodRestart` |
| `tls.serverSecret` | -- | Name of the TLS secret; its presence enables the CR's TLS block |

**Secrets** belong in the env file (or your own secret store). The tool never echoes
them: under `--dry-run`, values piped to a command on stdin are shown as
`<<< (N bytes on stdin)`, never as their contents.

## Global flags

These apply to every subcommand:

| Flag | Default | Meaning |
| --- | --- | --- |
| `--env <name\|path>` | `default` | Env file to load (`env/<name>.yaml`, or a path). |
| `--base-dir <dir>` | current dir | Directory containing `env/`. |
| `--gen` | `false` | Render the artifact this command would apply and print it; change nothing. |
| `--dry-run` | `false` | Print the external commands instead of running them. |
| `-y`, `--yes` | `false` | Skip confirmation prompts. Does **not** imply `--purge`. |

## Command reference (Kubernetes)

Run `solace k8s --help` (or `--help` on any subcommand) for the live tree. A `[role]`
positional accepts `p`|`b`|`m` or `primary`|`backup`|`monitor` and defaults to primary.

### Lifecycle

| Command | Description |
| --- | --- |
| `check` | Validate config, cluster reachability, and StorageClass. |
| `prep` | Prepare cluster prerequisites; with no subcommand runs all applicable steps in order. |
| `prep operator` | Install the EventBroker Operator (honors `--gen`). |
| `prep namespace` | Create the broker namespace. |
| `prep secrets` | Create admin/monitor, TLS, and image-pull secrets. |
| `prep labels` | Label nodes for primary/backup/monitor placement (interactive). |
| `deploy` | Render and apply the PubSubPlusEventBroker custom resource (honors `--gen`). `--keep-yaml` keeps the rendered manifest on disk. |
| `config` | Post-deploy configuration; with no subcommand runs all applicable steps. |
| `config leader` | Assert the config-sync leader (HA only). |
| `config server-cert` | Load/update the TLS server certificate. |
| `config domain-certs` | Load domain CA certificates. |
| `config disable-default-vpn` | Shut down the default message-VPN. |
| `config disable-default-users` | Shut down default client-usernames in all VPNs. |
| `config product-keys` | Apply product keys. |
| `config exec-cli [file]` | Run a Solace CLI script inside a pod. `--pod <p\|b\|m>` targets a role; `[file]` is required. |
| `verify` | Verify broker health; with no subcommand runs redundancy (HA) then a SEMP login. |
| `verify login [role]` | Test SEMP login. |
| `verify redundancy` | Exercise failover (HA only). |
| `verify diagnostics` | Gather show-command output and a diagnostics bundle. `--days <n>` sets the window (default 1). |

### Day-2 operations

| Command | Description |
| --- | --- |
| `status` | Show pods, services, and statefulset for the broker. |
| `logs [role]` | Tail broker pod logs. |
| `cli [role]` | Open an interactive Solace CLI in a pod. |
| `shell [role]` | Open an interactive shell in a pod. |
| `describe broker [role]` | Describe a broker pod. |
| `describe lb` | Describe the load-balancer service. |
| `copy from files...` | Copy files from a broker pod to the host. `--pod <role>`. |
| `copy into files...` | Copy files from the host into a broker pod. `--pod <role>`, `--dir <dest>`. |
| `replicas start` | Scale broker statefulset(s) to 1. |
| `replicas stop` | Scale broker statefulset(s) to 0. |
| `operator deploy` | Install the operator from the embedded bundle (honors `--gen`). |
| `operator delete` | Remove the operator. |
| `operator status` | Show operator deployment/pod status. |
| `operator logs` | Tail operator logs. |
| `operator describe` | Describe the operator deployment. |
| `gen [broker\|operator]` | Render a manifest to stdout without applying (default `broker`). |
| `show-all` | List all brokers across namespaces. |

### Teardown

| Command | Description |
| --- | --- |
| `delete` | Delete the broker CR (PVCs kept by default; supports the data-retention flags below). |
| `teardown secrets` | Delete broker secrets. |
| `teardown namespace` | Delete the broker namespace. |
| `teardown domain-certs` | Remove domain CA certificates. |

### Orchestration

| Command | Description |
| --- | --- |
| `up` | Orchestrate check -> prep -> deploy -> config leader (if HA). |
| `down` | Orchestrate delete -> teardown secrets -> teardown namespace. Leaves the operator in place. Supports the data-retention flags. |

## Data retention and safety

`delete` and `down` keep persistent data by default. Two independent decisions govern a
destructive run:

- **Whether to delete** -- `--yes` proceeds without a prompt; an interactive terminal is
  asked `[y/N]`; a non-interactive session without `--yes` refuses loudly rather than
  deleting unattended.
- **Whether to clear data (PVCs)** -- kept unless you opt in. `--purge` (alias
  `--clear-data`) clears it; `--keep-data` keeps it and skips the prompt; a
  non-interactive session keeps it; an interactive session clears only if you type an
  exact `yes`.

`--purge` and `--keep-data` are mutually exclusive (passing both is a parse error), and
**`--yes` never implies `--purge`** -- clearing data is always its own explicit choice.

## Rendering without applying

To review the exact manifest before it touches a cluster, use `--gen` on an
artifact-producing command, or the dedicated `gen` command. Both print to stdout and
change nothing:

```
solace k8s gen broker --env dev            # the PubSubPlusEventBroker CR
solace k8s gen operator --env dev          # the operator bundle
solace k8s deploy --env dev --gen          # equivalent to gen broker
solace k8s operator deploy --env dev --gen # equivalent to gen operator
```

`--gen` is only valid on artifact commands (`deploy`, `prep operator`,
`operator deploy`, `gen`); using it elsewhere is rejected with a clear error.

## Command reference (Docker / Podman)

The `docker` and `podman` command trees mirror the Kubernetes verbs for a **host-local**
broker: one container per host, driven over `<runtime> exec`/`cp` (no operator, no
cluster). The two share one tree; only the deploy artifact differs -- Docker renders a
compose file (`docker.mode: compose`, the default) or a `run` command line
(`docker.mode: run`), Podman a systemd **quadlet** `.container` unit. Run
`solace docker --help` / `solace podman --help` for the live tree.

A `[primary|backup|monitor]` positional (also `p`|`b`|`m`) selects the redundancy role and
defaults to primary. In standalone mode (`redundancy: no`) it is ignored; in an HA group
pass it explicitly per host on `deploy`/`up`/`gen`. For `config leader` and
`verify redundancy` it is optional -- omitted, the role is auto-detected by matching the
host name against the `nodes.*` table.

### Lifecycle

| Command | Description |
| --- | --- |
| `check` | Validate config, node-name DNS resolution, and the container runtime. |
| `prep` | Prepare the host; with no subcommand runs all steps. |
| `prep host` | Create/own the data dir, verify DNS, and (HA) generate the redundancy PSK. |
| `deploy [role]` | Render the deploy artifact and start the container/service (honors `--gen`). |
| `config` | Post-deploy configuration; with no subcommand runs all applicable steps **except** `leader`. |
| `config leader [role]` | Assert the config-sync leader (HA only; primary-only -- fails loud on backup/monitor). |
| `config server-cert` | Load/update the TLS server certificate. |
| `config domain-certs` | Load domain CA certificates. |
| `config disable-default-vpn` | Shut down the default message-VPN. |
| `config disable-default-users` | Shut down default client-usernames in all VPNs. |
| `config product-keys` | Apply product keys. |
| `config exec-cli [file]` | Run a Solace CLI script inside the container; `[file]` is required. |
| `verify` | Verify health; with no subcommand runs redundancy (HA; skipped on the monitor) then a SEMP login. |
| `verify login` | Test SEMP login. |
| `verify redundancy [role]` | Exercise failover on this host (HA only; run on primary and backup). |
| `verify diagnostics` | Gather show-command output and a diagnostics bundle. `--days <n>` sets the window (default 1). |

### Day-2 operations

| Command | Description |
| --- | --- |
| `status` | Show the local broker container/service status. |
| `logs` | Tail the local broker container logs. |
| `cli` | Open an interactive Solace CLI in the container. |
| `shell` | Open an interactive shell in the container. |
| `gen [role]` | Render the deploy artifact to stdout without applying (compose/`run` line for Docker, quadlet for Podman). |

### Teardown and orchestration

| Command | Description |
| --- | --- |
| `delete` | Stop and remove the container/unit (data folder kept by default; supports the data-retention flags). |
| `up [role]` | Orchestrate check -> prep host -> deploy `<role>`. |
| `down` | Delete the container/unit (data kept unless `--purge`). There is no layer above the broker, so `down` == `delete`. |

**HA is a two-host handshake.** The transport is node-local, so one invocation drives only
this host's broker. Bring the group up by running `up <role>` on each host with its own
role, then run `verify redundancy` on the **primary and backup concurrently**: the primary
releases activity and waits to fail back; the backup takes over, dwells ~10s, and reverts.
The monitor cannot run `verify redundancy` (rejected loud), and `config leader` runs only
on the primary. Running `verify redundancy` on a single host times out (bounded by the poll
budget) rather than hanging.

**Config source.** The container platform has no separate config namespace: its post-deploy
`config`/`verify` steps read the shared `k8s.*` fields -- `k8s.domainCerts`,
`k8s.productKeys`, `k8s.diagDir`, `k8s.cliScriptsFolder` -- plus `tls.cert`/`tls.certKey`
(server certificate) and `admin.user`/`admin.pass` (SEMP login); the `nodes.*` names drive
role detection for `config leader` / `verify redundancy`. The rest of the `nodes.*` table
and `nodes.psk` are consumed earlier, at `prep`/`deploy` (`prep host` generates the PSK and
bakes it into the deploy artifact). Container-only knobs live under `docker.*` / `podman.*`
(runtime, container name, data dir, network mode, compose mode, rootless). See
`env/sample.yaml`.

Example (HA -- run each line on the matching host):

```
solace podman up primary --env prod       # on the primary host
solace podman up backup  --env prod       # on the backup host
solace podman up monitor --env prod       # on the monitor host
solace podman config leader --env prod    # on the primary only
# then, concurrently on the primary and backup hosts:
solace podman verify redundancy --env prod
```

## Development

`scripts/dev.ps1` and `scripts/dev.sh` are behaviourally identical and own every
build/test/scan command. The workflows call task names only, so local runs match CI:

| Task | Does |
| --- | --- |
| `tidy` | `go mod tidy` |
| `vet` | `go vet ./...` |
| `build` | Compile -> `dist/solace-<os>-<arch>[.exe]`. `TARGET_OS`/`TARGET_ARCH` pick the target; unset means the host |
| `test` | `go test -count=1 ./...` (race on by default on `dev.sh`; opt-in on `dev.ps1`) |
| `cov` | Coverage profile -> `coverage/coverage.html` + `.out`, prints the total |
| `scan` | `govulncheck` -- **fatal** on findings, standalone or inside an aggregate |
| `dist` | Local convenience: cross-compile all four release targets into `dist/` |
| `graphify` | Refresh `graphify-out/`. Local only; skipped when `CI` is set |
| `all` | `build vet test` -- the fast inner loop; CI runs `all scan` |
| `full` | `all` + `cov scan graphify` -- the pre-tag sweep |

Run the local gate with `scripts/dev.ps1 all scan` (or `./scripts/dev.sh all scan`), and
`full` before tagging. Per-task logs land in `scripts/logs/<task>.log`, each closing with a
`<timestamp> | <task> | <duration>s | OK|FAILED` footer; coverage HTML in
`coverage/coverage.html`. Current test coverage is 89.1% (recorded in
`scripts/logs/cov.log`; the previous total is the local floor, not an enforced numeric gate).

### Releases

[.github/workflows/tag.yml](.github/workflows/tag.yml) is the only automatic pipeline:
pushing a `v*` tag runs the gates on Ubuntu and Windows, cross-compiles the targets in the
`BUILD_TARGETS` repo variable, and creates a GitHub release with the binaries and
`SHA256SUMS.txt` -- only if everything passed. [.github/workflows/ci.yml](.github/workflows/ci.yml)
runs the gates and is reused by the tag pipeline; it does **not** run on ordinary pushes or
PRs (only on `workflow_dispatch` and PRs that touch the workflows or the dev scripts), so
`full` on a clean checkout before tagging is what catches a file you never committed.

```
./scripts/dev.sh full          # clean checkout, everything green
git tag v0.1.0 && git push origin v0.1.0
```

## Repository layout

| Path | Contents |
| --- | --- |
| `main.go` | Entry point (`cli.Execute()`). |
| `internal/config` | Env-file schema, loading, defaults, validation, role helpers. |
| `internal/engine` | External-command runner (real exec; `--dry-run` echo; secrets never echoed). |
| `internal/render` | Templating for the broker CR, operator bundle, compose/run/quadlet artifacts. |
| `internal/broker` | Platform-agnostic config/verify operations over an injected transport. |
| `internal/k8s` | Kubernetes cluster/operator operations and the kubectl transport. |
| `internal/container` | Docker/Podman host operations (Manager) and the node-local `<runtime> exec`/`cp` transport. |
| `internal/cli` | Cobra command tree and handlers. |
| `env/` | Config files (`sample.yaml` is the annotated template). |
| `scripts/` | `dev.ps1` / `dev.sh` developer tooling. |
