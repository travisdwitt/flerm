package tui

import (
	"slices"

	cv "flerm/internal/canvas"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) handleNormalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if _, ok := arrowDeltas[key]; ok {
		return m.handleNavigation(key)
	}
	canvas := m.getCanvas()
	worldX, worldY := m.worldCursor()

	switch key {
	case "esc", "escape":
		m.zPanMode = false
		m.highlightMode = false
		m.cancelMouseLine()
		m.selectedBox, m.selectedText = -1, -1
		m.selBox, m.selText, m.selConn = -1, -1, -1
	case "ctrl+c", "q":
		if m.unsavedChanges() || m.config.Confirmations {
			m.confirm(ConfirmQuit, -1)
			return m, nil
		}
		return m, tea.Quit
	case "n":
		if m.config.Confirmations {
			m.confirm(ConfirmNewChart, -1)
		} else {
			m.newChart()
		}
	case "N":
		m.addNewBuffer(Buffer{canvas: cv.NewCanvas()})
		m.resetView()
	case "{", "}":
		if n := len(m.buffers); n > 1 {
			delta := 1
			if key == "{" {
				delta = -1
			}
			m.currentBufferIndex = (m.currentBufferIndex + delta + n) % n
		}
	case "?":
		m.help = !m.help
	case "z":
		m.zPanMode = !m.zPanMode
	case "b":
		m.zPanMode = false
		m.addBoxRecorded(worldX, worldY, "Box")
	case "B":
		m.mode = ModeBoxJump
		m.boxJumpInput = ""
	case "T":
		if boxID := canvas.GetBoxAt(worldX, worldY); boxID != -1 {
			m.beginEdit(ModeTitleEdit, boxID, -1, canvas.Boxes()[boxID].Title)
		}
	case "t":
		if boxID := canvas.GetBoxAt(worldX, worldY); boxID != -1 {
			m.beginTooltipEdit(boxID, m.cursorX, m.cursorY)
			break
		}
		m.textInputX, m.textInputY = worldX, worldY
		m.beginEdit(ModeTextInput, -1, -1, "")
	case "r":
		m.zPanMode = false
		if boxID := canvas.GetBoxAt(worldX, worldY); boxID != -1 {
			box := canvas.Boxes()[boxID]
			m.selectedBox = boxID
			m.originalWidth, m.originalHeight = box.Width, box.Height
			m.mode = ModeResize
		}
	case "m":
		m.zPanMode = false
		m.beginMove(worldX, worldY)
	case "M":
		m.zPanMode = false
		m.selectionStartX, m.selectionStartY = worldX, worldY
		m.selectedBoxes, m.selectedTexts = nil, nil
		m.mode = ModeMultiSelect
	case "e":
		if boxID := canvas.GetBoxAt(worldX, worldY); boxID != -1 {
			m.beginEdit(ModeEditing, boxID, -1, canvas.GetBoxText(boxID))
		} else if textID := canvas.GetTextAt(worldX, worldY); textID != -1 {
			m.beginEdit(ModeEditing, -1, textID, canvas.GetTextText(textID))
		}
	case "A":
		if idx, _, _ := canvas.FindNearestPointOnConnection(worldX, worldY); idx != -1 {
			old := canvas.Connections()[idx]
			canvas.CycleConnectionArrowState(idx)
			data := CycleArrowData{idx, old, canvas.Connections()[idx]}
			m.recordAction(ActionCycleArrow, data, data)
			m.successMessage = ""
		}
	case "a":
		if m.connectionFrom == -1 && m.connectionFromLine == -1 {
			m.beginLine(worldX, worldY)
		} else {
			m.extendLine(worldX, worldY)
		}
	case "d":
		m.deleteAt(worldX, worldY)
	case "D":
		m.clearHighlightsAt(worldX, worldY)
	case "s":
		m.beginFileOp(FileOpSave, m.currentChartName())
	case "o", "O":
		m.beginFileOp(FileOpOpen, "")
		m.openInNewBuffer = key == "O"
		m.scanTxtFiles()
	case "S":
		m.confirm(ConfirmChooseExportType, -1)
		m.filename = ""
		m.errorMessage, m.successMessage = "", ""
	case "x":
		if m.config.Confirmations {
			m.confirm(ConfirmCloseBuffer, -1)
		} else {
			m.closeBuffer()
		}
	case "u":
		m.undo()
		m.successMessage = ""
	case "U":
		m.redo()
		m.successMessage = ""
	case "c":
		if boxID := canvas.GetBoxAt(worldX, worldY); boxID != -1 {
			box := canvas.Boxes()[boxID]
			box.Lines = slices.Clone(box.Lines)
			m.clipboard = &box
		}
	case "y":
		m.zPanMode = false
		m.duplicateAt(worldX, worldY)
	case "p":
		if m.clipboard != nil {
			boxID := m.addBoxRecorded(worldX, worldY, m.clipboard.GetText())
			canvas.SetBoxSize(boxID, m.clipboard.Width, m.clipboard.Height)
			canvas.SetBorderStyle(boxID, m.clipboard.BorderStyle)
			canvas.SetBoxColor(boxID, m.clipboard.Color)
			canvas.SetBoxTitle(boxID, m.clipboard.Title)
		}
	case "tab":
		if m.highlightMode {
			m.menuTargetBox, m.menuTargetText, m.menuTargetConn = -1, -1, -1
			m.menuItems = colorSubmenu()[1:]
			m.menuIndex = m.selectedColor
			m.menuStack = nil
			m.menuX, m.menuY = 0, m.height
			m.mode = ModeContextMenu
		} else if boxID := canvas.GetBoxAt(worldX, worldY); boxID != -1 {
			old := canvas.CycleBorderStyle(boxID)
			data := BorderStyleData{BoxID: boxID, OldStyle: old, NewStyle: canvas.Boxes()[boxID].BorderStyle}
			m.recordAction(ActionChangeBorderStyle, data, data)
		} else {
			m.minimap = !m.minimap
		}
	case "C":
		m.openColorMenu(worldX, worldY)
	case "v":
		m.allTooltips = !m.allTooltips
		m.tooltipScroll = 0
	case "pgup", "pgdown":
		if m.allTooltips {
			delta := m.sidebarInnerH()
			if key == "pgup" {
				delta = -delta
			}
			m.scrollTooltipSidebar(delta)
		}
	case "Z":
		m.zPanMode = false
		if boxID := canvas.GetBoxAt(worldX, worldY); boxID != -1 {
			canvas.CycleBoxZLevel(boxID)
		}
	case " ":
		if !m.highlightMode {
			m.highlightMode = true
		} else if boxID := canvas.GetBoxAt(worldX, worldY); boxID != -1 {
			box := canvas.Boxes()[boxID]
			border := canvas.GetBoxBorderCells(boxID)
			var bottom []point
			for _, cell := range border {
				if cell.Y == box.Y+box.Height-1 {
					bottom = append(bottom, cell)
				}
			}
			m.recordHighlights(m.cycleHighlightGroups(m.anyHighlighted(bottom), border, canvas.GetBoxTitleDividerCells(boxID)))
		}
	case "enter":
		if !m.highlightMode {
			break
		}
		switch boxID, textID := canvas.GetBoxAt(worldX, worldY), canvas.GetTextAt(worldX, worldY); {
		case boxID != -1:
			content := canvas.GetBoxContentTextCells(boxID)
			m.recordHighlights(m.cycleHighlightGroups(m.anyHighlighted(content), content, canvas.GetBoxTitleTextCells(boxID)))
		case textID != -1:
			m.recordHighlights(m.paintCells(canvas.GetTextCells(textID), m.selectedColor))
		default:
			if idx, _, _ := canvas.FindNearestPointOnConnection(worldX, worldY); idx != -1 {
				m.recordHighlights(m.paintCells(canvas.GetConnectionCells(idx), m.selectedColor))
			}
		}
	}
	return m, nil
}

func (m *model) openColorMenu(worldX, worldY int) {
	canvas := m.getCanvas()
	m.menuTargetBox, m.menuTargetText, m.menuTargetConn = -1, -1, -1
	switch boxID, textID := canvas.GetBoxAt(worldX, worldY), canvas.GetTextAt(worldX, worldY); {
	case boxID != -1:
		m.menuTargetBox = boxID
	case textID != -1:
		m.menuTargetText = textID
	default:
		idx, _, _ := canvas.FindNearestPointOnConnection(worldX, worldY)
		if idx == -1 {
			return
		}
		m.menuTargetConn = idx
	}
	m.menuWorldX, m.menuWorldY = worldX, worldY
	m.menuItems = colorSubmenu()
	m.menuIndex = firstSelectableMenuIndex(m.menuItems)
	m.menuStack = nil
	m.menuX, m.menuY = m.cursorX, m.cursorY
	m.mode = ModeContextMenu
}

func (m *model) addBoxRecorded(x, y int, text string) int {
	canvas := m.getCanvas()
	id := len(canvas.Boxes())
	canvas.AddBox(x, y, text)
	m.recordAction(ActionAddBox, AddData{X: x, Y: y, Text: text, ID: id}, DeleteBoxData{ID: id})
	m.successMessage = ""
	m.ensureCursorInBounds()
	return id
}

func (m *model) deleteAt(x, y int) {
	canvas := m.getCanvas()
	if canvas.GetHighlight(x, y) != -1 {
		m.recordHighlights(m.paintCells([]point{{X: x, Y: y}}, -1))
		return
	}
	ask := m.config.Confirmations
	if idx, _, _ := canvas.FindNearestPointOnConnection(x, y); idx != -1 {
		if ask {
			m.confirm(ConfirmDeleteConnection, idx)
		} else {
			m.deleteConnByIdx(idx)
		}
	} else if boxID := canvas.GetBoxAt(x, y); boxID != -1 {
		if canvas.Boxes()[boxID].Tooltip != "" {
			m.confirm(ConfirmDeleteBoxOrTooltip, boxID)
		} else if ask {
			m.confirm(ConfirmDeleteBox, boxID)
		} else {
			m.deleteBoxByID(boxID)
		}
	} else if textID := canvas.GetTextAt(x, y); textID != -1 {
		if ask {
			m.confirm(ConfirmDeleteText, textID)
		} else {
			m.deleteTextByID(textID)
		}
	}
}

func (m *model) clearHighlightsAt(x, y int) {
	canvas := m.getCanvas()
	var cells []point
	switch boxID, textID := canvas.GetBoxAt(x, y), canvas.GetTextAt(x, y); {
	case boxID != -1:
		cells = canvas.GetBoxCells(boxID)
	case textID != -1:
		cells = canvas.GetTextCells(textID)
	default:
		if idx, _, _ := canvas.FindNearestPointOnConnection(x, y); idx != -1 {
			cells = canvas.GetConnectionCells(idx)
		} else if color := canvas.GetHighlight(x, y); color != -1 {
			cells = canvas.GetAdjacentHighlightsOfColor(x, y, color)
		}
	}
	m.recordHighlights(m.paintCells(m.highlighted(cells), -1))
}

func (m *model) highlighted(cells []point) []point {
	var out []point
	for _, cell := range cells {
		if m.getCanvas().GetHighlight(cell.X, cell.Y) != -1 {
			out = append(out, cell)
		}
	}
	return out
}

func (m *model) anyHighlighted(cells []point) bool {
	return len(m.highlighted(cells)) > 0
}

func (m *model) paintCell(x, y, color int) HighlightCell {
	old := m.getCanvas().GetHighlight(x, y)
	m.applyHighlight(x, y, color)
	return HighlightCell{X: x, Y: y, Color: color, HadColor: old != -1, OldColor: old}
}

func (m *model) paintCells(cells []point, color int) []HighlightCell {
	out := make([]HighlightCell, 0, len(cells))
	for _, cell := range cells {
		out = append(out, m.paintCell(cell.X, cell.Y, color))
	}
	return out
}

func (m *model) cycleHighlightGroups(primaryLit bool, primary, secondary []point) []HighlightCell {
	switch {
	case primaryLit && m.anyHighlighted(secondary):
		return append(m.paintCells(primary, -1), m.paintCells(secondary, -1)...)
	case primaryLit:
		return append(m.paintCells(primary, -1), m.paintCells(secondary, m.selectedColor)...)
	}
	return m.paintCells(primary, m.selectedColor)
}

func (m *model) recordHighlights(cells []HighlightCell) {
	if len(cells) == 0 {
		return
	}
	inverse := make([]HighlightCell, len(cells))
	for i, c := range cells {
		inverse[i] = HighlightCell{X: c.X, Y: c.Y, Color: c.OldColor, HadColor: c.HadColor, OldColor: c.Color}
	}
	m.recordAction(ActionHighlight, HighlightData{Cells: cells}, HighlightData{Cells: inverse})
}

func (m *model) beginLine(x, y int) bool {
	canvas := m.getCanvas()
	if boxID := canvas.GetBoxAt(x, y); boxID != -1 {
		m.connectionFrom, m.connectionFromLine = boxID, -1
		m.connectionFromX, m.connectionFromY = canvas.FindNearestEdgePoint(canvas.Boxes()[boxID], x, y)
	} else if idx, lx, ly := canvas.FindNearestPointOnConnection(x, y); idx != -1 {
		m.connectionFrom, m.connectionFromLine = -1, idx
		m.connectionFromX, m.connectionFromY = lx, ly
	} else {
		return false
	}
	m.connectionWaypoints = nil
	return true
}

func (m *model) extendLine(x, y int) {
	canvas := m.getCanvas()
	toID, toX, toY := -1, 0, 0
	if boxID := canvas.GetBoxAt(x, y); boxID != -1 {
		toID = boxID
		toX, toY = canvas.FindNearestEdgePoint(canvas.Boxes()[boxID], x, y)
	} else if idx, lx, ly := canvas.FindNearestPointOnConnection(x, y); idx != -1 {
		toX, toY = lx, ly
	} else {
		m.connectionWaypoints = append(m.connectionWaypoints, point{X: x, Y: y})
		return
	}
	if conn, ok := canvas.AddConnectionWithWaypoints(m.connectionFrom, toID, m.connectionFromX, m.connectionFromY, toX, toY, m.connectionWaypoints); ok {
		m.recordAction(ActionAddConnection, conn, conn)
	}
	m.successMessage = ""
	m.cancelMouseLine()
}

func (m *model) cancelMouseLine() {
	m.mouseLineDrawing = false
	m.connectionFrom, m.connectionFromLine = -1, -1
	m.connectionFromX, m.connectionFromY = 0, 0
	m.connectionWaypoints = nil
}

func (m *model) deleteBoxByID(boxID int) {
	canvas := m.getCanvas()
	if boxID < 0 || boxID >= len(canvas.Boxes()) {
		return
	}
	box := canvas.Boxes()[boxID]
	m.recordAction(ActionDeleteBox,
		DeleteBoxData{Box: box, ID: boxID, Connections: canvas.GetConnectionsForBox(boxID), Highlights: canvas.GetHighlightsForBox(boxID)},
		AddData{X: box.X, Y: box.Y, Text: box.GetText(), ID: box.ID})
	canvas.DeleteBox(boxID)
	m.ensureCursorInBounds()
}

func (m *model) beginTooltipEdit(boxID, anchorX, anchorY int) {
	m.tooltipX, m.tooltipY = anchorX, anchorY
	m.beginEdit(ModeTooltipEdit, boxID, -1, m.getCanvas().Boxes()[boxID].Tooltip)
}

func (m *model) deleteBoxTooltip(boxID int) {
	canvas := m.getCanvas()
	if boxID < 0 || boxID >= len(canvas.Boxes()) {
		return
	}
	old := canvas.Boxes()[boxID].Tooltip
	canvas.SetBoxTooltip(boxID, "")
	m.recordAction(ActionEditTooltip,
		EditData{ID: boxID, NewText: "", OldText: old},
		EditData{ID: boxID, NewText: old, OldText: ""})
}

func (m *model) deleteTextByID(textID int) {
	canvas := m.getCanvas()
	if textID < 0 || textID >= len(canvas.Texts()) {
		return
	}
	text := canvas.Texts()[textID]
	m.recordAction(ActionDeleteText,
		DeleteTextData{Text: text, ID: textID, Highlights: canvas.GetHighlightsForText(textID)},
		AddData{X: text.X, Y: text.Y, Text: text.GetText(), ID: text.ID})
	canvas.DeleteText(textID)
	m.ensureCursorInBounds()
}

func (m *model) deleteConnByIdx(connIdx int) {
	canvas := m.getCanvas()
	if connIdx < 0 || connIdx >= len(canvas.Connections()) {
		return
	}
	conn := canvas.Connections()[connIdx]
	canvas.RemoveSpecificConnection(conn)
	m.recordAction(ActionDeleteConnection, conn, conn)
	m.successMessage = ""
}

const dupOffsetX, dupOffsetY = 2, 1

func (m *model) duplicateAt(worldX, worldY int) {
	canvas := m.getCanvas()
	if boxID := canvas.GetBoxAt(worldX, worldY); boxID != -1 {
		m.duplicateByID(false, boxID)
	} else if textID := canvas.GetTextAt(worldX, worldY); textID != -1 {
		m.duplicateByID(true, textID)
	}
}

func (m *model) duplicateByID(isText bool, srcID int) {
	canvas := m.getCanvas()
	if srcID < 0 {
		return
	}
	data := DuplicateData{IsText: isText, SrcID: srcID, DX: dupOffsetX, DY: dupOffsetY}
	if isText {
		data.NewID = canvas.DuplicateText(srcID, data.DX, data.DY)
	} else {
		data.NewID = canvas.DuplicateBox(srcID, data.DX, data.DY)
	}
	if data.NewID == -1 {
		return
	}
	m.recordAction(ActionDuplicate, data, data)
	m.successMessage = ""
	m.ensureCursorInBounds()
}
