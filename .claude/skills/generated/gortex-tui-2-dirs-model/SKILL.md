---
name: gortex-tui-2-dirs-model
description: "Work in the tui +2 dirs · model area — 230 symbols across 11 files (64% cohesion)"
---

# tui +2 dirs · model

230 symbols | 11 files | 64% cohesion

## When to Use

Use this skill when working on files in:
- ``
- `internal/canvas/canvas.go`
- `internal/tui/app.go`
- `internal/tui/keys_modes.go`
- `internal/tui/logos.go`
- `internal/tui/mouse.go`
- `internal/tui/navigation.go`
- `internal/tui/types.go`
- `internal/tui/undo.go`
- `internal/tui/update.go`
- `internal/tui/utils.go`

## Key Files

| File | Symbols |
|------|---------|
| `` | Base |
| `internal/canvas/canvas.go` | NewCanvas |
| `internal/tui/app.go` | closeBuffer, panY, panX, openChart, canvas, ... |
| `internal/tui/keys_modes.go` | key, handleStartupKey, handleMultiSelectKey, worldX, ok, ... |
| `internal/tui/logos.go` | gravity, effectTick, closure@104, inStartMenuFlow, effectTickMsg, ... |
| `internal/tui/mouse.go` | mouseCanvasPos, handleNormalMouse, canvasY, dx, canvasX, ... |
| `internal/tui/navigation.go` | key, dx, y, oldText, cx, ... |
| `internal/tui/types.go` | panLastX, panY, canvas, overwriteOK, mode, ... |
| `internal/tui/undo.go` | action, n, action, redo, buf, ... |
| `internal/tui/update.go` | Init |
| `internal/tui/utils.go` | getPanOffset, buf, panY, panX, buf, ... |

## Entry Points

- `internal/tui/mouse.go::model.handleNormalMouse`
- `internal/tui/keys_modes.go::model.handleStartupKey`

## Connected Communities

- **tui +2 dirs · activateMenuItem** (13 cross-edges)
- **tui +2 dirs · overlayMinimap** (9 cross-edges)
- **tui +1 dirs · handleNormalKey** (9 cross-edges)
- **tui +1 dirs · finalizeMultiSelect** (6 cross-edges)
- **canvas +2 dirs · RenderRaw** (5 cross-edges)
- **tui +2 dirs · overlayTooltipOnRenderResult** (4 cross-edges)
- **canvas +2 dirs · len** (4 cross-edges)
- **canvas +2 dirs · applyAction** (3 cross-edges)
- **canvas +2 dirs · string** (3 cross-edges)
- **tui +2 dirs · Run** (2 cross-edges)
- **canvas +2 dirs · LoadFromFileWithPan** (1 cross-edges)
- **tui · paintCells** (1 cross-edges)
- **tui +1 dirs · handleFileInputKey** (1 cross-edges)
- **tui · handleMenuMouse** (1 cross-edges)
- **. +2 dirs · scanTxtFiles** (1 cross-edges)

## How to Explore

```
analyze(operation:"communities", id:"community-24")
explore(operation:"context", task:"understand tui +2 dirs · model", format:"gcx")
relations(operation:"usages", target:{symbol:"internal/tui/mouse.go::model.handleNormalMouse"}, format:"gcx")
```

_`format: "gcx"` returns the [GCX1 compact wire format](../../docs/wire-format.md) — round-trippable, ~27% fewer tokens than JSON. Drop it for JSON output; agents using `@gortex/wire` or the Go `github.com/gortexhq/gcx-go` package decode either._
