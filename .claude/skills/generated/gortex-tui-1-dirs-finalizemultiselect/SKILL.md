---
name: gortex-tui-1-dirs-finalizemultiselect
description: "Work in the tui +1 dirs · finalizeMultiSelect area — 91 symbols across 6 files (62% cohesion)"
---

# tui +1 dirs · finalizeMultiSelect

91 symbols | 6 files | 62% cohesion

## When to Use

Use this skill when working on files in:
- `internal/canvas/connection.go`
- `internal/tui/aliases.go`
- `internal/tui/keys_modes.go`
- `internal/tui/logos.go`
- `internal/tui/mouse.go`
- `internal/tui/move.go`

## Key Files

| File | Symbols |
|------|---------|
| `internal/canvas/connection.go` | boxID, result, SnapshotConnections, conn, conn, ... |
| `internal/tui/aliases.go` | Connection, point |
| `internal/tui/keys_modes.go` | key, d, msg, ok, handleMoveKey |
| `internal/tui/logos.go` | p, overlay, particleOverlay |
| `internal/tui/mouse.go` | box, worldX, canvas, beginBoxDrag, boxID, ... |
| `internal/tui/move.go` | box, bottom, hasGroupSelection, moved, moved, ... |

## Entry Points

- `internal/tui/move.go::model.finalizeMultiSelect`
- `internal/tui/move.go::model.commitGroupMove`

## Connected Communities

- **tui +2 dirs · activateMenuItem** (18 cross-edges)
- **canvas +2 dirs · len** (10 cross-edges)
- **tui +2 dirs · overlayTooltipOnRenderResult** (8 cross-edges)
- **tui +2 dirs · overlayMinimap** (5 cross-edges)
- **canvas +2 dirs · RenderRaw** (4 cross-edges)
- **. +2 dirs · moveContainedConnections** (4 cross-edges)
- **canvas +2 dirs · string** (3 cross-edges)
- **tui +1 dirs · handleNormalKey** (2 cross-edges)
- **canvas · box** (1 cross-edges)

## How to Explore

```
analyze(operation:"communities", id:"community-27")
explore(operation:"context", task:"understand tui +1 dirs · finalizeMultiSelect", format:"gcx")
relations(operation:"usages", target:{symbol:"internal/tui/move.go::model.finalizeMultiSelect"}, format:"gcx")
```

_`format: "gcx"` returns the [GCX1 compact wire format](../../docs/wire-format.md) — round-trippable, ~27% fewer tokens than JSON. Drop it for JSON output; agents using `@gortex/wire` or the Go `github.com/gortexhq/gcx-go` package decode either._
