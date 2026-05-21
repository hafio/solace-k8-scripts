---
type: community
cohesion: 1.00
members: 3
---

# Env Validation

**Cohesion:** 1.00 - tightly connected
**Members:** 3 nodes

## Members
- [[001-check-env.sh_1]] - code - 001-check-env.sh
- [[Env Var Validation (OKINFOERROR pre-checks)]] - rationale - 001-check-env.sh
- [[Mandatory Env Vars (SOLBK_NAMENSIMAGEIMG_TAGSTORAGE_MSGNODE)]] - rationale - CLAUDE.md

## Live Query (requires Dataview plugin)

```dataview
TABLE source_file, type FROM #community/Env_Validation
SORT file.name ASC
```

## Connections to other communities
- 2 edges to [[_COMMUNITY_Deploy Sequence & Docs]]
- 2 edges to [[_COMMUNITY_YAML Generation Pattern]]

## Top bridge nodes
- [[001-check-env.sh_1]] - degree 5, connects to 2 communities
- [[Mandatory Env Vars (SOLBK_NAMENSIMAGEIMG_TAGSTORAGE_MSGNODE)]] - degree 3, connects to 1 community