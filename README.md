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
  binary shells out to `kubectl`; it does not embed a Kubernetes client. Set `k8s.runtime`
  to use something else -- `oc`, `microk8s kubectl`, or a whole profile such as
  `kubectl --kubeconfig /path/.kubeconfig-dev`.
- **Run (Docker/Podman):** the `docker` or `podman` binary on your `PATH`, on the host that
  runs the broker. Podman deploys a systemd **quadlet** unit (its host also needs systemd);
  Docker uses `compose` (default) or `run`. The binary shells out to the runtime; it embeds
  no container client. `docker.runtime` / `podman.runtime` override the command the same way.
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

2. Dry-run first to see exactly what would happen (no cluster needed). `-e` takes the
   file name; it is found under `env/` because there is no `./dev.yaml`:

   ```
   solace k8s check -e dev.yaml --dry-run
   ```

3. Bring the broker up (check -> prep -> deploy -> config leader if HA):

   ```
   solace k8s up -e dev.yaml
   ```

4. Verify and inspect:

   ```
   solace k8s verify -e dev.yaml
   solace k8s status -e dev.yaml
   ```

5. Tear it down (persistent volumes are **kept** by default):

   ```
   solace k8s down -e dev.yaml            # keeps PVCs
   solace k8s down -e dev.yaml --purge    # also clears PVCs (irreversible)
   ```

## Configuration (`-e`/`--env`)

Every command reads one YAML env file, selected with `-e`/`--env`. The value is an actual
**file name**, taken literally -- no extension is ever inferred, so `-e dev` and
`-e dev.yaml` name different files:

- A **bare file name** is searched in the base directory, then in `<base-dir>/env` -- so
  `-e dev.yaml` finds `./dev.yaml` if it exists, otherwise `./env/dev.yaml`. The first hit
  wins, which means a copy in the base directory **shadows** the `env/` copy of the same
  name. Every run echoes the file it resolved to (`==> env file: ...`, on stderr), so the
  winner is never a surprise.
- `--base-dir` replaces the current directory for both lookups (default: current directory).
- A value carrying a **directory component** is used exactly as typed and is *not* retried
  under `env/` or joined with `--base-dir` -- e.g. `-e ./configs/prod.yaml`,
  `-e ../shared/prod.yaml`, or an absolute path.
- The default name is `env.yaml`, so a bare `solace k8s status` looks for `./env.yaml` then
  `./env/env.yaml`. Neither is shipped; copy `env/sample.yaml` to create your own.

When no candidate exists the error names every path that was tried.

Decoding is **strict**: an unknown or misspelled key is a hard error, so typos fail loud
instead of being silently ignored. A file that is not YAML at all is reported as such --
and if it looks like a legacy bash env file, the error points at `solace convert` (below).

### Migrating from the bash env files (`solace convert`)

The pre-Go scripts kept their configuration in shell files under `bash/env/`, sourced by
`000-env.sh`. `solace convert` turns one into the YAML this CLI reads:

```
solace convert bash/env/prod -o prod.yaml                 # kubernetes flavour
solace convert bash/docker-podman/env/prod -o prod.yaml   # docker/podman flavour
solace k8s check -e prod.yaml --dry-run
```

- The **platform section** is detected from the variables present (`SOLBK_NS`/`SOLOP_*` ->
  `k8s`, `SOLBK_NODE_*`/`DOCKER_MODE`/`PODMAN_ROOTLESS` -> `docker`/`podman`). Pass
  `--platform k8s|docker|podman` to choose it yourself; the choice is echoed either way.
- The source is read as an **env file, not a shell script**: one assignment per line
  (scalars, `( ... )` arrays, `declare -A` maps, `export`/`declare` prefixes, `${VAR}`
  references, and trailing comments are all understood). Shell constructs beyond
  assignments are skipped.
- Only what the env file **actually set** is written. Values the bash bootstrap defaulted
  are left out, so the Go defaults apply instead.
- A variable with no YAML equivalent is **named on stderr**, never dropped silently. So are
  a non-numeric value for a numeric field and an unrecognised `SOLBK_REDUNDANCY`.
- The converted file is re-read and validated, so a source env that was already missing
  mandatory values says so at conversion time.
- Without `-o` the YAML goes to stdout (warnings stay on stderr). With `-o` the file is
  written `0600` and an existing file is **not** overwritten unless you pass `--force`.

The output carries every secret from the source file verbatim -- treat it like the source,
and never commit it.

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
| `k8s.runtime` | `kubectl` | Cluster CLI (legacy `KUBE`). A scalar is split on whitespace, so it can be a drop-in (`oc`), a wrapper (`microk8s kubectl`), or a profile (`kubectl --kubeconfig <file>`). Use a list when a token contains a space |
| `docker.runtime` / `podman.runtime` | `docker` / `podman` | Container CLI (legacy `CONTAINER_RUNTIME`), same forms as `k8s.runtime` |
| `tls.serverSecret` | -- | Name of the TLS secret; its presence enables the CR's TLS block |
| `timezone` | -- | Broker timezone, all platforms (the CR's `timezone` and the containers' `TZ`). Omitted keeps the image default |
| `k8s.securityContext` | -- | `runAsUser`/`fsGroup` for the pod. Omitted entirely when unset |
| `k8s.containerSecurity` | -- | `runAsUser`/`runAsGroup`/`readOnlyRootFilesystem` for the broker container |

**Secrets** belong in the env file (or your own secret store). The tool never echoes
them: under `--dry-run`, values piped to a command on stdin are shown as
`<<< (N bytes on stdin)`, never as their contents.

## Global flags

These apply to every subcommand:

| Flag | Default | Meaning |
| --- | --- | --- |
| `-e`, `--env <file>` | `env.yaml` | Env file to load: a file name searched in the base dir then `<base-dir>/env`, or a path used as-is. |
| `--base-dir <dir>` | current dir | Directory searched for the env file, and holding `env/`. |
| `--gen` | `false` | Render the artifact this command would apply and print it; change nothing. |
| `--dry-run` | `false` | Print the external commands instead of running them. |
| `-y`, `--yes` | `false` | Skip confirmation prompts. Does **not** imply `--purge`. |

## Command reference (Kubernetes)

The tables below group the commands by lifecycle phase. For the **complete** surface --
every command, its arguments, and every flag with its default -- see
[docs/commands.md](docs/commands.md), which is generated from the command tree itself.

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
solace k8s gen broker -e dev.yaml            # the PubSubPlusEventBroker CR
solace k8s gen operator -e dev.yaml          # the operator bundle
solace k8s deploy -e dev.yaml --gen          # equivalent to gen broker
solace k8s operator deploy -e dev.yaml --gen # equivalent to gen operator
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

As with Kubernetes, [docs/commands.md](docs/commands.md) carries the complete generated
reference for both trees.

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
solace podman up primary -e prod.yaml       # on the primary host
solace podman up backup  -e prod.yaml       # on the backup host
solace podman up monitor -e prod.yaml       # on the monitor host
solace podman config leader -e prod.yaml    # on the primary only
# then, concurrently on the primary and backup hosts:
solace podman verify redundancy -e prod.yaml
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
| `scan` | `go tool govulncheck -format json` (version pinned in `go.mod`/`go.sum`), judged by [internal/tools/vulnjudge](internal/tools/vulnjudge) -- **fatal** on a fixable vulnerability this module calls, **warns and passes** on one with no released fix. Raw stream kept at `scripts/logs/scan.json` |
| `dist` | Local convenience: cross-compile all four release targets into `dist/` |
| `graphify` | Refresh `graphify-out/`. Local only; skipped when `CI` is set |
| `all` | `build vet test` -- the fast inner loop; CI runs `all scan` |
| `full` | `all` + `cov scan graphify` -- the pre-tag sweep |

Run the local gate with `scripts/dev.ps1 all scan` (or `./scripts/dev.sh all scan`), and
`full` before tagging. Per-task logs land in `scripts/logs/<task>.log`, each closing with a
`<timestamp> | <task> | <duration>s | OK|FAILED` footer; coverage HTML in
`coverage/coverage.html`. Current test coverage is 90.4% (recorded in
`scripts/logs/cov.log`; the previous total is the local floor, not an enforced numeric gate).

[docs/test.md](docs/test.md) catalogues every test in the repo -- what each one proves, the
per-package fixtures and doubles to reuse, and the injectable seams. Read it before adding a
test, and update it in the same change when you add or remove one.

The Go toolchain is pinned by the `toolchain` line in [go.mod](go.mod), not just the `go`
line: `go 1.26` is a minimum, so a machine with an older Go would otherwise build against
the *oldest* 1.26 patch and ship its unpatched standard library. Both dev scripts export
`GOTOOLCHAIN` from that line unless you set it yourself, so an exported `GOTOOLCHAIN=local`
cannot quietly bypass the pin. `scan` reports standard library vulnerabilities like any
other -- when it does, raise that `toolchain` line to the release that fixes them and
re-run the gate.

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
| `internal/convert` | Legacy bash env -> YAML converter behind `solace convert`. |
| `internal/cli` | Cobra command tree and handlers. |
| `internal/tools/vulnjudge` | Dev-only judge the `scan` task pipes govulncheck JSON through. |
| `env/` | Config files (`sample.yaml` is the annotated template). |
| `docs/` | [commands.md](docs/commands.md) -- generated CLI reference; [test.md](docs/test.md) -- the catalogue of every test. |
| `scripts/` | `dev.ps1` / `dev.sh` developer tooling. |
