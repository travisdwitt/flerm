---
name: gortex-canvas-2-dirs-loadfromfilewithpan
description: "Work in the canvas +2 dirs · LoadFromFileWithPan area — 106 symbols across 4 files (73% cohesion)"
---

# canvas +2 dirs · LoadFromFileWithPan

106 symbols | 4 files | 73% cohesion

## When to Use

Use this skill when working on files in:
- ``
- `cmd/flerm/main.go`
- `internal/canvas/box.go`
- `internal/canvas/storage.go`

## Key Files

| File | Symbols |
|------|---------|
| `` | Errorf, error, Cut, Atoi, strconv, ... |
| `cmd/flerm/main.go` | writeDefaultConfig, err, name, err, err, ... |
| `internal/canvas/box.go` | SetText, text |
| `internal/canvas/storage.go` | file, text, n, box, panY, ... |

## Entry Points

- `internal/canvas/storage.go::Canvas.LoadFromFileWithPan`
- `internal/canvas/storage.go::Canvas.loadSectionLine`

## Connected Communities

- **canvas +2 dirs · len** (13 cross-edges)
- **. +3 dirs** (9 cross-edges)
- **canvas +2 dirs · SaveToFileWithPan** (4 cross-edges)
- **canvas +2 dirs · RenderRaw** (3 cross-edges)
- **tui +2 dirs · overlayTooltipOnRenderResult** (3 cross-edges)
- **canvas · boxPtr** (2 cross-edges)
- **tui +2 dirs · model** (1 cross-edges)
- **tui +3 dirs** (1 cross-edges)
- **canvas · box** (1 cross-edges)
- **canvas +2 dirs · applyAction** (1 cross-edges)
- **tui +2 dirs · overlayMinimap** (1 cross-edges)
- **canvas +2 dirs · string** (1 cross-edges)
- **canvas · textPtr** (1 cross-edges)
- **canvas · AddTextWithID** (1 cross-edges)

## How to Explore

```
analyze(operation:"communities", id:"community-42")
explore(operation:"context", task:"understand canvas +2 dirs · LoadFromFileWithPan", format:"gcx")
relations(operation:"usages", target:{symbol:"internal/canvas/storage.go::Canvas.LoadFromFileWithPan"}, format:"gcx")
```

_`format: "gcx"` returns the [GCX1 compact wire format](../../docs/wire-format.md) — round-trippable, ~27% fewer tokens than JSON. Drop it for JSON output; agents using `@gortex/wire` or the Go `github.com/gortexhq/gcx-go` package decode either._
