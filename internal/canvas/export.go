package canvas

import (
	"fmt"
	"io"
	"strings"
)

func (c *Canvas) WriteText(w io.Writer) error {
	minX, minY, maxX, maxY := c.GetFullBounds()
	if minX > maxX || minY > maxY {
		return fmt.Errorf("nothing to export")
	}

	const padding = 1
	minX -= padding
	minY -= padding
	width := maxX - minX + padding + 1
	height := maxY - minY + padding + 1

	opts := NewRenderOpts(width, height)
	opts.PanX, opts.PanY = minX, minY
	r := c.RenderRaw(opts)
	for _, row := range r.Canvas {
		if _, err := fmt.Fprintln(w, strings.TrimRight(string(row), " ")); err != nil {
			return err
		}
	}
	return nil
}

func (c *Canvas) WriteMermaid(w io.Writer) error {
	boxes := c.boxes
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
		if b.BorderStyle == BorderStyleRounded {
			open, closing = "(", ")"
		}
		fmt.Fprintf(&sb, "    n%d%s\"%s\"%s\n", b.ID, open, label, closing)
	}
	conns := c.connections
	for i, cn := range conns {
		if cn.FromID < 0 {
			cn.FromID, cn.ArrowFrom = branchOrigin(conns, i, Point{X: cn.FromX, Y: cn.FromY}, 0), false
		}
		if cn.ToID < 0 {
			cn.ToID, cn.ArrowTo = branchOrigin(conns, i, Point{X: cn.ToX, Y: cn.ToY}, 0), false
		}
		if cn.FromID < 0 || cn.ToID < 0 || cn.FromID == cn.ToID {
			continue
		}
		from, to, link := cn.FromID, cn.ToID, "---"
		switch {
		case cn.ArrowFrom && cn.ArrowTo:
			link = "<-->"
		case cn.ArrowTo:
			link = "-->"
		case cn.ArrowFrom:
			from, to, link = to, from, "-->"
		}
		fmt.Fprintf(&sb, "    n%d %s n%d\n", from, link, to)
	}
	_, err := io.WriteString(w, sb.String())
	return err
}

func branchOrigin(conns []Connection, self int, p Point, depth int) int {
	if depth > len(conns) {
		return -1
	}
	for i, c := range conns {
		if i == self {
			continue
		}
		path := append(append([]Point{{X: c.FromX, Y: c.FromY}}, c.Waypoints...), Point{X: c.ToX, Y: c.ToY})
		for j := 1; j < len(path); j++ {
			a, b := path[j-1], path[j]
			if p.X >= min(a.X, b.X) && p.X <= max(a.X, b.X) && p.Y >= min(a.Y, b.Y) && p.Y <= max(a.Y, b.Y) {
				if c.FromID >= 0 {
					return c.FromID
				}
				return branchOrigin(conns, i, Point{X: c.FromX, Y: c.FromY}, depth+1)
			}
		}
	}
	return -1
}
