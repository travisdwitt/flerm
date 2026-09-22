package canvas

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func renderColorMap(c *Canvas, w, h int) [][]int {
	rr := c.RenderRaw(w, h, -1, CoordUnset, CoordUnset, nil, CoordUnset, CoordUnset, 0, 0, -1, -1, false, -1, -1, 0, "", CoordUnset, CoordUnset, CoordUnset, CoordUnset, CoordUnset, CoordUnset, false, -1, -1)
	return rr.ColorMap
}

func TestObjectColorRenders(t *testing.T) {
	c := NewCanvas()
	c.AddBox(2, 2, "Hi") // box 0
	c.boxes[0].Width, c.boxes[0].Height = 8, 3
	c.SetBoxColor(0, 2) // green

	cm := renderColorMap(c, 40, 20)
	if cm[2][2] != 2 {
		t.Fatalf("expected box border color 2 at (2,2), got %d", cm[2][2])
	}
	if cm[15][30] != -1 {
		t.Fatalf("expected uncolored cell -1, got %d", cm[15][30])
	}
}

func TestColorFollowsBoxOnMove(t *testing.T) {
	c := NewCanvas()
	c.AddBox(2, 2, "Hi")
	c.boxes[0].Width, c.boxes[0].Height = 8, 3
	c.SetBoxColor(0, 1) // red

	c.MoveBox(0, 15, 0) // slide right

	cm := renderColorMap(c, 60, 20)
	if cm[2][2] != -1 {
		t.Fatalf("expected old corner uncolored after move, got %d", cm[2][2])
	}
	if cm[2][17] != 1 {
		t.Fatalf("expected color 1 at new corner (17,2), got %d", cm[2][17])
	}
}

func TestColorSaveLoadRoundTrip(t *testing.T) {
	c := NewCanvas()
	c.AddBox(1, 1, "A")  // box 0
	c.AddBox(20, 1, "B") // box 1
	c.AddText(10, 10, "note")
	c.AddConnectionWithWaypoints(0, 1, 5, 2, 20, 2, nil) // conn 0
	c.SetBoxColor(0, 3)
	c.SetTextColor(0, 5)
	c.SetLineColor(0, 6)

	path := filepath.Join(t.TempDir(), "colors.sav")
	if err := c.SaveToFileWithPan(path, 0, 0); err != nil {
		t.Fatal(err)
	}
	loaded := NewCanvas()
	if _, _, err := loaded.LoadFromFileWithPan(path); err != nil {
		t.Fatal(err)
	}
	if loaded.boxes[0].Color != 3 {
		t.Fatalf("box color not preserved: got %d", loaded.boxes[0].Color)
	}
	if loaded.boxes[1].Color != -1 {
		t.Fatalf("uncolored box should stay -1: got %d", loaded.boxes[1].Color)
	}
	if loaded.texts[0].Color != 5 {
		t.Fatalf("text color not preserved: got %d", loaded.texts[0].Color)
	}
	if loaded.connections[0].Color != 6 {
		t.Fatalf("line color not preserved: got %d", loaded.connections[0].Color)
	}
}

func TestLoadOldFileWithoutColorSections(t *testing.T) {
	old := "FLOWCHART\n" +
		"BOXES:1\n" +
		"5,5,8,3,0,1,, Box \n" +
		"CONNECTIONS:0\n" +
		"TEXTS:1\n" +
		"3,3,hello\n" +
		"HIGHLIGHTS:0\n"
	path := filepath.Join(t.TempDir(), "old.sav")
	if err := os.WriteFile(path, []byte(old), 0644); err != nil {
		t.Fatal(err)
	}
	c := NewCanvas()
	if _, _, err := c.LoadFromFileWithPan(path); err != nil {
		t.Fatalf("old file should load: %v", err)
	}
	if len(c.boxes) != 1 || c.boxes[0].Color != -1 {
		t.Fatalf("expected 1 box with color -1, got %d boxes, color %d", len(c.boxes), c.boxes[0].Color)
	}
	if len(c.texts) != 1 || c.texts[0].Color != -1 {
		t.Fatalf("expected 1 text with color -1")
	}
}

func TestNegativeCoordinatesSurviveMoveAndRoundTrip(t *testing.T) {
	c := NewCanvas()
	c.AddBox(3, 2, "Alpha")
	c.AddText(4, 6, "note")
	c.SetHighlight(3, 2, 4)

	c.MoveBox(0, -10, -5)
	if got := c.Boxes()[0]; got.X != -7 || got.Y != -3 {
		t.Fatalf("expected the box at (-7,-3), got (%d,%d)", got.X, got.Y)
	}
	c.MoveText(0, -9, -8)
	if got := c.Texts()[0]; got.X != -5 || got.Y != -2 {
		t.Fatalf("expected the text at (-5,-2), got (%d,%d)", got.X, got.Y)
	}
	c.SetHighlight(-7, -3, 4)
	if c.GetHighlight(-7, -3) != 4 {
		t.Fatal("expected a highlight to be settable at a negative cell")
	}

	minX, minY, maxX, maxY := c.GetFullBounds()
	if minX != -7 || minY != -3 {
		t.Fatalf("bounds should reach into negative space, got (%d,%d)-(%d,%d)", minX, minY, maxX, maxY)
	}

	path := t.TempDir() + "/neg.txt"
	if err := c.SaveToFileWithPan(path, -12, -9); err != nil {
		t.Fatalf("save: %v", err)
	}
	loaded := NewCanvas()
	panX, panY, err := loaded.LoadFromFileWithPan(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if panX != -12 || panY != -9 {
		t.Fatalf("expected the negative pan restored, got (%d,%d)", panX, panY)
	}
	if got := loaded.Boxes()[0]; got.X != -7 || got.Y != -3 {
		t.Fatalf("box did not round-trip, got (%d,%d)", got.X, got.Y)
	}
	if got := loaded.Texts()[0]; got.X != -5 || got.Y != -2 {
		t.Fatalf("text did not round-trip, got (%d,%d)", got.X, got.Y)
	}
	if loaded.GetHighlight(-7, -3) != 4 {
		t.Fatal("highlight at a negative cell did not round-trip")
	}
}

func TestRenderShowsNegativeCoordinatesWhenPanned(t *testing.T) {
	c := NewCanvas()
	c.AddText(-6, -2, "left")

	if rr := c.RenderRaw(40, 10, -1, CoordUnset, CoordUnset, nil, CoordUnset, CoordUnset, 0, 0, -1, -1, false, -1, -1, 0, "", CoordUnset, CoordUnset, CoordUnset, CoordUnset, CoordUnset, CoordUnset, false, -1, -1); strings.Contains(joinCanvas(rr), "left") {
		t.Fatal("text at a negative position should be off-screen at pan 0")
	}
	rr := c.RenderRaw(40, 10, -1, CoordUnset, CoordUnset, nil, CoordUnset, CoordUnset, -8, -4, -1, -1, false, -1, -1, 0, "", CoordUnset, CoordUnset, CoordUnset, CoordUnset, CoordUnset, CoordUnset, false, -1, -1)
	if !strings.Contains(joinCanvas(rr), "left") {
		t.Fatal("panning to negative space should reveal the text")
	}
}

func joinCanvas(rr *RenderResult) string {
	var b strings.Builder
	for _, row := range rr.Canvas {
		b.WriteString(string(row))
		b.WriteByte('\n')
	}
	return b.String()
}
