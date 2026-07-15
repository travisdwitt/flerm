package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	cv "flerm/internal/canvas"

	tea "github.com/charmbracelet/bubbletea"
)

var arrowDeltas = map[string][2]int{
	"h": {-1, 0}, "left": {-1, 0}, "H": {-2, 0}, "shift+left": {-2, 0},
	"l": {1, 0}, "right": {1, 0}, "L": {2, 0}, "shift+right": {2, 0},
	"k": {0, -1}, "up": {0, -1}, "K": {0, -2}, "shift+up": {0, -2},
	"j": {0, 1}, "down": {0, 1}, "J": {0, 2}, "shift+down": {0, 2},
}

func (m model) handleStartupKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "n":

		m.buffers[0] = Buffer{
			canvas:    cv.NewCanvas(),
			undoStack: []Action{},
			redoStack: []Action{},
			filename:  "",
			panX:      0,
			panY:      0,
		}
		m.currentBufferIndex = 0
		m.mode = ModeNormal
		m.cursorX = 0
		m.cursorY = 0
		m.errorMessage = ""
		return m, nil
	case "o":

		m.mode = ModeFileInput
		m.fileOp = FileOpOpen
		m.filename = ""
		m.errorMessage = ""
		m.fromStartup = true
		m.openInNewBuffer = false
		m.scanTxtFiles()
		return m, nil
	case "q", "ctrl+c":
		return m, tea.Quit
	default:
		return m, nil
	}
}

func (m model) handleContextMenuKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "escape", "q":
		m.closeContextMenu()
		return m, nil
	case "j", "down":
		m.menuMoveSelection(1)
		return m, nil
	case "k", "up":
		m.menuMoveSelection(-1)
		return m, nil
	case "l", "right":

		m.menuDescend()
		return m, nil
	case "h", "left":

		m.menuAscend()
		return m, nil
	case "enter", " ":
		items := m.focusedItems()
		idx := m.focusedIndex()
		if idx >= 0 && idx < len(items) && !items[idx].Separator {
			if len(items[idx].Submenu) > 0 {
				m.menuDescend()
				return m, nil
			}
			cmd := m.activateMenuItem(items[idx].Action, items[idx].Arg)
			return m, cmd
		}
		return m, nil
	default:
		return m, nil
	}
}

func (m model) handleResizeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "escape":
		m.mode = ModeNormal
		m.selectedBox = -1
		return m, nil
	case "h", "left", "H", "shift+left", "l", "right", "L", "shift+right",
		"k", "up", "K", "shift+up", "j", "down", "J", "shift+down":
		if m.selectedBox != -1 {
			d := arrowDeltas[msg.String()]
			m.getCanvas().ResizeBox(m.selectedBox, d[0], d[1])
			m.ensureCursorInBounds()
		}
		return m, nil
	case "enter":

		if m.selectedBox != -1 && m.selectedBox < len(m.getCanvas().Boxes()) {
			currentBox := m.getCanvas().Boxes()[m.selectedBox]

			deltaWidth := currentBox.Width - m.originalWidth
			deltaHeight := currentBox.Height - m.originalHeight

			if deltaWidth != 0 || deltaHeight != 0 {
				resizeData := ResizeBoxData{ID: m.selectedBox, DeltaWidth: deltaWidth, DeltaHeight: deltaHeight}
				originalState := OriginalBoxState{ID: m.selectedBox, X: currentBox.X, Y: currentBox.Y, Width: m.originalWidth, Height: m.originalHeight}
				m.recordAction(ActionResizeBox, resizeData, originalState)
			}
		}
		m.mode = ModeNormal
		m.selectedBox = -1
		return m, nil
	}
	return m, nil
}

func (m model) handleMultiSelectKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "escape":
		m.mode = ModeNormal
		m.selectionStartX = -1
		m.selectionStartY = -1
		m.selectedBoxes = []int{}
		m.selectedTexts = []int{}
		return m, nil
	case "h", "left", "H", "shift+left":
		return m.handleNavigation(msg.String(), m.getMoveSpeed(msg.String()))
	case "l", "right", "L", "shift+l", "shift+right":
		return m.handleNavigation(msg.String(), m.getMoveSpeed(msg.String()))
	case "k", "up", "K", "shift+k", "shift+up":
		return m.handleNavigation(msg.String(), m.getMoveSpeed(msg.String()))
	case "j", "down", "J", "shift+j", "shift+down":
		return m.handleNavigation(msg.String(), m.getMoveSpeed(msg.String()))
	case "enter":
		panX, panY := m.getPanOffset()
		m.finalizeMultiSelect(m.cursorX+panX, m.cursorY+panY)
		return m, nil
	default:
		return m, nil
	}
}

func (m model) handleMoveKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "escape":
		m.mode = ModeNormal
		m.selectedBox = -1
		m.selectedText = -1
		m.selectedBoxes = []int{}
		m.selectedTexts = []int{}
		m.selectedConnections = []int{}
		m.originalBoxPositions = make(map[int]point)
		m.originalTextPositions = make(map[int]point)
		m.originalConnections = make(map[int]Connection)
		m.originalHighlights = make(map[point]int)
		return m, nil
	case "h", "left", "H", "shift+left", "l", "right", "L", "shift+right",
		"k", "up", "K", "shift+up", "j", "down", "J", "shift+down":
		d := arrowDeltas[msg.String()]
		if m.selectedBox != -1 || m.selectedText != -1 {
			m.handleSingleElementMove(d[0], d[1])
		} else if len(m.selectedBoxes) > 0 || len(m.selectedTexts) > 0 || len(m.selectedConnections) > 0 || len(m.originalHighlights) > 0 {
			m.handleMultiSelectMove(d[0], d[1])
		}
		return m, nil
	case "enter":
		m.commitMove()
		return m, nil
	}
	return m, nil
}

func (m model) handleFileInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Type == tea.KeyEscape:
		if m.fromStartup {

			m.mode = ModeStartup
			m.fromStartup = false
		} else {
			m.mode = ModeNormal
		}
		m.filename = ""
		m.errorMessage = ""
		return m, nil
	case msg.String() == "up":

		if m.fileOp == FileOpOpen && len(m.fileList) > 0 && !m.showingDeleteConfirm {

			matchesFile := false
			if m.selectedFileIndex >= 0 && m.selectedFileIndex < len(m.fileList) {
				selectedFile := m.fileList[m.selectedFileIndex]
				fileDisplayName := selectedFile
				if strings.HasSuffix(strings.ToLower(selectedFile), ".sav") {
					fileDisplayName = selectedFile[:len(selectedFile)-4]
				}
				matchesFile = (m.filename == fileDisplayName)
			}
			if matchesFile || m.filename == "" {
				if m.selectedFileIndex < 0 {
					m.selectedFileIndex = len(m.fileList) - 1
				} else if m.selectedFileIndex > 0 {
					m.selectedFileIndex--
				} else {
					m.selectedFileIndex = len(m.fileList) - 1
				}

				selectedFile := m.fileList[m.selectedFileIndex]
				if strings.HasSuffix(strings.ToLower(selectedFile), ".sav") {
					m.filename = selectedFile[:len(selectedFile)-4]
				} else {
					m.filename = selectedFile
				}
				return m, nil
			}
		}

	case msg.String() == "down":

		if m.fileOp == FileOpOpen && len(m.fileList) > 0 && !m.showingDeleteConfirm {

			matchesFile := false
			if m.selectedFileIndex >= 0 && m.selectedFileIndex < len(m.fileList) {
				selectedFile := m.fileList[m.selectedFileIndex]
				fileDisplayName := selectedFile
				if strings.HasSuffix(strings.ToLower(selectedFile), ".sav") {
					fileDisplayName = selectedFile[:len(selectedFile)-4]
				}
				matchesFile = (m.filename == fileDisplayName)
			}
			if matchesFile || m.filename == "" {
				if m.selectedFileIndex < 0 {
					m.selectedFileIndex = 0
				} else if m.selectedFileIndex < len(m.fileList)-1 {
					m.selectedFileIndex++
				} else {
					m.selectedFileIndex = 0
				}

				selectedFile := m.fileList[m.selectedFileIndex]
				if strings.HasSuffix(strings.ToLower(selectedFile), ".sav") {
					m.filename = selectedFile[:len(selectedFile)-4]
				} else {
					m.filename = selectedFile
				}
				return m, nil
			}
		}

	case msg.String() == "d":

		if m.fileOp == FileOpOpen && len(m.fileList) > 0 && m.selectedFileIndex >= 0 && m.selectedFileIndex < len(m.fileList) && !m.showingDeleteConfirm {
			m.showingDeleteConfirm = true
			m.confirmFileIndex = m.selectedFileIndex
			return m, nil
		}

		if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 {
			m.filename += string(msg.Runes)

			m.selectedFileIndex = -1
		}
		return m, nil
	case msg.String() == "y" || msg.String() == "Y":

		if m.fileOp == FileOpOpen && m.showingDeleteConfirm {
			if m.confirmFileIndex >= 0 && m.confirmFileIndex < len(m.fileList) {
				filename := m.fileList[m.confirmFileIndex]

				filepath := filename
				if m.config != nil {
					filepath = m.config.GetSavePath(filename)
				}
				err := os.Remove(filepath)
				if err != nil {
					m.errorMessage = fmt.Sprintf("Error deleting file: %s", err.Error())
				} else {

					m.fileList = append(m.fileList[:m.confirmFileIndex], m.fileList[m.confirmFileIndex+1:]...)

					if m.selectedFileIndex >= len(m.fileList) {
						m.selectedFileIndex = len(m.fileList) - 1
					}
					if m.selectedFileIndex < 0 && len(m.fileList) > 0 {
						m.selectedFileIndex = 0
					}

					if len(m.fileList) > 0 && m.selectedFileIndex >= 0 {
						selectedFile := m.fileList[m.selectedFileIndex]
						if strings.HasSuffix(strings.ToLower(selectedFile), ".sav") {
							m.filename = selectedFile[:len(selectedFile)-4]
						} else {
							m.filename = selectedFile
						}
					} else {
						m.filename = ""
					}
					displayName := filename
					if strings.HasSuffix(strings.ToLower(filename), ".sav") {
						displayName = filename[:len(filename)-4]
					}
					m.successMessage = fmt.Sprintf("Deleted %s", displayName)
				}
			}
			m.showingDeleteConfirm = false
			return m, nil
		}

		if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 {
			m.filename += string(msg.Runes)

			m.selectedFileIndex = -1
		}
		return m, nil
	case msg.String() == "n" || msg.String() == "N":

		if m.fileOp == FileOpOpen && m.showingDeleteConfirm {
			m.showingDeleteConfirm = false
			return m, nil
		}

		if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 {
			m.filename += string(msg.Runes)

			m.selectedFileIndex = -1
		}
		return m, nil
	case msg.Type == tea.KeyEscape:

		if m.fileOp == FileOpOpen && m.showingDeleteConfirm {
			m.showingDeleteConfirm = false
			return m, nil
		}

		if m.fromStartup {
			m.mode = ModeStartup
			m.fromStartup = false
		} else {
			m.mode = ModeNormal
		}
		m.filename = ""
		m.errorMessage = ""
		return m, nil
	case msg.Type == tea.KeyEnter:

		if m.fileOp == FileOpOpen && m.showingDeleteConfirm {
			return m, nil
		}

		filename := m.filename

		if m.fileOp == FileOpOpen && len(m.fileList) > 0 && m.selectedFileIndex >= 0 && m.selectedFileIndex < len(m.fileList) {
			selectedFile := m.fileList[m.selectedFileIndex]
			if filename == "" || (strings.HasSuffix(strings.ToLower(selectedFile), ".sav") && filename == selectedFile[:len(selectedFile)-4]) {
				filename = selectedFile
			}
		}
		switch m.fileOp {
		case FileOpSave, FileOpOpen:
			if m.fileOp == FileOpSave {

				if strings.TrimSpace(m.filename) == "" {
					m.errorMessage = "Please enter a filename"
					return m, nil
				}
			}
			if !strings.HasSuffix(strings.ToLower(filename), ".sav") {
				filename += ".sav"
			}
			if m.fileOp == FileOpSave {

				savePath := filename
				if m.config != nil {
					savePath = m.config.GetSavePath(filename)
				}

				if _, err := os.Stat(savePath); err == nil {
					if m.config != nil && m.config.Confirmations {

						m.mode = ModeConfirm
						m.confirmAction = ConfirmOverwriteFile

						m.filename = savePath
						return m, nil
					}

				}

				buf := m.getCurrentBuffer()
				panX, panY := 0, 0
				if buf != nil {
					panX, panY = buf.panX, buf.panY
				}
				err := m.getCanvas().SaveToFileWithPan(savePath, panX, panY)
				if err != nil {
					m.errorMessage = fmt.Sprintf("Error saving file: %s", err.Error())
					return m, nil
				} else {

					buf := m.getCurrentBuffer()
					if buf != nil {
						buf.filename = savePath
					}
					absPath, _ := filepath.Abs(savePath)
					m.successMessage = fmt.Sprintf("Saved to %s", absPath)
					m.errorMessage = ""
				}
			} else {

				loadPath := filename
				if m.config != nil && m.config.SaveDirectory != "" {
					saveDirPath := m.config.GetSavePath(filename)
					if _, err := os.Stat(saveDirPath); err == nil {
						loadPath = saveDirPath
					}
				}

				if _, err := os.Stat(loadPath); os.IsNotExist(err) {
					m.errorMessage = fmt.Sprintf("File not found: %s", filename)
					return m, nil
				}
				newCanvas := cv.NewCanvas()
				panX, panY, err := newCanvas.LoadFromFileWithPan(loadPath)
				if err != nil {
					m.errorMessage = fmt.Sprintf("Error opening file: %s", err.Error())
					return m, nil
				} else {

					if m.fromStartup {

						m.buffers[0] = Buffer{
							canvas:    newCanvas,
							undoStack: []Action{},
							redoStack: []Action{},
							filename:  loadPath,
							panX:      panX,
							panY:      panY,
						}
						m.currentBufferIndex = 0
						m.fromStartup = false
					} else if m.openInNewBuffer {

						m.addNewBufferWithPan(newCanvas, loadPath, panX, panY)
						m.openInNewBuffer = false
					} else {

						buf := m.getCurrentBuffer()
						if buf != nil {
							buf.canvas = newCanvas
							buf.filename = loadPath
							buf.panX = panX
							buf.panY = panY
							buf.undoStack = []Action{}
							buf.redoStack = []Action{}
						}
					}
					m.errorMessage = ""
				}
			}
		case FileOpSavePNG:

			baseFilename := filepath.Base(filename)
			if !strings.HasSuffix(strings.ToLower(baseFilename), ".png") {
				baseFilename += ".png"
			}

			savePath := baseFilename
			if m.config != nil {
				savePath = m.config.GetSavePath(baseFilename)
			}

			buf := m.getCurrentBuffer()
			panX, panY := 0, 0
			if buf != nil {
				panX, panY = buf.panX, buf.panY
			}

			showBufferBar := m.mode != ModeStartup && len(m.buffers) > 1
			renderWidth := m.width
			if renderWidth < 1 {
				renderWidth = 80
			}
			var renderHeight int
			if showBufferBar {
				renderHeight = m.height - 2
			} else {
				renderHeight = m.height - 1
			}
			if renderHeight < 1 {
				renderHeight = 24
			}
			err := m.getCanvas().ExportToPNG(savePath, renderWidth, renderHeight, panX, panY)
			if err != nil {
				m.errorMessage = fmt.Sprintf("Error exporting PNG: %s", err.Error())
				return m, nil
			} else {
				absPath, _ := filepath.Abs(savePath)
				m.successMessage = fmt.Sprintf("Exported to %s", absPath)
				m.errorMessage = ""
			}
		case FileOpSaveVisualTXT:

			baseFilename := filepath.Base(filename)
			if !strings.HasSuffix(strings.ToLower(baseFilename), ".txt") {
				baseFilename += ".txt"
			}

			savePath := baseFilename
			if m.config != nil {
				savePath = m.config.GetSavePath(baseFilename)
			}
			err := m.exportVisualTXT(savePath)
			if err != nil {
				m.errorMessage = fmt.Sprintf("Error exporting Visual TXT: %s", err.Error())
				return m, nil
			} else {
				absPath, _ := filepath.Abs(savePath)
				m.successMessage = fmt.Sprintf("Exported to %s", absPath)
				m.errorMessage = ""
			}
		}
		m.mode = ModeNormal
		m.filename = ""
		return m, nil
	case msg.Type == tea.KeyBackspace:

		if m.fileOp == FileOpOpen && m.showingDeleteConfirm {
			return m, nil
		}
		if len(m.filename) > 0 {
			m.filename = m.filename[:len(m.filename)-1]

			m.selectedFileIndex = -1
		}
		return m, nil
	case msg.Type == tea.KeySpace:

		if m.fileOp == FileOpOpen && m.showingDeleteConfirm {
			return m, nil
		}

		m.filename += " "

		m.selectedFileIndex = -1
		return m, nil
	default:

		if m.fileOp == FileOpOpen && m.showingDeleteConfirm {
			return m, nil
		}

		if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 {
			m.filename += string(msg.Runes)

			m.selectedFileIndex = -1
		}
		return m, nil
	}
	return m, nil
}

func (m model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":

		switch m.confirmAction {
		case ConfirmDeleteBox:

			if m.confirmBoxID >= 0 && m.confirmBoxID < len(m.getCanvas().Boxes()) {
				box := m.getCanvas().Boxes()[m.confirmBoxID]

				connectedConnections := make([]Connection, 0)
				for _, connection := range m.getCanvas().Connections() {
					if connection.FromID == m.confirmBoxID || connection.ToID == m.confirmBoxID {
						connectedConnections = append(connectedConnections, connection)
					}
				}
				highlights := m.getCanvas().GetHighlightsForBox(m.confirmBoxID)
				deleteData := DeleteBoxData{Box: box, ID: m.confirmBoxID, Connections: connectedConnections, Highlights: highlights}
				addData := AddBoxData{X: box.X, Y: box.Y, Text: box.GetText(), ID: box.ID}
				m.recordAction(ActionDeleteBox, deleteData, addData)
			}
			m.getCanvas().DeleteBox(m.confirmBoxID)
			m.ensureCursorInBounds()
		case ConfirmDeleteText:
			if m.confirmTextID >= 0 && m.confirmTextID < len(m.getCanvas().Texts()) {
				text := m.getCanvas().Texts()[m.confirmTextID]
				highlights := m.getCanvas().GetHighlightsForText(m.confirmTextID)
				deleteData := DeleteTextData{Text: text, ID: m.confirmTextID, Highlights: highlights}
				addData := AddTextData{X: text.X, Y: text.Y, Text: text.GetText(), ID: text.ID}
				m.recordAction(ActionDeleteText, deleteData, addData)
			}
			m.getCanvas().DeleteText(m.confirmTextID)
			m.ensureCursorInBounds()
		case ConfirmDeleteConnection:
			if m.confirmConnIdx >= 0 && m.confirmConnIdx < len(m.getCanvas().Connections()) {
				conn := m.getCanvas().Connections()[m.confirmConnIdx]
				deleteData := AddConnectionData{FromID: conn.FromID, ToID: conn.ToID, Connection: conn}
				m.getCanvas().RemoveSpecificConnection(conn)
				m.recordAction(ActionDeleteConnection, deleteData, deleteData)
				m.successMessage = ""
			}
			m.mode = ModeNormal
			return m, nil
		case ConfirmDeleteHighlight:
			oldColor := m.getCanvas().GetHighlight(m.confirmHighlightX, m.confirmHighlightY)
			if oldColor != -1 {
				m.getCanvas().ClearHighlight(m.confirmHighlightX, m.confirmHighlightY)

				highlightData := HighlightData{
					Cells: []HighlightCell{
						{
							X:        m.confirmHighlightX,
							Y:        m.confirmHighlightY,
							Color:    -1,
							HadColor: true,
							OldColor: oldColor,
						},
					},
				}
				inverseData := HighlightData{
					Cells: []HighlightCell{
						{
							X:        m.confirmHighlightX,
							Y:        m.confirmHighlightY,
							Color:    oldColor,
							HadColor: true,
							OldColor: -1,
						},
					},
				}
				m.recordAction(ActionHighlight, highlightData, inverseData)
			}
			m.mode = ModeNormal
			return m, nil
		case ConfirmQuit:
			return m, tea.Quit
		case ConfirmNewChart:
			if m.createNewBuffer {

				m.addNewBuffer(cv.NewCanvas(), "")
			} else {

				buf := m.getCurrentBuffer()
				if buf != nil {
					buf.canvas = cv.NewCanvas()
					buf.filename = ""
					buf.undoStack = []Action{}
					buf.redoStack = []Action{}
				}
			}
			m.cursorX = 0
			m.cursorY = 0
			m.errorMessage = ""
			m.successMessage = ""
			m.createNewBuffer = false
		case ConfirmCloseBuffer:

			if len(m.buffers) > 1 {

				newIndex := m.currentBufferIndex - 1
				if newIndex < 0 {
					newIndex = 0
				}

				m.buffers = append(m.buffers[:m.currentBufferIndex], m.buffers[m.currentBufferIndex+1:]...)

				m.currentBufferIndex = newIndex
			} else {

				canvas := cv.NewCanvas()
				m.buffers = []Buffer{
					{
						canvas:    canvas,
						undoStack: []Action{},
						redoStack: []Action{},
						filename:  "",
						panX:      0,
						panY:      0,
					},
				}
				m.currentBufferIndex = 0
				m.mode = ModeStartup
				m.cursorX = 0
				m.cursorY = 0
				m.errorMessage = ""
				m.successMessage = ""
				return m, nil
			}
			m.cursorX = 0
			m.cursorY = 0
			m.errorMessage = ""
			m.successMessage = ""
		case ConfirmOverwriteFile:

			filename := m.filename

			if m.config != nil && m.config.SaveDirectory != "" {

				baseName := filepath.Base(filename)
				filename = m.config.GetSavePath(baseName)
			}
			buf := m.getCurrentBuffer()
			panX, panY := 0, 0
			if buf != nil {
				panX, panY = buf.panX, buf.panY
			}
			err := m.getCanvas().SaveToFileWithPan(filename, panX, panY)
			if err != nil {
				m.errorMessage = fmt.Sprintf("Error saving file: %s", err.Error())
				m.mode = ModeFileInput
				return m, nil
			} else {

				buf := m.getCurrentBuffer()
				if buf != nil {
					buf.filename = filename
				}
				absPath, _ := filepath.Abs(filename)
				m.successMessage = fmt.Sprintf("Saved to %s", absPath)
				m.errorMessage = ""
			}
		case ConfirmChooseExportType:

			return m, nil
		}
		m.mode = ModeNormal
		m.filename = ""
		return m, nil
	case "p", "P":

		if m.confirmAction == ConfirmChooseExportType {
			m.mode = ModeFileInput
			m.fileOp = FileOpSavePNG

			buf := m.getCurrentBuffer()
			if buf != nil && buf.filename != "" {

				baseName := filepath.Base(buf.filename)

				if strings.HasSuffix(strings.ToLower(baseName), ".sav") {
					baseName = baseName[:len(baseName)-4]
				}
				m.filename = baseName
			} else {
				m.filename = "flowchart"
			}
			m.errorMessage = ""
			m.successMessage = ""
			m.fromStartup = false
			return m, nil
		}

	case "t", "T":

		if m.confirmAction == ConfirmChooseExportType {
			m.mode = ModeFileInput
			m.fileOp = FileOpSaveVisualTXT

			buf := m.getCurrentBuffer()
			if buf != nil && buf.filename != "" {

				baseName := filepath.Base(buf.filename)

				if strings.HasSuffix(strings.ToLower(baseName), ".sav") {
					baseName = baseName[:len(baseName)-4]
				}
				m.filename = baseName
			} else {
				m.filename = "flowchart"
			}
			m.errorMessage = ""
			m.successMessage = ""
			m.fromStartup = false
			return m, nil
		}

	case "n", "N", "esc", "escape":

		if m.confirmAction == ConfirmOverwriteFile {

			m.mode = ModeFileInput
			m.fileOp = FileOpSave
		} else if m.confirmAction == ConfirmChooseExportType {

			m.mode = ModeNormal
		} else {
			m.mode = ModeNormal
		}
		return m, nil
	default:

		return m, nil
	}
	return m, nil
}
