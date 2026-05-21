---
type: community
cohesion: 0.25
members: 9
---

# Local Image Registry

**Cohesion:** 0.25 - loosely connected
**Members:** 9 nodes

## Members
- [[.broker.yaml (PubSubPlusEventBroker CR)]] - code - .broker.yaml
- [[Broker Image (localhostsolace-pubsub-standard)]] - code - .broker.yaml
- [[Broker Storage (solace-storage-class)]] - code - .broker.yaml
- [[Local Docker Registry]] - rationale
- [[Redundancy Group (PrimaryBackupMonitor)]] - code - .broker.yaml
- [[System Scaling (maxConnectionsQueueSpool)]] - code - .broker.yaml
- [[add-insecure-repo.sh_1]] - code - add-insecure-repo.sh
- [[delete-registry.sh_1]] - code - delete-registry.sh
- [[run-registry.sh_1]] - code - run-registry.sh

## Live Query (requires Dataview plugin)

```dataview
TABLE source_file, type FROM #community/Local_Image_Registry
SORT file.name ASC
```

## Connections to other communities
- 3 edges to [[_COMMUNITY_Broker  Operator Inspection]]

## Top bridge nodes
- [[.broker.yaml (PubSubPlusEventBroker CR)]] - degree 5, connects to 1 community
- [[Redundancy Group (PrimaryBackupMonitor)]] - degree 2, connects to 1 community
- [[System Scaling (maxConnectionsQueueSpool)]] - degree 2, connects to 1 community