---
name: flerm
description: Create, edit, inspect, validate and export flerm flowcharts (.flerm files) with the flerm CLI or its MCP tools. Use when asked to draw, read or change a flowchart or diagram stored as a .flerm file, or to convert Mermaid to flerm.
---

# flerm

Never edit `.flerm` files by hand. Use the CLI (or the MCP tools of the same names, with `-` written as `_`).

## Workflow

1. New chart from a diagram: write Mermaid (`flowchart TD` or `LR`) and run `flerm import <src.mmd|-> <chart>`. It lays the boxes out and prints the box ids.
2. Existing chart: `flerm describe <chart> --format json` first. Ids are positions and shift when something is deleted, so re-describe after `rm-box` / `rm-conn`.
3. Small edits: `add-box`, `set-box`, `connect`, `rm-box`, `rm-conn`.
4. Always finish with `flerm validate <chart>`; fix every `line N:` error. `warning:` lines are advisory.
5. To show the result: `flerm export <chart> --format txt` (visual) or `--format mermaid` (structure).

## Commands

- `flerm import <source> <chart> [--force true] [--append true]` source is a path, `-` for stdin, or inline text; Mermaid or JSON `{"direction","boxes":[{"id","text","title","border"}],"edges":[{"from","to","arrow","label"}]}`. Edge labels become text labels. Subgraphs and styling are dropped with a `warning:`. `--append true` adds the graph below an existing chart; ids continue after the existing boxes.
- `flerm add-box <chart> --text T [--title T] [--tooltip T] [--at x,y] [--border ascii|single|double|rounded] [--color Name|none] [--size WxH] [--shadow 0-3]`
- `flerm set-box <chart> <id> [same options]` only given options change; an empty value clears.
- `flerm connect <chart> <from> <to> [--arrow to|from|both|none]`
- `flerm rm-box <chart> <id>` / `flerm rm-conn <chart> <id>` ids above the deleted one shift down; delete from the highest id down, or `find` again.
- `flerm add-text <chart> --text T [--at x,y] [--color Name]`, `flerm set-text <chart> <id> [--text T] [--at x,y] [--color Name]`, `flerm rm-text <chart> <id>`
- `flerm find <chart> <query>` box ids whose title, text or tooltip contain the query.
- `flerm batch <chart> '<json array>'` e.g. `[{"op":"add-box","text":"A"},{"op":"connect","from":0,"to":1}]`. One read and one write, all or nothing; prefer it over many single calls. A new box takes the next id, so later ops in the same batch can refer to it.
- `flerm describe <chart> [--format json]`, `flerm validate <chart>`, `flerm export <chart> --format mermaid|txt|png [--out file]`

Use `\n` for line breaks in text, title and tooltip. Colors: Gray, Red, Green, Yellow, Blue, Magenta, Cyan, White, Black and their Bright variants.

## Notes

- `import` refuses to overwrite an existing chart without `--force true`.
- Boxes size themselves to their text; do not try to set width or height.
- Do not open the interactive editor (`flerm <file>` with no command) from an agent session.
