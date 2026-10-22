---
name: gortex-canvas-box
description: "Work in the canvas · box area — 122 symbols across 4 files (67% cohesion)"
---

# canvas · box

122 symbols | 4 files | 67% cohesion

## When to Use

Use this skill when working on files in:
- `internal/canvas/box.go`
- `internal/canvas/canvas.go`
- `internal/canvas/connection.go`
- `internal/canvas/highlight.go`

## Key Files

| File | Symbols |
|------|---------|
| `internal/canvas/box.go` | Tooltip, Width, BorderStyle, ID, ZLevel, ... |
| `internal/canvas/canvas.go` | id, dx, ok, id, box, ... |
| `internal/canvas/connection.go` | AddConnection, toX, okFrom, okTo, cx, ... |
| `internal/canvas/highlight.go` | y, dividerY, ok, cells, row, ... |

## Connected Communities

- **tui +2 dirs · overlayTooltipOnRenderResult** (7 cross-edges)
- **canvas +2 dirs · RenderRaw** (7 cross-edges)
- **canvas +2 dirs · len** (4 cross-edges)
- **canvas · drawConnEndArrow** (2 cross-edges)
- **canvas · closestOnSegment** (2 cross-edges)
- **canvas +2 dirs · LoadFromFileWithPan** (2 cross-edges)
- **. +1 dirs · createFlexibleWaypoints** (1 cross-edges)
- **. +2 dirs · moveContainedConnections** (1 cross-edges)

## How to Explore

```
analyze(operation:"communities", id:"community-5")
explore(operation:"context", task:"understand canvas · box", format:"gcx")
```

_`format: "gcx"` returns the [GCX1 compact wire format](../../docs/wire-format.md) — round-trippable, ~27% fewer tokens than JSON. Drop it for JSON output; agents using `@gortex/wire` or the Go `github.com/gortexhq/gcx-go` package decode either._
