package tui

import (
	"strings"
	"unicode/utf8"
)

func runeLen(s string) int { return utf8.RuneCountInString(s) }

func splice(text string, start, end int, s string) string {
	r := []rune(text)
	return string(r[:start]) + s + string(r[end:])
}

func lineStartPos(text string, pos int) int {
	r := []rune(text)
	for i := pos - 1; i >= 0; i-- {
		if r[i] == '\n' {
			return i + 1
		}
	}
	return 0
}

func lineEndPos(text string, pos int) int {
	r := []rune(text)
	for i := pos; i < len(r); i++ {
		if r[i] == '\n' {
			return i
		}
	}
	return len(r)
}

func linearToRowCol(pos int, text string) (row, col int) {
	lines := strings.Split(text, "\n")
	current := 0
	for i, line := range lines {
		n := runeLen(line)
		if pos <= current+n {
			return i, pos - current
		}
		current += n + 1
	}
	return len(lines) - 1, runeLen(lines[len(lines)-1])
}

func rowColToLinear(row, col int, text string) int {
	lines := strings.Split(text, "\n")
	row = max(0, min(row, len(lines)-1))
	pos := 0
	for i := 0; i < row; i++ {
		pos += runeLen(lines[i]) + 1
	}
	return pos + max(0, min(col, runeLen(lines[row])))
}

func (m *model) editTarget() (editBoxID, editTextID, editTextX, editTextY int) {
	switch m.mode {
	case ModeEditing:
		return m.selectedBox, m.selectedText, CoordUnset, CoordUnset
	case ModeTextInput:
		return -1, -1, m.textInputX, m.textInputY
	case ModeTitleEdit:
		return m.selectedBox, -2, CoordUnset, CoordUnset
	}
	return -1, -1, CoordUnset, CoordUnset
}

func (m *model) editPosAt(screenX, screenY int) (int, bool) {
	originX, originY, ok := m.getCanvas().EditOrigin(m.editTarget())
	if !ok {
		return 0, false
	}
	panX, panY := m.getPanOffset()
	row := screenY - m.bufferBarOffset() + panY - originY
	col := screenX + panX - originX
	return rowColToLinear(row, col, m.editText), true
}

func (m *model) beginEdit(mode Mode, boxID, textID int, text string) {
	m.zPanMode = false
	m.mode = mode
	m.selectedBox, m.selectedText = boxID, textID
	m.editText, m.originalEditText = text, text
	m.editCursorPos = runeLen(text)
	m.clearEditSelection()
}

func (m *model) syncEditTarget() {
	canvas := m.getCanvas()
	switch m.mode {
	case ModeEditing:
		if m.selectedBox != -1 {
			canvas.SetBoxText(m.selectedBox, m.editText)
		} else {
			canvas.SetTextText(m.selectedText, m.editText)
		}
	case ModeTitleEdit:
		canvas.SetBoxTitle(m.selectedBox, m.editText)
	}
}

func (m *model) finishEdit() {
	canvas := m.getCanvas()
	changed := m.editText != m.originalEditText
	switch m.mode {
	case ModeEditing:
		if changed {
			typ, id := ActionEditBox, m.selectedBox
			if id == -1 {
				typ, id = ActionEditText, m.selectedText
			}
			m.recordAction(typ,
				EditData{ID: id, NewText: m.editText, OldText: m.originalEditText},
				EditData{ID: id, NewText: m.originalEditText, OldText: m.editText})
		}
	case ModeTextInput:
		if m.editText != "" {
			canvas.AddText(m.textInputX, m.textInputY, m.editText)
		}
	case ModeTitleEdit:
		canvas.SetBoxTitle(m.selectedBox, m.editText)
		if changed && m.selectedBox >= 0 && m.selectedBox < len(canvas.Boxes()) {
			m.recordAction(ActionEditTitle,
				EditData{ID: m.selectedBox, NewText: m.editText, OldText: m.originalEditText},
				EditData{ID: m.selectedBox, NewText: m.originalEditText, OldText: m.editText})
		}
	case ModeTooltipEdit:
		canvas.SetBoxTooltip(m.selectedBox, m.editText)
		if changed && m.selectedBox >= 0 && m.selectedBox < len(canvas.Boxes()) {
			m.recordAction(ActionEditTooltip,
				EditData{ID: m.selectedBox, NewText: m.editText, OldText: m.originalEditText},
				EditData{ID: m.selectedBox, NewText: m.originalEditText, OldText: m.editText})
		}
	}
	m.mode = ModeNormal
	m.editText, m.originalEditText = "", ""
	m.editCursorPos = 0
	m.clearEditSelection()
	m.selectedBox, m.selectedText = -1, -1
}

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

func (m *model) editSelectionBounds() (int, int) {
	return min(m.editSelectionStart, m.editSelectionEnd), max(m.editSelectionStart, m.editSelectionEnd)
}

func (m *model) deleteEditSelection() bool {
	if !m.hasEditSelection() {
		return false
	}
	start, end := m.editSelectionBounds()
	m.editText = splice(m.editText, start, end, "")
	m.editCursorPos = start
	m.clearEditSelection()
	return true
}
