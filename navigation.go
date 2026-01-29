package main

import tea "github.com/charmbracelet/bubbletea"

func (m *model) handleNavigation(key string, speed int) (tea.Model, tea.Cmd) {
	if m.zPanMode {
		return m.handlePan(key, speed), nil
	}

	// Store old tooltip state
	oldShowTooltip := m.showTooltip
	oldTooltipText := m.tooltipText

	// Move cursor
	updatedModel := m.handleCursorMove(key, speed)

	// Update tooltip state after movement
	updatedModel.updateTooltip()

	// If tooltip state changed, we want to force a re-render
	if oldShowTooltip != updatedModel.showTooltip ||
	   (updatedModel.showTooltip && oldTooltipText != updatedModel.tooltipText) {
		// Return a command to force re-render
		return updatedModel, func() tea.Msg { return struct{}{} }
	}

	return updatedModel, nil
}

func (m *model) handlePan(key string, speed int) *model {
	buf := m.getCurrentBuffer()
	if buf == nil {
		return m
	}
	switch key {
	case "h", "left", "H", "shift+left":
		buf.panX += speed
	case "l", "right", "L", "shift+right":
		buf.panX -= speed
	case "k", "up", "K", "shift+up":
		buf.panY += speed
	case "j", "down", "J", "shift+down":
		buf.panY -= speed
	}
	return m
}

func (m *model) handleCursorMove(key string, speed int) *model {
	oldX := m.cursorX
	oldY := m.cursorY
	
	switch key {
	case "h", "left", "H", "shift+left":
		m.cursorX -= speed
	case "l", "right", "L", "shift+right":
		m.cursorX += speed
	case "k", "up", "K", "shift+up":
		m.cursorY -= speed
	case "j", "down", "J", "shift+down":
		m.cursorY += speed
	}
	m.ensureCursorInBounds()
	if m.highlightMode {
		buf := m.getCurrentBuffer()
		panX, panY := 0, 0
		if buf != nil {
			panX, panY = buf.panX, buf.panY
		}
		worldX := m.cursorX + panX
		worldY := m.cursorY + panY
		oldWorldX := oldX + panX
		oldWorldY := oldY + panY
		highlightedCells := make([]HighlightCell, 0)
		addHighlightCell := func(x, y int) {
			oldColor := m.getCanvas().GetHighlight(x, y)
			m.getCanvas().SetHighlight(x, y, m.selectedColor)
			highlightedCells = append(highlightedCells, HighlightCell{
				X:        x,
				Y:        y,
				Color:    m.selectedColor,
				HadColor: oldColor != -1,
				OldColor: oldColor,
			})
		}
		addHighlightCell(worldX, worldY)
		if speed > 1 {
			dx := worldX - oldWorldX
			dy := worldY - oldWorldY
			if dx != 0 && dy == 0 {
				step := 1
				if dx < 0 {
					step = -1
				}
				for x := oldWorldX + step; x != worldX; x += step {
					addHighlightCell(x, worldY)
				}
			} else if dy != 0 && dx == 0 {
				step := 1
				if dy < 0 {
					step = -1
				}
				for y := oldWorldY + step; y != worldY; y += step {
					addHighlightCell(worldX, y)
				}
			}
		}
		if len(highlightedCells) > 0 {
			inverseCells := make([]HighlightCell, len(highlightedCells))
			for i, cell := range highlightedCells {
				oldColorForInverse := cell.OldColor
				if oldColorForInverse < 0 {
					oldColorForInverse = -1
				}
				inverseCells[i] = HighlightCell{
					X:        cell.X,
					Y:        cell.Y,
					Color:    oldColorForInverse,
					HadColor: cell.HadColor,
					OldColor: cell.Color,
				}
			}
			m.recordAction(ActionHighlight, HighlightData{Cells: highlightedCells}, HighlightData{Cells: inverseCells})
		}
	}
	return m
}

func (m *model) getMoveSpeed(key string) int {
	switch key {
	case "H", "L", "K", "J", "shift+left", "shift+right", "shift+up", "shift+down":
		return 2
	default:
		return 1
	}
}

// mindMapNavigate handles navigation between mind map nodes
// Returns true if navigation happened, false if caller should fall through to regular movement
func (m *model) mindMapNavigate(direction string) bool {
	panX, panY := m.getPanOffset()
	worldX, worldY := m.cursorX+panX, m.cursorY+panY
	currentBoxID := m.getCanvas().GetBoxAt(worldX, worldY)

	// If not on a node, try to find nearest node first
	if currentBoxID == -1 {
		currentBoxID = m.findNearestMindMapNode(worldX, worldY)
		if currentBoxID != -1 && currentBoxID < len(m.getCanvas().boxes) {
			// Move to the nearest node
			targetBox := m.getCanvas().boxes[currentBoxID]
			m.cursorX = targetBox.X + targetBox.Width/2 - panX
			m.cursorY = targetBox.Y + targetBox.Height/2 - panY
			m.ensureCursorInBounds()
			m.selectedMindMapNode = currentBoxID
			return true
		}
		return false
	}

	var targetBoxID int = -1

	switch direction {
	case "left":
		// Navigate to parent
		if parent, ok := m.mindMapParents[currentBoxID]; ok && parent >= 0 {
			targetBoxID = parent
		}
	case "right":
		// Navigate to first child
		targetBoxID = m.findFirstChild(currentBoxID)
	case "up":
		// Navigate to previous sibling
		targetBoxID = m.findPreviousSibling(currentBoxID)
	case "down":
		// Navigate to next sibling
		targetBoxID = m.findNextSibling(currentBoxID)
	}

	if targetBoxID != -1 && targetBoxID < len(m.getCanvas().boxes) {
		// Move cursor to target node
		targetBox := m.getCanvas().boxes[targetBoxID]
		// Position cursor at the center of the target node
		newWorldX := targetBox.X + targetBox.Width/2
		newWorldY := targetBox.Y + targetBox.Height/2
		m.cursorX = newWorldX - panX
		m.cursorY = newWorldY - panY
		m.ensureCursorInBounds()
		m.selectedMindMapNode = targetBoxID
		return true
	}

	// No target found, don't move
	return false
}

// findNearestMindMapNode finds the nearest node to the given position
func (m *model) findNearestMindMapNode(worldX, worldY int) int {
	canvas := m.getCanvas()
	if len(canvas.boxes) == 0 {
		return -1
	}

	nearestID := -1
	nearestDist := -1

	for _, box := range canvas.boxes {
		// Calculate distance to box center
		centerX := box.X + box.Width/2
		centerY := box.Y + box.Height/2
		dist := abs(worldX-centerX) + abs(worldY-centerY)
		if nearestDist == -1 || dist < nearestDist {
			nearestDist = dist
			nearestID = box.ID
		}
	}

	return nearestID
}

// findFirstChild finds the first child of the given node
func (m *model) findFirstChild(nodeID int) int {
	children := m.getMindMapChildren(nodeID)
	if len(children) == 0 {
		return -1
	}
	return children[0]
}

// findPreviousSibling finds the sibling above the current node
func (m *model) findPreviousSibling(nodeID int) int {
	parentID, hasParent := m.mindMapParents[nodeID]
	if !hasParent {
		parentID = -1
	}

	siblings := m.getSiblings(nodeID, parentID)
	if len(siblings) <= 1 {
		return -1
	}

	// Find current position in sibling order
	for i, sibID := range siblings {
		if sibID == nodeID && i > 0 {
			return siblings[i-1]
		}
	}
	return -1
}

// findNextSibling finds the sibling below the current node
func (m *model) findNextSibling(nodeID int) int {
	parentID, hasParent := m.mindMapParents[nodeID]
	if !hasParent {
		parentID = -1
	}

	siblings := m.getSiblings(nodeID, parentID)
	if len(siblings) <= 1 {
		return -1
	}

	// Find current position in sibling order
	for i, sibID := range siblings {
		if sibID == nodeID && i < len(siblings)-1 {
			return siblings[i+1]
		}
	}
	return -1
}

