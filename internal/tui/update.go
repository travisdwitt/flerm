package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case struct{}:

		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ensureCursorInBounds()
		return m, nil

	case tea.KeyMsg:
		if m.help && m.mode != ModeStartup {
			switch msg.String() {
			case "esc", "escape", "q", "?":
				m.help = false
				m.helpScroll = 0
				return m, nil
			case "j", "down":
				helpLines := helpText
				totalLines := len(helpLines)
				visibleHeight := m.height - 1
				if visibleHeight < 1 {
					visibleHeight = 1
				}
				maxScroll := totalLines - visibleHeight
				if maxScroll < 0 {
					maxScroll = 0
				}
				if m.helpScroll < maxScroll {
					m.helpScroll++
				}
				return m, nil
			case "k", "up":
				if m.helpScroll > 0 {
					m.helpScroll--
				}
				return m, nil
			default:
				m.help = false
				m.helpScroll = 0
				return m, nil
			}
		}

		switch m.mode {
		case ModeStartup:
			return m.handleStartupKey(msg)
		case ModeNormal:
			return m.handleNormalKey(msg)
		case ModeContextMenu:
			return m.handleContextMenuKey(msg)
		case ModeEditing:
			return m.handleEditingKey(msg)
		case ModeTextInput:
			return m.handleTextInputKey(msg)
		case ModeBoxJump:
			return m.handleBoxJumpKey(msg)
		case ModeTitleEdit:
			return m.handleTitleEditKey(msg)
		case ModeResize:
			return m.handleResizeKey(msg)
		case ModeMultiSelect:
			return m.handleMultiSelectKey(msg)
		case ModeMove:
			return m.handleMoveKey(msg)
		case ModeFileInput:
			return m.handleFileInputKey(msg)
		case ModeConfirm:
			return m.handleConfirmKey(msg)
		}

	case tea.MouseMsg:
		return m.handleMouse(msg)
	}

	return m, nil
}
