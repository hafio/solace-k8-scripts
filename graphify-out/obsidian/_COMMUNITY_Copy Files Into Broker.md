---
type: community
cohesion: 1.00
members: 2
---

# Copy Files Into Broker

**Cohesion:** 1.00 - tightly connected
**Members:** 2 nodes

## Members
- [[copy-files-into-broker.sh]] - code - copy-files-into-broker.sh
- [[echoUsage()_2]] - code - copy-files-into-broker.sh

## Live Query (requires Dataview plugin)

```dataview
TABLE source_file, type FROM #community/Copy_Files_Into_Broker
SORT file.name ASC
```
