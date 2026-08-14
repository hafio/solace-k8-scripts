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
  to use `oc` instead, or to carry a whole profile such as
  `kubectl --kubeconfig /path/.kubeconfig-dev`. A wrapper like `microk8s kubectl` needs
  `--allow-command` -- see **The command fields are executable content** below.
- **Run (Docker/Podman):** the `docker` or `podman` binary on your `PATH`, on the host that
  runs the broker. Podman deploys a systemd **quadlet** unit and uses podman's own secret
  store (its host also needs systemd); Docker deploys through **compose**, so that host also
  needs the compose plugin (`docker compose`) or the standalone `docker-compose` binary --
  set `docker.compose` when it is the latter. The binary shells out to the runtime; it embeds
  no container client. `docker.runtime` / `podman.runtime` override the command the same way.
- **Podman secrets:** `deploy` stores the broker's secrets with
  `podman secret create --replace` and mounts them into the container
  (`type=mount`), neither of which the oldest podman builds support. Confirm yours does
  (`podman secret create --help | grep -- --replace`) -- `check` only proves the runtime
  answers `version`, so an unsupported flag surfaces at deploy time.
- **Docker compose secrets:** the generated compose file sources each secret from a host
  environment variable, which needs **compose v2.23.1 or later** (`docker compose version`).
  On an older compose the `environment:` secret source is not understood and `up` fails.
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
and never commit it. (Switch the values to their `*Env` reference keys afterwards and it
becomes safe to commit; see **Secrets** below.) `SOLBK_USR_SECRET` converts to
`k8s.adminSecret`, and each `SOLBK_USR_PASS` entry to an `admin.additionalUsers` entry with
`accessLevel: none` -- the bash flow set no level, so the converter picks the least
privileged one and says so; raise it per user as needed.

**Renamed keys.** An env file written for an earlier version fails loud on the unknown key
(strict decoding), so the two renames are visible immediately rather than silently ignored:
`admin.userSecret` is now `k8s.adminSecret`, and the `admin.userPasswords`
`["user=password"]` list is now `admin.additionalUsers`.

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
| `redundancy` | `no` | `yes` = HA group (primary+backup+monitor); `no` = single standalone broker. HA provisions three brokers, so it must be asked for explicitly |
| `image.registry` | docker.io | Registry prefix for the image reference |
| `k8s.storage.class` | cluster default | StorageClass for the broker PVCs |
| `k8s.updateStrategy` | `automatedRolling` | `automatedRolling` or `manualPodRestart` |
| `k8s.runtime` | `kubectl` | Cluster CLI (legacy `KUBE`). A scalar is split on whitespace, so it can be a drop-in (`oc`) or a profile (`kubectl --kubeconfig <file>`). **Restricted** -- see [The command fields are executable content](#the-command-fields-are-executable-content) |
| `docker.runtime` / `podman.runtime` | `docker` / `podman` | Container CLI (legacy `CONTAINER_RUNTIME`), same forms and the same restrictions as `k8s.runtime` |
| `docker.compose` | `<runtime> compose` | The compose invocation. Set it to `docker-compose` on a host carrying only the standalone v1 binary; same forms and restrictions as `runtime`, plus the one permitted `compose` subcommand |
| `<docker\|podman>.container.healthCheck.enabled` | `false` | Adds an engine health check polling the broker's own `/health-check/readiness` on port 5550 every 5s, so `docker ps` and podman's auto-restart see readiness rather than liveness. Needs broker **10.26 or later** and a version-numbered `image.tag`; set `healthCheck.cmd` to supply your own probe instead (which skips the version check). Container-only by design -- on Kubernetes the operator already probes the pods |
| `tls.serverSecret` | -- | Name of the TLS secret; its presence enables the CR's TLS block |
| `k8s.adminSecret` | `solace-admin-secret` | Name of the Kubernetes Secret holding the admin/monitor credentials. Was `admin.userSecret` |
| `admin.additionalUsers` | -- | Extra CLI (management) users, each `{username, accessLevel, password\|passwordEnv}` with `accessLevel` one of `none`, `read-only`, `mesh-manager`, `read-write`, `admin`. Replaces the old `admin.userPasswords` `user=password` list. Created at boot on containers, and by `k8s config additional-users` on Kubernetes -- see below |
| `admin.passEnv` (and every other `*Env`) | -- | Name of an environment variable holding the secret, instead of the value itself. See **Secrets** below |
| `timezone` | -- | Broker timezone, all platforms (the CR's `timezone` and the containers' `TZ`). Omitted keeps the image default |
| `k8s.securityContext` | -- | `runAsUser`/`fsGroup` for the pod. Omitted entirely when unset |
| `k8s.containerSecurity` | -- | `runAsUser`/`runAsGroup`/`readOnlyRootFilesystem` for the broker container |

### The command fields are executable content

`k8s.runtime`, `docker.runtime`, `podman.runtime` and `docker.compose` each name a binary
this tool runs **on your machine**. Env files travel -- repositories, pull requests, shared
archives -- so the person who wrote one is routinely not the person who runs it. Treat an
env file the way you would treat a script someone sent you: **read the command fields before
running anything with it.**

To make that review short, the fields are restricted. A command is accepted only when:

1. **Every token is inert and visible.** No control characters, no whitespace inside a single
   argument (any Unicode whitespace, not just the ASCII space), no invisible formatting
   characters (zero-width spaces and joiners, bidirectional overrides), no quotes, no
   backslash, no backtick, and none of `$ ; | & < > ( ) * ? [ ] { } ~ # !`. Nothing is ever
   passed through a shell, so these are not injections -- but tokens end up in logs,
   `--dry-run` output and generated files, and a token you cannot see is one you cannot
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
solace k8s deploy --allow-command microk8s
solace docker up --allow-command lima
```

`--allow-command` is repeatable, takes a bare name (never a path), and exists **only** as a
command-line flag. There is deliberately no env-file key, environment variable, or any other
way for a config to widen its own allowlist: the authority to run something unusual belongs
to the person who can see what they are approving. It is rejected on `gen` and on any
`--gen-*-only` run, where nothing executes.

**Privilege escalation is never approvable**, by the config or by you: `sudo`, `doas`, `su`,
`pkexec`, `run0`, `runas` and `gsudo` are refused as `--allow-command` values. This is not a
ban on running as root -- rootful podman needs it. It is about *where* you elevate. A
`runtime: sudo podman` elevates every command this tool issues, for the whole life of an env
file, decided by whoever wrote that file. Elevate the tool instead, at the moment you run it,
so the privilege belongs to one invocation you chose:

```sh
sudo solace podman deploy -e prod.yaml     # yes
# runtime: sudo podman  in the env file    # never
```

The same check runs twice -- once when the env file is loaded, and again immediately before
any command line is built -- from a single implementation, so a hostile file is inert even
on a path that skipped validation.

**What this does not protect against.** Two things are out of scope, and no amount of
parsing would fix either:

- **A compromised machine.** If an attacker has already put a trojan `kubectl` on your
  `PATH`, they own the host; nothing this tool checks can help. What it does do is make the
  binary's real location visible -- every execution prints `exec: <resolved path> <args>` to
  stderr before it runs -- and refuse to resolve a bare name from the current directory.
- **Config that is malicious but perfectly legitimate in form.** `k8s.namespace: production`,
  or a valid `kubectl --context` aimed at the wrong cluster, is a review problem. So is a
  flag's *value*: this tool cannot know how many arguments a flag takes, so the token after
  `--kubeconfig` is accepted as that flag's value whatever it says. The hard guarantee covers
  the binary and every bare word -- not flag values.

Rendering executes nothing: `gen` and the `--gen-*-only` flags never run an external command,
so pointing them at an env file you did not write cannot run anything. Note that they still
*load* the file, so one whose command field breaks the rules above fails there rather than
printing an artifact -- which is itself the answer you wanted about that file. To read a
command field without loading anything at all, open the file.

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
`prep host` only generates a key when the literal `nodes.psk` is empty, so with the
reference form create it yourself (`openssl rand -base64 60`) and export the same value on
all three hosts.

The tool never echoes a secret. Values piped to a command on stdin show as
`<<< (N bytes on stdin)` under `--dry-run`, values passed to a child process's environment
as `NAME=***`, and `check`/`status` report only whether each one is set. The two
exceptions are explicit: `--gen-secrets-only` and `k8s gen secrets` print values, because
printing them is what those flags are for.

## Global flags

These apply to every subcommand:

| Flag | Default | Meaning |
| --- | --- | --- |
| `-e`, `--env <file>` | `env.yaml` | Env file to load: a file name searched in the base dir then `<base-dir>/env`, or a path used as-is. |
| `--base-dir <dir>` | current dir | Directory searched for the env file, and holding `env/`. |
| `--gen-only` | `false` | Render the deployment artifact this command would apply and print it; change nothing. |
| `--gen-secrets-only` | `false` | Render this deployment's secrets and print them; change nothing -- k8s Secret manifests, podman `secret create` commands, docker `export` lines to source. **Prints secret values.** |
| `--gen-env-only` | `false` | Render the container broker settings as an env file and print them; change nothing (docker/podman only). |
| `--dry-run` | `false` | Print the external commands instead of running them. |
| `-y`, `--yes` | `false` | Skip confirmation prompts. Does **not** imply `--purge`. |

On the `k8s`, `docker` and `podman` trees only:

| Flag | Default | Meaning |
| --- | --- | --- |
| `--allow-command <name>` | -- | Approve one extra binary for this env file's platform command, for this run only. Repeatable; a bare name, never a path. See [The command fields are executable content](#the-command-fields-are-executable-content). |

The three `--gen-*-only` flags are mutually exclusive (passing two is an error) and are
valid only on commands that render an artifact; passing one to any other command --
`delete` above all -- is rejected loudly rather than ignored, on every platform.
`--allow-command` is the mirror image: it is rejected on `gen` and on any `--gen-*-only`
run, because nothing executes there and a flag that is harmless everywhere gets pasted
everywhere.

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
there is no flag to skip it: the one legitimate reason to skip it is previewing without a
cluster, which is what `--dry-run` is for (there the probe is echoed and its answer
skipped, like the other cluster checks).

## Command reference (Kubernetes)

The tables below group the commands by lifecycle phase. For the **complete** surface --
every command, its arguments, and every flag with its default -- see
[docs/commands.md](docs/commands.md), which is generated from the command tree itself.

Run `solace k8s --help` (or `--help` on any subcommand) for the live tree. A `[role]`
positional accepts `p`|`b`|`m` or `primary`|`backup`|`monitor` and defaults to primary.

The verbs fall into two halves. `check`, `prep` and `deploy` build the deployment and are
what `up` orchestrates. Everything under **`config` and `verify` is post-deployment**: each
step drives the Solace CLI inside a broker that is already running, so none of it is part
of `deploy` and none of it is run by `up` -- wait for the pods to be ready, then run it.

### Lifecycle

| Command | Description |
| --- | --- |
| `check` | Validate config, cluster reachability, and StorageClass. |
| `prep` | Prepare cluster prerequisites; with no subcommand runs all applicable steps in order. |
| `prep operator` | Install the EventBroker Operator (honors `--gen-only`). |
| `prep namespace` | Create the broker namespace. |
| `prep secrets` | Create admin/monitor, TLS, and image-pull secrets (honors `--gen-secrets-only`). |
| `prep labels` | Label nodes for primary/backup/monitor placement (interactive). |
| `deploy` | Render and apply the PubSubPlusEventBroker custom resource (honors the `--gen-*-only` trio). `--keep-yaml` keeps the rendered manifest on disk. |
| `config` | **Post-deployment** configuration, applied over the Solace CLI to a broker that is already deployed and running. Not part of `deploy`, and not run by `up`. With no subcommand runs all applicable steps in order. |
| `config leader` | Assert the config-sync leader (HA only). |
| `config server-cert` | Load/update the TLS server certificate. |
| `config domain-certs` | Load domain CA certificates. |
| `config disable-default-vpn` | Shut down the default message-VPN. |
| `config disable-default-users` | Shut down default client-usernames in all VPNs. |
| `config additional-users` | Create the `admin.additionalUsers` CLI users over the broker CLI. **Not re-runnable** -- see **Extra CLI users differ by platform**. |
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
| `describe broker [role]` | Describe a broker pod. Also spelled `inspect`. |
| `describe lb` | Describe the load-balancer service. |
| `copy from files...` | Copy files from a broker pod to the host. `--pod <role>`. |
| `copy into files...` | Copy files from the host into a broker pod. `--pod <role>`, `--dir <dest>`. |
| `replicas start` | Scale broker statefulset(s) to 1. |
| `replicas stop` | Scale broker statefulset(s) to 0. |
| `restart [role]` | Delete a broker pod so the statefulset recreates it -- the step a `manualPodRestart` upgrade needs. No role restarts all of them in order (monitor, backup, primary), waiting for each. |
| `operator deploy` | Install the operator from the embedded bundle (honors `--gen-only`). |
| `operator delete` | Remove the operator. |
| `operator status` | Show operator deployment/pod status. |
| `operator logs` | Tail operator logs. |
| `operator describe` | Describe the operator deployment. |
| `gen [broker\|operator\|secrets]` | Render a manifest to stdout without applying (default `broker`). |
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

To review the exact artifact before it touches a cluster or a host, use a `--gen-*-only`
flag on an artifact-producing command, or the dedicated `gen` command. Both print to
stdout and change nothing -- they run no external command at all, which is what makes them
the safe way to inspect an env file you did not write (see
[The command fields are executable content](#the-command-fields-are-executable-content)):

```
solace k8s gen broker -e dev.yaml                 # the PubSubPlusEventBroker CR
solace k8s gen operator -e dev.yaml               # the operator bundle
solace k8s gen secrets -e dev.yaml                # the Secret manifests (secret values!)
solace k8s deploy -e dev.yaml --gen-only          # equivalent to gen broker
solace k8s operator deploy -e dev.yaml --gen-only # equivalent to gen operator
solace docker gen primary -e dev.yaml             # the compose file
solace podman gen primary -e dev.yaml             # the quadlet unit
solace docker gen primary -e dev.yaml --gen-env-only     # the broker settings, key=value
solace docker gen primary -e dev.yaml --gen-secrets-only # commands that supply the secrets
```

The flag selects the artifact, not the command: any artifact command honors all three.
They are valid only on artifact commands (`deploy`, `gen`, plus `prep secrets`,
`prep operator` and `operator deploy` on Kubernetes) -- using one elsewhere is rejected
with a clear error on every platform, so a gen flag can never turn into a real `delete`.

**Secrets are never part of a deployment artifact.** Each one lives in podman's secret
store, in a host environment variable the compose file names, or in a Kubernetes Secret --
and the quadlet unit, compose file, and CR reference it by name only. So `--gen-only`
output is safe to review, diff, and share, while **`--gen-secrets-only` prints the values
themselves** and must be handled exactly like the env file. `--gen-env-only` is
container-only; on Kubernetes the broker settings are part of the CR, so it is rejected
there.

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
- **Kubernetes** -- created post-deployment by **`solace k8s config additional-users`**,
  which builds a Solace CLI script and runs it on the primary. Verified against a live
  cluster: extra `username_<user>_password` keys in the credentials Secret are **ignored by
  the operator**, and the only declarative alternative (`extraEnvVars` /
  `extraEnvVarsSecret`) would publish the passwords in the pod's environment, where
  `kubectl describe` and every process in the container can read them. So the CLI is the
  route, and the Secret carries no extra users at all.

Two consequences on Kubernetes worth knowing:

- **It is not re-runnable.** The broker's `create username` fails if the user exists, and
  that is reported rather than reconciled -- re-setting a password an operator rotated on
  the broker would be worse. So a second `config` run stops at this step once the users
  exist; run the other steps individually, or drop the users from the env file.
- **The password charset is restricted.** The value goes onto a CLI line, and the broker
  rejects ``:()";'<>,`\*&|`` inside it. An env file using one of those fails to load *for
  k8s* with the offending character named (never the password). The same file stays valid
  for docker and podman, which write the password to a file instead.

## Command reference (Docker / Podman)

The `docker` and `podman` command trees mirror the Kubernetes verbs for a **host-local**
broker: one container per host, driven over `<runtime> exec`/`cp` (no operator, no
cluster). The two share one tree; only the deploy artifact differs -- Docker renders a
compose file and brings it up with `docker compose`, Podman a systemd **quadlet**
`.container` unit. (Docker's older `docker.mode: run` was removed: a bare `docker run`
cannot recreate an existing container, so re-deploying after an image-tag bump failed on a
name conflict where compose recreates cleanly. An env file still carrying it fails with
that explanation.) Run `solace docker --help` / `solace podman --help` for the live tree.

A `[primary|backup|monitor]` positional (also `p`|`b`|`m`) selects the redundancy role and
defaults to primary. In standalone mode (`redundancy: no`) it is ignored; in an HA group
pass it explicitly per host on `deploy`/`up`/`gen`. For `config leader` and
`verify redundancy` it is optional -- omitted, the role is auto-detected by matching the
host name against the `nodes.*` table.

As with Kubernetes, [docs/commands.md](docs/commands.md) carries the complete generated
reference for both trees. The same two halves apply: `check`, `prep host` and `deploy`
build the deployment and are what `up` orchestrates, while **`config` and `verify` are
post-deployment** -- they drive the Solace CLI inside a container that is already running,
so `up` does not touch them.

### Lifecycle

| Command | Description |
| --- | --- |
| `check` | Validate config, node-name DNS resolution, the container runtime, and (Docker) the compose command. |
| `prep` | Prepare the host; with no subcommand runs all steps. |
| `prep host` | Create/own the data dir, verify DNS, and (HA) generate the redundancy PSK. |
| `deploy [role]` | Supply the host's secrets, render the deploy artifact, and start the container/service. Re-runnable: an unchanged artifact is a no-op, a changed one asks before bouncing a running broker (`--restart` pre-approves, and on an unchanged artifact forces the recreate/restart that applies a rotated secret). Honors the `--gen-*-only` trio. |
| `config` | **Post-deployment** configuration, applied over the Solace CLI to a broker already running in this host's container. Not part of `deploy`, and not run by `up`. With no subcommand runs all applicable steps **except** `leader`. |
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
| `describe` | Detailed inspection (`<runtime> inspect`: health state, restart count, exit reason, mounts). Podman also shows the installed unit. Also spelled `inspect`. |
| `logs` | Tail the local broker container logs. |
| `cli` | Open an interactive Solace CLI in the container. |
| `shell` | Open an interactive shell in the container. |
| `copy from files...` | Copy files out of the broker container to the host. |
| `copy into files...` | Copy files from the host into the container. `--dir <dest>`. |
| `gen [role]` | Render the deploy artifact to stdout without applying (compose file for Docker, quadlet for Podman). Honors the `--gen-*-only` trio. |

### Teardown and orchestration

| Command | Description |
| --- | --- |
| `teardown domain-certs` | Remove domain CA certificates from the broker (the counterpart of `config domain-certs`). |
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

**Re-deploying is safe and explicit.** `deploy` renders the artifact and compares it with
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
  deliberately not `--yes`: dropping messaging traffic is its own decision.

This is what makes an image-tag bump a one-command upgrade: edit `image.tag`, then
`solace podman deploy <role> --restart` (or `solace docker deploy --restart`). Podman needs
this because `systemctl start` on an already-active unit is a no-op -- the old
behaviour rewrote the unit, reported success, and left the previous image running.

**Secrets.** `deploy` externalizes every secret before applying the artifact, so no value
is ever written into the compose file or quadlet unit (see the table under
[Rendering without applying](#rendering-without-applying) for the names and
paths). Podman loads them into its own secret store (`podman secret create --replace`,
value on stdin) and the unit mounts them; Docker's compose file names a host environment
variable per secret, and `deploy` sets those variables for its own `docker compose`
process, so no value ever reaches an argv or a file beside the compose file. A missing
value (notably `nodes.psk` before `prep host` has run) fails the deploy loudly rather
than starting a broker without a password.

What that does and does not buy you: **nothing is written next to the artifact** (no
plaintext file a project-directory backup, `tar`, or non-root user would pick up, and
nothing to clean up on teardown), but the value still ends up at rest -- Docker
materializes each secret into the container's own filesystem as a `0444` root-owned file
under `/run/secrets`, which is the same at-rest exposure class as podman's store. It is
not in the container's environment, so `docker inspect` does not show it.

`--gen-secrets-only` prints the equivalent shell, one line per secret, for running compose
yourself: `podman secret create` commands to run once on Podman, and `export` lines to
**source** in the shell you run `docker compose` from on Docker. A manual
`docker compose up` needs those variables exported -- unset, compose refuses.

**Rotating a secret** takes `--restart`. A new password or PSK changes no artifact (its
value lives in the config and, for Docker, only in the environment), so the ordinary
"unchanged, nothing to do" path cannot see it; `deploy --restart` recreates the container
(Docker) or restarts the service (Podman) to pick it up, and the no-op message names that
as the way to apply one.

**Config source.** The container platform has no separate config namespace: its post-deploy
`config`/`verify` steps read the shared `k8s.*` fields -- `k8s.domainCerts`,
`k8s.productKeys`, `k8s.diagDir`, `k8s.cliScriptsFolder` -- plus `tls.cert`/`tls.certKey`
(server certificate) and `admin.user`/`admin.pass` (SEMP login); the `nodes.*` names drive
role detection for `config leader` / `verify redundancy`. The rest of the `nodes.*` table
and `nodes.psk` are consumed earlier, at `prep`/`deploy` (`prep host` generates the PSK and
writes it back to the env file; `deploy` externalizes it as a secret). Container-only knobs live under `docker.*` / `podman.*`
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

## Upgrading a running broker

Changing the image tag (or any other setting) is the same edit on every platform --
bump `image.tag` in the env file -- but applying it differs:

**Kubernetes, `updateStrategy: automatedRolling` (the default)**

```
solace k8s deploy -e dev.yaml
```

`deploy` re-applies the custom resource; the operator sees the new tag and rolls the
pods itself (monitor, then backup, then the active node).

**Kubernetes, `updateStrategy: manualPodRestart`**

```
solace k8s deploy -e dev.yaml     # updates the statefulset template; no pod is touched
solace k8s restart -e dev.yaml    # bounces monitor -> backup -> primary, waiting for each
```

The operator deliberately waits for you here, so `deploy` alone changes nothing
visible. `restart <role>` bounces one pod if you would rather drive the order
yourself -- worth doing after a failover, since the order above is by configured
role and the active node may not be the configured primary. Check with
`solace k8s verify redundancy` first.

**Docker / Podman** (on each host, with its own role)

```
solace podman deploy primary -e prod.yaml --restart
solace docker deploy -e prod.yaml --restart
```

`deploy` compares the rendered artifact with the one on disk: unchanged is a no-op,
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
