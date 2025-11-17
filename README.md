# Fl(ow)(T)erm
A super basic flowchart editor for your terminal.
<br>
<img width="932" height="685" alt="image" src="https://github.com/user-attachments/assets/62cbfad3-f27e-471e-b219-2c3ed56d381d" />
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
# Flerm Config

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
- `Shift+h/←/j/↓/k/↑/l/→` - Move cursor 2x faster

### Box Operations
- `b` - Create new box at cursor position
- `e` - Edit text in box under cursor
- `r` - Resize box under cursor
- `m` - Move box under cursor
- `d` - Delete box under cursor
- `c` - Copy box under cursor
- `p` - Paste copied box at cursor position

### Text Operations
- `t` - Enter text mode at cursor position
- `e` - Edit text object under cursor
- `m` - Move text object under cursor
- `d` - Delete text under cursor

### Connection Operations
- `a` - Start/finish connection creation
  - Press 'a' on a box or line to start
  - Press 'a' on empty space to add waypoint
  - Press 'a' on a box or line to finish
  - Connections can start/end at boxes or existing lines
  - If a corner character gets messed up, redrawing the line helps.
- `A` - Toggle arrow state on connection line under cursor
  - Cycles through: no arrows → to arrow → from arrow → both arrows
  - Sometimes the arrows get wonky. When in doubt, just redraw the line.
- `Escape` - Cancel connection (if started but not finished)

### Highlight Mode
- `Space` - Toggle highlight mode on/off
- `Tab` - Cycle through 8 highlight colors (Gray, Red, Green, Yellow, Blue, Magenta, Cyan, White)
- `h/←/j/↓/k/↑/l/→` - Move cursor and leave colored trail 
- `Shift+h/j/k/l` - Move cursor faster 
- `Enter` - Highlight entire element at cursor position
- `Esc` - Exit highlight mode
<br>
<img width="932" height="686" alt="image" src="https://github.com/user-attachments/assets/0e58f946-3017-4b73-80d5-531adabb4e19" />


### Resize Mode
- `h/←/j/↓/k/↑/l/→` - Resize box
- `Shift+h/j/k/l` - Resize box 2x faster
- `Enter` - Finish resizing and return to normal mode
- `Esc` - Cancel resize and return to normal mode

### Move Mode 
- `h/←/j/↓/k/↑/l/→` - Move object around the screen
- `Shift+h/j/k/l` - Move object 2x faster
- `Enter` - Finish moving and return to normal mode
- `Esc` - Cancel move and return to normal mode

### File Operations
- `s` - Save flowchart 
- `S` - Export chart (prompts to choose PNG or Visual TXT format)
- `o` - Open flowchart in current buffer 
- `O` - Open flowchart in new buffer 
  - Press `p` to export as PNG image
  - Press `t` to export as Visual TXT file

**Note:** 
- .png exports are pretty wonky and terrible, but a fun curiosity. Stick to txt exports for the best results right now.
- All file operations respect the `savedirectory` setting in `~/.flermrc` if configured.

### Buffer Operations
- `{` - Switch to previous buffer
- `}` - Switch to next buffer
- `n` - Create new chart in current buffer 
- `N` - Create new chart in new buffer 
- `x` - Close current buffer
<br>
<img width="932" height="685" alt="flermguy" src="https://github.com/user-attachments/assets/d609f834-5048-46e3-a3ad-e0c5d10003d1" />

### General
- `u` - Undo last action
- `U` - Redo last undone action
- `Esc` - Clear selection/cancel current operation
- `?` - Toggle help screen
- `q` - Quit Flerm

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

- **BOXES**: Format is `X,Y,Width,Height,Text` 
- **CONNECTIONS**: Format is `FromID,ToID,FromX,FromY,ToX,ToY,WaypointCount|waypoints`
  - Waypoints format: `X:Y,X:Y,...`
  - FromID/ToID can be -1 for line-to-line connections
- **TEXTS**: Format is `X,Y,Text`

## Dependencies
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [gg](https://github.com/fogleman/gg) - 2D graphics library for PNG export
