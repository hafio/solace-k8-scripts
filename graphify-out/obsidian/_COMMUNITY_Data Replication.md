---
type: community
cohesion: 0.33
members: 7
---

# Data Replication

**Cohesion:** 0.33 - loosely connected
**Members:** 7 nodes

## Members
- [[$REPL_MATE  REPL_CONN_SSL  REPL_PSK]] - rationale
- [[Configure Broker-level Data Replication]] - code - 058-setup-data-replication.sh
- [[Configure VPN-level Data Replication]] - code - 05A-generate-data-replication-for-vpn.sh
- [[Disable All Default Client-Usernames]] - code - 054-disable-all-default-usernames.sh
- [[Solace Data Replication]] - rationale
- [[Solace Default Client-Usernames]] - rationale
- [[Solace Message VPN]] - rationale

## Live Query (requires Dataview plugin)

```dataview
TABLE source_file, type FROM #community/Data_Replication
SORT file.name ASC
```

## Connections to other communities
- 3 edges to [[_COMMUNITY_Env Variables (SOLBK_)]]

## Top bridge nodes
- [[Configure Broker-level Data Replication]] - degree 4, connects to 1 community
- [[Configure VPN-level Data Replication]] - degree 4, connects to 1 community
- [[Disable All Default Client-Usernames]] - degree 3, connects to 1 community