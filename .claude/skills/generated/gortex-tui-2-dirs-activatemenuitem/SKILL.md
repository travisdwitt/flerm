---
name: gortex-tui-2-dirs-activatemenuitem
description: "Work in the tui +2 dirs · activateMenuItem area — 207 symbols across 15 files (65% cohesion)"
---

# tui +2 dirs · activateMenuItem

207 symbols | 15 files | 65% cohesion

## When to Use

Use this skill when working on files in:
- ``
- `internal/canvas/accessors.go`
- `internal/canvas/canvas.go`
- `internal/tui/aliases.go`
- `internal/tui/app.go`
- `internal/tui/constants.go`
- `internal/tui/editing.go`
- `internal/tui/keys_modes.go`
- `internal/tui/keys_normal.go`
- `internal/tui/mouse.go`
- `internal/tui/move.go`
- `internal/tui/types.go`
- `internal/tui/undo.go`
- `internal/tui/utils.go`
- `internal/tui/view.go`

## Key Files

| File | Symbols |
|------|---------|
| `` | any |
| `internal/canvas/accessors.go` | Texts, Boxes, Connections |
| `internal/canvas/canvas.go` | deltaY, deltaX, t, id, MoveText |
| `internal/tui/aliases.go` | Canvas |
| `internal/tui/app.go` | ensureCursorInBounds |
| `internal/tui/constants.go` | ActionType, Mode, MenuAction |
| `internal/tui/editing.go` | changed, boxID, id, typ, beginEdit, ... |
| `internal/tui/keys_modes.go` | key, m, handleResizeKey, handleConfirmKey, op, ... |
| `internal/tui/keys_normal.go` | box, anchorY, duplicateByID, deleteTextByID, canvas, ... |
| `internal/tui/mouse.go` | conn, desiredY, old, kind, text, ... |
| `internal/tui/move.go` | recordTextMove, idx, ok, handleSingleElementMove, dy, ... |
| `internal/tui/types.go` | OldText, ID, EditData, NewText |
| `internal/tui/undo.go` | kind, applyObjectColor, color, id |
| `internal/tui/utils.go` | actionType, data, buf, getCanvas, recordAction, ... |
| `internal/tui/view.go` | x, screenX, worldX, updateTooltip, y, ... |

## Entry Points

- `internal/tui/mouse.go::model.activateMenuItem`
- `internal/tui/move.go::model.handleMultiSelectMove`
- `internal/tui/keys_modes.go::model.handleConfirmKey`
- `internal/tui/keys_modes.go::model.handleResizeKey`

## Connected Communities

- **canvas +2 dirs · len** (29 cross-edges)
- **canvas · boxPtr** (9 cross-edges)
- **tui +2 dirs · model** (8 cross-edges)
- **tui +2 dirs · overlayMinimap** (6 cross-edges)
- **canvas +2 dirs · applyAction** (5 cross-edges)
- **tui +1 dirs · handleNormalKey** (4 cross-edges)
- **canvas · textPtr** (3 cross-edges)
- **. +2 dirs · moveContainedConnections** (3 cross-edges)
- **canvas +2 dirs · RenderRaw** (3 cross-edges)
- **tui +3 dirs** (2 cross-edges)
- **canvas · highlightsForCells** (2 cross-edges)
- **canvas +2 dirs · string** (2 cross-edges)
- **tui +2 dirs · handleTextEditKey** (2 cross-edges)
- **tui +1 dirs · finalizeMultiSelect** (2 cross-edges)
- **tui +2 dirs · overlayTooltipOnRenderResult** (2 cross-edges)
- **canvas · AddTextWithID** (1 cross-edges)
- **canvas · box** (1 cross-edges)
- **tui · handleMenuMouse** (1 cross-edges)
- **canvas +1 dirs** (1 cross-edges)

## How to Explore

```
analyze(operation:"communities", id:"community-23")
explore(operation:"context", task:"understand tui +2 dirs · activateMenuItem", format:"gcx")
relations(operation:"usages", target:{symbol:"internal/tui/mouse.go::model.activateMenuItem"}, format:"gcx")
```

_`format: "gcx"` returns the [GCX1 compact wire format](../../docs/wire-format.md) — round-trippable, ~27% fewer tokens than JSON. Drop it for JSON output; agents using `@gortex/wire` or the Go `github.com/gortexhq/gcx-go` package decode either._
