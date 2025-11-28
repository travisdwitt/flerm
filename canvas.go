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
	highlights  map[string]int // Map of "x,y" to color index (0-7)
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
		highlights:  make(map[string]int),
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

// MoveConnection moves all points of a connection (endpoints and waypoints) by the given delta
func (c *Canvas) MoveConnection(connIdx int, deltaX, deltaY int) {
	if connIdx < 0 || connIdx >= len(c.connections) {
		return
	}
	conn := &c.connections[connIdx]
	
	// Move endpoints
	conn.FromX += deltaX
	conn.FromY += deltaY
	conn.ToX += deltaX
	conn.ToY += deltaY
	
	// Move waypoints
	for i := range conn.Waypoints {
		conn.Waypoints[i].X += deltaX
		conn.Waypoints[i].Y += deltaY
	}
	
	// Ensure coordinates don't go negative
	if conn.FromX < 0 {
		conn.FromX = 0
	}
	if conn.FromY < 0 {
		conn.FromY = 0
	}
	if conn.ToX < 0 {
		conn.ToX = 0
	}
	if conn.ToY < 0 {
		conn.ToY = 0
	}
	for i := range conn.Waypoints {
		if conn.Waypoints[i].X < 0 {
			conn.Waypoints[i].X = 0
		}
		if conn.Waypoints[i].Y < 0 {
			conn.Waypoints[i].Y = 0
		}
	}
}

// MoveConnectionWaypoints moves only the waypoints of a connection by the given delta
// This is used when endpoints are already recalculated by MoveBox
func (c *Canvas) MoveConnectionWaypoints(connIdx int, deltaX, deltaY int) {
	if connIdx < 0 || connIdx >= len(c.connections) {
		return
	}
	conn := &c.connections[connIdx]
	
	// Move only waypoints (endpoints are already updated by MoveBox)
	for i := range conn.Waypoints {
		conn.Waypoints[i].X += deltaX
		conn.Waypoints[i].Y += deltaY
		
		// Ensure coordinates don't go negative
		if conn.Waypoints[i].X < 0 {
			conn.Waypoints[i].X = 0
		}
		if conn.Waypoints[i].Y < 0 {
			conn.Waypoints[i].Y = 0
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

func (c *Canvas) Render(width, height int, selectedBox int, previewFromX, previewFromY int, previewWaypoints []point, previewToX, previewToY int, panX, panY int, cursorX, cursorY int, showCursor bool, editBoxID int, editTextID int, editCursorPos int, editText string, editTextX int, editTextY int, selectionStartX, selectionStartY, selectionEndX, selectionEndY int) []string {
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
	// Note: previewFromX, previewFromY, previewToX, previewToY, and waypoints are in world coordinates
	if previewFromX >= 0 && previewFromY >= 0 {
		previewConnection := Connection{
			FromID:    -1,
			ToID:      -1,
			FromX:     previewFromX, // World coordinates
			FromY:     previewFromY, // World coordinates
			ToX:       previewToX,   // World coordinates
			ToY:       previewToY,   // World coordinates
			Waypoints: make([]point, len(previewWaypoints)),
		}
		for i, wp := range previewWaypoints {
			previewConnection.Waypoints[i] = point{X: wp.X, Y: wp.Y} // World coordinates
		}
		c.drawConnectionWithPan(canvas, previewConnection, panX, panY)
	}

	// Draw texts (plain text without borders)
	for _, text := range c.texts {
		c.drawTextWithPan(canvas, text, panX, panY)
	}
	
	// Draw text input preview if in text input mode
	if editTextX >= 0 && editTextY >= 0 && editText != "" {
		previewText := Text{
			X:     editTextX,
			Y:     editTextY,
			Lines: strings.Split(editText, "\n"),
			ID:    -1,
		}
		c.drawTextWithPan(canvas, previewText, panX, panY)
	}

	// Draw boxes last (so they appear on top)
	for i, box := range c.boxes {
		isSelected := (i == selectedBox)
		c.drawBoxWithPan(canvas, box, isSelected, panX, panY)
	}

	// Draw text editing cursor if in edit mode
	if editBoxID >= 0 && editBoxID < len(c.boxes) {
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
		// Text input mode - show cursor at input position
		editCursorX, editCursorY := c.calculateTextCursorPositionForNewText(editTextX, editTextY, editCursorPos, editText, panX, panY)
		if editCursorY >= 0 && editCursorY < height && editCursorX >= 0 && editCursorX < width {
			if editCursorY < len(canvas) && editCursorX < len(canvas[editCursorY]) {
				canvas[editCursorY][editCursorX] = '█'
			}
		}
	}

	// Draw navigation cursor if needed (before applying colors)
	if showCursor && cursorY >= 0 && cursorY < height && cursorX >= 0 && cursorX < width {
		if cursorY < len(canvas) && cursorX < len(canvas[cursorY]) {
			canvas[cursorY][cursorX] = '█'
		}
	}

	// Draw selection rectangle if in multi-select mode (before applying colors)
	if selectionStartX >= 0 && selectionStartY >= 0 {
		// Convert world coordinates to screen coordinates
		startScreenX := selectionStartX - panX
		startScreenY := selectionStartY - panY
		endScreenX := selectionEndX - panX
		endScreenY := selectionEndY - panY
		
		// Calculate rectangle bounds
		minX := startScreenX
		if endScreenX < startScreenX {
			minX = endScreenX
		}
		maxX := startScreenX
		if endScreenX > startScreenX {
			maxX = endScreenX
		}
		minY := startScreenY
		if endScreenY < startScreenY {
			minY = endScreenY
		}
		maxY := startScreenY
		if endScreenY > startScreenY {
			maxY = endScreenY
		}
		
		// Clamp to visible area
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
		
		// Draw selection rectangle border
		// Top and bottom edges
		for x := minX; x <= maxX && x < width; x++ {
			if minY >= 0 && minY < height && x >= 0 {
				if x == minX || x == maxX {
					// Corners
					if minY == maxY {
						// Single line
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
					// Horizontal edge
					if len(canvas[minY]) > x {
						canvas[minY][x] = '─'
					}
				}
				if maxY != minY && maxY >= 0 && maxY < height {
					if x == minX || x == maxX {
						// Corners
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
						// Horizontal edge
						if len(canvas[maxY]) > x {
							canvas[maxY][x] = '─'
						}
					}
				}
			}
		}
		
		// Left and right edges
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

	// Apply highlights (colored backgrounds)
	for key, colorIndex := range c.highlights {
		var x, y int
		fmt.Sscanf(key, "%d,%d", &x, &y)
		// Convert world coordinates to screen coordinates
		screenX := x - panX
		screenY := y - panY
		if screenY >= 0 && screenY < height && screenX >= 0 && screenX < width {
			// Mark this cell as having a color
			if screenY < len(colorMap) && screenX < len(colorMap[screenY]) {
				colorMap[screenY][screenX] = colorIndex
			}
		}
	}

	// Convert to strings with consistent line lengths and apply colors
	result := make([]string, height)
	for i, row := range canvas {
		// Ensure each line is exactly the right width
		line := make([]rune, width)
		copy(line, row)
		for j := len(row); j < width; j++ {
			line[j] = ' '
		}

		// Build string with color codes
		var coloredLine strings.Builder
		currentColor := -1
		for j, char := range line {
			cellColor := -1
			if i < len(colorMap) && j < len(colorMap[i]) {
				cellColor = colorMap[i][j]
			}

			// Apply color if it changed
			if cellColor != currentColor {
				if currentColor != -1 {
					coloredLine.WriteString(colorReset)
				}
				if cellColor != -1 {
					coloredLine.WriteString(getColorCode(cellColor))
				}
				currentColor = cellColor
			}

			coloredLine.WriteRune(char)
		}

		// Reset color at end of line if needed
		if currentColor != -1 {
			coloredLine.WriteString(colorReset)
		}

		result[i] = coloredLine.String()
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

// calculateTextCursorPosition calculates the screen position of the text editing cursor for a box
func (c *Canvas) calculateTextCursorPosition(box Box, cursorPos int, text string, panX, panY int) (int, int) {
	// Split text into lines
	lines := strings.Split(text, "\n")
	
	// Find which line the cursor is on
	currentPos := 0
	for lineIdx, line := range lines {
		lineLength := len([]rune(line))
		if cursorPos <= currentPos+lineLength {
			// Cursor is on this line
			posInLine := cursorPos - currentPos
			// Box text starts at box.X + 1, box.Y + 1 + lineIdx
			cursorX := box.X + 1 + posInLine - panX
			cursorY := box.Y + 1 + lineIdx - panY
			return cursorX, cursorY
		}
		currentPos += lineLength + 1 // +1 for newline
	}
	
	// Cursor is at the end - place it after the last line
	if len(lines) > 0 {
		lastLine := lines[len(lines)-1]
		cursorX := box.X + 1 + len([]rune(lastLine)) - panX
		cursorY := box.Y + 1 + len(lines) - 1 - panY
		return cursorX, cursorY
	}
	
	// Empty text - cursor at start
	cursorX := box.X + 1 - panX
	cursorY := box.Y + 1 - panY
	return cursorX, cursorY
}

// calculateTextCursorPositionForText calculates the screen position of the text editing cursor for a text object
func (c *Canvas) calculateTextCursorPositionForText(text Text, cursorPos int, textContent string, panX, panY int) (int, int) {
	// Split text into lines
	lines := strings.Split(textContent, "\n")
	
	// Find which line the cursor is on
	currentPos := 0
	for lineIdx, line := range lines {
		lineLength := len([]rune(line))
		if cursorPos <= currentPos+lineLength {
			// Cursor is on this line
			posInLine := cursorPos - currentPos
			// Text starts at text.X, text.Y + lineIdx
			cursorX := text.X + posInLine - panX
			cursorY := text.Y + lineIdx - panY
			return cursorX, cursorY
		}
		currentPos += lineLength + 1 // +1 for newline
	}
	
	// Cursor is at the end - place it after the last line
	if len(lines) > 0 {
		lastLine := lines[len(lines)-1]
		cursorX := text.X + len([]rune(lastLine)) - panX
		cursorY := text.Y + len(lines) - 1 - panY
		return cursorX, cursorY
	}
	
	// Empty text - cursor at start
	cursorX := text.X - panX
	cursorY := text.Y - panY
	return cursorX, cursorY
}

// calculateTextCursorPositionForNewText calculates the screen position of the text editing cursor for new text input
func (c *Canvas) calculateTextCursorPositionForNewText(textX, textY int, cursorPos int, textContent string, panX, panY int) (int, int) {
	// Split text into lines
	lines := strings.Split(textContent, "\n")
	
	// Find which line the cursor is on
	currentPos := 0
	for lineIdx, line := range lines {
		lineLength := len([]rune(line))
		if cursorPos <= currentPos+lineLength {
			// Cursor is on this line
			posInLine := cursorPos - currentPos
			// Text starts at textX, textY + lineIdx
			cursorX := textX + posInLine - panX
			cursorY := textY + lineIdx - panY
			return cursorX, cursorY
		}
		currentPos += lineLength + 1 // +1 for newline
	}
	
	// Cursor is at the end - place it after the last line
	if len(lines) > 0 {
		lastLine := lines[len(lines)-1]
		cursorX := textX + len([]rune(lastLine)) - panX
		cursorY := textY + len(lines) - 1 - panY
		return cursorX, cursorY
	}
	
	// Empty text - cursor at start
	cursorX := textX - panX
	cursorY := textY - panY
	return cursorX, cursorY
}

func (c *Canvas) isPointInBox(x, y int, excludeFromID, excludeToID int) bool {
	for i, box := range c.boxes {
		if x > box.X && x < box.X+box.Width-1 && y > box.Y && y < box.Y+box.Height-1 {
			if i != excludeFromID && i != excludeToID {
				return true
			}
		}
	}
	return false
}

func (c *Canvas) isPointInBoxScreen(x, y int, excludeFromID, excludeToID int, panX, panY int) bool {
	for i, box := range c.boxes {
		boxScreenX := box.X - panX
		boxScreenY := box.Y - panY
		if x > boxScreenX && x < boxScreenX+box.Width-1 && y > boxScreenY && y < boxScreenY+box.Height-1 {
			if i != excludeFromID && i != excludeToID {
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
			if c.isValidPos(canvas, fromX, y) && !c.isPointInBoxScreen(fromX, y, excludeFromID, excludeToID, panX, panY) {
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
				fromBoxScreenY := fromBox.Y - panY
				fromOnTopEdge := (fromY == fromBoxScreenY)
				fromOnBottomEdge := (fromY == fromBoxScreenY+fromBox.Height-1)
				if !fromOnTopEdge && !fromOnBottomEdge {
					if abs(fromY-fromBoxScreenY) < abs(fromY-(fromBoxScreenY+fromBox.Height-1)) {
						fromOnTopEdge = true
					} else {
						fromOnBottomEdge = true
					}
				}
				if fromOnTopEdge {
					fromArrowY = fromBoxScreenY - 1
					fromArrowChar = '▼'
				} else if fromOnBottomEdge {
					fromArrowY = fromBoxScreenY + fromBox.Height
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
			if c.isValidPos(canvas, fromX, fromArrowY) && !c.isPointInBoxScreen(fromX, fromArrowY, excludeFromID, excludeToID, panX, panY) {
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
			toBoxScreenX := toBox.X - panX
			onLeftEdge = (toX == toBoxScreenX)
			onRightEdge = (toX == toBoxScreenX+toBox.Width-1)
			if !onLeftEdge && !onRightEdge {
				if abs(toX-toBoxScreenX) < abs(toX-(toBoxScreenX+toBox.Width-1)) {
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
			if c.isValidPos(canvas, x, fromY) && !c.isPointInBoxScreen(x, fromY, excludeFromID, excludeToID, panX, panY) {
				canvas[fromY][x] = '─'
			}
		}

		if drawArrowTo {
			if excludeToID >= 0 && excludeToID < len(c.boxes) {
				toBox := c.boxes[excludeToID]
				toBoxScreenX := toBox.X - panX
				if arrowX >= toBoxScreenX && arrowX < toBoxScreenX+toBox.Width {
					if onLeftEdge {
						arrowX = toBoxScreenX - 1
					} else if onRightEdge {
						arrowX = toBoxScreenX + toBox.Width
					}
				}
			}
			if c.isValidPos(canvas, arrowX, toY) {
				if !c.isPointInBoxScreen(arrowX, toY, excludeFromID, excludeToID, panX, panY) {
					canvas[toY][arrowX] = arrowChar
				}
			}
		}
		if drawArrowFrom {
			var fromArrowX int
			var fromArrowChar rune
			if excludeFromID >= 0 && excludeFromID < len(c.boxes) {
				fromBox := c.boxes[excludeFromID]
				fromBoxScreenX := fromBox.X - panX
				fromOnLeftEdge := (fromX == fromBoxScreenX)
				fromOnRightEdge := (fromX == fromBoxScreenX+fromBox.Width-1)
				if !fromOnLeftEdge && !fromOnRightEdge {
					if abs(fromX-fromBoxScreenX) < abs(fromX-(fromBoxScreenX+fromBox.Width-1)) {
						fromOnLeftEdge = true
					} else {
						fromOnRightEdge = true
					}
				}
				if fromOnLeftEdge {
					fromArrowX = fromBoxScreenX - 1
					fromArrowChar = '▶'
				} else if fromOnRightEdge {
					fromArrowX = fromBoxScreenX + fromBox.Width
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
				if !c.isPointInBoxScreen(fromArrowX, fromY, excludeFromID, excludeToID, panX, panY) {
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
				if c.isValidPos(canvas, x, fromY) && !c.isPointInBoxScreen(x, fromY, excludeFromID, excludeToID, panX, panY) {
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
				if c.isValidPos(canvas, cornerX, y) && !c.isPointInBoxScreen(cornerX, y, excludeFromID, excludeToID, panX, panY) {
					canvas[y][cornerX] = '│'
				}
			}
		}

		if !skipCorner && c.isValidPos(canvas, cornerX, cornerY) && !c.isPointInBoxScreen(cornerX, cornerY, excludeFromID, excludeToID, panX, panY) {
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
			// Use screen coordinates for calculations

			if excludeToID >= 0 && excludeToID < len(c.boxes) {
				toBox := c.boxes[excludeToID]
				toBoxScreenX := toBox.X - panX
				toBoxScreenY := toBox.Y - panY
				origToX := originalConnection.ToX
				origToY := originalConnection.ToY
				origToXScreen := origToX - panX
				origToYScreen := origToY - panY

				onLeftEdge := (origToXScreen == toBoxScreenX)
				onRightEdge := (origToXScreen == toBoxScreenX+toBox.Width-1)
				onTopEdge := (origToYScreen == toBoxScreenY)
				onBottomEdge := (origToYScreen == toBoxScreenY+toBox.Height-1)

				if !onLeftEdge && !onRightEdge && !onTopEdge && !onBottomEdge {
					if abs(origToXScreen-toBoxScreenX) <= abs(origToXScreen-(toBoxScreenX+toBox.Width-1)) && abs(origToXScreen-toBoxScreenX) <= abs(origToYScreen-toBoxScreenY) && abs(origToXScreen-toBoxScreenX) <= abs(origToYScreen-(toBoxScreenY+toBox.Height-1)) {
						onLeftEdge = true
					} else if abs(origToXScreen-(toBoxScreenX+toBox.Width-1)) <= abs(origToYScreen-toBoxScreenY) && abs(origToXScreen-(toBoxScreenX+toBox.Width-1)) <= abs(origToYScreen-(toBoxScreenY+toBox.Height-1)) {
						onRightEdge = true
					} else if abs(origToYScreen-toBoxScreenY) <= abs(origToYScreen-(toBoxScreenY+toBox.Height-1)) {
						onTopEdge = true
					} else {
						onBottomEdge = true
					}
				}

				if onLeftEdge {
					arrowX = toBoxScreenX - 1
					arrowY = origToYScreen
					arrowChar = '▶'
				} else if onRightEdge {
					arrowX = toBoxScreenX + toBox.Width
					arrowY = origToYScreen
					arrowChar = '◀'
				} else if onTopEdge {
					arrowX = origToXScreen
					arrowY = toBoxScreenY - 1
					arrowChar = '▲'
				} else if onBottomEdge {
					arrowX = origToXScreen
					arrowY = toBoxScreenY + toBox.Height
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
						if origToXScreen < toBoxScreenX+toBox.Width/2 {
							arrowX = toBoxScreenX - 1
							arrowY = origToYScreen
							arrowChar = '▶'
						} else {
							arrowX = toBoxScreenX + toBox.Width
							arrowY = origToYScreen
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
					if c.isValidPos(canvas, x, arrowY) && !c.isPointInBoxScreen(x, arrowY, excludeFromID, excludeToID, panX, panY) {
						canvas[arrowY][x] = '─'
					}
				}
			}

			if c.isValidPos(canvas, arrowX, arrowY) && !c.isPointInBoxScreen(arrowX, arrowY, excludeFromID, excludeToID, panX, panY) {
				canvas[arrowY][arrowX] = arrowChar
			}
		}
		if drawArrowFrom {
			var fromArrowX, fromArrowY int
			var fromArrowChar rune
			// Use screen coordinates for calculations
			if excludeFromID >= 0 && excludeFromID < len(c.boxes) {
				fromBox := c.boxes[excludeFromID]
				fromBoxScreenX := fromBox.X - panX
				fromBoxScreenY := fromBox.Y - panY
				origFromX := originalConnection.FromX
				origFromY := originalConnection.FromY
				origFromXScreen := origFromX - panX
				origFromYScreen := origFromY - panY

				fromOnLeftEdge := (origFromXScreen == fromBoxScreenX)
				fromOnRightEdge := (origFromXScreen == fromBoxScreenX+fromBox.Width-1)
				fromOnTopEdge := (origFromYScreen == fromBoxScreenY)
				fromOnBottomEdge := (origFromYScreen == fromBoxScreenY+fromBox.Height-1)
				if !fromOnLeftEdge && !fromOnRightEdge {
					if abs(origFromXScreen-fromBoxScreenX) < abs(origFromXScreen-(fromBoxScreenX+fromBox.Width-1)) {
						fromOnLeftEdge = true
					} else {
						fromOnRightEdge = true
					}
				}
				if !fromOnTopEdge && !fromOnBottomEdge {
					if abs(origFromYScreen-fromBoxScreenY) < abs(origFromYScreen-(fromBoxScreenY+fromBox.Height-1)) {
						fromOnTopEdge = true
					} else {
						fromOnBottomEdge = true
					}
				}
				if fromOnTopEdge {
					fromArrowX = origFromXScreen
					fromArrowY = fromBoxScreenY - 1
					fromArrowChar = '▼'
				} else if fromOnBottomEdge {
					fromArrowX = origFromXScreen
					fromArrowY = fromBoxScreenY + fromBox.Height
					fromArrowChar = '▲'
				} else if fromOnLeftEdge {
					fromArrowX = fromBoxScreenX - 1
					fromArrowY = origFromYScreen
					fromArrowChar = '▶'
				} else if fromOnRightEdge {
					fromArrowX = fromBoxScreenX + fromBox.Width
					fromArrowY = origFromYScreen
					fromArrowChar = '◀'
				} else {
					if origFromXScreen == cornerX {
						if origFromYScreen < cornerY {
							fromArrowX = fromX
							fromArrowY = fromY - 1
							fromArrowChar = '▼'
						} else {
							fromArrowX = fromX
							fromArrowY = fromY + 1
							fromArrowChar = '▲'
						}
					} else if origFromYScreen == cornerY {
						if origFromXScreen < cornerX {
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
			if c.isValidPos(canvas, fromArrowX, fromArrowY) && !c.isPointInBoxScreen(fromArrowX, fromArrowY, excludeFromID, excludeToID, panX, panY) {
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
					if c.isValidPos(canvas, prevPoint.X, y) && !c.isPointInBoxScreen(prevPoint.X, y, connection.FromID, connection.ToID, panX, panY) {
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
						if c.isValidPos(canvas, cornerX, cornerY) && !c.isPointInBoxScreen(cornerX, cornerY, connection.FromID, connection.ToID, panX, panY) {
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
					if c.isValidPos(canvas, fromArrowX, fromArrowY) && !c.isPointInBoxScreen(fromArrowX, fromArrowY, connection.FromID, connection.ToID, panX, panY) {
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
					if c.isValidPos(canvas, toArrowX, toArrowY) && !c.isPointInBoxScreen(toArrowX, toArrowY, connection.FromID, connection.ToID, panX, panY) {
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
					if c.isValidPos(canvas, x, prevPoint.Y) && !c.isPointInBoxScreen(x, prevPoint.Y, connection.FromID, connection.ToID, panX, panY) {
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
						if c.isValidPos(canvas, cornerX, cornerY) && !c.isPointInBoxScreen(cornerX, cornerY, connection.FromID, connection.ToID, panX, panY) {
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
					if c.isValidPos(canvas, fromArrowX, fromArrowY) && !c.isPointInBoxScreen(fromArrowX, fromArrowY, connection.FromID, connection.ToID, panX, panY) {
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
					if c.isValidPos(canvas, toArrowX, toArrowY) && !c.isPointInBoxScreen(toArrowX, toArrowY, connection.FromID, connection.ToID, panX, panY) {
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
							if c.isValidPos(canvas, x, prevPoint.Y) && !c.isPointInBoxScreen(x, prevPoint.Y, connection.FromID, connection.ToID, panX, panY) {
								canvas[prevPoint.Y][x] = '─'
							}
						}
					} else if prevPoint.X > cornerX {
						for x := cornerX + 1; x < prevPoint.X; x++ {
							if c.isValidPos(canvas, x, prevPoint.Y) && !c.isPointInBoxScreen(x, prevPoint.Y, connection.FromID, connection.ToID, panX, panY) {
								canvas[prevPoint.Y][x] = '─'
							}
						}
					}

					if cornerY < nextPoint.Y {
						for y := cornerY + 1; y < nextPoint.Y; y++ {
							if c.isValidPos(canvas, cornerX, y) && !c.isPointInBoxScreen(cornerX, y, connection.FromID, connection.ToID, panX, panY) {
								canvas[y][cornerX] = '│'
							}
						}
					} else if cornerY > nextPoint.Y {
						for y := nextPoint.Y + 1; y < cornerY; y++ {
							if c.isValidPos(canvas, cornerX, y) && !c.isPointInBoxScreen(cornerX, y, connection.FromID, connection.ToID, panX, panY) {
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
					if c.isValidPos(canvas, cornerX, cornerY) && !c.isPointInBoxScreen(cornerX, cornerY, connection.FromID, connection.ToID, panX, panY) {
						canvas[cornerY][cornerX] = cornerChar
					}
				} else {
					// Vertical from prev -> corner, then horizontal corner -> next
					cornerX = prevPoint.X
					cornerY = nextPoint.Y

					if prevPoint.Y < cornerY {
						for y := prevPoint.Y + 1; y < cornerY; y++ {
							if c.isValidPos(canvas, cornerX, y) && !c.isPointInBoxScreen(cornerX, y, connection.FromID, connection.ToID, panX, panY) {
								canvas[y][cornerX] = '│'
							}
						}
					} else if prevPoint.Y > cornerY {
						for y := cornerY + 1; y < prevPoint.Y; y++ {
							if c.isValidPos(canvas, cornerX, y) && !c.isPointInBoxScreen(cornerX, y, connection.FromID, connection.ToID, panX, panY) {
								canvas[y][cornerX] = '│'
							}
						}
					}

					if cornerX < nextPoint.X {
						for x := cornerX + 1; x < nextPoint.X; x++ {
							if c.isValidPos(canvas, x, cornerY) && !c.isPointInBoxScreen(x, cornerY, connection.FromID, connection.ToID, panX, panY) {
								canvas[cornerY][x] = '─'
							}
						}
					} else if cornerX > nextPoint.X {
						for x := nextPoint.X + 1; x < cornerX; x++ {
							if c.isValidPos(canvas, x, cornerY) && !c.isPointInBoxScreen(x, cornerY, connection.FromID, connection.ToID, panX, panY) {
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
					if c.isValidPos(canvas, cornerX, cornerY) && !c.isPointInBoxScreen(cornerX, cornerY, connection.FromID, connection.ToID, panX, panY) {
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
					if c.isValidPos(canvas, fromArrowX, fromArrowY) && !c.isPointInBoxScreen(fromArrowX, fromArrowY, connection.FromID, connection.ToID, panX, panY) {
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
					if c.isValidPos(canvas, toArrowX, toArrowY) && !c.isPointInBoxScreen(toArrowX, toArrowY, connection.FromID, connection.ToID, panX, panY) {
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

// Helper function to get ANSI color code for background
func getColorCode(colorIndex int) string {
	// ANSI background color codes: 40-47 for standard colors
	// Using standard colors for better terminal compatibility
	// Colors: Gray (white), Red, Green, Yellow, Blue, Magenta, Cyan, White
	colors := []int{47, 41, 42, 43, 44, 45, 46, 47}
	if colorIndex < 0 || colorIndex >= len(colors) {
		return ""
	}
	// Use proper ANSI escape sequence format - ensure it's a valid escape sequence
	return fmt.Sprintf("\x1b[%dm", colors[colorIndex])
}

// Reset color code
const colorReset = "\x1b[0m"

// Set highlight at world coordinates
func (c *Canvas) SetHighlight(x, y int, colorIndex int) {
	if colorIndex < 0 || colorIndex >= numColors {
		return
	}
	key := fmt.Sprintf("%d,%d", x, y)
	c.highlights[key] = colorIndex
}

// Get highlight color at world coordinates
func (c *Canvas) GetHighlight(x, y int) int {
	key := fmt.Sprintf("%d,%d", x, y)
	if color, ok := c.highlights[key]; ok {
		return color
	}
	return -1
}

// Clear highlight at world coordinates
func (c *Canvas) ClearHighlight(x, y int) {
	key := fmt.Sprintf("%d,%d", x, y)
	delete(c.highlights, key)
}

// MoveHighlight moves a highlight from one position to another
func (c *Canvas) MoveHighlight(fromX, fromY, toX, toY int) {
	fromKey := fmt.Sprintf("%d,%d", fromX, fromY)
	if colorIndex, exists := c.highlights[fromKey]; exists {
		// Remove from old position
		delete(c.highlights, fromKey)
		// Set at new position (only if new position is valid)
		if toX >= 0 && toY >= 0 {
			toKey := fmt.Sprintf("%d,%d", toX, toY)
			c.highlights[toKey] = colorIndex
		}
	}
}

// MoveHighlightsInRegion moves all highlights in a rectangular region by the given delta
func (c *Canvas) MoveHighlightsInRegion(minX, minY, maxX, maxY, deltaX, deltaY int) {
	// Collect all highlights in the region
	highlightsToMove := make(map[string]int)
	for key, colorIndex := range c.highlights {
		var x, y int
		fmt.Sscanf(key, "%d,%d", &x, &y)
		if x >= minX && x <= maxX && y >= minY && y <= maxY {
			highlightsToMove[key] = colorIndex
		}
	}
	
	// Move each highlight
	for key, colorIndex := range highlightsToMove {
		var x, y int
		fmt.Sscanf(key, "%d,%d", &x, &y)
		// Remove from old position
		delete(c.highlights, key)
		// Set at new position
		newX := x + deltaX
		newY := y + deltaY
		if newX >= 0 && newY >= 0 {
			newKey := fmt.Sprintf("%d,%d", newX, newY)
			c.highlights[newKey] = colorIndex
		}
	}
}

// Get all cells that make up a box
func (c *Canvas) GetBoxCells(boxID int) []point {
	if boxID < 0 || boxID >= len(c.boxes) {
		return nil
	}
	box := c.boxes[boxID]
	cells := make([]point, 0)
	for y := box.Y; y < box.Y+box.Height; y++ {
		for x := box.X; x < box.X+box.Width; x++ {
			cells = append(cells, point{X: x, Y: y})
		}
	}
	return cells
}

// Get all cells that make up a text
func (c *Canvas) GetTextCells(textID int) []point {
	if textID < 0 || textID >= len(c.texts) {
		return nil
	}
	text := c.texts[textID]
	cells := make([]point, 0)
	for lineIdx, line := range text.Lines {
		lineY := text.Y + lineIdx
		for x := text.X; x < text.X+len(line); x++ {
			cells = append(cells, point{X: x, Y: lineY})
		}
	}
	return cells
}

// Get all cells that make up a connection
func (c *Canvas) GetConnectionCells(connIdx int) []point {
	if connIdx < 0 || connIdx >= len(c.connections) {
		return nil
	}
	conn := c.connections[connIdx]
	cells := make([]point, 0)

	// Build path from waypoints
	points := []point{{conn.FromX, conn.FromY}}
	points = append(points, conn.Waypoints...)
	points = append(points, point{conn.ToX, conn.ToY})

	// Add all cells along the path
	for i := 0; i < len(points)-1; i++ {
		from := points[i]
		to := points[i+1]

		// Add cells along the line segment
		if from.X == to.X {
			// Vertical line
			startY, endY := from.Y, to.Y
			if startY > endY {
				startY, endY = endY, startY
			}
			for y := startY; y <= endY; y++ {
				cells = append(cells, point{X: from.X, Y: y})
			}
		} else if from.Y == to.Y {
			// Horizontal line
			startX, endX := from.X, to.X
			if startX > endX {
				startX, endX = endX, startX
			}
			for x := startX; x <= endX; x++ {
				cells = append(cells, point{X: x, Y: from.Y})
			}
		} else {
			// Diagonal line - use L-shaped path
			cornerX := to.X
			cornerY := from.Y

			// Horizontal segment
			startX, endX := from.X, cornerX
			if startX > endX {
				startX, endX = endX, startX
			}
			for x := startX; x <= endX; x++ {
				cells = append(cells, point{X: x, Y: from.Y})
			}

			// Vertical segment
			startY, endY := cornerY, to.Y
			if startY > endY {
				startY, endY = endY, startY
			}
			for y := startY; y <= endY; y++ {
				cells = append(cells, point{X: cornerX, Y: y})
			}
		}
	}

	return cells
}

// Get all adjacent highlighted cells of the same color (flood fill)
func (c *Canvas) GetAdjacentHighlightsOfColor(startX, startY int, targetColor int) []point {
	if targetColor < 0 || targetColor >= numColors {
		return nil
	}

	visited := make(map[string]bool)
	queue := []point{{startX, startY}}
	result := make([]point, 0)

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		key := fmt.Sprintf("%d,%d", current.X, current.Y)
		if visited[key] {
			continue
		}

		// Check if this cell has the target color
		if c.GetHighlight(current.X, current.Y) != targetColor {
			continue
		}

		visited[key] = true
		result = append(result, current)

		// Check all 4 adjacent cells (up, down, left, right)
		adjacent := []point{
			{current.X, current.Y - 1}, // up
			{current.X, current.Y + 1}, // down
			{current.X - 1, current.Y}, // left
			{current.X + 1, current.Y}, // right
		}

		for _, adj := range adjacent {
			adjKey := fmt.Sprintf("%d,%d", adj.X, adj.Y)
			if !visited[adjKey] {
				queue = append(queue, adj)
			}
		}
	}

	return result
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

	fmt.Fprintf(file, "HIGHLIGHTS:%d\n", len(c.highlights))
	for key, colorIndex := range c.highlights {
		var x, y int
		fmt.Sscanf(key, "%d,%d", &x, &y)
		fmt.Fprintf(file, "%d,%d,%d\n", x, y, colorIndex)
	}

	return nil
}

// SaveToFileWithPan saves the canvas and pan position to a file
func (c *Canvas) SaveToFileWithPan(filename string, panX, panY int) error {
	err := c.SaveToFile(filename)
	if err != nil {
		return err
	}

	// Append pan position to the file
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Fprintf(file, "PAN:%d,%d\n", panX, panY)
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
	c.highlights = make(map[string]int)

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

	// Read highlights (optional, for backward compatibility)
	if scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "HIGHLIGHTS:") {
			highlightCountStr := strings.TrimPrefix(line, "HIGHLIGHTS:")
			highlightCount, err := strconv.Atoi(highlightCountStr)
			if err == nil {
				for i := 0; i < highlightCount; i++ {
					if !scanner.Scan() {
						break
					}
					parts := strings.Split(scanner.Text(), ",")
					if len(parts) >= 3 {
						x, _ := strconv.Atoi(parts[0])
						y, _ := strconv.Atoi(parts[1])
						colorIndex, _ := strconv.Atoi(parts[2])
						if colorIndex >= 0 && colorIndex < numColors {
							c.SetHighlight(x, y, colorIndex)
						}
					}
				}
			}
		}
		// If it wasn't HIGHLIGHTS, we've already scanned past it, so we're done
	}

	return scanner.Err()
}

// LoadFromFileWithPan loads the canvas and pan position from a file
// Returns the canvas, panX, panY, and any error
func (c *Canvas) LoadFromFileWithPan(filename string) (int, int, error) {
	err := c.LoadFromFile(filename)
	if err != nil {
		return 0, 0, err
	}

	// Try to read pan position from file
	file, err := os.Open(filename)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	panX, panY := 0, 0

	// Scan through file to find PAN line
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "PAN:") {
			panStr := strings.TrimPrefix(line, "PAN:")
			parts := strings.Split(panStr, ",")
			if len(parts) >= 2 {
				panX, _ = strconv.Atoi(parts[0])
				panY, _ = strconv.Atoi(parts[1])
			}
			break
		}
	}

	return panX, panY, scanner.Err()
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
