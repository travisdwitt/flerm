package agent

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"slices"
	"strings"

	cv "flerm/internal/canvas"
)

type node struct {
	id, text, title string
	round           bool
}

type edge struct {
	from, to  int
	arrowFrom bool
	arrowTo   bool
	label     string
}

type graph struct {
	nodes []node
	index map[string]int
	edges []edge
	lr    bool
	notes []string
}

func (g *graph) node(id string) int {
	if i, ok := g.index[id]; ok {
		return i
	}
	g.index[id] = len(g.nodes)
	g.nodes = append(g.nodes, node{id: id, text: id})
	return len(g.nodes) - 1
}

var (
	linkRe  = regexp.MustCompile(`^\s*(<)?(--+|==+|-\.+-?)(>)?`)
	labelRe = regexp.MustCompile(`^\s*--\s+(.+?)\s+(--+|==+)(>?)`)
	pipeRe  = regexp.MustCompile(`^\s*\|([^|]*)\|`)
	idRe    = regexp.MustCompile(`^\s*([A-Za-z0-9_]+)`)
	skipRe  = regexp.MustCompile(`^\s*(subgraph|end|style|classDef|class|linkStyle|click|direction)\b`)
	headRe  = regexp.MustCompile(`^\s*(flowchart|graph)\b\s*(TD|TB|BT|LR|RL)?`)
)

var shapes = [][2]string{{"[[", "]]"}, {"[(", ")]"}, {"([", "])"}, {"((", "))"}, {"[", "]"}, {"(", ")"}, {"{", "}"}}

func (g *graph) scanNode(s string) (int, string, bool) {
	m := idRe.FindStringSubmatchIndex(s)
	if m == nil {
		return 0, s, false
	}
	id := s[m[2]:m[3]]
	rest := s[m[1]:]
	i := g.node(id)
	trimmed := strings.TrimLeft(rest, " ")
	for _, sh := range shapes {
		body, ok := strings.CutPrefix(trimmed, sh[0])
		if !ok {
			continue
		}
		var label string
		if strings.HasPrefix(body, `"`) {
			end := strings.Index(body[1:], `"`)
			if end < 0 {
				return 0, s, false
			}
			label, body = body[1:end+1], body[end+2:]
			cut := strings.Index(body, sh[1])
			if cut < 0 {
				return 0, s, false
			}
			rest = body[cut+len(sh[1]):]
		} else {
			cut := strings.Index(body, sh[1])
			if cut < 0 {
				return 0, s, false
			}
			label, rest = body[:cut], body[cut+len(sh[1]):]
		}
		label = strings.NewReplacer("#quot;", `"`, "<br/>", "\n", "<br>", "\n", `\n`, "\n").Replace(label)
		g.nodes[i].text = strings.TrimSpace(label)
		g.nodes[i].round = strings.HasPrefix(sh[0], "(")
		break
	}
	return i, rest, true
}

func (g *graph) statement(s string) error {
	from, rest, ok := g.scanNode(s)
	if !ok {
		return fmt.Errorf("cannot parse %q", s)
	}
	for strings.TrimSpace(rest) != "" {
		var e edge
		if m := labelRe.FindStringSubmatch(rest); m != nil {
			e.arrowTo = m[3] == ">"
			e.label = strings.TrimSpace(m[1])
			rest = rest[len(m[0]):]
		} else if m := linkRe.FindStringSubmatch(rest); m != nil {
			e.arrowFrom, e.arrowTo = m[1] == "<", m[3] == ">"
			rest = rest[len(m[0]):]
			if pm := pipeRe.FindStringSubmatch(rest); pm != nil {
				e.label = strings.Trim(strings.TrimSpace(pm[1]), `"`)
				rest = rest[len(pm[0]):]
			}
		} else {
			return fmt.Errorf("cannot parse %q", strings.TrimSpace(rest))
		}
		to, next, ok := g.scanNode(rest)
		if !ok {
			return fmt.Errorf("missing node after link in %q", s)
		}
		e.from, e.to = from, to
		g.edges = append(g.edges, e)
		from, rest = to, next
	}
	return nil
}

func parseMermaid(src string) (*graph, error) {
	g := &graph{index: map[string]int{}}
	for _, line := range strings.Split(src, "\n") {
		if i := strings.Index(line, "%%"); i >= 0 {
			line = line[:i]
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		if skipRe.MatchString(line) {
			g.notes = append(g.notes, fmt.Sprintf("line ignored: %s", strings.TrimSpace(line)))
			continue
		}
		if m := headRe.FindStringSubmatch(line); m != nil {
			g.lr = m[2] == "LR" || m[2] == "RL"
			continue
		}
		for _, st := range strings.Split(line, ";") {
			if strings.TrimSpace(st) == "" {
				continue
			}
			if err := g.statement(st); err != nil {
				return nil, err
			}
		}
	}
	return g, nil
}

type spec struct {
	Direction string `json:"direction"`
	Boxes     []struct {
		ID, Text, Title, Border string
	} `json:"boxes"`
	Edges []struct {
		From, To, Arrow, Label string
	} `json:"edges"`
}

func parseSpec(src string) (*graph, error) {
	var s spec
	if err := json.Unmarshal([]byte(src), &s); err != nil {
		return nil, fmt.Errorf("json spec: %w", err)
	}
	g := &graph{index: map[string]int{}, lr: strings.EqualFold(s.Direction, "LR")}
	for _, b := range s.Boxes {
		if b.ID == "" {
			return nil, fmt.Errorf("json spec: every box needs an id")
		}
		i := g.node(b.ID)
		g.nodes[i].text, g.nodes[i].title = text(b.Text), text(b.Title)
		g.nodes[i].round = strings.EqualFold(b.Border, "rounded")
	}
	for _, e := range s.Edges {
		for _, id := range []string{e.From, e.To} {
			if _, ok := g.index[id]; !ok {
				return nil, fmt.Errorf("json spec: edge references unknown box %q", id)
			}
		}
		arrow := strings.ToLower(e.Arrow)
		g.edges = append(g.edges, edge{
			from: g.index[e.From], to: g.index[e.To],
			arrowTo:   arrow == "" || arrow == "to" || arrow == "both",
			arrowFrom: arrow == "from" || arrow == "both",
			label:     text(e.Label),
		})
	}
	return g, nil
}

func (g *graph) layers() []int {
	layer := make([]int, len(g.nodes))
	out := make([][]int, len(g.nodes))
	for _, e := range g.edges {
		out[e.from] = append(out[e.from], e.to)
	}
	state := make([]int, len(g.nodes))
	var fwd [][2]int
	var walk func(u int)
	walk = func(u int) {
		state[u] = 1
		for _, v := range out[u] {
			switch state[v] {
			case 0:
				fwd = append(fwd, [2]int{u, v})
				walk(v)
			case 2:
				fwd = append(fwd, [2]int{u, v})
			}
		}
		state[u] = 2
	}
	for u := range g.nodes {
		if state[u] == 0 {
			walk(u)
		}
	}
	for range g.nodes {
		for _, e := range fwd {
			layer[e[1]] = max(layer[e[1]], layer[e[0]]+1)
		}
	}
	return layer
}

func (g *graph) place(c *cv.Canvas, base, startX, startY int) {
	layer := g.layers()
	rows := map[int][]int{}
	top := 0
	for i, l := range layer {
		rows[l] = append(rows[l], base+i)
		top = max(top, l)
	}
	boxes := c.Boxes()
	size := func(i int) (along, across int) {
		if g.lr {
			return boxes[i].Width, boxes[i].Height
		}
		return boxes[i].Height, boxes[i].Width
	}
	gapAlong, gapAcross := 4, 4
	if g.lr {
		gapAlong, gapAcross = 8, 2
		for _, e := range g.edges {
			gapAlong = max(gapAlong, len(e.label)+6)
		}
	}
	span := func(row []int) int {
		total := gapAcross * (len(row) - 1)
		for _, i := range row {
			_, a := size(i)
			total += a
		}
		return total
	}
	widest := 0
	for _, row := range rows {
		widest = max(widest, span(row))
	}
	pos, across0 := startY, startX
	if g.lr {
		pos, across0 = startX, startY
	}
	for l := 0; l <= top; l++ {
		cross, depth := across0+(widest-span(rows[l]))/2, 0
		for _, i := range rows[l] {
			al, ac := size(i)
			depth = max(depth, al)
			if g.lr {
				c.SetBoxPositionOnly(i, pos, cross)
			} else {
				c.SetBoxPositionOnly(i, cross, pos)
			}
			cross += ac + gapAcross
		}
		pos += depth + gapAlong
	}
}

type end struct {
	conn  int
	to    bool
	other int
}

func faceOf(b cv.Box, x, y int) string {
	switch {
	case y == b.Y:
		return "top"
	case y == b.Y+b.Height-1:
		return "bottom"
	case x == b.X:
		return "left"
	}
	return "right"
}

func spread(c *cv.Canvas, first int) {
	boxes := c.Boxes()
	conns := c.Connections()
	groups := map[string][]end{}
	for i := first; i < len(conns); i++ {
		cn := conns[i]
		for _, to := range []bool{false, true} {
			id, x, y, ox, oy := cn.FromID, cn.FromX, cn.FromY, cn.ToX, cn.ToY
			if to {
				id, x, y, ox, oy = cn.ToID, cn.ToX, cn.ToY, cn.FromX, cn.FromY
			}
			face := faceOf(boxes[id], x, y)
			other := ox
			if face == "left" || face == "right" {
				other = oy
			}
			key := fmt.Sprintf("%d/%s", id, face)
			groups[key] = append(groups[key], end{i, to, other})
		}
	}
	for key, ends := range groups {
		if len(ends) < 2 {
			continue
		}
		var id int
		var face string
		fmt.Sscanf(key, "%d/%s", &id, &face)
		b := boxes[id]
		slices.SortStableFunc(ends, func(x, y end) int { return x.other - y.other })
		side := face == "left" || face == "right"
		for i, e := range ends {
			cn := &conns[e.conn]
			v := b.X + 1 + (2*i+1)*(b.Width-2)/(2*len(ends))
			if side {
				v = b.Y + 1 + (2*i+1)*(b.Height-2)/(2*len(ends))
			}
			switch {
			case side && e.to:
				cn.ToY = v
			case side:
				cn.FromY = v
			case e.to:
				cn.ToX = v
			default:
				cn.FromX = v
			}
		}
	}
}

func labelPos(cn cv.Connection, horizontal bool) (int, int) {
	switch {
	case !horizontal && cn.FromX != cn.ToX:
		return min(cn.FromX, cn.ToX) + 2, (cn.FromY+cn.ToY)/2 - 1
	case !horizontal:
		return cn.FromX + 2, (cn.FromY + cn.ToY) / 2
	case cn.FromY != cn.ToY:
		return min(cn.FromX, cn.ToX) + 2, min(cn.FromY, cn.ToY) - 1
	}
	return min(cn.FromX, cn.ToX) + 2, cn.FromY - 1
}

func readSource(s string) (string, error) {
	if s == "-" {
		b, err := io.ReadAll(os.Stdin)
		return string(b), err
	}
	if b, err := os.ReadFile(s); err == nil {
		return string(b), nil
	}
	return s, nil
}

func runImport(p map[string]string) (string, error) {
	src, err := readSource(p["source"])
	if err != nil {
		return "", err
	}
	path := chartPath(p["chart"])
	_, statErr := os.Stat(path)
	appendTo := p["append"] == "true"
	if statErr == nil && !appendTo && p["force"] != "true" {
		return "", fmt.Errorf("%s exists; pass --force true to overwrite, or --append true to add below it", path)
	}
	var g *graph
	if strings.HasPrefix(strings.TrimSpace(src), "{") {
		g, err = parseSpec(src)
	} else {
		g, err = parseMermaid(src)
	}
	if err != nil {
		return "", err
	}
	if len(g.nodes) == 0 {
		return "", fmt.Errorf("no nodes found in source")
	}
	c := cv.NewCanvas()
	panX, panY := 0, 0
	startX, startY := 2, 1
	if appendTo && statErr == nil {
		if panX, panY, err = c.LoadFromFileWithPan(path); err != nil {
			return "", fmt.Errorf("reading %s: %w", path, err)
		}
		startX, startY = autoPos(c)
		startY++
	}
	base, firstConn := len(c.Boxes()), len(c.Connections())
	for _, n := range g.nodes {
		c.AddBox(0, 0, n.text)
		id := len(c.Boxes()) - 1
		c.SetBoxTitle(id, n.title)
		if n.round {
			c.SetBorderStyle(id, cv.BorderStyleRounded)
		}
	}
	g.place(c, base, startX, startY)
	var kept []edge
	for _, e := range g.edges {
		if e.from == e.to {
			g.notes = append(g.notes, fmt.Sprintf("self-loop on %s dropped", g.nodes[e.from].id))
			continue
		}
		c.AddConnectionAxis(base+e.from, base+e.to, g.lr)
		conns := c.Connections()
		conns[len(conns)-1].ArrowFrom, conns[len(conns)-1].ArrowTo = e.arrowFrom, e.arrowTo
		kept = append(kept, e)
	}
	spread(c, firstConn)
	conns := c.Connections()
	for i, e := range kept {
		cn := &conns[firstConn+i]
		zRoute(cn, g.lr)
		if e.label != "" {
			lx, ly := labelPos(*cn, g.lr)
			c.AddText(lx, ly, e.label)
		}
	}
	if err := c.SaveToFileWithPan(path, panX, panY); err != nil {
		return "", err
	}
	ids := make([]string, len(g.nodes))
	for i, n := range g.nodes {
		ids[i] = fmt.Sprintf("%s=%d", n.id, base+i)
	}
	out := fmt.Sprintf("wrote %s: %d boxes, %d connections (box ids: %s)", path, len(g.nodes), len(kept), strings.Join(ids, " "))
	for _, n := range g.notes {
		out += "\nwarning: " + n
	}
	return out, nil
}
