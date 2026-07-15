package canvas

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

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

		encodedTitle := strings.ReplaceAll(box.Title, "\n", "\\n")
		encodedTitle = strings.ReplaceAll(encodedTitle, ",", "\\,")

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

		var x, y, width, height, zLevel int
		var borderStyle BorderStyle
		var title, text string

		fields := splitBoxLine(line)

		if len(fields) < 3 {
			return fmt.Errorf("invalid box format")
		}

		x, _ = strconv.Atoi(fields[0])
		y, _ = strconv.Atoi(fields[1])

		if len(fields) >= 8 {

			width, _ = strconv.Atoi(fields[2])
			height, _ = strconv.Atoi(fields[3])
			zLevel, _ = strconv.Atoi(fields[4])
			if zLevel < 0 || zLevel > 3 {
				zLevel = 0
			}
			borderStyleInt, _ := strconv.Atoi(fields[5])
			borderStyle = BorderStyle(borderStyleInt)

			title = strings.ReplaceAll(fields[6], "\\,", ",")
			title = strings.ReplaceAll(title, "\\n", "\n")
			text = strings.ReplaceAll(strings.Join(fields[7:], ","), "\\n", "\n")
		} else if len(fields) >= 6 {

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

			width, _ = strconv.Atoi(fields[2])
			height, _ = strconv.Atoi(fields[3])
			zLevel = 0
			borderStyle = BorderStyleASCII
			title = ""
			text = strings.ReplaceAll(strings.Join(fields[4:], ","), "\\n", "\n")
		} else {

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

			var waypoints []Point
			if len(parts) > 1 && waypointCount > 0 {
				waypointParts := strings.Split(parts[1], ",")
				for j := 0; j < waypointCount && j < len(waypointParts); j++ {
					wpParts := strings.Split(waypointParts[j], ":")
					if len(wpParts) == 2 {
						wpX, _ := strconv.Atoi(wpParts[0])
						wpY, _ := strconv.Atoi(wpParts[1])
						waypoints = append(waypoints, Point{wpX, wpY})
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
						if colorIndex >= 0 && colorIndex < NumColors {
							c.SetHighlight(x, y, colorIndex)
						}
					}
				}
			}
		}
	}

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
			if err1 != nil || err2 != nil || col < 0 || col >= NumColors {
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
