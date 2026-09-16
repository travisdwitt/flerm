package canvas

type Canvas struct {
	boxes       []Box
	connections []Connection
	texts       []Text
	highlights  map[Point]int
}

func NewCanvas() *Canvas {
	return &Canvas{highlights: make(map[Point]int)}
}

func (c *Canvas) box(id int) (Box, bool) {
	if id < 0 || id >= len(c.boxes) {
		return Box{}, false
	}
	return c.boxes[id], true
}

// connPoints returns a connection's full polyline: origin, waypoints, endpoint.
func connPoints(conn Connection) []Point {
	pts := make([]Point, 0, len(conn.Waypoints)+2)
	pts = append(pts, Point{conn.FromX, conn.FromY})
	pts = append(pts, conn.Waypoints...)
	return append(pts, Point{conn.ToX, conn.ToY})
}

func (c *Canvas) GetFullBounds() (int, int, int, int) {
	var minX, minY, maxX, maxY int
	has := false
	grow := func(x0, y0, x1, y1 int) {
		if !has {
			minX, minY, maxX, maxY, has = x0, y0, x1, y1, true
			return
		}
		minX, minY = min(minX, x0), min(minY, y0)
		maxX, maxY = max(maxX, x1), max(maxY, y1)
	}

	for _, box := range c.boxes {
		grow(box.X, box.Y, box.X+box.Width, box.Y+box.Height)
	}
	for _, conn := range c.connections {
		for _, pt := range connPoints(conn) {
			grow(pt.X, pt.Y, pt.X, pt.Y)
		}
	}
	for _, text := range c.texts {
		maxTextX := text.X
		for _, line := range text.Lines {
			maxTextX = max(maxTextX, text.X+len(line))
		}
		grow(text.X, text.Y, maxTextX, text.Y+len(text.Lines))
	}

	if !has {
		return 1, 1, 0, 0
	}
	return minX, minY, maxX, maxY
}

func (c *Canvas) AddBox(x, y int, text string) {
	c.AddBoxWithID(x, y, text, len(c.boxes))
}

func (c *Canvas) AddText(x, y int, text string) {
	c.AddTextWithID(x, y, text, len(c.texts))
}

func (c *Canvas) AddTextWithID(x, y int, text string, id int) {
	textObj := Text{X: x, Y: y, ID: id, Color: -1}
	textObj.SetText(text)
	c.texts = append(c.texts, textObj)
	for i := id + 1; i < len(c.texts); i++ {
		c.texts[i].ID = i
	}
}

func (c *Canvas) AddBoxWithID(x, y int, text string, id int) {
	box := Box{X: x, Y: y, ID: id, Color: -1}
	box.SetText(text)
	if id >= len(c.boxes) {
		for len(c.boxes) <= id {
			c.boxes = append(c.boxes, Box{})
		}
		c.boxes[id] = box
		return
	}
	c.boxes = append(c.boxes, Box{})
	copy(c.boxes[id+1:], c.boxes[id:])
	c.boxes[id] = box
	for i := id + 1; i < len(c.boxes); i++ {
		c.boxes[i].ID = i
	}
}

func (c *Canvas) GetBoxAt(x, y int) int {
	for i, box := range c.boxes {
		if x >= box.X && x < box.X+box.Width && y >= box.Y && y < box.Y+box.Height {
			return i
		}
	}
	return -1
}

func (c *Canvas) GetTextAt(x, y int) int {
	for i, text := range c.texts {
		for lineIdx, line := range text.Lines {
			if y == text.Y+lineIdx && x >= text.X && x < text.X+len(line) {
				return i
			}
		}
	}
	return -1
}

func (c *Canvas) DeleteText(id int) {
	if id < 0 || id >= len(c.texts) {
		return
	}
	c.deleteHighlights(c.GetTextCells(id))
	c.texts = append(c.texts[:id], c.texts[id+1:]...)
	for i := id; i < len(c.texts); i++ {
		c.texts[i].ID = i
	}
}

func (c *Canvas) DeleteBox(id int) {
	if id < 0 || id >= len(c.boxes) {
		return
	}
	c.deleteHighlights(c.GetBoxCells(id))
	c.boxes = append(c.boxes[:id], c.boxes[id+1:]...)
	for i := id; i < len(c.boxes); i++ {
		c.boxes[i].ID = i
	}
	kept := make([]Connection, 0, len(c.connections))
	for _, conn := range c.connections {
		if conn.FromID == id || conn.ToID == id {
			continue
		}
		if conn.FromID > id {
			conn.FromID--
		}
		if conn.ToID > id {
			conn.ToID--
		}
		kept = append(kept, conn)
	}
	c.connections = kept
}

func (c *Canvas) GetBoxText(id int) string {
	box, ok := c.box(id)
	if !ok {
		return ""
	}
	return box.GetText()
}

func (c *Canvas) SetBoxText(id int, text string) {
	if id >= 0 && id < len(c.boxes) {
		c.boxes[id].SetText(text)
	}
}

func (c *Canvas) GetTextText(id int) string {
	if id < 0 || id >= len(c.texts) {
		return ""
	}
	return c.texts[id].GetText()
}

func (c *Canvas) SetTextText(id int, text string) {
	if id >= 0 && id < len(c.texts) {
		c.texts[id].SetText(text)
	}
}

func (c *Canvas) ResizeBox(id, deltaWidth, deltaHeight int) {
	if id < 0 || id >= len(c.boxes) {
		return
	}
	box := &c.boxes[id]
	w := max(box.Width+deltaWidth, minBoxWidth)
	h := max(box.Height+deltaHeight, minBoxHeight)
	oldX, oldWidth := box.X, box.Width
	if w != box.Width || h != box.Height {
		box.fitTextToSize(w, h)
	}
	box.Width, box.Height = w, h
	c.reanchorConnectionsForResize(id, oldX, oldWidth)
}

// SetBoxSize sets an exact size, leaving the text lines alone (undo restores
// the size a resize produced; the lines it produced are restored separately).
func (c *Canvas) SetBoxSize(id, width, height int) {
	if id < 0 || id >= len(c.boxes) {
		return
	}
	box := &c.boxes[id]
	oldX, oldWidth := box.X, box.Width
	width, height = max(width, minBoxWidth), max(height, minBoxHeight)
	if width == box.Width && height == box.Height {
		return
	}
	box.Width, box.Height = width, height
	c.reanchorConnectionsForResize(id, oldX, oldWidth)
}

// resizeAnchorX re-pins a horizontal connection endpoint to whichever vertical
// edge of the box it sat on before the resize. tieRight breaks a dead-centre tie.
func resizeAnchorX(oldX, oldBX, oldBW int, box Box, tieRight bool) int {
	mid := oldBX + oldBW/2
	right := oldX == oldBX+oldBW-1 || oldX > mid
	if oldX == oldBX {
		right = false
	} else if oldX == mid && oldX != oldBX+oldBW-1 {
		right = tieRight
	}
	if right {
		return box.X + box.Width - 1
	}
	return box.X
}

func (c *Canvas) reanchorConnectionsForResize(id, oldBoxX, oldBoxWidth int) {
	for i := range c.connections {
		conn := &c.connections[i]
		fromID, toID := conn.FromID, conn.ToID
		if fromID != id && toID != id {
			continue
		}
		other := fromID
		if fromID == id {
			other = toID
		}
		if other < 0 || other >= len(c.boxes) {
			continue
		}

		horiz := conn.FromY == conn.ToY
		oldFromX, oldToX := conn.FromX, conn.ToX
		conn.FromX, conn.FromY, conn.ToX, conn.ToY =
			c.calculateConnectionPointsPreservingOrientation(fromID, toID, horiz)
		if !horiz {
			continue
		}
		// Decide the side from each endpoint's pre-resize geometry.
		fromBX, fromBW := c.boxes[fromID].X, c.boxes[fromID].Width
		toBX, toBW := c.boxes[toID].X, c.boxes[toID].Width
		if fromID == id {
			fromBX, fromBW = oldBoxX, oldBoxWidth
		} else {
			toBX, toBW = oldBoxX, oldBoxWidth
		}
		conn.FromX = resizeAnchorX(oldFromX, fromBX, fromBW, c.boxes[fromID], false)
		conn.ToX = resizeAnchorX(oldToX, toBX, toBW, c.boxes[toID], true)
	}
}

func clampNonNeg(x, y int) (int, int) { return max(x, 0), max(y, 0) }

func (c *Canvas) MoveBoxOnly(id, deltaX, deltaY int) {
	if id >= 0 && id < len(c.boxes) {
		box := &c.boxes[id]
		box.X, box.Y = clampNonNeg(box.X+deltaX, box.Y+deltaY)
	}
}

func (c *Canvas) MoveBox(id, deltaX, deltaY int) {
	if id < 0 || id >= len(c.boxes) {
		return
	}
	oldX, oldY := c.boxes[id].X, c.boxes[id].Y
	c.MoveBoxOnly(id, deltaX, deltaY)
	c.rerouteConnectionsForMovedBox(id, c.boxes[id].X-oldX, c.boxes[id].Y-oldY)
}

func (c *Canvas) SetBoxPositionOnly(id, x, y int) {
	if id >= 0 && id < len(c.boxes) {
		c.boxes[id].X, c.boxes[id].Y = clampNonNeg(x, y)
	}
}

func (c *Canvas) MoveText(id, deltaX, deltaY int) {
	if id >= 0 && id < len(c.texts) {
		t := &c.texts[id]
		t.X, t.Y = clampNonNeg(t.X+deltaX, t.Y+deltaY)
	}
}

func (c *Canvas) SetTextPosition(id, x, y int) {
	if id >= 0 && id < len(c.texts) {
		c.texts[id].X, c.texts[id].Y = clampNonNeg(x, y)
	}
}

func (c *Canvas) CycleBoxZLevel(id int) {
	if id >= 0 && id < len(c.boxes) {
		c.boxes[id].ZLevel = (c.boxes[id].ZLevel + 1) % numZLevels
	}
}

func (c *Canvas) CycleBorderStyle(boxID int) BorderStyle {
	if boxID < 0 || boxID >= len(c.boxes) {
		return BorderStyleASCII
	}
	old := c.boxes[boxID].BorderStyle
	next := old + 1
	if next < BorderStyleASCII || next > BorderStyleRounded {
		next = BorderStyleASCII
	}
	c.boxes[boxID].BorderStyle = next
	return old
}

func (c *Canvas) SetBorderStyle(boxID int, style BorderStyle) {
	if boxID >= 0 && boxID < len(c.boxes) {
		c.boxes[boxID].BorderStyle = style
	}
}

func (c *Canvas) SetBoxColor(boxID, color int) {
	if boxID >= 0 && boxID < len(c.boxes) {
		c.boxes[boxID].Color = color
	}
}

func (c *Canvas) SetTextColor(textID, color int) {
	if textID >= 0 && textID < len(c.texts) {
		c.texts[textID].Color = color
	}
}

func (c *Canvas) SetLineColor(connIdx, color int) {
	if connIdx >= 0 && connIdx < len(c.connections) {
		c.connections[connIdx].Color = color
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
