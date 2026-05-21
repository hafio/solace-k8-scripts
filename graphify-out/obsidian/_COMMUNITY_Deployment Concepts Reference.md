---
type: community
cohesion: 0.18
members: 23
---

# Deployment Concepts Reference

**Cohesion:** 0.18 - loosely connected
**Members:** 23 nodes

## Members
- [[Admin & Monitor Credentials (Secrets)]] - rationale - EventBrokerOperatorParametersReference.md
- [[Air-gapped  Private Registry Deployment]] - rationale - EventBrokerOperatorUserGuide.md
- [[Broker Scaling Parameters (SystemScaling)]] - rationale - EventBrokerOperatorUserGuide.md
- [[Broker Security Context  Container SecurityContext]] - rationale - EventBrokerOperatorParametersReference.md
- [[Env File Configuration Variables (SOLBK_, SOLOP_)]] - rationale - README.md
- [[Helm Chart (alternative deployment path)]] - rationale - EventBrokerOperatorUserGuide.md
- [[High Availability (PrimaryBackupMonitor, PDB)]] - rationale - EventBrokerOperatorUserGuide.md
- [[Image Registry & Pull Secrets]] - rationale - EventBrokerOperatorUserGuide.md
- [[Interactive YAML Generation (broker.yaml output)]] - rationale - solace-yaml-generator.html
- [[Load Balancer  Service Exposure (MetalLB)]] - rationale - README.md
- [[Local Image Mirror (ctrcrictl import)]] - rationale - 003-local-images.md
- [[Node Assignment (Affinity, Tolerations, NodeSelector)]] - rationale - EventBrokerOperatorParametersReference.md
- [[Persistent Storage (PVC, StorageClass)]] - rationale - EventBrokerOperatorUserGuide.md
- [[Prometheus Monitoring (Exporter, ServiceMonitor)]] - rationale - EventBrokerOperatorUserGuide.md
- [[PubSubPlusEventBroker CR Schema]] - rationale - EventBrokerOperatorParametersReference.md
- [[README - Project Overview & Script Workflow]] - document - README.md
- [[Solace Brand Design System (colorstypography)]] - rationale - solace-theme-reference.md
- [[Solace Event Broker Operator Parameters Reference]] - document - EventBrokerOperatorParametersReference.md
- [[Solace Event Broker Operator User Guide]] - document - EventBrokerOperatorUserGuide.md
- [[Solace Theme Reference (branddesign system)]] - document - solace-theme-reference.md
- [[Solace YAML Generator (interactive HTML tool)]] - document - solace-yaml-generator.html
- [[TLS  SSL Certificates (server + domain CA)]] - rationale - EventBrokerOperatorUserGuide.md
- [[Using Local Tar Images for kubectlcontainerd]] - document - 003-local-images.md

## Live Query (requires Dataview plugin)

```dataview
TABLE source_file, type FROM #community/Deployment_Concepts_Reference
SORT file.name ASC
```

## Connections to other communities
- 8 edges to [[_COMMUNITY_Operator Install & Secrets]]
- 4 edges to [[_COMMUNITY_Env Variables (SOLBK_)]]
- 2 edges to [[_COMMUNITY_TLS Certs & Diagnostics]]

## Top bridge nodes
- [[TLS  SSL Certificates (server + domain CA)]] - degree 7, connects to 2 communities
- [[High Availability (PrimaryBackupMonitor, PDB)]] - degree 6, connects to 2 communities
- [[Admin & Monitor Credentials (Secrets)]] - degree 5, connects to 2 communities
- [[Solace Event Broker Operator User Guide]] - degree 14, connects to 1 community
- [[README - Project Overview & Script Workflow]] - degree 11, connects to 1 community