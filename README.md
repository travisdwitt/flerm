# Flerm - Terminal Flowchart Editor

A simple, terminal-based flowchart editor.

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

### Box Operations
- `b` - Create new box at cursor position
- `Enter/Space` - Select box under cursor
- `e` - Edit text in box under cursor
- `d` - Delete box under cursor

### Connection Operations
- `a` - Start/finish connection creation between boxes
  - Press 'a' on source box, then 'a' on target box

### File Operations
- `s` - Save flowchart to 'flowchart.txt'
- `o` - Open flowchart from 'flowchart.txt'
- `x` - Export as PNG image to 'flowchart.png'

### Editing Mode
- `Type` - Add text to box
- `Backspace` - Delete last character
- `Enter` - Save changes and return to normal mode
- `Escape` - Cancel changes and return to normal mode

### General
- `Escape` - Clear selection/cancel current operation
- `?` - Toggle help screen
- `q/Ctrl+C` - Quit application

## File Format

Flowcharts are saved in a simple text format:
```
FLOWCHART
BOXES:2
10,5,10,3,Start
25,10,12,3,Process
CONNECTIONS:1
0,1
```

## Dependencies

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [gg](https://github.com/fogleman/gg) - 2D graphics library for PNG export
