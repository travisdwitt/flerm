# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run

```bash
go build -o flerm    # Build the binary
./flerm              # Run the application
```

## Architecture

Flerm is a terminal-based flowchart and mind-map editor built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

### Core Structure

**Model-View-Update Pattern**: The app follows Bubble Tea's MVU architecture where `model` is the central state container.

- **main.go** (~4000 lines): Contains `model` struct methods, keyboard input handling, the `Update()` and `View()` functions, save/load logic, and rendering coordination
- **canvas.go** (~3500 lines): The `Canvas` type manages all diagram elements (boxes, connections, texts, highlights) and rendering via `RenderRaw()`
- **types.go**: Data structures for `model`, `Buffer`, `Box`, `Connection`, `Text`, action types for undo/redo
- **constants.go**: Mode enums (`ModeNormal`, `ModeEditing`, `ModeResize`, etc.), action types, border styles

### Key Concepts

**Buffers**: Multiple flowcharts can be open simultaneously. Each `Buffer` contains its own `Canvas`, undo/redo stacks, filename, and pan offset.

**Canvas Elements**:
- `Box`: Bordered rectangle with text content, optional title, z-level (0-3 for shadows), border style
- `Connection`: Lines between boxes with waypoints and arrow states
- `Text`: Freeform text placed anywhere on canvas
- `highlights`: Map of (x,y) -> color index for cell highlighting

**Modes**: The editor uses modal editing (similar to vim). Current mode stored in `model.mode`:
- `ModeNormal`: Default navigation and commands
- `ModeEditing`: Editing text inside a box
- `ModeResize`/`ModeMove`: Resizing or moving elements
- `ModeMultiSelect`: Select and move multiple elements
- `ModeFileInput`: File save/open dialogs
- `ModeHighlight`: Drawing colored highlights

**Undo/Redo**: Action-based system in `undo.go`. Each undoable operation creates an `Action` with forward data and inverse data, stored on per-buffer stacks.

### Supporting Files

- **navigation.go**: Cursor movement and pan mode handling
- **utils.go**: Buffer/canvas accessors, clipboard operations, RTF/HTML text extraction
- **config.go**: Loads `~/.flermrc` for save directory, start menu, confirmations settings
- **export.go**: Visual TXT export (PNG export is in canvas.go)
- **help.go**: In-app help text

### File Format

Flowcharts save as `.sav` files with sections: `FLOWCHART`, `BOXES:N`, `CONNECTIONS:N`, `TEXTS:N`. Format details in README.md.

## Response Style
- Provide direct technical answers only
- No apologetic language
- No validation phrases ("you're right", "good point", etc.)
- No emotional language or personality
- No preamble or pleasantries
- Get straight to the solution
