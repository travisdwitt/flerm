---
name: gortex-3-dirs
description: "Work in the . +3 dirs area — 90 symbols across 4 files (77% cohesion)"
---

# . +3 dirs

90 symbols | 4 files | 77% cohesion

## When to Use

Use this skill when working on files in:
- ``
- `cmd/flerm/main.go`
- `internal/config/config.go`
- `internal/tui/app.go`

## Key Files

| File | Symbols |
|------|---------|
| `` | TrimSpace, IsAbs, WriteFile, SplitN, ToLower, ... |
| `cmd/flerm/main.go` | dir, path, n, ext, err, ... |
| `internal/config/config.go` | data, err, err, path, info, ... |
| `internal/tui/app.go` | hasChartExt, i, m, r, initialModel, ... |

## Entry Points

- `internal/config/config.go::Load`

## Connected Communities

- **canvas +2 dirs · len** (3 cross-edges)
- **tui +2 dirs · overlayTooltipOnRenderResult** (2 cross-edges)
- **tui +3 dirs** (2 cross-edges)
- **tui +2 dirs · statusLine** (1 cross-edges)
- **. +2 dirs · moveContainedConnections** (1 cross-edges)
- **tui +2 dirs · model** (1 cross-edges)
- **tui +1 dirs · finalizeMultiSelect** (1 cross-edges)
- **canvas +2 dirs · string** (1 cross-edges)

## How to Explore

```
analyze(operation:"communities", id:"community-15")
explore(operation:"context", task:"understand . +3 dirs", format:"gcx")
relations(operation:"usages", target:{symbol:"internal/config/config.go::Load"}, format:"gcx")
```

_`format: "gcx"` returns the [GCX1 compact wire format](../../docs/wire-format.md) — round-trippable, ~27% fewer tokens than JSON. Drop it for JSON output; agents using `@gortex/wire` or the Go `github.com/gortexhq/gcx-go` package decode either._
