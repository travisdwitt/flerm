package canvas

import (
	"fmt"
	"strings"
)

func (c *Canvas) SetHighlight(x, y int, colorIndex int) {
	if colorIndex < 0 || colorIndex >= NumColors {
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

func (c *Canvas) GetBoxCells(boxID int) []Point {
	if boxID < 0 || boxID >= len(c.boxes) {
		return nil
	}
	box := c.boxes[boxID]
	cells := make([]Point, 0)
	for y := box.Y; y < box.Y+box.Height; y++ {
		for x := box.X; x < box.X+box.Width; x++ {
			cells = append(cells, Point{X: x, Y: y})
		}
	}
	return cells
}

func (c *Canvas) GetBoxBorderCells(boxID int) []Point {
	if boxID < 0 || boxID >= len(c.boxes) {
		return nil
	}
	box := c.boxes[boxID]
	cells := make([]Point, 0)

	for x := box.X; x < box.X+box.Width; x++ {
		cells = append(cells, Point{X: x, Y: box.Y})
	}

	for x := box.X; x < box.X+box.Width; x++ {
		cells = append(cells, Point{X: x, Y: box.Y + box.Height - 1})
	}

	for y := box.Y + 1; y < box.Y+box.Height-1; y++ {
		cells = append(cells, Point{X: box.X, Y: y})
		cells = append(cells, Point{X: box.X + box.Width - 1, Y: y})
	}
	return cells
}

func (c *Canvas) GetBoxTitleDividerCells(boxID int) []Point {
	if boxID < 0 || boxID >= len(c.boxes) {
		return nil
	}
	box := c.boxes[boxID]
	cells := make([]Point, 0)

	if box.Title != "" {
		titleLines := strings.Split(box.Title, "\n")
		dividerY := box.Y + 1 + len(titleLines)
		for x := box.X + 1; x < box.X+box.Width-1; x++ {
			cells = append(cells, Point{X: x, Y: dividerY})
		}
	}
	return cells
}

func (c *Canvas) GetBoxTitleBarCells(boxID int) []Point {

	if boxID < 0 || boxID >= len(c.boxes) {
		return nil
	}
	box := c.boxes[boxID]
	cells := make([]Point, 0)

	if box.Title == "" {
		return cells
	}

	titleLines := strings.Split(box.Title, "\n")

	for x := box.X; x < box.X+box.Width; x++ {
		cells = append(cells, Point{X: x, Y: box.Y})
	}

	dividerY := box.Y + 1 + len(titleLines)
	for y := box.Y + 1; y <= dividerY; y++ {
		cells = append(cells, Point{X: box.X, Y: y})
		cells = append(cells, Point{X: box.X + box.Width - 1, Y: y})
	}

	for x := box.X + 1; x < box.X+box.Width-1; x++ {
		cells = append(cells, Point{X: x, Y: dividerY})
	}

	return cells
}

func (c *Canvas) GetBoxTitleTextCells(boxID int) []Point {

	if boxID < 0 || boxID >= len(c.boxes) {
		return nil
	}
	box := c.boxes[boxID]
	cells := make([]Point, 0)

	if box.Title == "" {
		return cells
	}

	titleLines := strings.Split(box.Title, "\n")

	for lineIdx, line := range titleLines {
		titleY := box.Y + 1 + lineIdx

		for i := 0; i < len(line) && i < box.Width-2; i++ {
			cells = append(cells, Point{X: box.X + 1 + i, Y: titleY})
		}
	}

	return cells
}

func (c *Canvas) GetBoxContentTextCells(boxID int) []Point {

	if boxID < 0 || boxID >= len(c.boxes) {
		return nil
	}
	box := c.boxes[boxID]
	cells := make([]Point, 0)

	contentStartY := box.Y + 1
	if box.Title != "" {
		titleLines := strings.Split(box.Title, "\n")
		contentStartY = box.Y + 1 + len(titleLines) + 1
	}

	for lineIdx, line := range box.Lines {
		lineY := contentStartY + lineIdx

		for i := 0; i < len(line) && i < box.Width-2; i++ {
			cells = append(cells, Point{X: box.X + 1 + i, Y: lineY})
		}
	}

	return cells
}

func (c *Canvas) GetTextCells(textID int) []Point {
	if textID < 0 || textID >= len(c.texts) {
		return nil
	}
	text := c.texts[textID]
	cells := make([]Point, 0)
	for lineIdx, line := range text.Lines {
		lineY := text.Y + lineIdx
		for x := text.X; x < text.X+len(line); x++ {
			cells = append(cells, Point{X: x, Y: lineY})
		}
	}
	return cells
}

func (c *Canvas) highlightsForCells(cells []Point) []HighlightCell {
	highlights := make([]HighlightCell, 0)
	for _, cell := range cells {
		key := fmt.Sprintf("%d,%d", cell.X, cell.Y)
		if colorIndex, exists := c.highlights[key]; exists {
			highlights = append(highlights, HighlightCell{X: cell.X, Y: cell.Y, Color: colorIndex, HadColor: true, OldColor: colorIndex})
		}
	}
	return highlights
}

func (c *Canvas) GetHighlightsForBox(boxID int) []HighlightCell {
	return c.highlightsForCells(c.GetBoxCells(boxID))
}

func (c *Canvas) GetHighlightsForText(textID int) []HighlightCell {
	return c.highlightsForCells(c.GetTextCells(textID))
}

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

	for key, colorIndex := range c.highlights {
		var wx, wy int
		fmt.Sscanf(key, "%d,%d", &wx, &wy)

		relCol := wx - (box.X + 1)
		relRow := wy - (box.Y + 1)

		if relRow < 0 || relRow >= len(lines) {
			continue
		}
		if relCol < 0 || relCol >= len(lines[relRow]) {
			continue
		}

		charIndex := 0
		for i := 0; i < relRow; i++ {
			charIndex += len(lines[i]) + 1
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

func (c *Canvas) GetAdjacentHighlightsOfColor(startX, startY int, targetColor int) []Point {
	if targetColor < 0 || targetColor >= NumColors {
		return nil
	}
	visited := make(map[string]bool)
	queue := []Point{{startX, startY}}
	result := make([]Point, 0)
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
		adjacent := []Point{
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
