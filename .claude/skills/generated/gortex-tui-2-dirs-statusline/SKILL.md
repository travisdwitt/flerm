---
name: gortex-tui-2-dirs-statusline
description: "Work in the tui +2 dirs · statusLine area — 68 symbols across 4 files (72% cohesion)"
---

# tui +2 dirs · statusLine

68 symbols | 4 files | 72% cohesion

## When to Use

Use this skill when working on files in:
- ``
- `internal/canvas/describe.go`
- `internal/tui/utils.go`
- `internal/tui/view.go`

## Key Files

| File | Symbols |
|------|---------|
| `` | Sprintf |
| `internal/canvas/describe.go` | n, id, ok, s, noun, ... |
| `internal/tui/utils.go` | unsavedChanges, buf |
| `internal/tui/view.go` | swatch, fileMenuTitle, status, pos, canvas, ... |

## Entry Points

- `internal/tui/view.go::model.statusLine`
- `internal/tui/view.go::model.cursorObject`

## Connected Communities

- **canvas +2 dirs · len** (4 cross-edges)
- **tui +2 dirs · activateMenuItem** (4 cross-edges)
- **tui +1 dirs · handleNormalKey** (3 cross-edges)
- **canvas +2 dirs · string** (3 cross-edges)
- **canvas +2 dirs · SaveToFileWithPan** (3 cross-edges)
- **tui +2 dirs · handleTextEditKey** (2 cross-edges)
- **canvas · box** (1 cross-edges)
- **tui +3 dirs** (1 cross-edges)
- **tui +2 dirs · overlayTooltipOnRenderResult** (1 cross-edges)
- **tui +1 dirs · finalizeMultiSelect** (1 cross-edges)
- **tui +2 dirs · model** (1 cross-edges)

## How to Explore

```
analyze(operation:"communities", id:"community-32")
explore(operation:"context", task:"understand tui +2 dirs · statusLine", format:"gcx")
relations(operation:"usages", target:{symbol:"internal/tui/view.go::model.statusLine"}, format:"gcx")
```

_`format: "gcx"` returns the [GCX1 compact wire format](../../docs/wire-format.md) — round-trippable, ~27% fewer tokens than JSON. Drop it for JSON output; agents using `@gortex/wire` or the Go `github.com/gortexhq/gcx-go` package decode either._
