package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// newTestModel builds a normal-mode model with a known canvas for mouse tests.
func newTestModel() model {
	m := initialModel()
	m.mode = ModeNormal
	m.width = 120
	m.height = 40
	// Single buffer -> no buffer bar, so mouse Y maps directly to canvas Y.
	buf := m.getCurrentBuffer()
	buf.panX, buf.panY = 0, 0
	c := m.getCanvas()
	c.AddBox(5, 3, "Alpha")  // box 0 spans roughly (5,3)-(5+w,3+h)
	c.AddBox(40, 20, "Beta") // box 1
	return m
}

func press(button tea.MouseButton, x, y int) tea.MouseMsg {
	return tea.MouseMsg{X: x, Y: y, Action: tea.MouseActionPress, Button: button}
}

func release(x, y int) tea.MouseMsg {
	return tea.MouseMsg{X: x, Y: y, Action: tea.MouseActionRelease, Button: tea.MouseButtonNone}
}

func motion(x, y int) tea.MouseMsg {
	return tea.MouseMsg{X: x, Y: y, Action: tea.MouseActionMotion, Button: tea.MouseButtonNone}
}

// dragMotion is a motion event with the left button held (a drag).
func dragMotion(x, y int) tea.MouseMsg {
	return tea.MouseMsg{X: x, Y: y, Action: tea.MouseActionMotion, Button: tea.MouseButtonLeft}
}

// click performs a full press+release at the same spot.
func click(m model, button tea.MouseButton, x, y int) model {
	out, _ := m.Update(press(button, x, y))
	m = out.(model)
	out, _ = m.Update(release(x, y))
	return out.(model)
}

func TestLeftClickSelectsBox(t *testing.T) {
	m := newTestModel()
	// Click inside box 0.
	m = click(m, tea.MouseButtonLeft, 6, 4)
	if m.selBox != 0 {
		t.Fatalf("expected selBox 0, got selBox=%d selText=%d selConn=%d", m.selBox, m.selText, m.selConn)
	}
	// Click empty space clears the selection.
	m = click(m, tea.MouseButtonLeft, 100, 35)
	if m.selBox != -1 || m.selText != -1 || m.selConn != -1 {
		t.Fatalf("expected selection cleared, got selBox=%d selText=%d selConn=%d", m.selBox, m.selText, m.selConn)
	}
}

func TestDragMovesBoxAndReroutesConnection(t *testing.T) {
	m := newTestModel()
	c := m.getCanvas()
	// Connect box 0 -> box 1 so we can verify the line follows the move.
	fx, fy, tx, ty := c.CalculateConnectionPoints(0, 1)
	c.AddConnectionWithWaypoints(0, 1, fx, fy, tx, ty, nil)
	origConn := c.Connections()[0]
	box0StartX, box0StartY := c.Boxes()[0].X, c.Boxes()[0].Y

	// Press on box 0, drag right+down by 10, release.
	out, _ := m.Update(press(tea.MouseButtonLeft, 6, 4))
	m = out.(model)
	if !m.draggingBox || m.dragBoxID != 0 {
		t.Fatalf("expected drag of box 0, got dragging=%v id=%d", m.draggingBox, m.dragBoxID)
	}
	out, _ = m.Update(dragMotion(16, 14))
	m = out.(model)
	out, _ = m.Update(release(16, 14))
	m = out.(model)

	if m.draggingBox {
		t.Fatal("expected drag to end on release")
	}
	c = m.getCanvas()
	gotDX := c.Boxes()[0].X - box0StartX
	gotDY := c.Boxes()[0].Y - box0StartY
	if gotDX != 10 || gotDY != 10 {
		t.Fatalf("expected box moved by (10,10), got (%d,%d)", gotDX, gotDY)
	}
	// The connection's box-0 endpoint should have moved with the box.
	if c.Connections()[0].FromX == origConn.FromX && c.Connections()[0].FromY == origConn.FromY {
		t.Fatalf("expected connection endpoint to re-route after move (was %d,%d)", origConn.FromX, origConn.FromY)
	}
	// The move should be undoable.
	m.undo()
	c = m.getCanvas()
	if c.Boxes()[0].X != box0StartX || c.Boxes()[0].Y != box0StartY {
		t.Fatalf("expected undo to restore box to (%d,%d), got (%d,%d)", box0StartX, box0StartY, c.Boxes()[0].X, c.Boxes()[0].Y)
	}
}

// TestMoveBoxTranslatesAnchorWithBox verifies that when a box moves, an
// attached connection's anchor stays on the same edge and travels with the box
// (instead of sliding along the edge / flipping, which mangled the routing).
func TestMoveBoxTranslatesAnchorWithBox(t *testing.T) {
	m := newTestModel()
	c := m.getCanvas()
	// box 0 (Alpha) is up-left of box 1 (Beta), so the connection leaves box 0
	// from its right edge and enters box 1 from its left edge.
	fx, fy, tx, ty := c.CalculateConnectionPoints(0, 1)
	c.AddConnectionWithWaypoints(0, 1, fx, fy, tx, ty, nil)

	fromEdgeBefore := c.GetConnectionEdge(c.Boxes()[0], c.Connections()[0].FromX, c.Connections()[0].FromY)
	offsetBefore := c.Connections()[0].FromY - c.Boxes()[0].Y // anchor's offset down the edge

	// Move box 0 straight down by 8. box 0 is still left of box 1, so its right
	// edge still faces box 1: the anchor should just ride down with the box.
	c.MoveBox(0, 0, 8)

	conn := c.Connections()[0]
	fromEdgeAfter := c.GetConnectionEdge(c.Boxes()[0], conn.FromX, conn.FromY)
	offsetAfter := conn.FromY - c.Boxes()[0].Y

	if fromEdgeAfter != fromEdgeBefore {
		t.Fatalf("anchor edge flipped: before=%q after=%q", fromEdgeBefore, fromEdgeAfter)
	}
	if offsetAfter != offsetBefore {
		t.Fatalf("anchor slid along the edge: offset before=%d after=%d (should stay constant as the box moves)", offsetBefore, offsetAfter)
	}
	// The far (box 1) anchor should not have moved.
	if conn.ToX != tx || conn.ToY != ty {
		t.Fatalf("far anchor moved: was (%d,%d) now (%d,%d)", tx, ty, conn.ToX, conn.ToY)
	}
}

// TestMoveBoxPastTargetReanchors verifies that when a box is dragged so its
// anchored edge no longer faces the far endpoint, the anchor is re-picked to a
// sensible edge rather than translated behind the box.
func TestMoveBoxPastTargetReanchors(t *testing.T) {
	m := newTestModel()
	c := m.getCanvas()
	fx, fy, tx, ty := c.CalculateConnectionPoints(0, 1)
	c.AddConnectionWithWaypoints(0, 1, fx, fy, tx, ty, nil)
	// Sanity: connection starts on box 0's right edge.
	if e := c.GetConnectionEdge(c.Boxes()[0], c.Connections()[0].FromX, c.Connections()[0].FromY); e != "right" {
		t.Fatalf("expected initial anchor on right edge, got %q", e)
	}

	// Drag box 0 far to the right, well past box 1. The right edge no longer
	// faces box 1, so the anchor should move to the left edge.
	c.MoveBox(0, 120, 0)

	conn := c.Connections()[0]
	if e := c.GetConnectionEdge(c.Boxes()[0], conn.FromX, conn.FromY); e != "left" {
		t.Fatalf("expected anchor re-picked to left edge after crossing target, got %q", e)
	}
}

// TestMoveBoxKeepsSharedTrunkAndBranches reproduces the kiosk-flow case: a
// connection (Payment Made -> Cash) is routed through a vertical "trunk" at
// x=42, and other lines branch off that trunk. Moving Cash must keep the trunk
// in place so the branch points stay attached, rather than snapping the whole
// line to a fresh midpoint elbow.
func TestMoveBoxKeepsSharedTrunkAndBranches(t *testing.T) {
	m := newTestModel()
	c := m.getCanvas()
	c.Reset()

	// Payment Made (box 0): X=10,Y=20,W=16,H=3  -> right edge at x=25
	c.AddBox(10, 20, "Payment Made")
	c.Boxes()[0].Width, c.Boxes()[0].Height = 16, 3
	// Cash (box 1): X=47,Y=10,W=8,H=3 -> left edge at x=47
	c.AddBox(47, 10, "Cash")
	c.Boxes()[1].Width, c.Boxes()[1].Height = 8, 3
	// Gift Card (box 2): X=47,Y=14,W=13,H=3 -> left edge at x=47
	c.AddBox(47, 14, "Gift Card")
	c.Boxes()[2].Width, c.Boxes()[2].Height = 13, 3

	// Trunk: Payment Made right edge (25,21) -> Cash left edge (47,11),
	// routed up the x=42 trunk via waypoints.
	c.AddConnectionWithWaypoints(0, 1, 25, 21, 47, 11, []point{{X: 42, Y: 21}, {X: 42, Y: 11}})
	// Branch: a point on the trunk (42,15) -> Gift Card left edge (47,15).
	c.AddConnectionWithWaypoints(-1, 2, 42, 15, 47, 15, nil)

	// Move Cash straight up by 6.
	c.MoveBox(1, 0, -6)

	trunk := c.Connections()[0]
	// Every trunk waypoint must remain on the x=42 vertical.
	for _, wp := range trunk.Waypoints {
		if wp.X != 42 {
			t.Fatalf("trunk drifted off x=42: waypoints=%v", trunk.Waypoints)
		}
	}
	// The Cash anchor should have followed the box up (left edge, y 11->5).
	if trunk.ToX != 47 || trunk.ToY != 5 {
		t.Fatalf("expected Cash anchor at (47,5), got (%d,%d)", trunk.ToX, trunk.ToY)
	}
	// Payment Made's anchor must not have moved.
	if trunk.FromX != 25 || trunk.FromY != 21 {
		t.Fatalf("far anchor moved: got (%d,%d)", trunk.FromX, trunk.FromY)
	}

	// The Gift Card branch point must still sit on the trunk's new path.
	branch := c.Connections()[1]
	path := []point{{X: trunk.FromX, Y: trunk.FromY}}
	path = append(path, trunk.Waypoints...)
	path = append(path, point{X: trunk.ToX, Y: trunk.ToY})
	if !c.PointWasOnPath(branch.FromX, branch.FromY, path) {
		t.Fatalf("branch point (%d,%d) no longer on trunk path %v", branch.FromX, branch.FromY, path)
	}
}

// TestMoveBoxFacesAdjacentWaypointNotFarEnd reproduces the loop-back case: a
// connection leaves its far end on the right but loops around and enters the
// target box from the LEFT via its waypoints. Moving the box must keep the
// anchor on the left edge (facing the adjacent waypoint), not flip it to the
// right edge (facing the far endpoint).
func TestMoveBoxFacesAdjacentWaypointNotFarEnd(t *testing.T) {
	m := newTestModel()
	c := m.getCanvas()
	c.Reset()

	// Prompt (box 0): X=10,Y=14,W=22,H=3 -> left edge x=10, right edge x=31.
	c.AddBox(10, 14, "Prompt for Payment")
	c.Boxes()[0].Width, c.Boxes()[0].Height = 22, 3

	// A loop-back line whose far end (111,33) is far to the RIGHT, but which
	// routes down, left to x=6, up, and into Prompt's LEFT edge at (10,15).
	c.AddConnectionWithWaypoints(-1, 0, 111, 33, 10, 15,
		[]point{{X: 111, Y: 35}, {X: 6, Y: 35}, {X: 6, Y: 15}})

	c.MoveBox(0, 0, 3) // drag Prompt down 3

	conn := c.Connections()[0]
	if e := c.GetConnectionEdge(c.Boxes()[0], conn.ToX, conn.ToY); e != "left" {
		t.Fatalf("anchor flipped to %q edge; expected it to stay on the left (facing its waypoint)", e)
	}
	if conn.ToX != 10 || conn.ToY != 18 {
		t.Fatalf("expected left-edge anchor to ride down to (10,18), got (%d,%d)", conn.ToX, conn.ToY)
	}
	// The loop's vertical return leg at x=6 must be preserved.
	last := conn.Waypoints[len(conn.Waypoints)-1]
	if last.X != 6 {
		t.Fatalf("loop return leg drifted off x=6: waypoints=%v", conn.Waypoints)
	}
}

// TestMoveBoxRoundTripLeavesStraightLine checks that dragging a box away and
// back to its original spot restores a clean straight connection with no
// leftover (collinear/duplicate) waypoints, which would render as a stray
// junction character on the line.
func TestMoveBoxRoundTripLeavesStraightLine(t *testing.T) {
	m := newTestModel()
	c := m.getCanvas()
	c.Reset()

	// Prompt (box 0) directly above Payment Made (box 1), joined by a straight
	// vertical line at x=18.
	c.AddBox(10, 14, "Prompt for Payment")
	c.Boxes()[0].Width, c.Boxes()[0].Height = 22, 3
	c.AddBox(10, 20, "Payment Made")
	c.Boxes()[1].Width, c.Boxes()[1].Height = 16, 3
	c.AddConnectionWithWaypoints(0, 1, 18, 16, 18, 20, nil)

	// Drag Payment Made left 5 and back 5 (returning to its exact origin).
	for i := 0; i < 5; i++ {
		c.MoveBox(1, -1, 0)
	}
	for i := 0; i < 5; i++ {
		c.MoveBox(1, 1, 0)
	}

	conn := c.Connections()[0]
	if conn.FromX != 18 || conn.ToX != 18 || conn.FromY != 16 || conn.ToY != 20 {
		t.Fatalf("endpoints not restored: From=(%d,%d) To=(%d,%d)", conn.FromX, conn.FromY, conn.ToX, conn.ToY)
	}
	if len(conn.Waypoints) != 0 {
		t.Fatalf("straight line should have no waypoints, got %v", conn.Waypoints)
	}
}

// TestDragAroundAndBackRestoresAllLines verifies a drag is idempotent: dragging
// a box in a big loop and back to its exact starting point must leave every
// connection — the moved box's own lines, its branches, and unrelated lines the
// sweep passes over — exactly as they started, with no accumulated corruption.
func TestDragAroundAndBackRestoresAllLines(t *testing.T) {
	m := newTestModel()
	c := m.getCanvas()
	c.Reset()

	// A trunk with branches, plus an unrelated line elsewhere on the canvas.
	c.AddBox(10, 20, "Payment Made") // box 0, right edge x=25
	c.Boxes()[0].Width, c.Boxes()[0].Height = 16, 3
	c.AddBox(47, 10, "Cash") // box 1, left edge x=47
	c.Boxes()[1].Width, c.Boxes()[1].Height = 8, 3
	c.AddBox(47, 14, "Gift Card") // box 2
	c.Boxes()[2].Width, c.Boxes()[2].Height = 13, 3
	c.AddBox(47, 40, "Far Box") // box 3, an unrelated box far away
	c.Boxes()[3].Width, c.Boxes()[3].Height = 10, 3

	// Trunk box0 -> box1 routed up the x=42 vertical.
	c.AddConnectionWithWaypoints(0, 1, 25, 21, 47, 11, []point{{X: 42, Y: 21}, {X: 42, Y: 11}})
	// Branch off the trunk to Gift Card.
	c.AddConnectionWithWaypoints(-1, 2, 42, 15, 47, 15, nil)
	// An unrelated line the sweep may pass over, between two fixed points.
	c.AddConnectionWithWaypoints(-1, -1, 30, 42, 60, 42, nil)

	before := c.SnapshotConnections()

	box := c.Boxes()[0]
	m.beginBoxDrag(0, box.X+1, box.Y+1)
	gx, gy := box.X+1, box.Y+1
	loop := [][2]int{
		{gx + 45, gy}, {gx + 45, gy + 25}, {gx, gy + 25}, {gx, gy}, // back to start
	}
	for _, p := range loop {
		m.dragMoveTo(p[0], p[1])
	}
	m.finishBoxDrag()

	for i := range c.Connections() {
		b, a := before[i], c.Connections()[i]
		if b.FromX != a.FromX || b.FromY != a.FromY || b.ToX != a.ToX || b.ToY != a.ToY {
			t.Fatalf("conn[%d] %d->%d not restored: before From=(%d,%d) To=(%d,%d), after From=(%d,%d) To=(%d,%d)",
				i, a.FromID, a.ToID, b.FromX, b.FromY, b.ToX, b.ToY, a.FromX, a.FromY, a.ToX, a.ToY)
		}
		if len(b.Waypoints) != len(a.Waypoints) {
			t.Fatalf("conn[%d] waypoints not restored: before %v after %v", i, b.Waypoints, a.Waypoints)
		}
		for j := range b.Waypoints {
			if b.Waypoints[j] != a.Waypoints[j] {
				t.Fatalf("conn[%d] waypoint %d not restored: before %v after %v", i, j, b.Waypoints, a.Waypoints)
			}
		}
	}
}

// TestMoveBoxKeepsArrowAndUnrelatedLine covers two coupled regressions: moving a
// box must not (a) drop the arrowhead on a line entering it, nor (b) corrupt an
// unrelated line that merely shares a corner/junction with one of the moved
// box's connections.
func TestMoveBoxKeepsArrowAndUnrelatedLine(t *testing.T) {
	m := newTestModel()
	c := m.getCanvas()
	c.Reset()

	// Record (box 0): left edge x=20; Writer (box 1) below-right of it.
	c.AddBox(20, 10, "Record")
	c.Boxes()[0].Width, c.Boxes()[0].Height = 12, 3
	c.AddBox(40, 20, "Writer")
	c.Boxes()[1].Width, c.Boxes()[1].Height = 10, 3
	c.AddBox(5, 4, "Prompt") // box 2, target of the unrelated loop-back line
	c.Boxes()[2].Width, c.Boxes()[2].Height = 10, 3

	// conn0: an incoming arrow into Record's left edge (Success -> Record "Yes").
	c.AddConnectionWithWaypoints(-1, 0, 5, 15, 20, 12, []point{{X: 12, Y: 15}, {X: 12, Y: 12}})
	c.Connections()[0].ArrowTo = true
	// conn1: Record -> Writer, turning a corner at (44,12).
	c.AddConnectionWithWaypoints(0, 1, 31, 12, 44, 20, []point{{X: 44, Y: 12}})
	c.Connections()[1].ArrowTo = true
	// conn2: an UNRELATED line that starts exactly at conn1's corner (44,12) and
	// loops up-and-over to Prompt. It must not be dragged along when Record moves.
	c.AddConnectionWithWaypoints(-1, 2, 44, 12, 15, 6, []point{{X: 44, Y: 2}, {X: 15, Y: 2}})

	unrelatedBefore := c.Connections()[2]

	// Move Record down; conn1's corner follows, but conn2 must keep its shape.
	c.MoveBox(0, 0, 3)

	// conn0's incoming arrow must still be present at Record's (new) left edge.
	rr := c.RenderRaw(120, 40, -1, -1, -1, nil, -1, -1, 0, 0, -1, -1, false, -1, -1, 0, "", -1, -1, -1, -1, -1, -1, false, -1, -1)
	to := c.Connections()[0]
	arrow := false
	for yy := to.ToY - 1; yy <= to.ToY+1; yy++ {
		for xx := to.ToX - 2; xx <= to.ToX+1; xx++ {
			if yy >= 0 && yy < len(rr.Canvas) && xx >= 0 && xx < len(rr.Canvas[yy]) {
				if r := rr.Canvas[yy][xx]; r == '▶' || r == '◀' || r == '▲' || r == '▼' {
					arrow = true
				}
			}
		}
	}
	if !arrow {
		t.Fatalf("incoming arrow into Record was lost after the move")
	}

	// conn2's routing (its up-and-over waypoints) must be preserved.
	after := c.Connections()[2]
	if len(after.Waypoints) != len(unrelatedBefore.Waypoints) {
		t.Fatalf("unrelated line reshaped: before %v after %v", unrelatedBefore.Waypoints, after.Waypoints)
	}
	for i := range unrelatedBefore.Waypoints {
		if after.Waypoints[i] != unrelatedBefore.Waypoints[i] {
			t.Fatalf("unrelated line waypoint %d changed: before %v after %v", i, unrelatedBefore.Waypoints, after.Waypoints)
		}
	}
}

// TestMoveBoxFlipKeepsAnchorOffset checks that when a box moves so a line flips
// to the opposite edge, the anchor keeps its offset along the edge (a mid-edge
// anchor stays mid-edge) instead of being clamped to the neighbouring point.
func TestMoveBoxFlipKeepsAnchorOffset(t *testing.T) {
	m := newTestModel()
	c := m.getCanvas()
	c.Reset()
	c.AddBox(40, 5, "Box") // left x=40, right x=47, rows 5..7, middle y=6
	c.Boxes()[0].Width, c.Boxes()[0].Height = 8, 3
	// Enters the LEFT edge middle; its path bends far below (y=20).
	c.AddConnectionWithWaypoints(-1, 0, 10, 20, 40, 6, []point{{X: 25, Y: 20}, {X: 25, Y: 20}})

	c.MoveBox(0, -35, 0) // slide the box left past the line, flipping it to the right edge

	conn := c.Connections()[0]
	midY := c.Boxes()[0].Y + c.Boxes()[0].Height/2
	if e := c.GetConnectionEdge(c.Boxes()[0], conn.ToX, conn.ToY); e != "right" {
		t.Fatalf("expected anchor to flip to the right edge, got %q", e)
	}
	if conn.ToY != midY {
		t.Fatalf("expected anchor to stay on the middle row %d, got %d", midY, conn.ToY)
	}
}

func TestClickWithoutMoveDoesNotRecordUndo(t *testing.T) {
	m := newTestModel()
	undosBefore := len(m.getCurrentBuffer().undoStack)
	m = click(m, tea.MouseButtonLeft, 6, 4)
	if m.selBox != 0 {
		t.Fatalf("expected box selected, got selBox=%d", m.selBox)
	}
	if len(m.getCurrentBuffer().undoStack) != undosBefore {
		t.Fatal("a click without movement should not record an undo action")
	}
}

func TestMenuHoverHighlights(t *testing.T) {
	m := newTestModel()
	out, _ := m.Update(press(tea.MouseButtonRight, 6, 4))
	m = out.(model)
	x, y, _, _ := m.menuBounds()
	// Box menu: Edit Box(0), Edit Title(1), Border(2), New Line(3),
	// Delete Box(4), separator(5), New Box(6), New Text(7).
	// Hover "Delete Box" (index 4) without clicking.
	out, _ = m.Update(motion(x+2, y+1+4))
	m = out.(model)
	if m.menuIndex != 4 {
		t.Fatalf("expected hover to highlight item 4, got menuIndex=%d", m.menuIndex)
	}
	// Hover the separator row (index 5): selection should not move onto it.
	out, _ = m.Update(motion(x+2, y+1+5))
	m = out.(model)
	if m.menuIndex == 5 {
		t.Fatal("hover should not select a separator row")
	}
}

func TestRightClickOnBoxOpensMenu(t *testing.T) {
	m := newTestModel()
	out, _ := m.Update(press(tea.MouseButtonRight, 6, 4))
	m = out.(model)
	if m.mode != ModeContextMenu {
		t.Fatalf("expected ModeContextMenu, got %v", m.mode)
	}
	if m.menuTargetBox != 0 {
		t.Fatalf("expected menuTargetBox 0, got %d", m.menuTargetBox)
	}
	// Box menu should offer Edit Box, New Line, Delete Box.
	labels := map[string]bool{}
	for _, it := range m.menuItems {
		labels[it.Label] = true
	}
	for _, want := range []string{"Edit Box", "New Line", "Delete Box", "New Box", "New Text"} {
		if !labels[want] {
			t.Fatalf("box menu missing %q; items=%v", want, m.menuItems)
		}
	}
}

func TestRightClickEmptyMenu(t *testing.T) {
	m := newTestModel()
	out, _ := m.Update(press(tea.MouseButtonRight, 100, 35))
	m = out.(model)
	if m.mode != ModeContextMenu {
		t.Fatalf("expected ModeContextMenu, got %v", m.mode)
	}
	if len(m.menuItems) != 2 {
		t.Fatalf("expected 2 items on empty menu, got %v", m.menuItems)
	}
}

func TestNewLineDrawAndConnect(t *testing.T) {
	m := newTestModel()
	// Right-click box 0, pick "New Line".
	out, _ := m.Update(press(tea.MouseButtonRight, 6, 4))
	m = out.(model)
	newLineIdx := -1
	for i, it := range m.menuItems {
		if it.Label == "New Line" {
			newLineIdx = i
		}
	}
	if newLineIdx == -1 {
		t.Fatal("no New Line item")
	}
	// Click the New Line menu row.
	x, y, _, _ := m.menuBounds()
	out, _ = m.Update(press(tea.MouseButtonLeft, x+2, y+1+newLineIdx))
	m = out.(model)
	if !m.mouseLineDrawing || m.connectionFrom != 0 {
		t.Fatalf("expected line drawing from box 0, got drawing=%v from=%d mode=%v", m.mouseLineDrawing, m.connectionFrom, m.mode)
	}

	// Move the mouse toward box 1 (preview should track cursor).
	out, _ = m.Update(motion(41, 21))
	m = out.(model)
	if m.cursorX != 41 || m.cursorY != 21 {
		t.Fatalf("expected cursor to follow mouse, got (%d,%d)", m.cursorX, m.cursorY)
	}

	connsBefore := len(m.getCanvas().Connections())
	// Left-click on box 1 to complete the connection.
	out, _ = m.Update(press(tea.MouseButtonLeft, 41, 21))
	m = out.(model)
	if m.mouseLineDrawing {
		t.Fatal("expected line drawing to finish")
	}
	connsAfter := len(m.getCanvas().Connections())
	if connsAfter != connsBefore+1 {
		t.Fatalf("expected a new connection, before=%d after=%d", connsBefore, connsAfter)
	}
	conn := m.getCanvas().Connections()[connsAfter-1]
	if conn.FromID != 0 || conn.ToID != 1 {
		t.Fatalf("expected connection 0->1, got %d->%d", conn.FromID, conn.ToID)
	}
}

func TestMenuDeleteBox(t *testing.T) {
	m := newTestModel()
	boxesBefore := len(m.getCanvas().Boxes())
	out, _ := m.Update(press(tea.MouseButtonRight, 6, 4))
	m = out.(model)
	delIdx := -1
	for i, it := range m.menuItems {
		if it.Label == "Delete Box" {
			delIdx = i
		}
	}
	x, y, _, _ := m.menuBounds()
	out, _ = m.Update(press(tea.MouseButtonLeft, x+2, y+1+delIdx))
	m = out.(model)
	if len(m.getCanvas().Boxes()) != boxesBefore-1 {
		t.Fatalf("expected box deleted, before=%d after=%d", boxesBefore, len(m.getCanvas().Boxes()))
	}
	if m.mode != ModeNormal {
		t.Fatalf("expected ModeNormal after delete, got %v", m.mode)
	}
}

func TestMenuKeyboardNavSkipsSeparator(t *testing.T) {
	m := newTestModel()
	out, _ := m.Update(press(tea.MouseButtonRight, 6, 4))
	m = out.(model)
	// items: Edit Box(0), Edit Title(1), Border(2), New Line(3), Delete Box(4),
	// separator(5), New Box(6), New Text(7)
	m.menuIndex = 4
	m.menuMoveSelection(1) // should skip the separator to New Box (6)
	if m.menuItems[m.menuIndex].Separator {
		t.Fatal("landed on separator")
	}
	if m.menuItems[m.menuIndex].Label != "New Box" {
		t.Fatalf("expected New Box, got %q (idx %d)", m.menuItems[m.menuIndex].Label, m.menuIndex)
	}
}
