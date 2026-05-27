# Graph Report - solace-k8-scripts  (2026-05-27)

## Corpus Check
- 42 files · ~29,507 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 167 nodes · 146 edges · 55 communities (35 shown, 20 thin omitted)
- Extraction: 94% EXTRACTED · 5% INFERRED · 1% AMBIGUOUS · INFERRED: 8 edges (avg confidence: 0.92)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `8bfd0ec6`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- [[_COMMUNITY_Community 0|Community 0]]
- [[_COMMUNITY_Community 1|Community 1]]
- [[_COMMUNITY_Community 2|Community 2]]
- [[_COMMUNITY_Community 3|Community 3]]
- [[_COMMUNITY_Community 4|Community 4]]
- [[_COMMUNITY_Community 5|Community 5]]
- [[_COMMUNITY_Community 6|Community 6]]
- [[_COMMUNITY_Community 9|Community 9]]
- [[_COMMUNITY_Community 12|Community 12]]
- [[_COMMUNITY_Community 13|Community 13]]
- [[_COMMUNITY_Community 14|Community 14]]
- [[_COMMUNITY_Community 15|Community 15]]
- [[_COMMUNITY_Community 17|Community 17]]
- [[_COMMUNITY_Community 20|Community 20]]
- [[_COMMUNITY_Community 21|Community 21]]
- [[_COMMUNITY_Community 23|Community 23]]
- [[_COMMUNITY_Community 24|Community 24]]
- [[_COMMUNITY_Community 26|Community 26]]
- [[_COMMUNITY_Community 27|Community 27]]
- [[_COMMUNITY_Community 31|Community 31]]
- [[_COMMUNITY_Community 37|Community 37]]
- [[_COMMUNITY_Community 40|Community 40]]
- [[_COMMUNITY_Community 41|Community 41]]
- [[_COMMUNITY_Community 44|Community 44]]
- [[_COMMUNITY_Community 53|Community 53]]
- [[_COMMUNITY_Community 54|Community 54]]

## God Nodes (most connected - your core abstractions)
1. `Environment Loader (000-env.sh)` - 20 edges
2. `000-env.sh (env file)` - 17 edges
3. `Additional Helper Scripts` - 5 edges
4. `How to use` - 4 edges
5. `Configuration Variables` - 4 edges
6. `Generate and Apply PubSubPlusEventBroker CR` - 4 edges
7. `Generic CLI Script Executor` - 3 edges
8. `Numbered Script Convention (Category/Action/Sequence)` - 3 edges
9. `Mandatory Env Vars (SOLBK_NAME/NS/IMAGE/IMG_TAG/STORAGE_MSGNODE)` - 3 edges
10. `Solace YAML Generator (interactive)` - 3 edges

## Surprising Connections (you probably didn't know these)
- `Env Var Validation (OK/INFO/ERROR pre-checks)` --conceptually_related_to--> `Mandatory Env Vars (SOLBK_NAME/NS/IMAGE/IMG_TAG/STORAGE_MSGNODE)`  [INFERRED]
  001-check-env.sh → CLAUDE.md
- `Solace YAML Generator (interactive)` --semantically_similar_to--> `gen_yaml() Heredoc Pattern`  [INFERRED] [semantically similar]
  solace-yaml-generator.html → CLAUDE.md
- `Solace YAML Generator (interactive)` --semantically_similar_to--> `PubSubPlusEventBroker CR Schema`  [INFERRED] [semantically similar]
  solace-yaml-generator.html → CLAUDE.md
- `Verify StorageClass Settings` --calls--> `Environment Loader (000-env.sh)`  [EXTRACTED]
  009-check-storage-class.sh → 000-env.sh
- `Create Broker Namespace` --calls--> `Environment Loader (000-env.sh)`  [EXTRACTED]
  011-create-namespace.sh → 000-env.sh

## Hyperedges (group relationships)
- **Documentation Entry Points** — doc_readme, doc_claude_md, s001_check_env [INFERRED 0.85]
- **Broker YAML Generation Surfaces** — tool_yaml_generator, s020_deploy_broker, concept_gen_yaml_pattern, concept_pubsubplus_cr_schema [INFERRED 0.85]

## Communities (55 total, 20 thin omitted)

### Community 0 - "Community 0"
Cohesion: 0.12
Nodes (27): Canonical Deploy Sequence, Env File Pattern (--env arg sources env/<name>), Env Var Validation (OK/INFO/ERROR pre-checks), Mandatory Env Vars (SOLBK_NAME/NS/IMAGE/IMG_TAG/STORAGE_MSGNODE), Numbered Script Convention (Category/Action/Sequence), Environment Loader (000-env.sh), Verify StorageClass Settings, Deploy Solace Operator (bundled YAML) (+19 more)

### Community 2 - "Community 2"
Cohesion: 0.50
Nodes (4): gen_yaml() Heredoc Pattern, PubSubPlusEventBroker CR Schema, Solace YAML Generator (interactive), Generate and Apply PubSubPlusEventBroker CR

### Community 4 - "Community 4"
Cohesion: 0.12
Nodes (16): Additional Helper Scripts, code:bash (./desc-broker.sh --env dev backup), Common / Kubernetes / Environment, Configuration Variables, File Organization System, How to use, Inspection, Interactive access (+8 more)

### Community 5 - "Community 5"
Cohesion: 0.67
Nodes (3): 069-gather-configs.sh script, echoUsage(), nodes

### Community 53 - "Community 53"
Cohesion: 0.18
Nodes (9): CLI scripts directory, code:bash (./001-check-env.sh --env dev), code:bash (source "$(dirname "$0")/000-env.sh"), File-naming convention, `gen_yaml()` pattern — large scripts, How the scripts compose, Knowledge graph, Running a script (+1 more)

### Community 54 - "Community 54"
Cohesion: 0.50
Nodes (3): Minikube Steps, Steps, Using local tar images for kubectl setup

## Ambiguous Edges - Review These
- `Environment Loader (000-env.sh)` → `001-check-env.sh`  [AMBIGUOUS]
  001-check-env.sh · relation: calls

## Knowledge Gaps
- **45 isolated node(s):** `000-env.sh script`, `009-check-storage-class.sh script`, `050-assert-leader.sh script`, `051-load-server-cert.sh script`, `053-disable-default-vpn.sh script` (+40 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **20 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **What is the exact relationship between `Environment Loader (000-env.sh)` and `001-check-env.sh`?**
  _Edge tagged AMBIGUOUS (relation: calls) - confidence is low._
- **Why does `Environment Loader (000-env.sh)` connect `Community 0` to `Community 2`?**
  _High betweenness centrality (0.017) - this node is a cross-community bridge._
- **What connects `000-env.sh script`, `009-check-storage-class.sh script`, `050-assert-leader.sh script` to the rest of the system?**
  _46 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Community 0` be split into smaller, more focused modules?**
  _Cohesion score 0.11724137931034483 - nodes in this community are weakly interconnected._
- **Should `Community 1` be split into smaller, more focused modules?**
  _Cohesion score 0.1111111111111111 - nodes in this community are weakly interconnected._
- **Should `Community 4` be split into smaller, more focused modules?**
  _Cohesion score 0.11764705882352941 - nodes in this community are weakly interconnected._