---
type: community
cohesion: 0.17
members: 12
---

# TLS Certs & Diagnostics

**Cohesion:** 0.17 - loosely connected
**Members:** 12 nodes

## Members
- [[$SOLBK_DIAG_DIR]] - rationale
- [[$SOLBK_DOMAINCERT_FILES  FOLDER]] - rationale
- [[$SOLBK_TLS_CERT  $SOLBK_TLS_CERTKEY]] - rationale
- [[Disable Default Message VPN Services]] - code - 053-disable-default-vpn.sh
- [[Gather Configs and Diagnostics]] - code - 069-gather-configs.sh
- [[Load Domain Certificate Authorities]] - code - 052-load-domain-certs.sh
- [[Load TLS Server Certificate]] - code - 051-load-server-cert.sh
- [[Solace CLI (cliscripts execution)]] - rationale
- [[Solace Default Message VPN]] - rationale
- [[Solace Domain Certificate Authorities]] - rationale
- [[Solace Gather-Diagnostics]] - rationale
- [[Solace TLS Server Certificate]] - rationale

## Live Query (requires Dataview plugin)

```dataview
TABLE source_file, type FROM #community/TLS_Certs__Diagnostics
SORT file.name ASC
```

## Connections to other communities
- 9 edges to [[_COMMUNITY_Env Variables (SOLBK_)]]
- 2 edges to [[_COMMUNITY_Operator Install & Secrets]]
- 2 edges to [[_COMMUNITY_Deployment Concepts Reference]]

## Top bridge nodes
- [[Load TLS Server Certificate]] - degree 7, connects to 3 communities
- [[Load Domain Certificate Authorities]] - degree 6, connects to 3 communities
- [[Solace CLI (cliscripts execution)]] - degree 6, connects to 1 community
- [[Gather Configs and Diagnostics]] - degree 4, connects to 1 community
- [[Disable Default Message VPN Services]] - degree 3, connects to 1 community