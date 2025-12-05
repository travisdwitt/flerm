```
+--------------------------------+
|   ___ __                       |
| .'  _|  |.-----.----.--------. |
| |   _|  ||  -__|   _|        | |
| |__| |__||_____|__| |__|__|__| |
|                                |
+--------------------------------+
```
**A quick and easy flowchart editor for the terminal.**
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
- `d` - remove hightlight from under the cursor
- `D` - remove ALL highlight from the element under the cursor
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

## Known Bugs
There's some weird stuff that happens and I haven't been able to fix it yet.
I'm currently using this application in a professional setting, but YMMV so
be careful and don't expect this to be a bug-free battle-hardened productivity
tool.

- Moving boxes around when it's connected to other boxes.
  - This _technically_ works, but the connectors and pointers get super wonky. The
    better option right now is to just delete the connections, move the box, then
    reconnect everything.

- Highlighting and coloring weirdness
  - Sometimes when you're highlighting text and you move to a space without text,
    there will be an extra line of color that just kind of appears?
  - Making a big box, highlighting it, then making the box smaller will leave a box
    of color behind. 
  - Sometimes when drawing with the highlighter, it will look like the highlighter
    isn't working. It is, when you hit Ctrl+S any missing color should show up.

- PNG Exports
  - Ugly, probably not super useful.

- TXT Exports
  - Mostly good, sometimes connections can look funky. Also colors don't translate.
