---
type: community
cohesion: 1.00
members: 2
---

# Generate VPN Replication

**Cohesion:** 1.00 - tightly connected
**Members:** 2 nodes

## Members
- [[05A-generate-data-replication-for-vpn.sh]] - code - 05A-generate-data-replication-for-vpn.sh
- [[gen_yaml()_3]] - code - 05A-generate-data-replication-for-vpn.sh

## Live Query (requires Dataview plugin)

```dataview
TABLE source_file, type FROM #community/Generate_VPN_Replication
SORT file.name ASC
```
