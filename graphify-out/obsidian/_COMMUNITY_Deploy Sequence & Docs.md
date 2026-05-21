---
type: community
cohesion: 0.15
members: 20
---

# Deploy Sequence & Docs

**Cohesion:** 0.15 - loosely connected
**Members:** 20 nodes

## Members
- [[Apply Solace Product Keys]] - code - 057-configure-product-keys.sh
- [[Assert HA Config-Sync Leader]] - code - 050-assert-leader.sh
- [[Canonical Deploy Sequence]] - rationale - README.md
- [[Create Broker Namespace]] - code - 011-create-namespace.sh
- [[Create Broker Secrets]] - code - 012-create-secrets.sh
- [[Delete Broker Namespace]] - code - 111-delete-namespace.sh
- [[Delete Broker Secrets]] - code - 112-delete-secrets.sh
- [[Delete PubSubPlusEventBroker CR and PVCs]] - code - 120-delete-broker.sh
- [[Disable All Default Client-Usernames]] - code - 054-disable-all-default-usernames.sh
- [[Disable Default Message VPN Services]] - code - 053-disable-default-vpn.sh
- [[Environment Loader (000-env.sh)]] - code - 000-env.sh
- [[Gather Configs and Diagnostics]] - code - 069-gather-configs.sh
- [[Label Cluster Nodes for Broker Pods]] - code - 013-label-nodes.sh
- [[Load Domain Certificate Authorities]] - code - 052-load-domain-certs.sh
- [[Load TLS Server Certificate]] - code - 051-load-server-cert.sh
- [[Numbered Script Convention (CategoryActionSequence)]] - rationale - CLAUDE.md
- [[README]] - document - README.md
- [[Test HA Redundancy Failover]] - code - 061-test-redundancy.sh
- [[Test SEMP API Login]] - code - 060-test-semp-login.sh
- [[Verify StorageClass Settings]] - code - 009-check-storage-class.sh

## Live Query (requires Dataview plugin)

```dataview
TABLE source_file, type FROM #community/Deploy_Sequence__Docs
SORT file.name ASC
```

## Connections to other communities
- 11 edges to [[_COMMUNITY_YAML Generation Pattern]]
- 2 edges to [[_COMMUNITY_Env Validation]]

## Top bridge nodes
- [[README]] - degree 20, connects to 2 communities
- [[Environment Loader (000-env.sh)]] - degree 20, connects to 2 communities
- [[Numbered Script Convention (CategoryActionSequence)]] - degree 3, connects to 1 community