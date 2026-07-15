package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

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

	m.menuIndex = labelIndex(m.menuItems, "Border")
	if m.menuIndex < 0 {
		t.Fatal("no Border item in box menu")
	}
	m.menuDescend()
	ci := labelIndex(m.focusedItems(), "Color")
	if ci < 0 {
		t.Fatal("no Color item in Border submenu")
	}
	m.setFocusedIndex(ci)
	m.menuDescend()
	items := m.focusedItems()
	gi := labelIndex(items, "Green")
	if gi < 0 {
		t.Fatal("no Green item in Color submenu")
	}
	m.activateMenuItem(items[gi].Action, items[gi].Arg)

	if m.getCanvas().Boxes()[0].Color != 2 {
		t.Fatalf("expected box 0 color 2 (green), got %d", m.getCanvas().Boxes()[0].Color)
	}
	if m.mode != ModeNormal {
		t.Fatalf("expected menu to close (ModeNormal), got %v", m.mode)
	}
	m.undo()
	if m.getCanvas().Boxes()[0].Color != -1 {
		t.Fatalf("expected undo to restore color -1, got %d", m.getCanvas().Boxes()[0].Color)
	}
	m.redo()
	if m.getCanvas().Boxes()[0].Color != 2 {
		t.Fatalf("expected redo to restore color 2, got %d", m.getCanvas().Boxes()[0].Color)
	}
}

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
