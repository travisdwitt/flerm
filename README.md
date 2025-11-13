# Flerm - Terminal Flowchart Editor

A simple, terminal-based flowchart editor with support for boxes, connections, and text.

## Installation

```bash
go build -o flerm
```

## Usage

```bash
./flerm
```

## Controls

### Navigation
- `h/←/j/↓/k/↑/l/→` - Move cursor around the screen
- `Shift+h/j/k/l` - Move cursor 2x faster

### Box Operations
- `b` - Create new box at cursor position
- `e` - Edit text in box under cursor
- `r` - Resize box under cursor (enters resize mode)
- `m` - Move box under cursor (enters move mode)
- `d` - Delete box under cursor (shows confirmation)
- `c` - Copy box under cursor
- `p` - Paste copied box at cursor position

### Text Operations
- `t` - Enter text mode at cursor position (plain text, no borders)
- `d` - Delete text under cursor (shows confirmation)

### Connection Operations
- `a` - Start/finish connection creation
  - Press 'a' on a box or line to start
  - Press 'a' on empty space to add waypoint (custom routing)
  - Press 'a' on a box or line to finish
  - Connections can start/end at boxes or existing lines
- `Escape` - Cancel connection (if started but not finished)

### Resize Mode (after pressing 'r')
- `h/←/j/↓/k/↑/l/→` - Resize box (shrink/expand width/height)
- `Shift+h/j/k/l` - Resize box 2x faster
- `Enter` - Finish resizing and return to normal mode
- `Escape` - Cancel resize and return to normal mode

### Move Mode (after pressing 'm')
- `h/←/j/↓/k/↑/l/→` - Move box around the screen
- `Shift+h/j/k/l` - Move box 2x faster
- `Enter` - Finish moving and return to normal mode
- `Escape` - Cancel move and return to normal mode

### File Operations
- `s` - Save flowchart (prompts for filename, adds .txt if missing)
- `o` - Open flowchart (prompts for filename, adds .txt if missing)
- `x` - Export as PNG image (prompts for filename, adds .png if missing)

### Editing Mode (after pressing 'e')
- `Type` - Add text to box
- `Enter` - Add new line to box text
- `Backspace` - Delete last character
- `Ctrl+S` - Save changes and return to normal mode
- `Escape` - Cancel changes and return to normal mode

### Text Mode (after pressing 't')
- `Type` - Add plain text at cursor position
- `Enter` - Add new line to text
- `Backspace` - Delete last character
- `Ctrl+S` - Save text and return to normal mode
- `Escape` - Cancel and return to normal mode

### General
- `u` - Undo last action
- `U` - Redo last undone action
- `Escape` - Clear selection/cancel current operation
- `?` - Toggle help screen
- `q/Ctrl+C` - Quit application (shows confirmation)

## File Format

Flowcharts are saved in a text format:
```
FLOWCHART
BOXES:2
10,5,12,3,Start
25,10,14,3,Process
CONNECTIONS:1
0,1,10,7,25,10,0
TEXTS:0
```

- **BOXES**: Format is `X,Y,Width,Height,Text` (Width/Height optional for backward compatibility)
- **CONNECTIONS**: Format is `FromID,ToID,FromX,FromY,ToX,ToY,WaypointCount|waypoints` (old format: `FromID,ToID`)
  - Waypoints format: `X:Y,X:Y,...`
  - FromID/ToID can be -1 for line-to-line connections
- **TEXTS**: Format is `X,Y,Text` (optional section)

## Dependencies

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [gg](https://github.com/fogleman/gg) - 2D graphics library for PNG export
