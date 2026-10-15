package tui

import (
	"fmt"
	"os"
	"strings"

	cv "flerm/internal/canvas"
)

func (m *model) exportVisualTXT(filename string) error {
	canvas := m.getCanvas()
	minX, minY, maxX, maxY := canvas.GetFullBounds()
	if minX > maxX || minY > maxY {
		return fmt.Errorf("nothing to export")
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	const padding = 1
	minX -= padding
	minY -= padding
	width := maxX - minX + padding + 1
	height := maxY - minY + padding + 1

	r := canvas.RenderRaw(width, height, -1, CoordUnset, CoordUnset, nil, CoordUnset, CoordUnset, minX, minY, -1, -1, false, -1, -1, 0, "", CoordUnset, CoordUnset, CoordUnset, CoordUnset, CoordUnset, CoordUnset, false, -1, -1)
	for _, row := range r.Canvas {
		fmt.Fprintln(file, strings.TrimRight(string(row), " "))
	}
	return nil
}

func (m *model) exportMermaid(filename string) error {
	canvas := m.getCanvas()
	boxes := canvas.Boxes()
	if len(boxes) == 0 {
		return fmt.Errorf("nothing to export")
	}

	var sb strings.Builder
	sb.WriteString("flowchart TD\n")
	for _, b := range boxes {
		label := strings.Join(b.Lines, `\n`)
		if b.Title != "" {
			label = strings.ReplaceAll(b.Title, "\n", `\n`) + `\n` + label
		}
		label = strings.ReplaceAll(label, `"`, "#quot;")
		open, closing := "[", "]"
		if b.BorderStyle == cv.BorderStyleRounded {
			open, closing = "(", ")"
		}
		fmt.Fprintf(&sb, "    n%d%s\"%s\"%s\n", b.ID, open, label, closing)
	}
	conns := canvas.Connections()
	for i, c := range conns {
		if c.FromID < 0 {
			c.FromID, c.ArrowFrom = branchOrigin(conns, i, cv.Point{X: c.FromX, Y: c.FromY}, 0), false
		}
		if c.ToID < 0 {
			c.ToID, c.ArrowTo = branchOrigin(conns, i, cv.Point{X: c.ToX, Y: c.ToY}, 0), false
		}
		if c.FromID < 0 || c.ToID < 0 || c.FromID == c.ToID {
			continue
		}
		from, to, link := c.FromID, c.ToID, "---"
		switch {
		case c.ArrowFrom && c.ArrowTo:
			link = "<-->"
		case c.ArrowTo:
			link = "-->"
		case c.ArrowFrom:
			from, to, link = to, from, "-->"
		}
		fmt.Fprintf(&sb, "    n%d %s n%d\n", from, link, to)
	}
	return os.WriteFile(filename, []byte(sb.String()), 0o644)
}

func branchOrigin(conns []cv.Connection, self int, p cv.Point, depth int) int {
	if depth > len(conns) {
		return -1
	}
	for i, c := range conns {
		if i == self {
			continue
		}
		path := append(append([]cv.Point{{X: c.FromX, Y: c.FromY}}, c.Waypoints...), cv.Point{X: c.ToX, Y: c.ToY})
		for j := 1; j < len(path); j++ {
			a, b := path[j-1], path[j]
			if p.X >= min(a.X, b.X) && p.X <= max(a.X, b.X) && p.Y >= min(a.Y, b.Y) && p.Y <= max(a.Y, b.Y) {
				if c.FromID >= 0 {
					return c.FromID
				}
				return branchOrigin(conns, i, cv.Point{X: c.FromX, Y: c.FromY}, depth+1)
			}
		}
	}
	return -1
}
