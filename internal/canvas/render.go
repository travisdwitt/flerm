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
					if char != ' ' {
						coloredLine.WriteString(getTextColorCode(cellColor))
					} else {
						coloredLine.WriteString(getColorCode(cellColor))
					}
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
	if height < 1 {
		height = 1
	}
	if width < 1 {
		width = 1
	}
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

	for _, connection := range c.connections {
		c.drawConnectionWithPan(canvas, connection, panX, panY)
	}
	if previewFromX >= 0 && previewFromY >= 0 {
		previewConnection := Connection{
			FromID:    -1,
			ToID:      -1,
			FromX:     previewFromX,
			FromY:     previewFromY,
			ToX:       previewToX,
			ToY:       previewToY,
			Waypoints: make([]Point, len(previewWaypoints)),
			Color:     -1,
		}
		for i, wp := range previewWaypoints {
			previewConnection.Waypoints[i] = Point{X: wp.X, Y: wp.Y}
		}
		c.drawConnectionWithPan(canvas, previewConnection, panX, panY)
	}
	for _, text := range c.texts {
		c.drawTextWithPan(canvas, text, panX, panY)
	}
	if editTextX >= 0 && editTextY >= 0 && editText != "" {
		previewText := Text{
			X:     editTextX,
			Y:     editTextY,
			Lines: strings.Split(editText, "\n"),
			ID:    -1,
			Color: -1,
		}
		c.drawTextWithPan(canvas, previewText, panX, panY)
	}
	boxOrder := make([]int, len(c.boxes))
	for i := range boxOrder {
		boxOrder[i] = i
	}
	for i := 0; i < len(boxOrder)-1; i++ {
		for j := i + 1; j < len(boxOrder); j++ {
			if c.boxes[boxOrder[i]].ZLevel > c.boxes[boxOrder[j]].ZLevel {
				boxOrder[i], boxOrder[j] = boxOrder[j], boxOrder[i]
			}
		}
	}
	for _, i := range boxOrder {
		box := c.boxes[i]
		isSelected := (i == selectedBox)
		if box.ZLevel > 0 {
			c.drawBoxShadow(canvas, box, box.ZLevel, panX, panY)
		}
		c.drawBoxWithPan(canvas, box, isSelected, panX, panY)
		if showBoxNumbers {

			boxScreenX := box.X - panX
			boxScreenY := box.Y - panY
			if boxScreenY >= 0 && boxScreenY < height && boxScreenX >= 0 && boxScreenX < width {
				numberStr := fmt.Sprintf("%d", i)
				for idx, char := range numberStr {
					posX := boxScreenX + 1 + idx
					if posX < boxScreenX+box.Width-1 && posX >= 0 && posX < width && boxScreenY >= 0 && boxScreenY < height {
						if boxScreenY < len(canvas) && posX < len(canvas[boxScreenY]) {
							canvas[boxScreenY][posX] = char
						}
					}
				}
			}
		}
	}

	if editSelStart >= 0 && editSelEnd >= 0 && editSelStart != editSelEnd {
		selStart, selEnd := editSelStart, editSelEnd
		if selStart > selEnd {
			selStart, selEnd = selEnd, selStart
		}

		for pos := selStart; pos < selEnd; pos++ {
			var screenX, screenY int
			if editTextID == -2 && editBoxID >= 0 && editBoxID < len(c.boxes) {

				box := c.boxes[editBoxID]
				screenX, screenY = c.calculateTitleCursorPosition(box, pos, editText, panX, panY)
			} else if editBoxID >= 0 && editBoxID < len(c.boxes) {
				box := c.boxes[editBoxID]
				screenX, screenY = c.calculateTextCursorPosition(box, pos, editText, panX, panY)
			} else if editTextID >= 0 && editTextID < len(c.texts) {
				text := c.texts[editTextID]
				screenX, screenY = c.calculateTextCursorPositionForText(text, pos, editText, panX, panY)
			} else if editTextX >= 0 && editTextY >= 0 {
				screenX, screenY = c.calculateTextCursorPositionForNewText(editTextX, editTextY, pos, editText, panX, panY)
			} else {
				continue
			}
			if screenY >= 0 && screenY < height && screenX >= 0 && screenX < width {
				if screenY < len(colorMap) && screenX < len(colorMap[screenY]) {
					colorMap[screenY][screenX] = colorEditSelect
				}
			}
		}
	}

	if editTextID == -2 && editBoxID >= 0 && editBoxID < len(c.boxes) {

		box := c.boxes[editBoxID]
		editCursorX, editCursorY := c.calculateTitleCursorPosition(box, editCursorPos, editText, panX, panY)
		if editCursorY >= 0 && editCursorY < height && editCursorX >= 0 && editCursorX < width {
			if editCursorY < len(canvas) && editCursorX < len(canvas[editCursorY]) {
				canvas[editCursorY][editCursorX] = '█'
			}
		}
	} else if editBoxID >= 0 && editBoxID < len(c.boxes) {
		box := c.boxes[editBoxID]
		editCursorX, editCursorY := c.calculateTextCursorPosition(box, editCursorPos, editText, panX, panY)
		if editCursorY >= 0 && editCursorY < height && editCursorX >= 0 && editCursorX < width {
			if editCursorY < len(canvas) && editCursorX < len(canvas[editCursorY]) {
				canvas[editCursorY][editCursorX] = '█'
			}
		}
	} else if editTextID >= 0 && editTextID < len(c.texts) {
		text := c.texts[editTextID]
		editCursorX, editCursorY := c.calculateTextCursorPositionForText(text, editCursorPos, editText, panX, panY)
		if editCursorY >= 0 && editCursorY < height && editCursorX >= 0 && editCursorX < width {
			if editCursorY < len(canvas) && editCursorX < len(canvas[editCursorY]) {
				canvas[editCursorY][editCursorX] = '█'
			}
		}
	} else if editTextX >= 0 && editTextY >= 0 {

		editCursorX, editCursorY := c.calculateTextCursorPositionForNewText(editTextX, editTextY, editCursorPos, editText, panX, panY)
		if editCursorY >= 0 && editCursorY < height && editCursorX >= 0 && editCursorX < width {
			if editCursorY < len(canvas) && editCursorX < len(canvas[editCursorY]) {
				canvas[editCursorY][editCursorX] = '█'
			}
		}
	}

	if showCursor && cursorY >= 0 && cursorY < height && cursorX >= 0 && cursorX < width {
		if cursorY < len(canvas) && cursorX < len(canvas[cursorY]) {
			canvas[cursorY][cursorX] = '█'
		}
	}
	if selectionStartX >= 0 && selectionStartY >= 0 {
		startScreenX := selectionStartX - panX
		startScreenY := selectionStartY - panY
		endScreenX := selectionEndX - panX
		endScreenY := selectionEndY - panY
		minX, maxX := startScreenX, startScreenX
		if endScreenX < startScreenX {
			minX = endScreenX
		} else if endScreenX > startScreenX {
			maxX = endScreenX
		}
		minY, maxY := startScreenY, startScreenY
		if endScreenY < startScreenY {
			minY = endScreenY
		} else if endScreenY > startScreenY {
			maxY = endScreenY
		}
		if minX < 0 {
			minX = 0
		}
		if maxX >= width {
			maxX = width - 1
		}
		if minY < 0 {
			minY = 0
		}
		if maxY >= height {
			maxY = height - 1
		}
		for x := minX; x <= maxX && x < width; x++ {
			if minY >= 0 && minY < height && x >= 0 {
				if x == minX || x == maxX {

					if minY == maxY {

						if len(canvas[minY]) > x {
							canvas[minY][x] = '█'
						}
					} else {
						if x == minX {
							if len(canvas[minY]) > x {
								canvas[minY][x] = '┌'
							}
						} else {
							if len(canvas[minY]) > x {
								canvas[minY][x] = '┐'
							}
						}
					}
				} else {

					if len(canvas[minY]) > x {
						canvas[minY][x] = '─'
					}
				}
				if maxY != minY && maxY >= 0 && maxY < height {
					if x == minX || x == maxX {

						if x == minX {
							if len(canvas[maxY]) > x {
								canvas[maxY][x] = '└'
							}
						} else {
							if len(canvas[maxY]) > x {
								canvas[maxY][x] = '┘'
							}
						}
					} else {

						if len(canvas[maxY]) > x {
							canvas[maxY][x] = '─'
						}
					}
				}
			}
		}

		for y := minY + 1; y < maxY && y < height; y++ {
			if y >= 0 {
				if minX >= 0 && minX < width {
					if len(canvas[y]) > minX {
						canvas[y][minX] = '│'
					}
				}
				if maxX >= 0 && maxX < width {
					if len(canvas[y]) > maxX {
						canvas[y][maxX] = '│'
					}
				}
			}
		}
	}

	paintCells := func(cells []Point, colorIndex int) {
		if colorIndex < 0 {
			return
		}
		for _, cell := range cells {
			sx, sy := cell.X-panX, cell.Y-panY
			if sy >= 0 && sy < height && sx >= 0 && sx < width &&
				sy < len(colorMap) && sx < len(colorMap[sy]) {
				colorMap[sy][sx] = colorIndex
			}
		}
	}
	for i := range c.connections {
		paintCells(c.GetConnectionCells(i), c.connections[i].Color)
	}
	for i := range c.texts {
		paintCells(c.GetTextCells(i), c.texts[i].Color)
	}
	for _, i := range boxOrder {
		box := c.boxes[i]
		if box.Color < 0 {
			continue
		}
		paintCells(c.GetBoxBorderCells(i), box.Color)
		paintCells(c.GetBoxTitleBarCells(i), box.Color)
	}

	for key, colorIndex := range c.highlights {
		var x, y int
		fmt.Sscanf(key, "%d,%d", &x, &y)

		screenX := x - panX
		screenY := y - panY
		if screenY >= 0 && screenY < height && screenX >= 0 && screenX < width {
			if screenY < len(colorMap) && screenX < len(colorMap[screenY]) {
				colorMap[screenY][screenX] = colorIndex
			}
		}
	}

	return &RenderResult{
		Canvas:   canvas,
		ColorMap: colorMap,
		Width:    width,
		Height:   height,
	}
}

func (c *Canvas) drawBoxShadow(canvas [][]rune, box Box, shadowOffset int, panX, panY int) {
	boxX := box.X - panX + shadowOffset
	boxY := box.Y - panY + shadowOffset
	height := len(canvas)
	width := 0
	if height > 0 {
		width = len(canvas[0])
	}
	shadowChar := '░'
	if shadowOffset >= 2 {
		shadowChar = '▒'
	}
	if shadowOffset >= 3 {
		shadowChar = '▓'
	}
	actualBoxX := box.X - panX
	actualBoxY := box.Y - panY
	for y := boxY; y < boxY+box.Height && y < height; y++ {
		if y < 0 {
			continue
		}
		for x := boxX; x < boxX+box.Width && x < width; x++ {
			if x < 0 {
				continue
			}
			if x >= actualBoxX && x < actualBoxX+box.Width && y >= actualBoxY && y < actualBoxY+box.Height {
				continue
			}
			if y < len(canvas) && x < len(canvas[y]) {
				canvas[y][x] = shadowChar
			}
		}
	}
}

func (c *Canvas) drawBoxWithPan(canvas [][]rune, box Box, isSelected bool, panX, panY int) {
	c.drawBoxAt(canvas, box, isSelected, box.X-panX, box.Y-panY)
}

func (c *Canvas) drawBoxAt(canvas [][]rune, box Box, isSelected bool, boxX, boxY int) {
	var topLeft, topRight, bottomLeft, bottomRight, horizontal, vertical rune

	if isSelected {
		topLeft, topRight, bottomLeft, bottomRight, horizontal, vertical = '#', '#', '#', '#', '#', '#'
	} else {
		switch box.BorderStyle {
		case BorderStyleASCII:
			topLeft, topRight, bottomLeft, bottomRight, horizontal, vertical = '+', '+', '+', '+', '-', '|'
		case BorderStyleSingle:
			topLeft, topRight, bottomLeft, bottomRight, horizontal, vertical = '┌', '┐', '└', '┘', '─', '│'
		case BorderStyleDouble:
			topLeft, topRight, bottomLeft, bottomRight, horizontal, vertical = '╔', '╗', '╚', '╝', '═', '║'
		case BorderStyleRounded:
			topLeft, topRight, bottomLeft, bottomRight, horizontal, vertical = '╭', '╮', '╰', '╯', '─', '│'
		default:
			topLeft, topRight, bottomLeft, bottomRight, horizontal, vertical = '+', '+', '+', '+', '-', '|'
		}
	}

	height := len(canvas)
	width := 0
	if height > 0 {
		width = len(canvas[0])
	}

	startY := boxY
	if startY < 0 {
		startY = 0
	}
	endY := boxY + box.Height
	if endY > height {
		endY = height
	}

	startX := boxX
	if startX < 0 {
		startX = 0
	}
	endX := boxX + box.Width
	if endX > width {
		endX = width
	}

	for y := startY; y < endY; y++ {
		for x := startX; x < endX; x++ {

			isTopRow := (y == boxY)
			isBottomRow := (y == boxY+box.Height-1)
			isLeftCol := (x == boxX)
			isRightCol := (x == boxX+box.Width-1)

			if isTopRow {
				if isLeftCol {
					canvas[y][x] = topLeft
				} else if isRightCol {
					canvas[y][x] = topRight
				} else {
					canvas[y][x] = horizontal
				}
			} else if isBottomRow {
				if isLeftCol {
					canvas[y][x] = bottomLeft
				} else if isRightCol {
					canvas[y][x] = bottomRight
				} else {
					canvas[y][x] = horizontal
				}
			} else if isLeftCol || isRightCol {
				canvas[y][x] = vertical
			}
		}
	}

	contentStartLine := 0
	if box.Title != "" {

		titleLines := strings.Split(box.Title, "\n")
		maxWidth := box.Width - 2
		if maxWidth < 0 {
			maxWidth = 0
		}

		for lineIdx, titleLine := range titleLines {
			titleY := boxY + 1 + lineIdx
			if titleY >= 0 && titleY < len(canvas) {
				titleText := titleLine
				if len(titleText) > maxWidth {
					titleText = titleText[:maxWidth]
				}

				titleX := boxX + 1

				for i, char := range titleText {
					if titleX+i >= 0 && titleX+i < len(canvas[titleY]) && titleX+i < boxX+box.Width-1 {
						canvas[titleY][titleX+i] = char
					}
				}
			}
		}

		dividerY := boxY + 1 + len(titleLines)
		if dividerY >= 0 && dividerY < len(canvas) {
			divStartX := boxX + 1
			if divStartX < 0 {
				divStartX = 0
			}
			divEndX := boxX + box.Width - 1
			if divEndX > len(canvas[dividerY]) {
				divEndX = len(canvas[dividerY])
			}
			for x := divStartX; x < divEndX; x++ {
				canvas[dividerY][x] = horizontal
			}
		}

		contentStartLine = 1 + len(titleLines) + 1
	} else {
		contentStartLine = 1
	}

	for lineIdx, line := range box.Lines {
		textY := boxY + contentStartLine + lineIdx
		textX := boxX + 1
		if textY >= 0 && textY < len(canvas) && textY < boxY+box.Height-1 {
			maxWidth := box.Width - 2
			if maxWidth < 0 {
				maxWidth = 0
			}
			displayText := line
			if len(displayText) > maxWidth {
				displayText = displayText[:maxWidth]
			}

			for i, char := range displayText {
				charX := textX + i
				if charX >= 0 && charX < len(canvas[textY]) && charX < boxX+box.Width-1 {
					canvas[textY][charX] = char
				}
			}
		}
	}
}

func (c *Canvas) drawTextWithPan(canvas [][]rune, text Text, panX, panY int) {
	c.drawTextAt(canvas, text, text.X-panX, text.Y-panY)
}

func (c *Canvas) drawTextAt(canvas [][]rune, text Text, textX, textY int) {
	for lineIdx, line := range text.Lines {
		lineY := textY + lineIdx
		lineX := textX
		if lineY >= 0 && lineY < len(canvas) {

			for i, char := range line {
				charX := lineX + i
				if charX >= 0 && charX < len(canvas[lineY]) {
					canvas[lineY][charX] = char
				}
			}
		}
	}
}

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
	if len(lines) > 0 {
		return originX + len([]rune(lines[len(lines)-1])) - panX, originY + len(lines) - 1 - panY
	}
	return originX - panX, originY - panY
}

func (c *Canvas) calculateTextCursorPosition(box Box, cursorPos int, text string, panX, panY int) (int, int) {
	contentStartY := 1
	if box.Title != "" {
		contentStartY = 1 + len(strings.Split(box.Title, "\n")) + 1
	}
	return cursorScreenPos(box.X+1, box.Y+contentStartY, cursorPos, text, panX, panY)
}

func (c *Canvas) calculateTitleCursorPosition(box Box, cursorPos int, text string, panX, panY int) (int, int) {
	return cursorScreenPos(box.X+1, box.Y+1, cursorPos, text, panX, panY)
}

func (c *Canvas) calculateTextCursorPositionForText(text Text, cursorPos int, textContent string, panX, panY int) (int, int) {
	return cursorScreenPos(text.X, text.Y, cursorPos, textContent, panX, panY)
}

func (c *Canvas) calculateTextCursorPositionForNewText(textX, textY int, cursorPos int, textContent string, panX, panY int) (int, int) {
	return cursorScreenPos(textX, textY, cursorPos, textContent, panX, panY)
}

func (c *Canvas) drawConnectionWithPan(canvas [][]rune, connection Connection, panX, panY int) {
	adjustedConnection := connection
	adjustedConnection.FromX = connection.FromX - panX
	adjustedConnection.FromY = connection.FromY - panY
	adjustedConnection.ToX = connection.ToX - panX
	adjustedConnection.ToY = connection.ToY - panY
	adjustedConnection.Waypoints = make([]Point, len(connection.Waypoints))
	for i, wp := range connection.Waypoints {
		adjustedConnection.Waypoints[i] = Point{X: wp.X - panX, Y: wp.Y - panY}
	}
	c.drawConnection(canvas, adjustedConnection, connection, panX, panY)
}

func (c *Canvas) drawConnection(canvas [][]rune, connection Connection, originalConnection Connection, panX, panY int) {
	pts := []Point{{connection.FromX, connection.FromY}}
	pts = append(pts, connection.Waypoints...)
	pts = append(pts, Point{connection.ToX, connection.ToY})

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
			addV(Point{X: to.X, Y: from.Y})
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
			y0, y1 := a.Y, b.Y
			if y0 > y1 {
				y0, y1 = y1, y0
			}
			for y := y0; y <= y1; y++ {
				put(a.X, y, '│')
			}
		} else {
			x0, x1 := a.X, b.X
			if x0 > x1 {
				x0, x1 = x1, x0
			}
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
		c.drawConnEndArrow(canvas, connection.FromID, originalConnection.FromX, originalConnection.FromY, panX, panY)
	}
	if connection.ArrowTo {
		c.drawConnEndArrow(canvas, connection.ToID, originalConnection.ToX, originalConnection.ToY, panX, panY)
	}
}

func cornerChar(prev, cur, next Point) rune {
	if prev.Y == cur.Y && next.X == cur.X && prev.X != cur.X && next.Y != cur.Y {
		if prev.X < cur.X {
			if next.Y > cur.Y {
				return '┐'
			}
			return '┘'
		}
		if next.Y > cur.Y {
			return '┌'
		}
		return '└'
	}
	if prev.X == cur.X && next.Y == cur.Y && prev.Y != cur.Y && next.X != cur.X {
		if prev.Y < cur.Y {
			if next.X > cur.X {
				return '└'
			}
			return '┘'
		}
		if next.X > cur.X {
			return '┌'
		}
		return '┐'
	}
	return 0
}

func (c *Canvas) drawConnEndArrow(canvas [][]rune, boxID, ax, ay, panX, panY int) {
	if boxID < 0 || boxID >= len(c.boxes) {
		return
	}
	box := c.boxes[boxID]
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
	return y >= 0 && y < len(canvas) && x >= 0 && x < len(canvas[0])
}

func getColorCode(colorIndex int) string {

	if colorIndex == colorEditSelect {
		return "\x1b[7;36m"
	}
	if colorIndex == ColorMouseSelect {
		return "\x1b[103m"
	}
	if colorIndex == ColorMenuSelect {
		return "\x1b[7m"
	}
	if colorIndex == ColorMenuBorder {
		return "\x1b[32m"
	}
	colors := []int{47, 41, 42, 43, 44, 45, 46, 47}
	if colorIndex < 0 || colorIndex >= len(colors) {
		return ""
	}
	return fmt.Sprintf("\x1b[%dm", colors[colorIndex])
}

func getTextColorCode(colorIndex int) string {

	if colorIndex == colorEditSelect {
		return "\x1b[7;36m"
	}
	if colorIndex == ColorMouseSelect {
		return "\x1b[1;93m"
	}
	if colorIndex == ColorMenuSelect {
		return "\x1b[7m"
	}
	if colorIndex == ColorMenuBorder {
		return "\x1b[32m"
	}
	colors := []int{37, 31, 32, 33, 34, 35, 36, 37}
	if colorIndex < 0 || colorIndex >= len(colors) {
		return ""
	}
	return fmt.Sprintf("\x1b[%dm", colors[colorIndex])
}

const colorReset = "\x1b[0m"
