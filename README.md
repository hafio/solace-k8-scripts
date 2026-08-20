# solace-util

A single Go binary that deploys and operates Solace PubSub+ Event Brokers on
Kubernetes, Docker, or Podman. You describe the broker once in a YAML env file and drive
the whole lifecycle through one standardized command tree:

```
check deploy -> prepare all -> deploy all     build it
config ...                                    post-deployment, over the broker CLI
check semp-login / smoke redundancy           prove it works
stop broker / start broker                    pause it without removing it
remove all                                    tear it down
```

Every verb that owns more than one kind of object names the object it acts on --
`deploy broker`, `remove operator`, `status broker` -- and the bare verb prints what it
can act on rather than doing something implicit; `remove` on its own removes nothing.
The operator is cluster-scoped and shared between brokers, so it is installed and removed
on its own (`deploy operator`, `remove operator`) rather than as a side effect of a
broker's own bring-up or tear-down. See [Command reference](#command-reference) for the
full tree and [Bringing up a fresh cluster](#bringing-up-a-fresh-cluster) for what a first
run looks like end to end.

**Platform status**

| Platform | State |
| --- | --- |
| Kubernetes (via the Solace EventBroker Operator) | Fully supported |
| Docker / Podman (host-local containers, no operator) | Fully supported |

> Unsupported -- this is not a Solace product. Use at your own risk.

## Requirements

- **Build:** Go 1.26+.
- **Run (Kubernetes):** `kubectl` on your `PATH` and a reachable cluster/context. The
  binary shells out to `kubectl`; it does not embed a Kubernetes client. Set `kubernetes.runtime`
  to use `oc` instead, or to carry a whole profile such as
  `kubectl --kubeconfig /path/.kubeconfig-dev`. A wrapper like `microk8s kubectl` needs
  `--allow-command` -- see **The command fields are executable content** below.
- **Run (Docker/Podman):** the `docker` or `podman` binary on your `PATH`, on the host that
  runs the broker. Podman deploys a systemd **quadlet** unit and uses podman's own secret
  store (its host also needs systemd); Docker deploys through **compose**, so that host also
  needs the compose plugin (`docker compose`) or the standalone `docker-compose` binary --
  set `docker.compose` when it is the latter. The binary shells out to the runtime; it embeds
  no container client. `docker.runtime` / `podman.runtime` override the command the same way.
- **Podman secrets:** `deploy broker` stores the broker's secrets with
  `podman secret create --replace` and mounts them into the container
  (`type=mount`), neither of which the oldest podman builds support. Confirm yours does
  (`podman secret create --help | grep -- --replace`) -- `check deploy` only proves the
  runtime answers `version`, so an unsupported flag surfaces at deploy time.
- **Docker compose secrets:** the generated compose file sources each secret from a host
  environment variable, which needs **compose v2.23.1 or later** (`docker compose version`).
  On an older compose the `environment:` secret source is not understood and `deploy broker`
  fails.
- `generate <target>` needs no cluster, runtime, or `kubectl`/`docker`/`podman` binary -- it
  renders the artifact a command would apply and prints it, without running anything. It is
  the only way to preview a command's effect before running it; there is no `--dry-run` flag.

## Build

Quick local build:

```
go build -o solace-util .
```

Release binaries (cross-compiled, stripped) via the dev scripts:

```
scripts/dev.ps1 dist      # Windows
./scripts/dev.sh dist     # Linux/macOS
```

This emits `dist/solace-util-<os>-<arch>` for linux/amd64, linux/arm64, darwin/arm64,
and windows/amd64 (Windows gets a `.exe` suffix), built with
`CGO_ENABLED=0 -trimpath -ldflags '-s -w -X solace/internal/cli.version=<version>'`.
`scripts/dev.sh build` compiles a single target instead -- the host's by default, or
`TARGET_OS`/`TARGET_ARCH` when set, which is how the release pipeline drives it.

`<version>` is `git describe --tags --dirty --always`: on a tag push HEAD is exactly
the pushed tag, so a release binary reports e.g. `v1.2.3`, matching the GitHub
release; a local build between tags reports a pseudo-version like
`v1.2.3-5-gabc1234-dirty`. The quick local `go build` above stamps nothing and
reports `dev`.

## Version

`solace-util version` prints the stamped version (or `dev`), the Go toolchain, and
the OS/arch the binary was built for:

```
solace-util v1.2.3 go1.26.5 linux/amd64
```

## Shell completion

`solace-util auto-complete <shell>` prints a completion script on stdout. Load it into
the current shell:

```
source <(solace-util auto-complete bash)                               # bash
source <(solace-util auto-complete zsh)                                # zsh
solace-util auto-complete fish | source                                # fish
solace-util auto-complete powershell | Out-String | Invoke-Expression  # PowerShell
```

Load it for every session:

```
solace-util auto-complete bash > /etc/bash_completion.d/solace-util           # bash
solace-util auto-complete zsh > "${fpath[1]}/_solace-util"                    # zsh (compinit enabled)
solace-util auto-complete fish > ~/.config/fish/completions/solace-util.fish  # fish
solace-util auto-complete powershell > solace-util.ps1                        # PowerShell: source from $PROFILE
```

Beyond commands and flag names, it completes the values they take:

- `-e`/`--env` -- the `.yaml`/`.yml` files it would actually resolve, searched in
  the base dir then `<base-dir>/env`, the same order described under
  [Configuration](#configuration--e--env). A value you type with a directory in it
  falls back to ordinary file completion, because that is how it resolves.
- the `[role]` positionals and `--pod` -- `primary`, `backup`, `monitor` (the commands
  themselves also accept the short forms `p`/`b`/`m`; completion offers only the long
  names, since abbreviations exist to save typing something you already know).
- `--base-dir` and `--dir` -- directories only.
- `--platform` and `convert --platform` -- `kubernetes`, `docker`, `podman` (the canonical
  names only; the `kube`/`dk`/`pm` abbreviations exist to save typing something you already
  know, so a completion does not offer a second spelling for the same platform -- and
  neither `k8s` nor `k8` is accepted anywhere, as either an abbreviation or a section name).

Completion never reads your env file: a TAB press cannot parse config, run a
command, or print anything into the shell. Add `--no-descriptions` to any of the
above to drop the help text shown beside each suggestion.

## Quick start (Kubernetes)

1. Copy the annotated sample to your own env file and edit it:

   ```
   cp env/sample.yaml env/dev.yaml
   ```

   At minimum set `image.repo`, `image.tag`, `admin.pass`, `kubernetes.name`,
   `kubernetes.namespace`, and `kubernetes.storage.msgNode`. Delete the `docker:` and
   `podman:` sections that `env/sample.yaml` also carries -- the CLI is one flat command
   tree, and it picks the platform from which of those sections your file declares (see
   [Which platform runs](#which-platform-runs) below).

2. Render the broker manifest first to see exactly what would be applied -- this needs no
   cluster at all, since `generate` never contacts one. `-e` takes the file name; it is
   found under `env/` because there is no `./dev.yaml`:

   ```
   solace-util generate broker -e dev.yaml
   ```

3. A fresh cluster needs the EventBroker operator once. It is cluster-scoped and shared,
   so it is installed on its own rather than as part of bringing up any one broker (see
   [Bringing up a fresh cluster](#bringing-up-a-fresh-cluster) below):

   ```
   solace-util deploy operator -e dev.yaml
   ```

4. Check prerequisites, then bring the broker up (`deploy all` runs
   check -> prepare -> deploy -> assert the config-sync leader if HA):

   ```
   solace-util check deploy -e dev.yaml
   solace-util deploy all -e dev.yaml
   ```

5. Prove it works, and inspect:

   ```
   solace-util check semp-login -e dev.yaml
   solace-util status broker -e dev.yaml
   ```

6. Tear it down. `remove all` keeps persistent data by default, asks before deleting it,
   and leaves the operator installed (see
   [Removing a broker: what stays, what goes](#removing-a-broker-what-stays-what-goes)
   below for the full contract):

   ```
   solace-util remove all -e dev.yaml                  # asks about the PVCs; keeps them if you decline
   solace-util remove all -e dev.yaml --delete-data     # deletes them without asking (irreversible)
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
- The default name is `env.yaml`, so a bare `solace-util check deploy` looks for
  `./env.yaml` then `./env/env.yaml`. Neither is shipped; copy `env/sample.yaml` to create
  your own.

When no candidate exists the error names every path that was tried.

Decoding is **strict**: an unknown or misspelled key is a hard error, so typos fail loud
instead of being silently ignored. A file that is not YAML at all is reported as such --
and if it looks like a legacy bash env file, the error points at `solace-util convert` (below).

### Which platform runs

The command tree is flat and identical on every platform -- there is no `kubernetes`,
`docker` or `podman` subtree to type. The platform is a property of the deployment the env
file already describes, so it is resolved from that file rather than repeated on the
command line. Resolution happens in this order:

1. **`--platform <name>`**, if given. It accepts the canonical names `kubernetes`, `docker`,
   `podman`, or the abbreviations `kube`, `dk`, `pm`. It must name a platform section the env
   file actually declares -- passing `--platform docker` against a file with no `docker:`
   section fails loudly rather than running against a platform the file never described.
2. **Otherwise the env file decides**, from whichever of the top-level `kubernetes:`,
   `docker:`, and `podman:` sections it declares:
   - **exactly one** -> used silently, and named in the preamble
     (`==> platform: docker (from ./dev.yaml)`)
   - **none** -> a loud error telling you to add one
   - **more than one** -> an interactive prompt listing them; a non-interactive run
     (piped, CI) fails loudly instead and tells you to pass `--platform`

An env file **must** declare its platform section even when every setting under it
defaults -- write `docker: {}` rather than leaving the section out. This matters because
docker and podman have no mandatory field of their own, so without the marker the file
would be indistinguishable from a kubernetes one that simply hasn't set any docker keys.

The tree itself is the union of every platform's commands, and it renders the same way
regardless of which platform an env file names -- `--help` and shell completion never load
one, so a command cannot appear or disappear depending on a file they have not read. A
command that does not apply to the platform a file resolves to says so in its help text
(for example "(kubernetes only)") and refuses at run time with a named error rather than
silently doing nothing.

### Migrating from the bash env files (`solace-util convert`)

The pre-Go scripts kept their configuration in shell files under `bash/env/`, sourced by
`000-env.sh`. `solace-util convert` turns one into the YAML this CLI reads:

```
solace-util convert bash/env/prod -o prod.yaml                 # kubernetes flavour
solace-util convert bash/docker-podman/env/prod -o prod.yaml   # docker/podman flavour
solace-util check deploy -e prod.yaml
```

- The **platform section** is detected from the variables present (`SOLBK_NS`/`SOLOP_*` ->
  `kubernetes`, `SOLBK_NODE_*`/`DOCKER_MODE`/`PODMAN_ROOTLESS` -> `docker`/`podman`). Pass
  `--platform kubernetes|docker|podman` to choose it yourself; the choice is echoed either way.
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
and never commit it. (Switch the values to their `*Env` reference keys afterwards and it
becomes safe to commit; see **Secrets** below.) `SOLBK_USR_SECRET` converts to
`kubernetes.adminSecret`, and each `SOLBK_USR_PASS` entry to an `admin.additionalUsers` entry with
`accessLevel: none` -- the bash flow set no level, so the converter picks the least
privileged one and says so; raise it per user as needed.

**Renamed keys.** An env file written for an earlier version fails loud on the unknown key
(strict decoding), so every rename is visible immediately rather than silently ignored: the
whole `k8s:` section is now `kubernetes:`, `admin.userSecret` is now
`kubernetes.adminSecret`, and the `admin.userPasswords` `["user=password"]` list is now
`admin.additionalUsers`. There is deliberately no alias for `k8s:` -- a section answering to
two names would leave both spellings of the schema in circulation forever.

**The command tree changed too, and this time there is no alias either.** The old
per-platform trees (`solace-util kubernetes ...`, `solace-util docker ...`,
`solace-util podman ...`, with `k8s` accepted as a synonym for `kubernetes`) are gone: the
tree is flat now, and the platform comes from the env file (or `--platform`) instead of the
first word on the command line -- see [Which platform runs](#which-platform-runs). `k8s` is
not accepted anywhere any more, neither as a subtree name nor as a `--platform` value
(`kube` is the abbreviation now). An old runbook line

```
solace-util k8s deploy -e prod.yaml
```

becomes

```
solace-util deploy broker -e prod.yaml
```

with nothing else to change beyond naming the object (see the next section), as long as
`prod.yaml` declares exactly one platform section (pass `--platform kubernetes` instead if
it declares more than one).

**The verb tree changed again since then, to remove every implicit action.** A verb that
owns more than one kind of object no longer acts when run bare -- it prints what it can act
on, so `remove` on its own removes nothing and `config` on its own configures nothing. Every
rename below is permanent; there is no alias for any of them, the same policy as `k8s:`
above:

| Old | New |
| --- | --- |
| `up` | `deploy all` |
| `down` | `remove all` |
| `delete` | `remove broker` |
| `prep` | `prepare` |
| `verify login` | `check semp-login` |
| `verify redundancy` | `smoke redundancy` |
| `verify diagnostics` | `diagnostics` |
| `verify` (bare -- the redundancy+login bundle) | gone, no replacement |
| `config exec-cli FILE` | `cli --input FILE` |
| `gen` | `generate <target>` |
| `completion` | `auto-complete` |
| `show-all` | `status broker --all` |
| `describe broker` / `describe lb` / `inspect` | `status broker --detail` |
| `replicas start` / `replicas stop` | `start broker` / `stop broker` |
| `operator deploy` / `delete` / `status` / `logs` / `describe` | `deploy operator` / `remove operator` / `status operator` / `logs operator` / `status operator --detail` |
| `teardown domain-certs` | `config delete domain-certs` |
| `teardown secrets` / `teardown namespace` | `remove secrets` / `remove namespace` |
| `config` (bare -- the run-everything step) | gone, no replacement (see [Post-deployment configuration order](#post-deployment-configuration-order)) |

Four flags went with them, with no replacement: `--dry-run`, the `--gen-only` /
`--gen-secrets-only` / `--gen-env-only` trio, `deploy --keep-yaml`, and the
`--purge` / `--clear-data` / `--keep-data` trio. `generate <target>` is now the only way to
preview an artifact before applying it (see
[Rendering without applying](#rendering-without-applying)), and the retained-data flags are
now `--delete-data` / `--delete-crd` / `--no-prompt` (see
[Removing a broker: what stays, what goes](#removing-a-broker-what-stays-what-goes)).

**Removed keys: `kubernetes.msgNode.cpu`, `scaling.maxPool`, and `docker.mode`.** Broker CPU
is now fixed by the scaling tier rather than set by hand, and `maxPool` named the same
broker setting as `scaling.maxSpoolUsageMB` -- one concept under two platform-specific keys,
which the scaling block no longer has. Both fail to load naming their replacement;
`kubernetes.msgNode.mem` is unaffected. `docker.mode` is gone outright -- docker always
deploys through compose now, so there is no mode to choose, and a file still carrying the
key fails strict decoding as an unknown field. See [Scaling](#scaling).

**`env/sample.yaml` is the authoritative, fully annotated schema** -- start there rather
than from this README. The most-used keys:

Minimum required (Kubernetes):

| Key | Purpose |
| --- | --- |
| `image.repo` | Broker image repository |
| `image.tag` | Image tag |
| `admin.pass` | Broker admin password (never defaulted) |
| `kubernetes.name` | Broker / custom-resource name |
| `kubernetes.namespace` | Target namespace |
| `kubernetes.storage.msgNode` | Message-node PVC size (e.g. `30Gi`) |

Common optional knobs:

| Key | Default | Purpose |
| --- | --- | --- |
| `redundancy` | `no` | `yes` = HA group (primary+backup+monitor); `no` = single standalone broker. HA provisions three brokers, so it must be asked for explicitly |
| `image.registry` | docker.io | Registry prefix for the image reference |
| `kubernetes.storage.class` | cluster default | StorageClass for the broker PVCs |
| `kubernetes.updateStrategy` | `automatedRolling` | `automatedRolling` or `manualPodRestart` |
| `kubernetes.runtime` | `kubectl` | Cluster CLI (legacy `KUBE`). A scalar is split on whitespace, so it can be a drop-in (`oc`) or a profile (`kubectl --kubeconfig <file>`). **Restricted** -- see [The command fields are executable content](#the-command-fields-are-executable-content) |
| `docker.runtime` / `podman.runtime` | `docker` / `podman` | Container CLI (legacy `CONTAINER_RUNTIME`), same forms and the same restrictions as `kubernetes.runtime` |
| `docker.compose` | `<runtime> compose` | The compose invocation. Set it to `docker-compose` on a host carrying only the standalone v1 binary; same forms and restrictions as `runtime`, plus the one permitted `compose` subcommand |
| `<docker\|podman>.container.healthCheck.enabled` | `false` | Adds an engine health check polling the broker's own `/health-check/readiness` on port 5550 every 5s, so `docker ps` and podman's auto-restart see readiness rather than liveness. Needs broker **10.26 or later** and a version-numbered `image.tag`; set `healthCheck.cmd` to supply your own probe instead (which skips the version check). Container-only by design -- on Kubernetes the operator already probes the pods |
| `tls.serverSecret` | -- | Name of the TLS secret; its presence enables the CR's TLS block |
| `kubernetes.adminSecret` | `solace-admin-secret` | Name of the Kubernetes Secret holding the admin/monitor credentials. Was `admin.userSecret` |
| `admin.additionalUsers` | -- | Extra CLI (management) users, each `{username, accessLevel, password\|passwordEnv}` with `accessLevel` one of `none`, `read-only`, `mesh-manager`, `read-write`, `admin`. Replaces the old `admin.userPasswords` `user=password` list. Created at boot on containers, and by `config apply additional-users` on Kubernetes -- see below |
| `admin.user` | `admin` | Broker admin username. **docker/podman only** -- it names the container's `username_<user>_globalaccesslevel` setting, its mounted password file and the SEMP login. On Kubernetes the operator reads the fixed `username_admin_password` key out of `kubernetes.adminSecret` and creates the user itself, so the admin user is always `admin` there and any other value is a load-time error rather than a silently ignored key |
| `admin.passEnv` (and every other `*Env`) | -- | Name of an environment variable holding the secret, instead of the value itself. See **Secrets** below |
| `timezone` | -- | Broker timezone, all platforms (the CR's `timezone` and the containers' `TZ`). Omitted keeps the image default |
| `broker.cliScriptsFolder` / `broker.diagDir` / `broker.productKeys` / `broker.domainCerts` | -- | Platform-neutral: local folder for `cli --input` scripts, local folder for `diagnostics` output, the list `config apply product-keys` applies, and the CA files `config apply domain-certs` loads. Every platform runs these same post-deployment steps identically, which is why the section sits at the top level rather than under `kubernetes.*` |
| `kubernetes.securityContext` | -- | `runAsUser`/`fsGroup` for the pod. Omitted entirely when unset |
| `kubernetes.containerSecurity` | -- | `runAsUser`/`runAsGroup`/`readOnlyRootFilesystem` for the broker container |
| `scaling.*` | see [Scaling](#scaling) | Broker sizing, applied on every platform -- the CR's `spec.systemScaling` on Kubernetes, container environment variables on docker and podman |
| `scaling.maxConnections` | `100` (Kubernetes) / `1000` (container) | The Solace scaling tier. Fixes the broker's CPU and defaults its memory on every platform -- see [Scaling tiers](#scaling-tiers) |
| `<docker\|podman>.container.mem` | the tier's memory | Container memory limit, in docker's and podman's own `b\|k\|m\|g` suffix (not Kubernetes' `Mi`/`Gi`). There is no matching cpu key: CPU is fixed by the tier |

### Scaling

Every key under `scaling` applies to every platform. Only the delivery differs: Kubernetes
writes them into the broker CR's `spec.systemScaling`, while docker and podman pass them to
the container as environment variables under the broker's own setting names. One env file
therefore sizes the same broker whichever platform runs it.

| `scaling` key | Broker setting (container env var / CR field) |
| --- | --- |
| `maxConnections` | `system_scaling_maxconnectioncount` |
| `maxQueueMessages` | `system_scaling_maxqueuemessagecount` |
| `maxKafkaBridge` | `system_scaling_maxkafkabridgecount` |
| `maxKafkaConnections` | `system_scaling_maxkafkabrokerconnectioncount` |
| `maxBridges` | `system_scaling_maxbridgecount` |
| `maxSubscriptions` | `system_scaling_maxsubscriptioncount` |
| `maxGuaranteedMsgMB` | `system_scaling_maxguaranteedmessagesize` |
| `maxSpoolUsageMB` | `messagespool_maxspoolusage` (the CR spells it `maxSpoolUsage`) |

Defaults are identical across platforms except `maxConnections` (100 on Kubernetes, 1000 on
containers) and `maxSpoolUsageMB` (10000 on Kubernetes, 100000 on containers).

#### Scaling tiers

`scaling.maxConnections` is the Solace scaling tier, and it decides the broker's CPU on all
three platforms. CPU is **not** configurable: sizing a broker by connection count and then
sizing its CPU independently is how a 200k-connection broker ends up on two cores. Memory is
the tier's default and stays yours to override; storage is untouched by the tier.

| `scaling.maxConnections` | CPU cores (fixed) | Memory default (Kubernetes / container) |
| --- | --- | --- |
| `100` (Kubernetes default) | 2 | `3410Mi` / `3410m` |
| `1000` (container default) | 2 | `6898Mi` / `6898m` |
| `10000` | 4 | `12435Mi` / `12435m` |
| `100000` | 8 | `30925Mi` / `30925m` |
| `200000` | 12 | `52581Mi` / `52581m` |

The value must be **exactly** one of those five. A value between tiers is rejected rather
than rounded, because Solace publishes no sizing for it. Override memory with
`kubernetes.msgNode.mem` (a Kubernetes quantity, `Mi`/`Gi`) or `<docker|podman>.container.mem`
(docker's and podman's own `b|k|m|g` suffix -- the engines reject `Mi`, so the two spellings
are not interchangeable and the loader says so).

Docker and podman previously emitted no CPU or memory limit at all. They now carry the
tier's cap in the generated compose file (`cpus:`, `mem_limit:`) and quadlet unit
(`PodmanArgs=--cpus=`, `Memory=`), so an existing container deployment must be **redeployed**
-- not just restarted -- to pick them up. In an HA group the monitor host gets the same caps
as the messaging hosts; these are ceilings rather than reservations, so an oversized monitor
limit costs nothing.

#### File descriptors on rootless podman

Both container artifacts ask the engine for `<docker|podman>.container.ulimits.nofile`
(default `2448:1048576`). A **rootless** container cannot raise `nofile` above the hard limit
of the user invoking podman -- the kernel refuses -- so `prepare host` checks it on a podman
env file and stops with the exact drop-in to add when it is too low:

```
solace-util prepare host -e env/prod.yaml
...
error: rootless podman: this user's hard nofile limit is 1024, but
podman.container.ulimits.nofile needs 1048576 -- a rootless container cannot raise it
above the user's own hard limit, so the broker would start under-provisioned.
  Add this as root to /etc/security/limits.d/99-solace.conf, replacing <user> with the
  account that runs the container, then log out and back in:
    <user> hard nofile 1048576
    <user> soft nofile 2448
```

Prep reports and refuses rather than fixing it: raising a hard limit means editing host-wide
security configuration as root, which is what a rootless deployment exists to avoid. Rootful
podman and docker are unaffected -- their privileged engine raises the limit itself -- so the
check runs only for `podman.rootless: true`.

### The command fields are executable content

`kubernetes.runtime`, `docker.runtime`, `podman.runtime` and `docker.compose` each name a binary
this tool runs **on your machine**. Env files travel -- repositories, pull requests, shared
archives -- so the person who wrote one is routinely not the person who runs it. Treat an
env file the way you would treat a script someone sent you: **read the command fields before
running anything with it.**

To make that review short, the fields are restricted. A command is accepted only when:

1. **Every token is inert and visible.** No control characters, no whitespace inside a single
   argument (any Unicode whitespace, not just the ASCII space), no invisible formatting
   characters (zero-width spaces and joiners, bidirectional overrides), no quotes, no
   backslash, no backtick, and none of `$ ; | & < > ( ) * ? [ ] { } ~ # !`. Nothing is ever
   passed through a shell, so these are not injections -- but tokens end up in logs and in
   the `-v/--verbose` exec trace, and a token you cannot see is one you cannot
   review. A Windows path in a flag value therefore needs forward slashes:
   `--kubeconfig C:/Users/you/.kube/config`.
2. **The binary is a bare name from the allowlist.** No `/` or `\` anywhere in it: a path
   would run a file the env file chose -- such as a `./kubectl` unpacked beside it -- rather
   than the one on your `PATH`. One optional `.exe` is stripped, then the name must be:

   | Platform | Allowed |
   | --- | --- |
   | Kubernetes | `kubectl`, `oc` |
   | Docker | `docker`, `docker-compose`, `nerdctl` |
   | Podman | `podman` |

3. **Nothing after it is a bare word.** Flags and their values are fine
   (`kubectl --context prod -n solace`); a bare word is not, because this tool appends its
   own subcommand and a word in that position would run ahead of it. `kubectl delete` in a
   config is exactly the attack. The literal `--` is refused for the same reason.

Anything else -- a wrapper such as `microk8s kubectl` or `lima nerdctl`, a site-specific
shim -- runs only when **you** approve it, per invocation:

```sh
solace-util deploy broker --allow-command microk8s   # kubernetes env file wrapping kubectl in microk8s
solace-util deploy all --allow-command lima          # docker/podman env file wrapping the runtime in lima
```

`--allow-command` is repeatable, takes a bare name (never a path), and exists **only** as a
command-line flag. There is deliberately no env-file key, environment variable, or any other
way for a config to widen its own allowlist: the authority to run something unusual belongs
to the person who can see what they are approving. It is rejected on any `generate` command,
where nothing executes.

**Privilege escalation is never approvable**, by the config or by you: `sudo`, `doas`, `su`,
`pkexec`, `run0`, `runas` and `gsudo` are refused as `--allow-command` values. This is not a
ban on running as root -- rootful podman needs it. It is about *where* you elevate. A
`runtime: sudo podman` elevates every command this tool issues, for the whole life of an env
file, decided by whoever wrote that file. Elevate the tool instead, at the moment you run it,
so the privilege belongs to one invocation you chose:

```sh
sudo solace-util deploy all -e prod.yaml   # yes (prod.yaml is a podman env file)
# runtime: sudo podman  in the env file      # never
```

The same check runs twice -- once when the env file is loaded, and again immediately before
any command line is built -- from a single implementation, so a hostile file is inert even
on a path that skipped validation.

**What this does not protect against.** Two things are out of scope, and no amount of
parsing would fix either:

- **A compromised machine.** If an attacker has already put a trojan `kubectl` on your
  `PATH`, they own the host; nothing this tool checks can help. What it does do is make the
  binary's real location visible -- before any work starts, each binary this env file names
  (`kubernetes.runtime`, `docker.runtime`, `podman.runtime`, `docker.compose`) is resolved and
  printed as `==> using <name>: <resolved path>`, and `-v/--verbose` prints every command as
  it runs -- and refuse to resolve a bare name from the current directory.
- **Config that is malicious but perfectly legitimate in form.** `kubernetes.namespace: production`,
  or a valid `kubectl --context` aimed at the wrong cluster, is a review problem. So is a
  flag's *value*: this tool cannot know how many arguments a flag takes, so the token after
  `--kubeconfig` is accepted as that flag's value whatever it says. The hard guarantee covers
  the binary and every bare word -- not flag values.

Rendering executes nothing: every `generate` command only ever calls the templating package,
never an external command, so pointing one at an env file you did not write cannot run
anything. Note that it still *loads* the file, so one whose command field breaks the rules
above fails there rather than printing an artifact -- which is itself the answer you wanted
about that file. To read a command field without loading anything at all, open the file.

### Secrets

Every secret field takes either the value itself or -- through a sibling `*Env` key --
the **name of an environment variable** to read it from. Setting both is an error, and so
is naming a variable that is unset or empty: the load fails naming the key and the
variable rather than deploying a broker with a blank password.

| Value key | Reference key |
| --- | --- |
| `admin.pass` | `admin.passEnv` |
| `admin.monitorPass` | `admin.monitorPassEnv` |
| `admin.additionalUsers[].password` | `admin.additionalUsers[].passwordEnv` |
| `tls.certPassphrase` | `tls.certPassphraseEnv` |
| `image.pass` | `image.passEnv` |
| `nodes.psk` | `nodes.pskEnv` |

```yaml
admin:
  passEnv: SOLACE_ADMIN_PASS     # export SOLACE_ADMIN_PASS before any command
```

With the `*Env` form the env file carries no secret and is safe to commit and share. A
value is otherwise used **verbatim** on every platform -- a `$VAR` or `${VAR}` inside one
is a literal password, never expanded. `nodes.pskEnv` also opts out of PSK generation:
`prepare host` only generates a key when the literal `nodes.psk` is empty, so with the
reference form create it yourself (`openssl rand -base64 60`) and export the same value on
all three hosts.

The tool never echoes a secret. Values piped to a command on stdin show as
`<<< (N bytes on stdin)` under `-v/--verbose`, values passed to a child process's environment
as `NAME=***`, and `check deploy`/`status broker` report only whether each one is set. The
one exception is explicit: `generate secrets` prints the values themselves, because printing
them is what that command is for.

## Global flags

These apply to every subcommand:

| Flag | Default | Meaning |
| --- | --- | --- |
| `-e`, `--env <file>` | `env.yaml` | Env file to load: a file name searched in the base dir then `<base-dir>/env`, or a path used as-is. |
| `--base-dir <dir>` | current dir | Directory searched for the env file, and holding `env/`. |
| `--platform <name>` | -- | Platform to drive: `kubernetes` (`kube`), `docker` (`dk`) or `podman` (`pm`). Default: the one the env file declares, or a prompt if it declares several. See [Which platform runs](#which-platform-runs). |
| `-v`, `--verbose` | `false` | Announce every external command as it runs (`==> exec: <resolved path> <args>`). By default the binaries this env file names are resolved and listed once, up front. |

There is no global `--yes`. Confirmation is per-command, because only the commands that
destroy something ask: `--no-prompt` is declared on each of those (`remove *`, `restart
broker`) and silences every question that command would ask, while still keeping the
retained layer unless `--delete-data`/`--delete-crd` names it. See
[Removing a broker: what stays, what goes](#removing-a-broker-what-stays-what-goes).

On every command that executes something (that is, every command except the `generate`
tree, where nothing does):

| Flag | Default | Meaning |
| --- | --- | --- |
| `--allow-command <name>` | -- | Approve one extra binary for this env file's platform command, for this run only. Repeatable; a bare name, never a path. See [The command fields are executable content](#the-command-fields-are-executable-content). |

`--allow-command` is rejected on any `generate` command, because nothing executes there and
a flag that is harmless everywhere gets pasted everywhere.

There is no `--dry-run` and no per-command render flag any more: `generate <target>` is the
one, dedicated way to preview an artifact before applying it -- see
[Rendering without applying](#rendering-without-applying).

### Before anything changes: the preflight

Every command that writes a file or changes remote state runs one cheap **read-only** probe
first, using the same command the real work will use:

| Platform | Probe | Answers |
| --- | --- | --- |
| Kubernetes | `<runtime> auth can-i <verb> <resource> -n <namespace>` | Is the context live, and may this identity do the thing? |
| Docker / Podman | `<runtime> info` | Is the daemon up and reachable as this user? |

If it fails, the command stops nonzero **before the first byte is written**, passes the
CLI's own error through, and adds one line saying what to do -- log in, ask for a role
binding, start the daemon. It never logs you in or starts anything on your behalf, and
there is no flag to skip it: previewing a command's effect without touching a cluster is
what `generate` is for, and a render-only command never runs the preflight because it never
runs anything.

## Command reference

The tables below group the commands by lifecycle phase, in the one flat tree every
platform shares (see [Which platform runs](#which-platform-runs)). For the **complete**
surface -- every command, its arguments, and every flag with its default -- see
[docs/commands.md](docs/commands.md), which is generated from the command tree itself. Run
`solace-util --help` (or `--help` on any subcommand) for the live tree.

Two rules apply everywhere and are stated here once rather than on every row:

- **No implicit actions.** A verb that owns more than one kind of object (`config`,
  `status`, `remove`, and so on) never acts when run bare -- it prints the objects it can
  act on, and you name one to make it do anything. `remove` alone removes nothing.
- **Abbreviations.** Every verb below has a short form, and `broker`/`operator` carry the
  same short form (`br`/`op`) under whichever verb takes them: `check`=`ck`, `config`=`cfg`,
  `convert`=`cv`, `copy`=`cp`, `deploy`=`dp`, `diagnostics`=`diag`, `generate`=`gen`,
  `logs`=`lg`, `prepare`=`pre`, `remove`=`rm`, `restart`=`rs`, `shell`=`sh`, `status`=`sts`,
  `version`=`ver`. So `dp op`, `sts br --all` and `rm all` all work. `start` and `stop` have
  deliberately no abbreviation -- any short form is ambiguous between them and `status`,
  and that is not where to save a keystroke.

Each table's **Platform** column names where a row applies: `all` when every platform has
it, `kubernetes` or `docker/podman` otherwise. A command that exists on both but behaves
differently says so in its Description -- most often for a `[role]` positional, which means
two different things depending on the platform family, and which always also accepts the
short letters `p`/`b`/`m` (completion offers only the long names -- see
[Shell completion](#shell-completion)):

- **Kubernetes** -- `[role]` picks which pod a command targets: `p`|`b`|`m` or
  `primary`|`backup`|`monitor`, defaulting to primary.
- **Docker / Podman** -- one container per host, so `[role]` instead names which host in
  the redundancy group *this* invocation is running on: required on `deploy broker`/
  `deploy all`/`generate broker` in an HA group (ignored in standalone), and
  optional on `config leader`/`smoke redundancy`, where omitting it auto-detects the role by
  matching the host name against the `nodes.*` table.

Passing a `[role]` where it means nothing, or running a command on a platform it does not
apply to, is refused with a named error rather than silently ignored or dropped.

The verbs fall into phases. `check`, `prepare` (`prepare host` on docker/podman) and
`deploy` build the deployment, and `deploy all` orchestrates the first three. The operator
is cluster-scoped and shared between brokers, so it is deployed and removed on its own
(`deploy operator`, `remove operator`) rather than as a side effect of any one broker's
bring-up -- see [Bringing up a fresh cluster](#bringing-up-a-fresh-cluster). Everything
under **`config` is post-deployment**: each step drives the Solace CLI inside a broker that
is already running, so none of it is part of `deploy` and none of it is run by `deploy all`
-- wait for the broker to be ready (the pods, or the container/service), then run it in the
order under [Post-deployment configuration order](#post-deployment-configuration-order).

### Build the deployment

| Command | Platform | Description |
| --- | --- | --- |
| `check deploy` | all | Validate config and platform prerequisites: cluster reachability and StorageClass on Kubernetes (also warns when the operator/CRD is missing); node-name DNS resolution, the container runtime, and (Docker) the compose command on docker/podman. |
| `check semp-login [role]` | all | Test an authenticated SEMP request against a running broker. Kubernetes: `[role]` picks the pod. Docker/podman: no `[role]`. |
| `smoke redundancy [role]` | all | Exercise a real failover and fail back (HA only) -- this is the one check that DISTURBS the broker, which is why it lives under `smoke` rather than `check`. Kubernetes: no `[role]`, drives the whole group from one context. Docker/podman: `[role]` is this host; run concurrently on the primary and backup. |
| `prepare namespace` | kubernetes | Create the broker namespace. |
| `prepare secrets` | kubernetes | Create admin/monitor, TLS, and image-pull secrets. |
| `prepare labels` | kubernetes | Label cluster nodes for primary/backup/monitor placement. **Interactive and deliberately outside `prepare all`** -- the env file names the label each role wants, but only you can say which machine carries it, so it prompts. Run it once when provisioning the cluster, not per deployment; without a terminal it refuses rather than half-working. |
| `prepare host` | docker/podman | Create/own the data dir, verify DNS, and (HA) generate the redundancy PSK. |
| `prepare all` | all | Run the prepare steps needed every time, in order: namespace then secrets on Kubernetes, the host on docker/podman. Nothing here asks a question, so it is safe to script; `prepare labels` is excluded for exactly that reason. |
| `deploy broker [role] --restart` | all | Deploy the broker. Kubernetes: render and apply the PubSubPlusEventBroker custom resource (no `[role]`). Docker/podman: supply this host's secrets, render the deploy artifact, and start the container/service (`[role]` required in HA); `--restart` pre-approves bouncing an already-running broker. Re-runnable on docker/podman -- see [Re-deploying is safe and explicit](#re-deploying-is-safe-and-explicit) below. |
| `deploy operator` | kubernetes | Install the cluster-scoped EventBroker Operator, once per cluster. Not run by `deploy all`. |
| `deploy all [role] --restart` | all | Orchestrate the whole bring-up. Kubernetes: check -> prepare -> deploy -> assert the config-sync leader (HA); no `[role]`; never installs the operator. Docker/podman: check -> prepare host -> deploy `<role>`; `--restart` pre-approves bouncing an already-running broker whose artifact changed. |

### Post-deployment configuration

| Command | Platform | Description |
| --- | --- | --- |
| `config leader [role]` | all | Assert the config-sync leader (HA only). Kubernetes: no `[role]`. Docker/podman: primary-only -- fails loud on backup/monitor; `[role]` is this host, detected from its name when omitted. |
| `config apply server-cert` | all | Load/update the TLS server certificate. |
| `config apply domain-certs` | all | Load the configured domain CA certificates. |
| `config apply product-keys` | all | Apply the configured product keys. |
| `config apply additional-users` | kubernetes | Create the `admin.additionalUsers` CLI users over the broker CLI. **Not re-runnable** -- see [Extra CLI users differ by platform](#extra-cli-users-differ-by-platform). Docker/podman create these same users at container boot instead, so there is no equivalent command there. |
| `config delete domain-certs` | all | Remove the configured domain CA certificates -- the only step under `config` that can be undone. |
| `config disable default-vpn` | all | Shut down the default message-VPN. One-way: no command re-enables it. |
| `config disable default-users` | all | Shut down default client-usernames in all VPNs. One-way. |

There is deliberately no run-everything `config` step; see
[Post-deployment configuration order](#post-deployment-configuration-order) for the sequence
that works on a fresh broker.

#### Post-deployment configuration order

Each step talks to a broker that is already deployed and running, over its own CLI, and
they are not uniformly re-runnable -- `config apply additional-users` fails outright on a
user that already exists -- so the order lives here rather than in a run-everything command
that would stop partway through a second run. On a fresh broker:

1. `config leader` (HA only; on containers, run it on the primary)
2. `config apply server-cert` (when TLS is configured)
3. `config apply domain-certs` (when any are listed)
4. `config disable default-vpn`
5. `config disable default-users`
6. `config apply additional-users` (Kubernetes only; run after the hardening steps above,
   and **not re-runnable** -- the broker refuses to create a user that already exists)
7. `config apply product-keys` (when any are listed)

Only step 3 can be undone from here (`config delete domain-certs`). There is no un-harden,
and no way to withdraw a server certificate or a product key through this tool.

### Day-2 operations

| Command | Platform | Description |
| --- | --- | --- |
| `start broker` | all | Start a broker that is deployed but not running. Kubernetes: scale the statefulset(s) to 1. Docker/podman: start the container. |
| `stop broker` | all | Stop a running broker without removing it -- the deployment, its persistent data and its configuration all survive. Kubernetes: scale to 0. Docker/podman: stop the container. |
| `restart broker [role]` | all | Bounce a running broker; applies nothing new (a changed deploy artifact needs `deploy broker` first). Kubernetes: delete a pod so the statefulset recreates it, the step a `manualPodRestart` upgrade needs -- no role restarts every pod in the safe order (monitor, backup, primary). Docker/podman: no `[role]`; restarts this host's container in place. |
| `restart operator` | kubernetes | Restart the operator's controller deployment. |
| `status broker [role] --all --detail` | all | Show the broker's deployment status: pods, services, and statefulset on Kubernetes; the local container/service on docker/podman. `--all` (kubernetes only) reports every broker in the cluster instead of just this env file's. `--detail` adds the static artifacts -- full description, load balancer included. |
| `status operator --detail` | kubernetes | Show the operator's controller status; `--detail` adds the full deployment description. |
| `logs broker [role]` | all | Tail broker logs. Kubernetes: `[role]` picks the pod. Docker/podman: no `[role]` (always this host's container). |
| `logs operator` | kubernetes | Tail the operator's controller logs. |
| `cli [role] --input/-i <file> --pod` | all | Open an interactive Solace CLI in the broker. `--input` runs a script through it instead of opening a session (a bare filename resolves under `broker.cliScriptsFolder`). Kubernetes: `[role]` picks the pod, `--pod` overrides it for `--input`. Docker/podman: no `[role]`/`--pod`. |
| `shell [role]` | all | Open an interactive shell in the broker. Kubernetes: `[role]` picks the pod. Docker/podman: no `[role]`. |
| `copy from files...` | all | Copy files from the broker to the host. `--pod <role>` (kubernetes only). |
| `copy into files...` | all | Copy files from the host into the broker. `--pod <role>` (kubernetes only), `--dir <dest>`. |
| `diagnostics --days <n>` | all | Gather show-command output and a diagnostics bundle into `broker.diagDir`. `--days` sets the window (default 1). |

### Rendering artifacts (`generate`)

`generate <target>` is the only way to see an artifact before applying it -- there is no
`--dry-run` and no per-command render flag. See
[Rendering without applying](#rendering-without-applying) below.

| Command | Platform | Description |
| --- | --- | --- |
| `generate broker [role]` | all | Render what `deploy broker` would apply: the PubSubPlusEventBroker custom resource on Kubernetes, or this host's compose file / systemd quadlet unit on docker and podman. The container artifact is per-host, so `[role]` selects which node's to render (rejected on Kubernetes, where one CR covers the group). |
| `generate operator` | kubernetes | Render the operator install bundle. There is no container operator, so this is scoped because the thing does not exist elsewhere -- not because it goes by another name. |
| `generate secrets` | all | Render the secret-creation artifact: Kubernetes Secret manifests, or (docker/podman) a shell script -- `podman secret create` commands, or `export` lines for `docker compose` to read. **Prints secret values.** |

### Removal

| Command | Platform | Description |
| --- | --- | --- |
| `remove broker --delete-data --no-prompt` | all | Remove the deployed broker. Kubernetes: the CR. Docker/podman: stop and remove the container/unit. Keeps persistent data by default -- see [Removing a broker: what stays, what goes](#removing-a-broker-what-stays-what-goes). |
| `remove operator --delete-crd --no-prompt` | kubernetes | Remove the cluster-scoped EventBroker Operator. Keeps the CRDs by default -- deleting them cascades to every PubSubPlusEventBroker in the cluster. |
| `remove secrets` | kubernetes | Delete the broker's secrets. |
| `remove namespace` | kubernetes | Delete the broker's namespace. |
| `remove all --delete-data --no-prompt` | all | Remove the broker, its secrets and its namespace. Leaves the operator installed -- it is cluster-scoped and may serve other brokers. Docker/podman: equivalent to `remove broker`, there being no layer above it. |

### Other

| Command | Description |
| --- | --- |
| `convert <bash-env-file> -o/--out --force --platform` | Convert a legacy bash env file into a YAML env file. See [Migrating from the bash env files](#migrating-from-the-bash-env-files-solace-util-convert). |
| `auto-complete {bash,zsh,fish,powershell}` | Print a shell completion script. See [Shell completion](#shell-completion). |
| `version` | Print the stamped version, Go toolchain, and OS/arch this binary was built for. |

### Docker and Podman mechanics

The `docker` and `podman` halves of the tree above share one implementation for a
**host-local** broker: one container per host, driven over `<runtime> exec`/`cp` (no
operator, no cluster). Only the deploy artifact differs -- Docker renders a compose file
and brings it up with `docker compose`, Podman a systemd **quadlet** `.container` unit.
(Docker always deploys through compose -- there is no `docker.mode` key. A bare `docker run`
cannot recreate an existing container, so re-deploying after an image-tag bump would fail on
a name conflict where compose recreates cleanly; that mismatch is why the older
`docker.mode: run` option, and now the `docker.mode` key itself, were removed. An env file
still carrying the key fails strict decoding as an unknown field.)

**HA is a two-host handshake.** The transport is node-local, so one invocation drives only
this host's broker. Bring the group up by running `deploy all <role>` on each host with its
own role, then run `smoke redundancy` on the **primary and backup concurrently**: the
primary releases activity and waits to fail back; the backup takes over, dwells ~10s, and
reverts. The monitor cannot run `smoke redundancy` (rejected loud), and `config leader`
runs only on the primary. Running `smoke redundancy` on a single host times out (bounded by
the poll budget) rather than hanging.

#### Re-deploying is safe and explicit

`deploy broker` renders the artifact and compares it with
what is already on disk, so the three outcomes are distinguishable:

- **Unchanged, broker running** -- reported as nothing to do; the broker is not touched.
  With `--restart` it is recreated/restarted anyway, which is how a rotated secret is
  applied (nothing in the artifact changes when a password does).
- **Broker not running** -- recreated from the current config, so a rotated secret takes
  effect without `--restart`. A stopped container would otherwise be *started* with the
  credentials it was created with; there is no traffic to protect here, so no consent is
  asked. `--restart` is only needed for a running broker, which is the case where applying
  a change costs a bounce. Confirmed on Docker (`--force-recreate`); on Podman this relies
  on quadlet's own container replacement at unit start, which is assumed but has not been
  independently verified.
- **Changed, broker not running** -- written and started.
- **Changed, broker running** -- written, then you are asked before it is bounced.
  `--restart` pre-approves; a non-interactive run declines, leaving the new artifact in
  place and warning that the running broker is still on the previous one. `--restart` is
  deliberately its own flag: dropping messaging traffic is its own decision.

This is what makes an image-tag bump a one-command upgrade: edit `image.tag`, then
`solace-util deploy broker <role> --restart -e prod.yaml` (podman) or
`solace-util deploy broker --restart -e prod.yaml` (docker). Podman needs this because
`systemctl start` on an already-active unit is a no-op -- the old behaviour rewrote the unit,
reported success, and left the previous image running.

**Secrets.** `deploy broker` externalizes every secret before applying the artifact, so no
value is ever written into the compose file or quadlet unit (see the table under
[Rendering without applying](#rendering-without-applying) for the names and
paths). Podman loads them into its own secret store (`podman secret create --replace`,
value on stdin) and the unit mounts them; Docker's compose file names a host environment
variable per secret, and `deploy broker` sets those variables for its own `docker compose`
process, so no value ever reaches an argv or a file beside the compose file. A missing
value (notably `nodes.psk` before `prepare host` has run) fails the deploy loudly rather
than starting a broker without a password.

What that does and does not buy you: **nothing is written next to the artifact** (no
plaintext file a project-directory backup, `tar`, or non-root user would pick up, and
nothing to clean up on teardown), but the value still ends up at rest -- Docker
materializes each secret into the container's own filesystem as a `0444` root-owned file
under `/run/secrets`, which is the same at-rest exposure class as podman's store. It is
not in the container's environment, so `docker inspect` does not show it.

`generate secrets` prints the equivalent shell, one line per secret, for running compose
yourself: `podman secret create` commands to run once on Podman, and `export` lines to
**source** in the shell you run `docker compose` from on Docker. A manual
`docker compose up` needs those variables exported -- unset, compose refuses.

**Rotating a secret** takes `--restart`. A new password or PSK changes no artifact (its
value lives in the config and, for Docker, only in the environment), so the ordinary
"unchanged, nothing to do" path cannot see it; `deploy broker --restart` recreates the
container (Docker) or restarts the service (Podman) to pick it up, and the no-op message
names that as the way to apply one.

**Config source.** The container platform has no separate config namespace: its post-deploy
`config` steps read the platform-neutral `broker.*` fields -- `broker.domainCerts`,
`broker.productKeys`, `broker.diagDir`, `broker.cliScriptsFolder` -- shared verbatim with
Kubernetes, plus `tls.cert`/`tls.certKey` (server certificate) and `admin.user`/`admin.pass`
(SEMP login); the `nodes.*` names drive role detection for `config leader` /
`smoke redundancy`. The rest of the `nodes.*` table and `nodes.psk` are consumed earlier,
at `prepare`/`deploy` (`prepare host` generates the PSK and writes it back to the env file;
`deploy broker` externalizes it as a secret). Container-only knobs live under `docker.*` /
`podman.*` (runtime, compose invocation, container name, data dir, network mode, rootless).
See `env/sample.yaml`.

Example (HA -- run each line on the matching host; `prod.yaml` is a podman env file):

```
solace-util deploy all primary -e prod.yaml   # on the primary host
solace-util deploy all backup  -e prod.yaml   # on the backup host
solace-util deploy all monitor -e prod.yaml   # on the monitor host
solace-util config leader -e prod.yaml        # on the primary only
# then, concurrently on the primary and backup hosts:
solace-util smoke redundancy -e prod.yaml
```

## Bringing up a fresh cluster

The EventBroker operator is cluster-scoped and shared between brokers, so it is installed
and removed on its own rather than as a side effect of any one broker's `deploy all` or
`remove all`. A cluster that has never run this tool (or any other operator install) needs
it once:

```
solace-util deploy operator -e dev.yaml
```

After that, any number of env files can each `deploy all` their own broker against the same
cluster without touching the operator again. `check deploy` warns rather than fails when
the operator or its CRD looks missing, so that warning is what tells you this step was
skipped -- the actual `deploy broker` (or `deploy all`) then fails once it tries to apply a
custom resource the cluster does not know how to reconcile.

End to end, a first run against a brand-new cluster looks like:

```
solace-util deploy operator -e dev.yaml     # once per cluster
solace-util check deploy -e dev.yaml        # prerequisites: cluster, StorageClass, operator
solace-util deploy all -e dev.yaml          # check -> namespace -> secrets -> CR -> leader (HA)
solace-util check semp-login -e dev.yaml    # prove it answers
```

Removing a broker (`remove broker` / `remove all`) never removes the operator either -- see
[Removing a broker: what stays, what goes](#removing-a-broker-what-stays-what-goes) below.
Uninstall it explicitly, and only once nothing else in the cluster still depends on it:

```
solace-util remove operator -e dev.yaml
```

## Removing a broker: what stays, what goes

`remove broker`, `remove operator` and `remove all` are the destructive commands, and each
one keeps the layer that is expensive or impossible to get back **by default**: `remove
broker` / `remove all` keep the broker's persistent data (Kubernetes PVCs, or the
container's data directory); `remove operator` keeps the operator's CRDs. Deleting the CRDs
is the sharper of the two, because they are cluster-wide -- it cascade-deletes **every**
PubSubPlusEventBroker in the cluster, including ones this env file has never heard of, not
just the one it describes. `remove all` also leaves the operator itself installed, for the
same reason: it is cluster-scoped and may be serving other brokers, so removing it is always
its own explicit command (see [Bringing up a fresh cluster](#bringing-up-a-fresh-cluster)).

Both removals ask about their retained layer the same way, so learning the contract on one
teaches the other:

- **`--delete-data`** (on `remove broker` / `remove all`) or **`--delete-crd`** (on `remove
  operator`) deletes the layer without asking.
- **`--no-prompt`** asks nothing at all: it confirms the removal and takes the safe answer
  to the layer question, so the layer is kept unless a `--delete-*` flag says otherwise.
  The two flags answer different questions and deliberately **compose** -- a fully
  unattended removal that also drops the data is `--delete-data --no-prompt`.
- **Interactively, with neither flag**, you are prompted and told what the layer is and
  what deleting it costs; only an exact, case-insensitive `yes` deletes it, and anything
  else -- including the lenient `y` that answers the removal prompt below -- keeps it.
- **Non-interactively with neither flag**, the layer is kept. A scripted or piped removal
  can never lose data by omission.
- **Either way, the outcome is printed** -- what was kept, or what was deleted -- so it is
  never left to be inferred from silence.

This is a second, independent decision from *whether to remove the broker (or operator) at
all*. **Every command that destroys something confirms first** -- `remove broker`,
`remove operator`, `remove secrets`, `remove namespace`, `remove all`, and `restart broker`,
which deletes pods. An interactive terminal is asked `[y/N]`; a non-interactive session
without `--no-prompt` refuses loudly rather than destroying anything unattended.

`--no-prompt` is the one flag that silences all of it, so a script switches off one thing
rather than one per question. It still cannot lose you data on its own: the layer stays
unless `--delete-data`/`--delete-crd` names it. Two flags, two questions, on purpose --
dropping messaging data and cascading a CRD deletion across the cluster are each too costly
to answer as a side effect of the other.

## Rendering without applying

`generate <target>` is the one, dedicated way to review the exact artifact a command would
apply before it touches a cluster or a host -- there is no `--dry-run` and no per-command
render flag any more. It prints to stdout and changes nothing: it runs no external command
at all, which is what makes it the safe way to inspect an env file you did not write (see
[The command fields are executable content](#the-command-fields-are-executable-content)):

```
solace-util generate broker -e dev.yaml                          # the PubSubPlusEventBroker CR (kubernetes)
solace-util generate operator -e dev.yaml                        # the operator bundle (kubernetes)
solace-util generate secrets -e dev.yaml                         # the Secret manifests (kubernetes; secret values!)
solace-util generate broker primary -e dev.yaml --platform docker    # the compose file
solace-util generate broker primary -e dev.yaml --platform podman    # the quadlet unit
solace-util generate secrets -e dev.yaml --platform docker           # commands that supply the secrets
```

`generate` is a command with a named target rather than a flag on `deploy broker`, on
purpose: an artifact you meant to inspect and a cluster you meant to change should not be
one typo apart. It runs no external command on any platform, so pointing it at an env file
you did not write cannot run anything -- see
[The command fields are executable content](#the-command-fields-are-executable-content).

**Secrets are never part of a deployment artifact.** Each one lives in podman's secret
store, in a host environment variable the compose file names, or in a Kubernetes Secret --
and the quadlet unit, compose file, and CR reference it by name only. So `generate broker`
and `generate operator` output is safe to review, diff, and share, while **`generate
secrets` prints the values themselves** and must be handled exactly like the env file.

The broker's own settings -- routername, redundancy, the scaling knobs -- are inlined into
whatever `generate broker` renders: `Environment=` lines in a quadlet unit, an
`environment:` block in a compose file, `spec.systemScaling` in the CR. There is no
separate command for them, because there is no separate artifact.

**Every platform hands the broker its secrets as files**, read through the setting's
`*filepath` variant, and named after the setting they feed -- so the layout inside a
container matches the data keys of the equivalent Kubernetes Secret:

| Secret | In-container path | Host-side name |
| --- | --- | --- |
| `admin.pass` | `/run/secrets/username_<admin.user>_password` | `<container.name>-admin-password` |
| `admin.additionalUsers[].password` | `/run/secrets/username_<username>_password` | `<container.name>-user-<username>-password` |
| `nodes.psk` (HA) | `/run/secrets/redundancy_authentication_presharedkey_key` | `<container.name>-redundancy-psk` |
| `tls.certPassphrase` | `/run/secrets/tls_servercertificate_passphrase` | `<container.name>-tls-passphrase` |

The host-side name carries `container.name` (default `solace`) so two brokers on one host
never share a podman store entry or a compose variable. On Kubernetes the operator mounts
the credentials Secret itself, so the only data keys that matter are
`username_admin_password` and `username_monitor_password`.

### Extra CLI users differ by platform

`admin.additionalUsers` reaches the broker two different ways, because the operator has no
declarative route for it:

- **Docker / Podman** -- created at container boot, from the mounted password file plus a
  `username_<username>_globalaccesslevel` setting in the artifact. Nothing to run
  afterwards.
- **Kubernetes** -- created post-deployment by **`solace-util config apply additional-users`**,
  which builds a Solace CLI script and runs it on the primary. Verified against a live
  cluster: extra `username_<user>_password` keys in the credentials Secret are **ignored by
  the operator**, and the only declarative alternative (`extraEnvVars` /
  `extraEnvVarsSecret`) would publish the passwords in the pod's environment, where
  `kubectl describe` and every process in the container can read them. So the CLI is the
  route, and the Secret carries no extra users at all.

Two consequences on Kubernetes worth knowing:

- **It is not re-runnable.** The broker's `create username` fails if the user exists, and
  that is reported rather than reconciled -- re-setting a password an operator rotated on
  the broker would be worse. So a repeated `config apply additional-users` fails once the
  users exist; run the other `config` steps in
  [Post-deployment configuration order](#post-deployment-configuration-order) individually
  instead, or drop the already-created users from the env file.
- **The password charset is restricted.** The value goes onto a CLI line, and the broker
  rejects ``:()";'<>,`\*&|`` inside it. An env file using one of those fails to load *for
  Kubernetes* with the offending character named (never the password). The same file stays valid
  for docker and podman, which write the password to a file instead.

## Upgrading a running broker

Changing the image tag (or any other setting) is the same edit on every platform --
bump `image.tag` in the env file -- but applying it differs:

**Kubernetes, `updateStrategy: automatedRolling` (the default)**

```
solace-util deploy broker -e dev.yaml
```

`deploy broker` re-applies the custom resource; the operator sees the new tag and rolls the
pods itself (monitor, then backup, then the active node).

**Kubernetes, `updateStrategy: manualPodRestart`**

```
solace-util deploy broker -e dev.yaml     # updates the statefulset template; no pod is touched
solace-util restart broker -e dev.yaml    # bounces monitor -> backup -> primary, waiting for each
```

The operator deliberately waits for you here, so `deploy broker` alone changes nothing
visible. `restart broker <role>` bounces one pod if you would rather drive the order
yourself -- worth doing after a failover, since the order above is by configured
role and the active node may not be the configured primary. Check with
`solace-util smoke redundancy` first.

**Docker / Podman** (on each host, with its own role)

```
solace-util deploy broker primary -e prod.yaml --restart    # prod.yaml: a podman env file
solace-util deploy broker -e prod.yaml --restart             # prod.yaml: a docker env file
```

`deploy broker` compares the rendered artifact with the one on disk: unchanged is a no-op,
changed is written and then applied to the running broker -- with `--restart`, or
after being asked. Without consent the new artifact is left in place and the command
says the broker is still on the previous one. In an HA group, upgrade the monitor and
backup before the primary.

## Development

`scripts/dev.ps1` and `scripts/dev.sh` are behaviourally identical and own every
build/test/scan command. The workflows call task names only, so local runs match CI:

| Task | Does |
| --- | --- |
| `tidy` | `go mod tidy` |
| `vet` | `go vet ./...` |
| `build` | Compile -> `dist/solace-util-<os>-<arch>[.exe]`. `TARGET_OS`/`TARGET_ARCH` pick the target; unset means the host. Stamps `solace-util version` from `git describe` (falls back to `dev`) |
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
`coverage/coverage.html`. Current test coverage is 92.3% (recorded in
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
| `internal/engine` | External-command runner (real exec; an echo variant used as a test seam; secrets never echoed). |
| `internal/render` | Templating for the broker CR, operator bundle, compose/run/quadlet artifacts. |
| `internal/broker` | Platform-agnostic config/verify operations over an injected transport. |
| `internal/k8s` | Kubernetes cluster/operator operations and the kubectl transport. |
| `internal/container` | Docker/Podman host operations (Manager) and the node-local `<runtime> exec`/`cp` transport. |
| `internal/convert` | Legacy bash env -> YAML converter behind `solace-util convert`. |
| `internal/cli` | Cobra command tree and handlers. |
| `internal/tools/vulnjudge` | Dev-only judge the `scan` task pipes govulncheck JSON through. |
| `env/` | Config files (`sample.yaml` is the annotated template). |
| `docs/` | [commands.md](docs/commands.md) -- generated CLI reference; [test.md](docs/test.md) -- the catalogue of every test. |
| `scripts/` | `dev.ps1` / `dev.sh` developer tooling. |
