package tui

func (m *model) captureHighlights(cells []point) {
	m.originalHighlights = make(map[point]int)
	for _, cell := range cells {
		if color := m.getCanvas().GetHighlight(cell.X, cell.Y); color != -1 {
			m.originalHighlights[cell] = color
		}
	}
	m.highlightMoveDelta = point{X: 0, Y: 0}
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

func (m *model) moveHighlightsOnSelectedObjects(cumulativeDeltaX, cumulativeDeltaY int) point {
	if len(m.originalHighlights) == 0 {
		return m.highlightMoveDelta
	}
	for origPos := range m.originalHighlights {
		m.getCanvas().ClearHighlight(origPos.X+m.highlightMoveDelta.X, origPos.Y+m.highlightMoveDelta.Y)
	}
	for origPos, color := range m.originalHighlights {
		m.getCanvas().SetHighlight(origPos.X+cumulativeDeltaX, origPos.Y+cumulativeDeltaY, color)
	}
	return point{X: cumulativeDeltaX, Y: cumulativeDeltaY}
}

func (m *model) moveContainedConnections(deltaX, deltaY int) {

	selectedBoxSet := make(map[int]bool)
	for _, boxID := range m.selectedBoxes {
		selectedBoxSet[boxID] = true
	}

	selectedConnSet := make(map[int]bool)
	for _, connIdx := range m.selectedConnections {
		selectedConnSet[connIdx] = true
	}

	for i := range m.getCanvas().Connections() {
		conn := &m.getCanvas().Connections()[i]

		if conn.FromID >= 0 && conn.ToID >= 0 {
			if selectedBoxSet[conn.FromID] && selectedBoxSet[conn.ToID] {

				conn.FromX += deltaX
				conn.FromY += deltaY
				conn.ToX += deltaX
				conn.ToY += deltaY
				for j := range conn.Waypoints {
					conn.Waypoints[j].X += deltaX
					conn.Waypoints[j].Y += deltaY
				}
			}

			continue
		}

		if selectedConnSet[i] {
			if conn.FromID == -1 || selectedBoxSet[conn.FromID] {
				conn.FromX += deltaX
				conn.FromY += deltaY
			}
			if conn.ToID == -1 || selectedBoxSet[conn.ToID] {
				conn.ToX += deltaX
				conn.ToY += deltaY
			}
			for j := range conn.Waypoints {
				conn.Waypoints[j].X += deltaX
				conn.Waypoints[j].Y += deltaY
			}
		}
	}
}

func (m *model) handleSingleElementMove(deltaX, deltaY int) {
	if m.selectedBox != -1 {
		m.getCanvas().MoveBox(m.selectedBox, deltaX, deltaY)
		if len(m.originalHighlights) > 0 {
			cumulativeDeltaX := m.getCanvas().Boxes()[m.selectedBox].X - m.originalMoveX
			cumulativeDeltaY := m.getCanvas().Boxes()[m.selectedBox].Y - m.originalMoveY
			m.highlightMoveDelta = m.moveHighlightsOnSelectedObjects(cumulativeDeltaX, cumulativeDeltaY)
		}
		m.ensureCursorInBounds()
	} else if m.selectedText != -1 {
		m.getCanvas().MoveText(m.selectedText, deltaX, deltaY)
		if len(m.originalHighlights) > 0 {
			cumulativeDeltaX := m.getCanvas().Texts()[m.selectedText].X - m.originalTextMoveX
			cumulativeDeltaY := m.getCanvas().Texts()[m.selectedText].Y - m.originalTextMoveY
			m.highlightMoveDelta = m.moveHighlightsOnSelectedObjects(cumulativeDeltaX, cumulativeDeltaY)
		}
		m.ensureCursorInBounds()
	}
}

func (m *model) finalizeMultiSelect(endX, endY int) {
	if m.selectionStartX == CoordUnset || m.selectionStartY == CoordUnset {
		return
	}
	minX, maxX := m.selectionStartX, m.selectionStartX
	if endX < m.selectionStartX {
		minX = endX
	} else if endX > m.selectionStartX {
		maxX = endX
	}
	minY, maxY := m.selectionStartY, m.selectionStartY
	if endY < m.selectionStartY {
		minY = endY
	} else if endY > m.selectionStartY {
		maxY = endY
	}

	m.selectedBoxes = []int{}
	m.selectedTexts = []int{}
	m.selectedConnections = []int{}
	m.originalBoxPositions = make(map[int]point)
	m.originalTextPositions = make(map[int]point)
	m.originalConnections = make(map[int]Connection)
	m.originalBoxConnections = make(map[int][]Connection)
	for i, box := range m.getCanvas().Boxes() {
		boxRight, boxBottom := box.X+box.Width-1, box.Y+box.Height-1
		if !(boxRight < minX || box.X > maxX || boxBottom < minY || box.Y > maxY) {
			m.selectedBoxes = append(m.selectedBoxes, i)
			m.originalBoxPositions[i] = point{X: box.X, Y: box.Y}
			m.originalBoxConnections[i] = m.getCanvas().GetConnectionsForBox(i)
		}
	}
	for i, text := range m.getCanvas().Texts() {
		textRight, textBottom := text.X, text.Y
		for _, line := range text.Lines {
			if text.X+len(line) > textRight {
				textRight = text.X + len(line)
			}
		}
		if len(text.Lines) > 0 {
			textBottom = text.Y + len(text.Lines) - 1
		}
		if !(textRight < minX || text.X > maxX || textBottom < minY || text.Y > maxY) {
			m.selectedTexts = append(m.selectedTexts, i)
			m.originalTextPositions[i] = point{X: text.X, Y: text.Y}
		}
	}
	selectedBoxSet := make(map[int]bool)
	for _, boxID := range m.selectedBoxes {
		selectedBoxSet[boxID] = true
	}
	pointInSelection := func(x, y int) bool {
		return x >= minX && x <= maxX && y >= minY && y <= maxY
	}
	shouldSelectConnection := func(conn Connection) bool {
		if conn.FromID >= 0 && conn.ToID >= 0 && selectedBoxSet[conn.FromID] && selectedBoxSet[conn.ToID] {
			return true
		}
		if conn.FromID >= 0 && selectedBoxSet[conn.FromID] && conn.ToID == -1 && pointInSelection(conn.ToX, conn.ToY) {
			return true
		}
		if conn.ToID >= 0 && selectedBoxSet[conn.ToID] && conn.FromID == -1 && pointInSelection(conn.FromX, conn.FromY) {
			return true
		}
		if conn.FromID == -1 && conn.ToID == -1 && pointInSelection(conn.FromX, conn.FromY) && pointInSelection(conn.ToX, conn.ToY) {
			return true
		}
		pointsInSelection, totalPoints := 0, 2+len(conn.Waypoints)
		if pointInSelection(conn.FromX, conn.FromY) {
			pointsInSelection++
		}
		if pointInSelection(conn.ToX, conn.ToY) {
			pointsInSelection++
		}
		for _, wp := range conn.Waypoints {
			if pointInSelection(wp.X, wp.Y) {
				pointsInSelection++
			}
		}
		return totalPoints > 0 && pointsInSelection*2 >= totalPoints
	}
	for i, conn := range m.getCanvas().Connections() {
		if shouldSelectConnection(conn) {
			m.selectedConnections = append(m.selectedConnections, i)
			connCopy := conn
			connCopy.Waypoints = make([]point, len(conn.Waypoints))
			copy(connCopy.Waypoints, conn.Waypoints)
			m.originalConnections[i] = connCopy
		}
	}
	m.groupConnSnapshot = m.getCanvas().SnapshotConnections()
	m.originalHighlights = make(map[point]int)
	m.highlightMoveDelta = point{X: 0, Y: 0}
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			if color := m.getCanvas().GetHighlight(x, y); color != -1 {
				m.originalHighlights[point{X: x, Y: y}] = color
			}
		}
	}

	if len(m.selectedBoxes) > 0 || len(m.selectedTexts) > 0 || len(m.selectedConnections) > 0 || len(m.originalHighlights) > 0 {
		m.mode = ModeMove
		m.selectedBox = -1
		m.selectedText = -1
	} else {
		m.mode = ModeNormal
		m.selectionStartX = CoordUnset
		m.selectionStartY = CoordUnset
	}
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
	before := GroupMoveState{
		BoxPositions:  make(map[int]point),
		TextPositions: make(map[int]point),
		Connections:   m.groupConnSnapshot,
	}
	after := GroupMoveState{
		BoxPositions:  make(map[int]point),
		TextPositions: make(map[int]point),
		Connections:   canvas.SnapshotConnections(),
	}
	moved := false

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
		if cur.FromX != orig.FromX || cur.FromY != orig.FromY || cur.ToX != orig.ToX || cur.ToY != orig.ToY {
			moved = true
		}
	}
	for pos, color := range m.originalHighlights {
		before.Highlights = append(before.Highlights, HighlightCell{X: pos.X, Y: pos.Y, Color: color})
		after.Highlights = append(after.Highlights, HighlightCell{
			X: pos.X + m.highlightMoveDelta.X, Y: pos.Y + m.highlightMoveDelta.Y, Color: color,
		})
	}
	if m.highlightMoveDelta.X != 0 || m.highlightMoveDelta.Y != 0 {
		moved = true
	}

	if moved {
		data := GroupMoveData{Before: before, After: after}
		m.recordAction(ActionGroupMove, data, data)
	}
}

func (m *model) commitMove() {
	if len(m.selectedBoxes) > 0 || len(m.selectedTexts) > 0 || len(m.selectedConnections) > 0 || len(m.originalHighlights) > 0 {
		m.commitGroupMove()
	}

	if m.selectedBox != -1 && m.selectedBox < len(m.getCanvas().Boxes()) {
		cur := m.getCanvas().Boxes()[m.selectedBox]
		if cur.X != m.originalMoveX || cur.Y != m.originalMoveY {
			moveData := MoveBoxData{ID: m.selectedBox, DeltaX: cur.X - m.originalMoveX, DeltaY: cur.Y - m.originalMoveY}
			originalState := OriginalBoxState{
				ID: m.selectedBox, X: m.originalMoveX, Y: m.originalMoveY, Width: cur.Width, Height: cur.Height,
				Connections: m.originalBoxConnections[m.selectedBox], Highlights: m.capturedHighlightCells(),
			}
			m.recordAction(ActionMoveBox, moveData, originalState)
		}
	} else if m.selectedText != -1 && m.selectedText < len(m.getCanvas().Texts()) {
		cur := m.getCanvas().Texts()[m.selectedText]
		if cur.X != m.originalTextMoveX || cur.Y != m.originalTextMoveY {
			moveData := MoveTextData{ID: m.selectedText, DeltaX: cur.X - m.originalTextMoveX, DeltaY: cur.Y - m.originalTextMoveY}
			originalState := OriginalTextState{
				ID: m.selectedText, X: m.originalTextMoveX, Y: m.originalTextMoveY,
				Highlights: m.capturedHighlightCells(),
			}
			m.recordAction(ActionMoveText, moveData, originalState)
		}
	}
	m.mode = ModeNormal
	m.selectedBox = -1
	m.selectedText = -1
	m.selectedBoxes = []int{}
	m.selectedTexts = []int{}
	m.selectedConnections = []int{}
	m.originalBoxPositions = make(map[int]point)
	m.originalTextPositions = make(map[int]point)
	m.originalConnections = make(map[int]Connection)
	m.originalHighlights = make(map[point]int)
	m.highlightMoveDelta = point{X: 0, Y: 0}
	m.groupConnSnapshot = nil
}

func (m *model) handleMultiSelectMove(deltaX, deltaY int) {
	for _, boxID := range m.selectedBoxes {
		m.getCanvas().MoveBoxOnly(boxID, deltaX, deltaY)
	}
	for _, textID := range m.selectedTexts {
		m.getCanvas().MoveText(textID, deltaX, deltaY)
	}
	m.moveContainedConnections(deltaX, deltaY)
	var cumulativeDeltaX, cumulativeDeltaY int
	if len(m.selectedBoxes) > 0 {
		boxID := m.selectedBoxes[0]
		if boxID >= 0 && boxID < len(m.getCanvas().Boxes()) {
			if originalPos, hasOriginal := m.originalBoxPositions[boxID]; hasOriginal {
				currentBox := m.getCanvas().Boxes()[boxID]
				cumulativeDeltaX, cumulativeDeltaY = currentBox.X-originalPos.X, currentBox.Y-originalPos.Y
			}
		}
	} else if len(m.selectedTexts) > 0 {
		textID := m.selectedTexts[0]
		if textID >= 0 && textID < len(m.getCanvas().Texts()) {
			if originalPos, hasOriginal := m.originalTextPositions[textID]; hasOriginal {
				currentText := m.getCanvas().Texts()[textID]
				cumulativeDeltaX, cumulativeDeltaY = currentText.X-originalPos.X, currentText.Y-originalPos.Y
			}
		}
	} else if len(m.selectedConnections) > 0 {
		connIdx := m.selectedConnections[0]
		if connIdx >= 0 && connIdx < len(m.getCanvas().Connections()) {
			conn := m.getCanvas().Connections()[connIdx]
			if originalConn, hasOriginal := m.originalConnections[connIdx]; hasOriginal {
				cumulativeDeltaX, cumulativeDeltaY = conn.FromX-originalConn.FromX, conn.FromY-originalConn.FromY
			}
		}
	} else if len(m.originalHighlights) > 0 {
		cumulativeDeltaX, cumulativeDeltaY = m.highlightMoveDelta.X+deltaX, m.highlightMoveDelta.Y+deltaY
	}
	m.highlightMoveDelta = m.moveHighlightsOnSelectedObjects(cumulativeDeltaX, cumulativeDeltaY)
	m.ensureCursorInBounds()
}
