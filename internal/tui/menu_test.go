package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	cv "flerm/internal/canvas"

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

// TestFileSearchAndScroll covers the open dialog's fuzzy search and the
// 10-item scroll window.
func TestFileSearchAndScroll(t *testing.T) {
	m := newTestModel()
	m.mode = ModeFileInput
	m.fileOp = FileOpOpen
	m.width, m.height = 96, 30
	for _, n := range []string{"alpha", "beta", "delta", "gamma", "omega", "sigma",
		"theta", "zeta", "kappa", "lambda", "mu", "nu"} {
		m.allFiles = append(m.allFiles, n+".sav")
	}
	m.selectedFileIndex = 0
	m.applyFileFilter()

	// Scroll window: moving past item 10 scrolls, and the view shows 10 rows.
	for i := 0; i < 10; i++ {
		m.moveFileSelection(1)
	}
	if m.selectedFileIndex != 10 || m.fileScroll != 1 {
		t.Fatalf("expected selection 10 with scroll 1, got %d/%d", m.selectedFileIndex, m.fileScroll)
	}
	if !strings.Contains(m.View(), "> mu") || strings.Contains(m.View(), "alpha") {
		t.Fatal("view should show the scrolled window, not the first item")
	}
	m.moveFileSelection(-1) // back inside the window: no further scrolling
	if m.fileScroll != 1 {
		t.Fatalf("expected scroll to stay at 1, got %d", m.fileScroll)
	}

	// '?' enters search; typed runes fuzzy-filter; Enter opens the match.
	out, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	m = out.(model)
	if !m.fileSearch {
		t.Fatal("expected '?' to start the fuzzy search")
	}
	for _, r := range "gma" {
		out, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = out.(model)
	}
	if got := m.fileList; len(got) != 2 || got[0] != "gamma.sav" || got[1] != "sigma.sav" {
		t.Fatalf("expected gamma/sigma to match \"gma\", got %v", got)
	}
	if m.fileScroll != 0 || m.filename != "gamma" {
		t.Fatalf("expected filter to reset scroll and select gamma, got scroll %d name %q", m.fileScroll, m.filename)
	}

	// Esc leaves search with the full list back.
	out, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	m = out.(model)
	if m.fileSearch || m.mode != ModeFileInput || len(m.fileList) != 12 {
		t.Fatalf("expected Esc to exit search only, got search=%v mode=%v %d files", m.fileSearch, m.mode, len(m.fileList))
	}
}

// TestResumeLastChart covers the start menu's resume option: what gets
// remembered, what the menu offers, and the flermrc toggle.
func TestResumeLastChart(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	chart := filepath.Join(home, "resume-me.sav")
	c := cv.NewCanvas()
	c.AddBox(3, 3, "Resumed")
	if err := c.SaveToFileWithPan(chart, 0, 0); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{Resume: true}
	cfg.RememberLastFile(chart)
	if got := cfg.LastFile(); got != chart {
		t.Fatalf("expected %q remembered, got %q", chart, got)
	}

	m := newTestModel()
	m.mode = ModeStartup
	m.config = cfg
	m.lastFile = cfg.LastFile()
	if !strings.Contains(m.View(), "r: Resume resume-me") {
		t.Fatalf("start menu should offer the resume option:\n%s", m.View())
	}

	out, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m = out.(model)
	if m.mode != ModeNormal {
		t.Fatalf("expected 'r' to open the chart, got mode %v", m.mode)
	}
	if got := m.getCurrentBuffer().filename; got != chart {
		t.Fatalf("expected buffer filename %q, got %q", chart, got)
	}
	if len(m.getCanvas().Boxes()) != 1 || m.getCanvas().Boxes()[0].GetText() != "Resumed" {
		t.Fatal("expected the resumed chart's contents")
	}

	// A deleted chart is not offered, and resume=false hides the option.
	os.Remove(chart)
	if got := cfg.LastFile(); got != "" {
		t.Fatalf("expected a missing chart to be forgotten, got %q", got)
	}
	off := &Config{Resume: false}
	off.RememberLastFile(chart)
	if got := off.LastFile(); got != "" {
		t.Fatalf("expected resume=false to report no last chart, got %q", got)
	}

	start := newTestModel()
	start.mode = ModeStartup
	start.lastFile = ""
	if strings.Contains(start.View(), "Resume") {
		t.Fatal("start menu should not offer resume without a remembered chart")
	}
}
