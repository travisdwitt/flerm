package tui

import tea "github.com/charmbracelet/bubbletea"

var arrowDeltas = map[string][2]int{
	"h": {-1, 0}, "left": {-1, 0}, "H": {-2, 0}, "shift+left": {-2, 0},
	"l": {1, 0}, "right": {1, 0}, "L": {2, 0}, "shift+right": {2, 0},
	"k": {0, -1}, "up": {0, -1}, "K": {0, -2}, "shift+up": {0, -2},
	"j": {0, 1}, "down": {0, 1}, "J": {0, 2}, "shift+down": {0, 2},
}

func (m *model) handleNavigation(key string) (tea.Model, tea.Cmd) {
	d := arrowDeltas[key]
	if m.zPanMode {
		buf := m.getCurrentBuffer()
		buf.panX -= d[0]
		buf.panY -= d[1]
		return m, nil
	}

	oldShow, oldText := m.showTooltip, m.tooltipText
	m.moveCursor(d[0], d[1])
	m.updateTooltip()
	if oldShow != m.showTooltip || (m.showTooltip && oldText != m.tooltipText) {
		return m, func() tea.Msg { return struct{}{} }
	}
	return m, nil
}

func (m *model) moveCursor(dx, dy int) {
	oldX, oldY := m.worldCursor()
	m.cursorX += dx
	m.cursorY += dy
	m.ensureCursorInBounds()
	if !m.highlightMode {
		return
	}
	x, y := m.worldCursor()
	cells := []point{{X: x, Y: y}}
	if mx, my := x-oldX, y-oldY; (abs(dx) > 1 || abs(dy) > 1) && (mx == 0) != (my == 0) {
		sx, sy := sign(mx), sign(my)
		for cx, cy := oldX+sx, oldY+sy; cx != x || cy != y; cx, cy = cx+sx, cy+sy {
			cells = append(cells, point{X: cx, Y: cy})
		}
	}
	m.recordHighlights(m.paintCells(cells, m.selectedColor))
}
