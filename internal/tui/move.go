package tui

import "slices"

func (m *model) clearGroupSelection() {
	m.selectedBoxes, m.selectedTexts, m.selectedConnections = nil, nil, nil
	m.originalBoxPositions = map[int]point{}
	m.originalTextPositions = map[int]point{}
	m.originalConnections = map[int]Connection{}
	m.originalBoxConnections = map[int][]Connection{}
	m.originalHighlights = map[point]int{}
	m.highlightMoveDelta = point{}
	m.groupConnSnapshot = nil
}

func (m *model) hasGroupSelection() bool {
	return len(m.selectedBoxes) > 0 || len(m.selectedTexts) > 0 || len(m.selectedConnections) > 0 || len(m.originalHighlights) > 0
}

func (m *model) captureBoxMove(boxID int) {
	canvas := m.getCanvas()
	box := canvas.Boxes()[boxID]
	m.originalMoveX, m.originalMoveY = box.X, box.Y
	m.originalBoxConnections = map[int][]Connection{boxID: canvas.GetConnectionsForBox(boxID)}
	m.captureHighlights(canvas.GetBoxCells(boxID))
}

func (m *model) captureTextMove(textID int) {
	canvas := m.getCanvas()
	text := canvas.Texts()[textID]
	m.originalMoveX, m.originalMoveY = text.X, text.Y
	m.captureHighlights(canvas.GetTextCells(textID))
}

func (m *model) beginMove(worldX, worldY int) {
	canvas := m.getCanvas()
	m.clearGroupSelection()
	m.selectedBox, m.selectedText = -1, -1
	switch boxID, textID := canvas.GetBoxAt(worldX, worldY), canvas.GetTextAt(worldX, worldY); {
	case boxID != -1:
		m.selectedBox = boxID
		m.captureBoxMove(boxID)
	case textID != -1:
		m.selectedText = textID
		m.captureTextMove(textID)
	default:
		color := canvas.GetHighlight(worldX, worldY)
		if color == -1 {
			return
		}
		m.originalHighlights[point{X: worldX, Y: worldY}] = color
		m.groupConnSnapshot = canvas.SnapshotConnections()
	}
	m.mode = ModeMove
}

func (m *model) captureHighlights(cells []point) {
	m.originalHighlights = make(map[point]int)
	for _, cell := range cells {
		if color := m.getCanvas().GetHighlight(cell.X, cell.Y); color != -1 {
			m.originalHighlights[cell] = color
		}
	}
	m.highlightMoveDelta = point{}
}

func (m *model) capturedHighlightCells() []HighlightCell {
	var cells []HighlightCell
	for pos, color := range m.originalHighlights {
		cells = append(cells, HighlightCell{X: pos.X, Y: pos.Y, Color: color})
	}
	return cells
}

func (m *model) moveRecordedHighlights(cells []HighlightCell, fromDX, fromDY, toDX, toDY int) {
	for _, cell := range cells {
		m.getCanvas().ClearHighlight(cell.X+fromDX, cell.Y+fromDY)
	}
	for _, cell := range cells {
		m.getCanvas().SetHighlight(cell.X+toDX, cell.Y+toDY, cell.Color)
	}
}

func (m *model) moveCapturedHighlights(dx, dy int) {
	if len(m.originalHighlights) == 0 {
		return
	}
	canvas := m.getCanvas()
	for pos := range m.originalHighlights {
		canvas.ClearHighlight(pos.X+m.highlightMoveDelta.X, pos.Y+m.highlightMoveDelta.Y)
	}
	for pos, color := range m.originalHighlights {
		canvas.SetHighlight(pos.X+dx, pos.Y+dy, color)
	}
	m.highlightMoveDelta = point{X: dx, Y: dy}
}

func (m *model) moveContainedConnections(deltaX, deltaY int) {
	inBoxes := func(id int) bool { return slices.Contains(m.selectedBoxes, id) }
	shift := func(x, y *int) { *x += deltaX; *y += deltaY }

	for i := range m.getCanvas().Connections() {
		conn := &m.getCanvas().Connections()[i]
		moveFrom, moveTo := false, false
		switch {
		case conn.FromID >= 0 && conn.ToID >= 0:
			if !inBoxes(conn.FromID) || !inBoxes(conn.ToID) {
				continue
			}
			moveFrom, moveTo = true, true
		case slices.Contains(m.selectedConnections, i):
			moveFrom = conn.FromID == -1 || inBoxes(conn.FromID)
			moveTo = conn.ToID == -1 || inBoxes(conn.ToID)
		default:
			continue
		}
		if moveFrom {
			shift(&conn.FromX, &conn.FromY)
		}
		if moveTo {
			shift(&conn.ToX, &conn.ToY)
		}
		for j := range conn.Waypoints {
			shift(&conn.Waypoints[j].X, &conn.Waypoints[j].Y)
		}
	}
}

func (m *model) handleSingleElementMove(deltaX, deltaY int) {
	canvas := m.getCanvas()
	if m.selectedBox != -1 {
		canvas.MoveBox(m.selectedBox, deltaX, deltaY)
		box := canvas.Boxes()[m.selectedBox]
		m.moveCapturedHighlights(box.X-m.originalMoveX, box.Y-m.originalMoveY)
	} else if m.selectedText != -1 {
		canvas.MoveText(m.selectedText, deltaX, deltaY)
		text := canvas.Texts()[m.selectedText]
		m.moveCapturedHighlights(text.X-m.originalMoveX, text.Y-m.originalMoveY)
	}
	m.ensureCursorInBounds()
}

func (m *model) finalizeMultiSelect(endX, endY int) {
	if m.selectionStartX == CoordUnset || m.selectionStartY == CoordUnset {
		return
	}
	canvas := m.getCanvas()
	minX, maxX := min(m.selectionStartX, endX), max(m.selectionStartX, endX)
	minY, maxY := min(m.selectionStartY, endY), max(m.selectionStartY, endY)
	inSel := func(x, y int) bool { return x >= minX && x <= maxX && y >= minY && y <= maxY }
	overlaps := func(x0, y0, x1, y1 int) bool { return !(x1 < minX || x0 > maxX || y1 < minY || y0 > maxY) }

	m.clearGroupSelection()
	for i, box := range canvas.Boxes() {
		if overlaps(box.X, box.Y, box.X+box.Width-1, box.Y+box.Height-1) {
			m.selectedBoxes = append(m.selectedBoxes, i)
			m.originalBoxPositions[i] = point{X: box.X, Y: box.Y}
			m.originalBoxConnections[i] = canvas.GetConnectionsForBox(i)
		}
	}
	for i, text := range canvas.Texts() {
		right, bottom := text.X, text.Y
		for _, line := range text.Lines {
			right = max(right, text.X+len(line))
		}
		if len(text.Lines) > 0 {
			bottom = text.Y + len(text.Lines) - 1
		}
		if overlaps(text.X, text.Y, right, bottom) {
			m.selectedTexts = append(m.selectedTexts, i)
			m.originalTextPositions[i] = point{X: text.X, Y: text.Y}
		}
	}
	inBoxes := func(id int) bool { return id >= 0 && slices.Contains(m.selectedBoxes, id) }
	selectConn := func(conn Connection) bool {
		fromIn, toIn := inSel(conn.FromX, conn.FromY), inSel(conn.ToX, conn.ToY)
		switch {
		case inBoxes(conn.FromID) && inBoxes(conn.ToID),
			inBoxes(conn.FromID) && conn.ToID == -1 && toIn,
			inBoxes(conn.ToID) && conn.FromID == -1 && fromIn,
			conn.FromID == -1 && conn.ToID == -1 && fromIn && toIn:
			return true
		}
		count := 0
		for _, p := range connPoints(conn) {
			if inSel(p.X, p.Y) {
				count++
			}
		}
		return count*2 >= 2+len(conn.Waypoints)
	}
	for i, conn := range canvas.Connections() {
		if selectConn(conn) {
			m.selectedConnections = append(m.selectedConnections, i)
			conn.Waypoints = slices.Clone(conn.Waypoints)
			m.originalConnections[i] = conn
		}
	}
	m.groupConnSnapshot = canvas.SnapshotConnections()
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			if color := canvas.GetHighlight(x, y); color != -1 {
				m.originalHighlights[point{X: x, Y: y}] = color
			}
		}
	}

	if m.hasGroupSelection() {
		m.mode = ModeMove
		m.selectedBox, m.selectedText = -1, -1
	} else {
		m.mode = ModeNormal
		m.selectionStartX, m.selectionStartY = CoordUnset, CoordUnset
	}
}

func connPoints(conn Connection) []point {
	pts := append([]point{{X: conn.FromX, Y: conn.FromY}}, conn.Waypoints...)
	return append(pts, point{X: conn.ToX, Y: conn.ToY})
}

func (m *model) applyGroupMoveState(from, to GroupMoveState) {
	canvas := m.getCanvas()
	for id, pos := range to.BoxPositions {
		canvas.SetBoxPositionOnly(id, pos.X, pos.Y)
	}
	for id, pos := range to.TextPositions {
		canvas.SetTextPosition(id, pos.X, pos.Y)
	}
	canvas.RestoreConnectionsSnapshot(to.Connections)
	for _, cell := range from.Highlights {
		canvas.ClearHighlight(cell.X, cell.Y)
	}
	for _, cell := range to.Highlights {
		canvas.SetHighlight(cell.X, cell.Y, cell.Color)
	}
}

func (m *model) commitGroupMove() {
	canvas := m.getCanvas()
	before := GroupMoveState{BoxPositions: map[int]point{}, TextPositions: map[int]point{}, Connections: m.groupConnSnapshot}
	after := GroupMoveState{BoxPositions: map[int]point{}, TextPositions: map[int]point{}, Connections: canvas.SnapshotConnections()}
	moved := m.highlightMoveDelta != point{}

	for _, boxID := range m.selectedBoxes {
		orig, ok := m.originalBoxPositions[boxID]
		if !ok || boxID < 0 || boxID >= len(canvas.Boxes()) {
			continue
		}
		cur := canvas.Boxes()[boxID]
		before.BoxPositions[boxID] = orig
		after.BoxPositions[boxID] = point{X: cur.X, Y: cur.Y}
		moved = moved || cur.X != orig.X || cur.Y != orig.Y
	}
	for _, textID := range m.selectedTexts {
		orig, ok := m.originalTextPositions[textID]
		if !ok || textID < 0 || textID >= len(canvas.Texts()) {
			continue
		}
		cur := canvas.Texts()[textID]
		before.TextPositions[textID] = orig
		after.TextPositions[textID] = point{X: cur.X, Y: cur.Y}
		moved = moved || cur.X != orig.X || cur.Y != orig.Y
	}
	for _, connIdx := range m.selectedConnections {
		orig, ok := m.originalConnections[connIdx]
		if !ok || connIdx < 0 || connIdx >= len(canvas.Connections()) {
			continue
		}
		cur := canvas.Connections()[connIdx]
		moved = moved || cur.FromX != orig.FromX || cur.FromY != orig.FromY || cur.ToX != orig.ToX || cur.ToY != orig.ToY
	}
	for pos, color := range m.originalHighlights {
		before.Highlights = append(before.Highlights, HighlightCell{X: pos.X, Y: pos.Y, Color: color})
		after.Highlights = append(after.Highlights, HighlightCell{X: pos.X + m.highlightMoveDelta.X, Y: pos.Y + m.highlightMoveDelta.Y, Color: color})
	}

	if moved {
		data := GroupMoveData{Before: before, After: after}
		m.recordAction(ActionGroupMove, data, data)
	}
}

func (m *model) recordBoxMove(id int) {
	canvas := m.getCanvas()
	if id < 0 || id >= len(canvas.Boxes()) {
		return
	}
	cur := canvas.Boxes()[id]
	dx, dy := cur.X-m.originalMoveX, cur.Y-m.originalMoveY
	if dx == 0 && dy == 0 {
		return
	}
	m.recordAction(ActionMoveBox, MoveData{ID: id, DeltaX: dx, DeltaY: dy}, OriginalBoxState{
		ID: id, X: m.originalMoveX, Y: m.originalMoveY, Width: cur.Width, Height: cur.Height,
		Connections: m.originalBoxConnections[id], Highlights: m.capturedHighlightCells(),
	})
}

func (m *model) recordTextMove(id int) {
	canvas := m.getCanvas()
	if id < 0 || id >= len(canvas.Texts()) {
		return
	}
	cur := canvas.Texts()[id]
	dx, dy := cur.X-m.originalMoveX, cur.Y-m.originalMoveY
	if dx == 0 && dy == 0 {
		return
	}
	m.recordAction(ActionMoveText, MoveData{ID: id, DeltaX: dx, DeltaY: dy}, OriginalBoxState{
		ID: id, X: m.originalMoveX, Y: m.originalMoveY, Highlights: m.capturedHighlightCells(),
	})
}

func (m *model) commitMove() {
	switch {
	case m.hasGroupSelection():
		m.commitGroupMove()
	case m.selectedBox != -1:
		m.recordBoxMove(m.selectedBox)
	case m.selectedText != -1:
		m.recordTextMove(m.selectedText)
	}
	m.mode = ModeNormal
	m.selectedBox, m.selectedText = -1, -1
	m.clearGroupSelection()
}

func (m *model) handleMultiSelectMove(deltaX, deltaY int) {
	canvas := m.getCanvas()
	for _, boxID := range m.selectedBoxes {
		canvas.MoveBoxOnly(boxID, deltaX, deltaY)
	}
	for _, textID := range m.selectedTexts {
		canvas.MoveText(textID, deltaX, deltaY)
	}
	m.moveContainedConnections(deltaX, deltaY)

	var dx, dy int
	switch {
	case len(m.selectedBoxes) > 0:
		id := m.selectedBoxes[0]
		if orig, ok := m.originalBoxPositions[id]; ok && id >= 0 && id < len(canvas.Boxes()) {
			cur := canvas.Boxes()[id]
			dx, dy = cur.X-orig.X, cur.Y-orig.Y
		}
	case len(m.selectedTexts) > 0:
		id := m.selectedTexts[0]
		if orig, ok := m.originalTextPositions[id]; ok && id >= 0 && id < len(canvas.Texts()) {
			cur := canvas.Texts()[id]
			dx, dy = cur.X-orig.X, cur.Y-orig.Y
		}
	case len(m.selectedConnections) > 0:
		idx := m.selectedConnections[0]
		if orig, ok := m.originalConnections[idx]; ok && idx >= 0 && idx < len(canvas.Connections()) {
			cur := canvas.Connections()[idx]
			dx, dy = cur.FromX-orig.FromX, cur.FromY-orig.FromY
		}
	case len(m.originalHighlights) > 0:
		dx, dy = m.highlightMoveDelta.X+deltaX, m.highlightMoveDelta.Y+deltaY
	}
	m.moveCapturedHighlights(dx, dy)
	m.ensureCursorInBounds()
}
