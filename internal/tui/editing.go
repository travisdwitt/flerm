package tui

import (
	"strings"
)

func (m *model) linearToCursorPos(pos int, text string) (row, col int) {
	lines := strings.Split(text, "\n")
	currentPos := 0
	for lineIdx, line := range lines {
		lineLength := len([]rune(line))
		if pos <= currentPos+lineLength {
			return lineIdx, pos - currentPos
		}
		currentPos += lineLength + 1
	}

	if len(lines) > 0 {
		return len(lines) - 1, len([]rune(lines[len(lines)-1]))
	}
	return 0, 0
}

func (m *model) cursorPosToLinear(row, col int, text string) int {
	lines := strings.Split(text, "\n")
	if row < 0 {
		row = 0
	}
	if row >= len(lines) {

		pos := 0
		for _, line := range lines {
			pos += len([]rune(line)) + 1
		}
		return pos - 1
	}

	pos := 0
	for i := 0; i < row; i++ {
		pos += len([]rune(lines[i])) + 1
	}

	lineLength := len([]rune(lines[row]))
	if col < 0 {
		col = 0
	}
	if col > lineLength {
		col = lineLength
	}

	return pos + col
}

func (m *model) syncCursorPositions() {
	m.editCursorRow, m.editCursorCol = m.linearToCursorPos(m.editCursorPos, m.editText)
}

func (m *model) editTarget() (editBoxID, editTextID, editTextX, editTextY int) {
	switch m.mode {
	case ModeEditing:
		return m.selectedBox, m.selectedText, -1, -1
	case ModeTextInput:
		return -1, -1, m.textInputX, m.textInputY
	case ModeTitleEdit:
		return m.titleEditBoxID, -2, -1, -1
	}
	return -1, -1, -1, -1
}

// Returns a nil cursor outside the three edit modes.
func (m *model) editTextCursor() (string, *int) {
	switch m.mode {
	case ModeEditing:
		return m.editText, &m.editCursorPos
	case ModeTextInput:
		return m.textInputText, &m.textInputCursorPos
	case ModeTitleEdit:
		return m.titleEditText, &m.titleEditCursorPos
	}
	return "", nil
}

// The inverse of the renderer's cursorScreenPos; off-text clicks clamp to an edge.
func (m *model) editPosAt(screenX, screenY int, text string) (int, bool) {
	originX, originY, ok := m.getCanvas().EditOrigin(m.editTarget())
	if !ok {
		return 0, false
	}
	panX, panY := m.getPanOffset()
	row := screenY - m.bufferBarOffset() + panY - originY
	col := screenX + panX - originX
	return m.cursorPosToLinear(row, col, text), true
}

// startEditSelection anchors a selection at the cursor unless one is already
// running. A collapsed selection counts as none, so the anchor never goes
// stale after a shift-key round trip.
func (m *model) startEditSelection() {
	if !m.hasEditSelection() {
		m.editSelectionStart = m.editCursorPos
		m.editSelectionEnd = m.editCursorPos
	}
}

func (m *model) clearEditSelection() {
	m.editSelectionStart = -1
	m.editSelectionEnd = -1
}

func (m *model) hasEditSelection() bool {
	return m.editSelectionStart >= 0 && m.editSelectionEnd >= 0 && m.editSelectionStart != m.editSelectionEnd
}

func (m *model) getEditSelectionBounds() (int, int) {
	if m.editSelectionStart <= m.editSelectionEnd {
		return m.editSelectionStart, m.editSelectionEnd
	}
	return m.editSelectionEnd, m.editSelectionStart
}

func (m *model) deleteEditSelection() bool {
	if !m.hasEditSelection() {
		return false
	}
	start, end := m.getEditSelectionBounds()
	m.editText = m.editText[:start] + m.editText[end:]
	m.editCursorPos = start
	m.clearEditSelection()
	return true
}

func (m *model) getLineStartPos() int {
	if m.editCursorPos == 0 {
		return 0
	}

	for i := m.editCursorPos - 1; i >= 0; i-- {
		if m.editText[i] == '\n' {
			return i + 1
		}
	}
	return 0
}

func (m *model) getLineEndPos() int {

	for i := m.editCursorPos; i < len(m.editText); i++ {
		if m.editText[i] == '\n' {
			return i
		}
	}
	return len(m.editText)
}
