package main

import (
	tea "github.com/charmbracelet/bubbletea"
)

// sign returns -1, 0, or 1 depending on the sign of n.
func sign(n int) int {
	switch {
	case n > 0:
		return 1
	case n < 0:
		return -1
	default:
		return 0
	}
}

// bufferBarOffset returns the number of screen rows occupied above the canvas
// (the buffer bar, when more than one buffer is open). Mouse Y coordinates are
// reported relative to the terminal's top-left, so we subtract this to get the
// canvas row that matches m.cursorY.
func (m *model) bufferBarOffset() int {
	if m.mode != ModeStartup && len(m.buffers) > 1 {
		return 1
	}
	return 0
}

// handleMouse is the entry point for all mouse events. It only acts in the
// modes where mouse interaction makes sense; everything else is ignored so the
// keyboard-driven flows stay untouched.
func (m model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.mode {
	case ModeContextMenu:
		cmd = m.handleMenuMouse(msg)
	case ModeNormal:
		cmd = m.handleNormalMouse(msg)
	case ModeMultiSelect:
		cmd = m.handleMultiSelectMouse(msg)
	case ModeMove:
		cmd = m.handleMoveMouse(msg)
	}
	return m, cmd
}

// handleMoveMouse lets the selected group be dragged with the mouse: press
// grabs, motion moves the whole group live, release commits the move (undoable)
// and returns to normal mode.
func (m *model) handleMoveMouse(msg tea.MouseMsg) tea.Cmd {
	canvasX := msg.X
	canvasY := msg.Y - m.bufferBarOffset()
	if canvasY < 0 {
		canvasY = 0
	}
	switch {
	case msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft:
		m.draggingGroup = true
		m.groupLastX, m.groupLastY = canvasX, canvasY
	case msg.Action == tea.MouseActionMotion && m.draggingGroup:
		if dx, dy := canvasX-m.groupLastX, canvasY-m.groupLastY; dx != 0 || dy != 0 {
			m.handleMultiSelectMove(dx, dy)
			m.groupLastX, m.groupLastY = canvasX, canvasY
		}
	case msg.Action == tea.MouseActionRelease:
		m.draggingGroup = false
		m.commitMove()
	}
	return nil
}

// handleMultiSelectMouse lets the user drag out the selection rectangle: press
// sets the corner, motion drags the opposite corner (the cursor), release
// finalizes. Right-click cancels back to normal mode.
func (m *model) handleMultiSelectMouse(msg tea.MouseMsg) tea.Cmd {
	canvasX := msg.X
	canvasY := msg.Y - m.bufferBarOffset()
	if canvasY < 0 {
		canvasY = 0
	}
	panX, panY := m.getPanOffset()
	switch {
	case msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft:
		m.selectionStartX, m.selectionStartY = canvasX+panX, canvasY+panY
		m.cursorX, m.cursorY = canvasX, canvasY
		m.ensureCursorInBounds()
	case msg.Action == tea.MouseActionMotion && msg.Button == tea.MouseButtonLeft:
		m.cursorX, m.cursorY = canvasX, canvasY
		m.ensureCursorInBounds()
	case msg.Action == tea.MouseActionRelease:
		m.cursorX, m.cursorY = canvasX, canvasY
		m.ensureCursorInBounds()
		m.finalizeMultiSelect(canvasX+panX, canvasY+panY)
	case msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonRight:
		m.mode = ModeNormal
		m.selectionStartX, m.selectionStartY = -1, -1
		m.selectedBoxes, m.selectedTexts = []int{}, []int{}
	}
	return nil
}

// handleNormalMouse handles clicks, motion, and the scroll wheel while in the
// normal editing mode.
func (m *model) handleNormalMouse(msg tea.MouseMsg) tea.Cmd {
	// Scroll wheel pans the canvas.
	if tea.MouseEvent(msg).IsWheel() {
		if buf := m.getCurrentBuffer(); buf != nil {
			switch msg.Button {
			case tea.MouseButtonWheelUp:
				buf.panY -= 2
			case tea.MouseButtonWheelDown:
				buf.panY += 2
			case tea.MouseButtonWheelLeft:
				buf.panX -= 2
			case tea.MouseButtonWheelRight:
				buf.panX += 2
			}
		}
		return nil
	}

	canvasX := msg.X
	canvasY := msg.Y - m.bufferBarOffset()
	if canvasY < 0 {
		canvasY = 0
	}
	panX, panY := m.getPanOffset()

	// A box drag is in progress: motion moves the box, release finalizes it.
	// Any other event (e.g. a stray press) finalizes the drag first, then is
	// processed normally below.
	if m.draggingBox {
		switch msg.Action {
		case tea.MouseActionMotion:
			m.dragMoveTo(canvasX, canvasY)
			return nil
		case tea.MouseActionRelease:
			m.finishBoxDrag()
			return nil
		default:
			m.finishBoxDrag()
		}
	}

	// A text drag is in progress: motion moves the text, release finalizes it.
	if m.draggingText {
		switch msg.Action {
		case tea.MouseActionMotion:
			m.dragTextMoveTo(canvasX, canvasY)
			return nil
		case tea.MouseActionRelease:
			m.finishTextDrag()
			return nil
		default:
			m.finishTextDrag()
		}
	}

	// Dragging an empty area pans the view; a release without movement instead
	// clears the selection (handled in the empty-press branch below).
	if m.panningView {
		switch msg.Action {
		case tea.MouseActionMotion:
			m.panViewTo(canvasX, canvasY)
			return nil
		case tea.MouseActionRelease:
			if !m.panMoved {
				m.selectAtMouse(canvasX+panX, canvasY+panY)
			}
			m.panningView = false
			return nil
		default:
			m.panningView = false
		}
	}

	// While drawing a line from the context menu, the preview follows the mouse
	// until the user clicks a target (box or line) or an empty cell (waypoint).
	if m.mouseLineDrawing && (m.connectionFrom != -1 || m.connectionFromLine != -1) {
		m.cursorX = canvasX
		m.cursorY = canvasY
		m.ensureCursorInBounds()
		if msg.Action == tea.MouseActionPress {
			switch msg.Button {
			case tea.MouseButtonLeft:
				m.completeMouseLine()
			case tea.MouseButtonRight:
				m.cancelMouseLine()
			}
		}
		return nil
	}

	worldX, worldY := canvasX+panX, canvasY+panY

	// In highlight mode, left click-drag paints cells in the selected color.
	// Right-click still opens the context menu (handled below).
	if m.highlightMode {
		switch {
		case msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft:
			m.beginHighlightPaint(worldX, worldY)
			return nil
		case msg.Action == tea.MouseActionMotion && m.paintingHighlight:
			m.paintHighlightTo(worldX, worldY)
			return nil
		case msg.Action == tea.MouseActionRelease && m.paintingHighlight:
			m.finishHighlightPaint()
			return nil
		}
	}

	if msg.Action != tea.MouseActionPress {
		return nil
	}

	switch msg.Button {
	case tea.MouseButtonLeft:
		// Keep the keyboard cursor in sync with where the user clicked.
		m.cursorX = canvasX
		m.cursorY = canvasY
		m.ensureCursorInBounds()
		// Pressing on a box (or text) begins a drag (and selects it); the drag
		// becomes a plain selection if released without moving. Boxes take
		// priority over text when they overlap.
		if canvas := m.getCanvas(); canvas != nil {
			if boxID := canvas.GetBoxAt(worldX, worldY); boxID != -1 {
				m.beginBoxDrag(boxID, worldX, worldY)
				return nil
			}
			if textID := canvas.GetTextAt(worldX, worldY); textID != -1 {
				m.beginTextDrag(textID, worldX, worldY)
				return nil
			}
		}
		// Empty area: begin a pan-drag. A release without movement clears the
		// selection instead (see the panningView block above).
		m.panningView = true
		m.panLastX, m.panLastY = canvasX, canvasY
		m.panMoved = false
	case tea.MouseButtonRight:
		m.openContextMenu(canvasX, canvasY)
	}
	return nil
}

// beginBoxDrag starts dragging the given box, recording the state needed to
// move it (and its connections) and to undo the move later.
func (m *model) beginBoxDrag(boxID, worldX, worldY int) {
	canvas := m.getCanvas()
	if canvas == nil || boxID < 0 || boxID >= len(canvas.boxes) {
		return
	}
	box := canvas.boxes[boxID]
	m.draggingBox = true
	m.dragBoxID = boxID
	m.dragGrabOffsetX = box.X - worldX
	m.dragGrabOffsetY = box.Y - worldY
	m.originalMoveX = box.X
	m.originalMoveY = box.Y

	// Capture original connection states so undo can restore rerouted lines.
	m.originalBoxConnections = make(map[int][]Connection)
	m.originalBoxConnections[boxID] = canvas.GetConnectionsForBox(boxID)

	// Snapshot ALL connections so each pointer move re-routes from the original
	// state rather than the previous step's result. This keeps the drag
	// idempotent: dragging around and back to the start restores the diagram
	// exactly, and a sweeping line can't progressively corrupt other lines.
	m.dragConnSnapshot = canvas.SnapshotConnections()

	// Capture highlights on the box so they travel with it (like keyboard move).
	m.originalHighlights = make(map[point]int)
	for y := box.Y; y < box.Y+box.Height; y++ {
		for x := box.X; x < box.X+box.Width; x++ {
			if color := canvas.GetHighlight(x, y); color != -1 {
				m.originalHighlights[point{X: x, Y: y}] = color
			}
		}
	}
	m.highlightMoveDelta = point{X: 0, Y: 0}

	// Reflect the drag as a selection.
	m.selBox = boxID
	m.selText = -1
	m.selConn = -1
}

// dragMoveTo moves the dragged box so the grabbed point tracks the pointer.
// Connections attached to the box are re-routed by MoveBox.
func (m *model) dragMoveTo(canvasX, canvasY int) {
	canvas := m.getCanvas()
	if canvas == nil || m.dragBoxID < 0 || m.dragBoxID >= len(canvas.boxes) {
		m.draggingBox = false
		return
	}
	panX, panY := m.getPanOffset()
	worldX, worldY := canvasX+panX, canvasY+panY

	desiredX := worldX + m.dragGrabOffsetX
	desiredY := worldY + m.dragGrabOffsetY

	// Re-derive routing from the drag-start snapshot each time: restore the
	// original connections, put the box back at its origin, then apply the total
	// move in one step. This makes the drag idempotent — no per-step snapping
	// error accumulates, so a box dragged around and back lands exactly where it
	// started with its lines intact.
	if m.dragConnSnapshot != nil {
		canvas.RestoreConnectionsSnapshot(m.dragConnSnapshot)
	}
	canvas.SetBoxPositionOnly(m.dragBoxID, m.originalMoveX, m.originalMoveY)
	canvas.MoveBox(m.dragBoxID, desiredX-m.originalMoveX, desiredY-m.originalMoveY)
	// Re-place highlights by the cumulative offset (idempotent): this also
	// resets them when the box has returned to its origin.
	if len(m.originalHighlights) > 0 {
		cumX := canvas.boxes[m.dragBoxID].X - m.originalMoveX
		cumY := canvas.boxes[m.dragBoxID].Y - m.originalMoveY
		m.highlightMoveDelta = m.moveHighlightsOnSelectedObjects(cumX, cumY)
	}

	m.cursorX = canvasX
	m.cursorY = canvasY
	m.ensureCursorInBounds()
}

// finishBoxDrag ends a drag, recording an undoable move if the box actually
// changed position.
func (m *model) finishBoxDrag() {
	canvas := m.getCanvas()
	if canvas != nil && m.dragBoxID >= 0 && m.dragBoxID < len(canvas.boxes) {
		cur := canvas.boxes[m.dragBoxID]
		deltaX := cur.X - m.originalMoveX
		deltaY := cur.Y - m.originalMoveY
		if deltaX != 0 || deltaY != 0 {
			moveData := MoveBoxData{ID: m.dragBoxID, DeltaX: deltaX, DeltaY: deltaY}
			var highlightCells []HighlightCell
			for origPos, color := range m.originalHighlights {
				highlightCells = append(highlightCells, HighlightCell{X: origPos.X, Y: origPos.Y, Color: color})
			}
			originalState := OriginalBoxState{
				ID:          m.dragBoxID,
				X:           m.originalMoveX,
				Y:           m.originalMoveY,
				Width:       cur.Width,
				Height:      cur.Height,
				Connections: m.originalBoxConnections[m.dragBoxID],
				Highlights:  highlightCells,
			}
			m.recordAction(ActionMoveBox, moveData, originalState)
		}
	}
	m.draggingBox = false
	m.originalHighlights = make(map[point]int)
	m.originalBoxConnections = make(map[int][]Connection)
	m.dragConnSnapshot = nil
	m.highlightMoveDelta = point{X: 0, Y: 0}
}

// beginTextDrag starts dragging the given text object. Text has no attached
// connections, so this is simpler than a box drag.
func (m *model) beginTextDrag(textID, worldX, worldY int) {
	canvas := m.getCanvas()
	if canvas == nil || textID < 0 || textID >= len(canvas.texts) {
		return
	}
	text := canvas.texts[textID]
	m.draggingText = true
	m.dragTextID = textID
	m.dragGrabOffsetX = text.X - worldX
	m.dragGrabOffsetY = text.Y - worldY
	m.originalTextMoveX = text.X
	m.originalTextMoveY = text.Y
	m.selText = textID
	m.selBox = -1
	m.selConn = -1
}

// dragTextMoveTo moves the dragged text so the grabbed point tracks the pointer.
func (m *model) dragTextMoveTo(canvasX, canvasY int) {
	canvas := m.getCanvas()
	if canvas == nil || m.dragTextID < 0 || m.dragTextID >= len(canvas.texts) {
		m.draggingText = false
		return
	}
	panX, panY := m.getPanOffset()
	worldX, worldY := canvasX+panX, canvasY+panY
	desiredX := worldX + m.dragGrabOffsetX
	desiredY := worldY + m.dragGrabOffsetY
	cur := canvas.texts[m.dragTextID]
	canvas.MoveText(m.dragTextID, desiredX-cur.X, desiredY-cur.Y)
	m.cursorX = canvasX
	m.cursorY = canvasY
	m.ensureCursorInBounds()
}

// finishTextDrag ends a text drag, recording an undoable move if it moved.
func (m *model) finishTextDrag() {
	canvas := m.getCanvas()
	if canvas != nil && m.dragTextID >= 0 && m.dragTextID < len(canvas.texts) {
		cur := canvas.texts[m.dragTextID]
		deltaX := cur.X - m.originalTextMoveX
		deltaY := cur.Y - m.originalTextMoveY
		if deltaX != 0 || deltaY != 0 {
			moveData := MoveTextData{ID: m.dragTextID, DeltaX: deltaX, DeltaY: deltaY}
			originalState := OriginalTextState{ID: m.dragTextID, X: m.originalTextMoveX, Y: m.originalTextMoveY}
			m.recordAction(ActionMoveText, moveData, originalState)
		}
	}
	m.draggingText = false
}

// panViewTo pans the view so the grabbed empty cell tracks the pointer.
func (m *model) panViewTo(canvasX, canvasY int) {
	buf := m.getCurrentBuffer()
	if buf == nil {
		return
	}
	buf.panX -= canvasX - m.panLastX
	buf.panY -= canvasY - m.panLastY
	m.panLastX, m.panLastY = canvasX, canvasY
	m.panMoved = true
}

// beginHighlightPaint starts a paint stroke at a cell.
func (m *model) beginHighlightPaint(worldX, worldY int) {
	if m.getCanvas() == nil {
		return
	}
	m.paintingHighlight = true
	m.paintedCells = nil
	m.paintedSeen = map[point]bool{}
	m.lastPaintX, m.lastPaintY = worldX, worldY
	m.paintHighlightCell(worldX, worldY)
}

// paintHighlightCell paints one cell in the selected color, once per stroke.
func (m *model) paintHighlightCell(x, y int) {
	p := point{x, y}
	if x < 0 || y < 0 || m.paintedSeen[p] {
		return
	}
	m.paintedSeen[p] = true
	old := m.getCanvas().GetHighlight(x, y)
	m.getCanvas().SetHighlight(x, y, m.selectedColor)
	m.paintedCells = append(m.paintedCells, HighlightCell{X: x, Y: y, Color: m.selectedColor, HadColor: old != -1, OldColor: old})
}

// paintHighlightTo paints along the line from the last painted cell to (x,y) so
// fast drags leave no gaps.
func (m *model) paintHighlightTo(worldX, worldY int) {
	x, y := m.lastPaintX, m.lastPaintY
	dx, dy := abs(worldX-x), -abs(worldY-y)
	sx, sy := sign(worldX-x), sign(worldY-y)
	err := dx + dy
	for {
		m.paintHighlightCell(x, y)
		if x == worldX && y == worldY {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x += sx
		}
		if e2 <= dx {
			err += dx
			y += sy
		}
	}
	m.lastPaintX, m.lastPaintY = worldX, worldY
}

// finishHighlightPaint records the stroke as one undoable action.
func (m *model) finishHighlightPaint() {
	if len(m.paintedCells) > 0 {
		inverse := make([]HighlightCell, len(m.paintedCells))
		for i, c := range m.paintedCells {
			inverse[i] = HighlightCell{X: c.X, Y: c.Y, Color: c.OldColor, HadColor: c.HadColor, OldColor: c.Color}
		}
		m.recordAction(ActionHighlight, HighlightData{Cells: m.paintedCells}, HighlightData{Cells: inverse})
	}
	m.paintingHighlight = false
	m.paintedCells = nil
	m.paintedSeen = nil
}

// selectAtMouse selects (persistently highlights) the box, text, or line under
// the given world coordinates. Clicking empty space clears any selection.
func (m *model) selectAtMouse(worldX, worldY int) {
	canvas := m.getCanvas()
	m.selBox, m.selText, m.selConn = -1, -1, -1
	if canvas == nil {
		return
	}
	if boxID := canvas.GetBoxAt(worldX, worldY); boxID != -1 {
		m.selBox = boxID
		return
	}
	if textID := canvas.GetTextAt(worldX, worldY); textID != -1 {
		m.selText = textID
		return
	}
	if connIdx, _, _ := canvas.findNearestPointOnConnection(worldX, worldY); connIdx != -1 {
		m.selConn = connIdx
	}
}

// cancelMouseLine aborts an in-progress mouse line draw.
func (m *model) cancelMouseLine() {
	m.mouseLineDrawing = false
	m.connectionFrom = -1
	m.connectionFromLine = -1
	m.connectionFromX = 0
	m.connectionFromY = 0
	m.connectionWaypoints = nil
}

// completeMouseLine finishes (or extends) a line being drawn with the mouse.
// Clicking a box or existing line creates the connection; clicking empty space
// drops a waypoint so the user can route the line around obstacles.
func (m *model) completeMouseLine() {
	canvas := m.getCanvas()
	if canvas == nil {
		return
	}
	panX, panY := m.getPanOffset()
	worldX, worldY := m.cursorX+panX, m.cursorY+panY

	boxID := canvas.GetBoxAt(worldX, worldY)
	lineConnIdx, lineX, lineY := canvas.findNearestPointOnConnection(worldX, worldY)

	if boxID != -1 {
		toBox := canvas.boxes[boxID]
		toX, toY := canvas.findNearestEdgePoint(toBox, worldX, worldY)
		connection := Connection{
			FromID:    m.connectionFrom,
			ToID:      boxID,
			FromX:     m.connectionFromX,
			FromY:     m.connectionFromY,
			ToX:       toX,
			ToY:       toY,
			Waypoints: m.connectionWaypoints,
			Color:     -1,
		}
		canvas.AddConnectionWithWaypoints(m.connectionFrom, boxID, m.connectionFromX, m.connectionFromY, toX, toY, m.connectionWaypoints)
		connData := AddConnectionData{FromID: m.connectionFrom, ToID: boxID, Connection: connection}
		m.recordAction(ActionAddConnection, connData, connData)
		m.cancelMouseLine()
		m.successMessage = ""
	} else if lineConnIdx != -1 {
		connection := Connection{
			FromID:    m.connectionFrom,
			ToID:      -1,
			FromX:     m.connectionFromX,
			FromY:     m.connectionFromY,
			ToX:       lineX,
			ToY:       lineY,
			Waypoints: m.connectionWaypoints,
			Color:     -1,
		}
		canvas.AddConnectionWithWaypoints(m.connectionFrom, -1, m.connectionFromX, m.connectionFromY, lineX, lineY, m.connectionWaypoints)
		connData := AddConnectionData{FromID: m.connectionFrom, ToID: -1, Connection: connection}
		m.recordAction(ActionAddConnection, connData, connData)
		m.cancelMouseLine()
		m.successMessage = ""
	} else {
		// Empty space: add a waypoint and keep drawing.
		m.connectionWaypoints = append(m.connectionWaypoints, point{worldX, worldY})
	}
}

// openContextMenu builds and opens the right-click menu at the given canvas
// (screen-relative) coordinates.
func (m *model) openContextMenu(canvasX, canvasY int) {
	canvas := m.getCanvas()
	if canvas == nil {
		return
	}
	panX, panY := m.getPanOffset()
	worldX, worldY := canvasX+panX, canvasY+panY

	m.menuWorldX, m.menuWorldY = worldX, worldY
	m.menuTargetBox, m.menuTargetText, m.menuTargetConn = -1, -1, -1

	m.menuTargetBox = canvas.GetBoxAt(worldX, worldY)
	if m.menuTargetBox == -1 {
		m.menuTargetText = canvas.GetTextAt(worldX, worldY)
		if m.menuTargetText == -1 {
			m.menuTargetConn, _, _ = canvas.findNearestPointOnConnection(worldX, worldY)
		}
	}

	m.menuItems = buildMenuItems(m.menuTargetBox, m.menuTargetText, m.menuTargetConn)
	m.menuIndex = firstSelectableMenuIndex(m.menuItems)
	m.menuStack = nil
	m.menuX = canvasX
	m.menuY = canvasY
	m.mode = ModeContextMenu
}

// colorSubmenu returns the palette color choices ("None" clears the color).
func colorSubmenu() []MenuItem {
	names := []string{"Gray", "Red", "Green", "Yellow", "Blue", "Magenta", "Cyan", "White"}
	items := []MenuItem{{Label: "None", Action: MenuSetColor, Arg: -1}}
	for i, n := range names {
		items = append(items, MenuItem{Label: n, Action: MenuSetColor, Arg: i})
	}
	return items
}

// borderStyleSubmenu returns the four border-style choices.
func borderStyleSubmenu() []MenuItem {
	return []MenuItem{
		{Label: "ASCII", Action: MenuSetBorderStyle, Arg: int(BorderStyleASCII)},
		{Label: "Single", Action: MenuSetBorderStyle, Arg: int(BorderStyleSingle)},
		{Label: "Double", Action: MenuSetBorderStyle, Arg: int(BorderStyleDouble)},
		{Label: "Rounded", Action: MenuSetBorderStyle, Arg: int(BorderStyleRounded)},
	}
}

// buildMenuItems assembles the context menu based on what was clicked.
func buildMenuItems(box, text, conn int) []MenuItem {
	var items []MenuItem
	switch {
	case box != -1:
		items = append(items,
			MenuItem{Label: "Edit Box", Action: MenuEditBox},
			MenuItem{Label: "Edit Title", Action: MenuEditTitle},
			MenuItem{Label: "Border", Action: MenuSubmenu, Submenu: []MenuItem{
				{Label: "Style", Action: MenuSubmenu, Submenu: borderStyleSubmenu()},
				{Label: "Color", Action: MenuSubmenu, Submenu: colorSubmenu()},
			}},
			MenuItem{Label: "New Line", Action: MenuNewLine},
			MenuItem{Label: "Delete Box", Action: MenuDeleteBox},
			MenuItem{Separator: true},
		)
	case text != -1:
		items = append(items,
			MenuItem{Label: "Edit Text", Action: MenuEditText},
			MenuItem{Label: "Color", Action: MenuSubmenu, Submenu: colorSubmenu()},
			MenuItem{Label: "Delete Text", Action: MenuDeleteText},
			MenuItem{Separator: true},
		)
	case conn != -1:
		items = append(items,
			MenuItem{Label: "New Line", Action: MenuNewLine},
			MenuItem{Label: "Color", Action: MenuSubmenu, Submenu: colorSubmenu()},
			MenuItem{Label: "Delete Line", Action: MenuDeleteLine},
			MenuItem{Separator: true},
		)
	}
	items = append(items,
		MenuItem{Label: "New Box", Action: MenuNewBox},
		MenuItem{Label: "New Text", Action: MenuNewText},
	)
	return items
}

func firstSelectableMenuIndex(items []MenuItem) int {
	for i, item := range items {
		if !item.Separator {
			return i
		}
	}
	return 0
}

// allMenuLevels returns every open menu level, root first, focused level last.
func (m *model) allMenuLevels() []menuLevel {
	levels := []menuLevel{{items: m.menuItems, index: m.menuIndex, x: m.menuX, y: m.menuY}}
	levels = append(levels, m.menuStack...)
	return levels
}

// focusedItems / focusedIndex / setFocusedIndex operate on the deepest open level.
func (m *model) focusedItems() []MenuItem {
	if len(m.menuStack) > 0 {
		return m.menuStack[len(m.menuStack)-1].items
	}
	return m.menuItems
}

func (m *model) focusedIndex() int {
	if len(m.menuStack) > 0 {
		return m.menuStack[len(m.menuStack)-1].index
	}
	return m.menuIndex
}

func (m *model) setFocusedIndex(idx int) {
	if len(m.menuStack) > 0 {
		m.menuStack[len(m.menuStack)-1].index = idx
		return
	}
	m.menuIndex = idx
}

// menuMoveSelection moves the highlighted item on the focused level by dir
// (+1/-1), skipping separators and wrapping around.
func (m *model) menuMoveSelection(dir int) {
	items := m.focusedItems()
	n := len(items)
	if n == 0 {
		return
	}
	idx := m.focusedIndex()
	for i := 0; i < n; i++ {
		idx = (idx + dir + n) % n
		if !items[idx].Separator {
			m.setFocusedIndex(idx)
			return
		}
	}
}

// menuDescend opens the focused item's submenu (if it has one), positioned to
// the right of its parent row.
func (m *model) menuDescend() {
	items := m.focusedItems()
	idx := m.focusedIndex()
	if idx < 0 || idx >= len(items) || len(items[idx].Submenu) == 0 {
		return
	}
	levels := m.allMenuLevels()
	px, py, pw, _ := m.levelBounds(levels[len(levels)-1])
	child := menuLevel{
		items: items[idx].Submenu,
		index: firstSelectableMenuIndex(items[idx].Submenu),
		x:     px + pw,
		y:     py + 1 + idx,
	}
	m.menuStack = append(m.menuStack, child)
}

// menuAscend closes the deepest submenu, or the whole menu at the root.
func (m *model) menuAscend() {
	if len(m.menuStack) > 0 {
		m.menuStack = m.menuStack[:len(m.menuStack)-1]
		return
	}
	m.closeContextMenu()
}

// closeContextMenu dismisses the context menu without taking an action.
func (m *model) closeContextMenu() {
	m.mode = ModeNormal
	m.menuItems = nil
	m.menuStack = nil
}

// handleMenuMouse handles hover and clicks while the context menu is open.
func (m *model) handleMenuMouse(msg tea.MouseMsg) tea.Cmd {
	canvasX := msg.X
	canvasY := msg.Y - m.bufferBarOffset()

	levelIdx, itemIdx, inside := m.menuHitTest(canvasX, canvasY)

	switch {
	case msg.Action == tea.MouseActionMotion:
		if inside && itemIdx >= 0 {
			m.focusMenuLevel(levelIdx, itemIdx)
		}
	case msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft:
		if inside && itemIdx >= 0 {
			m.focusMenuLevel(levelIdx, itemIdx)
			items := m.focusedItems()
			item := items[m.focusedIndex()]
			if len(item.Submenu) > 0 {
				m.menuDescend()
				return nil
			}
			return m.activateMenuItem(item.Action, item.Arg)
		}
		if !inside {
			m.closeContextMenu()
		}
	case msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonRight:
		m.closeContextMenu()
	}
	return nil
}

// focusMenuLevel makes level `levelIdx` the focused level (truncating deeper
// levels), selects `itemIdx` on it, and opens that item's submenu if it has one.
func (m *model) focusMenuLevel(levelIdx, itemIdx int) {
	if levelIdx < 0 {
		return
	}
	// levels: 0 = root, 1.. = menuStack[0..]. Keep levels 0..levelIdx.
	if levelIdx < len(m.menuStack)+1 {
		m.menuStack = m.menuStack[:levelIdx]
	}
	var items []MenuItem
	if levelIdx == 0 {
		items = m.menuItems
	} else {
		items = m.menuStack[levelIdx-1].items
	}
	if itemIdx < 0 || itemIdx >= len(items) || items[itemIdx].Separator {
		return
	}
	if levelIdx == 0 {
		m.menuIndex = itemIdx
	} else {
		m.menuStack[levelIdx-1].index = itemIdx
	}
	if len(items[itemIdx].Submenu) > 0 {
		m.menuDescend()
	}
}

// menuHitTest returns the (levelIdx, itemIdx, inside) at a canvas coordinate,
// checking the deepest open level first. itemIdx is -1 on a border/separator.
func (m *model) menuHitTest(canvasX, canvasY int) (int, int, bool) {
	levels := m.allMenuLevels()
	for li := len(levels) - 1; li >= 0; li-- {
		x, y, w, h := m.levelBounds(levels[li])
		if canvasX < x || canvasX >= x+w || canvasY < y || canvasY >= y+h {
			continue
		}
		itemRow := canvasY - (y + 1)
		if itemRow < 0 || itemRow >= len(levels[li].items) || levels[li].items[itemRow].Separator {
			return li, -1, true
		}
		return li, itemRow, true
	}
	return -1, -1, false
}

// levelBounds returns a menu level's rectangle (x, y, width, height) clamped to
// stay on screen.
func (m *model) levelBounds(level menuLevel) (int, int, int, int) {
	inner := menuInnerWidth(level.items)
	w := inner + 2
	h := len(level.items) + 2

	maxW := m.width
	maxH := m.height - 1 - m.bufferBarOffset()
	if maxH < 1 {
		maxH = 1
	}

	x := level.x
	y := level.y
	if x+w > maxW {
		x = maxW - w
	}
	if x < 0 {
		x = 0
	}
	if y+h > maxH {
		y = maxH - h
	}
	if y < 0 {
		y = 0
	}
	return x, y, w, h
}

// menuBounds returns the root menu's rectangle (used by tests and hit-testing).
func (m *model) menuBounds() (int, int, int, int) {
	return m.levelBounds(menuLevel{items: m.menuItems, index: m.menuIndex, x: m.menuX, y: m.menuY})
}

// menuInnerWidth returns the interior width of the menu (label column) in cells,
// reserving space for the submenu marker when any item has a submenu.
func menuInnerWidth(items []MenuItem) int {
	maxLabel := 0
	hasSubmenu := false
	for _, item := range items {
		if item.Separator {
			continue
		}
		if len(item.Label) > maxLabel {
			maxLabel = len(item.Label)
		}
		if len(item.Submenu) > 0 {
			hasSubmenu = true
		}
	}
	// One space of padding on each side of the label.
	inner := maxLabel + 2
	if hasSubmenu {
		inner += 2 // room for a trailing " ▸"
	}
	return inner
}

// activateMenuItem performs the action for the chosen (leaf) menu entry and
// closes the menu (unless the action enters another mode). arg carries an
// action-specific value (color index, border-style value).
func (m *model) activateMenuItem(action MenuAction, arg int) tea.Cmd {
	canvas := m.getCanvas()
	if canvas == nil {
		m.closeContextMenu()
		return nil
	}
	// Any leaf action dismisses the whole cascade.
	m.menuStack = nil

	switch action {
	case MenuSubmenu:
		// Not a leaf; nothing to do (submenu opening is handled before this).
		return nil
	case MenuNewBox:
		boxID := len(canvas.boxes)
		canvas.AddBox(m.menuWorldX, m.menuWorldY, "Box")
		addData := AddBoxData{X: m.menuWorldX, Y: m.menuWorldY, Text: "Box", ID: boxID}
		deleteData := DeleteBoxData{ID: boxID, Connections: nil, Highlights: nil}
		m.recordAction(ActionAddBox, addData, deleteData)
		m.mode = ModeNormal
		m.menuItems = nil
		m.ensureCursorInBounds()

	case MenuNewText:
		m.mode = ModeTextInput
		m.textInputX, m.textInputY = m.menuWorldX, m.menuWorldY
		m.textInputText = ""
		m.textInputCursorPos = 0
		m.menuItems = nil

	case MenuEditBox:
		if m.menuTargetBox >= 0 && m.menuTargetBox < len(canvas.boxes) {
			m.selectedBox = m.menuTargetBox
			m.selectedText = -1
			m.mode = ModeEditing
			m.editText = canvas.GetBoxText(m.menuTargetBox)
			m.originalEditText = m.editText
			m.editCursorPos = len(m.editText)
			m.editSelectionStart = -1
			m.editSelectionEnd = -1
			m.syncCursorPositions()
		} else {
			m.mode = ModeNormal
		}
		m.menuItems = nil

	case MenuEditText:
		if m.menuTargetText >= 0 && m.menuTargetText < len(canvas.texts) {
			m.selectedText = m.menuTargetText
			m.selectedBox = -1
			m.mode = ModeEditing
			m.editText = canvas.GetTextText(m.menuTargetText)
			m.originalEditText = m.editText
			m.editCursorPos = len(m.editText)
			m.editSelectionStart = -1
			m.editSelectionEnd = -1
			m.syncCursorPositions()
		} else {
			m.mode = ModeNormal
		}
		m.menuItems = nil

	case MenuNewLine:
		if m.menuTargetBox >= 0 && m.menuTargetBox < len(canvas.boxes) {
			fromBox := canvas.boxes[m.menuTargetBox]
			m.connectionFrom = m.menuTargetBox
			m.connectionFromLine = -1
			m.connectionFromX, m.connectionFromY = canvas.findNearestEdgePoint(fromBox, m.menuWorldX, m.menuWorldY)
			m.connectionWaypoints = nil
			m.mouseLineDrawing = true
			m.cursorX, m.cursorY = m.menuX, m.menuY
			m.ensureCursorInBounds()
		} else if m.menuTargetConn >= 0 && m.menuTargetConn < len(canvas.connections) {
			_, px, py := canvas.findNearestPointOnConnection(m.menuWorldX, m.menuWorldY)
			m.connectionFrom = -1
			m.connectionFromLine = m.menuTargetConn
			m.connectionFromX, m.connectionFromY = px, py
			m.connectionWaypoints = nil
			m.mouseLineDrawing = true
			m.cursorX, m.cursorY = m.menuX, m.menuY
			m.ensureCursorInBounds()
		}
		m.mode = ModeNormal
		m.menuItems = nil

	case MenuDeleteBox:
		m.deleteBoxByID(m.menuTargetBox)
		m.selBox, m.selText, m.selConn = -1, -1, -1
		m.mode = ModeNormal
		m.menuItems = nil

	case MenuDeleteText:
		m.deleteTextByID(m.menuTargetText)
		m.selBox, m.selText, m.selConn = -1, -1, -1
		m.mode = ModeNormal
		m.menuItems = nil

	case MenuDeleteLine:
		m.deleteConnByIdx(m.menuTargetConn)
		m.selBox, m.selText, m.selConn = -1, -1, -1
		m.mode = ModeNormal
		m.menuItems = nil

	case MenuEditTitle:
		if m.menuTargetBox >= 0 && m.menuTargetBox < len(canvas.boxes) {
			m.mode = ModeTitleEdit
			m.titleEditBoxID = m.menuTargetBox
			m.titleEditText = canvas.boxes[m.menuTargetBox].Title
			m.originalTitleText = m.titleEditText
			m.titleEditCursorPos = len(m.titleEditText)
		} else {
			m.mode = ModeNormal
		}
		m.menuItems = nil

	case MenuSetBorderStyle:
		if m.menuTargetBox >= 0 && m.menuTargetBox < len(canvas.boxes) {
			oldStyle := canvas.boxes[m.menuTargetBox].BorderStyle
			newStyle := BorderStyle(arg)
			canvas.SetBorderStyle(m.menuTargetBox, newStyle)
			borderData := BorderStyleData{BoxID: m.menuTargetBox, OldStyle: oldStyle, NewStyle: newStyle}
			m.recordAction(ActionChangeBorderStyle, borderData, borderData)
		}
		m.mode = ModeNormal
		m.menuItems = nil

	case MenuSetColor:
		m.applyMenuColor(arg)
		m.mode = ModeNormal
		m.menuItems = nil
	}

	return nil
}

// applyMenuColor sets the color (arg, or -1 for none) of whichever object the
// menu targets, recording the change for undo.
func (m *model) applyMenuColor(color int) {
	canvas := m.getCanvas()
	if canvas == nil {
		return
	}
	var kind, id, old int
	switch {
	case m.menuTargetBox >= 0 && m.menuTargetBox < len(canvas.boxes):
		kind, id = ColorKindBox, m.menuTargetBox
		old = canvas.boxes[id].Color
		canvas.SetBoxColor(id, color)
	case m.menuTargetConn >= 0 && m.menuTargetConn < len(canvas.connections):
		kind, id = ColorKindLine, m.menuTargetConn
		old = canvas.connections[id].Color
		canvas.SetLineColor(id, color)
	case m.menuTargetText >= 0 && m.menuTargetText < len(canvas.texts):
		kind, id = ColorKindText, m.menuTargetText
		old = canvas.texts[id].Color
		canvas.SetTextColor(id, color)
	default:
		return
	}
	if old != color {
		data := ColorData{Kind: kind, ID: id, OldColor: old, NewColor: color}
		m.recordAction(ActionSetColor, data, data)
	}
}

// deleteBoxByID removes a box and records the action for undo.
func (m *model) deleteBoxByID(boxID int) {
	canvas := m.getCanvas()
	if canvas == nil || boxID < 0 || boxID >= len(canvas.boxes) {
		return
	}
	box := canvas.boxes[boxID]
	connectedConnections := make([]Connection, 0)
	for _, connection := range canvas.connections {
		if connection.FromID == boxID || connection.ToID == boxID {
			connectedConnections = append(connectedConnections, connection)
		}
	}
	highlights := canvas.getHighlightsForBox(boxID)
	deleteData := DeleteBoxData{Box: box, ID: boxID, Connections: connectedConnections, Highlights: highlights}
	addData := AddBoxData{X: box.X, Y: box.Y, Text: box.GetText(), ID: box.ID}
	m.recordAction(ActionDeleteBox, deleteData, addData)
	canvas.DeleteBox(boxID)
	m.ensureCursorInBounds()
}

// deleteTextByID removes a text object and records the action for undo.
func (m *model) deleteTextByID(textID int) {
	canvas := m.getCanvas()
	if canvas == nil || textID < 0 || textID >= len(canvas.texts) {
		return
	}
	text := canvas.texts[textID]
	highlights := canvas.getHighlightsForText(textID)
	deleteData := DeleteTextData{Text: text, ID: textID, Highlights: highlights}
	addData := AddTextData{X: text.X, Y: text.Y, Text: text.GetText(), ID: text.ID}
	m.recordAction(ActionDeleteText, deleteData, addData)
	canvas.DeleteText(textID)
	m.ensureCursorInBounds()
}

// deleteConnByIdx removes a connection and records the action for undo.
func (m *model) deleteConnByIdx(connIdx int) {
	canvas := m.getCanvas()
	if canvas == nil || connIdx < 0 || connIdx >= len(canvas.connections) {
		return
	}
	conn := canvas.connections[connIdx]
	deleteData := AddConnectionData{FromID: conn.FromID, ToID: conn.ToID, Connection: conn}
	canvas.RemoveSpecificConnection(conn)
	m.recordAction(ActionDeleteConnection, deleteData, deleteData)
}

// overlaySelection tints the currently mouse-selected element on the raw render
// result so it stands out. Called before ANSI colors are applied.
func (m model) overlaySelection(r *RenderResult, panX, panY int) {
	canvas := m.getCanvas()
	if canvas == nil {
		return
	}
	var cells []point
	switch {
	case m.selBox >= 0 && m.selBox < len(canvas.boxes):
		cells = canvas.GetBoxBorderCells(m.selBox)
	case m.selText >= 0 && m.selText < len(canvas.texts):
		cells = canvas.GetTextCells(m.selText)
	case m.selConn >= 0 && m.selConn < len(canvas.connections):
		cells = canvas.GetConnectionCells(m.selConn)
	}
	// A multi-select group (in ModeMove) tints every selected element.
	for _, id := range m.selectedBoxes {
		if id >= 0 && id < len(canvas.boxes) {
			cells = append(cells, canvas.GetBoxBorderCells(id)...)
		}
	}
	for _, id := range m.selectedTexts {
		if id >= 0 && id < len(canvas.texts) {
			cells = append(cells, canvas.GetTextCells(id)...)
		}
	}
	for _, id := range m.selectedConnections {
		if id >= 0 && id < len(canvas.connections) {
			cells = append(cells, canvas.GetConnectionCells(id)...)
		}
	}
	for _, cell := range cells {
		sx := cell.X - panX
		sy := cell.Y - panY
		if sy >= 0 && sy < len(r.ColorMap) && sx >= 0 && sx < len(r.ColorMap[sy]) {
			r.ColorMap[sy][sx] = colorMouseSelect
		}
	}
}

// overlayContextMenu draws the right-click menu (and any open submenus) onto the
// raw render result, cascading each level to the right of its parent.
func (m model) overlayContextMenu(r *RenderResult) {
	if len(m.menuItems) == 0 {
		return
	}
	for _, level := range m.allMenuLevels() {
		m.drawMenuLevel(r, level)
	}
}

// drawMenuLevel renders a single (sub)menu box.
func (m model) drawMenuLevel(r *RenderResult, level menuLevel) {
	x, y, w, h := m.levelBounds(level)
	inner := w - 2

	setCell := func(px, py int, ch rune, colorIdx int) {
		if py < 0 || py >= len(r.Canvas) || px < 0 || px >= len(r.Canvas[py]) {
			return
		}
		r.Canvas[py][px] = ch
		if py < len(r.ColorMap) && px < len(r.ColorMap[py]) {
			r.ColorMap[py][px] = colorIdx
		}
	}

	// The menu frame (borders, corners, separators) is drawn in green.
	for row := 0; row < h; row++ {
		py := y + row
		switch {
		case row == 0:
			setCell(x, py, '┌', colorMenuBorder)
			for i := 0; i < inner; i++ {
				setCell(x+1+i, py, '─', colorMenuBorder)
			}
			setCell(x+w-1, py, '┐', colorMenuBorder)
		case row == h-1:
			setCell(x, py, '└', colorMenuBorder)
			for i := 0; i < inner; i++ {
				setCell(x+1+i, py, '─', colorMenuBorder)
			}
			setCell(x+w-1, py, '┘', colorMenuBorder)
		default:
			itemIdx := row - 1
			item := level.items[itemIdx]
			if item.Separator {
				setCell(x, py, '├', colorMenuBorder)
				for i := 0; i < inner; i++ {
					setCell(x+1+i, py, '─', colorMenuBorder)
				}
				setCell(x+w-1, py, '┤', colorMenuBorder)
				continue
			}
			rowColor := -1
			if itemIdx == level.index {
				rowColor = colorMenuSelect
			}
			setCell(x, py, '│', colorMenuBorder)
			// Interior: " label" left-aligned; a "▸" on the far right marks a submenu.
			label := []rune(" " + item.Label)
			for i := 0; i < inner; i++ {
				ch := ' '
				if i < len(label) {
					ch = label[i]
				}
				setCell(x+1+i, py, ch, rowColor)
			}
			if len(item.Submenu) > 0 {
				setCell(x+w-2, py, '▸', rowColor)
			}
			setCell(x+w-1, py, '│', colorMenuBorder)
		}
	}
}
