package tui

import (
	"os"
	"path/filepath"
	"strings"

	cv "flerm/internal/canvas"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) handleStartupKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "n":
		m.buffers[0] = Buffer{canvas: cv.NewCanvas()}
		m.currentBufferIndex = 0
		m.mode = ModeNormal
		m.resetView()
	case "r":
		if m.lastFile == "" {
			break
		}
		if err := m.openChart(m.lastFile); err != nil {
			m.errorMessage = "Error opening " + filepath.Base(m.lastFile) + ": " + err.Error()
			m.lastFile = ""
			break
		}
		m.mode = ModeNormal
		m.resetView()
	case "o":
		m.beginFileOp(FileOpOpen, "")
		m.fromStartup = true
		m.openInNewBuffer = false
		m.scanTxtFiles()
	case "q", "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

func (m model) handleContextMenuKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "escape", "q":
		m.closeContextMenu()
	case "j", "down":
		m.menuMoveSelection(1)
	case "k", "up":
		m.menuMoveSelection(-1)
	case "l", "right":
		m.menuDescend()
	case "h", "left":
		m.menuAscend()
	case "enter", " ":
		items := m.focusedItems()
		idx := m.focusedIndex()
		if idx >= 0 && idx < len(items) && !items[idx].Separator {
			if len(items[idx].Submenu) > 0 {
				m.menuDescend()
			} else {
				m.activateMenuItem(items[idx].Action, items[idx].Arg)
			}
		}
	}
	return m, nil
}

func (m model) handleResizeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch {
	case key == "esc" || key == "escape":
		m.mode = ModeNormal
		m.selectedBox = -1
	case key == "enter":
		if m.selectedBox >= 0 && m.selectedBox < len(m.getCanvas().Boxes()) {
			box := m.getCanvas().Boxes()[m.selectedBox]
			dw, dh := box.Width-m.originalWidth, box.Height-m.originalHeight
			if dw != 0 || dh != 0 {
				m.recordAction(ActionResizeBox,
					ResizeBoxData{ID: m.selectedBox, DeltaWidth: dw, DeltaHeight: dh},
					OriginalBoxState{ID: m.selectedBox, X: box.X, Y: box.Y, Width: m.originalWidth, Height: m.originalHeight})
			}
		}
		m.mode = ModeNormal
		m.selectedBox = -1
	default:
		if d, ok := arrowDeltas[key]; ok && m.selectedBox != -1 {
			m.getCanvas().ResizeBox(m.selectedBox, d[0], d[1])
			m.ensureCursorInBounds()
		}
	}
	return m, nil
}

func (m model) handleMultiSelectKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if _, ok := arrowDeltas[key]; ok {
		return m.handleNavigation(key)
	}
	switch key {
	case "esc", "escape":
		m.mode = ModeNormal
		m.selectionStartX, m.selectionStartY = CoordUnset, CoordUnset
		m.selectedBoxes, m.selectedTexts = nil, nil
	case "enter":
		worldX, worldY := m.worldCursor()
		if m.selectionStartX == CoordUnset || m.selectionStartY == CoordUnset {
			m.selectionStartX, m.selectionStartY = worldX, worldY
		} else {
			m.finalizeMultiSelect(worldX, worldY)
		}
	}
	return m, nil
}

func (m model) handleMoveKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch {
	case key == "esc" || key == "escape":
		m.mode = ModeNormal
		m.selectedBox, m.selectedText = -1, -1
		m.clearGroupSelection()
	case key == "enter":
		m.commitMove()
	default:
		if d, ok := arrowDeltas[key]; ok {
			if m.selectedBox != -1 || m.selectedText != -1 {
				m.handleSingleElementMove(d[0], d[1])
			} else if m.hasGroupSelection() {
				m.handleMultiSelectMove(d[0], d[1])
			}
		}
	}
	return m, nil
}

func (m model) handleFileInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	open := m.fileOp == FileOpOpen
	if open && m.showingDeleteConfirm {
		switch msg.String() {
		case "y", "Y":
			m.deleteSelectedFile()
		case "n", "N", "esc", "escape":
		default:
			return m, nil
		}
		m.showingDeleteConfirm = false
		return m, nil
	}
	if open && m.fileSearch {
		return m.handleFileSearchKey(msg)
	}
	hasSelection := m.selectedFileIndex >= 0 && m.selectedFileIndex < len(m.fileList)

	switch {
	case open && msg.String() == "?":
		m.fileSearch = true
		m.fileFilter = ""
		m.errorMessage = ""
		m.applyFileFilter()
	case msg.Type == tea.KeyEscape:
		m.mode = ModeNormal
		if m.fromStartup {
			m.mode = ModeStartup
			m.fromStartup = false
		}
		m.filename = ""
		m.errorMessage = ""
	case open && msg.Type == tea.KeyUp && len(m.fileList) > 0 && m.fileSelectionCurrent():
		m.moveFileSelection(-1)
	case open && msg.Type == tea.KeyDown && len(m.fileList) > 0 && m.fileSelectionCurrent():
		m.moveFileSelection(1)
	case open && msg.String() == "d" && hasSelection:
		m.showingDeleteConfirm = true
		m.confirmFileIndex = m.selectedFileIndex
	case msg.Type == tea.KeyEnter:
		return m.submitFileInput()
	case msg.Type == tea.KeyBackspace:
		if r := []rune(m.filename); len(r) > 0 {
			m.filename = string(r[:len(r)-1])
			m.selectedFileIndex = -1
		}
	case msg.Type == tea.KeyRunes, msg.Type == tea.KeySpace:
		m.filename += string(msg.Runes)
		m.selectedFileIndex = -1
	}
	return m, nil
}

func (m model) submitFileInput() (tea.Model, tea.Cmd) {
	filename := m.filename
	if m.fileOp == FileOpOpen && m.selectedFileIndex >= 0 && m.selectedFileIndex < len(m.fileList) {
		if sel := m.fileList[m.selectedFileIndex]; filename == "" || filename == chartDisplayName(sel) {
			filename = sel
		}
	}
	ok := false
	switch m.fileOp {
	case FileOpSave:
		if strings.TrimSpace(m.filename) == "" {
			m.errorMessage = "Please enter a filename"
			return m, nil
		}
		if !hasChartExt(filename) {
			filename += saveExt
		}
		path := m.config.GetSavePath(filename)
		if _, err := os.Stat(path); err == nil && m.config.Confirmations {
			m.confirm(ConfirmOverwriteFile, -1)
			m.filename = path
			return m, nil
		}
		ok = m.saveChart(path)
	case FileOpOpen:
		path := m.resolveChartPath(filename)
		if path == "" {
			m.errorMessage = "File not found: " + filename
			return m, nil
		}
		if err := m.openChart(path); err != nil {
			m.errorMessage = "Error opening file: " + err.Error()
			return m, nil
		}
		ok = true
	case FileOpSavePNG:
		ok = m.export(filename, ".png", "PNG", m.getCanvas().ExportToPNG)
	case FileOpSaveVisualTXT:
		ok = m.export(filename, ".txt", "Visual TXT", m.exportVisualTXT)
	}
	if ok {
		m.mode = ModeNormal
		m.filename = ""
	}
	return m, nil
}

func (m model) handleFileSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape:
		m.fileSearch = false
		m.fileFilter = ""
		m.applyFileFilter()
	case tea.KeyEnter:
		m.fileSearch = false
		if len(m.fileList) == 0 {
			m.fileFilter = ""
			m.applyFileFilter()
			return m, nil
		}
		return m.submitFileInput()
	case tea.KeyUp:
		m.moveFileSelection(-1)
	case tea.KeyDown:
		m.moveFileSelection(1)
	case tea.KeyBackspace:
		if r := []rune(m.fileFilter); len(r) > 0 {
			m.fileFilter = string(r[:len(r)-1])
			m.refilterFromTop()
		}
	case tea.KeyRunes, tea.KeySpace:
		m.fileFilter += string(msg.Runes)
		m.refilterFromTop()
	}
	return m, nil
}

func (m model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "y", "Y":
		switch m.confirmAction {
		case ConfirmDeleteBox:
			m.deleteBoxByID(m.confirmID)
		case ConfirmDeleteText:
			m.deleteTextByID(m.confirmID)
		case ConfirmDeleteConnection:
			m.deleteConnByIdx(m.confirmID)
		case ConfirmQuit:
			return m, tea.Quit
		case ConfirmNewChart:
			m.newChart()
		case ConfirmCloseBuffer:
			m.closeBuffer()
			if m.mode == ModeStartup {
				return m, nil
			}
		case ConfirmOverwriteFile:
			if !m.saveChart(m.filename) {
				m.mode = ModeFileInput
				return m, nil
			}
		case ConfirmChooseExportType, ConfirmDeleteBoxOrTooltip:
			return m, nil
		}
		m.mode = ModeNormal
		m.filename = ""
	case "b", "B":
		if m.confirmAction == ConfirmDeleteBoxOrTooltip {
			m.deleteBoxByID(m.confirmID)
			m.mode = ModeNormal
		}
	case "p", "P", "t", "T":
		if m.confirmAction == ConfirmDeleteBoxOrTooltip {
			if key == "t" || key == "T" {
				m.deleteBoxTooltip(m.confirmID)
				m.mode = ModeNormal
			}
			break
		}
		if m.confirmAction == ConfirmChooseExportType {
			op := FileOpSavePNG
			if key == "t" || key == "T" {
				op = FileOpSaveVisualTXT
			}
			name := m.currentChartName()
			if name == "" {
				name = "flowchart"
			}
			m.beginFileOp(op, name)
		}
	case "n", "N", "esc", "escape":
		m.mode = ModeNormal
		if m.confirmAction == ConfirmOverwriteFile {
			m.mode = ModeFileInput
			m.fileOp = FileOpSave
		}
	}
	return m, nil
}
