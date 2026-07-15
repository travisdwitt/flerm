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
	highlights  map[string]int
}

type Text struct {
	X     int
	Y     int
	Lines []string
	ID    int
	Color int // palette index 0-7, or -1 for none
}

func (t *Text) GetText() string {
	return strings.Join(t.Lines, "\n")
}

func (t *Text) SetText(text string) {
	t.Lines = strings.Split(text, "\n")
}

type Box struct {
	X            int
	Y            int
	Width        int
	Height       int
	Lines        []string
	ID           int
	ZLevel       int
	BorderStyle  BorderStyle
	OriginalText string
	Title        string
	Color        int // palette index 0-7, or -1 for none
}

func (b *Box) GetText() string {
	if b.OriginalText != "" {
		return b.OriginalText
	}
	return strings.Join(b.Lines, "\n")
}

func (b *Box) SetText(text string) {
	b.OriginalText = text
	b.Lines = strings.Split(text, "\n")
	b.updateSize()
}

func (b *Box) updateSize() {
	if len(b.Lines) == 0 {
		b.Lines = []string{""}
	}

	maxWidth := minBoxWidth

	// Consider title width (handle multiline titles)
	titleLines := []string{}
	if b.Title != "" {
		titleLines = strings.Split(b.Title, "\n")
		for _, titleLine := range titleLines {
			titleWidth := len(titleLine) + 2
			if titleWidth > maxWidth {
				maxWidth = titleWidth
			}
		}
	}

	// Consider content width
	for _, line := range b.Lines {
		if len(line)+2 > maxWidth {
			maxWidth = len(line) + 2
		}
	}
	b.Width = maxWidth

	// Add extra height for title and divider
	extraHeight := 0
	if b.Title != "" {
		extraHeight = len(titleLines) + 1 // N lines for title, 1 for divider line
	}
	b.Height = len(b.Lines) + 2 + extraHeight
}

func (b *Box) getMinimumWidth() int {
	minWidth := minBoxWidth
	text := b.GetText()
	if text == "" {
		return minWidth
	}

	// Find the longest line to determine minimum width
	lines := strings.Split(text, "\n")
	longestLine := 0
	for _, line := range lines {
		if len(line) > longestLine {
			longestLine = len(line)
		}
	}

	// Also find the longest word for absolute minimum
	words := strings.Fields(text)
	longestWord := 0
	for _, word := range words {
		if len(word) > longestWord {
			longestWord = len(word)
		}
	}

	// Use the shorter of longest line or longest word for minimum
	requiredWidth := longestWord
	if requiredWidth < 4 { // Minimum to show "..."
		requiredWidth = 4
	}

	if requiredWidth+2 > minWidth {
		minWidth = requiredWidth + 2
	}

	return minWidth
}

func (b *Box) getMinimumHeight() int {
	text := b.GetText()
	if text == "" {
		return minBoxHeight
	}

	// Minimum height should be enough to show at least one line plus ellipsis
	// This allows for more flexible resizing
	minHeight := minBoxHeight

	// If we have text, ensure we can show at least one character
	if text != "" && minHeight < 3 {
		minHeight = 3
	}

	return minHeight
}

func (b *Box) wrapTextToWidth(text string, width int) []string {
	if width < 1 {
		return []string{text}
	}

	var result []string
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}

	var currentLine strings.Builder
	for _, word := range words {
		// If this is the first word in the line
		if currentLine.Len() == 0 {
			currentLine.WriteString(word)
		} else {
			// Check if adding this word would exceed the width
			if currentLine.Len()+1+len(word) <= width {
				currentLine.WriteString(" " + word)
			} else {
				// Start a new line
				result = append(result, currentLine.String())
				currentLine.Reset()
				currentLine.WriteString(word)
			}
		}

		// If the current line is longer than the width, break it
		lineText := currentLine.String()
		for len(lineText) > width {
			result = append(result, lineText[:width])
			lineText = lineText[width:]
		}
		if lineText != currentLine.String() {
			currentLine.Reset()
			currentLine.WriteString(lineText)
		}
	}

	if currentLine.Len() > 0 {
		result = append(result, currentLine.String())
	}

	return result
}

func (b *Box) fitTextToSize(newWidth, newHeight int) {
	text := b.GetText()
	if text == "" {
		b.Lines = []string{""}
		return
	}

	contentWidth := newWidth - 2
	contentHeight := newHeight - 2

	if contentWidth < 1 {
		contentWidth = 1
	}
	if contentHeight < 1 {
		contentHeight = 1
	}

	originalLines := strings.Split(text, "\n")

	// Check if original text fits without modification
	fitsWidth := true
	for _, line := range originalLines {
		if len(line) > contentWidth {
			fitsWidth = false
			break
		}
	}

	fitsHeight := len(originalLines) <= contentHeight

	// If text fits naturally, use original layout
	if fitsWidth && fitsHeight {
		b.Lines = originalLines
		return
	}

	// Otherwise, we need to truncate/fit the text
	var resultLines []string

	for i, line := range originalLines {
		if i >= contentHeight {
			// No more room for lines
			break
		}

		if i == contentHeight-1 && (len(originalLines) > contentHeight || !fitsWidth) {
			// This is the last line we can show, add ellipsis
			if len(line) > contentWidth-3 {
				line = line[:contentWidth-3] + "..."
			} else if len(originalLines) > contentHeight {
				// There are more lines below
				if len(line)+3 <= contentWidth {
					line = line + "..."
				} else {
					line = line[:contentWidth-3] + "..."
				}
			}
		} else if len(line) > contentWidth {
			// Line is too long, truncate with ellipsis
			if contentWidth > 3 {
				line = line[:contentWidth-3] + "..."
			} else {
				line = line[:contentWidth]
			}
		}

		resultLines = append(resultLines, line)
	}

	if len(resultLines) == 0 {
		resultLines = []string{""}
	}

	b.Lines = resultLines
}

func (b *Box) isTextTruncated() bool {
	if b.OriginalText == "" {
		return false
	}

	originalLines := strings.Split(b.OriginalText, "\n")

	// Check if we have the same number of lines
	if len(originalLines) != len(b.Lines) {
		return true
	}

	// Check if any line has been truncated (contains ...)
	for _, line := range b.Lines {
		if strings.HasSuffix(line, "...") {
			return true
		}
	}

	return false
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
	Color     int // palette index 0-7, or -1 for none
}

func NewCanvas() *Canvas {
	return &Canvas{
		boxes:       make([]Box, 0),
		connections: make([]Connection, 0),
		texts:       make([]Text, 0),
		highlights:  make(map[string]int),
	}
}

// GetFullBounds returns the bounding box of all content on the canvas
// Returns minX, minY, maxX, maxY
func (c *Canvas) GetFullBounds() (int, int, int, int) {
	minX, minY := 0, 0
	maxX, maxY := 0, 0
	hasElements := false

	// Check all boxes
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

	// Check all connections
	for _, conn := range c.connections {
		points := []point{{X: conn.FromX, Y: conn.FromY}}
		points = append(points, conn.Waypoints...)
		points = append(points, point{X: conn.ToX, Y: conn.ToY})

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

	// Check all text objects
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
		// Calculate text extent
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
		return 1, 1, 0, 0 // Return invalid bounds to indicate empty canvas
	}

	return minX, minY, maxX, maxY
}

func (c *Canvas) AddBox(x, y int, text string) {
	box := Box{
		X:     x,
		Y:     y,
		ID:    len(c.boxes),
		Color: -1,
	}
	box.SetText(text)
	c.boxes = append(c.boxes, box)
}

func (c *Canvas) AddText(x, y int, text string) {
	textObj := Text{
		X:     x,
		Y:     y,
		ID:    len(c.texts),
		Color: -1,
	}
	textObj.SetText(text)
	c.texts = append(c.texts, textObj)
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
		Color:  -1,
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
		Color:     -1,
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

		// Use only the basic minimum constraints
		if newWidth < minBoxWidth {
			newWidth = minBoxWidth
		}
		if newHeight < minBoxHeight {
			newHeight = minBoxHeight
		}

		// Fit text to the new size
		if newWidth != box.Width || newHeight != box.Height {
			box.fitTextToSize(newWidth, newHeight)
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
		box := &c.boxes[id]
		oldX, oldY := box.X, box.Y
		box.X += deltaX
		box.Y += deltaY
		if box.X < 0 {
			box.X = 0
		}
		if box.Y < 0 {
			box.Y = 0
		}

		c.rerouteConnectionsForMovedBox(id, box.X-oldX, box.Y-oldY)
	}
}

// connectionPathInfo stores information about a connection's old path for branch updates
type connectionPathInfo struct {
	connIdx int
	points  []point
}

// updateBranchConnections moves line-point endpoints that sat on a moved
// connection's old path onto its new path. If the attachment point didn't move
// it's a shared junction, not a branch, so the line is left alone.
func (c *Canvas) updateBranchConnections(updatedConnections []connectionPathInfo) {
	for i := range c.connections {
		conn := &c.connections[i]

		if conn.FromID < 0 {
			for _, updated := range updatedConnections {
				if updated.connIdx == i {
					continue
				}
				if c.pointWasOnPath(conn.FromX, conn.FromY, updated.points) {
					updatedConn := c.connections[updated.connIdx]
					newPoints := []point{{X: updatedConn.FromX, Y: updatedConn.FromY}}
					newPoints = append(newPoints, updatedConn.Waypoints...)
					newPoints = append(newPoints, point{X: updatedConn.ToX, Y: updatedConn.ToY})
					newX, newY := c.findNearestPointOnPath(conn.FromX, conn.FromY, newPoints)
					if newX == conn.FromX && newY == conn.FromY {
						break
					}
					if !adjustEndpointKeepingPath(conn, true, newX-conn.FromX, newY-conn.FromY) {
						if conn.ToID >= 0 && conn.ToID < len(c.boxes) {
							toBox := c.boxes[conn.ToID]
							conn.Waypoints = c.createFlexibleWaypointsForLineConnection(conn, nil, &toBox)
						}
					}
					simplifyConnectionPath(conn)
					break
				}
			}
		}

		if conn.ToID < 0 {
			for _, updated := range updatedConnections {
				if updated.connIdx == i {
					continue
				}
				if c.pointWasOnPath(conn.ToX, conn.ToY, updated.points) {
					updatedConn := c.connections[updated.connIdx]
					newPoints := []point{{X: updatedConn.FromX, Y: updatedConn.FromY}}
					newPoints = append(newPoints, updatedConn.Waypoints...)
					newPoints = append(newPoints, point{X: updatedConn.ToX, Y: updatedConn.ToY})
					newX, newY := c.findNearestPointOnPath(conn.ToX, conn.ToY, newPoints)
					if newX == conn.ToX && newY == conn.ToY {
						break
					}
					if !adjustEndpointKeepingPath(conn, false, newX-conn.ToX, newY-conn.ToY) {
						if conn.FromID >= 0 && conn.FromID < len(c.boxes) {
							fromBox := c.boxes[conn.FromID]
							conn.Waypoints = c.createFlexibleWaypointsForLineConnection(conn, &fromBox, nil)
						}
					}
					simplifyConnectionPath(conn)
					break
				}
			}
		}
	}
}

// pointWasOnPath checks if a point was on (or very near) a connection path
func (c *Canvas) pointWasOnPath(x, y int, pathPoints []point) bool {
	for i := 0; i < len(pathPoints)-1; i++ {
		if c.pointOnSegment(x, y, pathPoints[i].X, pathPoints[i].Y, pathPoints[i+1].X, pathPoints[i+1].Y) {
			return true
		}
	}
	return false
}

// pointOnSegment checks if a point is on (or very near) a line segment
func (c *Canvas) pointOnSegment(px, py, x1, y1, x2, y2 int) bool {
	// Check if point is within bounding box of segment (with tolerance)
	minX, maxX := min(x1, x2), max(x1, x2)
	minY, maxY := min(y1, y2), max(y1, y2)

	tolerance := 2
	if px < minX-tolerance || px > maxX+tolerance || py < minY-tolerance || py > maxY+tolerance {
		return false
	}

	// For horizontal segment
	if y1 == y2 {
		return abs(py-y1) <= tolerance && px >= minX-tolerance && px <= maxX+tolerance
	}

	// For vertical segment
	if x1 == x2 {
		return abs(px-x1) <= tolerance && py >= minY-tolerance && py <= maxY+tolerance
	}

	// For diagonal segments (rare in this app), use distance to line
	// Simplified: just check if point is close to either endpoint
	dist1 := abs(px-x1) + abs(py-y1)
	dist2 := abs(px-x2) + abs(py-y2)
	return dist1 <= tolerance*2 || dist2 <= tolerance*2
}

// findNearestPointOnPath finds the nearest point on a path to the given point
func (c *Canvas) findNearestPointOnPath(x, y int, pathPoints []point) (int, int) {
	bestX, bestY := x, y
	bestDist := -1

	for i := 0; i < len(pathPoints)-1; i++ {
		segX, segY := c.findClosestPointOnSegment(
			pathPoints[i].X, pathPoints[i].Y,
			pathPoints[i+1].X, pathPoints[i+1].Y,
			x, y,
		)
		dist := abs(segX-x) + abs(segY-y)
		if bestDist == -1 || dist < bestDist {
			bestDist = dist
			bestX, bestY = segX, segY
		}
	}

	return bestX, bestY
}

// recalculateConnectionRoute recalculates connection endpoints and creates clean routing
func (c *Canvas) recalculateConnectionRoute(conn *Connection) {
	if conn.FromID < 0 || conn.FromID >= len(c.boxes) ||
		conn.ToID < 0 || conn.ToID >= len(c.boxes) {
		return
	}

	// Calculate optimal endpoints based on box positions
	conn.FromX, conn.FromY, conn.ToX, conn.ToY = c.calculateConnectionPoints(conn.FromID, conn.ToID)

	// Create clean routing waypoints
	conn.Waypoints = c.createSmartWaypoints(conn)
}

// rerouteConnectionsForMovedBox re-routes connections touching box id after it
// moved by (deltaX, deltaY), which must be the box's ACTUAL (post-clamp) motion
// so anchors stay on the edges. Anchored endpoints translate with the box,
// keeping the existing path; an anchor is only re-picked when its edge stops
// facing the path (the box was dragged past it).
func (c *Canvas) rerouteConnectionsForMovedBox(id, deltaX, deltaY int) {
	if id < 0 || id >= len(c.boxes) || (deltaX == 0 && deltaY == 0) {
		return
	}
	box := &c.boxes[id]

	var updatedConnections []connectionPathInfo

	for i := range c.connections {
		conn := &c.connections[i]

		isFromThisBox := conn.FromID == id
		isToThisBox := conn.ToID == id
		if !isFromThisBox && !isToThisBox {
			continue
		}

		// Old path is captured before mutating so branch points can follow it.
		oldPoints := []point{{X: conn.FromX, Y: conn.FromY}}
		oldPoints = append(oldPoints, conn.Waypoints...)
		oldPoints = append(oldPoints, point{X: conn.ToX, Y: conn.ToY})
		updatedConnections = append(updatedConnections, connectionPathInfo{connIdx: i, points: oldPoints})

		fromIsValidBox := conn.FromID >= 0 && conn.FromID < len(c.boxes)
		toIsValidBox := conn.ToID >= 0 && conn.ToID < len(c.boxes)

		// Face the ADJACENT waypoint, not the far endpoint: a line can leave its
		// far end one way yet approach this box from another via its waypoints.
		regenerate := false
		if isFromThisBox {
			refX, refY := conn.ToX, conn.ToY
			if len(conn.Waypoints) > 0 {
				refX, refY = conn.Waypoints[0].X, conn.Waypoints[0].Y
			}
			if c.anchorFacesTarget(*box, conn.FromX+deltaX, conn.FromY+deltaY, refX, refY) {
				if !adjustEndpointKeepingPath(conn, true, deltaX, deltaY) {
					regenerate = true
				}
			} else {
				conn.FromX, conn.FromY = c.reanchor(*box, conn.FromX+deltaX, conn.FromY+deltaY, refX, refY)
				regenerate = true
			}
		} else { // isToThisBox
			refX, refY := conn.FromX, conn.FromY
			if len(conn.Waypoints) > 0 {
				refX, refY = conn.Waypoints[len(conn.Waypoints)-1].X, conn.Waypoints[len(conn.Waypoints)-1].Y
			}
			if c.anchorFacesTarget(*box, conn.ToX+deltaX, conn.ToY+deltaY, refX, refY) {
				if !adjustEndpointKeepingPath(conn, false, deltaX, deltaY) {
					regenerate = true
				}
			} else {
				conn.ToX, conn.ToY = c.reanchor(*box, conn.ToX+deltaX, conn.ToY+deltaY, refX, refY)
				regenerate = true
			}
		}

		if regenerate {
			if fromIsValidBox && toIsValidBox {
				conn.Waypoints = c.createFlexibleWaypoints(conn, c.boxes[conn.FromID], c.boxes[conn.ToID])
			} else if isFromThisBox {
				conn.Waypoints = c.createFlexibleWaypointsForLineConnection(conn, box, nil)
			} else {
				conn.Waypoints = c.createFlexibleWaypointsForLineConnection(conn, nil, box)
			}
		}
		simplifyConnectionPath(conn)
	}

	c.updateBranchConnections(updatedConnections)
}

// reanchor picks a new anchor when the box moved so the old edge no longer faces
// the target. (ax, ay) is the old anchor translated with the box. For a straight
// flip to the opposite edge on the same axis it keeps the anchor's offset along
// the edge (so e.g. a mid-edge anchor stays mid-edge); otherwise it falls back to
// findBestAnchorPoint.
func (c *Canvas) reanchor(box Box, ax, ay, targetX, targetY int) (int, int) {
	switch c.getConnectionEdge(box, ax, ay) {
	case "left", "right":
		if targetX <= box.X {
			return box.X, ay
		}
		if targetX >= box.X+box.Width-1 {
			return box.X + box.Width - 1, ay
		}
	case "top", "bottom":
		if targetY <= box.Y {
			return ax, box.Y
		}
		if targetY >= box.Y+box.Height-1 {
			return ax, box.Y + box.Height - 1
		}
	}
	return c.findBestAnchorPoint(box, targetX, targetY)
}

// anchorFacesTarget reports whether the box edge the anchor sits on still points
// toward the target (if not, the anchor should be re-picked, not translated).
func (c *Canvas) anchorFacesTarget(box Box, anchorX, anchorY, targetX, targetY int) bool {
	switch c.getConnectionEdge(box, anchorX, anchorY) {
	case "right":
		return targetX >= box.X+box.Width-1
	case "left":
		return targetX <= box.X
	case "top":
		return targetY <= box.Y
	case "bottom":
		return targetY >= box.Y+box.Height-1
	default:
		return false
	}
}

// adjustEndpointKeepingPath translates the moved endpoint by (dx,dy) and keeps
// the rest of the path fixed, stretching only the segment to the neighbouring
// waypoint (so a shared trunk and its branches stay put). Returns false when the
// path can't be preserved (no waypoints, or the adjacent segment wasn't
// axis-aligned); the anchor is still translated and the caller should regenerate.
func adjustEndpointKeepingPath(conn *Connection, movingFrom bool, dx, dy int) bool {
	if movingFrom {
		oldX, oldY := conn.FromX, conn.FromY
		conn.FromX += dx
		conn.FromY += dy
		if len(conn.Waypoints) == 0 {
			return false
		}
		w := &conn.Waypoints[0]
		switch {
		case w.Y == oldY:
			w.Y = conn.FromY
		case w.X == oldX:
			w.X = conn.FromX
		default:
			return false
		}
		return true
	}

	oldX, oldY := conn.ToX, conn.ToY
	conn.ToX += dx
	conn.ToY += dy
	if len(conn.Waypoints) == 0 {
		return false
	}
	w := &conn.Waypoints[len(conn.Waypoints)-1]
	switch {
	case w.Y == oldY:
		w.Y = conn.ToY
	case w.X == oldX:
		w.X = conn.ToX
	default:
		return false
	}
	return true
}

// simplifyConnectionPath drops redundant waypoints (duplicates, and points
// collinear with their neighbours) that would otherwise render as a stray
// junction character on a straightened line.
func simplifyConnectionPath(conn *Connection) {
	if len(conn.Waypoints) == 0 {
		return
	}
	pts := make([]point, 0, len(conn.Waypoints)+2)
	pts = append(pts, point{X: conn.FromX, Y: conn.FromY})
	pts = append(pts, conn.Waypoints...)
	pts = append(pts, point{X: conn.ToX, Y: conn.ToY})

	kept := pts[:1]
	for i := 1; i < len(pts)-1; i++ {
		prev := kept[len(kept)-1]
		cur := pts[i]
		next := pts[i+1]
		if cur == prev {
			continue
		}
		if (prev.X == cur.X && cur.X == next.X) || (prev.Y == cur.Y && cur.Y == next.Y) {
			continue
		}
		kept = append(kept, cur)
	}
	if end := pts[len(pts)-1]; end != kept[len(kept)-1] {
		kept = append(kept, end)
	}

	if len(kept) <= 2 {
		conn.Waypoints = nil
	} else {
		conn.Waypoints = append([]point(nil), kept[1:len(kept)-1]...)
	}
}

// findBestAnchorPoint finds the best anchor point on a box to connect to a target point
// It chooses the edge that creates the cleanest orthogonal path
func (c *Canvas) findBestAnchorPoint(box Box, targetX, targetY int) (int, int) {
	boxCenterX := box.X + box.Width/2
	boxCenterY := box.Y + box.Height/2

	// Determine which edge faces the target point most directly
	// We want an edge that allows a clean orthogonal connection

	// Calculate the direction to the target
	dx := targetX - boxCenterX
	dy := targetY - boxCenterY

	// Determine primary direction
	if abs(dx) > abs(dy) {
		// Horizontal is primary
		if dx > 0 {
			// Target is to the right - use right edge
			// Y position: try to align with target Y if possible, otherwise use center
			y := targetY
			if y < box.Y {
				y = box.Y
			} else if y >= box.Y+box.Height {
				y = box.Y + box.Height - 1
			}
			return box.X + box.Width - 1, y
		} else {
			// Target is to the left - use left edge
			y := targetY
			if y < box.Y {
				y = box.Y
			} else if y >= box.Y+box.Height {
				y = box.Y + box.Height - 1
			}
			return box.X, y
		}
	} else {
		// Vertical is primary
		if dy > 0 {
			// Target is below - use bottom edge
			x := targetX
			if x < box.X {
				x = box.X
			} else if x >= box.X+box.Width {
				x = box.X + box.Width - 1
			}
			return x, box.Y + box.Height - 1
		} else {
			// Target is above - use top edge
			x := targetX
			if x < box.X {
				x = box.X
			} else if x >= box.X+box.Width {
				x = box.X + box.Width - 1
			}
			return x, box.Y
		}
	}
}

// createFlexibleWaypoints creates waypoints that handle any anchor/box configuration
func (c *Canvas) createFlexibleWaypoints(conn *Connection, fromBox, toBox Box) []point {
	// If endpoints are aligned (straight line), no waypoints needed
	if conn.FromX == conn.ToX || conn.FromY == conn.ToY {
		return nil
	}

	fromEdge := c.getConnectionEdge(fromBox, conn.FromX, conn.FromY)
	toEdge := c.getConnectionEdge(toBox, conn.ToX, conn.ToY)

	// Calculate midpoints and offsets for routing
	// We want to route around boxes, not through them

	switch fromEdge {
	case "right":
		switch toEdge {
		case "left":
			// Standard horizontal: go right, then vertical, then to target
			if conn.FromX < conn.ToX {
				// Normal case: fromBox is left of toBox
				midX := (conn.FromX + conn.ToX) / 2
				return []point{{X: midX, Y: conn.FromY}, {X: midX, Y: conn.ToY}}
			} else {
				// Boxes have crossed: need to go around
				// Go right from source, down/up, then left to target
				offsetX := max(conn.FromX, conn.ToX) + 3
				return []point{
					{X: offsetX, Y: conn.FromY},
					{X: offsetX, Y: conn.ToY},
				}
			}
		case "top":
			// Go right, then down to top of target
			if conn.FromX < conn.ToX && conn.FromY < conn.ToY {
				// Target is right and below: L-path
				return []point{{X: conn.ToX, Y: conn.FromY}}
			} else {
				// Need to route around
				offsetX := max(conn.FromX+1, conn.ToX+3)
				offsetY := min(conn.FromY, conn.ToY) - 2
				return []point{
					{X: offsetX, Y: conn.FromY},
					{X: offsetX, Y: offsetY},
					{X: conn.ToX, Y: offsetY},
				}
			}
		case "bottom":
			// Go right, then up to bottom of target
			if conn.FromX < conn.ToX && conn.FromY > conn.ToY {
				// Target is right and above: L-path
				return []point{{X: conn.ToX, Y: conn.FromY}}
			} else {
				// Need to route around
				offsetX := max(conn.FromX+1, conn.ToX+3)
				offsetY := max(conn.FromY, conn.ToY) + 2
				return []point{
					{X: offsetX, Y: conn.FromY},
					{X: offsetX, Y: offsetY},
					{X: conn.ToX, Y: offsetY},
				}
			}
		case "right":
			// Both on right edges - go further right and connect
			offsetX := max(fromBox.X+fromBox.Width, toBox.X+toBox.Width) + 3
			return []point{{X: offsetX, Y: conn.FromY}, {X: offsetX, Y: conn.ToY}}
		default:
			return []point{{X: conn.ToX, Y: conn.FromY}}
		}

	case "left":
		switch toEdge {
		case "right":
			if conn.FromX > conn.ToX {
				// Normal case
				midX := (conn.FromX + conn.ToX) / 2
				return []point{{X: midX, Y: conn.FromY}, {X: midX, Y: conn.ToY}}
			} else {
				// Boxes have crossed
				offsetX := min(conn.FromX, conn.ToX) - 3
				return []point{
					{X: offsetX, Y: conn.FromY},
					{X: offsetX, Y: conn.ToY},
				}
			}
		case "top":
			if conn.FromX > conn.ToX && conn.FromY < conn.ToY {
				return []point{{X: conn.ToX, Y: conn.FromY}}
			} else {
				offsetX := min(conn.FromX-1, conn.ToX-3)
				offsetY := min(conn.FromY, conn.ToY) - 2
				return []point{
					{X: offsetX, Y: conn.FromY},
					{X: offsetX, Y: offsetY},
					{X: conn.ToX, Y: offsetY},
				}
			}
		case "bottom":
			if conn.FromX > conn.ToX && conn.FromY > conn.ToY {
				return []point{{X: conn.ToX, Y: conn.FromY}}
			} else {
				offsetX := min(conn.FromX-1, conn.ToX-3)
				offsetY := max(conn.FromY, conn.ToY) + 2
				return []point{
					{X: offsetX, Y: conn.FromY},
					{X: offsetX, Y: offsetY},
					{X: conn.ToX, Y: offsetY},
				}
			}
		case "left":
			offsetX := min(fromBox.X, toBox.X) - 3
			return []point{{X: offsetX, Y: conn.FromY}, {X: offsetX, Y: conn.ToY}}
		default:
			return []point{{X: conn.ToX, Y: conn.FromY}}
		}

	case "bottom":
		switch toEdge {
		case "top":
			if conn.FromY < conn.ToY {
				// Normal case
				midY := (conn.FromY + conn.ToY) / 2
				return []point{{X: conn.FromX, Y: midY}, {X: conn.ToX, Y: midY}}
			} else {
				// Boxes have crossed
				offsetY := max(conn.FromY, conn.ToY) + 2
				return []point{
					{X: conn.FromX, Y: offsetY},
					{X: conn.ToX, Y: offsetY},
				}
			}
		case "left":
			if conn.FromY < conn.ToY && conn.FromX < conn.ToX {
				return []point{{X: conn.FromX, Y: conn.ToY}}
			} else {
				offsetY := max(conn.FromY+1, conn.ToY+3)
				offsetX := min(conn.FromX, conn.ToX) - 2
				return []point{
					{X: conn.FromX, Y: offsetY},
					{X: offsetX, Y: offsetY},
					{X: offsetX, Y: conn.ToY},
				}
			}
		case "right":
			if conn.FromY < conn.ToY && conn.FromX > conn.ToX {
				return []point{{X: conn.FromX, Y: conn.ToY}}
			} else {
				offsetY := max(conn.FromY+1, conn.ToY+3)
				offsetX := max(conn.FromX, conn.ToX) + 2
				return []point{
					{X: conn.FromX, Y: offsetY},
					{X: offsetX, Y: offsetY},
					{X: offsetX, Y: conn.ToY},
				}
			}
		case "bottom":
			offsetY := max(fromBox.Y+fromBox.Height, toBox.Y+toBox.Height) + 2
			return []point{{X: conn.FromX, Y: offsetY}, {X: conn.ToX, Y: offsetY}}
		default:
			return []point{{X: conn.FromX, Y: conn.ToY}}
		}

	case "top":
		switch toEdge {
		case "bottom":
			if conn.FromY > conn.ToY {
				// Normal case
				midY := (conn.FromY + conn.ToY) / 2
				return []point{{X: conn.FromX, Y: midY}, {X: conn.ToX, Y: midY}}
			} else {
				// Boxes have crossed
				offsetY := min(conn.FromY, conn.ToY) - 2
				return []point{
					{X: conn.FromX, Y: offsetY},
					{X: conn.ToX, Y: offsetY},
				}
			}
		case "left":
			if conn.FromY > conn.ToY && conn.FromX < conn.ToX {
				return []point{{X: conn.FromX, Y: conn.ToY}}
			} else {
				offsetY := min(conn.FromY-1, conn.ToY-3)
				offsetX := min(conn.FromX, conn.ToX) - 2
				return []point{
					{X: conn.FromX, Y: offsetY},
					{X: offsetX, Y: offsetY},
					{X: offsetX, Y: conn.ToY},
				}
			}
		case "right":
			if conn.FromY > conn.ToY && conn.FromX > conn.ToX {
				return []point{{X: conn.FromX, Y: conn.ToY}}
			} else {
				offsetY := min(conn.FromY-1, conn.ToY-3)
				offsetX := max(conn.FromX, conn.ToX) + 2
				return []point{
					{X: conn.FromX, Y: offsetY},
					{X: offsetX, Y: offsetY},
					{X: offsetX, Y: conn.ToY},
				}
			}
		case "top":
			offsetY := min(fromBox.Y, toBox.Y) - 2
			return []point{{X: conn.FromX, Y: offsetY}, {X: conn.ToX, Y: offsetY}}
		default:
			return []point{{X: conn.FromX, Y: conn.ToY}}
		}
	}

	// Fallback: simple L-path
	return []point{{X: conn.ToX, Y: conn.FromY}}
}

// createFlexibleWaypointsForLineConnection creates waypoints for connections involving a line point
// fromBox is set if the connection is FROM a box TO a line point
// toBox is set if the connection is FROM a line point TO a box
func (c *Canvas) createFlexibleWaypointsForLineConnection(conn *Connection, fromBox, toBox *Box) []point {
	// If endpoints are aligned (straight line), no waypoints needed
	if conn.FromX == conn.ToX || conn.FromY == conn.ToY {
		return nil
	}

	// Determine direction and create appropriate routing
	dx := conn.ToX - conn.FromX
	dy := conn.ToY - conn.FromY

	if fromBox != nil {
		// Connection FROM box TO line point
		fromEdge := c.getConnectionEdge(*fromBox, conn.FromX, conn.FromY)

		switch fromEdge {
		case "right":
			if dx > 0 {
				// Target is to the right - simple L-path
				return []point{{X: conn.ToX, Y: conn.FromY}}
			} else {
				// Target is to the left - go right first, then route
				offsetX := conn.FromX + 3
				return []point{{X: offsetX, Y: conn.FromY}, {X: offsetX, Y: conn.ToY}}
			}
		case "left":
			if dx < 0 {
				return []point{{X: conn.ToX, Y: conn.FromY}}
			} else {
				offsetX := conn.FromX - 3
				return []point{{X: offsetX, Y: conn.FromY}, {X: offsetX, Y: conn.ToY}}
			}
		case "bottom":
			if dy > 0 {
				return []point{{X: conn.FromX, Y: conn.ToY}}
			} else {
				offsetY := conn.FromY + 2
				return []point{{X: conn.FromX, Y: offsetY}, {X: conn.ToX, Y: offsetY}}
			}
		case "top":
			if dy < 0 {
				return []point{{X: conn.FromX, Y: conn.ToY}}
			} else {
				offsetY := conn.FromY - 2
				return []point{{X: conn.FromX, Y: offsetY}, {X: conn.ToX, Y: offsetY}}
			}
		}
	} else if toBox != nil {
		// Connection FROM line point TO box
		toEdge := c.getConnectionEdge(*toBox, conn.ToX, conn.ToY)

		switch toEdge {
		case "left":
			if dx > 0 {
				// Coming from the left - simple L-path
				return []point{{X: conn.FromX, Y: conn.ToY}}
			} else {
				// Coming from the right - route around
				return []point{{X: conn.FromX, Y: conn.ToY}}
			}
		case "right":
			if dx < 0 {
				return []point{{X: conn.FromX, Y: conn.ToY}}
			} else {
				return []point{{X: conn.FromX, Y: conn.ToY}}
			}
		case "top":
			if dy > 0 {
				// Coming from above - simple L-path
				return []point{{X: conn.ToX, Y: conn.FromY}}
			} else {
				return []point{{X: conn.ToX, Y: conn.FromY}}
			}
		case "bottom":
			if dy < 0 {
				return []point{{X: conn.ToX, Y: conn.FromY}}
			} else {
				return []point{{X: conn.ToX, Y: conn.FromY}}
			}
		}
	}

	// Fallback: simple L-path
	return []point{{X: conn.ToX, Y: conn.FromY}}
}

// createSmartWaypoints generates waypoints for clean orthogonal routing
func (c *Canvas) createSmartWaypoints(conn *Connection) []point {
	if conn.FromID < 0 || conn.ToID < 0 {
		return nil
	}

	fromBox := c.boxes[conn.FromID]
	toBox := c.boxes[conn.ToID]

	// Determine which edges the connection uses
	fromEdge := c.getConnectionEdge(fromBox, conn.FromX, conn.FromY)
	toEdge := c.getConnectionEdge(toBox, conn.ToX, conn.ToY)

	// If endpoints are aligned (straight line), no waypoints needed
	if conn.FromX == conn.ToX || conn.FromY == conn.ToY {
		return nil
	}

	// Create routing based on edge combination
	switch fromEdge {
	case "right":
		switch toEdge {
		case "left":
			// Horizontal connection - route through middle
			midX := (conn.FromX + conn.ToX) / 2
			return []point{{X: midX, Y: conn.FromY}, {X: midX, Y: conn.ToY}}
		case "top", "bottom":
			// L-path: go horizontal first, then vertical
			return []point{{X: conn.ToX, Y: conn.FromY}}
		case "right":
			// Both on right - go around
			midX := max(fromBox.X+fromBox.Width, toBox.X+toBox.Width) + 3
			return []point{{X: midX, Y: conn.FromY}, {X: midX, Y: conn.ToY}}
		}
	case "left":
		switch toEdge {
		case "right":
			// Horizontal connection - route through middle
			midX := (conn.FromX + conn.ToX) / 2
			return []point{{X: midX, Y: conn.FromY}, {X: midX, Y: conn.ToY}}
		case "top", "bottom":
			// L-path: go horizontal first, then vertical
			return []point{{X: conn.ToX, Y: conn.FromY}}
		case "left":
			// Both on left - go around
			midX := min(fromBox.X, toBox.X) - 3
			return []point{{X: midX, Y: conn.FromY}, {X: midX, Y: conn.ToY}}
		}
	case "bottom":
		switch toEdge {
		case "top":
			// Vertical connection - route through middle
			midY := (conn.FromY + conn.ToY) / 2
			return []point{{X: conn.FromX, Y: midY}, {X: conn.ToX, Y: midY}}
		case "left", "right":
			// L-path: go vertical first, then horizontal
			return []point{{X: conn.FromX, Y: conn.ToY}}
		case "bottom":
			// Both on bottom - go around
			midY := max(fromBox.Y+fromBox.Height, toBox.Y+toBox.Height) + 2
			return []point{{X: conn.FromX, Y: midY}, {X: conn.ToX, Y: midY}}
		}
	case "top":
		switch toEdge {
		case "bottom":
			// Vertical connection - route through middle
			midY := (conn.FromY + conn.ToY) / 2
			return []point{{X: conn.FromX, Y: midY}, {X: conn.ToX, Y: midY}}
		case "left", "right":
			// L-path: go vertical first, then horizontal
			return []point{{X: conn.FromX, Y: conn.ToY}}
		case "top":
			// Both on top - go around
			midY := min(fromBox.Y, toBox.Y) - 2
			return []point{{X: conn.FromX, Y: midY}, {X: conn.ToX, Y: midY}}
		}
	}

	// Fallback: simple L-path
	return []point{{X: conn.ToX, Y: conn.FromY}}
}

// getConnectionEdge determines which edge of a box a connection point is on
func (c *Canvas) getConnectionEdge(box Box, x, y int) string {
	if x == box.X+box.Width-1 {
		return "right"
	} else if x == box.X {
		return "left"
	} else if y == box.Y+box.Height-1 {
		return "bottom"
	} else if y == box.Y {
		return "top"
	}
	return "unknown"
}

func (c *Canvas) CycleBoxZLevel(id int) {
	if id >= 0 && id < len(c.boxes) {
		c.boxes[id].ZLevel = (c.boxes[id].ZLevel + 1) % 4
	}
}

func (c *Canvas) GetBoxZLevel(id int) int {
	if id >= 0 && id < len(c.boxes) {
		return c.boxes[id].ZLevel
	}
	return 0
}

func (c *Canvas) SetBoxPosition(id int, x, y int) {
	if id >= 0 && id < len(c.boxes) {
		oldX, oldY := c.boxes[id].X, c.boxes[id].Y

		box := &c.boxes[id]
		box.X, box.Y = x, y
		if box.X < 0 {
			box.X = 0
		}
		if box.Y < 0 {
			box.Y = 0
		}

		if box.X != oldX || box.Y != oldY {
			c.rerouteConnectionsForMovedBox(id, box.X-oldX, box.Y-oldY)
		}
	}
}

// SetBoxPositionOnly sets a box position without modifying any connections
// Used for undo operations where connections are restored separately
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

// SnapshotConnections deep-copies every connection so a drag can re-derive
// routing from the original state each move (idempotent) instead of accumulating.
func (c *Canvas) SnapshotConnections() []Connection {
	snap := make([]Connection, len(c.connections))
	for i, conn := range c.connections {
		snap[i] = conn
		if len(conn.Waypoints) > 0 {
			snap[i].Waypoints = append([]point(nil), conn.Waypoints...)
		} else {
			snap[i].Waypoints = nil
		}
	}
	return snap
}

// RestoreConnectionsSnapshot restores a SnapshotConnections result by index
// (no-op unless the count matches, which holds during a drag).
func (c *Canvas) RestoreConnectionsSnapshot(snap []Connection) {
	if len(snap) != len(c.connections) {
		return
	}
	for i, conn := range snap {
		c.connections[i] = conn
		if len(conn.Waypoints) > 0 {
			c.connections[i].Waypoints = append([]point(nil), conn.Waypoints...)
		} else {
			c.connections[i].Waypoints = nil
		}
	}
}

// RestoreConnections restores a list of connections to their original states
func (c *Canvas) RestoreConnections(connections []Connection) {
	for _, origConn := range connections {
		// Find the connection by FromID and ToID
		for i := range c.connections {
			if c.connections[i].FromID == origConn.FromID && c.connections[i].ToID == origConn.ToID {
				// Restore all fields
				c.connections[i] = origConn
				break
			}
		}
	}
}

// GetConnectionsForBox returns copies of all connections involving the specified box
func (c *Canvas) GetConnectionsForBox(boxID int) []Connection {
	var result []Connection
	for _, conn := range c.connections {
		if conn.FromID == boxID || conn.ToID == boxID {
			connCopy := Connection{
				FromID:    conn.FromID,
				ToID:      conn.ToID,
				FromX:     conn.FromX,
				FromY:     conn.FromY,
				ToX:       conn.ToX,
				ToY:       conn.ToY,
				ArrowFrom: conn.ArrowFrom,
				ArrowTo:   conn.ArrowTo,
				Waypoints: make([]point, len(conn.Waypoints)),
				Color:     conn.Color,
			}
			copy(connCopy.Waypoints, conn.Waypoints)
			result = append(result, connCopy)
		}
	}
	return result
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

func (c *Canvas) MoveConnection(connIdx int, deltaX, deltaY int) {
	if connIdx < 0 || connIdx >= len(c.connections) {
		return
	}
	conn := &c.connections[connIdx]
	conn.FromX += deltaX
	conn.FromY += deltaY
	conn.ToX += deltaX
	conn.ToY += deltaY
	for i := range conn.Waypoints {
		conn.Waypoints[i].X += deltaX
		conn.Waypoints[i].Y += deltaY
	}
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

func (c *Canvas) MoveConnectionWaypoints(connIdx int, deltaX, deltaY int) {
	if connIdx < 0 || connIdx >= len(c.connections) {
		return
	}
	conn := &c.connections[connIdx]
	for i := range conn.Waypoints {
		conn.Waypoints[i].X += deltaX
		conn.Waypoints[i].Y += deltaY
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

// RenderResult holds the raw canvas data and color information separately
// This allows tooltips and overlays to be applied before ANSI codes are added
type RenderResult struct {
	Canvas   [][]rune
	ColorMap [][]int
	Width    int
	Height   int
}

// ApplyColors converts the raw canvas and colorMap into ANSI-colored strings
// This should be called AFTER all overlays (tooltips, etc.) have been applied
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

// RenderRaw returns the raw canvas and colorMap without applying ANSI codes
// Use this when you need to overlay content (like tooltips) before final rendering
func (c *Canvas) RenderRaw(width, height int, selectedBox int, previewFromX, previewFromY int, previewWaypoints []point, previewToX, previewToY int, panX, panY int, cursorX, cursorY int, showCursor bool, editBoxID int, editTextID int, editCursorPos int, editText string, editTextX int, editTextY int, selectionStartX, selectionStartY, selectionEndX, selectionEndY int, showBoxNumbers bool, editSelStart, editSelEnd int) *RenderResult {
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
			Waypoints: make([]point, len(previewWaypoints)),
			Color:     -1,
		}
		for i, wp := range previewWaypoints {
			previewConnection.Waypoints[i] = point{X: wp.X, Y: wp.Y}
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
			// Draw box number in top left corner
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
	// Apply edit text selection highlighting
	if editSelStart >= 0 && editSelEnd >= 0 && editSelStart != editSelEnd {
		selStart, selEnd := editSelStart, editSelEnd
		if selStart > selEnd {
			selStart, selEnd = selEnd, selStart
		}
		// Calculate screen positions for each character in the selection
		for pos := selStart; pos < selEnd; pos++ {
			var screenX, screenY int
			if editTextID == -2 && editBoxID >= 0 && editBoxID < len(c.boxes) {
				// Title editing mode
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
		// Title editing mode
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
		// Text input mode - show cursor at input position
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

	// Object colors, before highlights so an explicit highlight still wins.
	paintCells := func(cells []point, colorIndex int) {
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

// Render returns the canvas with ANSI color codes applied
// This is a convenience method that calls RenderRaw and ApplyColors
func (c *Canvas) Render(width, height int, selectedBox int, previewFromX, previewFromY int, previewWaypoints []point, previewToX, previewToY int, panX, panY int, cursorX, cursorY int, showCursor bool, editBoxID int, editTextID int, editCursorPos int, editText string, editTextX int, editTextY int, selectionStartX, selectionStartY, selectionEndX, selectionEndY int, showBoxNumbers bool, editSelStart, editSelEnd int) []string {
	result := c.RenderRaw(width, height, selectedBox, previewFromX, previewFromY, previewWaypoints, previewToX, previewToY, panX, panY, cursorX, cursorY, showCursor, editBoxID, editTextID, editCursorPos, editText, editTextX, editTextY, selectionStartX, selectionStartY, selectionEndX, selectionEndY, showBoxNumbers, editSelStart, editSelEnd)
	return result.ApplyColors()
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

func (c *Canvas) drawBox(canvas [][]rune, box Box, isSelected bool) {
	c.drawBoxAt(canvas, box, isSelected, box.X, box.Y)
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

	// Calculate visible range for the box
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
			// Determine what character should be at this position
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

	// Draw title and divider if present
	contentStartLine := 0
	if box.Title != "" {
		// Handle multiline titles
		titleLines := strings.Split(box.Title, "\n")
		maxWidth := box.Width - 2
		if maxWidth < 0 {
			maxWidth = 0
		}

		// Draw each line of the title (left-aligned like content text)
		for lineIdx, titleLine := range titleLines {
			titleY := boxY + 1 + lineIdx
			if titleY >= 0 && titleY < len(canvas) {
				titleText := titleLine
				if len(titleText) > maxWidth {
					titleText = titleText[:maxWidth]
				}
				// Left-align the title (same as content text)
				titleX := boxX + 1

				for i, char := range titleText {
					if titleX+i >= 0 && titleX+i < len(canvas[titleY]) && titleX+i < boxX+box.Width-1 {
						canvas[titleY][titleX+i] = char
					}
				}
			}
		}

		// Draw divider line below title
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

		contentStartLine = 1 + len(titleLines) + 1 // Content starts after border, title lines, and divider
	} else {
		contentStartLine = 1 // Content starts after border
	}

	// Draw content lines
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
			// Draw each character, handling negative starting X
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

func (c *Canvas) drawText(canvas [][]rune, text Text) {
	c.drawTextAt(canvas, text, text.X, text.Y)
}

func (c *Canvas) drawTextAt(canvas [][]rune, text Text, textX, textY int) {
	for lineIdx, line := range text.Lines {
		lineY := textY + lineIdx
		lineX := textX
		if lineY >= 0 && lineY < len(canvas) {
			// Draw each character, handling negative starting X
			for i, char := range line {
				charX := lineX + i
				if charX >= 0 && charX < len(canvas[lineY]) {
					canvas[lineY][charX] = char
				}
			}
		}
	}
}

func (c *Canvas) calculateTextCursorPosition(box Box, cursorPos int, text string, panX, panY int) (int, int) {
	lines := strings.Split(text, "\n")
	currentPos := 0

	// Calculate content start offset (accounts for title area if present)
	contentStartY := 1 // After top border
	if box.Title != "" {
		titleLines := strings.Split(box.Title, "\n")
		contentStartY = 1 + len(titleLines) + 1 // After border, title lines, and divider
	}

	for lineIdx, line := range lines {
		lineLength := len([]rune(line))
		if cursorPos <= currentPos+lineLength {
			posInLine := cursorPos - currentPos
			return box.X + 1 + posInLine - panX, box.Y + contentStartY + lineIdx - panY
		}
		currentPos += lineLength + 1
	}
	if len(lines) > 0 {
		lastLine := lines[len(lines)-1]
		return box.X + 1 + len([]rune(lastLine)) - panX, box.Y + contentStartY + len(lines) - 1 - panY
	}
	return box.X + 1 - panX, box.Y + contentStartY - panY
}

func (c *Canvas) calculateTitleCursorPosition(box Box, cursorPos int, text string, panX, panY int) (int, int) {
	lines := strings.Split(text, "\n")
	currentPos := 0

	for lineIdx, line := range lines {
		lineLength := len([]rune(line))
		if cursorPos <= currentPos+lineLength {
			posInLine := cursorPos - currentPos
			// Title is left-aligned like content text
			return box.X + 1 + posInLine - panX, box.Y + 1 + lineIdx - panY
		}
		currentPos += lineLength + 1
	}
	if len(lines) > 0 {
		lastLine := lines[len(lines)-1]
		return box.X + 1 + len([]rune(lastLine)) - panX, box.Y + 1 + len(lines) - 1 - panY
	}
	return box.X + 1 - panX, box.Y + 1 - panY
}

func (c *Canvas) calculateTextCursorPositionForText(text Text, cursorPos int, textContent string, panX, panY int) (int, int) {
	lines := strings.Split(textContent, "\n")
	currentPos := 0
	for lineIdx, line := range lines {
		lineLength := len([]rune(line))
		if cursorPos <= currentPos+lineLength {
			posInLine := cursorPos - currentPos
			return text.X + posInLine - panX, text.Y + lineIdx - panY
		}
		currentPos += lineLength + 1
	}
	if len(lines) > 0 {
		lastLine := lines[len(lines)-1]
		return text.X + len([]rune(lastLine)) - panX, text.Y + len(lines) - 1 - panY
	}
	return text.X - panX, text.Y - panY
}

func (c *Canvas) calculateTextCursorPositionForNewText(textX, textY int, cursorPos int, textContent string, panX, panY int) (int, int) {
	lines := strings.Split(textContent, "\n")
	currentPos := 0
	for lineIdx, line := range lines {
		lineLength := len([]rune(line))
		if cursorPos <= currentPos+lineLength {
			posInLine := cursorPos - currentPos
			return textX + posInLine - panX, textY + lineIdx - panY
		}
		currentPos += lineLength + 1
	}
	if len(lines) > 0 {
		lastLine := lines[len(lines)-1]
		return textX + len([]rune(lastLine)) - panX, textY + len(lines) - 1 - panY
	}
	return textX - panX, textY - panY
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

func (c *Canvas) drawConnectionWithPan(canvas [][]rune, connection Connection, panX, panY int) {
	adjustedConnection := connection
	adjustedConnection.FromX = connection.FromX - panX
	adjustedConnection.FromY = connection.FromY - panY
	adjustedConnection.ToX = connection.ToX - panX
	adjustedConnection.ToY = connection.ToY - panY
	adjustedConnection.Waypoints = make([]point, len(connection.Waypoints))
	for i, wp := range connection.Waypoints {
		adjustedConnection.Waypoints[i] = point{X: wp.X - panX, Y: wp.Y - panY}
	}
	c.drawConnection(canvas, adjustedConnection, connection, panX, panY)
}

// drawConnection renders an orthogonal line through its waypoints. The path is
// drawn cell-inclusive so it never breaks; bends get the matching elbow glyph,
// and endpoints anchored to a box get an arrow.
func (c *Canvas) drawConnection(canvas [][]rune, connection Connection, originalConnection Connection, panX, panY int) {
	pts := []point{{connection.FromX, connection.FromY}}
	pts = append(pts, connection.Waypoints...)
	pts = append(pts, point{connection.ToX, connection.ToY})

	// Orthogonal vertex chain, matching GetConnectionCells (elbow at to.X,from.Y).
	var verts []point
	addV := func(p point) {
		if len(verts) == 0 || verts[len(verts)-1] != p {
			verts = append(verts, p)
		}
	}
	for i := 0; i < len(pts)-1; i++ {
		from, to := pts[i], pts[i+1]
		addV(from)
		if from.X != to.X && from.Y != to.Y {
			addV(point{X: to.X, Y: from.Y})
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

// cornerChar returns the elbow glyph for the bend prev->cur->next, or 0 when the
// three points are collinear (no turn).
func cornerChar(prev, cur, next point) rune {
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

// drawConnEndArrow draws an arrowhead pointing into boxID at the box edge nearest
// the connection's world anchor (ax, ay). No-op for non-box endpoints.
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

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func getColorCode(colorIndex int) string {
	// Special handling for edit selection color (reverse video cyan background)
	if colorIndex == colorEditSelect {
		return "\x1b[7;36m" // Reverse video with cyan
	}
	if colorIndex == colorMouseSelect {
		return "\x1b[103m" // Bright yellow background
	}
	if colorIndex == colorMenuSelect {
		return "\x1b[7m" // Reverse video
	}
	if colorIndex == colorMenuBorder {
		return "\x1b[32m" // Green (matches the startup logo)
	}
	colors := []int{47, 41, 42, 43, 44, 45, 46, 47}
	if colorIndex < 0 || colorIndex >= len(colors) {
		return ""
	}
	return fmt.Sprintf("\x1b[%dm", colors[colorIndex])
}

func getTextColorCode(colorIndex int) string {
	// Special handling for edit selection color (reverse video cyan)
	if colorIndex == colorEditSelect {
		return "\x1b[7;36m" // Reverse video with cyan
	}
	if colorIndex == colorMouseSelect {
		return "\x1b[1;93m" // Bright bold yellow
	}
	if colorIndex == colorMenuSelect {
		return "\x1b[7m" // Reverse video
	}
	if colorIndex == colorMenuBorder {
		return "\x1b[32m" // Green (matches the startup logo)
	}
	colors := []int{37, 31, 32, 33, 34, 35, 36, 37}
	if colorIndex < 0 || colorIndex >= len(colors) {
		return ""
	}
	return fmt.Sprintf("\x1b[%dm", colors[colorIndex])
}

const colorReset = "\x1b[0m"

func (c *Canvas) SetHighlight(x, y int, colorIndex int) {
	if colorIndex < 0 || colorIndex >= numColors {
		return
	}
	c.highlights[fmt.Sprintf("%d,%d", x, y)] = colorIndex
}

func (c *Canvas) GetHighlight(x, y int) int {
	if color, ok := c.highlights[fmt.Sprintf("%d,%d", x, y)]; ok {
		return color
	}
	return -1
}

func (c *Canvas) ClearHighlight(x, y int) {
	delete(c.highlights, fmt.Sprintf("%d,%d", x, y))
}

func (c *Canvas) MoveHighlight(fromX, fromY, toX, toY int) {
	fromKey := fmt.Sprintf("%d,%d", fromX, fromY)
	if colorIndex, exists := c.highlights[fromKey]; exists {
		delete(c.highlights, fromKey)
		if toX >= 0 && toY >= 0 {
			c.highlights[fmt.Sprintf("%d,%d", toX, toY)] = colorIndex
		}
	}
}

func (c *Canvas) MoveHighlightsInRegion(minX, minY, maxX, maxY, deltaX, deltaY int) {
	type highlightInfo struct {
		x, y  int
		color int
	}
	highlightsToMove := make([]highlightInfo, 0)
	keysToDelete := make([]string, 0)
	for key, colorIndex := range c.highlights {
		var x, y int
		fmt.Sscanf(key, "%d,%d", &x, &y)
		if x >= minX && x <= maxX && y >= minY && y <= maxY {
			highlightsToMove = append(highlightsToMove, highlightInfo{x: x, y: y, color: colorIndex})
			keysToDelete = append(keysToDelete, key)
		}
	}
	for _, key := range keysToDelete {
		delete(c.highlights, key)
	}
	for _, h := range highlightsToMove {
		newX := h.x + deltaX
		newY := h.y + deltaY
		if newX >= 0 && newY >= 0 {
			c.highlights[fmt.Sprintf("%d,%d", newX, newY)] = h.color
		}
	}
}

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

func (c *Canvas) GetBoxBorderCells(boxID int) []point {
	if boxID < 0 || boxID >= len(c.boxes) {
		return nil
	}
	box := c.boxes[boxID]
	cells := make([]point, 0)
	// Top row (entire width)
	for x := box.X; x < box.X+box.Width; x++ {
		cells = append(cells, point{X: x, Y: box.Y})
	}
	// Bottom row (entire width)
	for x := box.X; x < box.X+box.Width; x++ {
		cells = append(cells, point{X: x, Y: box.Y + box.Height - 1})
	}
	// Left and right columns (excluding top and bottom corners already added)
	for y := box.Y + 1; y < box.Y+box.Height-1; y++ {
		cells = append(cells, point{X: box.X, Y: y})
		cells = append(cells, point{X: box.X + box.Width - 1, Y: y})
	}
	return cells
}

func (c *Canvas) GetBoxBorderCellsWithoutDivider(boxID int) []point {
	// Returns border cells excluding the title bar area
	if boxID < 0 || boxID >= len(c.boxes) {
		return nil
	}
	box := c.boxes[boxID]
	cells := make([]point, 0)

	// If box has a title, exclude the title bar area
	startY := box.Y + 1
	if box.Title != "" {
		titleLines := strings.Split(box.Title, "\n")
		startY = box.Y + 1 + len(titleLines) + 1 // Skip past title lines and divider
	}

	// Bottom row (entire width)
	for x := box.X; x < box.X+box.Width; x++ {
		cells = append(cells, point{X: x, Y: box.Y + box.Height - 1})
	}

	// Left and right columns (excluding title bar area and bottom corners)
	for y := startY; y < box.Y+box.Height-1; y++ {
		cells = append(cells, point{X: box.X, Y: y})
		cells = append(cells, point{X: box.X + box.Width - 1, Y: y})
	}

	return cells
}

func (c *Canvas) GetBoxTitleDividerCells(boxID int) []point {
	if boxID < 0 || boxID >= len(c.boxes) {
		return nil
	}
	box := c.boxes[boxID]
	cells := make([]point, 0)
	// Only return divider cells if box has a title
	if box.Title != "" {
		titleLines := strings.Split(box.Title, "\n")
		dividerY := box.Y + 1 + len(titleLines)
		for x := box.X + 1; x < box.X+box.Width-1; x++ {
			cells = append(cells, point{X: x, Y: dividerY})
		}
	}
	return cells
}

func (c *Canvas) GetBoxTitleBarCells(boxID int) []point {
	// Returns all cells in the title bar area (top row, sides up to divider, and divider)
	if boxID < 0 || boxID >= len(c.boxes) {
		return nil
	}
	box := c.boxes[boxID]
	cells := make([]point, 0)

	// Only return title bar cells if box has a title
	if box.Title == "" {
		return cells
	}

	titleLines := strings.Split(box.Title, "\n")

	// Top row (entire width)
	for x := box.X; x < box.X+box.Width; x++ {
		cells = append(cells, point{X: x, Y: box.Y})
	}

	// Left and right edges from row 1 to divider row (up to and including divider row)
	dividerY := box.Y + 1 + len(titleLines)
	for y := box.Y + 1; y <= dividerY; y++ {
		cells = append(cells, point{X: box.X, Y: y})
		cells = append(cells, point{X: box.X + box.Width - 1, Y: y})
	}

	// Divider line (interior cells on divider row)
	for x := box.X + 1; x < box.X+box.Width-1; x++ {
		cells = append(cells, point{X: x, Y: dividerY})
	}

	return cells
}

func (c *Canvas) GetBoxTitleTextCells(boxID int) []point {
	// Returns cells containing the title text (not the border or divider)
	if boxID < 0 || boxID >= len(c.boxes) {
		return nil
	}
	box := c.boxes[boxID]
	cells := make([]point, 0)

	if box.Title == "" {
		return cells
	}

	titleLines := strings.Split(box.Title, "\n")

	// Title text is inside the box, starting at row box.Y+1
	for lineIdx, line := range titleLines {
		titleY := box.Y + 1 + lineIdx
		// Title is left-aligned, starting at box.X + 1
		for i := 0; i < len(line) && i < box.Width-2; i++ {
			cells = append(cells, point{X: box.X + 1 + i, Y: titleY})
		}
	}

	return cells
}

func (c *Canvas) GetBoxContentTextCells(boxID int) []point {
	// Returns cells containing the box content text (not the title, border, or divider)
	if boxID < 0 || boxID >= len(c.boxes) {
		return nil
	}
	box := c.boxes[boxID]
	cells := make([]point, 0)

	// Calculate content start offset (accounts for title area if present)
	contentStartY := box.Y + 1 // After top border
	if box.Title != "" {
		titleLines := strings.Split(box.Title, "\n")
		contentStartY = box.Y + 1 + len(titleLines) + 1 // After border, title lines, and divider
	}

	// Content text is inside the box
	for lineIdx, line := range box.Lines {
		lineY := contentStartY + lineIdx
		// Content is left-aligned, starting at box.X + 1
		for i := 0; i < len(line) && i < box.Width-2; i++ {
			cells = append(cells, point{X: box.X + 1 + i, Y: lineY})
		}
	}

	return cells
}

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

func (c *Canvas) getHighlightsForBox(boxID int) []HighlightCell {
	cells := c.GetBoxCells(boxID)
	highlights := make([]HighlightCell, 0)
	for _, cell := range cells {
		key := fmt.Sprintf("%d,%d", cell.X, cell.Y)
		if colorIndex, exists := c.highlights[key]; exists {
			highlights = append(highlights, HighlightCell{
				X:        cell.X,
				Y:        cell.Y,
				Color:    colorIndex,
				HadColor: true,
				OldColor: colorIndex,
			})
		}
	}
	return highlights
}

func (c *Canvas) getHighlightsForText(textID int) []HighlightCell {
	cells := c.GetTextCells(textID)
	highlights := make([]HighlightCell, 0)
	for _, cell := range cells {
		key := fmt.Sprintf("%d,%d", cell.X, cell.Y)
		if colorIndex, exists := c.highlights[key]; exists {
			highlights = append(highlights, HighlightCell{
				X:        cell.X,
				Y:        cell.Y,
				Color:    colorIndex,
				HadColor: true,
				OldColor: colorIndex,
			})
		}
	}
	return highlights
}

// GetBoxContentHighlights returns a map of character indices to color indices
// for all highlights in the box's content area. The character index is the
// position within the box's OriginalText (or Lines joined by newlines).
// This is used for displaying colored text in tooltips.
func (c *Canvas) GetBoxContentHighlights(boxID int) map[int]int {
	result := make(map[int]int)
	if boxID < 0 || boxID >= len(c.boxes) {
		return result
	}

	box := c.boxes[boxID]
	text := box.GetText()
	if text == "" {
		return result
	}

	lines := strings.Split(text, "\n")

	// For each highlight, check if it's in this box's content area
	// and map it to a character index
	for key, colorIndex := range c.highlights {
		var wx, wy int
		fmt.Sscanf(key, "%d,%d", &wx, &wy)

		// Calculate position relative to box content area
		relCol := wx - (box.X + 1)
		relRow := wy - (box.Y + 1)

		// Check if this position is within valid text bounds
		if relRow < 0 || relRow >= len(lines) {
			continue
		}
		if relCol < 0 || relCol >= len(lines[relRow]) {
			continue
		}

		// Convert (row, col) to character index in the full text
		charIndex := 0
		for i := 0; i < relRow; i++ {
			charIndex += len(lines[i]) + 1 // +1 for newline
		}
		charIndex += relCol

		result[charIndex] = colorIndex
	}

	return result
}

func (c *Canvas) deleteHighlightsForBox(boxID int) {
	cells := c.GetBoxCells(boxID)
	for _, cell := range cells {
		key := fmt.Sprintf("%d,%d", cell.X, cell.Y)
		delete(c.highlights, key)
	}
}

func (c *Canvas) deleteHighlightsForText(textID int) {
	cells := c.GetTextCells(textID)
	for _, cell := range cells {
		key := fmt.Sprintf("%d,%d", cell.X, cell.Y)
		delete(c.highlights, key)
	}
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

// SetBoxColor sets a box's color (palette index 0-7, or -1 for none).
func (c *Canvas) SetBoxColor(boxID, color int) {
	if boxID >= 0 && boxID < len(c.boxes) {
		c.boxes[boxID].Color = color
	}
}

// SetTextColor sets a text object's color (palette index 0-7, or -1 for none).
func (c *Canvas) SetTextColor(textID, color int) {
	if textID >= 0 && textID < len(c.texts) {
		c.texts[textID].Color = color
	}
}

// SetLineColor sets a connection's color (palette index 0-7, or -1 for none).
func (c *Canvas) SetLineColor(connIdx, color int) {
	if connIdx >= 0 && connIdx < len(c.connections) {
		c.connections[connIdx].Color = color
	}
}

func (c *Canvas) GetConnectionCells(connIdx int) []point {
	if connIdx < 0 || connIdx >= len(c.connections) {
		return nil
	}
	conn := c.connections[connIdx]
	cells := make([]point, 0)
	points := []point{{conn.FromX, conn.FromY}}
	points = append(points, conn.Waypoints...)
	points = append(points, point{conn.ToX, conn.ToY})
	for i := 0; i < len(points)-1; i++ {
		from := points[i]
		to := points[i+1]
		if from.X == to.X {
			startY, endY := from.Y, to.Y
			if startY > endY {
				startY, endY = endY, startY
			}
			for y := startY; y <= endY; y++ {
				cells = append(cells, point{X: from.X, Y: y})
			}
		} else if from.Y == to.Y {
			startX, endX := from.X, to.X
			if startX > endX {
				startX, endX = endX, startX
			}
			for x := startX; x <= endX; x++ {
				cells = append(cells, point{X: x, Y: from.Y})
			}
		} else {
			cornerX := to.X
			cornerY := from.Y
			startX, endX := from.X, cornerX
			if startX > endX {
				startX, endX = endX, startX
			}
			for x := startX; x <= endX; x++ {
				cells = append(cells, point{X: x, Y: from.Y})
			}
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
		if c.GetHighlight(current.X, current.Y) != targetColor {
			continue
		}
		visited[key] = true
		result = append(result, current)
		adjacent := []point{
			{current.X, current.Y - 1},
			{current.X, current.Y + 1},
			{current.X - 1, current.Y},
			{current.X + 1, current.Y},
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
		// Escape commas in title to prevent parsing issues (title is a fixed field)
		encodedTitle := strings.ReplaceAll(box.Title, "\n", "\\n")
		encodedTitle = strings.ReplaceAll(encodedTitle, ",", "\\,")
		// Format: X,Y,Width,Height,ZLevel,BorderStyle,Title,Text
		fmt.Fprintf(file, "%d,%d,%d,%d,%d,%d,%s,%s\n",
			box.X, box.Y, box.Width, box.Height, box.ZLevel, box.BorderStyle, encodedTitle, encodedText)
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

	// Optional color sections (index,color), after HIGHLIGHTS so old readers skip them.
	writeColors := func(header string, colors []int) {
		var lines []string
		for i, col := range colors {
			if col >= 0 {
				lines = append(lines, fmt.Sprintf("%d,%d", i, col))
			}
		}
		fmt.Fprintf(file, "%s:%d\n", header, len(lines))
		for _, line := range lines {
			fmt.Fprintln(file, line)
		}
	}
	boxColors := make([]int, len(c.boxes))
	for i, b := range c.boxes {
		boxColors[i] = b.Color
	}
	lineColors := make([]int, len(c.connections))
	for i, cn := range c.connections {
		lineColors[i] = cn.Color
	}
	textColors := make([]int, len(c.texts))
	for i, t := range c.texts {
		textColors[i] = t.Color
	}
	writeColors("BOXCOLORS", boxColors)
	writeColors("LINECOLORS", lineColors)
	writeColors("TEXTCOLORS", textColors)

	return nil
}

func (c *Canvas) SaveToFileWithPan(filename string, panX, panY int) error {
	err := c.SaveToFile(filename)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Fprintf(file, "PAN:%d,%d\n", panX, panY)
	return nil
}

// splitBoxLine splits a box line respecting escaped commas in the title field
// The format is: X,Y,Width,Height,ZLevel,BorderStyle,Title,Text
// Title field (index 6) may contain escaped commas (\,)
func splitBoxLine(line string) []string {
	var fields []string
	var current strings.Builder
	escaped := false

	for i := 0; i < len(line); i++ {
		ch := line[i]
		if escaped {
			current.WriteByte(ch)
			escaped = false
		} else if ch == '\\' && i+1 < len(line) && line[i+1] == ',' {
			// Escaped comma - write the backslash and let next iteration handle the comma
			current.WriteByte(ch)
			escaped = true
		} else if ch == ',' {
			fields = append(fields, current.String())
			current.Reset()
		} else {
			current.WriteByte(ch)
		}
	}
	fields = append(fields, current.String())
	return fields
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
	if !scanner.Scan() || scanner.Text() != "FLOWCHART" {
		return fmt.Errorf("invalid file format")
	}
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
		line := scanner.Text()

		// Parse box data by finding comma positions, respecting escaped commas
		var x, y, width, height, zLevel int
		var borderStyle BorderStyle
		var title, text string

		// Split line into fields, but handle escaped commas in title field
		// Format: X,Y,Width,Height,ZLevel,BorderStyle,Title,Text
		// The first 6 fields are numeric, then title (may have escaped commas), then text
		fields := splitBoxLine(line)

		if len(fields) < 3 {
			return fmt.Errorf("invalid box format")
		}

		x, _ = strconv.Atoi(fields[0])
		y, _ = strconv.Atoi(fields[1])

		// Parse based on number of fields for backward compatibility
		if len(fields) >= 8 {
			// New format: X,Y,Width,Height,ZLevel,BorderStyle,Title,Text
			width, _ = strconv.Atoi(fields[2])
			height, _ = strconv.Atoi(fields[3])
			zLevel, _ = strconv.Atoi(fields[4])
			if zLevel < 0 || zLevel > 3 {
				zLevel = 0
			}
			borderStyleInt, _ := strconv.Atoi(fields[5])
			borderStyle = BorderStyle(borderStyleInt)
			// Unescape commas and newlines in title
			title = strings.ReplaceAll(fields[6], "\\,", ",")
			title = strings.ReplaceAll(title, "\\n", "\n")
			text = strings.ReplaceAll(strings.Join(fields[7:], ","), "\\n", "\n")
		} else if len(fields) >= 6 {
			// Old format: X,Y,Width,Height,ZLevel,Text
			width, _ = strconv.Atoi(fields[2])
			height, _ = strconv.Atoi(fields[3])
			zLevel, _ = strconv.Atoi(fields[4])
			if zLevel < 0 || zLevel > 3 {
				zLevel = 0
			}
			borderStyle = BorderStyleASCII
			title = ""
			text = strings.ReplaceAll(strings.Join(fields[5:], ","), "\\n", "\n")
		} else if len(fields) >= 5 {
			// Older format: X,Y,Width,Height,Text
			width, _ = strconv.Atoi(fields[2])
			height, _ = strconv.Atoi(fields[3])
			zLevel = 0
			borderStyle = BorderStyleASCII
			title = ""
			text = strings.ReplaceAll(strings.Join(fields[4:], ","), "\\n", "\n")
		} else {
			// Oldest format: X,Y,Text
			text = strings.ReplaceAll(strings.Join(fields[2:], ","), "\\n", "\n")
			box := Box{
				X:           x,
				Y:           y,
				ID:          i,
				ZLevel:      0,
				BorderStyle: BorderStyleASCII,
				Title:       "",
				Color:       -1,
			}
			box.SetText(text)
			c.boxes = append(c.boxes, box)
			continue
		}

		box := Box{
			X:           x,
			Y:           y,
			ID:          i,
			ZLevel:      zLevel,
			BorderStyle: borderStyle,
			Title:       title,
			Color:       -1,
		}
		box.SetText(text)
		box.Width = width
		box.Height = height
		c.boxes = append(c.boxes, box)
	}

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
				Color:     -1,
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
				line := scanner.Text()
				// Find the first two commas to extract X and Y
				// Everything after the second comma is the text content
				// This handles text that contains commas correctly
				firstComma := strings.Index(line, ",")
				if firstComma == -1 {
					continue
				}
				secondComma := strings.Index(line[firstComma+1:], ",")
				if secondComma == -1 {
					continue
				}
				secondComma += firstComma + 1

				x, err1 := strconv.Atoi(line[:firstComma])
				y, err2 := strconv.Atoi(line[firstComma+1 : secondComma])
				if err1 != nil || err2 != nil {
					continue
				}
				text := strings.ReplaceAll(line[secondComma+1:], "\\n", "\n")
				c.AddText(x, y, text)
			}
		}
	}

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
	}

	// Optional color sections; unknown lines (e.g. PAN) are skipped.
	for scanner.Scan() {
		line := scanner.Text()
		var header string
		switch {
		case strings.HasPrefix(line, "BOXCOLORS:"):
			header = "BOXCOLORS"
		case strings.HasPrefix(line, "LINECOLORS:"):
			header = "LINECOLORS"
		case strings.HasPrefix(line, "TEXTCOLORS:"):
			header = "TEXTCOLORS"
		default:
			continue
		}
		count, err := strconv.Atoi(strings.TrimPrefix(line, header+":"))
		if err != nil {
			continue
		}
		for i := 0; i < count; i++ {
			if !scanner.Scan() {
				break
			}
			parts := strings.Split(scanner.Text(), ",")
			if len(parts) < 2 {
				continue
			}
			idx, err1 := strconv.Atoi(parts[0])
			col, err2 := strconv.Atoi(parts[1])
			if err1 != nil || err2 != nil || col < 0 || col >= numColors {
				continue
			}
			switch header {
			case "BOXCOLORS":
				if idx >= 0 && idx < len(c.boxes) {
					c.boxes[idx].Color = col
				}
			case "LINECOLORS":
				if idx >= 0 && idx < len(c.connections) {
					c.connections[idx].Color = col
				}
			case "TEXTCOLORS":
				if idx >= 0 && idx < len(c.texts) {
					c.texts[idx].Color = col
				}
			}
		}
	}

	return scanner.Err()
}

func (c *Canvas) LoadFromFileWithPan(filename string) (int, int, error) {
	err := c.LoadFromFile(filename)
	if err != nil {
		return 0, 0, err
	}

	file, err := os.Open(filename)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	panX, panY := 0, 0
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "PAN:") {
			parts := strings.Split(strings.TrimPrefix(line, "PAN:"), ",")
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

	charWidth := 8.0
	charHeight := 16.0
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

	padding := 2
	minX -= padding
	minY -= padding
	maxX += padding
	maxY += padding
	imageWidth := int(float64(maxX-minX) * charWidth)
	imageHeight := int(float64(maxY-minY) * charHeight)
	dc := gg.NewContext(imageWidth, imageHeight)
	dc.SetColor(color.White)
	dc.Clear()
	dc.SetColor(color.Black)
	fontData := gomono.TTF
	ttfFont, err := truetype.Parse(fontData)
	if err != nil {
		return fmt.Errorf("failed to parse font: %v", err)
	}
	face := truetype.NewFace(ttfFont, &truetype.Options{
		Size:    12.0,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	dc.SetFontFace(face)
	for _, conn := range c.connections {
		c.drawConnectionPNG(dc, conn, minX, minY, charWidth, charHeight)
	}
	for _, text := range c.texts {
		c.drawTextPNG(dc, text, minX, minY, charWidth, charHeight)
	}
	for _, box := range c.boxes {
		c.drawBoxPNG(dc, box, minX, minY, charWidth, charHeight)
	}

	return dc.SavePNG(filename)
}

// pngColor maps a palette index (0-7) to an RGB color for PNG export, mirroring
// the ANSI foreground palette. Returns black for -1/out-of-range.
func pngColor(index int) color.Color {
	switch index {
	case 0:
		return color.RGBA{128, 128, 128, 255} // Gray
	case 1:
		return color.RGBA{205, 0, 0, 255} // Red
	case 2:
		return color.RGBA{0, 160, 0, 255} // Green
	case 3:
		return color.RGBA{190, 160, 0, 255} // Yellow
	case 4:
		return color.RGBA{0, 0, 220, 255} // Blue
	case 5:
		return color.RGBA{190, 0, 190, 255} // Magenta
	case 6:
		return color.RGBA{0, 170, 170, 255} // Cyan
	case 7:
		return color.RGBA{230, 230, 230, 255} // White
	default:
		return color.Black
	}
}

func (c *Canvas) drawConnectionPNG(dc *gg.Context, conn Connection, minX, minY int, charWidth, charHeight float64) {
	points := []point{{conn.FromX, conn.FromY}}
	points = append(points, conn.Waypoints...)
	points = append(points, point{conn.ToX, conn.ToY})
	if len(points) < 2 {
		return
	}
	dc.SetLineWidth(1.0)
	if conn.Color >= 0 {
		dc.SetColor(pngColor(conn.Color))
	} else {
		dc.SetColor(color.Black)
	}
	for i := 0; i < len(points)-1; i++ {
		x1 := float64(points[i].X-minX) * charWidth
		y1 := float64(points[i].Y-minY) * charHeight
		x2 := float64(points[i+1].X-minX) * charWidth
		y2 := float64(points[i+1].Y-minY) * charHeight
		dc.DrawLine(x1, y1, x2, y2)
		dc.Stroke()
	}
	if conn.ArrowFrom && len(points) > 1 {
		c.drawArrowPNG(dc, points[1].X, points[1].Y, points[0].X, points[0].Y, minX, minY, charWidth, charHeight)
	}
	if conn.ArrowTo && len(points) > 1 {
		c.drawArrowPNG(dc, points[len(points)-2].X, points[len(points)-2].Y, points[len(points)-1].X, points[len(points)-1].Y, minX, minY, charWidth, charHeight)
	}
}

func (c *Canvas) drawArrowPNG(dc *gg.Context, fromX, fromY, toX, toY, minX, minY int, charWidth, charHeight float64) {
	fx := float64(fromX-minX) * charWidth
	fy := float64(fromY-minY) * charHeight
	tx := float64(toX-minX) * charWidth
	ty := float64(toY-minY) * charHeight
	dx := tx - fx
	dy := ty - fy
	length := math.Sqrt(dx*dx + dy*dy)
	if length < 0.1 {
		return
	}
	dx /= length
	dy /= length
	arrowSize := 6.0
	arrowAngle := 0.5
	tipX, tipY := tx, ty
	baseX1 := tx - arrowSize*dx + arrowSize*dy*arrowAngle
	baseY1 := ty - arrowSize*dy - arrowSize*dx*arrowAngle
	baseX2 := tx - arrowSize*dx - arrowSize*dy*arrowAngle
	baseY2 := ty - arrowSize*dy + arrowSize*dx*arrowAngle
	dc.MoveTo(tipX, tipY)
	dc.LineTo(baseX1, baseY1)
	dc.LineTo(baseX2, baseY2)
	dc.ClosePath()
	dc.Fill()
}

func (c *Canvas) drawBoxPNG(dc *gg.Context, box Box, minX, minY int, charWidth, charHeight float64) {
	x := float64(box.X-minX) * charWidth
	y := float64(box.Y-minY) * charHeight
	width := float64(box.Width) * charWidth
	height := float64(box.Height) * charHeight
	dc.SetLineWidth(1.0)
	if box.Color >= 0 {
		dc.SetColor(pngColor(box.Color))
	} else {
		dc.SetColor(color.Black)
	}
	dc.DrawRectangle(x, y, width, height)
	dc.Stroke()
	// Content text stays black for readability, matching the terminal render.
	dc.SetColor(color.Black)
	textY := y + charHeight
	for i, line := range box.Lines {
		dc.DrawString(line, x+charWidth, textY+float64(i)*charHeight)
	}
}

func (c *Canvas) drawTextPNG(dc *gg.Context, text Text, minX, minY int, charWidth, charHeight float64) {
	x := float64(text.X-minX) * charWidth
	y := float64(text.Y-minY) * charHeight
	if text.Color >= 0 {
		dc.SetColor(pngColor(text.Color))
	} else {
		dc.SetColor(color.Black)
	}
	for i, line := range text.Lines {
		dc.DrawString(line, x, y+float64(i)*charHeight)
	}
}
