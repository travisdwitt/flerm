package main

import tea "github.com/charmbracelet/bubbletea"

func (m *model) handleNavigation(key string, speed int) (tea.Model, tea.Cmd) {
	if m.zPanMode {
		return m.handlePan(key, speed), nil
	}
	return m.handleCursorMove(key, speed), nil
}

func (m *model) handlePan(key string, speed int) tea.Model {
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

func (m *model) handleCursorMove(key string, speed int) tea.Model {
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
	
	// If in highlight mode, leave a colored trail and track for undo
	if m.highlightMode {
		buf := m.getCurrentBuffer()
		panX, panY := 0, 0
		if buf != nil {
			panX, panY = buf.panX, buf.panY
		}
		// Highlight all cells between old and new position
		worldX := m.cursorX + panX
		worldY := m.cursorY + panY
		oldWorldX := oldX + panX
		oldWorldY := oldY + panY
		
		// Collect all cells that will be highlighted
		highlightedCells := make([]HighlightCell, 0)
		
		// Helper function to add a cell to the highlight list
		addHighlightCell := func(x, y int) {
			oldColor := m.getCanvas().GetHighlight(x, y)
			hadColor := (oldColor != -1)
			m.getCanvas().SetHighlight(x, y, m.selectedColor)
			highlightedCells = append(highlightedCells, HighlightCell{
				X:        x,
				Y:        y,
				Color:    m.selectedColor,
				HadColor: hadColor,
				OldColor: oldColor,
			})
		}
		
		// Highlight the new position
		addHighlightCell(worldX, worldY)
		
		// If moving multiple steps, highlight intermediate positions
		if speed > 1 {
			dx := worldX - oldWorldX
			dy := worldY - oldWorldY
			// Highlight all positions along the path
			if dx != 0 && dy == 0 {
				// Horizontal movement
				step := 1
				if dx < 0 {
					step = -1
				}
				for x := oldWorldX + step; x != worldX; x += step {
					addHighlightCell(x, worldY)
				}
			} else if dy != 0 && dx == 0 {
				// Vertical movement
				step := 1
				if dy < 0 {
					step = -1
				}
				for y := oldWorldY + step; y != worldY; y += step {
					addHighlightCell(worldX, y)
				}
			}
		}
		
		// Record the highlight action for undo
		if len(highlightedCells) > 0 {
			// Create inverse data (restore previous state)
			inverseCells := make([]HighlightCell, len(highlightedCells))
			for i, cell := range highlightedCells {
				// Inverse: restore to old state
				// If cell had a color before, restore it; otherwise clear it
				oldColorForInverse := cell.OldColor
				if oldColorForInverse < 0 {
					oldColorForInverse = -1 // Ensure -1 for no color
				}
				inverseCells[i] = HighlightCell{
					X:        cell.X,
					Y:        cell.Y,
					Color:    oldColorForInverse, // Restore to old color (or -1 to clear)
					HadColor: cell.HadColor,      // Whether it had a color before
					OldColor: cell.Color,         // Current color becomes old for inverse
				}
			}
			
			highlightData := HighlightData{Cells: highlightedCells}
			inverseData := HighlightData{Cells: inverseCells}
			m.recordAction(ActionHighlight, highlightData, inverseData)
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

