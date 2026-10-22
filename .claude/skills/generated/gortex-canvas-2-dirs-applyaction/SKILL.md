---
name: gortex-canvas-2-dirs-applyaction
description: "Work in the canvas +2 dirs · applyAction area — 160 symbols across 7 files (75% cohesion)"
---

# canvas +2 dirs · applyAction

160 symbols | 7 files | 75% cohesion

## When to Use

Use this skill when working on files in:
- ``
- `internal/canvas/canvas.go`
- `internal/canvas/connection.go`
- `internal/canvas/highlight.go`
- `internal/tui/move.go`
- `internal/tui/types.go`
- `internal/tui/undo.go`

## Key Files

| File | Symbols |
|------|---------|
| `` | delete, Delete |
| `internal/canvas/canvas.go` | x, id, i, t, kept, ... |
| `internal/canvas/connection.go` | connections, i, connection, RestoreConnection, RestoreConnections, ... |
| `internal/canvas/highlight.go` | SetHighlight, y, cell, ClearHighlight, x, ... |
| `internal/tui/move.go` | canvas, cell, cells, fromDY, id, ... |
| `internal/tui/types.go` | BoxPositions, Box, Text, Data, BoxID, ... |
| `internal/tui/undo.go` | forward, add, add, conn, d, ... |

## Entry Points

- `internal/tui/undo.go::model.applyAction`

## Connected Communities

- **canvas · boxPtr** (8 cross-edges)
- **tui +2 dirs · activateMenuItem** (8 cross-edges)
- **canvas +2 dirs · len** (5 cross-edges)
- **canvas · textPtr** (3 cross-edges)
- **canvas · box** (2 cross-edges)
- **canvas +2 dirs · RenderRaw** (2 cross-edges)
- **tui +2 dirs · overlayTooltipOnRenderResult** (2 cross-edges)
- **. +2 dirs · moveContainedConnections** (2 cross-edges)
- **canvas · AddTextWithID** (1 cross-edges)
- **canvas +1 dirs** (1 cross-edges)
- **tui +2 dirs · overlayMinimap** (1 cross-edges)

## How to Explore

```
analyze(operation:"communities", id:"community-29")
explore(operation:"context", task:"understand canvas +2 dirs · applyAction", format:"gcx")
relations(operation:"usages", target:{symbol:"internal/tui/undo.go::model.applyAction"}, format:"gcx")
```

_`format: "gcx"` returns the [GCX1 compact wire format](../../docs/wire-format.md) — round-trippable, ~27% fewer tokens than JSON. Drop it for JSON output; agents using `@gortex/wire` or the Go `github.com/gortexhq/gcx-go` package decode either._
