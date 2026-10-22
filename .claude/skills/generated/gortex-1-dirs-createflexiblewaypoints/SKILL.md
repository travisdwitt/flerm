---
name: gortex-1-dirs-createflexiblewaypoints
description: "Work in the . +1 dirs · createFlexibleWaypoints area — 126 symbols across 2 files (73% cohesion)"
---

# . +1 dirs · createFlexibleWaypoints

126 symbols | 2 files | 73% cohesion

## When to Use

Use this skill when working on files in:
- ``
- `internal/canvas/connection.go`

## Key Files

| File | Symbols |
|------|---------|
| `` | bool |
| `internal/canvas/connection.go` | offB, horiz, ay, y, GetConnectionEdge, ... |

## Entry Points

- `internal/canvas/connection.go::Canvas.createFlexibleWaypoints`
- `internal/canvas/connection.go::Canvas.rerouteConnectionsForMovedBox`

## Connected Communities

- **canvas +2 dirs · len** (5 cross-edges)
- **canvas · box** (4 cross-edges)
- **tui +2 dirs · overlayTooltipOnRenderResult** (3 cross-edges)
- **tui +2 dirs · overlayMinimap** (2 cross-edges)
- **canvas · closestOnSegment** (1 cross-edges)
- **canvas · segmentCells** (1 cross-edges)
- **canvas · boxPtr** (1 cross-edges)

## How to Explore

```
analyze(operation:"communities", id:"community-8")
explore(operation:"context", task:"understand . +1 dirs · createFlexibleWaypoints", format:"gcx")
relations(operation:"usages", target:{symbol:"internal/canvas/connection.go::Canvas.createFlexibleWaypoints"}, format:"gcx")
```

_`format: "gcx"` returns the [GCX1 compact wire format](../../docs/wire-format.md) — round-trippable, ~27% fewer tokens than JSON. Drop it for JSON output; agents using `@gortex/wire` or the Go `github.com/gortexhq/gcx-go` package decode either._
