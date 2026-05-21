---
type: community
cohesion: 0.18
members: 17
---

# Operator Install & Secrets

**Cohesion:** 0.18 - loosely connected
**Members:** 17 nodes

## Members
- [[$IMAGEREPO_HOSTUSERPASSSECRET]] - rationale
- [[$SOLBK_STORAGECLASS]] - rationale
- [[$SOLBK_SVR_SECRET (TLS secret)]] - rationale
- [[$SOLOP_ (operator env vars)]] - rationale
- [[Create Broker Secrets]] - code - 012-create-secrets.sh
- [[Delete Broker Secrets]] - code - 112-delete-secrets.sh
- [[Delete Solace Operator (bundled YAML)]] - code - 110-delete-operator.sh
- [[Deploy Solace Operator (bundled YAML)]] - code - 010-deploy-operator.sh
- [[Environment Validation (001-check-env.sh)]] - code - 001-check-env.sh
- [[How-To Deploy Solace - Bastion Workflow Guide]] - document - howto-deploy-solace.md
- [[Kubernetes Secret]] - code
- [[Kubernetes StorageClass]] - code
- [[Message VPN (broker logical partition)]] - rationale - howto-deploy-solace.md
- [[Operator Installation]] - rationale - EventBrokerOperatorUserGuide.md
- [[Solace PubSub+ Operator (Deployment)]] - code
- [[Verify StorageClass Settings]] - code - 009-check-storage-class.sh
- [[kubectl tool]] - rationale

## Live Query (requires Dataview plugin)

```dataview
TABLE source_file, type FROM #community/Operator_Install__Secrets
SORT file.name ASC
```

## Connections to other communities
- 16 edges to [[_COMMUNITY_Env Variables (SOLBK_)]]
- 8 edges to [[_COMMUNITY_Deployment Concepts Reference]]
- 2 edges to [[_COMMUNITY_TLS Certs & Diagnostics]]
- 2 edges to [[_COMMUNITY_Broker  Operator Inspection]]

## Top bridge nodes
- [[How-To Deploy Solace - Bastion Workflow Guide]] - degree 21, connects to 4 communities
- [[Create Broker Secrets]] - degree 8, connects to 2 communities
- [[Verify StorageClass Settings]] - degree 7, connects to 2 communities
- [[Deploy Solace Operator (bundled YAML)]] - degree 7, connects to 1 community
- [[Environment Validation (001-check-env.sh)]] - degree 5, connects to 1 community