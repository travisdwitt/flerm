package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	cv "flerm/internal/canvas"
	"flerm/internal/config"
)

type Op struct {
	Desc string
	Pos  []string
	Opts []string
	Run  func(p map[string]string) (string, error)
}

var Ops = map[string]Op{
	"add-box": {
		Desc: "Add a box to a chart (created if missing). Prints the new box id. at defaults to below the lowest box. border: ascii|single|double|rounded. color: a color name or none. size: WxH (boxes size themselves to their text; this only enlarges). shadow: 0-3.",
		Pos:  []string{"chart"}, Opts: []string{"text", "title", "tooltip", "at", "border", "color", "size", "shadow"},
		Run: func(p map[string]string) (string, error) { return edit(p, addBox) },
	},
	"set-box": {
		Desc: "Change a box. Only the options given are changed; an empty value clears. at moves the box to x,y.",
		Pos:  []string{"chart", "id"}, Opts: []string{"text", "title", "tooltip", "at", "border", "color", "size", "shadow"},
		Run: func(p map[string]string) (string, error) { return edit(p, setBox) },
	},
	"rm-box": {
		Desc: "Delete a box and its connections. Higher box ids shift down by one.",
		Pos:  []string{"chart", "id"},
		Run:  func(p map[string]string) (string, error) { return edit(p, rmBox) },
	},
	"connect": {
		Desc: "Connect box from to box to. Prints the connection id. arrow: to (default)|from|both|none.",
		Pos:  []string{"chart", "from", "to"}, Opts: []string{"arrow"},
		Run: func(p map[string]string) (string, error) { return edit(p, connect) },
	},
	"rm-conn": {
		Desc: "Delete a connection by id. Higher connection ids shift down by one.",
		Pos:  []string{"chart", "id"},
		Run:  func(p map[string]string) (string, error) { return edit(p, rmConn) },
	},
	"add-text": {
		Desc: "Add a free text label to a chart. at defaults to below everything. color: a color name or none. Prints the new text id.",
		Pos:  []string{"chart"}, Opts: []string{"text", "at", "color"},
		Run: func(p map[string]string) (string, error) { return edit(p, addText) },
	},
	"set-text": {
		Desc: "Change a text label. Only the options given are changed. at moves it to x,y.",
		Pos:  []string{"chart", "id"}, Opts: []string{"text", "at", "color"},
		Run: func(p map[string]string) (string, error) { return edit(p, setText) },
	},
	"rm-text": {
		Desc: "Delete a text label by id. Higher text ids shift down by one.",
		Pos:  []string{"chart", "id"},
		Run:  func(p map[string]string) (string, error) { return edit(p, rmText) },
	},
	"batch": {
		Desc: `Run several edits with one read and one write. ops is a JSON array such as [{"op":"add-box","text":"A"},{"op":"connect","from":0,"to":1}], or - to read it from stdin. Ops are add-box, set-box, rm-box, connect, rm-conn, add-text, set-text, rm-text with the same options as the commands. Ids refer to the chart as earlier ops left it (new boxes get the next id). All or nothing: on any error nothing is saved. Prints one line per op.`,
		Pos:  []string{"chart", "ops"},
		Run:  runBatch,
	},
	"find": {
		Desc: "List boxes whose title, text or tooltip contains query (case-insensitive).",
		Pos:  []string{"chart", "query"},
		Run:  find,
	},
	"import": {
		Desc: "Create a chart from a Mermaid flowchart or JSON spec, auto-laid-out; edge labels become text labels. source is a file path, - for stdin, or the inline text. Refuses to overwrite unless force=true. append=true adds the new graph below an existing chart instead (box ids continue after the existing ones).",
		Pos:  []string{"source", "chart"}, Opts: []string{"force", "append"},
		Run: runImport,
	},
	"describe": {
		Desc: "Print a chart's contents. format: text (default)|json.",
		Pos:  []string{"chart"}, Opts: []string{"format"},
		Run: describe,
	},
	"validate": {
		Desc: "Check a chart file for format errors. Fails when there are errors; warnings are informational.",
		Pos:  []string{"chart"},
		Run:  validate,
	},
	"export": {
		Desc: "Export a chart. format: mermaid|txt|png. mermaid and txt print to stdout unless out is given; png needs out (default: chart name in the current directory).",
		Pos:  []string{"chart"}, Opts: []string{"format", "out"},
		Run: export,
	},
}

func FindChart(name string) string {
	names := []string{name}
	if filepath.Ext(name) == "" {
		names = nil
		for _, ext := range cv.ChartExts {
			names = append(names, name+ext)
		}
	}
	dirs := []string{""}
	if dir := config.Load().SaveDirectory; dir != "" {
		dirs = append(dirs, dir)
	}
	for _, n := range names {
		for _, dir := range dirs {
			path := filepath.Join(dir, n)
			if info, err := os.Stat(path); err == nil && !info.IsDir() {
				return path
			}
		}
	}
	return ""
}

func ParseArgs(name string, args []string) (map[string]string, error) {
	op := Ops[name]
	p := map[string]string{}
	var pos []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		key, ok := strings.CutPrefix(arg, "--")
		if !ok {
			pos = append(pos, arg)
			continue
		}
		val := ""
		if k, v, has := strings.Cut(key, "="); has {
			key, val = k, v
		} else if i+1 < len(args) {
			i++
			val = args[i]
		} else {
			return nil, fmt.Errorf("--%s needs a value", key)
		}
		if !slices.Contains(op.Opts, key) {
			return nil, fmt.Errorf("unknown option --%s for %s", key, name)
		}
		p[key] = val
	}
	if len(pos) != len(op.Pos) {
		return nil, fmt.Errorf("usage: flerm %s %s", name, usage(op))
	}
	for i, k := range op.Pos {
		p[k] = pos[i]
	}
	return p, nil
}

func usage(op Op) string {
	parts := []string{}
	for _, k := range op.Pos {
		parts = append(parts, "<"+k+">")
	}
	for _, k := range op.Opts {
		parts = append(parts, "[--"+k+" v]")
	}
	return strings.Join(parts, " ")
}

func text(s string) string { return strings.ReplaceAll(s, `\n`, "\n") }

func chartPath(name string) string {
	if path := FindChart(name); path != "" {
		return path
	}
	if !slices.Contains(cv.ChartExts, strings.ToLower(filepath.Ext(name))) {
		name += ".flerm"
	}
	return name
}

func edit(p map[string]string, fn func(*cv.Canvas, map[string]string) (string, error)) (string, error) {
	path := chartPath(p["chart"])
	c := cv.NewCanvas()
	panX, panY := 0, 0
	if _, err := os.Stat(path); err == nil {
		var err error
		if panX, panY, err = c.LoadFromFileWithPan(path); err != nil {
			return "", fmt.Errorf("reading %s: %w", path, err)
		}
	}
	out, err := fn(c, p)
	if err != nil {
		return "", err
	}
	if err := c.SaveToFileWithPan(path, panX, panY); err != nil {
		return "", err
	}
	return out, nil
}

func idArg(p map[string]string, key string, n int, what string) (int, error) {
	id, err := strconv.Atoi(p[key])
	if err != nil || id < 0 || id >= n {
		return 0, fmt.Errorf("%s %q not found (have %d)", what, p[key], n)
	}
	return id, nil
}

func parseAt(s string) (int, int, error) {
	xs, ys, ok := strings.Cut(s, ",")
	x, err1 := strconv.Atoi(strings.TrimSpace(xs))
	y, err2 := strconv.Atoi(strings.TrimSpace(ys))
	if !ok || err1 != nil || err2 != nil {
		return 0, 0, fmt.Errorf("at %q: want x,y", s)
	}
	return x, y, nil
}

func autoPos(c *cv.Canvas) (int, int) {
	boxes := c.Boxes()
	if len(boxes) == 0 {
		return 2, 1
	}
	x, y := boxes[0].X, 0
	for _, b := range boxes {
		x, y = min(x, b.X), max(y, b.Y+b.Height)
	}
	return x, y + 3
}

var borders = []string{"ascii", "single", "double", "rounded"}

func style(c *cv.Canvas, id int, p map[string]string) error {
	if v, ok := p["border"]; ok {
		i := slices.Index(borders, strings.ToLower(v))
		if i < 0 {
			return fmt.Errorf("border %q: want %s", v, strings.Join(borders, "|"))
		}
		c.SetBorderStyle(id, cv.BorderStyle(i))
	}
	if v, ok := p["shadow"]; ok {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 || n > 3 {
			return fmt.Errorf("shadow %q: want 0-3", v)
		}
		c.Boxes()[id].ZLevel = n
	}
	if v, ok := p["size"]; ok {
		ws, hs, _ := strings.Cut(strings.ToLower(v), "x")
		w, err1 := strconv.Atoi(ws)
		h, err2 := strconv.Atoi(hs)
		if err1 != nil || err2 != nil {
			return fmt.Errorf("size %q: want WxH", v)
		}
		b := c.Boxes()[id]
		c.SetBoxSize(id, max(w, b.Width), max(h, b.Height))
	}
	if v, ok := p["color"]; ok {
		i := slices.IndexFunc(cv.ColorNames, func(n string) bool { return strings.EqualFold(n, v) })
		if i < 0 && v != "" && !strings.EqualFold(v, "none") {
			return fmt.Errorf("color %q: want none or one of %s", v, strings.Join(cv.ColorNames, ", "))
		}
		c.SetBoxColor(id, i)
	}
	return nil
}

func fields(c *cv.Canvas, id int, p map[string]string) {
	if v, ok := p["title"]; ok {
		c.SetBoxTitle(id, text(v))
	}
	if v, ok := p["tooltip"]; ok {
		c.SetBoxTooltip(id, text(v))
	}
}

func addBox(c *cv.Canvas, p map[string]string) (string, error) {
	x, y := autoPos(c)
	if at, ok := p["at"]; ok {
		var err error
		if x, y, err = parseAt(at); err != nil {
			return "", err
		}
	}
	c.AddBox(x, y, text(p["text"]))
	id := len(c.Boxes()) - 1
	fields(c, id, p)
	if err := style(c, id, p); err != nil {
		return "", err
	}
	return fmt.Sprintf("box %d", id), nil
}

func setBox(c *cv.Canvas, p map[string]string) (string, error) {
	id, err := idArg(p, "id", len(c.Boxes()), "box")
	if err != nil {
		return "", err
	}
	if v, ok := p["text"]; ok {
		c.SetBoxText(id, text(v))
	}
	fields(c, id, p)
	if err := style(c, id, p); err != nil {
		return "", err
	}
	if at, ok := p["at"]; ok {
		x, y, err := parseAt(at)
		if err != nil {
			return "", err
		}
		b := c.Boxes()[id]
		c.MoveBox(id, x-b.X, y-b.Y)
	}
	return fmt.Sprintf("box %d", id), nil
}

func rmBox(c *cv.Canvas, p map[string]string) (string, error) {
	id, err := idArg(p, "id", len(c.Boxes()), "box")
	if err != nil {
		return "", err
	}
	c.DeleteBox(id)
	return fmt.Sprintf("deleted box %d", id), nil
}

func connect(c *cv.Canvas, p map[string]string) (string, error) {
	from, err := idArg(p, "from", len(c.Boxes()), "box")
	if err != nil {
		return "", err
	}
	to, err := idArg(p, "to", len(c.Boxes()), "box")
	if err != nil {
		return "", err
	}
	arrow := strings.ToLower(p["arrow"])
	if !slices.Contains([]string{"", "to", "from", "both", "none"}, arrow) {
		return "", fmt.Errorf("arrow %q: want to|from|both|none", arrow)
	}
	boxes := c.Boxes()
	horizontal := horizAxis(boxes[from], boxes[to])
	c.AddConnectionAxis(from, to, horizontal)
	conns := c.Connections()
	cn := &conns[len(conns)-1]
	cn.ArrowTo = arrow == "" || arrow == "to" || arrow == "both"
	cn.ArrowFrom = arrow == "from" || arrow == "both"
	zRoute(cn, horizontal)
	return fmt.Sprintf("connection %d", len(conns)-1), nil
}

func rmConn(c *cv.Canvas, p map[string]string) (string, error) {
	id, err := idArg(p, "id", len(c.Connections()), "connection")
	if err != nil {
		return "", err
	}
	c.RemoveSpecificConnection(c.Connections()[id])
	return fmt.Sprintf("deleted connection %d", id), nil
}

func load(name string) (*cv.Canvas, string, error) {
	path := FindChart(name)
	if path == "" {
		return nil, "", fmt.Errorf("chart not found: %s", name)
	}
	c := cv.NewCanvas()
	if _, _, err := c.LoadFromFileWithPan(path); err != nil {
		return nil, "", fmt.Errorf("reading %s: %w", path, err)
	}
	return c, path, nil
}

func describe(p map[string]string) (string, error) {
	c, path, err := load(p["chart"])
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	switch p["format"] {
	case "", "text":
		c.Describe(&buf, path)
	case "json":
		err = c.DescribeJSON(&buf, path)
	default:
		err = fmt.Errorf("format %q: want text|json", p["format"])
	}
	return buf.String(), err
}

func validate(p map[string]string) (string, error) {
	path := FindChart(p["chart"])
	if path == "" {
		return "", fmt.Errorf("chart not found: %s", p["chart"])
	}
	msgs, err := cv.Validate(path)
	if err != nil {
		return "", err
	}
	hard := 0
	for _, m := range msgs {
		if !strings.HasPrefix(m, "warning:") {
			hard++
		}
	}
	if len(msgs) == 0 {
		return "ok", nil
	}
	out := strings.Join(msgs, "\n")
	if hard > 0 {
		return out, fmt.Errorf("%d errors", hard)
	}
	return out, nil
}

func export(p map[string]string) (string, error) {
	c, path, err := load(p["chart"])
	if err != nil {
		return "", err
	}
	format, out := p["format"], p["out"]
	writers := map[string]func(io.Writer) error{"mermaid": c.WriteMermaid, "txt": c.WriteText}
	if format == "png" {
		if out == "" {
			out = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)) + ".png"
		}
		if err := c.ExportToPNG(out); err != nil {
			return "", err
		}
		return "wrote " + out, nil
	}
	write, ok := writers[format]
	if !ok {
		return "", fmt.Errorf("format %q: want mermaid|txt|png", format)
	}
	if out == "" || out == "-" {
		var buf bytes.Buffer
		err := write(&buf)
		return buf.String(), err
	}
	file, err := os.Create(out)
	if err != nil {
		return "", err
	}
	err = write(file)
	if cerr := file.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(out)
		return "", err
	}
	return "wrote " + out, nil
}

func horizAxis(a, b cv.Box) bool {
	switch {
	case a.Y+a.Height <= b.Y || b.Y+b.Height <= a.Y:
		return false
	case a.X+a.Width <= b.X || b.X+b.Width <= a.X:
		return true
	}
	return abs(a.X+a.Width/2-b.X-b.Width/2) > abs(a.Y+a.Height/2-b.Y-b.Height/2)
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func zRoute(cn *cv.Connection, horizontal bool) {
	cn.Waypoints = nil
	switch {
	case !horizontal && cn.FromX != cn.ToX:
		mid := (cn.FromY + cn.ToY) / 2
		cn.Waypoints = []cv.Point{{X: cn.FromX, Y: mid}, {X: cn.ToX, Y: mid}}
	case horizontal && cn.FromY != cn.ToY:
		mid := (cn.FromX + cn.ToX) / 2
		cn.Waypoints = []cv.Point{{X: mid, Y: cn.FromY}, {X: mid, Y: cn.ToY}}
	}
}

func addText(c *cv.Canvas, p map[string]string) (string, error) {
	if p["text"] == "" {
		return "", fmt.Errorf("text is required")
	}
	x, y := autoPos(c)
	if at, ok := p["at"]; ok {
		var err error
		if x, y, err = parseAt(at); err != nil {
			return "", err
		}
	}
	c.AddText(x, y, text(p["text"]))
	id := len(c.Texts()) - 1
	if err := textColor(c, id, p); err != nil {
		return "", err
	}
	return fmt.Sprintf("text %d", id), nil
}

func textColor(c *cv.Canvas, id int, p map[string]string) error {
	v, ok := p["color"]
	if !ok {
		return nil
	}
	i := slices.IndexFunc(cv.ColorNames, func(n string) bool { return strings.EqualFold(n, v) })
	if i < 0 && v != "" && !strings.EqualFold(v, "none") {
		return fmt.Errorf("color %q: want none or one of %s", v, strings.Join(cv.ColorNames, ", "))
	}
	c.SetTextColor(id, i)
	return nil
}

func setText(c *cv.Canvas, p map[string]string) (string, error) {
	id, err := idArg(p, "id", len(c.Texts()), "text")
	if err != nil {
		return "", err
	}
	if v, ok := p["text"]; ok {
		c.SetTextText(id, text(v))
	}
	if err := textColor(c, id, p); err != nil {
		return "", err
	}
	if at, ok := p["at"]; ok {
		x, y, err := parseAt(at)
		if err != nil {
			return "", err
		}
		t := c.Texts()[id]
		c.MoveText(id, x-t.X, y-t.Y)
	}
	return fmt.Sprintf("text %d", id), nil
}

func rmText(c *cv.Canvas, p map[string]string) (string, error) {
	id, err := idArg(p, "id", len(c.Texts()), "text")
	if err != nil {
		return "", err
	}
	c.DeleteText(id)
	return fmt.Sprintf("deleted text %d", id), nil
}

var editors = map[string]func(*cv.Canvas, map[string]string) (string, error){
	"add-box": addBox, "set-box": setBox, "rm-box": rmBox,
	"connect": connect, "rm-conn": rmConn, "add-text": addText,
	"set-text": setText, "rm-text": rmText,
}

func toStr(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	b, _ := json.Marshal(v)
	return string(b)
}

func runBatch(p map[string]string) (string, error) {
	src, err := readSource(p["ops"])
	if err != nil {
		return "", err
	}
	var steps []map[string]any
	dec := json.NewDecoder(strings.NewReader(src))
	dec.UseNumber()
	if err := dec.Decode(&steps); err != nil {
		return "", fmt.Errorf("ops: want a JSON array of objects: %w", err)
	}
	var lines []string
	_, err = edit(p, func(c *cv.Canvas, _ map[string]string) (string, error) {
		for i, step := range steps {
			name := strings.ReplaceAll(toStr(step["op"]), "_", "-")
			run, ok := editors[name]
			if !ok {
				return "", fmt.Errorf("op %d: unknown op %q", i, name)
			}
			q := map[string]string{}
			for k, v := range step {
				if k != "op" {
					q[k] = toStr(v)
				}
			}
			out, err := run(c, q)
			if err != nil {
				return "", fmt.Errorf("op %d (%s): %w", i, name, err)
			}
			lines = append(lines, fmt.Sprintf("%d %s: %s", i, name, out))
		}
		return "", nil
	})
	if err != nil {
		return "", err
	}
	return strings.Join(lines, "\n"), nil
}

func find(p map[string]string) (string, error) {
	c, _, err := load(p["chart"])
	if err != nil {
		return "", err
	}
	q := strings.ToLower(p["query"])
	var lines []string
	for _, b := range c.Boxes() {
		if !strings.Contains(strings.ToLower(b.Title+"\n"+b.GetText()+"\n"+b.Tooltip), q) {
			continue
		}
		label := strings.Join(strings.Fields(b.Title+" "+b.GetText()), " ")
		lines = append(lines, fmt.Sprintf("box %d at %d,%d: %s", b.ID, b.X, b.Y, label))
	}
	if len(lines) == 0 {
		return "no matches", nil
	}
	return strings.Join(lines, "\n"), nil
}
