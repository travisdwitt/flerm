package tui

func (m *model) moveHighlightsOnSelectedObjects(cumulativeDeltaX, cumulativeDeltaY int) point {
	if len(m.originalHighlights) == 0 {
		return m.highlightMoveDelta
	}
	for origPos := range m.originalHighlights {
		m.getCanvas().ClearHighlight(origPos.X+m.highlightMoveDelta.X, origPos.Y+m.highlightMoveDelta.Y)
	}
	for origPos, color := range m.originalHighlights {
		newX, newY := origPos.X+cumulativeDeltaX, origPos.Y+cumulativeDeltaY
		if newX >= 0 && newY >= 0 {
			m.getCanvas().SetHighlight(newX, newY, color)
		}
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
		m.selectionStartX = -1
		m.selectionStartY = -1
	}
}

func (m *model) commitMove() {
	for _, boxID := range m.selectedBoxes {
		if boxID < 0 || boxID >= len(m.getCanvas().Boxes()) {
			continue
		}
		cur := m.getCanvas().Boxes()[boxID]
		orig, ok := m.originalBoxPositions[boxID]
		if ok && (cur.X != orig.X || cur.Y != orig.Y) {
			moveData := MoveBoxData{ID: boxID, DeltaX: cur.X - orig.X, DeltaY: cur.Y - orig.Y}
			originalState := OriginalBoxState{
				ID: boxID, X: orig.X, Y: orig.Y, Width: cur.Width, Height: cur.Height,
				Connections: m.originalBoxConnections[boxID],
			}
			m.recordAction(ActionMoveBox, moveData, originalState)
		}
	}
	for _, textID := range m.selectedTexts {
		if textID < 0 || textID >= len(m.getCanvas().Texts()) {
			continue
		}
		cur := m.getCanvas().Texts()[textID]
		orig, ok := m.originalTextPositions[textID]
		if ok && (cur.X != orig.X || cur.Y != orig.Y) {
			moveData := MoveTextData{ID: textID, DeltaX: cur.X - orig.X, DeltaY: cur.Y - orig.Y}
			m.recordAction(ActionMoveText, moveData, OriginalTextState{ID: textID, X: orig.X, Y: orig.Y})
		}
	}

	if m.selectedBox != -1 && m.selectedBox < len(m.getCanvas().Boxes()) {
		cur := m.getCanvas().Boxes()[m.selectedBox]
		if cur.X != m.originalMoveX || cur.Y != m.originalMoveY {
			moveData := MoveBoxData{ID: m.selectedBox, DeltaX: cur.X - m.originalMoveX, DeltaY: cur.Y - m.originalMoveY}
			var highlightCells []HighlightCell
			for origPos, color := range m.originalHighlights {
				highlightCells = append(highlightCells, HighlightCell{X: origPos.X, Y: origPos.Y, Color: color})
			}
			originalState := OriginalBoxState{
				ID: m.selectedBox, X: m.originalMoveX, Y: m.originalMoveY, Width: cur.Width, Height: cur.Height,
				Connections: m.originalBoxConnections[m.selectedBox], Highlights: highlightCells,
			}
			m.recordAction(ActionMoveBox, moveData, originalState)
		}
	} else if m.selectedText != -1 && m.selectedText < len(m.getCanvas().Texts()) {
		cur := m.getCanvas().Texts()[m.selectedText]
		if cur.X != m.originalTextMoveX || cur.Y != m.originalTextMoveY {
			moveData := MoveTextData{ID: m.selectedText, DeltaX: cur.X - m.originalTextMoveX, DeltaY: cur.Y - m.originalTextMoveY}
			m.recordAction(ActionMoveText, moveData, OriginalTextState{ID: m.selectedText, X: m.originalTextMoveX, Y: m.originalTextMoveY})
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
