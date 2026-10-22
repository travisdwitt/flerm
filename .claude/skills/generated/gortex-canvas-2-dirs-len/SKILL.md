---
name: gortex-canvas-2-dirs-len
description: "Work in the canvas +2 dirs · len area — 174 symbols across 10 files (50% cohesion)"
---

# canvas +2 dirs · len

174 symbols | 10 files | 50% cohesion

## When to Use

Use this skill when working on files in:
- ``
- `internal/canvas/box.go`
- `internal/canvas/canvas.go`
- `internal/canvas/color.go`
- `internal/canvas/connection.go`
- `internal/canvas/render.go`
- `internal/tui/editing.go`
- `internal/tui/keys_text.go`
- `internal/tui/mouse.go`
- `internal/tui/view.go`

## Key Files

| File | Symbols |
|------|---------|
| `` | Insert, len, rune |
| `internal/canvas/box.go` | line, line, line, fitsWidth, contentHeight, ... |
| `internal/canvas/canvas.go` | box, AddBoxWithID, x, text, id, ... |
| `internal/canvas/color.go` | ColorEnabled |
| `internal/canvas/connection.go` | i, cur, end, kept, kept, ... |
| `internal/canvas/render.go` | panX, x, editTextY, from, ch, ... |
| `internal/tui/editing.go` | lineEndPos, text, pos, r, i |
| `internal/tui/keys_text.go` | s, msg, box, boxNum, panY, ... |
| `internal/tui/mouse.go` | r, closure@841, rowColor, ch, items, ... |
| `internal/tui/view.go` | rows, menuWidth, w, j, r, ... |

## Entry Points

- `internal/tui/view.go::model.overlayTooltipSidebar`
- `internal/tui/keys_text.go::model.handleBoxJumpKey`
- `internal/tui/mouse.go::model.drawMenuLevel`
- `internal/canvas/render.go::Canvas.drawConnectionWithPan`

## Connected Communities

- **tui +2 dirs · overlayMinimap** (13 cross-edges)
- **tui +2 dirs · overlayTooltipOnRenderResult** (10 cross-edges)
- **tui +2 dirs · activateMenuItem** (5 cross-edges)
- **canvas +2 dirs · RenderRaw** (4 cross-edges)
- **canvas +2 dirs · LoadFromFileWithPan** (4 cross-edges)
- **tui +2 dirs · model** (2 cross-edges)
- **canvas · segmentCells** (2 cross-edges)
- **canvas +2 dirs · SaveToFileWithPan** (2 cross-edges)
- **canvas · drawConnEndArrow** (2 cross-edges)
- **canvas +2 dirs · string** (2 cross-edges)
- **tui · handleMenuMouse** (2 cross-edges)
- **tui +2 dirs · statusLine** (1 cross-edges)
- **. +3 dirs** (1 cross-edges)
- **canvas · boxRune** (1 cross-edges)
- **. +2 dirs · moveContainedConnections** (1 cross-edges)
- **tui +1 dirs · handleFileInputKey** (1 cross-edges)

## How to Explore

```
analyze(operation:"communities", id:"community-14")
explore(operation:"context", task:"understand canvas +2 dirs · len", format:"gcx")
relations(operation:"usages", target:{symbol:"internal/tui/view.go::model.overlayTooltipSidebar"}, format:"gcx")
```

_`format: "gcx"` returns the [GCX1 compact wire format](../../docs/wire-format.md) — round-trippable, ~27% fewer tokens than JSON. Drop it for JSON output; agents using `@gortex/wire` or the Go `github.com/gortexhq/gcx-go` package decode either._
