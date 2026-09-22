package tui

import (
	"os"
	"strings"
	"testing"

	cv "flerm/internal/canvas"

	"github.com/atotto/clipboard"

	tea "github.com/charmbracelet/bubbletea"
)

func menuLabelIndex(items []MenuItem, label string) int {
	for i, it := range items {
		if it.Label == label {
			return i
		}
	}
	return -1
}

func TestDragText(t *testing.T) {
	m := newTestModel()
	c := m.getCanvas()
	c.AddText(60, 10, "hello")
	sx, sy := c.Texts()[0].X, c.Texts()[0].Y

	out, _ := m.Update(press(tea.MouseButtonLeft, 61, 10))
	m = out.(model)
	if !m.draggingText || m.dragTextID != 0 {
		t.Fatalf("expected text drag, got dragging=%v id=%d", m.draggingText, m.dragTextID)
	}
	out, _ = m.Update(dragMotion(71, 15))
	m = out.(model)
	out, _ = m.Update(release(71, 15))
	m = out.(model)

	c = m.getCanvas()
	if c.Texts()[0].X != sx+10 || c.Texts()[0].Y != sy+5 {
		t.Fatalf("expected text moved by (10,5), got (%d,%d)", c.Texts()[0].X-sx, c.Texts()[0].Y-sy)
	}
	m.undo()
	if m.getCanvas().Texts()[0].X != sx || m.getCanvas().Texts()[0].Y != sy {
		t.Fatal("expected undo to restore text position")
	}
}

func TestTextClickWithoutMoveNoUndo(t *testing.T) {
	m := newTestModel()
	m.getCanvas().AddText(60, 10, "hello")
	undosBefore := len(m.getCurrentBuffer().undoStack)
	m = click(m, tea.MouseButtonLeft, 61, 10)
	if len(m.getCurrentBuffer().undoStack) != undosBefore {
		t.Fatal("a text click without movement should not record an undo")
	}
}

func TestNewLineFromLine(t *testing.T) {
	m := newTestModel()
	c := m.getCanvas()
	c.AddConnectionWithWaypoints(0, 1, 11, 4, 40, 21, []point{{X: 25, Y: 4}, {X: 25, Y: 21}})
	connsBefore := len(c.Connections())

	out, _ := m.Update(press(tea.MouseButtonRight, 25, 10))
	m = out.(model)
	if m.menuTargetConn == -1 {
		t.Fatalf("expected right-click to target the connection, got conn=%d box=%d", m.menuTargetConn, m.menuTargetBox)
	}
	idx := menuLabelIndex(m.menuItems, "New Line")
	if idx == -1 {
		t.Fatal("connection menu missing New Line")
	}
	m.activateMenuItem(m.menuItems[idx].Action, m.menuItems[idx].Arg)
	if !m.mouseLineDrawing || m.connectionFrom != -1 || m.connectionFromLine == -1 {
		t.Fatalf("expected line-origin draw, got drawing=%v from=%d fromLine=%d", m.mouseLineDrawing, m.connectionFrom, m.connectionFromLine)
	}

	out, _ = m.Update(press(tea.MouseButtonLeft, 42, 21))
	m = out.(model)
	c = m.getCanvas()
	if len(c.Connections()) != connsBefore+1 {
		t.Fatalf("expected a new connection, before=%d after=%d", connsBefore, len(c.Connections()))
	}
	nc := c.Connections()[len(c.Connections())-1]
	if nc.FromID != -1 || nc.ToID != 1 {
		t.Fatalf("expected new line from a line point (-1) to box 1, got %d->%d", nc.FromID, nc.ToID)
	}
}

func TestNewLineNodeBend(t *testing.T) {
	m := newTestModel()
	out, _ := m.Update(press(tea.MouseButtonRight, 8, 4))
	m = out.(model)
	idx := menuLabelIndex(m.menuItems, "New Line")
	m.activateMenuItem(m.menuItems[idx].Action, m.menuItems[idx].Arg)
	if !m.mouseLineDrawing || m.connectionFrom != 0 {
		t.Fatalf("expected box-origin draw from box 0, got drawing=%v from=%d", m.mouseLineDrawing, m.connectionFrom)
	}

	out, _ = m.Update(press(tea.MouseButtonLeft, 8, 15))
	m = out.(model)
	if len(m.connectionWaypoints) != 1 || m.connectionWaypoints[0] != (point{X: 8, Y: 15}) {
		t.Fatalf("expected a node at (8,15), got %v", m.connectionWaypoints)
	}
	out, _ = m.Update(press(tea.MouseButtonLeft, 42, 21))
	m = out.(model)
	c := m.getCanvas()
	nc := c.Connections()[len(c.Connections())-1]
	if len(nc.Waypoints) == 0 {
		t.Fatal("expected the finished line to keep its node/bend")
	}
}

func TestDragEmptyAreaPansView(t *testing.T) {
	m := newTestModel()
	buf := m.getCurrentBuffer()
	buf.panX, buf.panY = 0, 0
	// Press on empty space, drag, release.
	out, _ := m.Update(press(tea.MouseButtonLeft, 80, 30))
	m = out.(model)
	if !m.panningView {
		t.Fatal("expected pan-drag to start on empty press")
	}
	out, _ = m.Update(dragMotion(70, 25)) // move pointer left/up by (10,5)
	m = out.(model)
	out, _ = m.Update(release(70, 25))
	m = out.(model)
	// Grab-and-pull: pointer moved (-10,-5), so panX/panY increase by (10,5).
	if b := m.getCurrentBuffer(); b.panX != 10 || b.panY != 5 {
		t.Fatalf("expected pan (10,5), got (%d,%d)", b.panX, b.panY)
	}
	if m.panningView {
		t.Fatal("expected pan to end on release")
	}
}

func TestClickEmptyStillClearsSelection(t *testing.T) {
	m := newTestModel()
	m = click(m, tea.MouseButtonLeft, 6, 4) // select box 0
	if m.selBox != 0 {
		t.Fatalf("expected box selected, got %d", m.selBox)
	}
	m = click(m, tea.MouseButtonLeft, 100, 35) // click empty (no drag)
	if m.selBox != -1 {
		t.Fatalf("expected empty click to clear selection, got selBox=%d", m.selBox)
	}
	if m.panningView {
		t.Fatal("pan flag should be reset after a plain click")
	}
}

func TestMultiSelectDragSelectsBoxes(t *testing.T) {
	m := newTestModel() // box0 Alpha @ (5,3), box1 Beta @ (40,20)
	out, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("M")})
	m = out.(model)
	if m.mode != ModeMultiSelect {
		t.Fatalf("expected ModeMultiSelect, got %v", m.mode)
	}
	// Drag a rectangle covering box 0 only.
	out, _ = m.Update(press(tea.MouseButtonLeft, 4, 2))
	m = out.(model)
	out, _ = m.Update(dragMotion(20, 8))
	m = out.(model)
	out, _ = m.Update(release(20, 8))
	m = out.(model)
	if len(m.selectedBoxes) != 1 || m.selectedBoxes[0] != 0 {
		t.Fatalf("expected box 0 selected, got %v", m.selectedBoxes)
	}
	if m.mode != ModeMove {
		t.Fatalf("expected ModeMove after selecting, got %v", m.mode)
	}
}

func TestGroupDragMovesAndHighlights(t *testing.T) {
	m := newTestModel() // box0 @ (5,3), box1 @ (40,20)
	out, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("M")})
	m = out.(model)
	// Rubber-band a rectangle over both boxes.
	out, _ = m.Update(press(tea.MouseButtonLeft, 2, 1))
	m = out.(model)
	out, _ = m.Update(dragMotion(60, 30))
	m = out.(model)
	out, _ = m.Update(release(60, 30))
	m = out.(model)
	if m.mode != ModeMove || len(m.selectedBoxes) != 2 {
		t.Fatalf("expected ModeMove with 2 boxes, got mode=%v boxes=%v", m.mode, m.selectedBoxes)
	}

	c := m.getCanvas()
	b0, b1 := c.Boxes()[0], c.Boxes()[1]

	// The whole group is highlighted.
	rr := c.RenderRaw(120, 40, -1, CoordUnset, CoordUnset, nil, CoordUnset, CoordUnset, 0, 0, -1, -1, false, -1, -1, 0, "", CoordUnset, CoordUnset, CoordUnset, CoordUnset, CoordUnset, CoordUnset, false, -1, -1)
	m.overlaySelection(rr, 0, 0)
	if rr.ColorMap[b0.Y][b0.X] != colorMouseSelect || rr.ColorMap[b1.Y][b1.X] != colorMouseSelect {
		t.Fatal("expected both boxes highlighted while selected")
	}

	// Drag the group by (+5,+2) with the mouse.
	out, _ = m.Update(press(tea.MouseButtonLeft, 6, 4))
	m = out.(model)
	out, _ = m.Update(dragMotion(11, 6))
	m = out.(model)
	out, _ = m.Update(release(11, 6))
	m = out.(model)
	c = m.getCanvas()
	if c.Boxes()[0].X != b0.X+5 || c.Boxes()[0].Y != b0.Y+2 || c.Boxes()[1].X != b1.X+5 || c.Boxes()[1].Y != b1.Y+2 {
		t.Fatalf("group didn't move by (5,2): box0=(%d,%d) box1=(%d,%d)", c.Boxes()[0].X, c.Boxes()[0].Y, c.Boxes()[1].X, c.Boxes()[1].Y)
	}
	if m.mode != ModeNormal {
		t.Fatalf("expected ModeNormal after drop, got %v", m.mode)
	}
	// The move is undoable (one entry per box).
	m.undo()
	m.undo()
	c = m.getCanvas()
	if c.Boxes()[0].X != b0.X || c.Boxes()[0].Y != b0.Y || c.Boxes()[1].X != b1.X || c.Boxes()[1].Y != b1.Y {
		t.Fatalf("undo didn't restore the group: box0=(%d,%d) box1=(%d,%d)", c.Boxes()[0].X, c.Boxes()[0].Y, c.Boxes()[1].X, c.Boxes()[1].Y)
	}
}

func TestMultiSelectEscCancels(t *testing.T) {
	m := newTestModel()
	out, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("M")})
	m = out.(model)
	if m.mode != ModeMultiSelect {
		t.Fatalf("expected ModeMultiSelect, got %v", m.mode)
	}
	out, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	m = out.(model)
	if m.mode != ModeNormal {
		t.Fatalf("expected Esc to cancel to ModeNormal, got %v", m.mode)
	}
	if m.selectionStartX != CoordUnset {
		t.Fatalf("expected selection reset on cancel, got selectionStartX=%d", m.selectionStartX)
	}
}

func TestConnectionRendersWithoutGaps(t *testing.T) {
	c := cv.NewCanvas()
	c.AddBox(2, 2, "A")
	c.Boxes()[0].Width, c.Boxes()[0].Height = 7, 3
	// A path that doubles back vertically (up at x=30) and ends at a free point.
	c.AddConnectionWithWaypoints(0, -1, 8, 3, 40, 12,
		[]point{{X: 20, Y: 3}, {X: 20, Y: 8}, {X: 30, Y: 8}, {X: 30, Y: 3}})
	rr := c.RenderRaw(50, 16, -1, CoordUnset, CoordUnset, nil, CoordUnset, CoordUnset, 0, 0, -1, -1, false, -1, -1, 0, "", CoordUnset, CoordUnset, CoordUnset, CoordUnset, CoordUnset, CoordUnset, false, -1, -1)

	// Every cell on the path must be drawn (no breaks).
	for _, p := range c.GetConnectionCells(0) {
		if p.Y < 0 || p.Y >= len(rr.Canvas) || p.X < 0 || p.X >= len(rr.Canvas[p.Y]) {
			continue
		}
		if rr.Canvas[p.Y][p.X] == ' ' {
			t.Fatalf("gap in line at (%d,%d)", p.X, p.Y)
		}
	}
	// Bends use the right elbow glyphs (incl. the doubling-back turn at (30,3)).
	if rr.Canvas[3][20] != '┐' || rr.Canvas[8][20] != '└' || rr.Canvas[8][30] != '┘' || rr.Canvas[3][30] != '┌' {
		t.Fatalf("wrong corner glyphs: (20,3)=%q (20,8)=%q (30,8)=%q (30,3)=%q",
			string(rr.Canvas[3][20]), string(rr.Canvas[8][20]), string(rr.Canvas[8][30]), string(rr.Canvas[3][30]))
	}
}

func TestHighlightPaintDrag(t *testing.T) {
	m := newTestModel()
	m.highlightMode = true
	m.selectedColor = 2 // green

	out, _ := m.Update(press(tea.MouseButtonLeft, 30, 30))
	m = out.(model)
	if !m.paintingHighlight {
		t.Fatal("expected painting to start")
	}
	out, _ = m.Update(dragMotion(34, 30))
	m = out.(model)
	out, _ = m.Update(release(34, 30))
	m = out.(model)

	c := m.getCanvas()
	for x := 30; x <= 34; x++ {
		if c.GetHighlight(x, 30) != 2 {
			t.Fatalf("expected highlight color 2 at (%d,30), got %d", x, c.GetHighlight(x, 30))
		}
	}
	rr := c.RenderRaw(120, 40, -1, CoordUnset, CoordUnset, nil, CoordUnset, CoordUnset, 0, 0, -1, -1, false, -1, -1, 0, "", CoordUnset, CoordUnset, CoordUnset, CoordUnset, CoordUnset, CoordUnset, false, -1, -1)
	if rr.ColorMap[30][32] != 2 {
		t.Fatalf("expected painted cell to render color 2, got %d", rr.ColorMap[30][32])
	}
	m.undo()
	if m.getCanvas().GetHighlight(32, 30) != -1 {
		t.Fatal("expected undo to clear the painted stroke")
	}
}

func TestEscapeKeepsTypedText(t *testing.T) {
	typeKeys := func(m model, s string) model {
		for _, r := range s {
			out, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			m = out.(model)
		}
		out, _ := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
		return out.(model)
	}

	m := newTestModel()
	m.cursorX, m.cursorY = 6, 4
	out, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m = out.(model)
	if m.mode != ModeEditing {
		t.Fatalf("expected ModeEditing, got %v", m.mode)
	}
	for range "Alpha" {
		out, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
		m = out.(model)
	}
	m = typeKeys(m, "Kept")
	if m.mode != ModeNormal {
		t.Fatalf("expected Esc to return to normal mode, got %v", m.mode)
	}
	if got := m.getCanvas().Boxes()[0].GetText(); got != "Kept" {
		t.Fatalf("expected box text %q, got %q", "Kept", got)
	}
	m.undo()
	if got := m.getCanvas().Boxes()[0].GetText(); got != "Alpha" {
		t.Fatalf("expected undo to restore %q, got %q", "Alpha", got)
	}

	m = newTestModel()
	m.cursorX, m.cursorY = 6, 4
	out, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'T'}})
	m = out.(model)
	if m.mode != ModeTitleEdit {
		t.Fatalf("expected ModeTitleEdit, got %v", m.mode)
	}
	m = typeKeys(m, "Title")
	if got := m.getCanvas().Boxes()[0].Title; got != "Title" {
		t.Fatalf("expected title %q, got %q", "Title", got)
	}
	m.undo()
	if got := m.getCanvas().Boxes()[0].Title; got != "" {
		t.Fatalf("expected undo to clear the title, got %q", got)
	}

	m = newTestModel()
	m.cursorX, m.cursorY = 60, 30
	out, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	m = out.(model)
	if m.mode != ModeTextInput {
		t.Fatalf("expected ModeTextInput, got %v", m.mode)
	}
	before := len(m.getCanvas().Texts())
	m = typeKeys(m, "note")
	texts := m.getCanvas().Texts()
	if len(texts) != before+1 || texts[len(texts)-1].GetText() != "note" {
		t.Fatalf("expected Esc to add the typed text, got %d texts", len(texts))
	}
}

func TestShiftHomeEndSelectAndCopy(t *testing.T) {
	m := newTestModel()
	m.mode = ModeEditing
	m.selectedBox = 0
	m.editText = "hello world"
	m.originalEditText = m.editText
	m.editCursorPos = len(m.editText)
	m.clearEditSelection()
	m.syncCursorPositions()

	key := func(m model, t tea.KeyType) model {
		out, _ := m.Update(tea.KeyMsg{Type: t})
		return out.(model)
	}

	m = key(m, tea.KeyShiftHome)
	if start, end := m.getEditSelectionBounds(); start != 0 || end != len(m.editText) {
		t.Fatalf("expected the whole line selected, got %d-%d", start, end)
	}
	m = key(m, tea.KeyShiftEnd)
	if m.hasEditSelection() {
		t.Fatal("expected Shift+End to collapse the selection back to the line end")
	}
	m.editCursorPos = 5 // "hello|"
	m.syncCursorPositions()
	m = key(m, tea.KeyShiftHome)
	if start, end := m.getEditSelectionBounds(); start != 0 || end != 5 {
		t.Fatalf("expected 0-5 selected, got %d-%d", start, end)
	}

	restore, _ := clipboard.ReadAll()
	if err := clipboard.WriteAll("probe"); err != nil {
		t.Skipf("no usable clipboard here: %v", err)
	}
	defer clipboard.WriteAll(restore)

	out, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = out.(model)
	if m.editText != "hello world" {
		t.Fatalf("expected 'y' to leave the text alone, got %q", m.editText)
	}
	if got, err := clipboard.ReadAll(); err != nil || got != "hello" {
		t.Fatalf("expected %q on the clipboard, got %q (err %v)", "hello", got, err)
	}
	if !m.hasEditSelection() {
		t.Fatal("expected the selection to survive a copy")
	}

	m.clearEditSelection()
	out, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = out.(model)
	if m.editText != "yhello world" {
		t.Fatalf("expected 'y' to be typed, got %q", m.editText)
	}
}

func TestHelpScrollAndExit(t *testing.T) {
	key := func(m model, s string) model {
		out, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)})
		return out.(model)
	}
	special := func(m model, k tea.KeyType) model {
		out, _ := m.Update(tea.KeyMsg{Type: k})
		return out.(model)
	}

	m := newTestModel()
	m.height = 11
	m = key(m, "?")
	if !m.help {
		t.Fatal("expected '?' to open the help screen")
	}

	m = special(m, tea.KeyPgDown)
	if m.helpScroll != 10 {
		t.Fatalf("expected PgDn to scroll one page (10), got %d", m.helpScroll)
	}
	m = special(m, tea.KeyPgUp)
	if m.helpScroll != 0 {
		t.Fatalf("expected PgUp to scroll back to the top, got %d", m.helpScroll)
	}
	m = special(m, tea.KeyPgUp)
	if m.helpScroll != 0 {
		t.Fatalf("expected PgUp at the top to stay put, got %d", m.helpScroll)
	}
	for i := 0; i < len(helpText); i++ {
		m = special(m, tea.KeyPgDown)
	}
	if want := m.helpMaxScroll(); m.helpScroll != want {
		t.Fatalf("expected paging past the end to stop at %d, got %d", want, m.helpScroll)
	}

	for _, k := range []string{"?", "n", "x", "e"} {
		if m = key(m, k); !m.help {
			t.Fatalf("expected %q to leave the help screen open", k)
		}
	}
	if m = special(m, tea.KeyEnter); !m.help {
		t.Fatal("expected Enter to leave the help screen open")
	}

	if esc := special(m, tea.KeyEscape); esc.help || esc.helpScroll != 0 {
		t.Fatalf("expected Esc to close help and reset scroll, got help=%v scroll=%d", esc.help, esc.helpScroll)
	}
	if q := key(m, "q"); q.help || q.helpScroll != 0 {
		t.Fatalf("expected q to close help and reset scroll, got help=%v scroll=%d", q.help, q.helpScroll)
	}
}

func TestQuitWarnsAboutUnsavedChanges(t *testing.T) {
	key := func(m model, s string) model {
		out, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)})
		return out.(model)
	}

	m := newTestModel()
	m.config = &Config{Confirmations: false}
	if m.unsavedChanges() {
		t.Fatal("a freshly loaded chart should be clean")
	}
	if quit := key(m, "q"); quit.mode != ModeNormal {
		t.Fatalf("a clean chart with confirmations off should quit outright, got mode %v", quit.mode)
	}

	m.cursorX, m.cursorY = 60, 30
	m = key(m, "b")
	if !m.unsavedChanges() {
		t.Fatal("adding a box should mark the chart unsaved")
	}
	m = key(m, "q")
	if m.mode != ModeConfirm || m.confirmAction != ConfirmQuit {
		t.Fatalf("expected unsaved work to force a quit prompt, got mode %v action %v", m.mode, m.confirmAction)
	}
	if !strings.Contains(m.View(), "Unsaved changes will be lost") {
		t.Fatalf("expected the prompt to warn about unsaved changes:\n%s", m.View())
	}

	m.mode = ModeNormal
	m.undo()
	if m.unsavedChanges() {
		t.Fatal("undoing the only edit should leave the chart clean")
	}

	m = key(m, "b")
	buf := m.getCurrentBuffer()
	buf.savedAt = len(buf.undoStack)
	if m.unsavedChanges() {
		t.Fatal("a saved chart should be clean at its current undo depth")
	}
	m = key(m, "b")
	if !m.unsavedChanges() {
		t.Fatal("editing after a save should mark the chart unsaved again")
	}
}

func TestHighlightRightDragErases(t *testing.T) {
	m := newTestModel()
	m.highlightMode = true
	m.selectedColor = 2

	for _, msg := range []tea.MouseMsg{press(tea.MouseButtonLeft, 30, 30), dragMotion(34, 30), release(34, 30)} {
		out, _ := m.Update(msg)
		m = out.(model)
	}

	rightDrag := tea.MouseMsg{X: 33, Y: 30, Action: tea.MouseActionMotion, Button: tea.MouseButtonRight}
	out, _ := m.Update(press(tea.MouseButtonRight, 31, 30))
	m = out.(model)
	if !m.paintingHighlight {
		t.Fatal("expected a right press to start erasing")
	}
	if m.mode != ModeNormal {
		t.Fatalf("erasing must not open the context menu, got mode %v", m.mode)
	}
	out, _ = m.Update(rightDrag)
	m = out.(model)
	out, _ = m.Update(release(33, 30))
	m = out.(model)

	c := m.getCanvas()
	for x := 31; x <= 33; x++ {
		if c.GetHighlight(x, 30) != -1 {
			t.Fatalf("expected (%d,30) erased, got %d", x, c.GetHighlight(x, 30))
		}
	}
	for _, x := range []int{30, 34} {
		if c.GetHighlight(x, 30) != 2 {
			t.Fatalf("expected (%d,30) untouched, got %d", x, c.GetHighlight(x, 30))
		}
	}

	m.undo()
	for x := 30; x <= 34; x++ {
		if m.getCanvas().GetHighlight(x, 30) != 2 {
			t.Fatalf("expected undo to restore (%d,30), got %d", x, m.getCanvas().GetHighlight(x, 30))
		}
	}
	m.redo()
	for x := 31; x <= 33; x++ {
		if m.getCanvas().GetHighlight(x, 30) != -1 {
			t.Fatalf("expected redo to erase (%d,30) again, got %d", x, m.getCanvas().GetHighlight(x, 30))
		}
	}
}

func TestDuplicateBoxKeepsStyleAndHighlights(t *testing.T) {
	m := newTestModel()
	c := m.getCanvas()
	c.Boxes()[0].Title = "Title"
	c.Boxes()[0].BorderStyle = BorderStyleDouble
	c.SetBoxColor(0, 3)
	c.Boxes()[0].UpdateSize()
	src := c.Boxes()[0]
	c.SetHighlight(src.X, src.Y, 5)

	m.cursorX, m.cursorY = src.X+1, src.Y+1
	out, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	m = out.(model)

	c = m.getCanvas()
	if len(c.Boxes()) != 3 {
		t.Fatalf("expected a third box, got %d", len(c.Boxes()))
	}
	dup := c.Boxes()[2]
	if dup.X != src.X+dupOffsetX || dup.Y != src.Y+dupOffsetY {
		t.Fatalf("expected copy at (%d,%d), got (%d,%d)", src.X+dupOffsetX, src.Y+dupOffsetY, dup.X, dup.Y)
	}
	if dup.Title != "Title" || dup.BorderStyle != BorderStyleDouble || dup.Color != 3 || dup.GetText() != src.GetText() {
		t.Fatalf("copy lost styling: %+v", dup)
	}
	if got := c.GetHighlight(src.X+dupOffsetX, src.Y+dupOffsetY); got != 5 {
		t.Fatalf("expected the highlight copied to the offset cell, got %d", got)
	}
	if got := c.GetHighlight(src.X, src.Y); got != 5 {
		t.Fatalf("expected the source highlight untouched, got %d", got)
	}

	m.undo()
	if len(m.getCanvas().Boxes()) != 2 {
		t.Fatal("undo should remove the copy")
	}
	if got := m.getCanvas().GetHighlight(src.X+dupOffsetX, src.Y+dupOffsetY); got != -1 {
		t.Fatalf("undo should drop the copied highlight, got %d", got)
	}
	m.redo()
	if len(m.getCanvas().Boxes()) != 3 || m.getCanvas().Boxes()[2].Title != "Title" {
		t.Fatal("redo should restore the copy with its styling")
	}
}

func TestDuplicateTextFromContextMenu(t *testing.T) {
	m := newTestModel()
	c := m.getCanvas()
	c.AddText(60, 10, "hello")
	c.SetTextColor(0, 2)
	c.SetHighlight(60, 10, 4)

	m = click(m, tea.MouseButtonRight, 61, 10)
	idx := menuLabelIndex(m.menuItems, "Duplicate Text")
	if idx == -1 {
		t.Fatal("expected a Duplicate Text item in the text context menu")
	}
	m.menuIndex = idx
	out, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = out.(model)

	c = m.getCanvas()
	if len(c.Texts()) != 2 {
		t.Fatalf("expected a second text, got %d", len(c.Texts()))
	}
	dup := c.Texts()[1]
	if dup.X != 60+dupOffsetX || dup.Y != 10+dupOffsetY || dup.GetText() != "hello" || dup.Color != 2 {
		t.Fatalf("unexpected copy: %+v", dup)
	}
	if got := c.GetHighlight(60+dupOffsetX, 10+dupOffsetY); got != 4 {
		t.Fatalf("expected the highlight copied, got %d", got)
	}
}

func moveHighlightCase(t *testing.T, m model, hx, hy, dx, dy int, move func(model) model) {
	t.Helper()
	m.getCanvas().SetHighlight(hx, hy, 4)
	m = move(m)

	c := m.getCanvas()
	if got := c.GetHighlight(hx+dx, hy+dy); got != 4 {
		t.Fatalf("after move: want highlight at (%d,%d), got %d", hx+dx, hy+dy, got)
	}
	if got := c.GetHighlight(hx, hy); got != -1 {
		t.Fatalf("after move: highlight left behind at (%d,%d), got %d", hx, hy, got)
	}

	m.undo()
	c = m.getCanvas()
	if got := c.GetHighlight(hx, hy); got != 4 {
		t.Fatalf("after undo: want highlight back at (%d,%d), got %d", hx, hy, got)
	}
	if got := c.GetHighlight(hx+dx, hy+dy); got != -1 {
		t.Fatalf("after undo: highlight stranded at (%d,%d), got %d", hx+dx, hy+dy, got)
	}

	m.redo()
	c = m.getCanvas()
	if got := c.GetHighlight(hx+dx, hy+dy); got != 4 {
		t.Fatalf("after redo: want highlight at (%d,%d), got %d", hx+dx, hy+dy, got)
	}
	if got := c.GetHighlight(hx, hy); got != -1 {
		t.Fatalf("after redo: highlight left behind at (%d,%d), got %d", hx, hy, got)
	}
}

func keyMove(steps int) func(model) model {
	return func(m model) model {
		out, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
		m = out.(model)
		for i := 0; i < steps; i++ {
			out, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
			m = out.(model)
		}
		out, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		return out.(model)
	}
}

func TestMoveCarriesHighlights(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(model) model
		hx, hy, dx, dy int
		move           func(model) model
	}{
		{
			name: "box drag",
			hx:   5, hy: 3, dx: 10, dy: 5,
			move: func(m model) model {
				out, _ := m.Update(press(tea.MouseButtonLeft, 6, 4))
				m = out.(model)
				out, _ = m.Update(dragMotion(16, 9))
				m = out.(model)
				out, _ = m.Update(release(16, 9))
				return out.(model)
			},
		},
		{
			name:  "text drag",
			setup: func(m model) model { m.getCanvas().AddText(60, 10, "hello"); return m },
			hx:    60, hy: 10, dx: 10, dy: 5,
			move: func(m model) model {
				out, _ := m.Update(press(tea.MouseButtonLeft, 61, 10))
				m = out.(model)
				out, _ = m.Update(dragMotion(71, 15))
				m = out.(model)
				out, _ = m.Update(release(71, 15))
				return out.(model)
			},
		},
		{
			name:  "box keyboard move",
			setup: func(m model) model { m.cursorX, m.cursorY = 6, 4; return m },
			hx:    5, hy: 3, dx: 3, dy: 0,
			move: keyMove(3),
		},
		{
			name: "text keyboard move",
			setup: func(m model) model {
				m.getCanvas().AddText(60, 10, "hello")
				m.cursorX, m.cursorY = 60, 10
				return m
			},
			hx: 60, hy: 10, dx: 3, dy: 0,
			move: keyMove(3),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := newTestModel()
			if tc.setup != nil {
				m = tc.setup(m)
			}
			moveHighlightCase(t, m, tc.hx, tc.hy, tc.dx, tc.dy, tc.move)
		})
	}
}

func TestGroupMoveIsOneUndoStep(t *testing.T) {
	m := newTestModel()
	c := m.getCanvas()
	c.AddText(10, 8, "note")
	c.SetHighlight(5, 3, 4)
	c.SetHighlight(10, 8, 6)
	b0 := c.Boxes()[0]
	tx := c.Texts()[0]

	m.selectionStartX, m.selectionStartY = 0, 0
	m.mode = ModeMultiSelect
	m.finalizeMultiSelect(30, 15)
	if m.mode != ModeMove {
		t.Fatalf("expected ModeMove after selecting an area, got %v", m.mode)
	}
	if len(m.selectedBoxes) != 1 || len(m.selectedTexts) != 1 {
		t.Fatalf("expected box 0 and the text selected, got %v / %v", m.selectedBoxes, m.selectedTexts)
	}

	undosBefore := len(m.getCurrentBuffer().undoStack)
	m.handleMultiSelectMove(4, 2)
	m.commitMove()

	if got := len(m.getCurrentBuffer().undoStack) - undosBefore; got != 1 {
		t.Fatalf("a group move should record exactly one undo step, got %d", got)
	}
	c = m.getCanvas()
	if c.Boxes()[0].X != b0.X+4 || c.Texts()[0].X != tx.X+4 {
		t.Fatal("expected both objects moved")
	}
	if c.GetHighlight(9, 5) != 4 || c.GetHighlight(14, 10) != 6 {
		t.Fatalf("highlights did not follow the group: %d %d", c.GetHighlight(9, 5), c.GetHighlight(14, 10))
	}

	m.undo()
	c = m.getCanvas()
	if c.Boxes()[0].X != b0.X || c.Boxes()[0].Y != b0.Y || c.Texts()[0].X != tx.X || c.Texts()[0].Y != tx.Y {
		t.Fatal("one undo should restore every object's position")
	}
	if c.GetHighlight(5, 3) != 4 || c.GetHighlight(10, 8) != 6 {
		t.Fatal("undo should bring the highlights back")
	}
	if c.GetHighlight(9, 5) != -1 || c.GetHighlight(14, 10) != -1 {
		t.Fatal("undo should not strand highlights at the moved positions")
	}

	m.redo()
	c = m.getCanvas()
	if c.Boxes()[0].X != b0.X+4 || c.Texts()[0].X != tx.X+4 {
		t.Fatal("redo should reapply the group move")
	}
	if c.GetHighlight(9, 5) != 4 || c.GetHighlight(14, 10) != 6 {
		t.Fatal("redo should carry the highlights along")
	}
	if c.GetHighlight(5, 3) != -1 || c.GetHighlight(10, 8) != -1 {
		t.Fatal("redo should not leave highlights at the original positions")
	}
}

func TestGroupMoveRestoresConnections(t *testing.T) {
	m := newTestModel()
	c := m.getCanvas()
	c.AddConnection(0, 1)
	before := c.SnapshotConnections()

	m.selectionStartX, m.selectionStartY = 0, 0
	m.mode = ModeMultiSelect
	m.finalizeMultiSelect(80, 40)
	m.handleMultiSelectMove(5, 3)
	m.commitMove()

	m.undo()
	after := m.getCanvas().Connections()
	for i, conn := range before {
		if after[i].FromX != conn.FromX || after[i].FromY != conn.FromY ||
			after[i].ToX != conn.ToX || after[i].ToY != conn.ToY {
			t.Fatalf("connection %d not restored: %+v want %+v", i, after[i], conn)
		}
	}
}

func TestSelectAreaFromContextMenu(t *testing.T) {
	m := newTestModel()
	m = click(m, tea.MouseButtonRight, 90, 30)
	idx := menuLabelIndex(m.menuItems, "Select Area")
	if idx == -1 {
		t.Fatal("expected a Select Area item in the context menu")
	}
	m.menuIndex = idx
	out, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = out.(model)

	if m.mode != ModeMultiSelect {
		t.Fatalf("expected ModeMultiSelect, got %v", m.mode)
	}
	if m.selectionStartX != CoordUnset || m.selectionStartY != CoordUnset {
		t.Fatalf("expected no anchor until the user presses, got (%d,%d)", m.selectionStartX, m.selectionStartY)
	}

	out, _ = m.Update(release(90, 30))
	m = out.(model)
	if m.mode != ModeMultiSelect {
		t.Fatalf("the menu click's release ended the selection early, mode=%v", m.mode)
	}

	out, _ = m.Update(motion(20, 12))
	m = out.(model)
	if m.cursorX != 20 || m.cursorY != 12 {
		t.Fatalf("expected the cursor to follow the pointer, got (%d,%d)", m.cursorX, m.cursorY)
	}
	if m.selectionStartX != CoordUnset {
		t.Fatal("moving the mouse should not anchor the selection")
	}

	out, _ = m.Update(press(tea.MouseButtonLeft, 3, 2))
	m = out.(model)
	if m.selectionStartX != 3 || m.selectionStartY != 2 {
		t.Fatalf("expected the press to anchor at (3,2), got (%d,%d)", m.selectionStartX, m.selectionStartY)
	}
	out, _ = m.Update(dragMotion(30, 16))
	m = out.(model)
	out, _ = m.Update(release(30, 16))
	m = out.(model)

	if m.mode != ModeMove {
		t.Fatalf("expected ModeMove after the drag, got %v", m.mode)
	}
	if len(m.selectedBoxes) != 1 || m.selectedBoxes[0] != 0 {
		t.Fatalf("expected box 0 selected by the drag, got %v", m.selectedBoxes)
	}
}

func TestExportVisualTXTIncludesNegativeCoordinates(t *testing.T) {
	m := newTestModel()
	c := m.getCanvas()
	c.MoveBox(0, -20, -10)
	c.AddText(-18, -12, "offgrid")

	path := t.TempDir() + "/out.txt"
	if err := m.exportVisualTXT(path); err != nil {
		t.Fatalf("export: %v", err)
	}
	out, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	for _, want := range []string{"Alpha", "offgrid"} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("expected %q in the export, got:\n%s", want, out)
		}
	}
}

func TestExportPNGWithNegativeCoordinates(t *testing.T) {
	m := newTestModel()
	c := m.getCanvas()
	c.MoveBox(0, -30, -15)

	path := t.TempDir() + "/out.png"
	if err := c.ExportToPNG(path, 120, 40, 0, 0); err != nil {
		t.Fatalf("png export: %v", err)
	}
	if info, err := os.Stat(path); err != nil || info.Size() == 0 {
		t.Fatalf("expected a non-empty png, err=%v", err)
	}
}
