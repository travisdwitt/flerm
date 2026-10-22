---
name: gortex-canvas-2-dirs-string
description: "Work in the canvas +2 dirs · string area — 133 symbols across 8 files (60% cohesion)"
---

# canvas +2 dirs · string

133 symbols | 8 files | 60% cohesion

## When to Use

Use this skill when working on files in:
- ``
- `internal/canvas/color.go`
- `internal/canvas/describe.go`
- `internal/canvas/render.go`
- `internal/canvas/storage.go`
- `internal/tui/app.go`
- `internal/tui/utils.go`
- `internal/tui/view.go`

## Key Files

| File | Symbols |
|------|---------|
| `` | string, exec, copy, Command |
| `internal/canvas/color.go` | index, code256, Ansi, ColorFG, code16, ... |
| `internal/canvas/describe.go` | String |
| `internal/canvas/render.go` | line, row, cellColor, blank, colorIndex, ... |
| `internal/canvas/storage.go` | i, line, splitBoxLine, fields |
| `internal/tui/app.go` | err, rememberChart, saveChart, path, path, ... |
| `internal/tui/utils.go` | err, readClipboardText, err, output, output |
| `internal/tui/view.go` | editCaret, isFrame, displayLen, e, renderStartupMenu, ... |

## Entry Points

- `internal/canvas/render.go::RenderResult.ApplyColors`
- `internal/tui/view.go::model.renderBufferBar`
- `internal/tui/view.go::model.renderStartupMenu`

## Connected Communities

- **canvas +2 dirs · len** (13 cross-edges)
- **tui +2 dirs · overlayTooltipOnRenderResult** (3 cross-edges)
- **tui +2 dirs · overlayMinimap** (2 cross-edges)
- **tui +2 dirs · model** (2 cross-edges)
- **canvas +2 dirs · RenderRaw** (2 cross-edges)
- **tui +2 dirs · statusLine** (2 cross-edges)
- **. +3 dirs** (2 cross-edges)
- **canvas +2 dirs · SaveToFileWithPan** (1 cross-edges)
- **tui +1 dirs · finalizeMultiSelect** (1 cross-edges)
- **tui +2 dirs · handleTextEditKey** (1 cross-edges)
- **tui +1 dirs · handleFileInputKey** (1 cross-edges)

## How to Explore

```
analyze(operation:"communities", id:"community-30")
explore(operation:"context", task:"understand canvas +2 dirs · string", format:"gcx")
relations(operation:"usages", target:{symbol:"internal/canvas/render.go::RenderResult.ApplyColors"}, format:"gcx")
```

_`format: "gcx"` returns the [GCX1 compact wire format](../../docs/wire-format.md) — round-trippable, ~27% fewer tokens than JSON. Drop it for JSON output; agents using `@gortex/wire` or the Go `github.com/gortexhq/gcx-go` package decode either._
