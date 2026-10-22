---
name: gortex-tui-1-dirs-handlenormalkey
description: "Work in the tui +1 dirs · handleNormalKey area — 178 symbols across 8 files (67% cohesion)"
---

# tui +1 dirs · handleNormalKey

178 symbols | 8 files | 67% cohesion

## When to Use

Use this skill when working on files in:
- `internal/canvas/canvas.go`
- `internal/canvas/connection.go`
- `internal/canvas/highlight.go`
- `internal/tui/constants.go`
- `internal/tui/keys_normal.go`
- `internal/tui/mouse.go`
- `internal/tui/move.go`
- `internal/tui/utils.go`

## Key Files

| File | Symbols |
|------|---------|
| `internal/canvas/canvas.go` | y, text, box, i, line, ... |
| `internal/canvas/connection.go` | bestConnIdx, i, bestX, FindNearestPointOnConnection, bestDist, ... |
| `internal/canvas/highlight.go` | ok, x, GetHighlight, color, y |
| `internal/tui/constants.go` | FileOperation, ConfirmAction |
| `internal/tui/keys_normal.go` | content, recordHighlights, textID, x, delta, ... |
| `internal/tui/mouse.go` | worldY, openContextMenu, canvasX, selectAtMouse, panX, ... |
| `internal/tui/move.go` | worldY, color, canvas, beginMove, boxID, ... |
| `internal/tui/utils.go` | action, op, id, beginFileOp, name, ... |

## Entry Points

- `internal/tui/keys_normal.go::model.handleNormalKey`
- `internal/tui/keys_normal.go::model.deleteAt`

## Connected Communities

- **tui +2 dirs · activateMenuItem** (38 cross-edges)
- **tui +2 dirs · model** (11 cross-edges)
- **canvas · boxPtr** (7 cross-edges)
- **canvas +2 dirs · len** (6 cross-edges)
- **tui · paintCells** (6 cross-edges)
- **canvas +1 dirs** (5 cross-edges)
- **tui +2 dirs · overlayTooltipOnRenderResult** (4 cross-edges)
- **canvas · box** (3 cross-edges)
- **tui +1 dirs · finalizeMultiSelect** (3 cross-edges)
- **canvas +2 dirs · RenderRaw** (3 cross-edges)
- **tui · buildMenuItems** (3 cross-edges)
- **canvas · drawConnEndArrow** (2 cross-edges)
- **canvas · closestOnSegment** (2 cross-edges)
- **tui +2 dirs · overlayMinimap** (2 cross-edges)
- **tui +2 dirs · statusLine** (1 cross-edges)
- **. +2 dirs · moveContainedConnections** (1 cross-edges)
- **canvas · segmentCells** (1 cross-edges)
- **canvas · textPtr** (1 cross-edges)
- **canvas +2 dirs · string** (1 cross-edges)
- **. +2 dirs · scanTxtFiles** (1 cross-edges)

## How to Explore

```
analyze(operation:"communities", id:"community-19")
explore(operation:"context", task:"understand tui +1 dirs · handleNormalKey", format:"gcx")
relations(operation:"usages", target:{symbol:"internal/tui/keys_normal.go::model.handleNormalKey"}, format:"gcx")
```

_`format: "gcx"` returns the [GCX1 compact wire format](../../docs/wire-format.md) — round-trippable, ~27% fewer tokens than JSON. Drop it for JSON output; agents using `@gortex/wire` or the Go `github.com/gortexhq/gcx-go` package decode either._
