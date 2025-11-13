package main

import (
	"bufio"
	"fmt"
	"image/color"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
)

type Canvas struct {
	boxes       []Box
	connections []Connection
	texts       []Text
}

type Text struct {
	X     int
	Y     int
	Lines []string
	ID    int
}

func (t *Text) GetText() string {
	return strings.Join(t.Lines, "\n")
}

func (t *Text) SetText(text string) {
	t.Lines = strings.Split(text, "\n")
}

type Box struct {
	X      int
	Y      int
	Width  int
	Height int
	Lines  []string
	ID     int
}

func (b *Box) GetText() string {
	return strings.Join(b.Lines, "\n")
}

func (b *Box) SetText(text string) {
	b.Lines = strings.Split(text, "\n")
	b.updateSize()
}

func (b *Box) updateSize() {
	if len(b.Lines) == 0 {
		b.Lines = []string{""}
	}

	// Calculate width based on longest line
	maxWidth := minBoxWidth
	for _, line := range b.Lines {
		if len(line)+2 > maxWidth { // +2 for padding
			maxWidth = len(line) + 2
		}
	}
	b.Width = maxWidth

	// Height is number of lines + 2 for borders
	b.Height = len(b.Lines) + 2
}

type Connection struct {
	FromID    int
	ToID      int
	FromX     int
	FromY     int
	ToX       int
	ToY       int
	Waypoints []point
	ArrowFrom bool
	ArrowTo   bool
}

func NewCanvas() *Canvas {
	return &Canvas{
		boxes:       make([]Box, 0),
		connections: make([]Connection, 0),
		texts:       make([]Text, 0),
	}
}

func (c *Canvas) AddBox(x, y int, text string) {
	box := Box{
		X:  x,
		Y:  y,
		ID: len(c.boxes),
	}
	box.SetText(text)
	c.boxes = append(c.boxes, box)
}

func (c *Canvas) AddText(x, y int, text string) {
	textObj := Text{
		X:  x,
		Y:  y,
		ID: len(c.texts),
	}
	textObj.SetText(text)
	c.texts = append(c.texts, textObj)
}

func (c *Canvas) AddBoxWithID(x, y int, text string, id int) {
	box := Box{
		X:  x,
		Y:  y,
		ID: id,
	}
	box.SetText(text)

	// Insert box at the correct position to maintain ID order
	if id >= len(c.boxes) {
		// Extend slice if needed
		for len(c.boxes) <= id {
			c.boxes = append(c.boxes, Box{})
		}
		c.boxes[id] = box
	} else {
		// Insert at position and shift others
		c.boxes = append(c.boxes, Box{})
		copy(c.boxes[id+1:], c.boxes[id:])
		c.boxes[id] = box
		// Update IDs for shifted boxes
		for i := id + 1; i < len(c.boxes); i++ {
			c.boxes[i].ID = i
		}
	}
}

func (c *Canvas) findNearestPointOnConnection(cursorX, cursorY int) (int, int, int) {
	bestDist := -1
	bestConnIdx := -1
	bestX, bestY := -1, -1

	for i, conn := range c.connections {
		points := []point{
			{conn.FromX, conn.FromY},
		}
		points = append(points, conn.Waypoints...)
		points = append(points, point{conn.ToX, conn.ToY})

		for j := 0; j < len(points)-1; j++ {
			segX, segY := c.findClosestPointOnSegment(points[j].X, points[j].Y, points[j+1].X, points[j+1].Y, cursorX, cursorY)
			dist := abs(segX-cursorX) + abs(segY-cursorY)
			if bestDist == -1 || dist < bestDist {
				bestDist = dist
				bestConnIdx = i
				bestX, bestY = segX, segY
			}
		}
	}

	if bestDist != -1 && bestDist <= 2 {
		return bestConnIdx, bestX, bestY
	}
	return -1, -1, -1
}

func (c *Canvas) findClosestPointOnSegment(segX1, segY1, segX2, segY2, cursorX, cursorY int) (int, int) {
	if segX1 == segX2 {
		closestY := cursorY
		if closestY < min(segY1, segY2) {
			closestY = min(segY1, segY2)
		} else if closestY > max(segY1, segY2) {
			closestY = max(segY1, segY2)
		}
		return segX1, closestY
	} else if segY1 == segY2 {
		closestX := cursorX
		if closestX < min(segX1, segX2) {
			closestX = min(segX1, segX2)
		} else if closestX > max(segX1, segX2) {
			closestX = max(segX1, segX2)
		}
		return closestX, segY1
	} else {
		cornerX := segX2
		cornerY := segY1

		closestX1, closestY1 := c.findClosestPointOnSegment(segX1, segY1, cornerX, cornerY, cursorX, cursorY)
		dist1 := abs(closestX1-cursorX) + abs(closestY1-cursorY)

		closestX2, closestY2 := c.findClosestPointOnSegment(cornerX, cornerY, segX2, segY2, cursorX, cursorY)
		dist2 := abs(closestX2-cursorX) + abs(closestY2-cursorY)

		if dist1 < dist2 {
			return closestX1, closestY1
		}
		return closestX2, closestY2
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (c *Canvas) findNearestEdgePoint(box Box, cursorX, cursorY int) (int, int) {
	clampedX := cursorX
	if clampedX < box.X {
		clampedX = box.X
	} else if clampedX >= box.X+box.Width {
		clampedX = box.X + box.Width - 1
	}

	clampedY := cursorY
	if clampedY < box.Y {
		clampedY = box.Y
	} else if clampedY >= box.Y+box.Height {
		clampedY = box.Y + box.Height - 1
	}

	distToLeft := abs(cursorX - box.X)
	distToRight := abs(cursorX - (box.X + box.Width - 1))
	distToTop := abs(cursorY - box.Y)
	distToBottom := abs(cursorY - (box.Y + box.Height - 1))

	minDist := distToLeft
	edgeX := box.X
	edgeY := clampedY

	if distToRight < minDist {
		minDist = distToRight
		edgeX = box.X + box.Width - 1
		edgeY = clampedY
	}
	if distToTop < minDist {
		minDist = distToTop
		edgeX = clampedX
		edgeY = box.Y
	}
	if distToBottom < minDist {
		edgeX = clampedX
		edgeY = box.Y + box.Height - 1
	}

	return edgeX, edgeY
}

func (c *Canvas) calculateConnectionPoints(fromID, toID int) (fromX, fromY, toX, toY int) {
	if fromID < 0 || fromID >= len(c.boxes) || toID < 0 || toID >= len(c.boxes) {
		return 0, 0, 0, 0
	}

	fromBox := c.boxes[fromID]
	toBox := c.boxes[toID]

	fromCenterX := fromBox.X + fromBox.Width/2
	fromCenterY := fromBox.Y + fromBox.Height/2
	toCenterX := toBox.X + toBox.Width/2
	toCenterY := toBox.Y + toBox.Height/2
	if abs(fromCenterX-toCenterX) > abs(fromCenterY-toCenterY) {
		if fromCenterX < toCenterX {
			fromX = fromBox.X + fromBox.Width - 1
			fromY = fromCenterY
			toX = toBox.X
			toY = toCenterY
		} else {
			fromX = fromBox.X
			fromY = fromCenterY
			toX = toBox.X + toBox.Width - 1
			toY = toCenterY
		}
	} else {
		if fromCenterY < toCenterY {
			fromX = fromCenterX
			fromY = fromBox.Y + fromBox.Height - 1
			toX = toCenterX
			toY = toBox.Y
		} else {
			fromX = fromCenterX
			fromY = fromBox.Y
			toX = toCenterX
			toY = toBox.Y + toBox.Height - 1
		}
	}

	return fromX, fromY, toX, toY
}

func (c *Canvas) calculateConnectionPointsPreservingOrientation(fromID, toID int, preferHorizontal bool) (fromX, fromY, toX, toY int) {
	if fromID < 0 || fromID >= len(c.boxes) || toID < 0 || toID >= len(c.boxes) {
		return 0, 0, 0, 0
	}

	fromBox := c.boxes[fromID]
	toBox := c.boxes[toID]

	fromCenterX := fromBox.X + fromBox.Width/2
	fromCenterY := fromBox.Y + fromBox.Height/2
	toCenterX := toBox.X + toBox.Width/2
	toCenterY := toBox.Y + toBox.Height/2

	if preferHorizontal {
		if fromCenterX < toCenterX {
			fromX = fromBox.X + fromBox.Width - 1
			fromY = fromCenterY
			toX = toBox.X
			toY = toCenterY
		} else {
			fromX = fromBox.X
			fromY = fromCenterY
			toX = toBox.X + toBox.Width - 1
			toY = toCenterY
		}
	} else {
		if fromCenterY < toCenterY {
			fromX = fromCenterX
			fromY = fromBox.Y + fromBox.Height - 1
			toX = toCenterX
			toY = toBox.Y
		} else {
			fromX = fromCenterX
			fromY = fromBox.Y
			toX = toCenterX
			toY = toBox.Y + toBox.Height - 1
		}
	}

	return fromX, fromY, toX, toY
}

func (c *Canvas) AddConnection(fromID, toID int) {
	if fromID >= len(c.boxes) || toID >= len(c.boxes) {
		return
	}

	fromX, fromY, toX, toY := c.calculateConnectionPoints(fromID, toID)

	connection := Connection{
		FromID: fromID,
		ToID:   toID,
		FromX:  fromX,
		FromY:  fromY,
		ToX:    toX,
		ToY:    toY,
	}
	c.connections = append(c.connections, connection)
}

func (c *Canvas) AddConnectionWithPoints(fromID, toID, fromX, fromY, toX, toY int) {
	c.AddConnectionWithWaypoints(fromID, toID, fromX, fromY, toX, toY, nil)
}

func (c *Canvas) AddConnectionWithWaypoints(fromID, toID, fromX, fromY, toX, toY int, waypoints []point) {
	if fromID != -1 && fromID >= len(c.boxes) {
		return
	}
	if toID != -1 && toID >= len(c.boxes) {
		return
	}

	connection := Connection{
		FromID:    fromID,
		ToID:      toID,
		FromX:     fromX,
		FromY:     fromY,
		ToX:       toX,
		ToY:       toY,
		Waypoints: waypoints,
		ArrowFrom: false,
		ArrowTo:   true,
	}
	c.connections = append(c.connections, connection)
}

func (c *Canvas) RemoveConnection(fromID, toID int) {
	newConnections := make([]Connection, 0)
	for _, connection := range c.connections {
		if connection.FromID != fromID || connection.ToID != toID {
			newConnections = append(newConnections, connection)
		}
	}
	c.connections = newConnections
}

func (c *Canvas) RemoveSpecificConnection(target Connection) {
	newConnections := make([]Connection, 0)
	for _, connection := range c.connections {
		if !c.connectionsEqual(connection, target) {
			newConnections = append(newConnections, connection)
		}
	}
	c.connections = newConnections
}

func (c *Canvas) CycleConnectionArrowState(connIdx int) {
	if connIdx < 0 || connIdx >= len(c.connections) {
		return
	}
	conn := &c.connections[connIdx]
	if !conn.ArrowFrom && !conn.ArrowTo {
		conn.ArrowTo = true
	} else if !conn.ArrowFrom && conn.ArrowTo {
		conn.ArrowFrom = true
		conn.ArrowTo = false
	} else if conn.ArrowFrom && !conn.ArrowTo {
		conn.ArrowFrom = true
		conn.ArrowTo = true
	} else {
		conn.ArrowFrom = false
		conn.ArrowTo = false
	}
}

func (c *Canvas) connectionsEqual(a, b Connection) bool {
	if a.FromID != b.FromID || a.ToID != b.ToID {
		return false
	}
	if a.FromX != b.FromX || a.FromY != b.FromY || a.ToX != b.ToX || a.ToY != b.ToY {
		return false
	}
	if len(a.Waypoints) != len(b.Waypoints) {
		return false
	}
	for i := range a.Waypoints {
		if a.Waypoints[i].X != b.Waypoints[i].X || a.Waypoints[i].Y != b.Waypoints[i].Y {
			return false
		}
	}
	return true
}

func (c *Canvas) RestoreConnection(connection Connection) {
	c.connections = append(c.connections, connection)
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
		// Check if cursor is within the text area (including all lines)
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
		// Remove the text
		c.texts = append(c.texts[:id], c.texts[id+1:]...)

		// Update IDs for remaining texts
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
		// Remove the box
		c.boxes = append(c.boxes[:id], c.boxes[id+1:]...)

		// Update IDs for remaining boxes
		for i := id; i < len(c.boxes); i++ {
			c.boxes[i].ID = i
		}

		// Remove connections connected to this box and update connection IDs
		newConnections := make([]Connection, 0)
		for _, connection := range c.connections {
			if connection.FromID != id && connection.ToID != id {
				// Update IDs if they're greater than the deleted box ID
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

		// Set minimum size constraints
		minWidth := minBoxWidth
		minHeight := minBoxHeight

		// Calculate new size
		newWidth := box.Width + deltaWidth
		newHeight := box.Height + deltaHeight

		// Apply minimum constraints
		if newWidth < minWidth {
			newWidth = minWidth
		}
		if newHeight < minHeight {
			newHeight = minHeight
		}

		oldBoxX := box.X
		oldBoxWidth := box.Width

		box.Width = newWidth
		box.Height = newHeight

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
}

func (c *Canvas) MoveBox(id int, deltaX, deltaY int) {
	if id >= 0 && id < len(c.boxes) {
		box := &c.boxes[id]
		box.X += deltaX
		box.Y += deltaY

		// Ensure box doesn't go negative
		if box.X < 0 {
			box.X = 0
		}
		if box.Y < 0 {
			box.Y = 0
		}

		for i := range c.connections {
			conn := &c.connections[i]
			if conn.FromID == id && conn.ToID >= 0 && conn.ToID < len(c.boxes) {
				newFromX, newFromY, newToX, newToY := c.calculateConnectionPoints(id, conn.ToID)
				conn.FromX = newFromX
				conn.FromY = newFromY
				conn.ToX = newToX
				conn.ToY = newToY
			}
			if conn.ToID == id && conn.FromID >= 0 && conn.FromID < len(c.boxes) {
				newFromX, newFromY, newToX, newToY := c.calculateConnectionPoints(conn.FromID, id)
				conn.FromX = newFromX
				conn.FromY = newFromY
				conn.ToX = newToX
				conn.ToY = newToY
			}
		}
	}
}

func (c *Canvas) SetBoxPosition(id int, x, y int) {
	if id >= 0 && id < len(c.boxes) {
		box := &c.boxes[id]
		oldX := box.X
		oldY := box.Y
		box.X = x
		box.Y = y

		// Ensure box doesn't go negative
		if box.X < 0 {
			box.X = 0
		}
		if box.Y < 0 {
			box.Y = 0
		}

		deltaX := box.X - oldX
		deltaY := box.Y - oldY
		if deltaX != 0 || deltaY != 0 {
			for i := range c.connections {
				conn := &c.connections[i]
				if conn.FromID == id && conn.ToID >= 0 && conn.ToID < len(c.boxes) {
					newFromX, newFromY, newToX, newToY := c.calculateConnectionPoints(id, conn.ToID)
					conn.FromX = newFromX
					conn.FromY = newFromY
					conn.ToX = newToX
					conn.ToY = newToY
				}
				if conn.ToID == id && conn.FromID >= 0 && conn.FromID < len(c.boxes) {
					newFromX, newFromY, newToX, newToY := c.calculateConnectionPoints(conn.FromID, id)
					conn.FromX = newFromX
					conn.FromY = newFromY
					conn.ToX = newToX
					conn.ToY = newToY
				}
			}
		}
	}
}

func (c *Canvas) MoveText(id int, deltaX, deltaY int) {
	if id >= 0 && id < len(c.texts) {
		text := &c.texts[id]
		text.X += deltaX
		text.Y += deltaY

		// Ensure text doesn't go negative
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
		text.X = x
		text.Y = y

		// Ensure text doesn't go negative
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

		oldBoxX := box.X
		oldBoxWidth := box.Width
		oldWidth := box.Width
		oldHeight := box.Height

		// Set minimum size constraints
		minWidth := minBoxWidth
		minHeight := minBoxHeight

		// Apply minimum constraints
		if width < minWidth {
			width = minWidth
		}
		if height < minHeight {
			height = minHeight
		}

		// Update box size
		box.Width = width
		box.Height = height

		if box.Width != oldWidth || box.Height != oldHeight {
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
	}
}

func (c *Canvas) Render(width, height int, selectedBox int, previewFromX, previewFromY int, previewWaypoints []point, previewToX, previewToY int, panX, panY int) []string {
	// Ensure minimum dimensions
	if height < 1 {
		height = 1
	}
	if width < 1 {
		width = 1
	}

	canvas := make([][]rune, height)
	colorMap := make([][]int, height) // Track color for each cell (-1 = default)
	for i := range canvas {
		canvas[i] = make([]rune, width)
		colorMap[i] = make([]int, width)
		for j := range canvas[i] {
			canvas[i][j] = ' '
			colorMap[i][j] = -1
		}
	}

	// Draw connections first (so they appear behind boxes)
	for _, connection := range c.connections {
		c.drawConnectionWithPan(canvas, connection, panX, panY)
	}

	// Draw preview connection if in progress
	if previewFromX >= 0 && previewFromY >= 0 {
		previewConnection := Connection{
			FromID:    -1,
			ToID:      -1,
			FromX:     previewFromX - panX,
			FromY:     previewFromY - panY,
			ToX:       previewToX - panX,
			ToY:       previewToY - panY,
			Waypoints: make([]point, len(previewWaypoints)),
		}
		for i, wp := range previewWaypoints {
			previewConnection.Waypoints[i] = point{X: wp.X - panX, Y: wp.Y - panY}
		}
		c.drawConnectionWithPan(canvas, previewConnection, panX, panY)
	}

	// Draw texts (plain text without borders)
	for _, text := range c.texts {
		c.drawTextWithPan(canvas, text, panX, panY)
	}

	// Draw boxes last (so they appear on top)
	for i, box := range c.boxes {
		isSelected := (i == selectedBox)
		c.drawBoxWithPan(canvas, box, isSelected, panX, panY)
	}

	// Convert to strings with consistent line lengths
	result := make([]string, height)
	for i, row := range canvas {
		// Ensure each line is exactly the right width
		line := make([]rune, width)
		copy(line, row)
		for j := len(row); j < width; j++ {
			line[j] = ' '
		}
		result[i] = string(line)
	}

	return result
}

func (c *Canvas) drawBoxWithPan(canvas [][]rune, box Box, isSelected bool, panX, panY int) {
	// Apply pan offset to box coordinates
	boxX := box.X - panX
	boxY := box.Y - panY
	c.drawBoxAt(canvas, box, isSelected, boxX, boxY)
}

func (c *Canvas) drawBox(canvas [][]rune, box Box, isSelected bool) {
	c.drawBoxAt(canvas, box, isSelected, box.X, box.Y)
}

func (c *Canvas) drawBoxAt(canvas [][]rune, box Box, isSelected bool, boxX, boxY int) {
	// Choose border characters based on selection state
	var corner, horizontal, vertical rune
	if isSelected {
		corner = '#'
		horizontal = '#'
		vertical = '#'
	} else {
		corner = '+'
		horizontal = '-'
		vertical = '|'
	}

	// Draw box borders with bounds checking
	for y := boxY; y < boxY+box.Height && y < len(canvas) && y >= 0; y++ {
		if y >= len(canvas) {
			break
		}
		for x := boxX; x < boxX+box.Width && x < len(canvas[y]) && x >= 0; x++ {
			if y == boxY || y == boxY+box.Height-1 {
				// Top and bottom borders
				if x == boxX || x == boxX+box.Width-1 {
					// Corners
					canvas[y][x] = corner
				} else {
					canvas[y][x] = horizontal
				}
			} else if x == boxX || x == boxX+box.Width-1 {
				// Left and right borders
				canvas[y][x] = vertical
			}
		}
	}

	// Draw multi-line text inside box with bounds checking
	for lineIdx, line := range box.Lines {
		textY := boxY + 1 + lineIdx
		textX := boxX + 1

		if textY >= 0 && textY < len(canvas) && textY < boxY+box.Height-1 && textX >= 0 {
			// Truncate line if it's too long for the box
			displayText := line
			maxWidth := box.Width - 2
			if maxWidth < 0 {
				maxWidth = 0
			}
			if len(displayText) > maxWidth {
				displayText = displayText[:maxWidth]
			}

			for i, char := range displayText {
				if textX+i >= 0 && textX+i < len(canvas[textY]) && textX+i < boxX+box.Width-1 {
					canvas[textY][textX+i] = char
				}
			}
		}
	}
}

func (c *Canvas) drawTextWithPan(canvas [][]rune, text Text, panX, panY int) {
	// Apply pan offset to text coordinates
	textX := text.X - panX
	textY := text.Y - panY
	c.drawTextAt(canvas, text, textX, textY)
}

func (c *Canvas) drawText(canvas [][]rune, text Text) {
	c.drawTextAt(canvas, text, text.X, text.Y)
}

func (c *Canvas) drawTextAt(canvas [][]rune, text Text, textX, textY int) {
	// Draw multi-line text directly without borders
	for lineIdx, line := range text.Lines {
		lineY := textY + lineIdx
		lineX := textX

		if lineY >= 0 && lineY < len(canvas) && lineX >= 0 {
			for i, char := range line {
				if lineX+i >= 0 && lineX+i < len(canvas[lineY]) {
					canvas[lineY][lineX+i] = char
				}
			}
		}
	}
}

func (c *Canvas) isPointInBox(x, y int, excludeFromID, excludeToID int) bool {
	for i, box := range c.boxes {
		if i == excludeFromID || i == excludeToID {
			if x > box.X && x < box.X+box.Width-1 && y > box.Y && y < box.Y+box.Height-1 {
				return true
			}
		} else {
			if x > box.X && x < box.X+box.Width-1 && y > box.Y && y < box.Y+box.Height-1 {
				return true
			}
		}
	}
	return false
}

func (c *Canvas) drawLineSegment(canvas [][]rune, fromX, fromY, toX, toY int, excludeFromID, excludeToID int, drawArrowFrom, drawArrowTo bool, skipCorner bool, originalConnection Connection, panX, panY int) {
	if fromX == toX && fromY == toY {
		return
	}

	if fromX == toX {
		var lineStartY, lineEndY int
		var arrowY int
		var arrowChar rune

		if fromY < toY {
			lineStartY = fromY + 1
			lineEndY = toY - 1
			arrowY = toY - 1
			arrowChar = '▼'
		} else {
			lineStartY = toY + 1
			lineEndY = fromY - 1
			arrowY = toY + 1
			arrowChar = '▲'
		}

		for y := lineStartY; y <= lineEndY; y++ {
			if c.isValidPos(canvas, fromX, y) && !c.isPointInBox(fromX, y, excludeFromID, excludeToID) {
				canvas[y][fromX] = '│'
			}
		}

		if drawArrowTo && c.isValidPos(canvas, toX, arrowY) {
			canvas[arrowY][toX] = arrowChar
		}
		if drawArrowFrom {
			var fromArrowY int
			var fromArrowChar rune
			if excludeFromID >= 0 && excludeFromID < len(c.boxes) {
				fromBox := c.boxes[excludeFromID]
				fromOnTopEdge := (fromY == fromBox.Y)
				fromOnBottomEdge := (fromY == fromBox.Y+fromBox.Height-1)
				if !fromOnTopEdge && !fromOnBottomEdge {
					if abs(fromY-fromBox.Y) < abs(fromY-(fromBox.Y+fromBox.Height-1)) {
						fromOnTopEdge = true
					} else {
						fromOnBottomEdge = true
					}
				}
				if fromOnTopEdge {
					fromArrowY = fromBox.Y - 1
					fromArrowChar = '▼'
				} else if fromOnBottomEdge {
					fromArrowY = fromBox.Y + fromBox.Height
					fromArrowChar = '▲'
				} else {
					if fromY < toY {
						fromArrowY = fromY + 1
						fromArrowChar = '▲'
					} else {
						fromArrowY = fromY - 1
						fromArrowChar = '▼'
					}
				}
			} else {
				if fromY < toY {
					fromArrowY = fromY + 1
					fromArrowChar = '▲'
				} else {
					fromArrowY = fromY - 1
					fromArrowChar = '▼'
				}
			}
			if c.isValidPos(canvas, fromX, fromArrowY) && !c.isPointInBox(fromX, fromArrowY, excludeFromID, excludeToID) {
				canvas[fromArrowY][fromX] = fromArrowChar
			}
		}

	} else if fromY == toY {
		var lineStartX, lineEndX int
		var arrowX int
		var arrowChar rune

		var onLeftEdge, onRightEdge bool
		if excludeToID >= 0 && excludeToID < len(c.boxes) {
			toBox := c.boxes[excludeToID]
			onLeftEdge = (toX == toBox.X)
			onRightEdge = (toX == toBox.X+toBox.Width-1)
			if !onLeftEdge && !onRightEdge {
				if abs(toX-toBox.X) < abs(toX-(toBox.X+toBox.Width-1)) {
					onLeftEdge = true
				} else {
					onRightEdge = true
				}
			}
		}

		if onLeftEdge {
			arrowX = toX - 1
			arrowChar = '▶'
			if fromX < toX {
				lineStartX = fromX + 1
				lineEndX = toX - 1
			} else {
				lineStartX = toX - 1
				lineEndX = fromX - 1
			}
		} else if onRightEdge {
			arrowX = toX + 1
			arrowChar = '◀'
			if fromX < toX {
				lineStartX = fromX + 1
				lineEndX = toX + 1
			} else {
				lineStartX = toX + 1
				lineEndX = fromX - 1
			}
		} else {
			if fromX < toX {
				lineStartX = fromX + 1
				lineEndX = toX - 1
				arrowX = toX - 1
				arrowChar = '▶'
			} else {
				lineStartX = toX + 1
				lineEndX = fromX - 1
				arrowX = toX + 1
				arrowChar = '◀'
			}
		}

		for x := lineStartX; x <= lineEndX; x++ {
			if c.isValidPos(canvas, x, fromY) && !c.isPointInBox(x, fromY, excludeFromID, excludeToID) {
				canvas[fromY][x] = '─'
			}
		}

		if drawArrowTo {
			if excludeToID >= 0 && excludeToID < len(c.boxes) {
				toBox := c.boxes[excludeToID]
				if arrowX >= toBox.X && arrowX < toBox.X+toBox.Width {
					if onLeftEdge {
						arrowX = toBox.X - 1
					} else if onRightEdge {
						arrowX = toBox.X + toBox.Width
					}
				}
			}
			if c.isValidPos(canvas, arrowX, toY) {
				if !c.isPointInBox(arrowX, toY, excludeFromID, excludeToID) {
					canvas[toY][arrowX] = arrowChar
				}
			}
		}
		if drawArrowFrom {
			var fromArrowX int
			var fromArrowChar rune
			if excludeFromID >= 0 && excludeFromID < len(c.boxes) {
				fromBox := c.boxes[excludeFromID]
				fromOnLeftEdge := (fromX == fromBox.X)
				fromOnRightEdge := (fromX == fromBox.X+fromBox.Width-1)
				if !fromOnLeftEdge && !fromOnRightEdge {
					if abs(fromX-fromBox.X) < abs(fromX-(fromBox.X+fromBox.Width-1)) {
						fromOnLeftEdge = true
					} else {
						fromOnRightEdge = true
					}
				}
				if fromOnLeftEdge {
					fromArrowX = fromBox.X - 1
					fromArrowChar = '▶'
				} else if fromOnRightEdge {
					fromArrowX = fromBox.X + fromBox.Width
					fromArrowChar = '◀'
				} else {
					if fromX < toX {
						fromArrowX = fromX - 1
						fromArrowChar = '▶'
					} else {
						fromArrowX = fromX + 1
						fromArrowChar = '◀'
					}
				}
			} else {
				if fromX < toX {
					fromArrowX = fromX - 1
					fromArrowChar = '▶'
				} else {
					fromArrowX = fromX + 1
					fromArrowChar = '◀'
				}
			}
			if c.isValidPos(canvas, fromArrowX, fromY) {
				if !c.isPointInBox(fromArrowX, fromY, excludeFromID, excludeToID) {
					canvas[fromY][fromArrowX] = fromArrowChar
				}
			}
		}

	} else {
		cornerX := toX
		cornerY := fromY

		var hStartX, hEndX int
		if fromX < cornerX {
			hStartX = fromX + 1
			hEndX = cornerX - 1
		} else {
			hStartX = cornerX + 1
			hEndX = fromX - 1
		}

		if hStartX <= hEndX {
			for x := hStartX; x <= hEndX; x++ {
				if c.isValidPos(canvas, x, fromY) && !c.isPointInBox(x, fromY, excludeFromID, excludeToID) {
					canvas[fromY][x] = '─'
				}
			}
		}

		var vStartY, vEndY int
		if cornerY < toY {
			vStartY = cornerY + 1
			vEndY = toY - 1
		} else {
			vStartY = toY + 1
			vEndY = cornerY - 1
		}

		if vStartY <= vEndY {
			for y := vStartY; y <= vEndY; y++ {
				if c.isValidPos(canvas, cornerX, y) && !c.isPointInBox(cornerX, y, excludeFromID, excludeToID) {
					canvas[y][cornerX] = '│'
				}
			}
		}

		if !skipCorner && c.isValidPos(canvas, cornerX, cornerY) && !c.isPointInBox(cornerX, cornerY, excludeFromID, excludeToID) {
			if fromX < toX && fromY < toY {
				canvas[cornerY][cornerX] = '┐'
			} else if fromX < toX && fromY > toY {
				canvas[cornerY][cornerX] = '┘'
			} else if fromX > toX && fromY < toY {
				canvas[cornerY][cornerX] = '┌'
			} else if fromX > toX && fromY > toY {
				canvas[cornerY][cornerX] = '└'
			}
		}

		if drawArrowTo {
			var arrowX, arrowY int
			var arrowChar rune
			// Use original world coordinates for calculations
			origToX := originalConnection.ToX
			origToY := originalConnection.ToY

			if excludeToID >= 0 && excludeToID < len(c.boxes) {
				toBox := c.boxes[excludeToID]
				onLeftEdge := (origToX == toBox.X)
				onRightEdge := (origToX == toBox.X+toBox.Width-1)
				onTopEdge := (origToY == toBox.Y)
				onBottomEdge := (origToY == toBox.Y+toBox.Height-1)

				if !onLeftEdge && !onRightEdge && !onTopEdge && !onBottomEdge {
					if abs(origToX-toBox.X) <= abs(origToX-(toBox.X+toBox.Width-1)) && abs(origToX-toBox.X) <= abs(origToY-toBox.Y) && abs(origToX-toBox.X) <= abs(origToY-(toBox.Y+toBox.Height-1)) {
						onLeftEdge = true
					} else if abs(origToX-(toBox.X+toBox.Width-1)) <= abs(origToY-toBox.Y) && abs(origToX-(toBox.X+toBox.Width-1)) <= abs(origToY-(toBox.Y+toBox.Height-1)) {
						onRightEdge = true
					} else if abs(origToY-toBox.Y) <= abs(origToY-(toBox.Y+toBox.Height-1)) {
						onTopEdge = true
					} else {
						onBottomEdge = true
					}
				}

				if onLeftEdge {
					arrowX = (toBox.X - 1) - panX
					arrowY = origToY - panY
					arrowChar = '▶'
				} else if onRightEdge {
					arrowX = (toBox.X + toBox.Width) - panX
					arrowY = origToY - panY
					arrowChar = '◀'
				} else if onTopEdge {
					arrowX = origToX - panX
					arrowY = (toBox.Y - 1) - panY
					arrowChar = '▲'
				} else if onBottomEdge {
					arrowX = origToX - panX
					arrowY = (toBox.Y + toBox.Height) - panY
					arrowChar = '▼'
				} else {
					if cornerY < toY {
						arrowX = toX
						arrowY = toY - 1
						arrowChar = '▼'
					} else if cornerY > toY {
						arrowX = toX
						arrowY = toY + 1
						arrowChar = '▲'
					} else {
						if origToX < toBox.X+toBox.Width/2 {
							arrowX = (toBox.X - 1) - panX
							arrowY = origToY - panY
							arrowChar = '▶'
						} else {
							arrowX = (toBox.X + toBox.Width) - panX
							arrowY = origToY - panY
							arrowChar = '◀'
						}
					}
				}
			} else {
				if cornerY < toY {
					arrowX = toX
					arrowY = toY - 1
					arrowChar = '▼'
				} else if cornerY > toY {
					arrowX = toX
					arrowY = toY + 1
					arrowChar = '▲'
				} else {
					if fromX < toX {
						arrowX = toX - 1
						arrowY = toY
						arrowChar = '▶'
					} else {
						arrowX = toX + 1
						arrowY = toY
						arrowChar = '◀'
					}
				}
			}

			if arrowY == toY && arrowX != cornerX {
				var hSegStartX, hSegEndX int
				if cornerX < arrowX {
					hSegStartX = cornerX + 1
					hSegEndX = arrowX - 1
				} else {
					hSegStartX = arrowX + 1
					hSegEndX = cornerX - 1
				}

				for x := hSegStartX; x <= hSegEndX; x++ {
					if c.isValidPos(canvas, x, arrowY) && !c.isPointInBox(x, arrowY, excludeFromID, excludeToID) {
						canvas[arrowY][x] = '─'
					}
				}
			}

			if c.isValidPos(canvas, arrowX, arrowY) && !c.isPointInBox(arrowX, arrowY, excludeFromID, excludeToID) {
				canvas[arrowY][arrowX] = arrowChar
			}
		}
		if drawArrowFrom {
			var fromArrowX, fromArrowY int
			var fromArrowChar rune
			// Use original world coordinates for calculations
			origFromX := originalConnection.FromX
			origFromY := originalConnection.FromY
			if excludeFromID >= 0 && excludeFromID < len(c.boxes) {
				fromBox := c.boxes[excludeFromID]
				fromOnLeftEdge := (origFromX == fromBox.X)
				fromOnRightEdge := (origFromX == fromBox.X+fromBox.Width-1)
				fromOnTopEdge := (origFromY == fromBox.Y)
				fromOnBottomEdge := (origFromY == fromBox.Y+fromBox.Height-1)
				if !fromOnLeftEdge && !fromOnRightEdge {
					if abs(origFromX-fromBox.X) < abs(origFromX-(fromBox.X+fromBox.Width-1)) {
						fromOnLeftEdge = true
					} else {
						fromOnRightEdge = true
					}
				}
				if !fromOnTopEdge && !fromOnBottomEdge {
					if abs(origFromY-fromBox.Y) < abs(origFromY-(fromBox.Y+fromBox.Height-1)) {
						fromOnTopEdge = true
					} else {
						fromOnBottomEdge = true
					}
				}
				if fromOnTopEdge {
					fromArrowX = origFromX - panX
					fromArrowY = (fromBox.Y - 1) - panY
					fromArrowChar = '▼'
				} else if fromOnBottomEdge {
					fromArrowX = origFromX - panX
					fromArrowY = (fromBox.Y + fromBox.Height) - panY
					fromArrowChar = '▲'
				} else if fromOnLeftEdge {
					fromArrowX = (fromBox.X - 1) - panX
					fromArrowY = origFromY - panY
					fromArrowChar = '▶'
				} else if fromOnRightEdge {
					fromArrowX = (fromBox.X + fromBox.Width) - panX
					fromArrowY = origFromY - panY
					fromArrowChar = '◀'
				} else {
					if origFromX == cornerX+panX {
						if origFromY < cornerY+panY {
							fromArrowX = fromX
							fromArrowY = fromY - 1
							fromArrowChar = '▼'
						} else {
							fromArrowX = fromX
							fromArrowY = fromY + 1
							fromArrowChar = '▲'
						}
					} else if origFromY == cornerY+panY {
						if origFromX < cornerX+panX {
							fromArrowX = fromX - 1
							fromArrowY = fromY
							fromArrowChar = '▶'
						} else {
							fromArrowX = fromX + 1
							fromArrowY = fromY
							fromArrowChar = '◀'
						}
					}
				}
			} else {
				if fromX == cornerX {
					if fromY < cornerY {
						fromArrowX = fromX
						fromArrowY = fromY - 1
						fromArrowChar = '▼'
					} else {
						fromArrowX = fromX
						fromArrowY = fromY + 1
						fromArrowChar = '▲'
					}
				} else if fromY == cornerY {
					if fromX < cornerX {
						fromArrowX = fromX - 1
						fromArrowY = fromY
						fromArrowChar = '▶'
					} else {
						fromArrowX = fromX + 1
						fromArrowY = fromY
						fromArrowChar = '◀'
					}
				}
			}
			if c.isValidPos(canvas, fromArrowX, fromArrowY) && !c.isPointInBox(fromArrowX, fromArrowY, excludeFromID, excludeToID) {
				canvas[fromArrowY][fromArrowX] = fromArrowChar
			}
		}
	}
}

func (c *Canvas) drawCorner(canvas [][]rune, cornerX, cornerY int, prevX, prevY, nextX, nextY int, excludeFromID, excludeToID int) {
	if !c.isValidPos(canvas, cornerX, cornerY) {
		return
	}
	if c.isPointInBox(cornerX, cornerY, excludeFromID, excludeToID) {
		return
	}

	var cornerChar rune

	if prevX != cornerX && nextY != cornerY {
		if prevX < cornerX && cornerY < nextY {
			cornerChar = '┐'
		} else if prevX < cornerX && cornerY > nextY {
			cornerChar = '┘'
		} else if prevX > cornerX && cornerY < nextY {
			cornerChar = '┌'
		} else {
			cornerChar = '└'
		}
	} else if prevY != cornerY && nextX != cornerX {
		if prevY < cornerY && cornerX < nextX {
			cornerChar = '└'
		} else if prevY < cornerY && cornerX > nextX {
			cornerChar = '┘'
		} else if prevY > cornerY && cornerX < nextX {
			cornerChar = '┌'
		} else {
			cornerChar = '┐'
		}
	} else {
		return
	}

	canvas[cornerY][cornerX] = cornerChar
}

func (c *Canvas) drawConnectionWithPan(canvas [][]rune, connection Connection, panX, panY int) {
	// Apply pan offset to connection coordinates for drawing
	fromX := connection.FromX - panX
	fromY := connection.FromY - panY
	toX := connection.ToX - panX
	toY := connection.ToY - panY
	
	// Create a copy of connection with adjusted screen coordinates for drawing
	adjustedConnection := connection
	adjustedConnection.FromX = fromX
	adjustedConnection.FromY = fromY
	adjustedConnection.ToX = toX
	adjustedConnection.ToY = toY
	adjustedConnection.Waypoints = make([]point, len(connection.Waypoints))
	for i, wp := range connection.Waypoints {
		adjustedConnection.Waypoints[i] = point{X: wp.X - panX, Y: wp.Y - panY}
	}
	// Pass original connection for arrow calculations (uses world coordinates)
	c.drawConnection(canvas, adjustedConnection, connection, panX, panY)
}

func (c *Canvas) drawConnection(canvas [][]rune, connection Connection, originalConnection Connection, panX, panY int) {
	fromX, fromY := connection.FromX, connection.FromY
	toX, toY := connection.ToX, connection.ToY
	// Use original world coordinates for arrow calculations
	origFromX, origFromY := originalConnection.FromX, originalConnection.FromY
	origToX, origToY := originalConnection.ToX, originalConnection.ToY

	if len(connection.Waypoints) > 0 {
		points := make([]point, 0, len(connection.Waypoints)+2)
		points = append(points, point{fromX, fromY})
		points = append(points, connection.Waypoints...)
		points = append(points, point{toX, toY})

		for i := 0; i < len(points)-1; i++ {
			prevPoint := points[i]
			nextPoint := points[i+1]

			drawFromArrow := (i == 0 && connection.ArrowFrom)
			drawToArrow := (i == len(points)-2 && connection.ArrowTo)

			if prevPoint.X == nextPoint.X {
				startY := prevPoint.Y
				endY := nextPoint.Y
				if startY > endY {
					startY, endY = endY, startY
				}
				for y := startY + 1; y < endY; y++ {
					if c.isValidPos(canvas, prevPoint.X, y) && !c.isPointInBox(prevPoint.X, y, connection.FromID, connection.ToID) {
						canvas[y][prevPoint.X] = '│'
					}
				}
				// Draw a corner at prevPoint if the previous segment was horizontal
				if i > 0 {
					prevPrev := points[i-1]
					prevWasHorizontal := prevPrev.Y == prevPoint.Y
					if prevWasHorizontal {
						cornerX, cornerY := prevPoint.X, prevPoint.Y
						var cornerChar rune
						if prevPrev.X < prevPoint.X && nextPoint.Y > prevPoint.Y {
							cornerChar = '┐'
						} else if prevPrev.X < prevPoint.X && nextPoint.Y < prevPoint.Y {
							cornerChar = '┘'
						} else if prevPrev.X > prevPoint.X && nextPoint.Y > prevPoint.Y {
							cornerChar = '┌'
						} else {
							cornerChar = '└'
						}
						if c.isValidPos(canvas, cornerX, cornerY) && !c.isPointInBox(cornerX, cornerY, connection.FromID, connection.ToID) {
							canvas[cornerY][cornerX] = cornerChar
						}
					}
				}
				if drawFromArrow && i == 0 {
					var fromArrowX, fromArrowY int
					var fromArrowChar rune
					if connection.FromID >= 0 && connection.FromID < len(c.boxes) {
						fromBox := c.boxes[connection.FromID]
						// Use original world coordinates for calculations
						if abs(origFromX-fromBox.X) <= abs(origFromX-(fromBox.X+fromBox.Width-1)) && abs(origFromX-fromBox.X) <= abs(origFromY-fromBox.Y) && abs(origFromX-fromBox.X) <= abs(origFromY-(fromBox.Y+fromBox.Height-1)) {
							fromArrowX = (fromBox.X - 1) - panX
							fromArrowY = origFromY - panY
							fromArrowChar = '▶'
						} else if abs(origFromX-(fromBox.X+fromBox.Width-1)) <= abs(origFromY-fromBox.Y) && abs(origFromX-(fromBox.X+fromBox.Width-1)) <= abs(origFromY-(fromBox.Y+fromBox.Height-1)) {
							fromArrowX = (fromBox.X + fromBox.Width) - panX
							fromArrowY = origFromY - panY
							fromArrowChar = '◀'
						} else if abs(origFromY-fromBox.Y) <= abs(origFromY-(fromBox.Y+fromBox.Height-1)) {
							fromArrowX = origFromX - panX
							fromArrowY = (fromBox.Y - 1) - panY
							fromArrowChar = '▼'
						} else {
							fromArrowX = origFromX - panX
							fromArrowY = (fromBox.Y + fromBox.Height) - panY
							fromArrowChar = '▲'
						}
					}
					if c.isValidPos(canvas, fromArrowX, fromArrowY) && !c.isPointInBox(fromArrowX, fromArrowY, connection.FromID, connection.ToID) {
						canvas[fromArrowY][fromArrowX] = fromArrowChar
					}
				}
				if drawToArrow && i == len(points)-2 {
					var toArrowX, toArrowY int
					var toArrowChar rune
					if connection.ToID >= 0 && connection.ToID < len(c.boxes) {
						toBox := c.boxes[connection.ToID]
						// Use original world coordinates for calculations
						if abs(origToX-toBox.X) <= abs(origToX-(toBox.X+toBox.Width-1)) && abs(origToX-toBox.X) <= abs(origToY-toBox.Y) && abs(origToX-toBox.X) <= abs(origToY-(toBox.Y+toBox.Height-1)) {
							toArrowX = (toBox.X - 1) - panX
							toArrowY = origToY - panY
							toArrowChar = '▶'
						} else if abs(origToX-(toBox.X+toBox.Width-1)) <= abs(origToY-toBox.Y) && abs(origToX-(toBox.X+toBox.Width-1)) <= abs(origToY-(toBox.Y+toBox.Height-1)) {
							toArrowX = (toBox.X + toBox.Width) - panX
							toArrowY = origToY - panY
							toArrowChar = '◀'
						} else if abs(origToY-toBox.Y) <= abs(origToY-(toBox.Y+toBox.Height-1)) {
							toArrowX = origToX - panX
							toArrowY = (toBox.Y - 1) - panY
							toArrowChar = '▼'
						} else {
							toArrowX = origToX - panX
							toArrowY = (toBox.Y + toBox.Height) - panY
							toArrowChar = '▲'
						}
					}
					if c.isValidPos(canvas, toArrowX, toArrowY) && !c.isPointInBox(toArrowX, toArrowY, connection.FromID, connection.ToID) {
						canvas[toArrowY][toArrowX] = toArrowChar
					}
				}
			} else if prevPoint.Y == nextPoint.Y {
				startX := prevPoint.X
				endX := nextPoint.X
				if startX > endX {
					startX, endX = endX, startX
				}
				for x := startX + 1; x < endX; x++ {
					if c.isValidPos(canvas, x, prevPoint.Y) && !c.isPointInBox(x, prevPoint.Y, connection.FromID, connection.ToID) {
						canvas[prevPoint.Y][x] = '─'
					}
				}
				// Draw a corner at prevPoint if the previous segment was vertical
				if i > 0 {
					prevPrev := points[i-1]
					prevWasVertical := prevPrev.X == prevPoint.X
					if prevWasVertical {
						cornerX, cornerY := prevPoint.X, prevPoint.Y
						var cornerChar rune
						if prevPrev.Y < prevPoint.Y && nextPoint.X > prevPoint.X {
							cornerChar = '└'
						} else if prevPrev.Y < prevPoint.Y && nextPoint.X < prevPoint.X {
							cornerChar = '┘'
						} else if prevPrev.Y > prevPoint.Y && nextPoint.X > prevPoint.X {
							cornerChar = '┌'
						} else {
							cornerChar = '┐'
						}
						if c.isValidPos(canvas, cornerX, cornerY) && !c.isPointInBox(cornerX, cornerY, connection.FromID, connection.ToID) {
							canvas[cornerY][cornerX] = cornerChar
						}
					}
				}
				if drawFromArrow && i == 0 {
					var fromArrowX, fromArrowY int
					var fromArrowChar rune
					if connection.FromID >= 0 && connection.FromID < len(c.boxes) {
						fromBox := c.boxes[connection.FromID]
						// Use original world coordinates for calculations
						if abs(origFromX-fromBox.X) <= abs(origFromX-(fromBox.X+fromBox.Width-1)) && abs(origFromX-fromBox.X) <= abs(origFromY-fromBox.Y) && abs(origFromX-fromBox.X) <= abs(origFromY-(fromBox.Y+fromBox.Height-1)) {
							fromArrowX = (fromBox.X - 1) - panX
							fromArrowY = origFromY - panY
							fromArrowChar = '▶'
						} else if abs(origFromX-(fromBox.X+fromBox.Width-1)) <= abs(origFromY-fromBox.Y) && abs(origFromX-(fromBox.X+fromBox.Width-1)) <= abs(origFromY-(fromBox.Y+fromBox.Height-1)) {
							fromArrowX = (fromBox.X + fromBox.Width) - panX
							fromArrowY = origFromY - panY
							fromArrowChar = '◀'
						} else if abs(origFromY-fromBox.Y) <= abs(origFromY-(fromBox.Y+fromBox.Height-1)) {
							fromArrowX = origFromX - panX
							fromArrowY = (fromBox.Y - 1) - panY
							fromArrowChar = '▼'
						} else {
							fromArrowX = origFromX - panX
							fromArrowY = (fromBox.Y + fromBox.Height) - panY
							fromArrowChar = '▲'
						}
					}
					if c.isValidPos(canvas, fromArrowX, fromArrowY) && !c.isPointInBox(fromArrowX, fromArrowY, connection.FromID, connection.ToID) {
						canvas[fromArrowY][fromArrowX] = fromArrowChar
					}
				}
				if drawToArrow && i == len(points)-2 {
					var toArrowX, toArrowY int
					var toArrowChar rune
					if connection.ToID >= 0 && connection.ToID < len(c.boxes) {
						toBox := c.boxes[connection.ToID]
						// Use original world coordinates for calculations
						if abs(origToX-toBox.X) <= abs(origToX-(toBox.X+toBox.Width-1)) && abs(origToX-toBox.X) <= abs(origToY-toBox.Y) && abs(origToX-toBox.X) <= abs(origToY-(toBox.Y+toBox.Height-1)) {
							toArrowX = (toBox.X - 1) - panX
							toArrowY = origToY - panY
							toArrowChar = '▶'
						} else if abs(origToX-(toBox.X+toBox.Width-1)) <= abs(origToY-toBox.Y) && abs(origToX-(toBox.X+toBox.Width-1)) <= abs(origToY-(toBox.Y+toBox.Height-1)) {
							toArrowX = (toBox.X + toBox.Width) - panX
							toArrowY = origToY - panY
							toArrowChar = '◀'
						} else if abs(origToY-toBox.Y) <= abs(origToY-(toBox.Y+toBox.Height-1)) {
							toArrowX = origToX - panX
							toArrowY = (toBox.Y - 1) - panY
							toArrowChar = '▼'
						} else {
							toArrowX = origToX - panX
							toArrowY = (toBox.Y + toBox.Height) - panY
							toArrowChar = '▲'
						}
					}
					if c.isValidPos(canvas, toArrowX, toArrowY) && !c.isPointInBox(toArrowX, toArrowY, connection.FromID, connection.ToID) {
						canvas[toArrowY][toArrowX] = toArrowChar
					}
				}
			} else {
				// Determine whether to turn horizontal-then-vertical or vertical-then-horizontal
				firstHorizontal := true
				if i > 0 {
					prevPrev := points[i-1]
					if prevPrev.X == prevPoint.X {
						firstHorizontal = false
					} else if prevPrev.Y == prevPoint.Y {
						firstHorizontal = true
					}
				} else if i+2 < len(points) {
					nextNext := points[i+2]
					if nextPoint.X == nextNext.X {
						firstHorizontal = true
					} else if nextPoint.Y == nextNext.Y {
						firstHorizontal = false
					}
				}

				var cornerX, cornerY int
				if firstHorizontal {
					// Horizontal from prev -> corner, then vertical corner -> next
					cornerX = nextPoint.X
					cornerY = prevPoint.Y

					if prevPoint.X < cornerX {
						for x := prevPoint.X + 1; x < cornerX; x++ {
							if c.isValidPos(canvas, x, prevPoint.Y) && !c.isPointInBox(x, prevPoint.Y, connection.FromID, connection.ToID) {
								canvas[prevPoint.Y][x] = '─'
							}
						}
					} else if prevPoint.X > cornerX {
						for x := cornerX + 1; x < prevPoint.X; x++ {
							if c.isValidPos(canvas, x, prevPoint.Y) && !c.isPointInBox(x, prevPoint.Y, connection.FromID, connection.ToID) {
								canvas[prevPoint.Y][x] = '─'
							}
						}
					}

					if cornerY < nextPoint.Y {
						for y := cornerY + 1; y < nextPoint.Y; y++ {
							if c.isValidPos(canvas, cornerX, y) && !c.isPointInBox(cornerX, y, connection.FromID, connection.ToID) {
								canvas[y][cornerX] = '│'
							}
						}
					} else if cornerY > nextPoint.Y {
						for y := nextPoint.Y + 1; y < cornerY; y++ {
							if c.isValidPos(canvas, cornerX, y) && !c.isPointInBox(cornerX, y, connection.FromID, connection.ToID) {
								canvas[y][cornerX] = '│'
							}
						}
					}

					var cornerChar rune
					if prevPoint.X < cornerX && cornerY < nextPoint.Y {
						cornerChar = '┐'
					} else if prevPoint.X < cornerX && cornerY > nextPoint.Y {
						cornerChar = '┘'
					} else if prevPoint.X > cornerX && cornerY < nextPoint.Y {
						cornerChar = '┌'
					} else {
						cornerChar = '└'
					}
					if c.isValidPos(canvas, cornerX, cornerY) {
						canvas[cornerY][cornerX] = cornerChar
					}
				} else {
					// Vertical from prev -> corner, then horizontal corner -> next
					cornerX = prevPoint.X
					cornerY = nextPoint.Y

					if prevPoint.Y < cornerY {
						for y := prevPoint.Y + 1; y < cornerY; y++ {
							if c.isValidPos(canvas, cornerX, y) && !c.isPointInBox(cornerX, y, connection.FromID, connection.ToID) {
								canvas[y][cornerX] = '│'
							}
						}
					} else if prevPoint.Y > cornerY {
						for y := cornerY + 1; y < prevPoint.Y; y++ {
							if c.isValidPos(canvas, cornerX, y) && !c.isPointInBox(cornerX, y, connection.FromID, connection.ToID) {
								canvas[y][cornerX] = '│'
							}
						}
					}

					if cornerX < nextPoint.X {
						for x := cornerX + 1; x < nextPoint.X; x++ {
							if c.isValidPos(canvas, x, cornerY) && !c.isPointInBox(x, cornerY, connection.FromID, connection.ToID) {
								canvas[cornerY][x] = '─'
							}
						}
					} else if cornerX > nextPoint.X {
						for x := nextPoint.X + 1; x < cornerX; x++ {
							if c.isValidPos(canvas, x, cornerY) && !c.isPointInBox(x, cornerY, connection.FromID, connection.ToID) {
								canvas[cornerY][x] = '─'
							}
						}
					}

					var cornerChar rune
					if prevPoint.Y < cornerY && cornerX < nextPoint.X {
						cornerChar = '└'
					} else if prevPoint.Y < cornerY && cornerX > nextPoint.X {
						cornerChar = '┘'
					} else if prevPoint.Y > cornerY && cornerX < nextPoint.X {
						cornerChar = '┌'
					} else {
						cornerChar = '┐'
					}
					if c.isValidPos(canvas, cornerX, cornerY) {
						canvas[cornerY][cornerX] = cornerChar
					}
				}

				if drawFromArrow && i == 0 {
					var fromArrowX, fromArrowY int
					var fromArrowChar rune
					if connection.FromID >= 0 && connection.FromID < len(c.boxes) {
						fromBox := c.boxes[connection.FromID]
						fromX, fromY := connection.FromX, connection.FromY
						if abs(fromX-fromBox.X) <= abs(fromX-(fromBox.X+fromBox.Width-1)) && abs(fromX-fromBox.X) <= abs(fromY-fromBox.Y) && abs(fromX-fromBox.X) <= abs(fromY-(fromBox.Y+fromBox.Height-1)) {
							fromArrowX = fromBox.X - 1
							fromArrowY = fromY
							fromArrowChar = '▶'
						} else if abs(fromX-(fromBox.X+fromBox.Width-1)) <= abs(fromY-fromBox.Y) && abs(fromX-(fromBox.X+fromBox.Width-1)) <= abs(fromY-(fromBox.Y+fromBox.Height-1)) {
							fromArrowX = fromBox.X + fromBox.Width
							fromArrowY = fromY
							fromArrowChar = '◀'
						} else if abs(fromY-fromBox.Y) <= abs(fromY-(fromBox.Y+fromBox.Height-1)) {
							fromArrowX = fromX
							fromArrowY = fromBox.Y - 1
							fromArrowChar = '▼'
						} else {
							fromArrowX = fromX
							fromArrowY = fromBox.Y + fromBox.Height
							fromArrowChar = '▲'
						}
					}
					if c.isValidPos(canvas, fromArrowX, fromArrowY) && !c.isPointInBox(fromArrowX, fromArrowY, connection.FromID, connection.ToID) {
						canvas[fromArrowY][fromArrowX] = fromArrowChar
					}
				}

				if drawToArrow && i == len(points)-2 {
					var toArrowX, toArrowY int
					var toArrowChar rune
					if connection.ToID >= 0 && connection.ToID < len(c.boxes) {
						toBox := c.boxes[connection.ToID]
						toX, toY := connection.ToX, connection.ToY
						if abs(toX-toBox.X) <= abs(toX-(toBox.X+toBox.Width-1)) && abs(toX-toBox.X) <= abs(toY-toBox.Y) && abs(toX-toBox.X) <= abs(toY-(toBox.Y+toBox.Height-1)) {
							toArrowX = toBox.X - 1
							toArrowY = toY
							toArrowChar = '▶'
						} else if abs(toX-(toBox.X+toBox.Width-1)) <= abs(toY-toBox.Y) && abs(toX-(toBox.X+toBox.Width-1)) <= abs(toY-(toBox.Y+toBox.Height-1)) {
							toArrowX = toBox.X + toBox.Width
							toArrowY = toY
							toArrowChar = '◀'
						} else if abs(toY-toBox.Y) <= abs(toY-(toBox.Y+toBox.Height-1)) {
							toArrowX = toX
							toArrowY = toBox.Y - 1
							toArrowChar = '▼'
						} else {
							toArrowX = toX
							toArrowY = toBox.Y + toBox.Height
							toArrowChar = '▲'
						}
					}
					if c.isValidPos(canvas, toArrowX, toArrowY) && !c.isPointInBox(toArrowX, toArrowY, connection.FromID, connection.ToID) {
						canvas[toArrowY][toArrowX] = toArrowChar
					}
				}
			}
		}
	} else {
		c.drawLineSegment(canvas, fromX, fromY, toX, toY, connection.FromID, connection.ToID, connection.ArrowFrom, connection.ArrowTo, false, originalConnection, panX, panY)
	}
}

func (c *Canvas) isValidPos(canvas [][]rune, x, y int) bool {
	return y >= 0 && y < len(canvas) && x >= 0 && x < len(canvas[0])
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}


func (c *Canvas) SaveToFile(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Fprintf(file, "FLOWCHART\n")
	fmt.Fprintf(file, "BOXES:%d\n", len(c.boxes))
	for _, box := range c.boxes {
		encodedText := strings.ReplaceAll(box.GetText(), "\n", "\\n")
		fmt.Fprintf(file, "%d,%d,%d,%d,%s\n", box.X, box.Y, box.Width, box.Height, encodedText)
	}

	fmt.Fprintf(file, "CONNECTIONS:%d\n", len(c.connections))
	for _, connection := range c.connections {
		waypointsStr := ""
		if len(connection.Waypoints) > 0 {
			waypointParts := make([]string, len(connection.Waypoints))
			for i, wp := range connection.Waypoints {
				waypointParts[i] = fmt.Sprintf("%d:%d", wp.X, wp.Y)
			}
			waypointsStr = "|" + strings.Join(waypointParts, ",")
		}
		arrowFlags := 0
		if connection.ArrowFrom {
			arrowFlags |= 1
		}
		if connection.ArrowTo {
			arrowFlags |= 2
		}
		fmt.Fprintf(file, "%d,%d,%d,%d,%d,%d,%d,%d%s\n",
			connection.FromID, connection.ToID,
			connection.FromX, connection.FromY,
			connection.ToX, connection.ToY,
			len(connection.Waypoints), arrowFlags, waypointsStr)
	}

	fmt.Fprintf(file, "TEXTS:%d\n", len(c.texts))
	for _, text := range c.texts {
		encodedText := strings.ReplaceAll(text.GetText(), "\n", "\\n")
		fmt.Fprintf(file, "%d,%d,%s\n", text.X, text.Y, encodedText)
	}

	return nil
}

func (c *Canvas) LoadFromFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	c.boxes = c.boxes[:0]
	c.connections = c.connections[:0]
	c.texts = c.texts[:0]

	scanner := bufio.NewScanner(file)

	// Read header
	if !scanner.Scan() || scanner.Text() != "FLOWCHART" {
		return fmt.Errorf("invalid file format")
	}

	// Read boxes
	if !scanner.Scan() {
		return fmt.Errorf("missing boxes header")
	}
	boxCountStr := strings.TrimPrefix(scanner.Text(), "BOXES:")
	boxCount, err := strconv.Atoi(boxCountStr)
	if err != nil {
		return fmt.Errorf("invalid box count: %v", err)
	}

	for i := 0; i < boxCount; i++ {
		if !scanner.Scan() {
			return fmt.Errorf("missing box data")
		}
		parts := strings.Split(scanner.Text(), ",")

		if len(parts) < 3 {
			return fmt.Errorf("invalid box format")
		}

		x, _ := strconv.Atoi(parts[0])
		y, _ := strconv.Atoi(parts[1])

		var width, height int
		var text string

		if len(parts) >= 6 {
			// Old format with color (backward compatibility): X,Y,Width,Height,Color,Text
			// Skip the color field
			width, _ = strconv.Atoi(parts[2])
			height, _ = strconv.Atoi(parts[3])
			// Skip parts[4] which is the color
			text = strings.ReplaceAll(strings.Join(parts[5:], ","), "\\n", "\n")
		} else if len(parts) >= 5 {
			// Old format without color: X,Y,Width,Height,Text
			width, _ = strconv.Atoi(parts[2])
			height, _ = strconv.Atoi(parts[3])
			text = strings.ReplaceAll(strings.Join(parts[4:], ","), "\\n", "\n")
		} else {
			// Very old format: X,Y,Text
			text = strings.ReplaceAll(strings.Join(parts[2:], ","), "\\n", "\n")
			box := Box{
				X:  x,
				Y:  y,
				ID: i,
			}
			box.SetText(text)
			c.boxes = append(c.boxes, box)
			continue
		}

		box := Box{
			X:  x,
			Y:  y,
			ID: i,
		}
		box.SetText(text)
		box.Width = width
		box.Height = height
		c.boxes = append(c.boxes, box)
	}

	// Read connections
	if !scanner.Scan() {
		return fmt.Errorf("missing connections header")
	}
	connectionCountStr := strings.TrimPrefix(scanner.Text(), "CONNECTIONS:")
	connectionCount, err := strconv.Atoi(connectionCountStr)
	if err != nil {
		return fmt.Errorf("invalid connection count: %v", err)
	}

	for i := 0; i < connectionCount; i++ {
		if !scanner.Scan() {
			return fmt.Errorf("missing connection data")
		}
		line := scanner.Text()

		parts := strings.Split(line, "|")
		mainParts := strings.Split(parts[0], ",")

		if len(mainParts) == 2 {
			fromID, _ := strconv.Atoi(mainParts[0])
			toID, _ := strconv.Atoi(mainParts[1])
			if fromID >= 0 && fromID < len(c.boxes) && toID >= 0 && toID < len(c.boxes) {
				c.AddConnection(fromID, toID)
			}
		} else if len(mainParts) >= 7 {
			fromID, _ := strconv.Atoi(mainParts[0])
			toID, _ := strconv.Atoi(mainParts[1])
			fromX, _ := strconv.Atoi(mainParts[2])
			fromY, _ := strconv.Atoi(mainParts[3])
			toX, _ := strconv.Atoi(mainParts[4])
			toY, _ := strconv.Atoi(mainParts[5])
			waypointCount, _ := strconv.Atoi(mainParts[6])

			arrowFlags := 2
			if len(mainParts) >= 8 {
				arrowFlags, _ = strconv.Atoi(mainParts[7])
			}

			var waypoints []point
			if len(parts) > 1 && waypointCount > 0 {
				waypointParts := strings.Split(parts[1], ",")
				for j := 0; j < waypointCount && j < len(waypointParts); j++ {
					wpParts := strings.Split(waypointParts[j], ":")
					if len(wpParts) == 2 {
						wpX, _ := strconv.Atoi(wpParts[0])
						wpY, _ := strconv.Atoi(wpParts[1])
						waypoints = append(waypoints, point{wpX, wpY})
					}
				}
			}

			connection := Connection{
				FromID:    fromID,
				ToID:      toID,
				FromX:     fromX,
				FromY:     fromY,
				ToX:       toX,
				ToY:       toY,
				Waypoints: waypoints,
				ArrowFrom: (arrowFlags & 1) != 0,
				ArrowTo:   (arrowFlags & 2) != 0,
			}
			c.connections = append(c.connections, connection)
		} else {
			return fmt.Errorf("invalid connection format")
		}
	}

	if scanner.Scan() {
		textCountStr := strings.TrimPrefix(scanner.Text(), "TEXTS:")
		textCount, err := strconv.Atoi(textCountStr)
		if err == nil {
			for i := 0; i < textCount; i++ {
				if !scanner.Scan() {
					break
				}
				parts := strings.Split(scanner.Text(), ",")
				if len(parts) >= 3 {
					// Format: X,Y,Text (or X,Y,Color,Text for backward compatibility)
					x, _ := strconv.Atoi(parts[0])
					y, _ := strconv.Atoi(parts[1])
					// Skip color field if present (parts[2] might be color)
					textStart := 2
					if len(parts) >= 4 {
						// Has color field, skip it
						textStart = 3
					}
					text := strings.ReplaceAll(strings.Join(parts[textStart:], ","), "\\n", "\n")
					c.AddText(x, y, text)
				}
			}
		}
	}

	return scanner.Err()
}

func (c *Canvas) ExportToPNG(filename string, renderWidth, renderHeight int, panX, panY int) error {
	if len(c.boxes) == 0 && len(c.connections) == 0 && len(c.texts) == 0 {
		return fmt.Errorf("nothing to export")
	}

	// Character cell dimensions (pixels per character)
	charWidth := 8.0
	charHeight := 16.0

	// Calculate bounds of all elements
	minX, minY := 0, 0
	maxX, maxY := 0, 0
	hasElements := false

	// Check boxes
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

	// Check connections (including waypoints)
	for _, conn := range c.connections {
		points := []point{{conn.FromX, conn.FromY}}
		points = append(points, conn.Waypoints...)
		points = append(points, point{conn.ToX, conn.ToY})

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

	// Check texts
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
			// Estimate text bounds (rough)
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
	}

	if !hasElements {
		return fmt.Errorf("nothing to export")
	}

	// Add padding
	padding := 2
	minX -= padding
	minY -= padding
	maxX += padding
	maxY += padding

	// Calculate image dimensions
	imageWidth := int(float64(maxX-minX) * charWidth)
	imageHeight := int(float64(maxY-minY) * charHeight)

	// Create drawing context
	dc := gg.NewContext(imageWidth, imageHeight)
	dc.SetColor(color.White)
	dc.Clear()
	dc.SetColor(color.Black)

	// Load font for text rendering
	fontData := gomono.TTF
	ttfFont, err := truetype.Parse(fontData)
	if err != nil {
		return fmt.Errorf("failed to parse font: %v", err)
	}

	fontSize := 12.0
	face := truetype.NewFace(ttfFont, &truetype.Options{
		Size:    fontSize,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	dc.SetFontFace(face)

	// Draw connections first (so they appear behind boxes)
	for _, conn := range c.connections {
		c.drawConnectionPNG(dc, conn, minX, minY, charWidth, charHeight)
	}

	// Draw texts
	for _, text := range c.texts {
		c.drawTextPNG(dc, text, minX, minY, charWidth, charHeight)
	}

	// Draw boxes last (so they appear on top)
	for _, box := range c.boxes {
		c.drawBoxPNG(dc, box, minX, minY, charWidth, charHeight)
	}

	return dc.SavePNG(filename)
}

func (c *Canvas) drawConnectionPNG(dc *gg.Context, conn Connection, minX, minY int, charWidth, charHeight float64) {
	// Build path from waypoints
	points := []point{{conn.FromX, conn.FromY}}
	points = append(points, conn.Waypoints...)
	points = append(points, point{conn.ToX, conn.ToY})

	if len(points) < 2 {
		return
	}

	// Convert to pixel coordinates
	dc.SetLineWidth(1.0)
	dc.SetColor(color.Black)

	// Draw line segments
	for i := 0; i < len(points)-1; i++ {
		x1 := float64(points[i].X-minX) * charWidth
		y1 := float64(points[i].Y-minY) * charHeight
		x2 := float64(points[i+1].X-minX) * charWidth
		y2 := float64(points[i+1].Y-minY) * charHeight

		dc.DrawLine(x1, y1, x2, y2)
		dc.Stroke()
	}

	// Draw arrows
	if conn.ArrowFrom && len(points) > 0 {
		// Arrow at start
		if len(points) > 1 {
			c.drawArrowPNG(dc, points[1].X, points[1].Y, points[0].X, points[0].Y, minX, minY, charWidth, charHeight)
		}
	}
	if conn.ArrowTo && len(points) > 1 {
		// Arrow at end
		c.drawArrowPNG(dc, points[len(points)-2].X, points[len(points)-2].Y, points[len(points)-1].X, points[len(points)-1].Y, minX, minY, charWidth, charHeight)
	}
}

func (c *Canvas) drawArrowPNG(dc *gg.Context, fromX, fromY, toX, toY, minX, minY int, charWidth, charHeight float64) {
	// Convert to pixel coordinates
	fx := float64(fromX-minX) * charWidth
	fy := float64(fromY-minY) * charHeight
	tx := float64(toX-minX) * charWidth
	ty := float64(toY-minY) * charHeight

	// Calculate arrow direction
	dx := tx - fx
	dy := ty - fy
	length := math.Sqrt(dx*dx + dy*dy)
	if length < 0.1 {
		return
	}

	// Normalize
	dx /= length
	dy /= length

	// Arrow size
	arrowSize := 6.0
	arrowAngle := 0.5 // radians

	// Arrow tip
	tipX := tx
	tipY := ty

	// Arrow base points
	baseX1 := tx - arrowSize*dx + arrowSize*dy*arrowAngle
	baseY1 := ty - arrowSize*dy - arrowSize*dx*arrowAngle
	baseX2 := tx - arrowSize*dx - arrowSize*dy*arrowAngle
	baseY2 := ty - arrowSize*dy + arrowSize*dx*arrowAngle

	// Draw arrow
	dc.MoveTo(tipX, tipY)
	dc.LineTo(baseX1, baseY1)
	dc.LineTo(baseX2, baseY2)
	dc.ClosePath()
	dc.Fill()
}

func (c *Canvas) drawBoxPNG(dc *gg.Context, box Box, minX, minY int, charWidth, charHeight float64) {
	// Convert box coordinates to pixel coordinates
	x := float64(box.X-minX) * charWidth
	y := float64(box.Y-minY) * charHeight
	width := float64(box.Width) * charWidth
	height := float64(box.Height) * charHeight

	// Draw box border
	dc.SetLineWidth(1.0)
	dc.SetColor(color.Black)
	dc.DrawRectangle(x, y, width, height)
	dc.Stroke()

	// Draw box text
	dc.SetColor(color.Black)
	textY := y + charHeight
	for i, line := range box.Lines {
		textX := x + charWidth
		dc.DrawString(line, textX, textY+float64(i)*charHeight)
	}
}

func (c *Canvas) drawTextPNG(dc *gg.Context, text Text, minX, minY int, charWidth, charHeight float64) {
	// Convert text coordinates to pixel coordinates
	x := float64(text.X-minX) * charWidth
	y := float64(text.Y-minY) * charHeight

	// Draw text lines
	dc.SetColor(color.Black)
	for i, line := range text.Lines {
		dc.DrawString(line, x, y+float64(i)*charHeight)
	}
}
