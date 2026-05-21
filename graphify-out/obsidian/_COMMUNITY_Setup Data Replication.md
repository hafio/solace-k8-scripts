---
type: community
cohesion: 1.00
members: 2
---

# Setup Data Replication

**Cohesion:** 1.00 - tightly connected
**Members:** 2 nodes

## Members
- [[058-setup-data-replication.sh]] - code - 058-setup-data-replication.sh
- [[gen_yaml()_2]] - code - 058-setup-data-replication.sh

## Live Query (requires Dataview plugin)

```dataview
TABLE source_file, type FROM #community/Setup_Data_Replication
SORT file.name ASC
```
