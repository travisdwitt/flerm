# Fl(ow)(T)erm
A super basic flowchart editor for your terminal.

## Installation
```bash
go build -o flerm
```

## Usage
```bash
./flerm
```

## Configuration
You can create a `.flermrc` configuration file in your home directory to customize Flerm's behavior.

### Example Configuration File
```bash
# Flerm Configuration File
# Comments start with #

# Save all files to ~/Documents/flerm
savedirectory=~/Documents/flerm

# Skip the start menu on launch
startmenu=false

# Show confirmation dialogs
confirmations=true
```

## Keymaps

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
- `e` - Edit text object under cursor
- `m` - Move text object under cursor (enters move mode)
- `d` - Delete text under cursor (shows confirmation)

### Connection Operations
- `a` - Start/finish connection creation
  - Press 'a' on a box or line to start
  - Press 'a' on empty space to add waypoint (custom routing)
  - Press 'a' on a box or line to finish
  - Connections can start/end at boxes or existing lines
- `A` - Toggle arrow state on connection line under cursor
  - Cycles through: no arrows → to arrow → from arrow → both arrows
- `Escape` - Cancel connection (if started but not finished)

### Resize Mode (after pressing 'r')
- `h/←/j/↓/k/↑/l/→` - Resize box (shrink/expand width/height)
- `Shift+h/j/k/l` - Resize box 2x faster
- `Enter` - Finish resizing and return to normal mode
- `Escape` - Cancel resize and return to normal mode

### Move Mode (after pressing 'm' on a box or text)
- `h/←/j/↓/k/↑/l/→` - Move object around the screen
- `Shift+h/j/k/l` - Move object 2x faster
- `Enter` - Finish moving and return to normal mode
- `Escape` - Cancel move and return to normal mode

### File Operations
- `s` - Save flowchart (prompts for filename, adds .sav if missing)
- `S` - Export chart (prompts to choose PNG or Visual TXT format)
- `o` - Open flowchart in current buffer (replaces current chart, shows file list)
- `O` - Open flowchart in new buffer (creates new buffer, shows file list)
  - Press `p` to export as PNG image
  - Press `t` to export as Visual TXT file

**Note:** 
- When opening files (o/O), a list of available .sav files is shown. Use ↑/↓ or k/j to navigate, or type a filename manually.
- All file operations (save, open, export) respect the `savedirectory` setting in `~/.flermrc` if configured.

### Buffer Operations
- `{` - Switch to previous buffer
- `}` - Switch to next buffer
- `n` - Create new chart in current buffer (replaces current chart, shows confirmation)
- `N` - Create new chart in new buffer (creates new buffer, no confirmation)
- `x` - Close current buffer (shows confirmation, warns about unsaved changes)

### General
- `u` - Undo last action
- `U` - Redo last undone action
- `Esc` - Clear selection/cancel current operation
- `?` - Toggle help screen
- `q/Ctrl+C` - Quit application (shows confirmation)

## File Format
Flowcharts are saved in a text (.sav) format:
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
