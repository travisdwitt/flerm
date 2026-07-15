package main

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func renderColorMap(c *Canvas, w, h int) [][]int {
	rr := c.RenderRaw(w, h, -1, -1, -1, nil, -1, -1, 0, 0, -1, -1, false, -1, -1, 0, "", -1, -1, -1, -1, -1, -1, false, -1, -1)
	return rr.ColorMap
}

// TestObjectColorRenders verifies a box/text/line color lands on the object's
// own cells in the color map.
func TestObjectColorRenders(t *testing.T) {
	c := NewCanvas()
	c.AddBox(2, 2, "Hi") // box 0
	c.boxes[0].Width, c.boxes[0].Height = 8, 3
	c.SetBoxColor(0, 2) // green

	cm := renderColorMap(c, 40, 20)
	// Top-left border corner of the box.
	if cm[2][2] != 2 {
		t.Fatalf("expected box border color 2 at (2,2), got %d", cm[2][2])
	}
	// A cell outside the box stays uncolored.
	if cm[15][30] != -1 {
		t.Fatalf("expected uncolored cell -1, got %d", cm[15][30])
	}
}

// TestColorFollowsBoxOnMove is the core fix: object color derives from live
// geometry, so moving the box moves the color with it (no highlight desync).
func TestColorFollowsBoxOnMove(t *testing.T) {
	c := NewCanvas()
	c.AddBox(2, 2, "Hi")
	c.boxes[0].Width, c.boxes[0].Height = 8, 3
	c.SetBoxColor(0, 1) // red

	c.MoveBox(0, 15, 0) // slide right

	cm := renderColorMap(c, 60, 20)
	// Old corner is now empty/uncolored.
	if cm[2][2] != -1 {
		t.Fatalf("expected old corner uncolored after move, got %d", cm[2][2])
	}
	// New corner (x=17) carries the color.
	if cm[2][17] != 1 {
		t.Fatalf("expected color 1 at new corner (17,2), got %d", cm[2][17])
	}
}

// TestColorSaveLoadRoundTrip verifies colors survive a save/load cycle.
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
	if err := c.SaveToFile(path); err != nil {
		t.Fatal(err)
	}
	loaded := NewCanvas()
	if err := loaded.LoadFromFile(path); err != nil {
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

// TestLoadOldFileWithoutColorSections verifies a pre-color .sav (no color
// sections) loads with all colors defaulting to -1.
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
	if err := c.LoadFromFile(path); err != nil {
		t.Fatalf("old file should load: %v", err)
	}
	if len(c.boxes) != 1 || c.boxes[0].Color != -1 {
		t.Fatalf("expected 1 box with color -1, got %d boxes, color %d", len(c.boxes), c.boxes[0].Color)
	}
	if len(c.texts) != 1 || c.texts[0].Color != -1 {
		t.Fatalf("expected 1 text with color -1")
	}
}

// TestMenuCascadeSetsBoxColor drives the cascading menu (Border -> Color -> Green)
// and verifies the box color is set and undoable.
func TestMenuCascadeSetsBoxColor(t *testing.T) {
	m := newTestModel()
	out, _ := m.Update(press(tea.MouseButtonRight, 6, 4)) // right-click box 0
	m = out.(model)

	labelIndex := func(items []MenuItem, label string) int {
		for i, it := range items {
			if it.Label == label {
				return i
			}
		}
		return -1
	}

	// Focus "Border" and descend.
	m.menuIndex = labelIndex(m.menuItems, "Border")
	if m.menuIndex < 0 {
		t.Fatal("no Border item in box menu")
	}
	m.menuDescend()
	// Focus "Color" and descend.
	ci := labelIndex(m.focusedItems(), "Color")
	if ci < 0 {
		t.Fatal("no Color item in Border submenu")
	}
	m.setFocusedIndex(ci)
	m.menuDescend()
	// Pick "Green" (palette index 2).
	items := m.focusedItems()
	gi := labelIndex(items, "Green")
	if gi < 0 {
		t.Fatal("no Green item in Color submenu")
	}
	m.activateMenuItem(items[gi].Action, items[gi].Arg)

	if m.getCanvas().boxes[0].Color != 2 {
		t.Fatalf("expected box 0 color 2 (green), got %d", m.getCanvas().boxes[0].Color)
	}
	if m.mode != ModeNormal {
		t.Fatalf("expected menu to close (ModeNormal), got %v", m.mode)
	}
	// Undo restores the default.
	m.undo()
	if m.getCanvas().boxes[0].Color != -1 {
		t.Fatalf("expected undo to restore color -1, got %d", m.getCanvas().boxes[0].Color)
	}
	// Redo re-applies.
	m.redo()
	if m.getCanvas().boxes[0].Color != 2 {
		t.Fatalf("expected redo to restore color 2, got %d", m.getCanvas().boxes[0].Color)
	}
}

// TestMenuEditTitleEntersMode verifies the new Edit Title menu item.
func TestMenuEditTitleEntersMode(t *testing.T) {
	m := newTestModel()
	out, _ := m.Update(press(tea.MouseButtonRight, 6, 4))
	m = out.(model)
	idx := -1
	for i, it := range m.menuItems {
		if it.Label == "Edit Title" {
			idx = i
		}
	}
	if idx < 0 {
		t.Fatal("no Edit Title item")
	}
	cmd := m.activateMenuItem(m.menuItems[idx].Action, m.menuItems[idx].Arg)
	_ = cmd
	if m.mode != ModeTitleEdit {
		t.Fatalf("expected ModeTitleEdit, got %v", m.mode)
	}
	if m.titleEditBoxID != 0 {
		t.Fatalf("expected titleEditBoxID 0, got %d", m.titleEditBoxID)
	}
}
