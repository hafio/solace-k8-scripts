# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this repo is

A single Go binary, `solace`, that deploys and operates Solace PubSub+ Event Brokers on Kubernetes (via the Solace EventBroker Operator / PubSubPlusEventBroker CRD), Docker, and Podman. You describe the broker once in a YAML env file and drive the whole lifecycle from one standardized command tree. **Unsupported** -- not a Solace product. See [README.md](README.md) for the full variable reference and lifecycle.

## Go implementation (`solace` binary)

The `solace` binary presents one standardized lifecycle command tree across Kubernetes, Docker, and Podman. Build and the full lifecycle are in [README.md](README.md); package layout is `internal/{config,engine,render,broker,k8s,container,convert,cli}` + `main.go`,
plus `internal/tools/vulnjudge` -- a dev-only command the `scan` task pipes govulncheck's
JSON through, so a fixable vulnerability fails the gate and one with no released fix warns.

`internal/convert` is the one-way migration aid behind `solace convert`: it parses a legacy
bash env file (the pre-Go `bash/env/<name>` format), maps the `SOLBK_*`/`SOLOP_*`/
`IMAGEREPO_*`/`REPL_*` variables onto the YAML schema, and emits only what the source
actually set. It depends on `internal/config` (schema + validation) and nothing else, so
`config` must never import it -- the invalid-YAML hint in `config.Load` therefore carries its
own bash-file sniff rather than calling into `convert`.

Every test in the repo is catalogued in [docs/test.md](docs/test.md) -- what each one proves,
the per-package fixtures and doubles to reuse, and the injectable seams. Update it in the
same change when you add or remove a test.

[docs/commands.md](docs/commands.md) is the full CLI reference and is **generated** from the
cobra tree by `internal/cli/commanddoc_test.go`. It is a golden: `test` fails while it is
stale, so any command, flag, or `Short` change means regenerating it in the same change with
`go test ./internal/cli -update`. Never hand-edit it. The `--gen` column comes from the
`genAnnotation` marker, so a command that honours `--gen` must be wrapped in `genCapable`.

### Container platform (`internal/container`)

Docker and Podman are one **host-local** platform: one container per host, so there is no operator and no cross-node control point (contrast `internal/k8s`, which drives the whole redundancy group from a single `kubectl` context). The moving parts:

- **Node-local transport** ([internal/container/transport.go](internal/container/transport.go)): a `broker.Transport` over `<runtime> exec`/`cp`. It **ignores the role arg** -- every op targets this host's single container -- so the CLI wires the shared `broker.Ops` config/verify methods with a nominal `config.Primary`. No `--` separator (docker `exec` rejects it); uploads ride stdin via `sh -c 'cat > <dest>'` (secret-safe, body never in argv).
- **Host Manager** ([internal/container/manager.go](internal/container/manager.go)): the container analog of `k8s.Cluster` -- `Check`/`PrepHost`/`Deploy`/`Delete`/`Status`/`Logs`/`CLI`/`Shell`. Podman renders a systemd quadlet unit; Docker a compose file (default) or a `run` line. `Resolve`/`GenPSK`/`Geteuid` are injectable seams (defaults `net.LookupHost`, crypto/rand, `os.Geteuid`) so DNS, PSK generation, and the rootless/rootful euid guard are testable off a Linux host.
- **Node-local HA state machines** ([internal/broker/verify_local.go](internal/broker/verify_local.go)): the k8s `Leader`/`Redundancy` ops drive both nodes from one point, which a one-container-per-host model cannot, so containers get `LeaderLocal` and `RedundancyLocal`, each running only its own host's half. `LocalRole(arg)` detects the role from the host name against the `nodes.*` table when `arg` is empty (loud error on no match). `LeaderLocal` runs only on the primary (loud on backup/monitor); redundancy is a **two-host handshake** -- the primary releases activity and waits for fail-back, the backup dwells `ActiveDwell` (~10s) then reverts; the monitor is rejected loud.
- **Config reuse + one divergence**: container `config`/`verify` read the shared `k8s.*` fields (`domainCerts`, `productKeys`, `diagDir`, `cliScriptsFolder`) -- there is no separate container config namespace. The deliberate divergence from k8s `ConfigAll`: the container `config` (bare) runs every applicable step **except** `leader`, since leader is cross-host + primary-only and would fail loud on the other hosts -- run `config leader` explicitly on the primary.

## Knowledge graph

`graphify-out/` (tracked in git -- `.graphifyignore` scopes what graphify *reads*, a
separate mechanism from `.gitignore`, which has no graphify entry) contains a persistent graph of the repo: [graph.html](graphify-out/graph.html) (interactive viz), [GRAPH_REPORT.md](graphify-out/GRAPH_REPORT.md) (god nodes, surprising cross-document links, suggested questions), [graph.json](graphify-out/graph.json), plus an Obsidian vault and per-community wiki. Use it to answer "what calls X?" / "which files touch concept Y?" without re-reading dozens of files. Rebuild with `/graphify .` after notable changes; incremental updates use `/graphify --update .`. Files listed in [.graphifyignore](.graphifyignore) are excluded from the corpus (graphify does **not** read `.gitignore`, so the two files are maintained independently).

## graphify

This project has a knowledge graph at graphify-out/ with god nodes, community structure, and cross-file relationships.

Rules:
- For codebase questions, first run `graphify query "<question>"` when graphify-out/graph.json exists. Use `graphify path "<A>" "<B>"` for relationships and `graphify explain "<concept>"` for focused concepts. These return a scoped subgraph, usually much smaller than GRAPH_REPORT.md or raw grep output.
- If graphify-out/wiki/index.md exists, use it for broad navigation instead of raw source browsing.
- Read graphify-out/GRAPH_REPORT.md only for broad architecture review or when query/path/explain do not surface enough context.
- After modifying code, run `graphify update .` to keep the graph current (AST-only, no API cost).
