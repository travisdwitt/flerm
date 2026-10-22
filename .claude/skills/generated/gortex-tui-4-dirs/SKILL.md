---
name: gortex-tui-4-dirs
description: "Work in the tui +4 dirs area — 106 symbols across 6 files (87% cohesion)"
---

# tui +4 dirs

106 symbols | 6 files | 87% cohesion

## When to Use

Use this skill when working on files in:
- ``
- `external-call::dep:github.com/fogleman/gg`
- `external-call::dep:github.com/golang/freetype/truetype`
- `internal/canvas/png.go`
- `internal/tui/logos.go`
- `internal/tui/view.go`

## Key Files

| File | Symbols |
|------|---------|
| `` | float64, Float64, rand, Cos, Sin, ... |
| `external-call::dep:github.com/fogleman/gg` | github.com/fogleman/gg |
| `external-call::dep:github.com/golang/freetype/truetype` | github.com/golang/freetype/truetype |
| `internal/canvas/png.go` | hasH, hw, alpha, dc, runColor, ... |
| `internal/tui/logos.go` | i, sparks, colors, rocket, y, ... |
| `internal/tui/view.go` | startupLayout, logoW, logoOrigin, boxW, logo, ... |

## Entry Points

- `internal/canvas/png.go::Canvas.ExportToPNG`
- `internal/canvas/png.go::drawCellRune`
- `internal/tui/logos.go::model.spawn`

## Connected Communities

- **canvas +2 dirs · len** (6 cross-edges)
- **tui +2 dirs · overlayTooltipOnRenderResult** (5 cross-edges)
- **tui +2 dirs · overlayMinimap** (3 cross-edges)
- **canvas +2 dirs · RenderRaw** (3 cross-edges)
- **canvas +2 dirs · LoadFromFileWithPan** (2 cross-edges)
- **tui +2 dirs · model** (1 cross-edges)
- **canvas +2 dirs · string** (1 cross-edges)

## How to Explore

```
analyze(operation:"communities", id:"community-40")
explore(operation:"context", task:"understand tui +4 dirs", format:"gcx")
relations(operation:"usages", target:{symbol:"internal/canvas/png.go::Canvas.ExportToPNG"}, format:"gcx")
```

_`format: "gcx"` returns the [GCX1 compact wire format](../../docs/wire-format.md) — round-trippable, ~27% fewer tokens than JSON. Drop it for JSON output; agents using `@gortex/wire` or the Go `github.com/gortexhq/gcx-go` package decode either._
