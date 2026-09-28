package tui

import (
	"strconv"
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
)

func (m model) handleTextEditKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	selectable := m.mode == ModeEditing
	n := runeLen(m.editText)

	insert := func(s string) {
		if selectable {
			m.deleteEditSelection()
		}
		m.editText = splice(m.editText, m.editCursorPos, m.editCursorPos, s)
		m.editCursorPos += runeLen(s)
		m.syncEditTarget()
	}
	moveTo := func(pos int, extend bool) {
		if selectable && extend {
			m.startEditSelection()
			m.editSelectionEnd = pos
		} else {
			m.clearEditSelection()
		}
		m.editCursorPos = pos
	}
	rowDelta := func(d int) int {
		row, col := linearToRowCol(m.editCursorPos, m.editText)
		if row+d < 0 || row+d >= strings.Count(m.editText, "\n")+1 {
			return m.editCursorPos
		}
		return rowColToLinear(row+d, col, m.editText)
	}
	shift := false
	switch msg.Type {
	case tea.KeyShiftLeft, tea.KeyShiftRight, tea.KeyShiftUp, tea.KeyShiftDown, tea.KeyShiftHome, tea.KeyShiftEnd:
		shift = true
	}

	switch msg.Type {
	case tea.KeyCtrlS, tea.KeyEscape:
		m.finishEdit()
	case tea.KeyCtrlV:
		if clip, err := readClipboardText(); err == nil && clip != "" {
			insert(clip)
		}
	case tea.KeyHome, tea.KeyShiftHome:
		moveTo(lineStartPos(m.editText, m.editCursorPos), shift)
	case tea.KeyEnd, tea.KeyShiftEnd:
		moveTo(lineEndPos(m.editText, m.editCursorPos), shift)
	case tea.KeyLeft, tea.KeyShiftLeft:
		moveTo(max(m.editCursorPos-1, 0), shift)
	case tea.KeyRight, tea.KeyShiftRight:
		moveTo(min(m.editCursorPos+1, n), shift)
	case tea.KeyUp, tea.KeyShiftUp:
		moveTo(rowDelta(-1), shift)
	case tea.KeyDown, tea.KeyShiftDown:
		moveTo(rowDelta(1), shift)
	case tea.KeyEnter:
		insert("\n")
	case tea.KeyBackspace:
		if !(selectable && m.deleteEditSelection()) && m.editCursorPos > 0 {
			m.editText = splice(m.editText, m.editCursorPos-1, m.editCursorPos, "")
			m.editCursorPos--
		}
		m.syncEditTarget()
	case tea.KeyDelete:
		if !(selectable && m.deleteEditSelection()) && m.editCursorPos < n {
			m.editText = splice(m.editText, m.editCursorPos, m.editCursorPos+1, "")
		}
		m.syncEditTarget()
	case tea.KeySpace, tea.KeyRunes:
		if selectable && string(msg.Runes) == "y" && m.hasEditSelection() {
			start, end := m.editSelectionBounds()
			if err := clipboard.WriteAll(string([]rune(m.editText)[start:end])); err != nil {
				m.errorMessage = "Could not copy to clipboard: " + err.Error()
			}
			break
		}
		insert(string(msg.Runes))
	}
	return m, nil
}

func (m model) handleBoxJumpKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape, tea.KeyEnter:
		if boxNum, err := strconv.Atoi(m.boxJumpInput); msg.Type == tea.KeyEnter && err == nil && boxNum >= 0 && boxNum < len(m.getCanvas().Boxes()) {
			box := m.getCanvas().Boxes()[boxNum]
			panX, panY := m.getPanOffset()
			m.cursorX = box.X + box.Width/2 - panX
			m.cursorY = box.Y + box.Height/2 - panY
			m.ensureCursorInBounds()
		}
		m.mode = ModeNormal
		m.boxJumpInput = ""
	case tea.KeyBackspace:
		if len(m.boxJumpInput) > 0 {
			m.boxJumpInput = m.boxJumpInput[:len(m.boxJumpInput)-1]
		}
	case tea.KeyRunes:
		if s := string(msg.Runes); len(s) == 1 && s[0] >= '0' && s[0] <= '9' {
			m.boxJumpInput += s
		}
	}
	return m, nil
}
