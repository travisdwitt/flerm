---
name: gortex-tui-2-dirs-overlaytooltiponrenderresult
description: "Work in the tui +2 dirs · overlayTooltipOnRenderResult area — 141 symbols across 4 files (60% cohesion)"
---

# tui +2 dirs · overlayTooltipOnRenderResult

141 symbols | 4 files | 60% cohesion

## When to Use

Use this skill when working on files in:
- ``
- `internal/canvas/canvas.go`
- `internal/tui/logos.go`
- `internal/tui/view.go`

## Key Files

| File | Symbols |
|------|---------|
| `` | append, ContainsRune, Repeat |
| `internal/canvas/canvas.go` | conn, connPoints, pts, pts, pts |
| `internal/tui/logos.go` | kept, p, born, advanceEffect, kept |
| `internal/tui/view.go` | tooltipY, rows, closure@722, wrapWords, i, ... |

## Entry Points

- `internal/tui/view.go::model.overlayTooltipOnRenderResult`
- `internal/tui/view.go::model.renderFileMenu`

## Connected Communities

- **canvas +2 dirs · len** (23 cross-edges)
- **tui +2 dirs · overlayMinimap** (11 cross-edges)
- **canvas +2 dirs · string** (7 cross-edges)
- **tui +4 dirs** (3 cross-edges)
- **tui +2 dirs · activateMenuItem** (3 cross-edges)
- **tui +1 dirs · handleFileInputKey** (2 cross-edges)
- **canvas +2 dirs · RenderRaw** (2 cross-edges)
- **tui +2 dirs · statusLine** (2 cross-edges)
- **canvas +2 dirs · SaveToFileWithPan** (1 cross-edges)
- **tui +2 dirs · model** (1 cross-edges)
- **canvas +2 dirs · LoadFromFileWithPan** (1 cross-edges)
- **canvas · box** (1 cross-edges)

## How to Explore

```
analyze(operation:"communities", id:"community-33")
explore(operation:"context", task:"understand tui +2 dirs · overlayTooltipOnRenderResult", format:"gcx")
relations(operation:"usages", target:{symbol:"internal/tui/view.go::model.overlayTooltipOnRenderResult"}, format:"gcx")
```

_`format: "gcx"` returns the [GCX1 compact wire format](../../docs/wire-format.md) — round-trippable, ~27% fewer tokens than JSON. Drop it for JSON output; agents using `@gortex/wire` or the Go `github.com/gortexhq/gcx-go` package decode either._
