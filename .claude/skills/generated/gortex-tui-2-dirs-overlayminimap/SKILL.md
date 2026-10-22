---
name: gortex-tui-2-dirs-overlayminimap
description: "Work in the tui +2 dirs · overlayMinimap area — 226 symbols across 8 files (68% cohesion)"
---

# tui +2 dirs · overlayMinimap

226 symbols | 8 files | 68% cohesion

## When to Use

Use this skill when working on files in:
- ``
- `internal/canvas/canvas.go`
- `internal/tui/app.go`
- `internal/tui/editing.go`
- `internal/tui/help.go`
- `internal/tui/mouse.go`
- `internal/tui/update.go`
- `internal/tui/view.go`

## Key Files

| File | Symbols |
|------|---------|
| `` | RuneCountInString, utf8, max, min |
| `internal/canvas/canvas.go` | conn, box, maxTextX, pt, oldWidth, ... |
| `internal/tui/app.go` | rows, rows, delta, length, clampFileScroll, ... |
| `internal/tui/editing.go` | editTarget, col, row, pos, text, ... |
| `internal/tui/help.go` | helpPageHeight, helpMaxScroll |
| `internal/tui/mouse.go` | item, rows, bufferBarOffset, x, anchor, ... |
| `internal/tui/update.go` | Update, msg |
| `internal/tui/view.go` | closure@265, contentWidth, selectedBox, closure@258, spanX, ... |

## Entry Points

- `internal/tui/view.go::model.View`
- `internal/tui/view.go::model.overlayMinimap`
- `internal/tui/mouse.go::model.handleFileMouse`
- `internal/tui/update.go::model.Update`

## Connected Communities

- **canvas +2 dirs · len** (26 cross-edges)
- **tui +2 dirs · activateMenuItem** (9 cross-edges)
- **tui +2 dirs · model** (7 cross-edges)
- **canvas +2 dirs · string** (7 cross-edges)
- **tui +2 dirs · overlayTooltipOnRenderResult** (5 cross-edges)
- **canvas +2 dirs · RenderRaw** (4 cross-edges)
- **canvas +1 dirs** (3 cross-edges)
- **tui +1 dirs · handleFileInputKey** (3 cross-edges)
- **tui +2 dirs · statusLine** (3 cross-edges)
- **tui +2 dirs · handleTextEditKey** (3 cross-edges)
- **canvas +2 dirs · LoadFromFileWithPan** (2 cross-edges)
- **tui · handleMenuMouse** (2 cross-edges)
- **canvas +2 dirs · SaveToFileWithPan** (1 cross-edges)
- **tui +1 dirs · handleNormalKey** (1 cross-edges)
- **canvas · reanchorConnectionsForResize** (1 cross-edges)
- **canvas · boxPtr** (1 cross-edges)
- **tui +1 dirs · finalizeMultiSelect** (1 cross-edges)

## How to Explore

```
analyze(operation:"communities", id:"community-31")
explore(operation:"context", task:"understand tui +2 dirs · overlayMinimap", format:"gcx")
relations(operation:"usages", target:{symbol:"internal/tui/view.go::model.View"}, format:"gcx")
```

_`format: "gcx"` returns the [GCX1 compact wire format](../../docs/wire-format.md) — round-trippable, ~27% fewer tokens than JSON. Drop it for JSON output; agents using `@gortex/wire` or the Go `github.com/gortexhq/gcx-go` package decode either._
