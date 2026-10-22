---
name: gortex-canvas-2-dirs-renderraw
description: "Work in the canvas +2 dirs · RenderRaw area — 239 symbols across 9 files (52% cohesion)"
---

# canvas +2 dirs · RenderRaw

239 symbols | 9 files | 52% cohesion

## When to Use

Use this skill when working on files in:
- ``
- `internal/canvas/box.go`
- `internal/canvas/canvas.go`
- `internal/canvas/connection.go`
- `internal/canvas/highlight.go`
- `internal/canvas/render.go`
- `internal/tui/app.go`
- `internal/tui/mouse.go`
- `internal/tui/view.go`

## Key Files

| File | Symbols |
|------|---------|
| `` | make, int |
| `internal/canvas/box.go` | contentStartLine, b |
| `internal/canvas/canvas.go` | id, DuplicateText, b, cells, dx, ... |
| `internal/canvas/connection.go` | toX, by, i, excludeToID, panX, ... |
| `internal/canvas/highlight.go` | line, y, lineCells, box, i, ... |
| `internal/canvas/render.go` | ok, panY, boxY, showBoxNumbers, lines, ... |
| `internal/tui/app.go` | abs, sign, x, n |
| `internal/tui/mouse.go` | y, y, x, paintHighlightCell, x, ... |
| `internal/tui/view.go` | relY, dy, w, edgeY, h, ... |

## Entry Points

- `internal/canvas/render.go::Canvas.RenderRaw`
- `internal/canvas/render.go::Canvas.drawBoxAt`

## Connected Communities

- **canvas +2 dirs · len** (15 cross-edges)
- **tui +2 dirs · overlayMinimap** (7 cross-edges)
- **tui +2 dirs · overlayTooltipOnRenderResult** (6 cross-edges)
- **canvas +2 dirs · LoadFromFileWithPan** (5 cross-edges)
- **canvas +1 dirs** (4 cross-edges)
- **canvas · segmentCells** (2 cross-edges)
- **canvas · boxPtr** (2 cross-edges)
- **canvas · box** (2 cross-edges)
- **canvas · boxRune** (2 cross-edges)
- **tui +2 dirs · statusLine** (1 cross-edges)
- **. +2 dirs · moveContainedConnections** (1 cross-edges)
- **tui · paintCells** (1 cross-edges)
- **canvas · textPtr** (1 cross-edges)

## How to Explore

```
analyze(operation:"communities", id:"community-11")
explore(operation:"context", task:"understand canvas +2 dirs · RenderRaw", format:"gcx")
relations(operation:"usages", target:{symbol:"internal/canvas/render.go::Canvas.RenderRaw"}, format:"gcx")
```

_`format: "gcx"` returns the [GCX1 compact wire format](../../docs/wire-format.md) — round-trippable, ~27% fewer tokens than JSON. Drop it for JSON output; agents using `@gortex/wire` or the Go `github.com/gortexhq/gcx-go` package decode either._
