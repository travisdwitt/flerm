package canvas

import "strings"

func (c *Canvas) SetHighlight(x, y, colorIndex int) {
	if colorIndex < 0 || colorIndex >= NumColors {
		return
	}
	c.highlights[Point{x, y}] = colorIndex
}

func (c *Canvas) GetHighlight(x, y int) int {
	if color, ok := c.highlights[Point{x, y}]; ok {
		return color
	}
	return -1
}

func (c *Canvas) ClearHighlight(x, y int) {
	delete(c.highlights, Point{x, y})
}

func (c *Canvas) GetBoxCells(boxID int) []Point {
	box, ok := c.box(boxID)
	if !ok {
		return nil
	}
	return rectCells(box.X, box.Y, box.Width, box.Height)
}

func rectCells(x, y, w, h int) []Point {
	cells := make([]Point, 0, w*h)
	for cy := y; cy < y+h; cy++ {
		for cx := x; cx < x+w; cx++ {
			cells = append(cells, Point{cx, cy})
		}
	}
	return cells
}

func (c *Canvas) GetBoxBorderCells(boxID int) []Point {
	box, ok := c.box(boxID)
	if !ok {
		return nil
	}
	cells := make([]Point, 0, 2*(box.Width+box.Height))
	for x := box.X; x < box.X+box.Width; x++ {
		cells = append(cells, Point{x, box.Y}, Point{x, box.Y + box.Height - 1})
	}
	for y := box.Y + 1; y < box.Y+box.Height-1; y++ {
		cells = append(cells, Point{box.X, y}, Point{box.X + box.Width - 1, y})
	}
	return cells
}

// titleDividerY reports the row of the rule under a box title, or -1 if untitled.
func titleDividerY(box Box) int {
	if box.Title == "" {
		return -1
	}
	return box.Y + 1 + len(strings.Split(box.Title, "\n"))
}

func (c *Canvas) GetBoxTitleDividerCells(boxID int) []Point {
	box, ok := c.box(boxID)
	if !ok {
		return nil
	}
	dividerY := titleDividerY(box)
	if dividerY < 0 {
		return nil
	}
	cells := make([]Point, 0, box.Width)
	for x := box.X + 1; x < box.X+box.Width-1; x++ {
		cells = append(cells, Point{x, dividerY})
	}
	return cells
}

func (c *Canvas) GetBoxTitleBarCells(boxID int) []Point {
	box, ok := c.box(boxID)
	if !ok {
		return nil
	}
	dividerY := titleDividerY(box)
	if dividerY < 0 {
		return []Point{}
	}
	cells := make([]Point, 0)
	for x := box.X; x < box.X+box.Width; x++ {
		cells = append(cells, Point{x, box.Y})
	}
	for y := box.Y + 1; y <= dividerY; y++ {
		cells = append(cells, Point{box.X, y}, Point{box.X + box.Width - 1, y})
	}
	for x := box.X + 1; x < box.X+box.Width-1; x++ {
		cells = append(cells, Point{x, dividerY})
	}
	return cells
}

// lineCells returns the cells covered by lines laid out starting at (x, y),
// clipped to maxLen columns (a negative maxLen means unclipped).
func lineCells(lines []string, x, y, maxLen int) []Point {
	cells := make([]Point, 0)
	for i, line := range lines {
		n := len(line)
		if maxLen >= 0 && n > maxLen {
			n = maxLen
		}
		for j := 0; j < n; j++ {
			cells = append(cells, Point{x + j, y + i})
		}
	}
	return cells
}

func (c *Canvas) GetBoxTitleTextCells(boxID int) []Point {
	box, ok := c.box(boxID)
	if !ok {
		return nil
	}
	if box.Title == "" {
		return []Point{}
	}
	return lineCells(strings.Split(box.Title, "\n"), box.X+1, box.Y+1, box.Width-2)
}

func (c *Canvas) GetBoxContentTextCells(boxID int) []Point {
	box, ok := c.box(boxID)
	if !ok {
		return nil
	}
	return lineCells(box.Lines, box.X+1, box.Y+contentStartLine(box), box.Width-2)
}

func (c *Canvas) GetTextCells(textID int) []Point {
	if textID < 0 || textID >= len(c.texts) {
		return nil
	}
	t := c.texts[textID]
	return lineCells(t.Lines, t.X, t.Y, -1)
}

func (c *Canvas) highlightsForCells(cells []Point) []HighlightCell {
	highlights := make([]HighlightCell, 0)
	for _, cell := range cells {
		if colorIndex, exists := c.highlights[cell]; exists {
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

// GetBoxContentHighlights maps highlighted cells inside a box back to character
// offsets in its text, so a tooltip can re-colour the same characters.
func (c *Canvas) GetBoxContentHighlights(boxID int) map[int]int {
	result := make(map[int]int)
	box, ok := c.box(boxID)
	if !ok || box.GetText() == "" {
		return result
	}
	lines := strings.Split(box.GetText(), "\n")
	charIndex := 0
	for row, line := range lines {
		for col := range line {
			if color, exists := c.highlights[Point{box.X + 1 + col, box.Y + 1 + row}]; exists {
				result[charIndex+col] = color
			}
		}
		charIndex += len(line) + 1
	}
	return result
}

func (c *Canvas) deleteHighlights(cells []Point) {
	for _, cell := range cells {
		delete(c.highlights, cell)
	}
}

func (c *Canvas) GetAdjacentHighlightsOfColor(startX, startY, targetColor int) []Point {
	if targetColor < 0 || targetColor >= NumColors {
		return nil
	}
	visited := map[Point]bool{}
	queue := []Point{{startX, startY}}
	result := make([]Point, 0)
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if visited[cur] || c.GetHighlight(cur.X, cur.Y) != targetColor {
			continue
		}
		visited[cur] = true
		result = append(result, cur)
		queue = append(queue,
			Point{cur.X, cur.Y - 1}, Point{cur.X, cur.Y + 1},
			Point{cur.X - 1, cur.Y}, Point{cur.X + 1, cur.Y})
	}
	return result
}
