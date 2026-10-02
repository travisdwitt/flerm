package canvas

import (
	"fmt"
	"io"
	"strings"
)

var ColorNames = []string{"Gray", "Red", "Green", "Yellow", "Blue", "Magenta", "Cyan", "White",
	"Black", "Bright Red", "Bright Green", "Bright Yellow", "Bright Blue", "Bright Magenta", "Bright Cyan", "Bright White"}

var ChartExts = []string{".flerm", ".sav"}

func (s BorderStyle) String() string {
	switch s {
	case BorderStyleSingle:
		return "single"
	case BorderStyleDouble:
		return "double"
	case BorderStyleRounded:
		return "rounded"
	}
	return "ascii"
}

func colorNote(color int) string {
	if color < 0 || color >= len(ColorNames) {
		return ""
	}
	return ", color " + ColorNames[color]
}

func oneLine(s string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(s, "\n", " ")), " ")
}

func (c *Canvas) boxLabel(id int) string {
	b, ok := c.box(id)
	if !ok {
		return fmt.Sprintf("Box %d", id)
	}
	for _, s := range []string{b.Title, b.GetText()} {
		if label := oneLine(s); label != "" {
			return fmt.Sprintf("Box %d %q", id, label)
		}
	}
	return fmt.Sprintf("Box %d (empty)", id)
}

func (c *Canvas) endLabel(id, x, y int) string {
	if id >= 0 && id < len(c.boxes) {
		return c.boxLabel(id)
	}
	return fmt.Sprintf("the point %d,%d", x, y)
}

func count(n int, noun string) string {
	switch {
	case n == 1:
		return fmt.Sprintf("%d %s", n, noun)
	case strings.HasSuffix(noun, "x"):
		return fmt.Sprintf("%d %ses", n, noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

func arrowsRelative(conn Connection, nearIsFrom bool) string {
	near, far := conn.ArrowFrom, conn.ArrowTo
	if !nearIsFrom {
		near, far = far, near
	}
	switch {
	case near && far:
		return ", arrows at both ends"
	case far:
		return ", arrow at the far end"
	case near:
		return ", arrow at this box"
	}
	return ", no arrows"
}

func (c *Canvas) arrowNote(conn Connection) string {
	var notes []string
	if conn.ArrowTo {
		notes = append(notes, "arrow at "+c.endLabel(conn.ToID, conn.ToX, conn.ToY))
	}
	if conn.ArrowFrom {
		notes = append(notes, "arrow at "+c.endLabel(conn.FromID, conn.FromX, conn.FromY))
	}
	if len(notes) == 0 {
		return ", no arrows"
	}
	return ", " + strings.Join(notes, " and ")
}

func (c *Canvas) Describe(w io.Writer, name string) {
	fmt.Fprintf(w, "Chart: %s\n", name)
	fmt.Fprintf(w, "%s, %s, %s, %s\n", count(len(c.boxes), "box"), count(len(c.connections), "connection"),
		count(len(c.texts), "text label"), count(len(c.highlights), "highlighted cell"))

	if len(c.boxes) == 0 && len(c.connections) == 0 && len(c.texts) == 0 {
		fmt.Fprintln(w, "\n(empty chart)")
		return
	}

	for id, box := range c.boxes {
		fmt.Fprintf(w, "\n%s at %d,%d size %dx%d, %s border%s\n",
			c.boxLabel(id), box.X, box.Y, box.Width, box.Height, box.BorderStyle, colorNote(box.Color))
		if box.ZLevel > 0 {
			fmt.Fprintf(w, "  shadow depth %d\n", box.ZLevel)
		}
		for _, field := range []struct{ name, value string }{
			{"title", box.Title}, {"text", box.GetText()}, {"tooltip", box.Tooltip},
		} {
			if v := oneLine(field.value); v != "" {
				fmt.Fprintf(w, "  %s: %s\n", field.name, v)
			}
		}
		for _, conn := range c.connections {
			switch {
			case conn.FromID == id:
				fmt.Fprintf(w, "  connects to %s%s\n", c.endLabel(conn.ToID, conn.ToX, conn.ToY), arrowsRelative(conn, true))
			case conn.ToID == id:
				fmt.Fprintf(w, "  connected from %s%s\n", c.endLabel(conn.FromID, conn.FromX, conn.FromY), arrowsRelative(conn, false))
			}
		}
	}

	if len(c.texts) > 0 {
		fmt.Fprintln(w)
		for id, text := range c.texts {
			fmt.Fprintf(w, "Text %d at %d,%d%s: %s\n", id, text.X, text.Y, colorNote(text.Color), oneLine(text.GetText()))
		}
	}

	for id, conn := range c.connections {
		if conn.FromID != -1 || conn.ToID != -1 {
			continue
		}
		fmt.Fprintf(w, "\nConnection %d from %d,%d to %d,%d%s%s\n",
			id, conn.FromX, conn.FromY, conn.ToX, conn.ToY, colorNote(conn.Color), c.arrowNote(conn))
	}
}
