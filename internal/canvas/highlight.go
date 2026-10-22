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

func (c *Canvas) boxCells(id int, f func(Box) []Point) []Point {
	if box, ok := c.box(id); ok {
		return f(box)
	}
	return nil
}

func (c *Canvas) GetBoxCells(id int) []Point {
	return c.boxCells(id, func(b Box) []Point {
		cells := make([]Point, 0, b.Width*b.Height)
		for y := b.Y; y < b.Y+b.Height; y++ {
			for x := b.X; x < b.X+b.Width; x++ {
				cells = append(cells, Point{x, y})
			}
		}
		return cells
	})
}

func (c *Canvas) GetBoxBorderCells(id int) []Point {
	return c.boxCells(id, func(b Box) []Point {
		cells := make([]Point, 0, 2*(b.Width+b.Height))
		for x := b.X; x < b.X+b.Width; x++ {
			cells = append(cells, Point{x, b.Y}, Point{x, b.Y + b.Height - 1})
		}
		for y := b.Y + 1; y < b.Y+b.Height-1; y++ {
			cells = append(cells, Point{b.X, y}, Point{b.X + b.Width - 1, y})
		}
		return cells
	})
}

func titleDividerY(box Box) (int, bool) {
	if box.Title == "" {
		return 0, false
	}
	return box.Y + 1 + len(strings.Split(box.Title, "\n")), true
}

func dividerCells(b Box) []Point {
	dividerY, ok := titleDividerY(b)
	if !ok {
		return nil
	}
	cells := make([]Point, 0, b.Width)
	for x := b.X + 1; x < b.X+b.Width-1; x++ {
		cells = append(cells, Point{x, dividerY})
	}
	return cells
}

func (c *Canvas) GetBoxTitleDividerCells(id int) []Point {
	return c.boxCells(id, dividerCells)
}

func (c *Canvas) GetBoxTitleBarCells(id int) []Point {
	return c.boxCells(id, func(b Box) []Point {
		dividerY, ok := titleDividerY(b)
		if !ok {
			return nil
		}
		cells := make([]Point, 0)
		for x := b.X; x < b.X+b.Width; x++ {
			cells = append(cells, Point{x, b.Y})
		}
		for y := b.Y + 1; y <= dividerY; y++ {
			cells = append(cells, Point{b.X, y}, Point{b.X + b.Width - 1, y})
		}
		return append(cells, dividerCells(b)...)
	})
}

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

func (c *Canvas) GetBoxTitleTextCells(id int) []Point {
	return c.boxCells(id, func(b Box) []Point {
		if b.Title == "" {
			return nil
		}
		return lineCells(strings.Split(b.Title, "\n"), b.X+boxInsetX, b.Y+1, b.Width-2*boxInsetX)
	})
}

func (c *Canvas) GetBoxContentTextCells(id int) []Point {
	return c.boxCells(id, func(b Box) []Point {
		return lineCells(b.Lines, b.X+boxInsetX, b.Y+contentStartLine(b), b.Width-2*boxInsetX)
	})
}

func (c *Canvas) GetTextCells(textID int) []Point {
	t := c.textPtr(textID)
	if t == nil {
		return nil
	}
	return lineCells(t.Lines, t.X, t.Y, -1)
}

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
			if color, exists := c.highlights[Point{box.X + boxInsetX + col, box.Y + 1 + row}]; exists {
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
