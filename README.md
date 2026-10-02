<img width="932" height="685" alt="flerm start screen" src="https://github.com/user-attachments/assets/0b006b78-886a-4e6f-8e3b-d8e4b09efcbd" />

**A quick and easy flowchart editor for the terminal.**

## Installation

Build the binary and run it:

```bash
make build
./flerm
```

Or install it onto your `PATH` (into `$GOBIN`, or `$GOPATH/bin`) so you can run `flerm` from anywhere:

```bash
make install
flerm
```

## Usage

```bash
flerm                   -- start flerm
flerm --help            -- show a reminder of the next 3 things
flerm --no-resume       -- don't offer the r:esume function in the main flerm menu.
flerm --describe <name> -- print a chart's details in plain text so screen readers can describe it (experimental/might be wonky).
flerm --setup           -- create ~/.flermrc config with all the defaults pre-set.
```

## Configuration

You can create a `.flermrc` configuration file in your home directory to customize Flerm's behavior.

Run `flerm --setup` and it writes one for you, listing every setting at its default value, so
there's nothing to guess at — edit the values you care about. It won't touch an existing
`~/.flermrc`; move or delete that first if you want a clean one.

### Example Configuration File

```bash
# Flerm Config

# Save all files to ~/Documents/flerm
savedirectory=~/Documents/flerm

# Skip the start menu on launch
startmenu=false

# Show confirmation dialogs
confirmations=true

# Offer "r: Resume <chart>" on the start menu for the last chart you opened
# (set it to false, or pass --no-resume, to hide it)
resume=true

# Stop the animated particles on the holiday start screens (the logo stays)
particles=false

# Don't capture the mouse, so the terminal keeps its own click-drag text selection
mouse=false
```

`disableparticles=true` and `disablemouse=true` work too, if you prefer naming the thing
you're switching off.

<br>
<img width="932" height="685" alt="RABDARGAB plan" src="https://github.com/user-attachments/assets/5346afbf-7dbe-4fb6-9b4e-c806c3bba826" />

## Mouse Support

- **Left-click** a box, text, or line to select it.
- **Click and drag a box or text** to move it. Connected lines re-route themselves as you drag — this used to be a total disaster and is now actually pretty good.
- **Click and drag empty space** to pan the canvas around (scroll wheel pans vertically too!).
- **Right-click** anything for a context menu:
  - Box: Edit Box, Edit Title, Border ▸ (Style / Color), New Line, Delete Box
  - Text: Edit Text, Color, Delete Text
  - Line: New Line, Color, Delete Line
  - Empty space: New Box, New Text
  - Submenus pop out to the side and you can hover/click them, or use the arrow keys (→ to open, ← to back out).
- **Drawing lines with the mouse:** pick "New Line" from a box's _or_ a line's menu, then left-click to drop nodes. Click a box or line to finish.
- **Highlight mode:** click and drag to paint/draw in the selected color anywhere on the canvas.
- **Multi-select:** click and drag a rectangle around some boxes. Everything inside gets highlighted and you can drag the whole group around at once.

<br>
<img width="932" height="685" alt="image" src="https://github.com/user-attachments/assets/a11e03e0-d4c1-4045-adcb-de5f13133dd6" />

## Keymaps

### Navigation

- `h/←/j/↓/k/↑/l/→` - Move cursor around the screen
- `Shift+h/←/j/↓/k/↑/l/→` - Move cursor 2x faster

### Boxes

- `b` - Create new box at cursor position
- `T` - Add/edit title on box under cursor
- `e` - Edit text in box under cursor
- `r` - Resize box under cursor
- `m` - Move box under cursor
- `d` - Delete box under cursor
- `c` - Copy box under cursor
- `p` - Paste copied box at cursor position
- `Z` - Cycle box z-level (0-3) for drop shadow effect
- `Tab` - Cycle border style for box under cursor (ASCII, Single, Double, Rounded)
- `B` - Box jump - quickly jump to any box by entering its number
- `M` - Enter multi-select mode, then drag out a rectangle (or use the arrow keys + `Enter`) to select and move multiple boxes at once

### Text

- `t` - Enter text mode at cursor position
- `e` - Edit text object under cursor
- `m` - Move text object under cursor
- `d` - Delete text under cursor

### Connections

- `a` - Start/finish connection creation
  - Press 'a' on a box or line to start
  - Press 'a' on empty space to add a node
  - Press 'a' on a box or line to finish
  - Connections can start/end at boxes or existing lines
- `A` - Toggle arrow state on connection line under cursor
  - Cycles through: no arrows → to arrow → from arrow → both arrows
- `Escape` - Cancel

### Highlight Mode

- `Space` - Enter highlight mode
  - When in highlight mode on a box: cycle highlighting (divider → border → both → clear)
- `Tab` - Open the color picker (16 colors: Gray, Red, Green, Yellow, Blue, Magenta, Cyan, White, Black and the bright variants)
- `h/←/j/↓/k/↑/l/→` - Highlight under the curcor
- `Shift+h/j/k/l` - Move cursor faster
- `d` - remove hightlight from under the cursor
- `D` - remove ALL highlight from the element under the cursor
- `Enter` - Highlight entire element at cursor position
- `Esc` - Exit highlight mode
  <br>
  <img width="932" height="686" alt="an incredible flerm painting of flowers and the word gorgeous in text" src="https://github.com/user-attachments/assets/87eb7972-f60d-4122-a4f1-d81b4b92fce6" />

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

- .png exports are _way_ better than they were in the last build, but still might me a little odd at times
- All file operations respect the `savedirectory` setting in `~/.flermrc` if configured.

### Buffer Operations

- `{` - Switch to previous buffer
- `}` - Switch to next buffer
- `n` - Create new chart in current buffer
- `N` - Create new chart in new buffer
- `x` - Close current buffer

### General

- `C` - Set the color of the box, text, or line under the cursor
- `v` - Toggle the tooltip sidebar
- `u` - Undo last action
- `U` - Redo last undone action
- `z` - Toggle pan mode. You can also just click-drag empty space to pan
- `Esc` - Clear selection/cancel current operation
- `?` - Toggle help screen
- `q` - Quit Flerm

## Accessibility

**`flerm --describe <chart>`** prints a description of the chart as plaintext so screen
readers can (hopefully) have a nicer time reading it out. This is still super experimental
and might not be incredibly useful right now.

```
$ flerm --describe demo
Chart: demo.flerm
3 boxes, 2 connections, 1 text label, 2 highlighted cells

Box 0 "Start" at 10,5 size 12x3, ascii border
  text: Start
  connects to Box 1 "Stage", arrow at the far end

Box 1 "Stage" at 25,10 size 19x5, single border, color Red
  title: Stage
  text: Process nightly
  tooltip: takes about 4 minutes
  connected from Box 0 "Start", arrow at this box
```

In the editor:

- The **status line** names what's under the cursor and the color it's set to — `Box 1 ■ Red`,
  or `Connection Box 0 to Box 1 ■ Blue` — so no state is signalled by hue alone.
- **`C`** lets you set the color of whatever is under the cursor
- **`v`** opens a sidebar listing every tooltip labelled by box
- **`NO_COLOR=1`** (or `TERM=dumb`) turns off all color. A terminal without 256-color support
  gets 16-color codes rather than unsupported ones.
- **`particles=false`** stops the holiday animations, which flash and can't otherwise be
  turned off except by changing the date.
- **`mouse=false`** leaves mouse reporting off, so the terminal's own click-drag text
  selection keeps working — which is how you copy a chart out to read it elsewhere.

## File Format

Flowcharts are saved in a text (.flerm) format. Older .sav files still work but the formatting may be a little off:

```
FLOWCHART
BOXES:2
10,5,12,3,0,0,,Start
25,10,14,3,0,0,,Process
CONNECTIONS:1
0,1,10,7,25,10,0
TEXTS:0
```

- **BOXES**: Format is `X,Y,Width,Height,ZLevel,BorderStyle,Title,Text`
  - ZLevel: 0-3 (for drop shadow effect)
  - BorderStyle: 0=ASCII, 1=Single, 2=Double, 3=Rounded
  - Title: Optional centered title with divider (can be empty)
- **CONNECTIONS**: Format is `FromID,ToID,FromX,FromY,ToX,ToY,WaypointCount|waypoints`
  - Waypoints format: `X:Y,X:Y,...`
  - FromID/ToID can be -1 for line-to-line connections
- **TEXTS**: Format is `X,Y,Text`
- **BOXCOLORS / LINECOLORS / TEXTCOLORS**: Optional trailing sections listing `index,color` for any object that has a color set (color is a 0-15 palette index). Objects without a color are simply left out.

**Note:** The format is backward-compatible in both directions. Older files without ZLevel, BorderStyle, Title, or color sections load fine with defaults, and older versions of Flerm just ignore the color sections. Some files made with an earlier build might need to be tweaked to look nicer.

## Dependencies

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [gg](https://github.com/fogleman/gg) - 2D graphics library for PNG export
