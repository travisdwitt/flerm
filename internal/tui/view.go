package tui

import (
	"fmt"
	"path/filepath"
	"strings"
)

func (m *model) renderBufferBar(width int) string {
	if len(m.buffers) <= 1 {
		return strings.Repeat(" ", width)
	}
	var bar strings.Builder
	bar.WriteString("Open Charts: ")
	for i, buf := range m.buffers {
		if i > 0 {
			bar.WriteString(" | ")
		}
		var bufName string
		if buf.filename != "" {
			name := filepath.Base(buf.filename)
			if strings.HasSuffix(strings.ToLower(name), ".sav") {
				name = name[:len(name)-4]
			}
			bufName = name
		} else {
			bufName = fmt.Sprintf("Buffer %d", i+1)
		}
		if i == m.currentBufferIndex {

			bar.WriteString("\033[32m[\033[0m")
			bar.WriteString(bufName)
			bar.WriteString("\033[32m]\033[0m")
		} else {
			bar.WriteString(bufName)
		}
	}
	currentLen := bar.Len()

	visibleLen := currentLen - (2 * 9)
	if visibleLen < width {
		bar.WriteString(strings.Repeat(" ", width-visibleLen))
	} else {
		return bar.String()[:width]
	}
	return bar.String()
}

func (m *model) updateTooltip() {

	if m.mode != ModeNormal {
		m.showTooltip = false
		m.tooltipBoxID = -1
		return
	}

	panX, panY := m.getPanOffset()
	worldX, worldY := m.cursorX+panX, m.cursorY+panY

	if boxID := m.getCanvas().GetBoxAt(worldX, worldY); boxID != -1 {
		box := &m.getCanvas().Boxes()[boxID]
		if box.IsTextTruncated() {

			m.showTooltip = true
			m.tooltipText = box.GetText()
			m.tooltipX = m.cursorX
			m.tooltipY = m.cursorY
			m.tooltipBoxID = boxID
		} else {
			m.showTooltip = false
			m.tooltipBoxID = -1
		}
	} else {
		m.showTooltip = false
		m.tooltipBoxID = -1
	}
}

func (m model) View() string {
	if m.help && m.mode != ModeStartup {
		return m.helpView()
	}

	if m.mode == ModeStartup {
		return m.renderStartupMenu()
	}

	if m.mode == ModeFileInput && m.fileOp == FileOpOpen {
		return m.renderFileMenu()
	}

	var selectedBox int = -1
	if m.mode == ModeResize || m.mode == ModeMove {
		selectedBox = m.selectedBox
	}

	renderWidth := m.width
	if renderWidth < 1 {
		renderWidth = 1
	}

	showBufferBar := m.mode != ModeStartup && len(m.buffers) > 1
	var renderHeight int
	if showBufferBar {
		renderHeight = m.height - 2
	} else {
		renderHeight = m.height - 1
	}
	if renderHeight < 1 {
		renderHeight = 1
	}

	var previewFromX, previewFromY, previewToX, previewToY int = -1, -1, -1, -1
	var previewWaypoints []point
	if m.connectionFrom != -1 || m.connectionFromLine != -1 {
		previewFromX = m.connectionFromX
		previewFromY = m.connectionFromY
		previewWaypoints = m.connectionWaypoints

		buf := m.getCurrentBuffer()
		panX, panY := 0, 0
		if buf != nil {
			panX, panY = buf.panX, buf.panY
		}
		previewToX = m.cursorX + panX
		previewToY = m.cursorY + panY
	}

	buf := m.getCurrentBuffer()
	panX, panY := 0, 0
	if buf != nil {
		panX, panY = buf.panX, buf.panY
	}

	cursorX := m.cursorX
	cursorY := m.cursorY

	if cursorY >= renderHeight {
		cursorY = renderHeight - 1
	}
	if cursorY < 0 {
		cursorY = 0
	}
	if cursorX >= renderWidth {
		cursorX = renderWidth - 1
	}
	if cursorX < 0 {
		cursorX = 0
	}

	showCursor := (m.mode != ModeStartup && m.mode != ModeFileInput && m.mode != ModeEditing && m.mode != ModeTextInput && m.mode != ModeTitleEdit)

	var editBoxID, editTextID int = -1, -1
	var editCursorPos int = 0
	var editText string = ""
	var editTextX, editTextY int = -1, -1
	if m.mode == ModeEditing {
		editBoxID = m.selectedBox
		editTextID = m.selectedText
		editCursorPos = m.editCursorPos
		editText = m.editText
	} else if m.mode == ModeTextInput {

		editTextID = -1
		editCursorPos = m.textInputCursorPos
		editText = m.textInputText
		editTextX = m.textInputX
		editTextY = m.textInputY
	} else if m.mode == ModeTitleEdit {

		editBoxID = m.titleEditBoxID
		editCursorPos = m.titleEditCursorPos
		editText = m.titleEditText

		editTextID = -2
	}

	selectionStartX, selectionStartY := -1, -1
	selectionEndX, selectionEndY := -1, -1
	if m.mode == ModeMultiSelect && m.selectionStartX >= 0 && m.selectionStartY >= 0 {
		selectionStartX = m.selectionStartX
		selectionStartY = m.selectionStartY
		selectionEndX = m.cursorX + panX
		selectionEndY = m.cursorY + panY
	}

	showBoxNumbers := (m.mode == ModeBoxJump)

	editSelStart, editSelEnd := -1, -1
	if m.mode == ModeEditing && m.hasEditSelection() {
		editSelStart, editSelEnd = m.editSelectionStart, m.editSelectionEnd
	}

	renderResult := m.getCanvas().RenderRaw(renderWidth, renderHeight, selectedBox, previewFromX, previewFromY, previewWaypoints, previewToX, previewToY, panX, panY, cursorX, cursorY, showCursor, editBoxID, editTextID, editCursorPos, editText, editTextX, editTextY, selectionStartX, selectionStartY, selectionEndX, selectionEndY, showBoxNumbers, editSelStart, editSelEnd)

	m.overlaySelection(renderResult, panX, panY)

	if m.showTooltip && m.tooltipText != "" {
		m.overlayTooltipOnRenderResult(renderResult)
	}

	if m.mode == ModeContextMenu {
		m.overlayContextMenu(renderResult)
	}

	canvas := renderResult.ApplyColors()

	var result strings.Builder

	if showBufferBar {
		bufferBar := m.renderBufferBar(renderWidth)
		result.WriteString(bufferBar)
		result.WriteString("\n")
	}

	for i, line := range canvas {
		result.WriteString(line)
		if i < len(canvas)-1 {
			result.WriteString("\n")
		}
	}

	var statusLine string
	switch m.mode {
	case ModeStartup:
		statusLine = "Press 'n' for new flowchart, 'o' to open existing, or 'q' to quit"
	case ModeEditing:
		displayText := strings.ReplaceAll(m.editText, "\n", " ")
		cursorPos := m.editCursorPos
		if cursorPos > len(displayText) {
			cursorPos = len(displayText)
		}
		var cursorDisplay string
		if m.hasEditSelection() {

			start, end := m.getEditSelectionBounds()

			runes := []rune(displayText)
			if start > len(runes) {
				start = len(runes)
			}
			if end > len(runes) {
				end = len(runes)
			}
			before := string(runes[:start])
			selected := string(runes[start:end])
			after := string(runes[end:])
			cursorDisplay = before + "[" + selected + "]" + after
		} else if len(displayText) == 0 {
			cursorDisplay = "█"
		} else if cursorPos >= len(displayText) {
			cursorDisplay = displayText + "█"
		} else {

			runes := []rune(displayText)
			runes[cursorPos] = '█'
			cursorDisplay = string(runes)
		}
		selectionHint := ""
		if m.hasEditSelection() {
			start, end := m.getEditSelectionBounds()
			selectionHint = fmt.Sprintf(" | %d chars selected", end-start)
		}
		if m.selectedBox != -1 {
			statusLine = fmt.Sprintf("Mode: EDIT | Box %d | Text: %s%s | Home/End, Shift+←→↑↓=select, Ctrl+S=save", m.selectedBox, cursorDisplay, selectionHint)
		} else if m.selectedText != -1 {
			statusLine = fmt.Sprintf("Mode: EDIT | Text %d | Text: %s%s | Home/End, Shift+←→↑↓=select, Ctrl+S=save", m.selectedText, cursorDisplay, selectionHint)
		} else {
			statusLine = fmt.Sprintf("Mode: EDIT | Text: %s%s | Home/End, Shift+←→↑↓=select, Ctrl+S=save", cursorDisplay, selectionHint)
		}
	case ModeTextInput:
		displayText := strings.ReplaceAll(m.textInputText, "\n", " ")
		cursorPos := m.textInputCursorPos
		if cursorPos > len(displayText) {
			cursorPos = len(displayText)
		}
		var cursorDisplay string
		if len(displayText) == 0 {
			cursorDisplay = "█"
		} else if cursorPos >= len(displayText) {
			cursorDisplay = displayText + "█"
		} else {

			runes := []rune(displayText)
			runes[cursorPos] = '█'
			cursorDisplay = string(runes)
		}
		statusLine = fmt.Sprintf("Mode: TEXT | Text: %s | ←/→=move cursor, Enter=newline, Ctrl+S=save, Esc=cancel", cursorDisplay)
	case ModeResize:
		statusLine = fmt.Sprintf("Mode: RESIZE | Box %d | hjkl/arrows=resize, Enter=finish, Esc=cancel", m.selectedBox)
	case ModeMove:
		if len(m.selectedBoxes) > 0 || len(m.selectedTexts) > 0 || len(m.selectedConnections) > 0 || len(m.originalHighlights) > 0 {
			boxCount := len(m.selectedBoxes)
			textCount := len(m.selectedTexts)
			connCount := len(m.selectedConnections)
			parts := []string{}
			if boxCount > 0 {
				parts = append(parts, fmt.Sprintf("%d boxes", boxCount))
			}
			if textCount > 0 {
				parts = append(parts, fmt.Sprintf("%d texts", textCount))
			}
			if connCount > 0 {
				parts = append(parts, fmt.Sprintf("%d connections", connCount))
			}
			statusLine = fmt.Sprintf("Mode: MOVE | %s | hjkl/arrows=move, Enter=finish, Esc=cancel", strings.Join(parts, ", "))
		} else if m.selectedBox != -1 {
			statusLine = fmt.Sprintf("Mode: MOVE | Box %d | hjkl/arrows=move, Enter=finish, Esc=cancel", m.selectedBox)
		} else if m.selectedText != -1 {
			statusLine = fmt.Sprintf("Mode: MOVE | Text %d | hjkl/arrows=move, Enter=finish, Esc=cancel", m.selectedText)
		} else {
			statusLine = "Mode: MOVE | hjkl/arrows=move, Enter=finish, Esc=cancel"
		}
	case ModeMultiSelect:
		statusLine = "Mode: MULTI-SELECT | hjkl/arrows=draw selection, Enter=select and move, Esc=cancel"
	case ModeFileInput:
		var opStr string
		switch m.fileOp {
		case FileOpSave:
			opStr = "Save"
		case FileOpOpen:
			opStr = "Open"
		case FileOpSavePNG:
			opStr = "Export PNG"
		case FileOpSaveVisualTXT:
			opStr = "Export Visual TXT"
		}
		if m.errorMessage != "" {
			if m.fileOp == FileOpOpen {
				statusLine = fmt.Sprintf("Mode: FILE | ERROR: %s | %s filename: %s | ↑/↓=navigate, Enter=retry, Esc=cancel", m.errorMessage, opStr, m.filename)
			} else {
				statusLine = fmt.Sprintf("Mode: FILE | ERROR: %s | %s filename: %s | Enter=retry, Esc=cancel", m.errorMessage, opStr, m.filename)
			}
		} else {
			if m.fileOp == FileOpOpen {
				if m.showingDeleteConfirm {
					chartName := ""
					if m.confirmFileIndex >= 0 && m.confirmFileIndex < len(m.fileList) {
						chartName = m.fileList[m.confirmFileIndex]
						if strings.HasSuffix(strings.ToLower(chartName), ".sav") {
							chartName = chartName[:len(chartName)-4]
						}
					}
					statusLine = fmt.Sprintf("Mode: FILE | Are you sure you want to delete %s? (y/n)", chartName)
				} else {
					statusLine = fmt.Sprintf("Mode: FILE | %s filename: %s | ↑/↓=navigate list, d=delete, Type=enter name, Enter=confirm, Esc=cancel", opStr, m.filename)
				}
			} else {
				statusLine = fmt.Sprintf("Mode: FILE | %s filename: %s | Enter=confirm, Esc=cancel", opStr, m.filename)
			}
		}
	case ModeConfirm:
		var message string
		switch m.confirmAction {
		case ConfirmDeleteBox:
			message = "Delete this box? (y/n)"
		case ConfirmDeleteText:
			message = "Delete this text? (y/n)"
		case ConfirmDeleteConnection:
			message = "Delete this connection? (y/n)"
		case ConfirmDeleteHighlight:
			message = "Remove highlight? (y/n)"
		case ConfirmQuit:
			message = "Quit Flerm? (y/n)"
		case ConfirmNewChart:
			message = "Create new chart? Unsaved changes will be lost. (y/n)"
		case ConfirmCloseBuffer:
			message = "Close current buffer? Unsaved changes will be lost. (y/n)"
		case ConfirmOverwriteFile:
			message = fmt.Sprintf("File %s already exists. Overwrite? (y/n)", m.filename)
		case ConfirmChooseExportType:
			message = "Export as PNG (p) or Visual TXT (t)? Press Esc to cancel"
		}
		statusLine = fmt.Sprintf("Mode: CONFIRM | %s", message)
	case ModeContextMenu:
		statusLine = "Mode: MENU | ↑/↓ or hover=navigate, →/Enter=open submenu, ←=back, click=select, Esc/right-click=cancel"
	case ModeBoxJump:
		statusLine = fmt.Sprintf("Mode: BOX JUMP | Enter box number: %s | Enter=jump, Esc=cancel", m.boxJumpInput)
	case ModeTitleEdit:
		displayText := strings.ReplaceAll(m.titleEditText, "\n", " ")
		cursorPos := m.titleEditCursorPos
		if cursorPos > len(displayText) {
			cursorPos = len(displayText)
		}
		var cursorDisplay string
		if len(displayText) == 0 {
			cursorDisplay = "█"
		} else if cursorPos >= len(displayText) {
			cursorDisplay = displayText + "█"
		} else {

			runes := []rune(displayText)
			runes[cursorPos] = '█'
			cursorDisplay = string(runes)
		}
		statusLine = fmt.Sprintf("Mode: TITLE EDIT | Title: %s | ←/→/↑/↓=move cursor, Enter=newline, Ctrl+S=save, Esc=cancel", cursorDisplay)
	default:
		modeStr := m.modeString()
		if m.zPanMode {
			modeStr = "PAN"
		}
		if m.highlightMode {
			modeStr = "HIGHLIGHT"
		}
		colorNames := []string{"Gray", "Red", "Green", "Yellow", "Blue", "Magenta", "Cyan", "White"}
		status := fmt.Sprintf("Mode: %s | Cursor: (%d,%d)", modeStr, m.cursorX, m.cursorY)
		if m.highlightMode {
			status += fmt.Sprintf(" | Color: %s (%d/8)", colorNames[m.selectedColor], m.selectedColor+1)
		}
		if m.connectionFrom != -1 {
			status += fmt.Sprintf(" | Connection from box %d (select target)", m.connectionFrom)
		} else if m.connectionFromLine != -1 {
			status += " | Connection from line (select target)"
		}
		if m.selectedBox != -1 {
			status += fmt.Sprintf(" | Selected: Box %d", m.selectedBox)
		}
		if m.successMessage != "" {
			status += fmt.Sprintf(" | %s", m.successMessage)
		}
		if m.errorMessage != "" {
			status += fmt.Sprintf(" | ERROR: %s", m.errorMessage)
		} else if m.successMessage == "" {
			status += " | ? for help | q to quit"
		}
		statusLine = status
	}

	if m.mode != ModeStartup && !(m.mode == ModeFileInput && m.fileOp == FileOpOpen) {
		result.WriteString("\n")
		result.WriteString(statusLine)
	}

	return result.String()
}

type tooltipCharInfo struct {
	char        rune
	origCharIdx int
}

func (m model) overlayTooltipOnRenderResult(r *RenderResult) {
	if !m.showTooltip || m.tooltipText == "" {
		return
	}

	var charHighlights map[int]int
	if m.tooltipBoxID >= 0 {
		charHighlights = m.getCanvas().GetBoxContentHighlights(m.tooltipBoxID)
	}

	maxWidth := 45
	minWidth := 15

	longestWord := 0
	for _, word := range strings.Fields(m.tooltipText) {
		if len(word) > longestWord {
			longestWord = len(word)
		}
	}

	tooltipWidth := maxWidth
	if longestWord+4 < maxWidth {
		tooltipWidth = longestWord + 4
	}
	if tooltipWidth < minWidth {
		tooltipWidth = minWidth
	}

	contentWidth := tooltipWidth - 4

	type wordInfo struct {
		text     string
		startIdx int
	}

	wordsWithPos := []wordInfo{}
	origText := m.tooltipText
	inWord := false
	wordStart := 0

	for i, ch := range origText {
		isSpace := ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
		if !isSpace {
			if !inWord {
				inWord = true
				wordStart = i
			}
		} else {
			if inWord {

				wordsWithPos = append(wordsWithPos, wordInfo{
					text:     origText[wordStart:i],
					startIdx: wordStart,
				})
				inWord = false
			}
		}
	}

	if inWord {
		wordsWithPos = append(wordsWithPos, wordInfo{
			text:     origText[wordStart:],
			startIdx: wordStart,
		})
	}

	if len(wordsWithPos) == 0 {
		return
	}

	tooltipLines := [][]tooltipCharInfo{}

	topBorder := []tooltipCharInfo{}
	topBorder = append(topBorder, tooltipCharInfo{'┌', -1})
	for i := 0; i < tooltipWidth-2; i++ {
		topBorder = append(topBorder, tooltipCharInfo{'─', -1})
	}
	topBorder = append(topBorder, tooltipCharInfo{'┐', -1})
	tooltipLines = append(tooltipLines, topBorder)

	type lineBuilder struct {
		chars []tooltipCharInfo
		width int
	}
	currentLineBuilder := lineBuilder{chars: []tooltipCharInfo{}, width: 0}
	var contentLines [][]tooltipCharInfo

	for _, w := range wordsWithPos {
		wordLen := len([]rune(w.text))

		if currentLineBuilder.width+wordLen+1 <= contentWidth || currentLineBuilder.width == 0 {

			if currentLineBuilder.width > 0 {
				currentLineBuilder.chars = append(currentLineBuilder.chars, tooltipCharInfo{' ', -1})
				currentLineBuilder.width++
			}

			for i, ch := range w.text {
				currentLineBuilder.chars = append(currentLineBuilder.chars, tooltipCharInfo{ch, w.startIdx + i})
			}
			currentLineBuilder.width += wordLen
		} else {

			contentLines = append(contentLines, currentLineBuilder.chars)
			currentLineBuilder = lineBuilder{chars: []tooltipCharInfo{}, width: 0}

			for i, ch := range w.text {
				currentLineBuilder.chars = append(currentLineBuilder.chars, tooltipCharInfo{ch, w.startIdx + i})
			}
			currentLineBuilder.width = wordLen
		}
	}

	if currentLineBuilder.width > 0 {
		contentLines = append(contentLines, currentLineBuilder.chars)
	}

	for _, lineChars := range contentLines {
		fullLine := []tooltipCharInfo{}
		fullLine = append(fullLine, tooltipCharInfo{'│', -1})
		fullLine = append(fullLine, tooltipCharInfo{' ', -1})

		fullLine = append(fullLine, lineChars...)

		lineContentWidth := len(lineChars)
		paddingNeeded := contentWidth - lineContentWidth
		for i := 0; i < paddingNeeded; i++ {
			fullLine = append(fullLine, tooltipCharInfo{' ', -1})
		}

		fullLine = append(fullLine, tooltipCharInfo{' ', -1})
		fullLine = append(fullLine, tooltipCharInfo{'│', -1})
		tooltipLines = append(tooltipLines, fullLine)
	}

	bottomBorder := []tooltipCharInfo{}
	bottomBorder = append(bottomBorder, tooltipCharInfo{'└', -1})
	for i := 0; i < tooltipWidth-2; i++ {
		bottomBorder = append(bottomBorder, tooltipCharInfo{'─', -1})
	}
	bottomBorder = append(bottomBorder, tooltipCharInfo{'┘', -1})
	tooltipLines = append(tooltipLines, bottomBorder)

	tooltipX := m.tooltipX + 6
	tooltipY := m.tooltipY

	if tooltipX+tooltipWidth >= r.Width {
		tooltipX = m.tooltipX - tooltipWidth - 3
		if tooltipX < 0 {
			tooltipX = 1
		}
	}

	if tooltipY+len(tooltipLines) >= r.Height {
		tooltipY = m.tooltipY - len(tooltipLines) - 1
		if tooltipY < 0 {
			tooltipY = 0
		}
	}

	for i, tooltipLine := range tooltipLines {
		lineIdx := tooltipY + i
		if lineIdx >= 0 && lineIdx < r.Height {
			for j, charInfo := range tooltipLine {
				posX := tooltipX + j
				if posX >= 0 && posX < r.Width {

					r.Canvas[lineIdx][posX] = charInfo.char

					if charInfo.origCharIdx >= 0 && charHighlights != nil {
						if colorIdx, exists := charHighlights[charInfo.origCharIdx]; exists {
							r.ColorMap[lineIdx][posX] = colorIdx
						} else {
							r.ColorMap[lineIdx][posX] = -1
						}
					} else {

						r.ColorMap[lineIdx][posX] = -1
					}
				}
			}
		}
	}
}

func (m model) renderStartupMenu() string {
	logo := []string{
		"    ___ __                      ",
		"  .'  _|  |.-----.----.--------.",
		"  |   _|  ||  -__|   _|        |",
		"  |__| |__||_____|__| |__|__|__|",
	}

	menuItems := []string{
		"  n: New",
		"  o: Open",
		"  q: Quit",
	}

	logoWidth := len(logo[0])
	menuWidth := 0
	for _, item := range menuItems {
		if len(item) > menuWidth {
			menuWidth = len(item)
		}
	}

	contentWidth := logoWidth
	if menuWidth > contentWidth {
		contentWidth = menuWidth
	}

	boxWidth := contentWidth + 4
	boxHeight := len(logo) + len(menuItems) + 6

	centerX := m.width/2 - boxWidth/2
	centerY := m.height/2 - boxHeight/2

	var result strings.Builder

	for y := 0; y < m.height; y++ {
		for x := 0; x < m.width; x++ {
			if y < centerY || y >= centerY+boxHeight || x < centerX || x >= centerX+boxWidth {
				result.WriteString(" ")
			} else {
				relY := y - centerY
				relX := x - centerX

				if relY == 0 {
					if relX == 0 {
						result.WriteString("┌")
					} else if relX == boxWidth-1 {
						result.WriteString("┐")
					} else {
						result.WriteString("─")
					}
				} else if relY == boxHeight-1 {
					if relX == 0 {
						result.WriteString("└")
					} else if relX == boxWidth-1 {
						result.WriteString("┘")
					} else {
						result.WriteString("─")
					}
				} else if relX == 0 || relX == boxWidth-1 {
					result.WriteString("│")
				} else if relY == 1 {
					result.WriteString(" ")
				} else if relY >= 2 && relY < 2+len(logo) {
					logoLineIdx := relY - 2
					logoX := relX - 1 - (contentWidth-logoWidth)/2
					if logoX >= 0 && logoX < len(logo[logoLineIdx]) {
						result.WriteString("\033[32m" + string(logo[logoLineIdx][logoX]) + "\033[0m")
					} else {
						result.WriteString(" ")
					}
				} else if relY == 2+len(logo) || relY == 3+len(logo) {
					result.WriteString(" ")
				} else if relY >= 4+len(logo) && relY < 4+len(logo)+len(menuItems) {
					menuLineIdx := relY - 4 - len(logo)
					menuX := relX - 1
					if menuX >= 0 && menuX < len(menuItems[menuLineIdx]) {
						result.WriteString(string(menuItems[menuLineIdx][menuX]))
					} else {
						result.WriteString(" ")
					}
				} else {
					result.WriteString(" ")
				}
			}
		}
		if y < m.height-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

func (m model) renderFileMenu() string {
	title := "Select a saved chart:"

	var menuItems []string
	if len(m.fileList) == 0 {
		menuItems = []string{"  (No .sav files found in current directory)"}
	} else {
		for i, file := range m.fileList {
			displayName := file
			if strings.HasSuffix(strings.ToLower(file), ".sav") {
				displayName = file[:len(file)-4]
			}

			if i == m.selectedFileIndex {
				menuItems = append(menuItems, "> "+displayName)
			} else {
				menuItems = append(menuItems, "  "+displayName)
			}
		}
	}

	if m.showingDeleteConfirm && m.confirmFileIndex >= 0 && m.confirmFileIndex < len(m.fileList) {
		chartName := m.fileList[m.confirmFileIndex]
		if strings.HasSuffix(strings.ToLower(chartName), ".sav") {
			chartName = chartName[:len(chartName)-4]
		}
		menuItems = append(menuItems, "")
		menuItems = append(menuItems, fmt.Sprintf("  Are you sure you want to delete %s? (Y/N)", chartName))
	} else if len(m.fileList) > 0 {
		menuItems = append(menuItems, "")
		menuItems = append(menuItems, "  Enter: Open  d: Delete  Esc: Cancel")
	}

	contentWidth := len(title)
	for _, item := range menuItems {
		if len(item) > contentWidth {
			contentWidth = len(item)
		}
	}

	boxWidth := contentWidth + 4
	boxHeight := len(menuItems) + 4

	centerX := m.width/2 - boxWidth/2
	centerY := m.height/2 - boxHeight/2

	var result strings.Builder

	for y := 0; y < m.height; y++ {
		for x := 0; x < m.width; x++ {
			if y < centerY || y >= centerY+boxHeight || x < centerX || x >= centerX+boxWidth {
				result.WriteString(" ")
			} else {
				relY := y - centerY
				relX := x - centerX

				if relY == 0 {
					if relX == 0 {
						result.WriteString("┌")
					} else if relX == boxWidth-1 {
						result.WriteString("┐")
					} else {
						result.WriteString("─")
					}
				} else if relY == boxHeight-1 {
					if relX == 0 {
						result.WriteString("└")
					} else if relX == boxWidth-1 {
						result.WriteString("┘")
					} else {
						result.WriteString("─")
					}
				} else if relX == 0 || relX == boxWidth-1 {
					result.WriteString("│")
				} else if relY == 1 {
					titleX := relX - 1 - (contentWidth-len(title))/2
					if titleX >= 0 && titleX < len(title) {
						result.WriteString(string(title[titleX]))
					} else {
						result.WriteString(" ")
					}
				} else if relY == 2 {
					result.WriteString("─")
				} else if relY >= 3 && relY < 3+len(menuItems) {
					itemIdx := relY - 3
					itemX := relX - 1
					if itemX >= 0 && itemX < len(menuItems[itemIdx]) {
						result.WriteString(string(menuItems[itemIdx][itemX]))
					} else {
						result.WriteString(" ")
					}
				} else {
					result.WriteString(" ")
				}
			}
		}
		if y < m.height-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

func (m model) modeString() string {
	switch m.mode {
	case ModeStartup:
		return "STARTUP"
	case ModeNormal:
		return "NORMAL"
	case ModeEditing:
		return "EDIT"
	case ModeTextInput:
		return "TEXT"
	case ModeResize:
		return "RESIZE"
	case ModeMove:
		return "MOVE"
	case ModeMultiSelect:
		return "MULTI-SELECT"
	case ModeFileInput:
		return "FILE"
	case ModeConfirm:
		return "CONFIRM"
	case ModeBoxJump:
		return "BOX JUMP"
	case ModeTitleEdit:
		return "TITLE"
	case ModeContextMenu:
		return "MENU"
	default:
		return "UNKNOWN"
	}
}

func (m model) helpView() string {
	helpLines := helpText

	visibleHeight := m.height - 1
	if visibleHeight < 1 {
		visibleHeight = 1
	}

	startLine := m.helpScroll
	endLine := startLine + visibleHeight

	if startLine >= len(helpLines) {
		startLine = len(helpLines) - visibleHeight
		if startLine < 0 {
			startLine = 0
		}
		endLine = startLine + visibleHeight
	}

	if endLine > len(helpLines) {
		endLine = len(helpLines)
	}

	var visibleLines []string
	if startLine < len(helpLines) {
		visibleLines = helpLines[startLine:endLine]
	}

	result := strings.Join(visibleLines, "\n")

	statusLine := fmt.Sprintf("Help (%d-%d of %d lines) | j/k to scroll, Esc to close",
		startLine+1, endLine, len(helpLines))
	result += "\n" + statusLine

	return result
}
