# Graph Report - .  (2026-05-20)

## Corpus Check
- Corpus is ~29,500 words - fits in a single context window. You may not need a graph.

## Summary
- 117 nodes · 95 edges · 53 communities (47 shown, 6 thin omitted)
- Extraction: 91% EXTRACTED · 8% INFERRED · 1% AMBIGUOUS · INFERRED: 8 edges (avg confidence: 0.92)
- Token cost: 52,210 input · 9,214 output

## Community Hubs (Navigation)
- [[_COMMUNITY_Deploy Sequence & Docs|Deploy Sequence & Docs]]
- [[_COMMUNITY_Operational Utilities|Operational Utilities]]
- [[_COMMUNITY_YAML Generation Pattern|YAML Generation Pattern]]
- [[_COMMUNITY_Node Labelling|Node Labelling]]
- [[_COMMUNITY_Env Validation|Env Validation]]
- [[_COMMUNITY_Gather Configs Helper|Gather Configs Helper]]
- [[_COMMUNITY_Local Image Mirror|Local Image Mirror]]
- [[_COMMUNITY_Show All Brokers|Show All Brokers]]

## God Nodes (most connected - your core abstractions)
1. `Environment Loader (000-env.sh)` - 20 edges
2. `000-env.sh (env file)` - 17 edges
3. `Generate and Apply PubSubPlusEventBroker CR` - 4 edges
4. `Generic CLI Script Executor` - 3 edges
5. `Numbered Script Convention (Category/Action/Sequence)` - 3 edges
6. `Mandatory Env Vars (SOLBK_NAME/NS/IMAGE/IMG_TAG/STORAGE_MSGNODE)` - 3 edges
7. `Solace YAML Generator (interactive)` - 3 edges
8. `is_builtin_label()` - 2 edges
9. `extract_custom_labels()` - 2 edges
10. `Verify StorageClass Settings` - 2 edges

## Surprising Connections (you probably didn't know these)
- `Env Var Validation (OK/INFO/ERROR pre-checks)` --conceptually_related_to--> `Mandatory Env Vars (SOLBK_NAME/NS/IMAGE/IMG_TAG/STORAGE_MSGNODE)`  [INFERRED]
  001-check-env.sh → CLAUDE.md
- `Solace YAML Generator (interactive)` --semantically_similar_to--> `gen_yaml() Heredoc Pattern`  [INFERRED] [semantically similar]
  solace-yaml-generator.html → CLAUDE.md
- `Solace YAML Generator (interactive)` --semantically_similar_to--> `PubSubPlusEventBroker CR Schema`  [INFERRED] [semantically similar]
  solace-yaml-generator.html → CLAUDE.md
- `Configure Broker-level Data Replication` --calls--> `Environment Loader (000-env.sh)`  [EXTRACTED]
  058-setup-data-replication.sh → 000-env.sh
- `Configure VPN-level Data Replication` --calls--> `Environment Loader (000-env.sh)`  [EXTRACTED]
  05A-generate-data-replication-for-vpn.sh → 000-env.sh

## Hyperedges (group relationships)
- **Documentation Entry Points** — doc_readme, doc_claude_md, s001_check_env [INFERRED 0.85]
- **Broker YAML Generation Surfaces** — tool_yaml_generator, s020_deploy_broker, concept_gen_yaml_pattern, concept_pubsubplus_cr_schema [INFERRED 0.85]

## Communities (53 total, 6 thin omitted)

### Community 0 - "Deploy Sequence & Docs"
Cohesion: 0.15
Nodes (19): Canonical Deploy Sequence, Numbered Script Convention (Category/Action/Sequence), Environment Loader (000-env.sh), Verify StorageClass Settings, Create Broker Namespace, Create Broker Secrets, Label Cluster Nodes for Broker Pods, Assert HA Config-Sync Leader (+11 more)

### Community 2 - "YAML Generation Pattern"
Cohesion: 0.21
Nodes (10): Env File Pattern (--env arg sources env/<name>), gen_yaml() Heredoc Pattern, PubSubPlusEventBroker CR Schema, Solace YAML Generator (interactive), Deploy Solace Operator (bundled YAML), Generate and Apply PubSubPlusEventBroker CR, Configure Broker-level Data Replication, Generic CLI Script Executor (+2 more)

## Ambiguous Edges - Review These
- `Environment Loader (000-env.sh)` → `001-check-env.sh`  [AMBIGUOUS]
  001-check-env.sh · relation: calls

## Knowledge Gaps
- **10 isolated node(s):** `nodes`, `stores`, `Create Broker Secrets`, `Label Cluster Nodes for Broker Pods`, `Disable Default Message VPN Services` (+5 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **6 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **What is the exact relationship between `Environment Loader (000-env.sh)` and `001-check-env.sh`?**
  _Edge tagged AMBIGUOUS (relation: calls) - confidence is low._
- **Why does `Environment Loader (000-env.sh)` connect `Deploy Sequence & Docs` to `YAML Generation Pattern`, `Env Validation`?**
  _High betweenness centrality (0.034) - this node is a cross-community bridge._
- **Why does `Generate and Apply PubSubPlusEventBroker CR` connect `YAML Generation Pattern` to `Deploy Sequence & Docs`?**
  _High betweenness centrality (0.004) - this node is a cross-community bridge._
- **What connects `nodes`, `stores`, `Create Broker Secrets` to the rest of the system?**
  _11 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Operational Utilities` be split into smaller, more focused modules?**
  _Cohesion score 0.1111111111111111 - nodes in this community are weakly interconnected._