package canvas

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type checker struct {
	errs   []string
	line   int
	boxes  []Box
	nBoxes int
	nConns int
	nTexts int
}

func (v *checker) fail(format string, args ...any) {
	v.errs = append(v.errs, fmt.Sprintf("line %d: ", v.line)+fmt.Sprintf(format, args...))
}

func (v *checker) ints(fields []string, what string) ([]int, bool) {
	out := make([]int, len(fields))
	for i, f := range fields {
		n, err := strconv.Atoi(strings.TrimSpace(f))
		if err != nil {
			v.fail("%s: %q is not an integer", what, f)
			return nil, false
		}
		out[i] = n
	}
	return out, true
}

func (v *checker) boxLine(text string) {
	f := splitBoxLine(text)
	if len(f) < 8 {
		v.fail("box needs 8 fields (x,y,w,h,z,style,title,text), got %d", len(f))
		return
	}
	n, ok := v.ints(f[:6], "box")
	if !ok {
		return
	}
	if n[4] < 0 || n[4] >= numZLevels {
		v.fail("box z-level %d out of range 0-%d", n[4], numZLevels-1)
	}
	if n[5] < 0 || n[5] > int(BorderStyleRounded) {
		v.fail("box border style %d out of range 0-%d", n[5], int(BorderStyleRounded))
	}
	v.boxes = append(v.boxes, Box{ID: len(v.boxes), X: n[0], Y: n[1], Width: n[2], Height: n[3]})
}

func onBorder(b Box, x, y int) bool {
	right, bottom := b.X+b.Width-1, b.Y+b.Height-1
	if x < b.X || x > right || y < b.Y || y > bottom {
		return false
	}
	return x == b.X || x == right || y == b.Y || y == bottom
}

func (v *checker) connLine(text string) {
	main, wps, _ := strings.Cut(text, "|")
	f := strings.Split(main, ",")
	if len(f) != 2 && len(f) < 7 {
		v.fail("connection needs 2 or 7+ fields, got %d", len(f))
		return
	}
	n, ok := v.ints(f, "connection")
	if !ok {
		return
	}
	for _, id := range n[:2] {
		if id < -1 || id >= len(v.boxes) {
			v.fail("connection box id %d out of range (%d boxes)", id, len(v.boxes))
			return
		}
	}
	if len(n) < 7 {
		return
	}
	if len(n) >= 8 && (n[7] < 0 || n[7] > 3) {
		v.fail("connection arrow flags %d out of range 0-3", n[7])
	}
	var wp []string
	if wps != "" {
		wp = strings.Split(wps, ",")
	}
	if len(wp) != n[6] {
		v.fail("connection declares %d waypoints but lists %d", n[6], len(wp))
	}
	for _, w := range wp {
		if x, y, ok := strings.Cut(w, ":"); !ok || x == "" || y == "" {
			v.fail("bad waypoint %q (want x:y)", w)
		}
	}
	for i, end := range [][3]int{{n[0], n[2], n[3]}, {n[1], n[4], n[5]}} {
		if end[0] >= 0 && !onBorder(v.boxes[end[0]], end[1], end[2]) {
			v.errs = append(v.errs, fmt.Sprintf("warning: line %d: connection end %d (%d,%d) is not on the border of box %d", v.line, i+1, end[1], end[2], end[0]))
		}
	}
}

func (v *checker) sectionLine(header, text string) {
	f := strings.Split(text, ",")
	switch header {
	case "TEXTS":
		if len(f) < 3 {
			v.fail("text needs x,y,text")
			return
		}
		v.ints(f[:2], "text")
	case "TOOLTIPS":
		id, _, ok := strings.Cut(text, ",")
		n, err := strconv.Atoi(id)
		if !ok || err != nil || n < 0 || n >= len(v.boxes) {
			v.fail("tooltip needs a valid box id then text")
		}
	case "HIGHLIGHTS":
		if len(f) != 3 {
			v.fail("highlight needs x,y,color")
		} else if n, ok := v.ints(f, "highlight"); ok && (n[2] < 0 || n[2] >= NumColors) {
			v.fail("highlight color %d out of range 0-%d", n[2], NumColors-1)
		}
	case "BOXCOLORS", "LINECOLORS", "TEXTCOLORS":
		limit := map[string]int{"BOXCOLORS": v.nBoxes, "LINECOLORS": v.nConns, "TEXTCOLORS": v.nTexts}[header]
		if len(f) != 2 {
			v.fail("color entry needs index,color")
		} else if n, ok := v.ints(f, "color"); ok {
			if n[0] < 0 || n[0] >= limit {
				v.fail("%s index %d out of range (%d items)", header, n[0], limit)
			}
			if n[1] < 0 || n[1] >= NumColors {
				v.fail("color %d out of range 0-%d", n[1], NumColors-1)
			}
		}
	}
}

func Validate(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	v := &checker{}
	sc := bufio.NewScanner(file)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	next := func() (string, bool) {
		ok := sc.Scan()
		v.line++
		return sc.Text(), ok
	}
	if first, _ := next(); first != "FLOWCHART" {
		return []string{"line 1: first line must be FLOWCHART"}, nil
	}
	for {
		text, ok := next()
		if !ok {
			break
		}
		header, arg, ok := strings.Cut(text, ":")
		if !ok {
			v.fail("expected a section header like BOXES:3, got %q", text)
			continue
		}
		if header == "PAN" {
			v.ints(strings.Split(arg, ","), "pan")
			continue
		}
		count, err := strconv.Atoi(arg)
		if err != nil || count < 0 {
			v.fail("%s count %q is not a number", header, arg)
			continue
		}
		switch header {
		case "BOXES":
			v.nBoxes = count
		case "CONNECTIONS":
			v.nConns = count
		case "TEXTS":
			v.nTexts = count
		}
		for i := 0; i < count; i++ {
			line, ok := next()
			if !ok {
				v.fail("%s declares %d lines but the file ends after %d", header, count, i)
				break
			}
			switch header {
			case "BOXES":
				v.boxLine(line)
			case "CONNECTIONS":
				v.connLine(line)
			default:
				v.sectionLine(header, line)
			}
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	for i, a := range v.boxes {
		for _, b := range v.boxes[i+1:] {
			if a.X < b.X+b.Width && b.X < a.X+a.Width && a.Y < b.Y+b.Height && b.Y < a.Y+a.Height {
				v.errs = append(v.errs, fmt.Sprintf("warning: boxes %d and %d overlap", a.ID, b.ID))
			}
		}
	}
	return v.errs, nil
}
