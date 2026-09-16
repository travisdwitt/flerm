package canvas

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func escapeNewlines(s string) string { return strings.ReplaceAll(s, "\n", "\\n") }

func unescapeNewlines(s string) string { return strings.ReplaceAll(s, "\\n", "\n") }

func (c *Canvas) SaveToFileWithPan(filename string, panX, panY int) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Fprintln(file, "FLOWCHART")

	fmt.Fprintf(file, "BOXES:%d\n", len(c.boxes))
	for _, box := range c.boxes {
		title := strings.ReplaceAll(escapeNewlines(box.Title), ",", "\\,")
		fmt.Fprintf(file, "%d,%d,%d,%d,%d,%d,%s,%s\n",
			box.X, box.Y, box.Width, box.Height, box.ZLevel, box.BorderStyle,
			title, escapeNewlines(box.GetText()))
	}

	fmt.Fprintf(file, "CONNECTIONS:%d\n", len(c.connections))
	for _, conn := range c.connections {
		waypointsStr := ""
		if len(conn.Waypoints) > 0 {
			parts := make([]string, len(conn.Waypoints))
			for i, wp := range conn.Waypoints {
				parts[i] = fmt.Sprintf("%d:%d", wp.X, wp.Y)
			}
			waypointsStr = "|" + strings.Join(parts, ",")
		}
		arrowFlags := 0
		if conn.ArrowFrom {
			arrowFlags |= 1
		}
		if conn.ArrowTo {
			arrowFlags |= 2
		}
		fmt.Fprintf(file, "%d,%d,%d,%d,%d,%d,%d,%d%s\n",
			conn.FromID, conn.ToID, conn.FromX, conn.FromY, conn.ToX, conn.ToY,
			len(conn.Waypoints), arrowFlags, waypointsStr)
	}

	fmt.Fprintf(file, "TEXTS:%d\n", len(c.texts))
	for _, text := range c.texts {
		fmt.Fprintf(file, "%d,%d,%s\n", text.X, text.Y, escapeNewlines(text.GetText()))
	}

	fmt.Fprintf(file, "HIGHLIGHTS:%d\n", len(c.highlights))
	for cell, colorIndex := range c.highlights {
		fmt.Fprintf(file, "%d,%d,%d\n", cell.X, cell.Y, colorIndex)
	}

	// Colour sections list only the objects that have one, so older versions of
	// Flerm (which skip unknown sections) still read the file.
	writeColors := func(header string, colorOf func(i int) int, n int) {
		var lines []string
		for i := 0; i < n; i++ {
			if col := colorOf(i); col >= 0 {
				lines = append(lines, fmt.Sprintf("%d,%d", i, col))
			}
		}
		fmt.Fprintf(file, "%s:%d\n", header, len(lines))
		for _, line := range lines {
			fmt.Fprintln(file, line)
		}
	}
	writeColors("BOXCOLORS", func(i int) int { return c.boxes[i].Color }, len(c.boxes))
	writeColors("LINECOLORS", func(i int) int { return c.connections[i].Color }, len(c.connections))
	writeColors("TEXTCOLORS", func(i int) int { return c.texts[i].Color }, len(c.texts))

	fmt.Fprintf(file, "PAN:%d,%d\n", panX, panY)
	return nil
}

// splitBoxLine splits on commas, honouring the "\," escape used in box titles.
func splitBoxLine(line string) []string {
	var fields []string
	var current strings.Builder
	for i := 0; i < len(line); i++ {
		switch {
		case line[i] == '\\' && i+1 < len(line) && line[i+1] == ',':
			current.WriteString("\\,")
			i++
		case line[i] == ',':
			fields = append(fields, current.String())
			current.Reset()
		default:
			current.WriteByte(line[i])
		}
	}
	return append(fields, current.String())
}

func (c *Canvas) LoadFromFileWithPan(filename string) (int, int, error) {
	file, err := os.Open(filename)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	c.boxes, c.connections, c.texts = c.boxes[:0], c.connections[:0], c.texts[:0]
	c.highlights = make(map[Point]int)

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() || scanner.Text() != "FLOWCHART" {
		return 0, 0, fmt.Errorf("invalid file format")
	}

	// Section count line, e.g. "BOXES:3".
	section := func(name string) (int, error) {
		if !scanner.Scan() {
			return 0, fmt.Errorf("missing %s header", strings.ToLower(name))
		}
		n, err := strconv.Atoi(strings.TrimPrefix(scanner.Text(), name+":"))
		if err != nil {
			return 0, fmt.Errorf("invalid %s count: %v", strings.ToLower(name), err)
		}
		return n, nil
	}

	boxCount, err := section("BOXES")
	if err != nil {
		return 0, 0, err
	}
	for i := 0; i < boxCount; i++ {
		if !scanner.Scan() {
			return 0, 0, fmt.Errorf("missing box data")
		}
		box, err := parseBox(scanner.Text(), i)
		if err != nil {
			return 0, 0, err
		}
		c.boxes = append(c.boxes, box)
	}

	connCount, err := section("CONNECTIONS")
	if err != nil {
		return 0, 0, err
	}
	for i := 0; i < connCount; i++ {
		if !scanner.Scan() {
			return 0, 0, fmt.Errorf("missing connection data")
		}
		if err := c.parseConnection(scanner.Text()); err != nil {
			return 0, 0, err
		}
	}

	// Everything past this point is optional and self-describing.
	panX, panY := 0, 0
	for scanner.Scan() {
		header, arg, ok := strings.Cut(scanner.Text(), ":")
		if !ok {
			continue
		}
		if header == "PAN" {
			if parts := strings.Split(arg, ","); len(parts) >= 2 {
				panX, _ = strconv.Atoi(parts[0])
				panY, _ = strconv.Atoi(parts[1])
			}
			continue
		}
		count, err := strconv.Atoi(arg)
		if err != nil {
			continue
		}
		for i := 0; i < count && scanner.Scan(); i++ {
			c.loadSectionLine(header, scanner.Text())
		}
	}

	return panX, panY, scanner.Err()
}

// parseBox reads one BOXES line. Older files carry fewer fields; anything
// missing falls back to a default.
func parseBox(line string, id int) (Box, error) {
	fields := splitBoxLine(line)
	if len(fields) < 3 {
		return Box{}, fmt.Errorf("invalid box format")
	}
	box := Box{ID: id, Color: -1}
	box.X, _ = strconv.Atoi(fields[0])
	box.Y, _ = strconv.Atoi(fields[1])

	textFrom := 2
	width, height := 0, 0
	if len(fields) >= 5 {
		width, _ = strconv.Atoi(fields[2])
		height, _ = strconv.Atoi(fields[3])
		textFrom = 4
	}
	if len(fields) >= 6 {
		if z, _ := strconv.Atoi(fields[4]); z >= 0 && z < numZLevels {
			box.ZLevel = z
		}
		textFrom = 5
	}
	if len(fields) >= 8 {
		style, _ := strconv.Atoi(fields[5])
		box.BorderStyle = BorderStyle(style)
		box.Title = unescapeNewlines(strings.ReplaceAll(fields[6], "\\,", ","))
		textFrom = 7
	}

	box.SetText(unescapeNewlines(strings.Join(fields[textFrom:], ",")))
	if len(fields) >= 5 {
		box.Width, box.Height = width, height
	}
	return box, nil
}

func (c *Canvas) parseConnection(line string) error {
	main, waypointStr, _ := strings.Cut(line, "|")
	parts := strings.Split(main, ",")

	if len(parts) == 2 {
		fromID, _ := strconv.Atoi(parts[0])
		toID, _ := strconv.Atoi(parts[1])
		if fromID >= 0 && fromID < len(c.boxes) && toID >= 0 && toID < len(c.boxes) {
			c.AddConnection(fromID, toID)
		}
		return nil
	}
	if len(parts) < 7 {
		return fmt.Errorf("invalid connection format")
	}

	n := make([]int, 8)
	n[7] = 2 // default: arrow on the "to" end only
	for i := 0; i < len(parts) && i < 8; i++ {
		n[i], _ = strconv.Atoi(parts[i])
	}

	var waypoints []Point
	if waypointStr != "" && n[6] > 0 {
		for i, wp := range strings.Split(waypointStr, ",") {
			if i >= n[6] {
				break
			}
			if x, y, ok := strings.Cut(wp, ":"); ok {
				wx, _ := strconv.Atoi(x)
				wy, _ := strconv.Atoi(y)
				waypoints = append(waypoints, Point{wx, wy})
			}
		}
	}

	c.connections = append(c.connections, Connection{
		FromID: n[0], ToID: n[1],
		FromX: n[2], FromY: n[3], ToX: n[4], ToY: n[5],
		Waypoints: waypoints,
		ArrowFrom: n[7]&1 != 0,
		ArrowTo:   n[7]&2 != 0,
		Color:     -1,
	})
	return nil
}

func (c *Canvas) loadSectionLine(header, line string) {
	switch header {
	case "TEXTS":
		x, rest, ok := strings.Cut(line, ",")
		if !ok {
			return
		}
		y, text, ok := strings.Cut(rest, ",")
		if !ok {
			return
		}
		tx, err1 := strconv.Atoi(x)
		ty, err2 := strconv.Atoi(y)
		if err1 == nil && err2 == nil {
			c.AddText(tx, ty, unescapeNewlines(text))
		}
		return
	}

	parts := strings.Split(line, ",")
	if len(parts) < 2 {
		return
	}
	a, err1 := strconv.Atoi(parts[0])
	b, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return
	}

	if header == "HIGHLIGHTS" {
		if len(parts) >= 3 {
			if col, err := strconv.Atoi(parts[2]); err == nil {
				c.SetHighlight(a, b, col)
			}
		}
		return
	}
	if b < 0 || b >= NumColors {
		return
	}
	switch header {
	case "BOXCOLORS":
		c.SetBoxColor(a, b)
	case "LINECOLORS":
		c.SetLineColor(a, b)
	case "TEXTCOLORS":
		c.SetTextColor(a, b)
	}
}
