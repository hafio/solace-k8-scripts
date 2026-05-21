---
type: community
cohesion: 1.00
members: 2
---

# Copy Files From Broker

**Cohesion:** 1.00 - tightly connected
**Members:** 2 nodes

## Members
- [[copy-files-from-broker.sh]] - code - copy-files-from-broker.sh
- [[echoUsage()_1]] - code - copy-files-from-broker.sh

## Live Query (requires Dataview plugin)

```dataview
TABLE source_file, type FROM #community/Copy_Files_From_Broker
SORT file.name ASC
```
