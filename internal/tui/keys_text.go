package tui

import (
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) handleEditingKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Type == tea.KeyEscape:

		if m.selectedBox != -1 {
			m.getCanvas().SetBoxText(m.selectedBox, m.originalEditText)
		} else if m.selectedText != -1 {
			m.getCanvas().SetTextText(m.selectedText, m.originalEditText)
		}
		m.mode = ModeNormal
		m.editText = ""
		m.originalEditText = ""
		m.editCursorPos = 0
		m.editCursorRow = 0
		m.editCursorCol = 0
		m.clearEditSelection()
		m.selectedBox = -1
		m.selectedText = -1
		return m, nil
	case msg.Type == tea.KeyCtrlS:
		if m.selectedBox != -1 {

			editData := EditBoxData{ID: m.selectedBox, NewText: m.editText, OldText: m.originalEditText}
			inverseData := EditBoxData{ID: m.selectedBox, NewText: m.originalEditText, OldText: m.editText}
			m.recordAction(ActionEditBox, editData, inverseData)
		} else if m.selectedText != -1 {

			editData := EditTextData{ID: m.selectedText, NewText: m.editText, OldText: m.originalEditText}
			inverseData := EditTextData{ID: m.selectedText, NewText: m.originalEditText, OldText: m.editText}
			m.recordAction(ActionEditText, editData, inverseData)
		}
		m.mode = ModeNormal
		m.editText = ""
		m.originalEditText = ""
		m.editCursorPos = 0
		m.editCursorRow = 0
		m.editCursorCol = 0
		m.clearEditSelection()
		m.selectedBox = -1
		m.selectedText = -1
		return m, nil
	case msg.Type == tea.KeyCtrlV:

		if m.hasEditSelection() {
			m.deleteEditSelection()
		}

		clipText, err := readClipboardText()
		if err == nil && clipText != "" {

			m.editText = m.editText[:m.editCursorPos] + clipText + m.editText[m.editCursorPos:]
			m.editCursorPos += len([]rune(clipText))

			if m.selectedBox != -1 {
				m.getCanvas().SetBoxText(m.selectedBox, m.editText)
			} else if m.selectedText != -1 {
				m.getCanvas().SetTextText(m.selectedText, m.editText)
			}
		}
		return m, nil
	case msg.String() == "ctrl+v":

		if m.hasEditSelection() {
			m.deleteEditSelection()
		}

		clipText, err := readClipboardText()
		if err == nil && clipText != "" {

			m.editText = m.editText[:m.editCursorPos] + clipText + m.editText[m.editCursorPos:]
			m.editCursorPos += len([]rune(clipText))

			if m.selectedBox != -1 {
				m.getCanvas().SetBoxText(m.selectedBox, m.editText)
			} else if m.selectedText != -1 {
				m.getCanvas().SetTextText(m.selectedText, m.editText)
			}
		}
		return m, nil
	case msg.Type == tea.KeyHome:

		m.editCursorPos = m.getLineStartPos()
		m.clearEditSelection()
		m.syncCursorPositions()
		return m, nil
	case msg.Type == tea.KeyEnd:

		m.editCursorPos = m.getLineEndPos()
		m.clearEditSelection()
		m.syncCursorPositions()
		return m, nil
	case msg.Type == tea.KeyShiftLeft:

		if m.editSelectionStart < 0 {
			m.editSelectionStart = m.editCursorPos
			m.editSelectionEnd = m.editCursorPos
		}
		if m.editCursorPos > 0 {
			m.editCursorPos--
			m.editSelectionEnd = m.editCursorPos
		}
		m.syncCursorPositions()
		return m, nil
	case msg.Type == tea.KeyShiftRight:

		if m.editSelectionStart < 0 {
			m.editSelectionStart = m.editCursorPos
			m.editSelectionEnd = m.editCursorPos
		}
		if m.editCursorPos < len(m.editText) {
			m.editCursorPos++
			m.editSelectionEnd = m.editCursorPos
		}
		m.syncCursorPositions()
		return m, nil
	case msg.Type == tea.KeyShiftUp:

		if m.editSelectionStart < 0 {
			m.editSelectionStart = m.editCursorPos
			m.editSelectionEnd = m.editCursorPos
		}
		m.syncCursorPositions()
		if m.editCursorRow > 0 {
			m.editCursorRow--
			m.editCursorPos = m.cursorPosToLinear(m.editCursorRow, m.editCursorCol, m.editText)
			m.editSelectionEnd = m.editCursorPos
		}
		return m, nil
	case msg.Type == tea.KeyShiftDown:

		if m.editSelectionStart < 0 {
			m.editSelectionStart = m.editCursorPos
			m.editSelectionEnd = m.editCursorPos
		}
		m.syncCursorPositions()
		lines := strings.Split(m.editText, "\n")
		if m.editCursorRow < len(lines)-1 {
			m.editCursorRow++
			m.editCursorPos = m.cursorPosToLinear(m.editCursorRow, m.editCursorCol, m.editText)
			m.editSelectionEnd = m.editCursorPos
		}
		return m, nil
	case msg.String() == "left":
		m.clearEditSelection()
		if m.editCursorPos > 0 {
			m.editCursorPos--
		}
		m.syncCursorPositions()
		return m, nil
	case msg.String() == "right":
		m.clearEditSelection()
		if m.editCursorPos < len(m.editText) {
			m.editCursorPos++
		}
		m.syncCursorPositions()
		return m, nil
	case msg.String() == "up":

		m.clearEditSelection()
		m.syncCursorPositions()
		if m.editCursorRow > 0 {
			m.editCursorRow--
			m.editCursorPos = m.cursorPosToLinear(m.editCursorRow, m.editCursorCol, m.editText)
		}
		return m, nil
	case msg.String() == "down":

		m.clearEditSelection()
		m.syncCursorPositions()
		lines := strings.Split(m.editText, "\n")
		if m.editCursorRow < len(lines)-1 {
			m.editCursorRow++
			m.editCursorPos = m.cursorPosToLinear(m.editCursorRow, m.editCursorCol, m.editText)
		}
		return m, nil
	case msg.Type == tea.KeyEnter:

		if m.hasEditSelection() {
			m.deleteEditSelection()
		}
		m.editText = m.editText[:m.editCursorPos] + "\n" + m.editText[m.editCursorPos:]
		m.editCursorPos++

		if m.selectedBox != -1 {
			m.getCanvas().SetBoxText(m.selectedBox, m.editText)
		} else if m.selectedText != -1 {
			m.getCanvas().SetTextText(m.selectedText, m.editText)
		}
		return m, nil
	case msg.Type == tea.KeyBackspace:

		if m.hasEditSelection() {
			m.deleteEditSelection()

			if m.selectedBox != -1 {
				m.getCanvas().SetBoxText(m.selectedBox, m.editText)
			} else if m.selectedText != -1 {
				m.getCanvas().SetTextText(m.selectedText, m.editText)
			}
		} else if m.editCursorPos > 0 {
			m.editText = m.editText[:m.editCursorPos-1] + m.editText[m.editCursorPos:]
			m.editCursorPos--

			if m.selectedBox != -1 {
				m.getCanvas().SetBoxText(m.selectedBox, m.editText)
			} else if m.selectedText != -1 {
				m.getCanvas().SetTextText(m.selectedText, m.editText)
			}
		}
		return m, nil
	case msg.Type == tea.KeyDelete:

		if m.hasEditSelection() {
			m.deleteEditSelection()

			if m.selectedBox != -1 {
				m.getCanvas().SetBoxText(m.selectedBox, m.editText)
			} else if m.selectedText != -1 {
				m.getCanvas().SetTextText(m.selectedText, m.editText)
			}
		} else if m.editCursorPos < len(m.editText) {
			m.editText = m.editText[:m.editCursorPos] + m.editText[m.editCursorPos+1:]

			if m.selectedBox != -1 {
				m.getCanvas().SetBoxText(m.selectedBox, m.editText)
			} else if m.selectedText != -1 {
				m.getCanvas().SetTextText(m.selectedText, m.editText)
			}
		}
		return m, nil
	case msg.Type == tea.KeySpace:

		if m.hasEditSelection() {
			m.deleteEditSelection()
		}

		m.editText = m.editText[:m.editCursorPos] + " " + m.editText[m.editCursorPos:]
		m.editCursorPos++

		if m.selectedBox != -1 {
			m.getCanvas().SetBoxText(m.selectedBox, m.editText)
		} else if m.selectedText != -1 {
			m.getCanvas().SetTextText(m.selectedText, m.editText)
		}
		return m, nil
	default:

		if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 {

			if m.hasEditSelection() {
				m.deleteEditSelection()
			}
			runeStr := string(msg.Runes)
			m.editText = m.editText[:m.editCursorPos] + runeStr + m.editText[m.editCursorPos:]
			m.editCursorPos += len(msg.Runes)

			if m.selectedBox != -1 {
				m.getCanvas().SetBoxText(m.selectedBox, m.editText)
			} else if m.selectedText != -1 {
				m.getCanvas().SetTextText(m.selectedText, m.editText)
			}
		}
		return m, nil
	}
}

func (m model) handleTextInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Type == tea.KeyEscape:
		m.mode = ModeNormal
		m.textInputText = ""
		m.textInputCursorPos = 0
		return m, nil
	case msg.Type == tea.KeyCtrlS:
		if m.textInputText != "" {
			m.getCanvas().AddText(m.textInputX, m.textInputY, m.textInputText)
		}
		m.mode = ModeNormal
		m.textInputText = ""
		m.textInputCursorPos = 0
		return m, nil
	case msg.Type == tea.KeyCtrlV:

		clipText, err := readClipboardText()
		if err == nil && clipText != "" {

			m.textInputText = m.textInputText[:m.textInputCursorPos] + clipText + m.textInputText[m.textInputCursorPos:]
			m.textInputCursorPos += len([]rune(clipText))
		}
		return m, nil
	case msg.String() == "ctrl+v":

		clipText, err := readClipboardText()
		if err == nil && clipText != "" {

			m.textInputText = m.textInputText[:m.textInputCursorPos] + clipText + m.textInputText[m.textInputCursorPos:]
			m.textInputCursorPos += len([]rune(clipText))
		}
		return m, nil
	case msg.String() == "left":
		if m.textInputCursorPos > 0 {
			m.textInputCursorPos--
		}
		return m, nil
	case msg.String() == "right":
		if m.textInputCursorPos < len(m.textInputText) {
			m.textInputCursorPos++
		}
		return m, nil
	case msg.Type == tea.KeyEnter:
		m.textInputText = m.textInputText[:m.textInputCursorPos] + "\n" + m.textInputText[m.textInputCursorPos:]
		m.textInputCursorPos++
		return m, nil
	case msg.Type == tea.KeyBackspace:
		if m.textInputCursorPos > 0 {
			m.textInputText = m.textInputText[:m.textInputCursorPos-1] + m.textInputText[m.textInputCursorPos:]
			m.textInputCursorPos--
		}
		return m, nil
	case msg.Type == tea.KeyDelete:
		if m.textInputCursorPos < len(m.textInputText) {
			m.textInputText = m.textInputText[:m.textInputCursorPos] + m.textInputText[m.textInputCursorPos+1:]
		}
		return m, nil
	case msg.Type == tea.KeySpace:

		m.textInputText = m.textInputText[:m.textInputCursorPos] + " " + m.textInputText[m.textInputCursorPos:]
		m.textInputCursorPos++
		return m, nil
	default:

		if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 {
			runeStr := string(msg.Runes)
			m.textInputText = m.textInputText[:m.textInputCursorPos] + runeStr + m.textInputText[m.textInputCursorPos:]
			m.textInputCursorPos += len(msg.Runes)
		}
		return m, nil
	}
}

func (m model) handleBoxJumpKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Type == tea.KeyEscape:
		m.mode = ModeNormal
		m.boxJumpInput = ""
		return m, nil
	case msg.Type == tea.KeyEnter:

		if m.boxJumpInput != "" {
			boxNum, err := strconv.Atoi(m.boxJumpInput)
			if err == nil && boxNum >= 0 && boxNum < len(m.getCanvas().Boxes()) {
				box := m.getCanvas().Boxes()[boxNum]
				panX, panY := m.getPanOffset()

				m.cursorX = box.X + box.Width/2 - panX
				m.cursorY = box.Y + box.Height/2 - panY
				m.ensureCursorInBounds()
			}
		}
		m.mode = ModeNormal
		m.boxJumpInput = ""
		return m, nil
	case msg.Type == tea.KeyBackspace:
		if len(m.boxJumpInput) > 0 {
			m.boxJumpInput = m.boxJumpInput[:len(m.boxJumpInput)-1]
		}
		return m, nil
	default:

		if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 {
			runeStr := string(msg.Runes)

			if len(runeStr) == 1 && runeStr[0] >= '0' && runeStr[0] <= '9' {
				m.boxJumpInput += runeStr
			}
		}
		return m, nil
	}
}

func (m model) handleTitleEditKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Type == tea.KeyEscape:

		if m.titleEditBoxID != -1 && m.titleEditBoxID < len(m.getCanvas().Boxes()) {
			m.getCanvas().Boxes()[m.titleEditBoxID].Title = m.originalTitleText
			m.getCanvas().Boxes()[m.titleEditBoxID].UpdateSize()
		}
		m.mode = ModeNormal
		m.titleEditText = ""
		m.titleEditCursorPos = 0
		m.titleEditCursorRow = 0
		m.titleEditCursorCol = 0
		m.titleEditBoxID = -1
		return m, nil
	case msg.Type == tea.KeyCtrlS:

		if m.titleEditBoxID != -1 && m.titleEditBoxID < len(m.getCanvas().Boxes()) {
			oldTitle := m.originalTitleText
			newTitle := m.titleEditText

			m.getCanvas().Boxes()[m.titleEditBoxID].Title = newTitle
			m.getCanvas().Boxes()[m.titleEditBoxID].UpdateSize()

			editData := EditTitleData{BoxID: m.titleEditBoxID, NewTitle: newTitle, OldTitle: oldTitle}
			inverseData := EditTitleData{BoxID: m.titleEditBoxID, NewTitle: oldTitle, OldTitle: newTitle}
			m.recordAction(ActionEditTitle, editData, inverseData)
		}
		m.mode = ModeNormal
		m.titleEditText = ""
		m.titleEditCursorPos = 0
		m.titleEditCursorRow = 0
		m.titleEditCursorCol = 0
		m.titleEditBoxID = -1
		return m, nil
	case msg.Type == tea.KeyCtrlV:

		clipText, err := readClipboardText()
		if err == nil && clipText != "" {
			m.titleEditText = m.titleEditText[:m.titleEditCursorPos] + clipText + m.titleEditText[m.titleEditCursorPos:]
			m.titleEditCursorPos += len([]rune(clipText))

			if m.titleEditBoxID != -1 && m.titleEditBoxID < len(m.getCanvas().Boxes()) {
				m.getCanvas().Boxes()[m.titleEditBoxID].Title = m.titleEditText
				m.getCanvas().Boxes()[m.titleEditBoxID].UpdateSize()
			}
		}
		return m, nil
	case msg.String() == "ctrl+v":

		clipText, err := readClipboardText()
		if err == nil && clipText != "" {
			m.titleEditText = m.titleEditText[:m.titleEditCursorPos] + clipText + m.titleEditText[m.titleEditCursorPos:]
			m.titleEditCursorPos += len([]rune(clipText))

			if m.titleEditBoxID != -1 && m.titleEditBoxID < len(m.getCanvas().Boxes()) {
				m.getCanvas().Boxes()[m.titleEditBoxID].Title = m.titleEditText
				m.getCanvas().Boxes()[m.titleEditBoxID].UpdateSize()
			}
		}
		return m, nil
	case msg.String() == "left":
		if m.titleEditCursorPos > 0 {
			m.titleEditCursorPos--
		}
		m.titleEditCursorRow, m.titleEditCursorCol = m.linearToCursorPos(m.titleEditCursorPos, m.titleEditText)
		return m, nil
	case msg.String() == "right":
		if m.titleEditCursorPos < len(m.titleEditText) {
			m.titleEditCursorPos++
		}
		m.titleEditCursorRow, m.titleEditCursorCol = m.linearToCursorPos(m.titleEditCursorPos, m.titleEditText)
		return m, nil
	case msg.String() == "up":

		m.titleEditCursorRow, m.titleEditCursorCol = m.linearToCursorPos(m.titleEditCursorPos, m.titleEditText)
		if m.titleEditCursorRow > 0 {
			m.titleEditCursorRow--
			m.titleEditCursorPos = m.cursorPosToLinear(m.titleEditCursorRow, m.titleEditCursorCol, m.titleEditText)
		}
		return m, nil
	case msg.String() == "down":

		m.titleEditCursorRow, m.titleEditCursorCol = m.linearToCursorPos(m.titleEditCursorPos, m.titleEditText)
		lines := strings.Split(m.titleEditText, "\n")
		if m.titleEditCursorRow < len(lines)-1 {
			m.titleEditCursorRow++
			m.titleEditCursorPos = m.cursorPosToLinear(m.titleEditCursorRow, m.titleEditCursorCol, m.titleEditText)
		}
		return m, nil
	case msg.Type == tea.KeyEnter:
		m.titleEditText = m.titleEditText[:m.titleEditCursorPos] + "\n" + m.titleEditText[m.titleEditCursorPos:]
		m.titleEditCursorPos++

		if m.titleEditBoxID != -1 && m.titleEditBoxID < len(m.getCanvas().Boxes()) {
			m.getCanvas().Boxes()[m.titleEditBoxID].Title = m.titleEditText
			m.getCanvas().Boxes()[m.titleEditBoxID].UpdateSize()
		}
		return m, nil
	case msg.Type == tea.KeyBackspace:
		if m.titleEditCursorPos > 0 {
			m.titleEditText = m.titleEditText[:m.titleEditCursorPos-1] + m.titleEditText[m.titleEditCursorPos:]
			m.titleEditCursorPos--

			if m.titleEditBoxID != -1 && m.titleEditBoxID < len(m.getCanvas().Boxes()) {
				m.getCanvas().Boxes()[m.titleEditBoxID].Title = m.titleEditText
				m.getCanvas().Boxes()[m.titleEditBoxID].UpdateSize()
			}
		}
		return m, nil
	case msg.Type == tea.KeyDelete:
		if m.titleEditCursorPos < len(m.titleEditText) {
			m.titleEditText = m.titleEditText[:m.titleEditCursorPos] + m.titleEditText[m.titleEditCursorPos+1:]

			if m.titleEditBoxID != -1 && m.titleEditBoxID < len(m.getCanvas().Boxes()) {
				m.getCanvas().Boxes()[m.titleEditBoxID].Title = m.titleEditText
				m.getCanvas().Boxes()[m.titleEditBoxID].UpdateSize()
			}
		}
		return m, nil
	case msg.Type == tea.KeySpace:

		m.titleEditText = m.titleEditText[:m.titleEditCursorPos] + " " + m.titleEditText[m.titleEditCursorPos:]
		m.titleEditCursorPos++

		if m.titleEditBoxID != -1 && m.titleEditBoxID < len(m.getCanvas().Boxes()) {
			m.getCanvas().Boxes()[m.titleEditBoxID].Title = m.titleEditText
			m.getCanvas().Boxes()[m.titleEditBoxID].UpdateSize()
		}
		return m, nil
	default:

		if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 {
			runeStr := string(msg.Runes)
			m.titleEditText = m.titleEditText[:m.titleEditCursorPos] + runeStr + m.titleEditText[m.titleEditCursorPos:]
			m.titleEditCursorPos += len([]rune(runeStr))

			if m.titleEditBoxID != -1 && m.titleEditBoxID < len(m.getCanvas().Boxes()) {
				m.getCanvas().Boxes()[m.titleEditBoxID].Title = m.titleEditText
				m.getCanvas().Boxes()[m.titleEditBoxID].UpdateSize()
			}
		}
		return m, nil
	}
}
