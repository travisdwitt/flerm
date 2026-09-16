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

// TestFileSearchAndScroll covers the open dialog's fuzzy search and its
// scroll window, sized here to 10 rows by the terminal height.
func TestFileSearchAndScroll(t *testing.T) {
	m := newTestModel()
	m.mode = ModeFileInput
	m.fileOp = FileOpOpen
	m.width, m.height = 96, 18
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

	cfg := &Config{}
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
	start := newTestModel()
	start.mode = ModeStartup
	start.lastFile = ""
	if strings.Contains(start.View(), "Resume") {
		t.Fatal("start menu should not offer resume without a remembered chart")
	}
}

func TestBufferBarDoesNotShiftOnSelect(t *testing.T) {
	m := newTestModel()
	m.addNewBuffer(cv.NewCanvas(), "/tmp/second-chart.sav")
	m.addNewBuffer(cv.NewCanvas(), "")

	plain := func(i int) string {
		m.currentBufferIndex = i
		return ansiRE.ReplaceAllString(m.renderBufferBar(120), "")
	}
	first, second, third := plain(0), plain(1), plain(2)

	if len(first) != 120 || len(second) != 120 || len(third) != 120 {
		t.Fatalf("expected every bar to fill the width, got %d/%d/%d", len(first), len(second), len(third))
	}
	if !strings.Contains(first, "[Buffer 1]") || !strings.Contains(second, "[second-chart]") {
		t.Fatalf("expected the selected buffer to be bracketed:\n%q\n%q", first, second)
	}

	debracket := func(s string) string {
		return strings.NewReplacer("[", " ", "]", " ").Replace(s)
	}
	if debracket(first) != debracket(second) || debracket(second) != debracket(third) {
		t.Fatalf("selecting a buffer moved the others:\n%q\n%q\n%q",
			debracket(first), debracket(second), debracket(third))
	}
}

// TestSaveExtensionAndLegacyOpen covers the file extensions: new charts are
// written as .flerm, while charts left over as .sav still open by bare name.
func TestSaveExtensionAndLegacyOpen(t *testing.T) {
	dir := t.TempDir()
	legacy := filepath.Join(dir, "old-chart.sav")
	c := cv.NewCanvas()
	c.AddBox(3, 3, "Legacy")
	if err := c.SaveToFileWithPan(legacy, 0, 0); err != nil {
		t.Fatal(err)
	}

	m := newTestModel()
	m.config = &Config{SaveDirectory: dir}
	m.mode = ModeFileInput
	m.fileOp = FileOpSave
	m.filename = "new_chart"
	out, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = out.(model)
	if m.errorMessage != "" {
		t.Fatalf("save failed: %s", m.errorMessage)
	}
	if got := m.getCurrentBuffer().filename; got != filepath.Join(dir, "new_chart.flerm") {
		t.Fatalf("expected new_chart.flerm, got %q", got)
	}

	// The open dialog lists both extensions, stripped to the same display name.
	m.mode = ModeFileInput
	m.fileOp = FileOpOpen
	m.allFiles = nil
	m.scanTxtFiles()
	if len(m.allFiles) != 2 || m.allFiles[0] != "new_chart.flerm" || m.allFiles[1] != "old-chart.sav" {
		t.Fatalf("expected both charts listed, got %v", m.allFiles)
	}

	// A typed name without an extension still finds the legacy chart.
	m.filename = "old-chart"
	m.selectedFileIndex = -1
	m.fileList = nil
	out, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = out.(model)
	if m.errorMessage != "" {
		t.Fatalf("open failed: %s", m.errorMessage)
	}
	if got := m.getCurrentBuffer().filename; got != legacy {
		t.Fatalf("expected to open %q, got %q", legacy, got)
	}
	if boxes := m.getCanvas().Boxes(); len(boxes) != 1 || boxes[0].GetText() != "Legacy" {
		t.Fatalf("expected the legacy chart's contents, got %v", boxes)
	}
}

// TestFileMenuMouse covers the open dialog's mouse handling: the wheel
// scrolls, a click picks a row, a second click on it opens, and dragging the
// scrollbar scrolls without disturbing the highlight.
func TestFileMenuMouse(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := t.TempDir()
	for _, n := range []string{"alpha", "beta", "delta", "gamma", "omega", "sigma",
		"theta", "zeta", "kappa", "lambda", "mu", "nu"} {
		c := cv.NewCanvas()
		c.AddBox(2, 2, n)
		if err := c.SaveToFileWithPan(filepath.Join(dir, n+saveExt), 0, 0); err != nil {
			t.Fatal(err)
		}
	}

	m := newTestModel()
	m.config = &Config{SaveDirectory: dir}
	m.mode = ModeFileInput
	m.fileOp = FileOpOpen
	m.width, m.height = 96, 18 // a 10-row list window for 12 charts
	m.scanTxtFiles()

	x, y, w, rows := m.fileMenuBounds()
	if len(m.fileList) != 12 || rows != 10 {
		t.Fatalf("expected 12 charts in a 10-row window, got %d in %d", len(m.fileList), rows)
	}
	listY := y + 3 // first list row, matching renderFileMenu's chrome

	// The wheel scrolls the window and leaves the highlight where it was.
	out, _ := m.Update(press(tea.MouseButtonWheelDown, x+2, listY))
	m = out.(model)
	if m.fileScroll != 2 || m.selectedFileIndex != 0 {
		t.Fatalf("expected wheel to scroll to 2 keeping selection 0, got %d/%d", m.fileScroll, m.selectedFileIndex)
	}
	out, _ = m.Update(press(tea.MouseButtonWheelUp, x+2, listY))
	m = out.(model)
	if m.fileScroll != 0 {
		t.Fatalf("expected wheel up to scroll back to 0, got %d", m.fileScroll)
	}

	// A click picks the row under the pointer. Sorted, index 4 is kappa.
	out, _ = m.Update(press(tea.MouseButtonLeft, x+3, listY+4))
	m = out.(model)
	if m.selectedFileIndex != 4 || m.filename != "kappa" {
		t.Fatalf("expected the click to select kappa at 4, got %d/%q", m.selectedFileIndex, m.filename)
	}

	// Dragging the scrollbar to the bottom scrolls without reselecting.
	out, _ = m.Update(press(tea.MouseButtonLeft, x+w-2, listY+rows-1))
	m = out.(model)
	if !m.draggingFileScroll || m.fileScroll != m.fileScrollMax() {
		t.Fatalf("expected a scrollbar drag to the end, got dragging=%v scroll %d", m.draggingFileScroll, m.fileScroll)
	}
	out, _ = m.Update(dragMotion(x+w-2, listY))
	m = out.(model)
	if m.fileScroll != 0 {
		t.Fatalf("expected the drag back to the top to scroll to 0, got %d", m.fileScroll)
	}
	out, _ = m.Update(release(x+w-2, listY))
	m = out.(model)
	if m.draggingFileScroll || m.selectedFileIndex != 4 {
		t.Fatalf("expected release to end the drag with kappa still selected, got dragging=%v sel %d",
			m.draggingFileScroll, m.selectedFileIndex)
	}

	// Clicking the already-selected row opens it.
	out, _ = m.Update(press(tea.MouseButtonLeft, x+3, listY+4))
	m = out.(model)
	if m.mode != ModeNormal {
		t.Fatalf("expected the second click to open the chart, got mode %v err %q", m.mode, m.errorMessage)
	}
	if got := m.getCanvas().Boxes(); len(got) != 1 || got[0].GetText() != "kappa" {
		t.Fatalf("expected the opened chart's single kappa box, got %v", got)
	}
}
