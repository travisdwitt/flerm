---
name: gortex-2-dirs-movecontainedconnections
description: "Work in the . +2 dirs · moveContainedConnections area — 45 symbols across 3 files (67% cohesion)"
---

# . +2 dirs · moveContainedConnections

45 symbols | 3 files | 67% cohesion

## When to Use

Use this skill when working on files in:
- ``
- `internal/canvas/connection.go`
- `internal/tui/move.go`

## Key Files

| File | Symbols |
|------|---------|
| `` | Clone, Index, slices, Equal, DeleteFunc, ... |
| `internal/canvas/connection.go` | a, ToY, ArrowFrom, FromX, FromID, ... |
| `internal/tui/move.go` | conn, moveContainedConnections, deltaX, moveTo, moveFrom, ... |

## Connected Communities

- **tui +2 dirs · activateMenuItem** (2 cross-edges)
- **canvas +2 dirs · len** (1 cross-edges)

## How to Explore

```
analyze(operation:"communities", id:"community-6")
explore(operation:"context", task:"understand . +2 dirs · moveContainedConnections", format:"gcx")
```

_`format: "gcx"` returns the [GCX1 compact wire format](../../docs/wire-format.md) — round-trippable, ~27% fewer tokens than JSON. Drop it for JSON output; agents using `@gortex/wire` or the Go `github.com/gortexhq/gcx-go` package decode either._
