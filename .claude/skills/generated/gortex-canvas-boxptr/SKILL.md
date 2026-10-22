---
name: gortex-canvas-boxptr
description: "Work in the canvas · boxPtr area — 59 symbols across 3 files (47% cohesion)"
---

# canvas · boxPtr

59 symbols | 3 files | 47% cohesion

## When to Use

Use this skill when working on files in:
- `internal/canvas/canvas.go`
- `internal/canvas/constants.go`
- `internal/canvas/render.go`

## Key Files

| File | Symbols |
|------|---------|
| `internal/canvas/canvas.go` | oldWidth, b, id, highlights, id, ... |
| `internal/canvas/constants.go` | BorderStyle |
| `internal/canvas/render.go` | style, borderRunes, selected |

## Connected Communities

- **canvas +2 dirs · len** (2 cross-edges)
- **. +1 dirs · createFlexibleWaypoints** (1 cross-edges)
- **tui +2 dirs · overlayMinimap** (1 cross-edges)
- **canvas · reanchorConnectionsForResize** (1 cross-edges)

## How to Explore

```
analyze(operation:"communities", id:"community-1")
explore(operation:"context", task:"understand canvas · boxPtr", format:"gcx")
```

_`format: "gcx"` returns the [GCX1 compact wire format](../../docs/wire-format.md) — round-trippable, ~27% fewer tokens than JSON. Drop it for JSON output; agents using `@gortex/wire` or the Go `github.com/gortexhq/gcx-go` package decode either._
