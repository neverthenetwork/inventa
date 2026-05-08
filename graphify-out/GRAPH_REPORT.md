# Graph Report - .  (2026-05-08)

## Corpus Check
- Corpus is ~2,982 words - fits in a single context window. You may not need a graph.

## Summary
- 57 nodes · 67 edges · 8 communities (6 shown, 2 thin omitted)
- Extraction: 82% EXTRACTED · 18% INFERRED · 0% AMBIGUOUS · INFERRED: 12 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- [[_COMMUNITY_Community 0|Community 0]]
- [[_COMMUNITY_Community 1|Community 1]]
- [[_COMMUNITY_Community 2|Community 2]]
- [[_COMMUNITY_Community 3|Community 3]]
- [[_COMMUNITY_Community 4|Community 4]]
- [[_COMMUNITY_Community 5|Community 5]]
- [[_COMMUNITY_Community 6|Community 6]]

## God Nodes (most connected - your core abstractions)
1. `MyLogger` - 9 edges
2. `main()` - 6 edges
3. `JsHandler()` - 6 edges
4. `SetUpLogger()` - 4 edges
5. `ProcessBGPUpdates()` - 4 edges
6. `GetBestPathNames()` - 4 edges
7. `FindInArray()` - 4 edges
8. `InitConfig()` - 3 edges
9. `StripUnwanted()` - 3 edges
10. `filterNodes()` - 3 edges

## Surprising Connections (you probably didn't know these)
- `JsHandler()` --calls--> `GetPathSegments()`  [INFERRED]
  src/inventa/web/web.go → src/inventa/spf/spf.go
- `main()` --calls--> `SetUpLogger()`  [INFERRED]
  src/inventa/inventa.go → src/inventa/logging/logging.go
- `main()` --calls--> `MakePeerConfiguration()`  [INFERRED]
  src/inventa/inventa.go → src/inventa/input/bgpls/bgpls.go
- `ProcessBGPUpdates()` --calls--> `FindInArray()`  [INFERRED]
  src/inventa/input/bgpls/bgpls.go → src/inventa/utils/utils.go
- `JsHandler()` --calls--> `GetBestPathNames()`  [INFERRED]
  src/inventa/web/web.go → src/inventa/spf/spf.go

## Communities (8 total, 2 thin omitted)

### Community 0 - "Community 0"
Cohesion: 0.21
Nodes (9): MakePeerConfiguration(), ProcessBGPUpdates(), loadJSON(), main(), Conf, InitConfig(), StripUnwanted(), TestInitConfig() (+1 more)

### Community 2 - "Community 2"
Cohesion: 0.28
Nodes (7): BestPath, BestPaths, PathSegment, GetBestPathNames(), GetPathSegments(), makeDijkstra(), makeNameList()

### Community 3 - "Community 3"
Cohesion: 0.33
Nodes (5): FindInArray(), collapsePathPairs(), filterEdges(), filterNodes(), JsHandler()

### Community 4 - "Community 4"
Cohesion: 0.33
Nodes (5): code:block1 (cd src/inventa/), code:block2 (docker build -t inventa .), Docker, Inventa (latin for Discovery), To use

### Community 5 - "Community 5"
Cohesion: 0.4
Nodes (4): linkNLRI, nodeNLRI, prefixEntry, prefixNLRI

## Knowledge Gaps
- **10 isolated node(s):** `nodeNLRI`, `linkNLRI`, `prefixEntry`, `prefixNLRI`, `PathSegment` (+5 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **2 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `FindInArray()` connect `Community 3` to `Community 0`?**
  _High betweenness centrality (0.313) - this node is a cross-community bridge._
- **Why does `ProcessBGPUpdates()` connect `Community 0` to `Community 3`, `Community 5`?**
  _High betweenness centrality (0.311) - this node is a cross-community bridge._
- **Why does `main()` connect `Community 0` to `Community 1`?**
  _High betweenness centrality (0.302) - this node is a cross-community bridge._
- **Are the 4 inferred relationships involving `main()` (e.g. with `SetUpLogger()` and `InitConfig()`) actually correct?**
  _`main()` has 4 INFERRED edges - model-reasoned connections that need verification._
- **Are the 2 inferred relationships involving `JsHandler()` (e.g. with `GetBestPathNames()` and `GetPathSegments()`) actually correct?**
  _`JsHandler()` has 2 INFERRED edges - model-reasoned connections that need verification._
- **Are the 3 inferred relationships involving `ProcessBGPUpdates()` (e.g. with `main()` and `StripUnwanted()`) actually correct?**
  _`ProcessBGPUpdates()` has 3 INFERRED edges - model-reasoned connections that need verification._
- **What connects `nodeNLRI`, `linkNLRI`, `prefixEntry` to the rest of the system?**
  _10 weakly-connected nodes found - possible documentation gaps or missing edges._