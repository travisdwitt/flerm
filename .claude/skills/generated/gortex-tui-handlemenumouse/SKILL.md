---
name: gortex-tui-handlemenumouse
description: "Work in the tui · handleMenuMouse area — 57 symbols across 3 files (70% cohesion)"
---

# tui · handleMenuMouse

57 symbols | 3 files | 70% cohesion

## When to Use

Use this skill when working on files in:
- `internal/tui/keys_modes.go`
- `internal/tui/mouse.go`
- `internal/tui/types.go`

## Key Files

| File | Symbols |
|------|---------|
| `internal/tui/keys_modes.go` | handleContextMenuKey, items, idx, msg |
| `internal/tui/mouse.go` | closeContextMenu, levels, h, levels, overlayContextMenu, ... |
| `internal/tui/types.go` | items, menuLevel, y, x, index |

## Entry Points

- `internal/tui/keys_modes.go::model.handleContextMenuKey`
- `internal/tui/mouse.go::model.handleMenuMouse`

## Connected Communities

- **canvas +2 dirs · len** (13 cross-edges)
- **tui +2 dirs · overlayMinimap** (3 cross-edges)
- **tui +2 dirs · activateMenuItem** (2 cross-edges)
- **tui +2 dirs · overlayTooltipOnRenderResult** (2 cross-edges)
- **tui +1 dirs · handleNormalKey** (1 cross-edges)
- **canvas +2 dirs · string** (1 cross-edges)

## How to Explore

```
analyze(operation:"communities", id:"community-25")
explore(operation:"context", task:"understand tui · handleMenuMouse", format:"gcx")
relations(operation:"usages", target:{symbol:"internal/tui/keys_modes.go::model.handleContextMenuKey"}, format:"gcx")
```

_`format: "gcx"` returns the [GCX1 compact wire format](../../docs/wire-format.md) — round-trippable, ~27% fewer tokens than JSON. Drop it for JSON output; agents using `@gortex/wire` or the Go `github.com/gortexhq/gcx-go` package decode either._
