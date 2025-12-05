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

