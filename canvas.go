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
	maxWidth := 8 // minimum width
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
	Waypoints []struct{ X, Y int }
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
		points := []struct{ X, Y int }{
			{conn.FromX, conn.FromY},
		}
		points = append(points, conn.Waypoints...)
		points = append(points, struct{ X, Y int }{conn.ToX, conn.ToY})

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

func (c *Canvas) AddConnectionWithWaypoints(fromID, toID, fromX, fromY, toX, toY int, waypoints []struct{ X, Y int }) {
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
		minWidth := 8
		minHeight := 3

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
		box.X = x
		box.Y = y

		// Ensure box doesn't go negative
		if box.X < 0 {
			box.X = 0
		}
		if box.Y < 0 {
			box.Y = 0
		}
	}
}

func (c *Canvas) SetBoxSize(id int, width, height int) {
	if id >= 0 && id < len(c.boxes) {
		box := &c.boxes[id]

		// Set minimum size constraints
		minWidth := 8
		minHeight := 3

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
	}
}

func (c *Canvas) Render(width, height int, selectedBox int, previewFromX, previewFromY int, previewWaypoints []struct{ X, Y int }, previewToX, previewToY int) []string {
	// Ensure minimum dimensions
	if height < 1 {
		height = 1
	}
	if width < 1 {
		width = 1
	}

	canvas := make([][]rune, height)
	for i := range canvas {
		canvas[i] = make([]rune, width)
		for j := range canvas[i] {
			canvas[i][j] = ' '
		}
	}

	// Draw connections first (so they appear behind boxes)
	for _, connection := range c.connections {
		c.drawConnection(canvas, connection)
	}

	// Draw preview connection if in progress
	if previewFromX >= 0 && previewFromY >= 0 {
		previewConnection := Connection{
			FromID:    -1,
			ToID:      -1,
			FromX:     previewFromX,
			FromY:     previewFromY,
			ToX:       previewToX,
			ToY:       previewToY,
			Waypoints: previewWaypoints,
		}
		c.drawConnection(canvas, previewConnection)
	}

	// Draw texts first (plain text without borders)
	for _, text := range c.texts {
		c.drawText(canvas, text)
	}

	// Draw boxes
	for i, box := range c.boxes {
		isSelected := (i == selectedBox)
		c.drawBox(canvas, box, isSelected)
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

func (c *Canvas) drawBox(canvas [][]rune, box Box, isSelected bool) {
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
	for y := box.Y; y < box.Y+box.Height && y < len(canvas) && y >= 0; y++ {
		if y >= len(canvas) {
			break
		}
		for x := box.X; x < box.X+box.Width && x < len(canvas[y]) && x >= 0; x++ {
			if y == box.Y || y == box.Y+box.Height-1 {
				// Top and bottom borders
				if x == box.X || x == box.X+box.Width-1 {
					// Corners
					canvas[y][x] = corner
				} else {
					canvas[y][x] = horizontal
				}
			} else if x == box.X || x == box.X+box.Width-1 {
				// Left and right borders
				canvas[y][x] = vertical
			}
		}
	}

	// Draw multi-line text inside box with bounds checking
	for lineIdx, line := range box.Lines {
		textY := box.Y + 1 + lineIdx
		textX := box.X + 1

		if textY >= 0 && textY < len(canvas) && textY < box.Y+box.Height-1 && textX >= 0 {
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
				if textX+i >= 0 && textX+i < len(canvas[textY]) && textX+i < box.X+box.Width-1 {
					canvas[textY][textX+i] = char
				}
			}
		}
	}
}

func (c *Canvas) drawText(canvas [][]rune, text Text) {
	// Draw multi-line text directly without borders
	for lineIdx, line := range text.Lines {
		textY := text.Y + lineIdx
		textX := text.X

		if textY >= 0 && textY < len(canvas) && textX >= 0 {
			for i, char := range line {
				if textX+i >= 0 && textX+i < len(canvas[textY]) {
					canvas[textY][textX+i] = char
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

func (c *Canvas) drawLineSegment(canvas [][]rune, fromX, fromY, toX, toY int, excludeFromID, excludeToID int, drawArrow bool) {
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

		if drawArrow && c.isValidPos(canvas, toX, arrowY) {
			canvas[arrowY][toX] = arrowChar
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

		if drawArrow {
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

		for x := hStartX; x <= hEndX; x++ {
			if c.isValidPos(canvas, x, fromY) && !c.isPointInBox(x, fromY, excludeFromID, excludeToID) {
				canvas[fromY][x] = '─'
			}
		}

		var vStartY, vEndY int
		if cornerY < toY {
			vStartY = cornerY + 1
			vEndY = toY
		} else {
			vStartY = toY
			vEndY = cornerY - 1
		}

		for y := vStartY; y <= vEndY; y++ {
			if c.isValidPos(canvas, cornerX, y) && !c.isPointInBox(cornerX, y, excludeFromID, excludeToID) {
				canvas[y][cornerX] = '│'
			}
		}

		if c.isValidPos(canvas, cornerX, cornerY) && !c.isPointInBox(cornerX, cornerY, excludeFromID, excludeToID) {
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

		if drawArrow {
			var arrowX, arrowY int
			var arrowChar rune

			var onLeftEdge, onRightEdge, onTopEdge, onBottomEdge bool
			if excludeToID >= 0 && excludeToID < len(c.boxes) {
				toBox := c.boxes[excludeToID]
				onLeftEdge = (toX == toBox.X)
				onRightEdge = (toX == toBox.X+toBox.Width-1)
				onTopEdge = (toY == toBox.Y)
				onBottomEdge = (toY == toBox.Y+toBox.Height-1)
			}

			if cornerY < toY {
				if onLeftEdge {
					arrowX = toX - 1
					arrowY = toY
					arrowChar = '◀'
				} else if onRightEdge {
					arrowX = toX + 1
					arrowY = toY
					arrowChar = '▶'
				} else {
					arrowX = toX
					arrowY = toY - 1
					arrowChar = '▼'
				}
			} else if cornerY > toY {
				if onLeftEdge {
					arrowX = toX - 1
					arrowY = toY
					arrowChar = '◀'
				} else if onRightEdge {
					arrowX = toX + 1
					arrowY = toY
					arrowChar = '▶'
				} else {
					arrowX = toX
					arrowY = toY + 1
					arrowChar = '▲'
				}
			} else {
				if onTopEdge {
					arrowX = toX
					arrowY = toY - 1
					arrowChar = '▼'
				} else if onBottomEdge {
					arrowX = toX
					arrowY = toY + 1
					arrowChar = '▲'
				} else if onLeftEdge {
					arrowX = toX - 1
					arrowY = toY
					arrowChar = '◀'
				} else if onRightEdge {
					arrowX = toX + 1
					arrowY = toY
					arrowChar = '▶'
				} else {
					arrowX = toX
					arrowY = toY - 1
					arrowChar = '▼'
				}
			}

			if excludeToID >= 0 && excludeToID < len(c.boxes) {
				toBox := c.boxes[excludeToID]
				if arrowX >= toBox.X && arrowX < toBox.X+toBox.Width &&
					arrowY >= toBox.Y && arrowY < toBox.Y+toBox.Height {
					if onLeftEdge {
						arrowX = toBox.X - 1
					} else if onRightEdge {
						arrowX = toBox.X + toBox.Width
					} else if onTopEdge {
						arrowY = toBox.Y - 1
					} else if onBottomEdge {
						arrowY = toBox.Y + toBox.Height
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

func (c *Canvas) drawConnection(canvas [][]rune, connection Connection) {
	fromX, fromY := connection.FromX, connection.FromY
	toX, toY := connection.ToX, connection.ToY

	if len(connection.Waypoints) > 0 {
		prevX, prevY := fromX, fromY
		for i, waypoint := range connection.Waypoints {
			c.drawLineSegment(canvas, prevX, prevY, waypoint.X, waypoint.Y, connection.FromID, connection.ToID, false)

			if i > 0 {
				prevWaypoint := connection.Waypoints[i-1]
				var nextX, nextY int
				if i < len(connection.Waypoints)-1 {
					nextWaypoint := connection.Waypoints[i+1]
					nextX, nextY = nextWaypoint.X, nextWaypoint.Y
				} else {
					nextX, nextY = toX, toY
				}
				c.drawCorner(canvas, waypoint.X, waypoint.Y, prevWaypoint.X, prevWaypoint.Y, nextX, nextY, connection.FromID, connection.ToID)
			} else {
				var nextX, nextY int
				if i < len(connection.Waypoints)-1 {
					nextWaypoint := connection.Waypoints[i+1]
					nextX, nextY = nextWaypoint.X, nextWaypoint.Y
				} else {
					nextX, nextY = toX, toY
				}
				c.drawCorner(canvas, waypoint.X, waypoint.Y, fromX, fromY, nextX, nextY, connection.FromID, connection.ToID)
			}

			prevX, prevY = waypoint.X, waypoint.Y
		}
		if len(connection.Waypoints) > 0 {
			lastWaypoint := connection.Waypoints[len(connection.Waypoints)-1]
			c.drawCorner(canvas, lastWaypoint.X, lastWaypoint.Y, prevX, prevY, toX, toY, connection.FromID, connection.ToID)
		}
		c.drawLineSegment(canvas, prevX, prevY, toX, toY, connection.FromID, connection.ToID, true)
	} else {
		c.drawLineSegment(canvas, fromX, fromY, toX, toY, connection.FromID, connection.ToID, true)
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
		fmt.Fprintf(file, "%d,%d,%d,%d,%d,%d,%d%s\n",
			connection.FromID, connection.ToID,
			connection.FromX, connection.FromY,
			connection.ToX, connection.ToY,
			len(connection.Waypoints), waypointsStr)
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

		if len(parts) >= 5 {
			width, _ = strconv.Atoi(parts[2])
			height, _ = strconv.Atoi(parts[3])
			text = strings.ReplaceAll(strings.Join(parts[4:], ","), "\\n", "\n")
		} else {
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

			var waypoints []struct{ X, Y int }
			if len(parts) > 1 && waypointCount > 0 {
				waypointParts := strings.Split(parts[1], ",")
				for j := 0; j < waypointCount && j < len(waypointParts); j++ {
					wpParts := strings.Split(waypointParts[j], ":")
					if len(wpParts) == 2 {
						wpX, _ := strconv.Atoi(wpParts[0])
						wpY, _ := strconv.Atoi(wpParts[1])
						waypoints = append(waypoints, struct{ X, Y int }{wpX, wpY})
					}
				}
			}

			c.AddConnectionWithWaypoints(fromID, toID, fromX, fromY, toX, toY, waypoints)
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
					x, _ := strconv.Atoi(parts[0])
					y, _ := strconv.Atoi(parts[1])
					text := strings.ReplaceAll(strings.Join(parts[2:], ","), "\\n", "\n")
					c.AddText(x, y, text)
				}
			}
		}
	}

	return scanner.Err()
}

func (c *Canvas) ExportToPNG(filename string, width, height int) error {
	dc := gg.NewContext(width, height)
	dc.SetColor(color.White)
	dc.Clear()
	dc.SetColor(color.Black)
	dc.SetLineWidth(2)

	// Draw boxes
	for _, box := range c.boxes {
		x := float64(box.X * 10)
		y := float64(box.Y * 10)
		w := float64(box.Width * 10)
		h := float64(box.Height * 10)

		dc.DrawRectangle(x, y, w, h)
		dc.Stroke()

		// Draw multi-line text
		for i, line := range box.Lines {
			lineY := y + 15 + float64(i*15) // 15 pixels per line
			dc.DrawString(line, x+5, lineY)
		}
	}

	// Draw connections
	for _, connection := range c.connections {
		fromX := float64(connection.FromX * 10)
		fromY := float64(connection.FromY * 10)
		toX := float64(connection.ToX * 10)
		toY := float64(connection.ToY * 10)

		dc.DrawLine(fromX, fromY, toX, toY)
		dc.Stroke()

		// Draw connection head
		angle := math.Atan2(toY-fromY, toX-fromX)
		arrowLength := 10.0
		arrowAngle := 0.5

		dc.DrawLine(toX, toY,
			toX-arrowLength*math.Cos(angle-arrowAngle),
			toY-arrowLength*math.Sin(angle-arrowAngle))
		dc.DrawLine(toX, toY,
			toX-arrowLength*math.Cos(angle+arrowAngle),
			toY-arrowLength*math.Sin(angle+arrowAngle))
		dc.Stroke()
	}

	return dc.SavePNG(filename)
}
