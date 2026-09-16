package canvas

import (
	"fmt"
	"strings"
)

type RenderResult struct {
	Canvas   [][]rune
	ColorMap [][]int
	Width    int
	Height   int
}

func (r *RenderResult) ApplyColors() []string {
	result := make([]string, r.Height)
	for i, row := range r.Canvas {
		line := make([]rune, r.Width)
		copy(line, row)
		for j := len(row); j < r.Width; j++ {
			line[j] = ' '
		}
		var coloredLine strings.Builder
		currentColor := -1
		for j, char := range line {
			cellColor := -1
			if i < len(r.ColorMap) && j < len(r.ColorMap[i]) {
				cellColor = r.ColorMap[i][j]
			}
			if cellColor != currentColor {
				if currentColor != -1 {
					coloredLine.WriteString(colorReset)
				}
				if cellColor != -1 {
					coloredLine.WriteString(colorCode(cellColor, char == ' '))
				}
				currentColor = cellColor
			}
			coloredLine.WriteRune(char)
		}
		if currentColor != -1 {
			coloredLine.WriteString(colorReset)
		}
		result[i] = coloredLine.String()
	}
	return result
}

func (c *Canvas) RenderRaw(width, height int, selectedBox int, previewFromX, previewFromY int, previewWaypoints []Point, previewToX, previewToY int, panX, panY int, cursorX, cursorY int, showCursor bool, editBoxID int, editTextID int, editCursorPos int, editText string, editTextX int, editTextY int, selectionStartX, selectionStartY, selectionEndX, selectionEndY int, showBoxNumbers bool, editSelStart, editSelEnd int) *RenderResult {
	width, height = max(width, 1), max(height, 1)
	canvas := make([][]rune, height)
	colorMap := make([][]int, height)
	for i := range canvas {
		canvas[i] = make([]rune, width)
		colorMap[i] = make([]int, width)
		for j := range canvas[i] {
			canvas[i][j] = ' '
			colorMap[i][j] = -1
		}
	}
	putRune := func(x, y int, ch rune) {
		if c.isValidPos(canvas, x, y) {
			canvas[y][x] = ch
		}
	}
	putColor := func(x, y, color int) {
		if c.isValidPos(canvas, x, y) {
			colorMap[y][x] = color
		}
	}

	for _, connection := range c.connections {
		c.drawConnectionWithPan(canvas, connection, panX, panY)
	}
	if previewFromX >= 0 && previewFromY >= 0 {
		c.drawConnectionWithPan(canvas, Connection{
			FromID: -1, ToID: -1,
			FromX: previewFromX, FromY: previewFromY,
			ToX: previewToX, ToY: previewToY,
			Waypoints: append([]Point(nil), previewWaypoints...),
			Color:     -1,
		}, panX, panY)
	}
	for _, text := range c.texts {
		c.drawTextAt(canvas, text.Lines, text.X-panX, text.Y-panY)
	}
	if editTextX >= 0 && editTextY >= 0 && editText != "" {
		c.drawTextAt(canvas, strings.Split(editText, "\n"), editTextX-panX, editTextY-panY)
	}

	// Higher z-levels draw last so their shadows fall over lower boxes.
	boxOrder := make([]int, 0, len(c.boxes))
	for z := 0; z < numZLevels; z++ {
		for i, box := range c.boxes {
			if box.ZLevel == z {
				boxOrder = append(boxOrder, i)
			}
		}
	}
	for _, i := range boxOrder {
		box := c.boxes[i]
		if box.ZLevel > 0 {
			c.drawBoxShadow(canvas, box, box.ZLevel, panX, panY)
		}
		c.drawBoxAt(canvas, box, i == selectedBox, box.X-panX, box.Y-panY)
		if showBoxNumbers {
			boxScreenX, boxScreenY := box.X-panX, box.Y-panY
			if !c.isValidPos(canvas, boxScreenX, boxScreenY) {
				continue
			}
			for idx, char := range fmt.Sprintf("%d", i) {
				if posX := boxScreenX + 1 + idx; posX < boxScreenX+box.Width-1 {
					putRune(posX, boxScreenY, char)
				}
			}
		}
	}

	// editCursorAt maps a character offset in the text being edited to a screen cell.
	editCursorAt := func(pos int) (int, int, bool) {
		originX, originY, ok := c.EditOrigin(editBoxID, editTextID, editTextX, editTextY)
		if !ok {
			return 0, 0, false
		}
		x, y := cursorScreenPos(originX, originY, pos, editText, panX, panY)
		return x, y, true
	}

	if editSelStart >= 0 && editSelEnd >= 0 && editSelStart != editSelEnd {
		selStart, selEnd := min(editSelStart, editSelEnd), max(editSelStart, editSelEnd)
		for pos := selStart; pos < selEnd; pos++ {
			if x, y, ok := editCursorAt(pos); ok {
				putColor(x, y, colorEditSelect)
			}
		}
	}
	if x, y, ok := editCursorAt(editCursorPos); ok {
		putRune(x, y, '█')
	}
	if showCursor {
		putRune(cursorX, cursorY, '█')
	}

	if selectionStartX >= 0 && selectionStartY >= 0 {
		x0, x1 := minmax(selectionStartX-panX, selectionEndX-panX)
		y0, y1 := minmax(selectionStartY-panY, selectionEndY-panY)
		x0, y0 = max(x0, 0), max(y0, 0)
		x1, y1 = min(x1, width-1), min(y1, height-1)
		for x := x0; x <= x1; x++ {
			corner := x == x0 || x == x1
			if y0 == y1 && corner {
				putRune(x, y0, '█')
				continue
			}
			putRune(x, y0, boxRune(corner, x == x0, true))
			if y1 != y0 {
				putRune(x, y1, boxRune(corner, x == x0, false))
			}
		}
		for y := y0 + 1; y < y1; y++ {
			putRune(x0, y, '│')
			putRune(x1, y, '│')
		}
	}

	paintCells := func(cells []Point, colorIndex int) {
		if colorIndex < 0 {
			return
		}
		for _, cell := range cells {
			putColor(cell.X-panX, cell.Y-panY, colorIndex)
		}
	}
	for i := range c.connections {
		paintCells(c.GetConnectionCells(i), c.connections[i].Color)
	}
	for i := range c.texts {
		paintCells(c.GetTextCells(i), c.texts[i].Color)
	}
	for _, i := range boxOrder {
		paintCells(c.GetBoxBorderCells(i), c.boxes[i].Color)
		paintCells(c.GetBoxTitleBarCells(i), c.boxes[i].Color)
	}
	for cell, colorIndex := range c.highlights {
		putColor(cell.X-panX, cell.Y-panY, colorIndex)
	}

	return &RenderResult{Canvas: canvas, ColorMap: colorMap, Width: width, Height: height}
}

func minmax(a, b int) (int, int) { return min(a, b), max(a, b) }

// boxRune picks the light box-drawing rune for a selection rectangle edge.
func boxRune(corner, left, top bool) rune {
	if !corner {
		return '─'
	}
	switch {
	case top && left:
		return '┌'
	case top:
		return '┐'
	case left:
		return '└'
	}
	return '┘'
}

func (c *Canvas) drawBoxShadow(canvas [][]rune, box Box, shadowOffset, panX, panY int) {
	shadowChar := []rune{'░', '░', '▒', '▓'}[min(shadowOffset, 3)]
	boxX, boxY := box.X-panX, box.Y-panY
	for y := boxY + shadowOffset; y < boxY+shadowOffset+box.Height; y++ {
		for x := boxX + shadowOffset; x < boxX+shadowOffset+box.Width; x++ {
			if x >= boxX && x < boxX+box.Width && y >= boxY && y < boxY+box.Height {
				continue
			}
			if c.isValidPos(canvas, x, y) {
				canvas[y][x] = shadowChar
			}
		}
	}
}

// borderRunes returns topLeft, topRight, bottomLeft, bottomRight, horizontal, vertical.
func borderRunes(style BorderStyle, selected bool) [6]rune {
	if selected {
		return [6]rune{'#', '#', '#', '#', '#', '#'}
	}
	switch style {
	case BorderStyleSingle:
		return [6]rune{'┌', '┐', '└', '┘', '─', '│'}
	case BorderStyleDouble:
		return [6]rune{'╔', '╗', '╚', '╝', '═', '║'}
	case BorderStyleRounded:
		return [6]rune{'╭', '╮', '╰', '╯', '─', '│'}
	}
	return [6]rune{'+', '+', '+', '+', '-', '|'}
}

func (c *Canvas) drawBoxAt(canvas [][]rune, box Box, isSelected bool, boxX, boxY int) {
	b := borderRunes(box.BorderStyle, isSelected)
	put := func(x, y int, ch rune) {
		if c.isValidPos(canvas, x, y) {
			canvas[y][x] = ch
		}
	}

	right, bottom := boxX+box.Width-1, boxY+box.Height-1
	for x := boxX; x <= right; x++ {
		put(x, boxY, b[4])
		put(x, bottom, b[4])
	}
	for y := boxY + 1; y < bottom; y++ {
		put(boxX, y, b[5])
		put(right, y, b[5])
	}
	put(boxX, boxY, b[0])
	put(right, boxY, b[1])
	put(boxX, bottom, b[2])
	put(right, bottom, b[3])

	// Text is clipped to the inside of the border on every side.
	maxWidth := max(box.Width-2, 0)
	putLines := func(lines []string, startY, limitY int) {
		for i, line := range lines {
			y := startY + i
			if y >= limitY {
				break
			}
			for j, ch := range []rune(line) {
				if j >= maxWidth {
					break
				}
				put(boxX+1+j, y, ch)
			}
		}
	}

	if box.Title != "" {
		titleLines := strings.Split(box.Title, "\n")
		putLines(titleLines, boxY+1, bottom+len(titleLines))
		dividerY := boxY + 1 + len(titleLines)
		for x := boxX + 1; x < right; x++ {
			put(x, dividerY, b[4])
		}
	}
	putLines(box.Lines, boxY+contentStartLine(box), bottom)
}

func (c *Canvas) drawTextAt(canvas [][]rune, lines []string, textX, textY int) {
	for lineIdx, line := range lines {
		for i, char := range line {
			if c.isValidPos(canvas, textX+i, textY+lineIdx) {
				canvas[textY+lineIdx][textX+i] = char
			}
		}
	}
}

// editTextID -2 means the edit is a box title; a box id otherwise wins over a text id.
func (c *Canvas) EditOrigin(editBoxID, editTextID, editTextX, editTextY int) (int, int, bool) {
	switch {
	case editTextID == -2 && editBoxID >= 0 && editBoxID < len(c.boxes):
		return c.boxes[editBoxID].X + 1, c.boxes[editBoxID].Y + 1, true
	case editBoxID >= 0 && editBoxID < len(c.boxes):
		box := c.boxes[editBoxID]
		return box.X + 1, box.Y + contentStartLine(box), true
	case editTextID >= 0 && editTextID < len(c.texts):
		return c.texts[editTextID].X, c.texts[editTextID].Y, true
	case editTextX >= 0 && editTextY >= 0:
		return editTextX, editTextY, true
	}
	return 0, 0, false
}

// cursorScreenPos walks content to find the screen cell for a character offset.
func cursorScreenPos(originX, originY, cursorPos int, content string, panX, panY int) (int, int) {
	lines := strings.Split(content, "\n")
	currentPos := 0
	for lineIdx, line := range lines {
		lineLength := len([]rune(line))
		if cursorPos <= currentPos+lineLength {
			return originX + (cursorPos - currentPos) - panX, originY + lineIdx - panY
		}
		currentPos += lineLength + 1
	}
	last := lines[len(lines)-1]
	return originX + len([]rune(last)) - panX, originY + len(lines) - 1 - panY
}

func (c *Canvas) drawConnectionWithPan(canvas [][]rune, connection Connection, panX, panY int) {
	shifted := connection
	shifted.FromX, shifted.FromY = connection.FromX-panX, connection.FromY-panY
	shifted.ToX, shifted.ToY = connection.ToX-panX, connection.ToY-panY
	shifted.Waypoints = make([]Point, len(connection.Waypoints))
	for i, wp := range connection.Waypoints {
		shifted.Waypoints[i] = Point{wp.X - panX, wp.Y - panY}
	}
	pts := connPoints(shifted)

	// Diagonal hops become an L: across first, then down.
	var verts []Point
	addV := func(p Point) {
		if len(verts) == 0 || verts[len(verts)-1] != p {
			verts = append(verts, p)
		}
	}
	for i := 0; i < len(pts)-1; i++ {
		from, to := pts[i], pts[i+1]
		addV(from)
		if from.X != to.X && from.Y != to.Y {
			addV(Point{to.X, from.Y})
		}
		addV(to)
	}
	if len(verts) < 2 {
		return
	}

	put := func(x, y int, ch rune) {
		if c.isValidPos(canvas, x, y) && !c.isPointInBoxScreen(x, y, connection.FromID, connection.ToID, panX, panY) {
			canvas[y][x] = ch
		}
	}
	for i := 0; i < len(verts)-1; i++ {
		a, b := verts[i], verts[i+1]
		if a.X == b.X {
			y0, y1 := minmax(a.Y, b.Y)
			for y := y0; y <= y1; y++ {
				put(a.X, y, '│')
			}
		} else {
			x0, x1 := minmax(a.X, b.X)
			for x := x0; x <= x1; x++ {
				put(x, a.Y, '─')
			}
		}
	}
	for i := 1; i < len(verts)-1; i++ {
		if ch := cornerChar(verts[i-1], verts[i], verts[i+1]); ch != 0 {
			put(verts[i].X, verts[i].Y, ch)
		}
	}

	if connection.ArrowFrom {
		c.drawConnEndArrow(canvas, connection.FromID, connection.FromX, connection.FromY, panX, panY)
	}
	if connection.ArrowTo {
		c.drawConnEndArrow(canvas, connection.ToID, connection.ToX, connection.ToY, panX, panY)
	}
}

func cornerChar(prev, cur, next Point) rune {
	switch {
	case prev.Y == cur.Y && next.X == cur.X && prev.X != cur.X && next.Y != cur.Y:
		return boxRune(true, prev.X > cur.X, next.Y > cur.Y)
	case prev.X == cur.X && next.Y == cur.Y && prev.Y != cur.Y && next.X != cur.X:
		return boxRune(true, next.X > cur.X, prev.Y > cur.Y)
	}
	return 0
}

func (c *Canvas) drawConnEndArrow(canvas [][]rune, boxID, ax, ay, panX, panY int) {
	box, ok := c.box(boxID)
	if !ok {
		return
	}
	dl := abs(ax - box.X)
	dr := abs(ax - (box.X + box.Width - 1))
	dt := abs(ay - box.Y)
	db := abs(ay - (box.Y + box.Height - 1))
	var x, y int
	var ch rune
	switch {
	case dl <= dr && dl <= dt && dl <= db:
		x, y, ch = box.X-1-panX, ay-panY, '▶'
	case dr <= dt && dr <= db:
		x, y, ch = box.X+box.Width-panX, ay-panY, '◀'
	case dt <= db:
		x, y, ch = ax-panX, box.Y-1-panY, '▼'
	default:
		x, y, ch = ax-panX, box.Y+box.Height-panY, '▲'
	}
	if c.isValidPos(canvas, x, y) && !c.isPointInBoxScreen(x, y, boxID, boxID, panX, panY) {
		canvas[y][x] = ch
	}
}

func (c *Canvas) isValidPos(canvas [][]rune, x, y int) bool {
	return y >= 0 && y < len(canvas) && x >= 0 && x < len(canvas[y])
}

// colorCode returns the SGR sequence for a palette index: background for blank
// cells, foreground for cells holding a glyph.
func colorCode(colorIndex int, blank bool) string {
	switch colorIndex {
	case colorEditSelect:
		return "\x1b[7;36m"
	case ColorMouseSelect:
		if blank {
			return "\x1b[103m"
		}
		return "\x1b[1;93m"
	case ColorMenuSelect:
		return "\x1b[7m"
	case ColorMenuBorder:
		return "\x1b[32m"
	}
	if colorIndex < 0 || colorIndex >= NumColors {
		return ""
	}
	base := 30
	if blank {
		base = 40
	}
	// Index 0 (gray) and 7 (white) share the palette's white slot.
	offsets := []int{7, 1, 2, 3, 4, 5, 6, 7}
	return fmt.Sprintf("\x1b[%dm", base+offsets[colorIndex])
}

const colorReset = "\x1b[0m"
