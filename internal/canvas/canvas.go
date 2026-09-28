package canvas

import "slices"

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

func (c *Canvas) boxPtr(id int) *Box {
	if id < 0 || id >= len(c.boxes) {
		return nil
	}
	return &c.boxes[id]
}

func (c *Canvas) textPtr(id int) *Text {
	if id < 0 || id >= len(c.texts) {
		return nil
	}
	return &c.texts[id]
}

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
	c.boxes = slices.Insert(c.boxes, id, box)
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
	if c.textPtr(id) == nil {
		return
	}
	c.deleteHighlights(c.GetTextCells(id))
	c.texts = slices.Delete(c.texts, id, id+1)
	for i := id; i < len(c.texts); i++ {
		c.texts[i].ID = i
	}
}

func (c *Canvas) DeleteBox(id int) {
	if c.boxPtr(id) == nil {
		return
	}
	c.deleteHighlights(c.GetBoxCells(id))
	c.boxes = slices.Delete(c.boxes, id, id+1)
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
	if b := c.boxPtr(id); b != nil {
		return b.GetText()
	}
	return ""
}

func (c *Canvas) SetBoxText(id int, text string) {
	if b := c.boxPtr(id); b != nil {
		b.SetText(text)
	}
}

func (c *Canvas) SetBoxTitle(id int, title string) {
	if b := c.boxPtr(id); b != nil {
		b.Title = title
		b.UpdateSize()
	}
}

func (c *Canvas) GetTextText(id int) string {
	if t := c.textPtr(id); t != nil {
		return t.GetText()
	}
	return ""
}

func (c *Canvas) SetTextText(id int, text string) {
	if t := c.textPtr(id); t != nil {
		t.SetText(text)
	}
}

func (c *Canvas) ResizeBox(id, deltaWidth, deltaHeight int) {
	box := c.boxPtr(id)
	if box == nil {
		return
	}
	w := max(box.Width+deltaWidth, minBoxWidth)
	h := max(box.Height+deltaHeight, minBoxHeight)
	oldX, oldWidth := box.X, box.Width
	if w != box.Width || h != box.Height {
		box.fitTextToSize(w, h)
	}
	box.Width, box.Height = w, h
	c.reanchorConnectionsForResize(id, oldX, oldWidth)
}

func (c *Canvas) SetBoxSize(id, width, height int) {
	box := c.boxPtr(id)
	if box == nil {
		return
	}
	oldX, oldWidth := box.X, box.Width
	width, height = max(width, minBoxWidth), max(height, minBoxHeight)
	if width == box.Width && height == box.Height {
		return
	}
	box.Width, box.Height = width, height
	c.reanchorConnectionsForResize(id, oldX, oldWidth)
}

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
		if c.boxPtr(other) == nil {
			continue
		}

		horiz := conn.FromY == conn.ToY
		oldFromX, oldToX := conn.FromX, conn.ToX
		conn.FromX, conn.FromY, conn.ToX, conn.ToY =
			c.calculateConnectionPointsPreservingOrientation(fromID, toID, horiz)
		if !horiz {
			continue
		}
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

func (c *Canvas) MoveBoxOnly(id, deltaX, deltaY int) {
	if b := c.boxPtr(id); b != nil {
		b.X, b.Y = b.X+deltaX, b.Y+deltaY
	}
}

func (c *Canvas) MoveBox(id, deltaX, deltaY int) {
	if c.boxPtr(id) == nil {
		return
	}
	c.MoveBoxOnly(id, deltaX, deltaY)
	c.rerouteConnectionsForMovedBox(id, deltaX, deltaY)
}

func (c *Canvas) SetBoxPositionOnly(id, x, y int) {
	if b := c.boxPtr(id); b != nil {
		b.X, b.Y = x, y
	}
}

func (c *Canvas) MoveText(id, deltaX, deltaY int) {
	if t := c.textPtr(id); t != nil {
		t.X, t.Y = t.X+deltaX, t.Y+deltaY
	}
}

func (c *Canvas) SetTextPosition(id, x, y int) {
	if t := c.textPtr(id); t != nil {
		t.X, t.Y = x, y
	}
}

func (c *Canvas) CycleBoxZLevel(id int) {
	if b := c.boxPtr(id); b != nil {
		b.ZLevel = (b.ZLevel + 1) % numZLevels
	}
}

func (c *Canvas) CycleBorderStyle(id int) BorderStyle {
	b := c.boxPtr(id)
	if b == nil {
		return BorderStyleASCII
	}
	old := b.BorderStyle
	b.BorderStyle = BorderStyleASCII
	if old >= BorderStyleASCII && old < BorderStyleRounded {
		b.BorderStyle = old + 1
	}
	return old
}

func (c *Canvas) SetBorderStyle(id int, style BorderStyle) {
	if b := c.boxPtr(id); b != nil {
		b.BorderStyle = style
	}
}

func (c *Canvas) SetBoxColor(id, color int) {
	if b := c.boxPtr(id); b != nil {
		b.Color = color
	}
}

func (c *Canvas) SetTextColor(id, color int) {
	if t := c.textPtr(id); t != nil {
		t.Color = color
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

func (c *Canvas) duplicateHighlights(cells []Point, dx, dy int) {
	for _, cell := range cells {
		if color, ok := c.highlights[cell]; ok {
			c.highlights[Point{cell.X + dx, cell.Y + dy}] = color
		}
	}
}

func (c *Canvas) DuplicateBox(id, dx, dy int) int {
	box, ok := c.box(id)
	if !ok {
		return -1
	}
	cells := c.GetBoxCells(id)
	box.ID = len(c.boxes)
	box.X, box.Y = box.X+dx, box.Y+dy
	box.Lines = slices.Clone(box.Lines)
	c.boxes = append(c.boxes, box)
	c.duplicateHighlights(cells, dx, dy)
	return box.ID
}

func (c *Canvas) DuplicateText(id, dx, dy int) int {
	tp := c.textPtr(id)
	if tp == nil {
		return -1
	}
	cells := c.GetTextCells(id)
	t := *tp
	t.ID = len(c.texts)
	t.X, t.Y = t.X+dx, t.Y+dy
	t.Lines = slices.Clone(t.Lines)
	c.texts = append(c.texts, t)
	c.duplicateHighlights(cells, dx, dy)
	return t.ID
}
