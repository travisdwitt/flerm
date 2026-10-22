---
name: gortex-canvas-2-dirs-savetofilewithpan
description: "Work in the canvas +2 dirs · SaveToFileWithPan area — 111 symbols across 5 files (73% cohesion)"
---

# canvas +2 dirs · SaveToFileWithPan

111 symbols | 5 files | 73% cohesion

## When to Use

Use this skill when working on files in:
- ``
- `internal/canvas/box.go`
- `internal/canvas/describe.go`
- `internal/canvas/storage.go`
- `internal/tui/export.go`

## Key Files

| File | Symbols |
|------|---------|
| `` | Fprintln, Fields, TrimRight, ReplaceAll, Fprintf, ... |
| `internal/canvas/box.go` | GetText, GetText |
| `internal/canvas/describe.go` | far, near, near, field, text, ... |
| `internal/canvas/storage.go` | filename, text, closure@87, wp, box, ... |
| `internal/tui/export.go` | i, minY, link, canvas, r, ... |

## Entry Points

- `internal/canvas/storage.go::Canvas.SaveToFileWithPan`
- `internal/tui/export.go::model.exportMermaid`
- `internal/canvas/describe.go::Canvas.Describe`
- `internal/tui/export.go::model.exportVisualTXT`

## Connected Communities

- **canvas +2 dirs · len** (20 cross-edges)
- **tui +2 dirs · statusLine** (6 cross-edges)
- **tui +2 dirs · activateMenuItem** (4 cross-edges)
- **tui +2 dirs · overlayMinimap** (3 cross-edges)
- **tui +2 dirs · overlayTooltipOnRenderResult** (3 cross-edges)
- **canvas · arrowNote** (3 cross-edges)
- **canvas +2 dirs · string** (2 cross-edges)
- **canvas +2 dirs · LoadFromFileWithPan** (2 cross-edges)
- **canvas +2 dirs · RenderRaw** (2 cross-edges)
- **. +3 dirs** (1 cross-edges)

## How to Explore

```
analyze(operation:"communities", id:"community-38")
explore(operation:"context", task:"understand canvas +2 dirs · SaveToFileWithPan", format:"gcx")
relations(operation:"usages", target:{symbol:"internal/canvas/storage.go::Canvas.SaveToFileWithPan"}, format:"gcx")
```

_`format: "gcx"` returns the [GCX1 compact wire format](../../docs/wire-format.md) — round-trippable, ~27% fewer tokens than JSON. Drop it for JSON output; agents using `@gortex/wire` or the Go `github.com/gortexhq/gcx-go` package decode either._
