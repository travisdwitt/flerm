package canvas

import (
	"encoding/json"
	"io"
)

type jsonBox struct {
	ID      int    `json:"id"`
	X       int    `json:"x"`
	Y       int    `json:"y"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
	Title   string `json:"title,omitempty"`
	Text    string `json:"text,omitempty"`
	Tooltip string `json:"tooltip,omitempty"`
	Border  string `json:"border"`
	Color   string `json:"color,omitempty"`
	Shadow  int    `json:"shadow,omitempty"`
}

type jsonConn struct {
	ID        int     `json:"id"`
	From      *int    `json:"from"`
	To        *int    `json:"to"`
	FromPos   Point   `json:"from_pos"`
	ToPos     Point   `json:"to_pos"`
	Waypoints []Point `json:"waypoints,omitempty"`
	ArrowFrom bool    `json:"arrow_from"`
	ArrowTo   bool    `json:"arrow_to"`
	Color     string  `json:"color,omitempty"`
}

type jsonText struct {
	ID    int    `json:"id"`
	X     int    `json:"x"`
	Y     int    `json:"y"`
	Text  string `json:"text"`
	Color string `json:"color,omitempty"`
}

type jsonChart struct {
	Name        string         `json:"name"`
	Boxes       []jsonBox      `json:"boxes"`
	Connections []jsonConn     `json:"connections"`
	Texts       []jsonText     `json:"texts"`
	Highlights  map[string]int `json:"highlighted_cells"`
}

func colorName(color int) string {
	if color < 0 || color >= len(ColorNames) {
		return ""
	}
	return ColorNames[color]
}

func endID(id int) *int {
	if id < 0 {
		return nil
	}
	return &id
}

func (c *Canvas) DescribeJSON(w io.Writer, name string) error {
	out := jsonChart{
		Name:        name,
		Boxes:       []jsonBox{},
		Connections: []jsonConn{},
		Texts:       []jsonText{},
		Highlights:  map[string]int{},
	}
	for _, b := range c.boxes {
		out.Boxes = append(out.Boxes, jsonBox{
			ID: b.ID, X: b.X, Y: b.Y, Width: b.Width, Height: b.Height,
			Title: b.Title, Text: b.GetText(), Tooltip: b.Tooltip,
			Border: b.BorderStyle.String(), Color: colorName(b.Color), Shadow: b.ZLevel,
		})
	}
	for i, cn := range c.connections {
		out.Connections = append(out.Connections, jsonConn{
			ID: i, From: endID(cn.FromID), To: endID(cn.ToID),
			FromPos: Point{cn.FromX, cn.FromY}, ToPos: Point{cn.ToX, cn.ToY},
			Waypoints: cn.Waypoints, ArrowFrom: cn.ArrowFrom, ArrowTo: cn.ArrowTo,
			Color: colorName(cn.Color),
		})
	}
	for i, t := range c.texts {
		out.Texts = append(out.Texts, jsonText{ID: i, X: t.X, Y: t.Y, Text: t.GetText(), Color: colorName(t.Color)})
	}
	for _, color := range c.highlights {
		out.Highlights[ColorNames[color]]++
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
