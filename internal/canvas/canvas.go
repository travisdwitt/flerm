package canvas

type Canvas struct {
	boxes       []Box
	connections []Connection
	texts       []Text
	highlights  map[string]int
}

func NewCanvas() *Canvas {
	return &Canvas{
		boxes:       make([]Box, 0),
		connections: make([]Connection, 0),
		texts:       make([]Text, 0),
		highlights:  make(map[string]int),
	}
}

func (c *Canvas) GetFullBounds() (int, int, int, int) {
	minX, minY := 0, 0
	maxX, maxY := 0, 0
	hasElements := false

	for _, box := range c.boxes {
		if !hasElements {
			minX, minY = box.X, box.Y
			maxX, maxY = box.X+box.Width, box.Y+box.Height
			hasElements = true
		} else {
			if box.X < minX {
				minX = box.X
			}
			if box.Y < minY {
				minY = box.Y
			}
			if box.X+box.Width > maxX {
				maxX = box.X + box.Width
			}
			if box.Y+box.Height > maxY {
				maxY = box.Y + box.Height
			}
		}
	}

	for _, conn := range c.connections {
		points := []Point{{X: conn.FromX, Y: conn.FromY}}
		points = append(points, conn.Waypoints...)
		points = append(points, Point{X: conn.ToX, Y: conn.ToY})

		for _, pt := range points {
			if !hasElements {
				minX, minY = pt.X, pt.Y
				maxX, maxY = pt.X, pt.Y
				hasElements = true
			} else {
				if pt.X < minX {
					minX = pt.X
				}
				if pt.Y < minY {
					minY = pt.Y
				}
				if pt.X > maxX {
					maxX = pt.X
				}
				if pt.Y > maxY {
					maxY = pt.Y
				}
			}
		}
	}

	for _, text := range c.texts {
		if !hasElements {
			minX, minY = text.X, text.Y
			maxX, maxY = text.X, text.Y
			hasElements = true
		} else {
			if text.X < minX {
				minX = text.X
			}
			if text.Y < minY {
				minY = text.Y
			}
		}

		maxTextX := text.X
		for _, line := range text.Lines {
			if text.X+len(line) > maxTextX {
				maxTextX = text.X + len(line)
			}
		}
		if maxTextX > maxX {
			maxX = maxTextX
		}
		if text.Y+len(text.Lines) > maxY {
			maxY = text.Y + len(text.Lines)
		}
	}

	if !hasElements {
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
	textObj := Text{
		X:     x,
		Y:     y,
		ID:    id,
		Color: -1,
	}
	textObj.SetText(text)
	c.texts = append(c.texts, textObj)
	for i := id + 1; i < len(c.texts); i++ {
		c.texts[i].ID = i
	}
}

func (c *Canvas) AddBoxWithID(x, y int, text string, id int) {
	box := Box{
		X:     x,
		Y:     y,
		ID:    id,
		Color: -1,
	}
	box.SetText(text)
	if id >= len(c.boxes) {
		for len(c.boxes) <= id {
			c.boxes = append(c.boxes, Box{})
		}
		c.boxes[id] = box
	} else {
		c.boxes = append(c.boxes, Box{})
		copy(c.boxes[id+1:], c.boxes[id:])
		c.boxes[id] = box
		for i := id + 1; i < len(c.boxes); i++ {
			c.boxes[i].ID = i
		}
	}
}

func (c *Canvas) GetBoxAt(x, y int) int {
	for i, box := range c.boxes {
		if x >= box.X && x < box.X+box.Width &&
			y >= box.Y && y < box.Y+box.Height {
			return i
		}
	}
	return -1
}

func (c *Canvas) GetTextAt(x, y int) int {
	for i, text := range c.texts {

		for lineIdx, line := range text.Lines {
			lineY := text.Y + lineIdx
			if y == lineY && x >= text.X && x < text.X+len(line) {
				return i
			}
		}
	}
	return -1
}

func (c *Canvas) DeleteText(id int) {
	if id >= 0 && id < len(c.texts) {
		c.deleteHighlightsForText(id)
		c.texts = append(c.texts[:id], c.texts[id+1:]...)
		for i := id; i < len(c.texts); i++ {
			c.texts[i].ID = i
		}
	}
}

func (c *Canvas) GetBoxText(id int) string {
	if id >= 0 && id < len(c.boxes) {
		return c.boxes[id].GetText()
	}
	return ""
}

func (c *Canvas) SetBoxText(id int, text string) {
	if id >= 0 && id < len(c.boxes) {
		c.boxes[id].SetText(text)
	}
}

func (c *Canvas) GetTextText(id int) string {
	if id >= 0 && id < len(c.texts) {
		return c.texts[id].GetText()
	}
	return ""
}

func (c *Canvas) SetTextText(id int, text string) {
	if id >= 0 && id < len(c.texts) {
		c.texts[id].SetText(text)
	}
}

func (c *Canvas) DeleteBox(id int) {
	if id >= 0 && id < len(c.boxes) {
		c.deleteHighlightsForBox(id)
		c.boxes = append(c.boxes[:id], c.boxes[id+1:]...)
		for i := id; i < len(c.boxes); i++ {
			c.boxes[i].ID = i
		}
		newConnections := make([]Connection, 0)
		for _, connection := range c.connections {
			if connection.FromID != id && connection.ToID != id {
				if connection.FromID > id {
					connection.FromID--
				}
				if connection.ToID > id {
					connection.ToID--
				}
				newConnections = append(newConnections, connection)
			}
		}
		c.connections = newConnections
	}
}

func (c *Canvas) ResizeBox(id int, deltaWidth, deltaHeight int) {
	if id >= 0 && id < len(c.boxes) {
		box := &c.boxes[id]
		newWidth := box.Width + deltaWidth
		newHeight := box.Height + deltaHeight

		if newWidth < minBoxWidth {
			newWidth = minBoxWidth
		}
		if newHeight < minBoxHeight {
			newHeight = minBoxHeight
		}

		if newWidth != box.Width || newHeight != box.Height {
			box.fitTextToSize(newWidth, newHeight)
		}

		oldBoxX := box.X
		oldBoxWidth := box.Width

		box.Width = newWidth
		box.Height = newHeight

		c.reanchorConnectionsForResize(id, oldBoxX, oldBoxWidth)
	}
}

func (c *Canvas) reanchorConnectionsForResize(id, oldBoxX, oldBoxWidth int) {
	box := &c.boxes[id]
	for i := range c.connections {
		conn := &c.connections[i]
		if conn.FromID == id && conn.ToID >= 0 && conn.ToID < len(c.boxes) {
			wasHorizontal := (conn.FromY == conn.ToY)
			oldFromX := conn.FromX
			oldToX := conn.ToX
			newFromX, newFromY, newToX, newToY := c.calculateConnectionPointsPreservingOrientation(id, conn.ToID, wasHorizontal)
			if wasHorizontal {
				oldToBox := c.boxes[conn.ToID]
				wasOnLeft := (oldToX == oldToBox.X || (oldToX < oldToBox.X+oldToBox.Width/2))
				if wasOnLeft {
					newToX = oldToBox.X
				} else {
					newToX = oldToBox.X + oldToBox.Width - 1
				}
				wasFromRight := (oldFromX == oldBoxX+oldBoxWidth-1 || (oldFromX > oldBoxX+oldBoxWidth/2))
				if wasFromRight {
					newFromX = box.X + box.Width - 1
				} else {
					newFromX = box.X
				}
			}
			conn.FromX = newFromX
			conn.FromY = newFromY
			conn.ToX = newToX
			conn.ToY = newToY
		}
		if conn.ToID == id && conn.FromID >= 0 && conn.FromID < len(c.boxes) {
			wasHorizontal := (conn.FromY == conn.ToY)
			oldToX := conn.ToX
			oldFromX := conn.FromX
			newFromX, newFromY, newToX, newToY := c.calculateConnectionPointsPreservingOrientation(conn.FromID, id, wasHorizontal)
			if wasHorizontal {
				wasOnLeft := (oldToX == oldBoxX || (oldToX < oldBoxX+oldBoxWidth/2))
				if wasOnLeft {
					newToX = box.X
				} else {
					newToX = box.X + box.Width - 1
				}
				oldFromBox := c.boxes[conn.FromID]
				wasFromRight := (oldFromX == oldFromBox.X+oldFromBox.Width-1 || (oldFromX > oldFromBox.X+oldFromBox.Width/2))
				if wasFromRight {
					newFromX = oldFromBox.X + oldFromBox.Width - 1
				} else {
					newFromX = oldFromBox.X
				}
			}
			conn.FromX = newFromX
			conn.FromY = newFromY
			conn.ToX = newToX
			conn.ToY = newToY
		}
	}
}

func (c *Canvas) MoveBoxOnly(id int, deltaX, deltaY int) {
	if id >= 0 && id < len(c.boxes) {
		box := &c.boxes[id]
		box.X += deltaX
		box.Y += deltaY
		if box.X < 0 {
			box.X = 0
		}
		if box.Y < 0 {
			box.Y = 0
		}
	}
}

func (c *Canvas) MoveBox(id int, deltaX, deltaY int) {
	if id >= 0 && id < len(c.boxes) {
		oldX, oldY := c.boxes[id].X, c.boxes[id].Y
		c.MoveBoxOnly(id, deltaX, deltaY)
		c.rerouteConnectionsForMovedBox(id, c.boxes[id].X-oldX, c.boxes[id].Y-oldY)
	}
}

func (c *Canvas) CycleBoxZLevel(id int) {
	if id >= 0 && id < len(c.boxes) {
		c.boxes[id].ZLevel = (c.boxes[id].ZLevel + 1) % 4
	}
}

func (c *Canvas) SetBoxPositionOnly(id int, x, y int) {
	if id >= 0 && id < len(c.boxes) {
		box := &c.boxes[id]
		box.X, box.Y = x, y
		if box.X < 0 {
			box.X = 0
		}
		if box.Y < 0 {
			box.Y = 0
		}
	}
}

func (c *Canvas) MoveText(id int, deltaX, deltaY int) {
	if id >= 0 && id < len(c.texts) {
		text := &c.texts[id]
		text.X += deltaX
		text.Y += deltaY
		if text.X < 0 {
			text.X = 0
		}
		if text.Y < 0 {
			text.Y = 0
		}
	}
}

func (c *Canvas) SetTextPosition(id int, x, y int) {
	if id >= 0 && id < len(c.texts) {
		text := &c.texts[id]
		text.X, text.Y = x, y
		if text.X < 0 {
			text.X = 0
		}
		if text.Y < 0 {
			text.Y = 0
		}
	}
}

func (c *Canvas) SetBoxSize(id int, width, height int) {
	if id >= 0 && id < len(c.boxes) {
		box := &c.boxes[id]
		oldBoxX, oldBoxWidth := box.X, box.Width
		oldWidth, oldHeight := box.Width, box.Height
		if width < minBoxWidth {
			width = minBoxWidth
		}
		if height < minBoxHeight {
			height = minBoxHeight
		}
		box.Width, box.Height = width, height

		if box.Width != oldWidth || box.Height != oldHeight {
			c.reanchorConnectionsForResize(id, oldBoxX, oldBoxWidth)
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func (c *Canvas) CycleBorderStyle(boxID int) BorderStyle {
	if boxID >= 0 && boxID < len(c.boxes) {
		oldStyle := c.boxes[boxID].BorderStyle
		var newStyle BorderStyle
		switch oldStyle {
		case BorderStyleASCII:
			newStyle = BorderStyleSingle
		case BorderStyleSingle:
			newStyle = BorderStyleDouble
		case BorderStyleDouble:
			newStyle = BorderStyleRounded
		case BorderStyleRounded:
			newStyle = BorderStyleASCII
		default:
			newStyle = BorderStyleASCII
		}
		c.boxes[boxID].BorderStyle = newStyle
		return oldStyle
	}
	return BorderStyleASCII
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
