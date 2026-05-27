# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this repo is

A collection of bash scripts that deploy a Solace PubSub+ Event Broker on Kubernetes via the Solace EventBroker Operator (PubSubPlusEventBroker CRD). The scripts wrap `kubectl`/`helm` so users only fill in env-file variables instead of hand-crafting YAML. **Unsupported** — not a Solace product. See [README.md](README.md) for the full variable reference.

## Running a script

Every numbered script takes `--env <name>`:

```bash
./001-check-env.sh --env dev
./020-deploy-broker.sh --env dev.dns.hostname
```

`<name>` is a file in `env/` (e.g. `env/dev`). To create one, copy [env/sample](env/sample) — never edit `sample` itself; it's the template. If `--env` is omitted, the default is `env/default`. Pass-through args after `--env <name>` are preserved (the pattern is `PARAMS=...; set -- ${PARAMS}`).

## How the scripts compose

Every script begins by sourcing [000-env.sh](000-env.sh):

```bash
source "$(dirname "$0")/000-env.sh"
```

`000-env.sh` does three things in order:

1. Parses `--env <name>` and sources `env/<name>` (the user's vars).
2. Applies defaults to optional vars (`VAR=${VAR:-default}`) and derives values via `kubectl` (e.g. auto-detects `SOLOP_DERIVED_NS` by grep'ing the operator deployment).
3. Validates mandatory vars (`SOLBK_NAME`, `SOLBK_NS`, `SOLBK_IMAGE`, `SOLBK_IMG_TAG`, `SOLBK_STORAGE_MSGNODE`) and exits if any are empty.

So **`000-env.sh` is never run directly** — it's the bootstrap every other script relies on. Changes to env-loading semantics belong here, not in individual scripts. It also exposes a shared `pick_pod` helper that normalizes a pod-role argument (`p|primary`, `b|backup`, `m|monitor`) to the single-letter form used in pod names; helper scripts call it with `pod=$(pick_pod "$1") || exit 1`.

## File-naming convention

Scripts are prefixed `<Category><Action><Sequence>`:

| Position | Values |
| --- | --- |
| Category | `0` = create/deploy · `1` = delete/remove · `d` = infrastructure helpers (reserved prefix; no scripts currently use it) |
| Action | `0` = pre-checks · `1` = pre-actions · `2` = deploy · `5` = post-config · `6` = post-checks |
| Sequence | execution order within the action band |

Delete scripts run in reverse order. Anything without a number prefix (e.g. `desc-broker.sh`, `logs-broker.sh`, `enter-solace-cli.sh`, `replicas-start-broker.sh`) is an operational helper, not part of the deploy sequence.

The README's "How to use" section is the canonical run order; consult it before reordering steps.

## `gen_yaml()` pattern — large scripts

Several scripts ([010-deploy-operator.sh](010-deploy-operator.sh), [020-deploy-broker.sh](020-deploy-broker.sh), [110-delete-operator.sh](110-delete-operator.sh)) define an inline `gen_yaml()` function that emits a multi-doc Kubernetes manifest from a heredoc using the env vars, then pipes it to `${KUBE} apply -f -`. **`010-deploy-operator.sh` and `110-delete-operator.sh` are ~119 KB** — they embed the entire Solace operator bundle as YAML literals. Don't reformat blindly; line breaks inside the heredoc are part of the manifest.

There is also a browser tool [solace-yaml-generator.html](solace-yaml-generator.html) that produces broker CR YAML interactively — keep it in sync with the variables in [env/sample](env/sample) when adding new knobs.

## CLI scripts directory

`059-execute-cli.sh` runs Solace CLI commands inside the broker pod. CLI templates live in `cliscripts-templates/` (e.g. [setup-vpn-data-replication.cli](cliscripts-templates/setup-vpn-data-replication.cli)). The directory the script reads from is controlled by `SOLBK_CLISCRIPTS_FOLDER` (default `cli`). Some scripts auto-generate CLI files into that folder; others expect the user to author them.

## Knowledge graph

`graphify-out/` (gitignored via `.graphifyignore`) contains a persistent graph of the repo: [graph.html](graphify-out/graph.html) (interactive viz), [GRAPH_REPORT.md](graphify-out/GRAPH_REPORT.md) (god nodes, surprising cross-document links, suggested questions), [graph.json](graphify-out/graph.json), plus an Obsidian vault and per-community wiki. Use it to answer "what calls X?" / "which scripts touch concept Y?" without re-reading dozens of scripts. Rebuild with `/graphify .` after notable changes; incremental updates use `/graphify --update .`. Files listed in [.graphifyignore](.graphifyignore) are excluded from the corpus (graphify does **not** read `.gitignore`, so the two files are maintained independently).
