package tui

import (
	"testing"

	cv "flerm/internal/canvas"

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
	// Connection box0 -> box1 with an explicit vertical segment at x=25.
	c.AddConnectionWithWaypoints(0, 1, 11, 4, 40, 21, []point{{X: 25, Y: 4}, {X: 25, Y: 21}})
	connsBefore := len(c.Connections())

	// Right-click a point on the vertical segment (not over any box).
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

	// Complete onto box1.
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
	// Start a line from box 0 via its menu.
	out, _ := m.Update(press(tea.MouseButtonRight, 8, 4))
	m = out.(model)
	idx := menuLabelIndex(m.menuItems, "New Line")
	m.activateMenuItem(m.menuItems[idx].Action, m.menuItems[idx].Arg)
	if !m.mouseLineDrawing || m.connectionFrom != 0 {
		t.Fatalf("expected box-origin draw from box 0, got drawing=%v from=%d", m.mouseLineDrawing, m.connectionFrom)
	}

	// Click an empty cell to drop a node, then click box 1 to finish.
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
	rr := c.RenderRaw(120, 40, -1, -1, -1, nil, -1, -1, 0, 0, -1, -1, false, -1, -1, 0, "", -1, -1, -1, -1, -1, -1, false, -1, -1)
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
	if m.selectionStartX != -1 {
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
	rr := c.RenderRaw(50, 16, -1, -1, -1, nil, -1, -1, 0, 0, -1, -1, false, -1, -1, 0, "", -1, -1, -1, -1, -1, -1, false, -1, -1)

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
	// It renders on empty canvas (no visibility filter).
	rr := c.RenderRaw(120, 40, -1, -1, -1, nil, -1, -1, 0, 0, -1, -1, false, -1, -1, 0, "", -1, -1, -1, -1, -1, -1, false, -1, -1)
	if rr.ColorMap[30][32] != 2 {
		t.Fatalf("expected painted cell to render color 2, got %d", rr.ColorMap[30][32])
	}
	m.undo()
	if m.getCanvas().GetHighlight(32, 30) != -1 {
		t.Fatal("expected undo to clear the painted stroke")
	}
}
