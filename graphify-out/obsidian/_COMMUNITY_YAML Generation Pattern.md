---
type: community
cohesion: 0.21
members: 12
---

# YAML Generation Pattern

**Cohesion:** 0.21 - loosely connected
**Members:** 12 nodes

## Members
- [[CLAUDE]] - document - CLAUDE.md
- [[Configure Broker-level Data Replication]] - code - 058-setup-data-replication.sh
- [[Configure VPN-level Data Replication]] - code - 05A-generate-data-replication-for-vpn.sh
- [[Delete Solace Operator (bundled YAML)]] - code - 110-delete-operator.sh
- [[Deploy Solace Operator (bundled YAML)]] - code - 010-deploy-operator.sh
- [[Env File Pattern (--env arg sources envname)]] - rationale - CLAUDE.md
- [[Generate and Apply PubSubPlusEventBroker CR]] - code - 020-deploy-broker.sh
- [[Generic CLI Script Executor]] - code - 059-execute-cli.sh
- [[PubSubPlusEventBroker CR Schema]] - rationale - CLAUDE.md
- [[Solace YAML Generator (interactive)]] - rationale - solace-yaml-generator.html
- [[gen_yaml() Heredoc Pattern]] - rationale - CLAUDE.md
- [[solace-yaml-generator.html]] - document - solace-yaml-generator.html

## Live Query (requires Dataview plugin)

```dataview
TABLE source_file, type FROM #community/YAML_Generation_Pattern
SORT file.name ASC
```

## Connections to other communities
- 11 edges to [[_COMMUNITY_Deploy Sequence & Docs]]
- 2 edges to [[_COMMUNITY_Env Validation]]

## Top bridge nodes
- [[CLAUDE]] - degree 14, connects to 2 communities
- [[Generate and Apply PubSubPlusEventBroker CR]] - degree 4, connects to 1 community
- [[Generic CLI Script Executor]] - degree 3, connects to 1 community
- [[Env File Pattern (--env arg sources envname)]] - degree 2, connects to 1 community
- [[Deploy Solace Operator (bundled YAML)]] - degree 2, connects to 1 community