---
type: community
cohesion: 0.12
members: 30
---

# Env Variables (SOLBK_*)

**Cohesion:** 0.12 - loosely connected
**Members:** 30 nodes

## Members
- [[$KUBE (kubectl binary)]] - rationale
- [[$SOLBK_CLISCRIPTS_FOLDER]] - rationale
- [[$SOLBK_IMAGE  $SOLBK_IMG_TAG]] - rationale
- [[$SOLBK_NAME (broker CR name)]] - rationale
- [[$SOLBK_NODELABEL_PRIBKPMON]] - rationale
- [[$SOLBK_NS (broker namespace)]] - rationale
- [[$SOLBK_PRODUCTKEYS]] - rationale
- [[$SOLBK_REDUNDANCY (HA flag)]] - rationale
- [[$SOLBK_USR_SECRET (admin secret)]] - rationale
- [[Apply Solace Product Keys]] - code - 057-configure-product-keys.sh
- [[Assert HA Config-Sync Leader]] - code - 050-assert-leader.sh
- [[Create Broker Namespace]] - code - 011-create-namespace.sh
- [[Delete Broker Namespace]] - code - 111-delete-namespace.sh
- [[Delete PubSubPlusEventBroker CR and PVCs]] - code - 120-delete-broker.sh
- [[Environment Loader (000-env.sh)]] - code - 000-env.sh
- [[Generate and Apply PubSubPlusEventBroker CR]] - code - 020-deploy-broker.sh
- [[Generic CLI Script Executor]] - code - 059-execute-cli.sh
- [[Kubernetes LoadBalancer Service]] - code
- [[Kubernetes Namespace]] - code
- [[Kubernetes Node Label]] - code
- [[Label Cluster Nodes for Broker Pods]] - code - 013-label-nodes.sh
- [[PersistentVolumeClaim]] - code
- [[PubSubPlusEventBroker CRD]] - code
- [[Solace Config-Sync (HA Leader)]] - rationale
- [[Solace HA Redundancy (PrimaryBackupMonitor)]] - rationale
- [[Solace Product Key]] - rationale
- [[Solace SEMP API]] - rationale
- [[Test HA Redundancy Failover]] - code - 061-test-redundancy.sh
- [[Test SEMP API Login]] - code - 060-test-semp-login.sh
- [[envenv-file sourced by 000-env.sh]] - rationale

## Live Query (requires Dataview plugin)

```dataview
TABLE source_file, type FROM #community/Env_Variables_SOLBK_
SORT file.name ASC
```

## Connections to other communities
- 16 edges to [[_COMMUNITY_Operator Install & Secrets]]
- 9 edges to [[_COMMUNITY_TLS Certs & Diagnostics]]
- 4 edges to [[_COMMUNITY_Deployment Concepts Reference]]
- 3 edges to [[_COMMUNITY_Data Replication]]

## Top bridge nodes
- [[Environment Loader (000-env.sh)]] - degree 30, connects to 3 communities
- [[Generate and Apply PubSubPlusEventBroker CR]] - degree 23, connects to 3 communities
- [[Assert HA Config-Sync Leader]] - degree 8, connects to 2 communities
- [[Test SEMP API Login]] - degree 6, connects to 2 communities
- [[Create Broker Namespace]] - degree 6, connects to 1 community