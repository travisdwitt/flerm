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
	boxes  []Box
	arrows []Arrow
	texts  []Text
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

type Arrow struct {
	FromID int
	ToID   int
	FromX  int
	FromY  int
	ToX    int
	ToY    int
}

func NewCanvas() *Canvas {
	return &Canvas{
		boxes:  make([]Box, 0),
		arrows: make([]Arrow, 0),
		texts:  make([]Text, 0),
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

func (c *Canvas) AddArrow(fromID, toID int) {
	if fromID >= len(c.boxes) || toID >= len(c.boxes) {
		return
	}

	fromBox := c.boxes[fromID]
	toBox := c.boxes[toID]

	// Calculate the best connection points based on box positions
	var fromX, fromY, toX, toY int

	// Determine connection points based on relative positions
	fromCenterX := fromBox.X + fromBox.Width/2
	fromCenterY := fromBox.Y + fromBox.Height/2
	toCenterX := toBox.X + toBox.Width/2
	toCenterY := toBox.Y + toBox.Height/2

	// Choose connection points based on relative positions
	if abs(fromCenterX-toCenterX) > abs(fromCenterY-toCenterY) {
		// Horizontal connection is preferred
		if fromCenterX < toCenterX {
			// Connect from right side of fromBox to left side of toBox
			fromX = fromBox.X + fromBox.Width
			fromY = fromCenterY
			toX = toBox.X - 1
			toY = toCenterY
		} else {
			// Connect from left side of fromBox to right side of toBox
			fromX = fromBox.X - 1
			fromY = fromCenterY
			toX = toBox.X + toBox.Width
			toY = toCenterY
		}
	} else {
		// Vertical connection is preferred
		if fromCenterY < toCenterY {
			// Connect from bottom of fromBox to top of toBox
			fromX = fromCenterX
			fromY = fromBox.Y + fromBox.Height
			toX = toCenterX
			toY = toBox.Y - 1
		} else {
			// Connect from top of fromBox to bottom of toBox
			fromX = fromCenterX
			fromY = fromBox.Y - 1
			toX = toCenterX
			toY = toBox.Y + toBox.Height
		}
	}

	arrow := Arrow{
		FromID: fromID,
		ToID:   toID,
		FromX:  fromX,
		FromY:  fromY,
		ToX:    toX,
		ToY:    toY,
	}
	c.arrows = append(c.arrows, arrow)
}

func (c *Canvas) RemoveArrow(fromID, toID int) {
	newArrows := make([]Arrow, 0)
	for _, arrow := range c.arrows {
		if arrow.FromID != fromID || arrow.ToID != toID {
			newArrows = append(newArrows, arrow)
		}
	}
	c.arrows = newArrows
}

func (c *Canvas) RestoreArrow(arrow Arrow) {
	c.arrows = append(c.arrows, arrow)
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

		// Remove arrows connected to this box and update arrow IDs
		newArrows := make([]Arrow, 0)
		for _, arrow := range c.arrows {
			if arrow.FromID != id && arrow.ToID != id {
				// Update IDs if they're greater than the deleted box ID
				if arrow.FromID > id {
					arrow.FromID--
				}
				if arrow.ToID > id {
					arrow.ToID--
				}
				newArrows = append(newArrows, arrow)
			}
		}
		c.arrows = newArrows
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

		// Update box size
		box.Width = newWidth
		box.Height = newHeight
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

func (c *Canvas) Render(width, height int, selectedBox int) []string {
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

	// Draw arrows first (so they appear behind boxes)
	for _, arrow := range c.arrows {
		c.drawArrow(canvas, arrow)
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

func (c *Canvas) drawArrow(canvas [][]rune, arrow Arrow) {
	// Draw L-shaped arrow with line drawing characters
	fromX, fromY := arrow.FromX, arrow.FromY
	toX, toY := arrow.ToX, arrow.ToY

	// Create an L-shaped path: horizontal first, then vertical
	cornerX := toX
	cornerY := fromY

	// Draw horizontal line from start to corner (but not over the arrow head)
	if fromX != cornerX {
		startX := fromX
		endX := cornerX
		if startX > endX {
			startX, endX = endX, startX
		}

		for x := startX; x <= endX; x++ {
			if c.isValidPos(canvas, x, fromY) && canvas[fromY][x] == ' ' {
				// Don't draw over the final arrow position if this is a straight horizontal line
				if fromY == toY && x == toX {
					continue // Skip drawing here, arrow head will be placed
				}
				canvas[fromY][x] = '─' // horizontal line
			}
		}
	}

	// Draw vertical line from corner to end (but not over the arrow head)
	if cornerY != toY {
		startY := cornerY
		endY := toY
		if startY > endY {
			startY, endY = endY, startY
		}

		for y := startY; y <= endY; y++ {
			if c.isValidPos(canvas, cornerX, y) && canvas[y][cornerX] == ' ' {
				// Don't draw over the final arrow position if this is a straight vertical line
				if fromX == toX && y == toY {
					continue // Skip drawing here, arrow head will be placed
				}
				canvas[y][cornerX] = '│' // vertical line
			}
		}
	}

	// Draw corner if needed
	if fromX != toX && fromY != toY {
		if c.isValidPos(canvas, cornerX, cornerY) {
			// Corner is at (toX, fromY) - determine character based on the direction of turn
			// We're coming horizontally into the corner and leaving vertically
			if fromX < toX && fromY < toY {
				// Coming from left, going down
				canvas[cornerY][cornerX] = '┐' // top-right corner
			} else if fromX < toX && fromY > toY {
				// Coming from left, going up
				canvas[cornerY][cornerX] = '┘' // bottom-right corner
			} else if fromX > toX && fromY < toY {
				// Coming from right, going down
				canvas[cornerY][cornerX] = '┌' // top-left corner
			} else {
				// Coming from right, going up
				canvas[cornerY][cornerX] = '└' // bottom-left corner
			}
		}
	}

	// Always draw arrow head at the destination
	if c.isValidPos(canvas, toX, toY) {
		// Determine arrow direction based on how we approach the target
		if fromX != toX && fromY == toY {
			// Straight horizontal line - arrow points toward the target box
			if fromX < toX {
				canvas[toY][toX] = '▶' // pointing right (into box from left)
			} else {
				canvas[toY][toX] = '◀' // pointing left (into box from right)
			}
		} else if fromX == toX && fromY != toY {
			// Straight vertical line - arrow points toward the target box
			if fromY < toY {
				canvas[toY][toX] = '▼' // pointing down (into box from top)
			} else {
				canvas[toY][toX] = '▲' // pointing up (into box from bottom)
			}
		} else if fromX != toX && fromY != toY {
			// L-shaped line - arrow direction is based on the final segment into the target
			// The final segment is always vertical (from cornerY to toY)
			if cornerY < toY {
				canvas[toY][toX] = '▼' // pointing down (into box from top)
			} else if cornerY > toY {
				canvas[toY][toX] = '▲' // pointing up (into box from bottom)
			} else {
				// This case shouldn't happen with our L-shaped routing, but handle it
				// Fall back to horizontal direction
				if fromX < toX {
					canvas[toY][toX] = '▶' // pointing right
				} else {
					canvas[toY][toX] = '◀' // pointing left
				}
			}
		}
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
		// Encode multi-line text by replacing newlines with \n
		encodedText := strings.ReplaceAll(box.GetText(), "\n", "\\n")
		fmt.Fprintf(file, "%d,%d,%s\n", box.X, box.Y, encodedText)
	}

	fmt.Fprintf(file, "ARROWS:%d\n", len(c.arrows))
	for _, arrow := range c.arrows {
		fmt.Fprintf(file, "%d,%d\n", arrow.FromID, arrow.ToID)
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
	c.arrows = c.arrows[:0]

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
		if len(parts) != 3 {
			return fmt.Errorf("invalid box format")
		}

		x, _ := strconv.Atoi(parts[0])
		y, _ := strconv.Atoi(parts[1])
		// Decode multi-line text by replacing \n with newlines
		text := strings.ReplaceAll(parts[2], "\\n", "\n")

		box := Box{
			X:  x,
			Y:  y,
			ID: i,
		}
		box.SetText(text)
		c.boxes = append(c.boxes, box)
	}

	// Read arrows
	if !scanner.Scan() {
		return fmt.Errorf("missing arrows header")
	}
	arrowCountStr := strings.TrimPrefix(scanner.Text(), "ARROWS:")
	arrowCount, err := strconv.Atoi(arrowCountStr)
	if err != nil {
		return fmt.Errorf("invalid arrow count: %v", err)
	}

	for i := 0; i < arrowCount; i++ {
		if !scanner.Scan() {
			return fmt.Errorf("missing arrow data")
		}
		parts := strings.Split(scanner.Text(), ",")
		if len(parts) != 2 {
			return fmt.Errorf("invalid arrow format")
		}

		fromID, _ := strconv.Atoi(parts[0])
		toID, _ := strconv.Atoi(parts[1])

		c.AddArrow(fromID, toID)
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

	// Draw arrows
	for _, arrow := range c.arrows {
		fromX := float64(arrow.FromX * 10)
		fromY := float64(arrow.FromY * 10)
		toX := float64(arrow.ToX * 10)
		toY := float64(arrow.ToY * 10)

		dc.DrawLine(fromX, fromY, toX, toY)
		dc.Stroke()

		// Draw arrow head
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

