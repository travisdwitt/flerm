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

