package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m model) Init() tea.Cmd {
	return m.effectTick()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case struct{}:
		return m, nil

	case effectTickMsg:
		if !m.inStartMenuFlow() {
			m.effect = effectNone
			return m, nil
		}
		m.advanceEffect()
		return m, m.effectTick()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ensureCursorInBounds()
		m.clampFileScroll()
		return m, nil

	case tea.KeyMsg:
		if m.help && m.mode != ModeStartup {
			switch msg.String() {
			case "esc", "escape", "q":
				m.help = false
				m.helpScroll = 0
			case "j", "down":
				m.helpScroll = min(m.helpScroll+1, m.helpMaxScroll())
			case "k", "up":
				m.helpScroll = max(m.helpScroll-1, 0)
			case "pgdown":
				m.helpScroll = min(m.helpScroll+m.helpPageHeight(), m.helpMaxScroll())
			case "pgup":
				m.helpScroll = max(m.helpScroll-m.helpPageHeight(), 0)
			}
			return m, nil
		}

		switch m.mode {
		case ModeStartup:
			return m.handleStartupKey(msg)
		case ModeNormal:
			return m.handleNormalKey(msg)
		case ModeContextMenu:
			return m.handleContextMenuKey(msg)
		case ModeEditing, ModeTextInput, ModeTitleEdit, ModeTooltipEdit:
			return m.handleTextEditKey(msg)
		case ModeBoxJump:
			return m.handleBoxJumpKey(msg)
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
