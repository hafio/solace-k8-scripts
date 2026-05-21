---
type: community
cohesion: 0.16
members: 26
---

# Broker / Operator Inspection

**Cohesion:** 0.16 - loosely connected
**Members:** 26 nodes

## Members
- [[000-env.sh (env file)]] - code - 000-env.sh
- [[150-remove-domain-certs.sh_1]] - code - 150-remove-domain-certs.sh
- [[Broker Pod]] - rationale
- [[LoadBalancer Service]] - rationale
- [[LoadBalancer Service (SEMPSMFMQTTAMQPREST ports)]] - code - .broker.yaml
- [[Operator Pod]] - rationale
- [[Replicas  Scaling]] - rationale
- [[Solace CLI]] - rationale
- [[copy-files-from-broker.sh_1]] - code - copy-files-from-broker.sh
- [[copy-files-into-broker.sh_1]] - code - copy-files-into-broker.sh
- [[d037-assign-ip.sh_1]] - code - d037-assign-ip.sh
- [[d137-delete-ip.sh_1]] - code - d137-delete-ip.sh
- [[desc-broker.sh_1]] - code - desc-broker.sh
- [[desc-lb.sh_1]] - code - desc-lb.sh
- [[desc-op.sh_1]] - code - desc-op.sh
- [[enter-solace-cli.sh_1]] - code - enter-solace-cli.sh
- [[enter-solace-shell.sh_1]] - code - enter-solace-shell.sh
- [[get-broker-status.sh_1]] - code - get-broker-status.sh
- [[get-op-status.sh_1]] - code - get-op-status.sh
- [[kubectl describe]] - rationale
- [[logs-broker.sh_1]] - code - logs-broker.sh
- [[logs-op.sh_1]] - code - logs-op.sh
- [[replicas-start-broker.sh_1]] - code - replicas-start-broker.sh
- [[replicas-stop-broker.sh_1]] - code - replicas-stop-broker.sh
- [[show-all-brokers.sh_1]] - code - show-all-brokers.sh
- [[watch.sh_1]] - code - watch.sh

## Live Query (requires Dataview plugin)

```dataview
TABLE source_file, type FROM #community/Broker_/_Operator_Inspection
SORT file.name ASC
```

## Connections to other communities
- 3 edges to [[_COMMUNITY_Local Image Registry]]
- 2 edges to [[_COMMUNITY_Operator Install & Secrets]]

## Top bridge nodes
- [[Broker Pod]] - degree 12, connects to 1 community
- [[get-broker-status.sh_1]] - degree 5, connects to 1 community
- [[logs-broker.sh_1]] - degree 4, connects to 1 community
- [[Replicas  Scaling]] - degree 3, connects to 1 community
- [[LoadBalancer Service (SEMPSMFMQTTAMQPREST ports)]] - degree 2, connects to 1 community