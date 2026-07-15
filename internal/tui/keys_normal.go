package tui

import (
	"path/filepath"
	"strings"

	cv "flerm/internal/canvas"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) handleNormalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyEscape {
		m.zPanMode = false
		m.highlightMode = false
		m.connectionFrom = -1
		m.connectionFromLine = -1
		m.connectionFromX = 0
		m.connectionFromY = 0
		m.connectionWaypoints = nil
		m.mouseLineDrawing = false
		m.selectedBox = -1
		m.selectedText = -1
		m.selBox = -1
		m.selText = -1
		m.selConn = -1
		return m, nil
	}

	switch msg.String() {
	case "ctrl+c", "q":
		if m.config != nil && m.config.Confirmations {
			m.mode = ModeConfirm
			m.confirmAction = ConfirmQuit
			return m, nil
		}

		return m, tea.Quit
	case "n":
		if m.config != nil && m.config.Confirmations {
			m.mode = ModeConfirm
			m.confirmAction = ConfirmNewChart
			m.createNewBuffer = false
			return m, nil
		}

		buf := m.getCurrentBuffer()
		if buf != nil {
			buf.canvas = cv.NewCanvas()
			buf.filename = ""
			buf.undoStack = []Action{}
			buf.redoStack = []Action{}
		}
		m.cursorX = 0
		m.cursorY = 0
		m.errorMessage = ""
		m.successMessage = ""
		return m, nil
	case "N":

		m.addNewBuffer(cv.NewCanvas(), "")
		m.cursorX = 0
		m.cursorY = 0
		m.errorMessage = ""
		m.successMessage = ""
		return m, nil
	case "{":

		if len(m.buffers) > 1 {
			m.currentBufferIndex--
			if m.currentBufferIndex < 0 {
				m.currentBufferIndex = len(m.buffers) - 1
			}
		}
		return m, nil
	case "}":

		if len(m.buffers) > 1 {
			m.currentBufferIndex++
			if m.currentBufferIndex >= len(m.buffers) {
				m.currentBufferIndex = 0
			}
		}
		return m, nil
	case "?":
		m.help = !m.help
		return m, nil
	case "h", "left", "H", "shift+h", "shift+left":
		return m.handleNavigation(msg.String(), m.getMoveSpeed(msg.String()))
	case "l", "right", "L", "shift+l", "shift+right":
		return m.handleNavigation(msg.String(), m.getMoveSpeed(msg.String()))
	case "k", "up", "K", "shift+k", "shift+up":
		return m.handleNavigation(msg.String(), m.getMoveSpeed(msg.String()))
	case "j", "down", "J", "shift+j", "shift+down":
		return m.handleNavigation(msg.String(), m.getMoveSpeed(msg.String()))
	case "z":

		m.zPanMode = !m.zPanMode
		return m, nil
	case "b":
		m.zPanMode = false
		boxID := len(m.getCanvas().Boxes())
		panX, panY := m.getPanOffset()
		worldX, worldY := m.cursorX+panX, m.cursorY+panY
		m.getCanvas().AddBox(worldX, worldY, "Box")
		addData := AddBoxData{X: worldX, Y: worldY, Text: "Box", ID: boxID}
		deleteData := DeleteBoxData{ID: boxID, Connections: nil, Highlights: nil}
		m.recordAction(ActionAddBox, addData, deleteData)
		m.successMessage = ""
		m.ensureCursorInBounds()
		return m, nil
	case "B":

		m.mode = ModeBoxJump
		m.boxJumpInput = ""
		return m, nil
	case "T":

		m.zPanMode = false
		panX, panY := m.getPanOffset()
		worldX, worldY := m.cursorX+panX, m.cursorY+panY
		boxID := m.getCanvas().GetBoxAt(worldX, worldY)
		if boxID != -1 && boxID < len(m.getCanvas().Boxes()) {
			m.mode = ModeTitleEdit
			m.titleEditBoxID = boxID
			m.titleEditText = m.getCanvas().Boxes()[boxID].Title
			m.originalTitleText = m.titleEditText
			m.titleEditCursorPos = len(m.titleEditText)
		}
		return m, nil
	case "t":
		m.zPanMode = false
		m.mode = ModeTextInput
		panX, panY := m.getPanOffset()
		m.textInputX, m.textInputY = m.cursorX+panX, m.cursorY+panY
		m.textInputText = ""
		m.textInputCursorPos = 0
		return m, nil
	case "r":
		m.zPanMode = false
		panX, panY := m.getPanOffset()
		worldX, worldY := m.cursorX+panX, m.cursorY+panY
		boxID := m.getCanvas().GetBoxAt(worldX, worldY)
		if boxID != -1 {
			m.selectedBox = boxID
			if boxID < len(m.getCanvas().Boxes()) {
				m.originalWidth = m.getCanvas().Boxes()[boxID].Width
				m.originalHeight = m.getCanvas().Boxes()[boxID].Height
			}
			m.mode = ModeResize
		}
		return m, nil
	case "m":
		m.zPanMode = false
		panX, panY := m.getPanOffset()
		worldX, worldY := m.cursorX+panX, m.cursorY+panY
		boxID := m.getCanvas().GetBoxAt(worldX, worldY)
		textID := m.getCanvas().GetTextAt(worldX, worldY)
		if boxID != -1 {
			m.selectedBox = boxID
			m.selectedText = -1
			m.selectedBoxes = []int{}
			m.selectedTexts = []int{}
			m.selectedConnections = []int{}
			m.originalBoxPositions = make(map[int]point)
			m.originalTextPositions = make(map[int]point)
			m.originalConnections = make(map[int]Connection)
			m.originalHighlights = make(map[point]int)
			m.originalBoxConnections = make(map[int][]Connection)
			m.highlightMoveDelta = point{X: 0, Y: 0}
			if boxID < len(m.getCanvas().Boxes()) {
				box := m.getCanvas().Boxes()[boxID]
				m.originalMoveX, m.originalMoveY = box.X, box.Y

				m.originalBoxConnections[boxID] = m.getCanvas().GetConnectionsForBox(boxID)
				for y := box.Y; y < box.Y+box.Height; y++ {
					for x := box.X; x < box.X+box.Width; x++ {
						if color := m.getCanvas().GetHighlight(x, y); color != -1 {
							m.originalHighlights[point{X: x, Y: y}] = color
						}
					}
				}
			}
			m.mode = ModeMove
		} else if textID != -1 {
			m.selectedText = textID
			m.selectedBox = -1
			m.selectedBoxes = []int{}
			m.selectedTexts = []int{}
			m.selectedConnections = []int{}
			m.originalBoxPositions = make(map[int]point)
			m.originalTextPositions = make(map[int]point)
			m.originalConnections = make(map[int]Connection)
			m.originalHighlights = make(map[point]int)
			m.highlightMoveDelta = point{X: 0, Y: 0}
			if textID < len(m.getCanvas().Texts()) {
				text := m.getCanvas().Texts()[textID]
				m.originalTextMoveX, m.originalTextMoveY = text.X, text.Y
				maxWidth := 0
				for _, line := range text.Lines {
					if len(line) > maxWidth {
						maxWidth = len(line)
					}
				}
				for y := text.Y; y < text.Y+len(text.Lines); y++ {
					for x := text.X; x < text.X+maxWidth; x++ {
						if color := m.getCanvas().GetHighlight(x, y); color != -1 {
							m.originalHighlights[point{X: x, Y: y}] = color
						}
					}
				}
			}
			m.mode = ModeMove
		} else if highlightColor := m.getCanvas().GetHighlight(worldX, worldY); highlightColor != -1 {
			m.selectedBox = -1
			m.selectedText = -1
			m.selectedBoxes = []int{}
			m.selectedTexts = []int{}
			m.selectedConnections = []int{}
			m.originalBoxPositions = make(map[int]point)
			m.originalTextPositions = make(map[int]point)
			m.originalConnections = make(map[int]Connection)
			m.originalHighlights = make(map[point]int)
			m.originalHighlights[point{X: worldX, Y: worldY}] = highlightColor
			m.highlightMoveDelta = point{X: 0, Y: 0}
			m.mode = ModeMove
		}
		return m, nil
	case "M":
		m.zPanMode = false
		panX, panY := m.getPanOffset()
		m.selectionStartX = m.cursorX + panX
		m.selectionStartY = m.cursorY + panY
		m.selectedBoxes = []int{}
		m.selectedTexts = []int{}
		m.mode = ModeMultiSelect
		return m, nil
	case "e":
		panX, panY := m.getPanOffset()
		worldX, worldY := m.cursorX+panX, m.cursorY+panY
		boxID := m.getCanvas().GetBoxAt(worldX, worldY)
		textID := m.getCanvas().GetTextAt(worldX, worldY)
		if boxID != -1 {
			m.selectedBox = boxID
			m.selectedText = -1
			m.mode = ModeEditing
			m.editText = m.getCanvas().GetBoxText(boxID)
			m.originalEditText = m.editText
			m.editCursorPos = len(m.editText)
			m.editSelectionStart = -1
			m.editSelectionEnd = -1
			m.syncCursorPositions()
		} else if textID != -1 {
			m.selectedText = textID
			m.selectedBox = -1
			m.mode = ModeEditing
			m.editText = m.getCanvas().GetTextText(textID)
			m.originalEditText = m.editText
			m.editCursorPos = len(m.editText)
			m.editSelectionStart = -1
			m.editSelectionEnd = -1
			m.syncCursorPositions()
		}
		return m, nil
	case "A":
		panX, panY := m.getPanOffset()
		worldX, worldY := m.cursorX+panX, m.cursorY+panY
		lineConnIdx, _, _ := m.getCanvas().FindNearestPointOnConnection(worldX, worldY)
		if lineConnIdx != -1 {
			oldConn := m.getCanvas().Connections()[lineConnIdx]
			m.getCanvas().CycleConnectionArrowState(lineConnIdx)
			newConn := m.getCanvas().Connections()[lineConnIdx]
			cycleData := CycleArrowData{lineConnIdx, oldConn, newConn}
			m.recordAction(ActionCycleArrow, cycleData, cycleData)
			m.successMessage = ""
		}
		return m, nil
	case "a":
		panX, panY := m.getPanOffset()
		worldX, worldY := m.cursorX+panX, m.cursorY+panY
		boxID := m.getCanvas().GetBoxAt(worldX, worldY)
		lineConnIdx, lineX, lineY := m.getCanvas().FindNearestPointOnConnection(worldX, worldY)

		if m.connectionFrom == -1 && m.connectionFromLine == -1 {
			if boxID != -1 {
				fromBox := m.getCanvas().Boxes()[boxID]
				m.connectionFrom = boxID
				m.connectionFromLine = -1
				m.connectionFromX, m.connectionFromY = m.getCanvas().FindNearestEdgePoint(fromBox, worldX, worldY)
				m.connectionWaypoints = nil
			} else if lineConnIdx != -1 {
				m.connectionFrom = -1
				m.connectionFromLine = lineConnIdx
				m.connectionFromX, m.connectionFromY = lineX, lineY
				m.connectionWaypoints = nil
			}
		} else {
			if boxID != -1 {
				toBox := m.getCanvas().Boxes()[boxID]
				toX, toY := m.getCanvas().FindNearestEdgePoint(toBox, worldX, worldY)

				connection := Connection{
					FromID:    m.connectionFrom,
					ToID:      boxID,
					FromX:     m.connectionFromX,
					FromY:     m.connectionFromY,
					ToX:       toX,
					ToY:       toY,
					Waypoints: m.connectionWaypoints,
					Color:     -1,
				}

				m.getCanvas().AddConnectionWithWaypoints(m.connectionFrom, boxID, m.connectionFromX, m.connectionFromY, toX, toY, m.connectionWaypoints)
				addConnectionData := AddConnectionData{FromID: m.connectionFrom, ToID: boxID, Connection: connection}
				inverseConnectionData := AddConnectionData{FromID: m.connectionFrom, ToID: boxID, Connection: connection}
				m.recordAction(ActionAddConnection, addConnectionData, inverseConnectionData)
				m.successMessage = ""
				m.connectionFrom = -1
				m.connectionFromLine = -1
				m.connectionFromX = 0
				m.connectionFromY = 0
				m.connectionWaypoints = nil
			} else if lineConnIdx != -1 {
				toX, toY := lineX, lineY

				connection := Connection{
					FromID:    m.connectionFrom,
					ToID:      -1,
					FromX:     m.connectionFromX,
					FromY:     m.connectionFromY,
					ToX:       toX,
					ToY:       toY,
					Waypoints: m.connectionWaypoints,
					Color:     -1,
				}

				m.getCanvas().AddConnectionWithWaypoints(m.connectionFrom, -1, m.connectionFromX, m.connectionFromY, toX, toY, m.connectionWaypoints)
				addConnectionData := AddConnectionData{FromID: m.connectionFrom, ToID: -1, Connection: connection}
				inverseConnectionData := AddConnectionData{FromID: m.connectionFrom, ToID: -1, Connection: connection}
				m.recordAction(ActionAddConnection, addConnectionData, inverseConnectionData)
				m.successMessage = ""
				m.connectionFrom = -1
				m.connectionFromLine = -1
				m.connectionFromX = 0
				m.connectionFromY = 0
				m.connectionWaypoints = nil
			} else {
				m.connectionWaypoints = append(m.connectionWaypoints, point{X: worldX, Y: worldY})
			}
		}
		return m, nil
	case "d":
		panX, panY := m.getPanOffset()
		worldX, worldY := m.cursorX+panX, m.cursorY+panY
		if highlightColor := m.getCanvas().GetHighlight(worldX, worldY); highlightColor != -1 {
			m.getCanvas().ClearHighlight(worldX, worldY)
			cell := HighlightCell{X: worldX, Y: worldY, Color: -1, HadColor: true, OldColor: highlightColor}
			inverseCell := HighlightCell{X: worldX, Y: worldY, Color: highlightColor, HadColor: true, OldColor: -1}
			m.recordAction(ActionHighlight, HighlightData{Cells: []HighlightCell{cell}}, HighlightData{Cells: []HighlightCell{inverseCell}})
			return m, nil
		}

		lineConnIdx, _, _ := m.getCanvas().FindNearestPointOnConnection(worldX, worldY)
		if lineConnIdx != -1 {
			if m.config != nil && m.config.Confirmations {
				m.mode = ModeConfirm
				m.confirmAction = ConfirmDeleteConnection
				m.confirmConnIdx = lineConnIdx
				return m, nil
			}

			if lineConnIdx >= 0 && lineConnIdx < len(m.getCanvas().Connections()) {
				conn := m.getCanvas().Connections()[lineConnIdx]
				deleteData := AddConnectionData{FromID: conn.FromID, ToID: conn.ToID, Connection: conn}
				m.getCanvas().RemoveSpecificConnection(conn)
				m.recordAction(ActionDeleteConnection, deleteData, deleteData)
				m.successMessage = ""
			}
		} else {
			boxID := m.getCanvas().GetBoxAt(worldX, worldY)
			textID := m.getCanvas().GetTextAt(worldX, worldY)

			if boxID != -1 {
				if m.config != nil && m.config.Confirmations {
					m.mode = ModeConfirm
					m.confirmAction = ConfirmDeleteBox
					m.confirmBoxID = boxID
					return m, nil
				}

				if boxID >= 0 && boxID < len(m.getCanvas().Boxes()) {
					box := m.getCanvas().Boxes()[boxID]
					connectedConnections := make([]Connection, 0)
					for _, connection := range m.getCanvas().Connections() {
						if connection.FromID == boxID || connection.ToID == boxID {
							connectedConnections = append(connectedConnections, connection)
						}
					}
					highlights := m.getCanvas().GetHighlightsForBox(boxID)
					deleteData := DeleteBoxData{Box: box, ID: boxID, Connections: connectedConnections, Highlights: highlights}
					addData := AddBoxData{X: box.X, Y: box.Y, Text: box.GetText(), ID: box.ID}
					m.recordAction(ActionDeleteBox, deleteData, addData)
					m.getCanvas().DeleteBox(boxID)
					m.ensureCursorInBounds()
				}
			} else if textID != -1 {
				if m.config != nil && m.config.Confirmations {
					m.mode = ModeConfirm
					m.confirmAction = ConfirmDeleteText
					m.confirmTextID = textID
					return m, nil
				}

				if textID >= 0 && textID < len(m.getCanvas().Texts()) {
					text := m.getCanvas().Texts()[textID]
					highlights := m.getCanvas().GetHighlightsForText(textID)
					deleteData := DeleteTextData{Text: text, ID: textID, Highlights: highlights}
					addData := AddTextData{X: text.X, Y: text.Y, Text: text.GetText(), ID: text.ID}
					m.recordAction(ActionDeleteText, deleteData, addData)
				}
				m.getCanvas().DeleteText(textID)
				m.ensureCursorInBounds()
			}
		}
		return m, nil
	case "D":
		panX, panY := m.getPanOffset()
		worldX, worldY := m.cursorX+panX, m.cursorY+panY
		highlightedCells := make([]HighlightCell, 0)
		boxID := m.getCanvas().GetBoxAt(worldX, worldY)
		textID := m.getCanvas().GetTextAt(worldX, worldY)
		lineConnIdx, _, _ := m.getCanvas().FindNearestPointOnConnection(worldX, worldY)
		if boxID != -1 {
			for _, cell := range m.getCanvas().GetBoxCells(boxID) {
				if color := m.getCanvas().GetHighlight(cell.X, cell.Y); color != -1 {
					highlightedCells = append(highlightedCells, HighlightCell{X: cell.X, Y: cell.Y, Color: -1, HadColor: true, OldColor: color})
					m.getCanvas().ClearHighlight(cell.X, cell.Y)
				}
			}
		} else if textID != -1 {
			for _, cell := range m.getCanvas().GetTextCells(textID) {
				if color := m.getCanvas().GetHighlight(cell.X, cell.Y); color != -1 {
					highlightedCells = append(highlightedCells, HighlightCell{X: cell.X, Y: cell.Y, Color: -1, HadColor: true, OldColor: color})
					m.getCanvas().ClearHighlight(cell.X, cell.Y)
				}
			}
		} else if lineConnIdx != -1 {
			for _, cell := range m.getCanvas().GetConnectionCells(lineConnIdx) {
				if color := m.getCanvas().GetHighlight(cell.X, cell.Y); color != -1 {
					highlightedCells = append(highlightedCells, HighlightCell{X: cell.X, Y: cell.Y, Color: -1, HadColor: true, OldColor: color})
					m.getCanvas().ClearHighlight(cell.X, cell.Y)
				}
			}
		} else if highlightColor := m.getCanvas().GetHighlight(worldX, worldY); highlightColor != -1 {
			for _, cell := range m.getCanvas().GetAdjacentHighlightsOfColor(worldX, worldY, highlightColor) {
				if oldColor := m.getCanvas().GetHighlight(cell.X, cell.Y); oldColor != -1 {
					highlightedCells = append(highlightedCells, HighlightCell{X: cell.X, Y: cell.Y, Color: -1, HadColor: true, OldColor: oldColor})
					m.getCanvas().ClearHighlight(cell.X, cell.Y)
				}
			}
		}
		if len(highlightedCells) > 0 {
			inverseCells := make([]HighlightCell, len(highlightedCells))
			for i, cell := range highlightedCells {
				inverseCells[i] = HighlightCell{X: cell.X, Y: cell.Y, Color: cell.OldColor, HadColor: cell.HadColor, OldColor: -1}
			}
			m.recordAction(ActionHighlight, HighlightData{Cells: highlightedCells}, HighlightData{Cells: inverseCells})
		}
		return m, nil
	case "s":
		m.mode = ModeFileInput
		m.fileOp = FileOpSave
		if buf := m.getCurrentBuffer(); buf != nil && buf.filename != "" {
			baseName := filepath.Base(buf.filename)
			if strings.HasSuffix(strings.ToLower(baseName), ".sav") {
				baseName = baseName[:len(baseName)-4]
			}
			m.filename = baseName
		} else {
			m.filename = ""
		}
		m.errorMessage = ""
		m.successMessage = ""
		m.fromStartup = false
		return m, nil
	case "o":
		m.mode = ModeFileInput
		m.fileOp = FileOpOpen
		m.filename = ""
		m.errorMessage = ""
		m.successMessage = ""
		m.fromStartup = false
		m.openInNewBuffer = false
		m.scanTxtFiles()
		return m, nil
	case "O":
		m.mode = ModeFileInput
		m.fileOp = FileOpOpen
		m.filename = ""
		m.errorMessage = ""
		m.successMessage = ""
		m.fromStartup = false
		m.openInNewBuffer = true
		m.scanTxtFiles()
		return m, nil
	case "S":
		m.mode = ModeConfirm
		m.confirmAction = ConfirmChooseExportType
		m.filename = ""
		m.errorMessage = ""
		m.successMessage = ""
		return m, nil
	case "x":

		if len(m.buffers) > 0 {
			if m.config != nil && m.config.Confirmations {
				m.mode = ModeConfirm
				m.confirmAction = ConfirmCloseBuffer
				return m, nil
			}

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
			}
			m.cursorX = 0
			m.cursorY = 0
			m.errorMessage = ""
			m.successMessage = ""
		}
		return m, nil
	case "u":
		m.undo()
		m.successMessage = ""
		return m, nil
	case "U":
		m.redo()
		m.successMessage = ""
		return m, nil
	case "c":
		panX, panY := m.getPanOffset()
		worldX, worldY := m.cursorX+panX, m.cursorY+panY
		boxID := m.getCanvas().GetBoxAt(worldX, worldY)
		if boxID != -1 && boxID < len(m.getCanvas().Boxes()) {
			box := m.getCanvas().Boxes()[boxID]
			copiedBox := Box{
				X:           box.X,
				Y:           box.Y,
				Width:       box.Width,
				Height:      box.Height,
				ID:          box.ID,
				Lines:       make([]string, len(box.Lines)),
				Title:       box.Title,
				BorderStyle: box.BorderStyle,
				Color:       box.Color,
			}
			copy(copiedBox.Lines, box.Lines)
			m.clipboard = &copiedBox
		}
		return m, nil
	case "p":
		if m.clipboard != nil {
			boxID := len(m.getCanvas().Boxes())
			text := m.clipboard.GetText()
			panX, panY := m.getPanOffset()
			worldX, worldY := m.cursorX+panX, m.cursorY+panY
			m.getCanvas().AddBox(worldX, worldY, text)
			if boxID < len(m.getCanvas().Boxes()) {
				m.getCanvas().SetBoxSize(boxID, m.clipboard.Width, m.clipboard.Height)

				m.getCanvas().Boxes()[boxID].Title = m.clipboard.Title
				m.getCanvas().Boxes()[boxID].BorderStyle = m.clipboard.BorderStyle
				m.getCanvas().Boxes()[boxID].Color = m.clipboard.Color
				m.getCanvas().Boxes()[boxID].UpdateSize()
			}
			addData := AddBoxData{X: worldX, Y: worldY, Text: text, ID: boxID}
			deleteData := DeleteBoxData{ID: boxID, Connections: nil, Highlights: nil}
			m.recordAction(ActionAddBox, addData, deleteData)
			m.ensureCursorInBounds()
		}
		return m, nil
	case "esc", "escape":
		m.zPanMode = false
		m.highlightMode = false
		m.connectionFrom = -1
		m.connectionFromLine = -1
		m.connectionFromX = 0
		m.connectionFromY = 0
		m.connectionWaypoints = nil
		m.selectedBox = -1
		return m, nil
	case "tab":
		if m.highlightMode {

			m.selectedColor = (m.selectedColor + 1) % numColors
		} else {

			panX, panY := m.getPanOffset()
			worldX, worldY := m.cursorX+panX, m.cursorY+panY
			if boxID := m.getCanvas().GetBoxAt(worldX, worldY); boxID != -1 {

				oldStyle := m.getCanvas().CycleBorderStyle(boxID)
				newStyle := m.getCanvas().Boxes()[boxID].BorderStyle
				borderData := BorderStyleData{BoxID: boxID, OldStyle: oldStyle, NewStyle: newStyle}
				m.recordAction(ActionChangeBorderStyle, borderData, borderData)
			}
		}
		return m, nil
	case "Z":
		m.zPanMode = false
		panX, panY := m.getPanOffset()
		worldX, worldY := m.cursorX+panX, m.cursorY+panY
		if boxID := m.getCanvas().GetBoxAt(worldX, worldY); boxID != -1 {
			m.getCanvas().CycleBoxZLevel(boxID)
		}
		return m, nil
	case " ":
		if m.highlightMode {

			panX, panY := m.getPanOffset()
			worldX, worldY := m.cursorX+panX, m.cursorY+panY
			boxID := m.getCanvas().GetBoxAt(worldX, worldY)
			if boxID != -1 && boxID < len(m.getCanvas().Boxes()) {
				box := m.getCanvas().Boxes()[boxID]
				highlightedCells := make([]HighlightCell, 0)

				borderCells := m.getCanvas().GetBoxBorderCells(boxID)
				dividerCells := m.getCanvas().GetBoxTitleDividerCells(boxID)

				borderHighlighted := false
				titleBarHighlighted := false

				bottomY := box.Y + box.Height - 1
				for _, cell := range borderCells {
					if cell.Y == bottomY {
						if m.getCanvas().GetHighlight(cell.X, cell.Y) != -1 {
							borderHighlighted = true
							break
						}
					}
				}

				if box.Title != "" && len(dividerCells) > 0 {
					for _, cell := range dividerCells {
						if m.getCanvas().GetHighlight(cell.X, cell.Y) != -1 {
							titleBarHighlighted = true
							break
						}
					}
				}

				if box.Title == "" {
					if borderHighlighted {

						for _, cell := range borderCells {
							oldColor := m.getCanvas().GetHighlight(cell.X, cell.Y)
							m.getCanvas().ClearHighlight(cell.X, cell.Y)
							highlightedCells = append(highlightedCells, HighlightCell{
								X: cell.X, Y: cell.Y, Color: -1,
								HadColor: oldColor != -1, OldColor: oldColor,
							})
						}
					} else {

						for _, cell := range borderCells {
							oldColor := m.getCanvas().GetHighlight(cell.X, cell.Y)
							m.getCanvas().SetHighlight(cell.X, cell.Y, m.selectedColor)
							highlightedCells = append(highlightedCells, HighlightCell{
								X: cell.X, Y: cell.Y, Color: m.selectedColor,
								HadColor: oldColor != -1, OldColor: oldColor,
							})
						}
					}
				} else {

					if !borderHighlighted && !titleBarHighlighted {

						for _, cell := range borderCells {
							oldColor := m.getCanvas().GetHighlight(cell.X, cell.Y)
							m.getCanvas().SetHighlight(cell.X, cell.Y, m.selectedColor)
							highlightedCells = append(highlightedCells, HighlightCell{
								X: cell.X, Y: cell.Y, Color: m.selectedColor,
								HadColor: oldColor != -1, OldColor: oldColor,
							})
						}
					} else if borderHighlighted && !titleBarHighlighted {

						for _, cell := range borderCells {
							oldColor := m.getCanvas().GetHighlight(cell.X, cell.Y)
							m.getCanvas().ClearHighlight(cell.X, cell.Y)
							highlightedCells = append(highlightedCells, HighlightCell{
								X: cell.X, Y: cell.Y, Color: -1,
								HadColor: oldColor != -1, OldColor: oldColor,
							})
						}
						for _, cell := range dividerCells {
							oldColor := m.getCanvas().GetHighlight(cell.X, cell.Y)
							m.getCanvas().SetHighlight(cell.X, cell.Y, m.selectedColor)
							highlightedCells = append(highlightedCells, HighlightCell{
								X: cell.X, Y: cell.Y, Color: m.selectedColor,
								HadColor: oldColor != -1, OldColor: oldColor,
							})
						}
					} else if !borderHighlighted && titleBarHighlighted {

						for _, cell := range borderCells {
							oldColor := m.getCanvas().GetHighlight(cell.X, cell.Y)
							m.getCanvas().SetHighlight(cell.X, cell.Y, m.selectedColor)
							highlightedCells = append(highlightedCells, HighlightCell{
								X: cell.X, Y: cell.Y, Color: m.selectedColor,
								HadColor: oldColor != -1, OldColor: oldColor,
							})
						}
					} else {

						for _, cell := range borderCells {
							oldColor := m.getCanvas().GetHighlight(cell.X, cell.Y)
							m.getCanvas().ClearHighlight(cell.X, cell.Y)
							highlightedCells = append(highlightedCells, HighlightCell{
								X: cell.X, Y: cell.Y, Color: -1,
								HadColor: oldColor != -1, OldColor: oldColor,
							})
						}
						for _, cell := range dividerCells {
							oldColor := m.getCanvas().GetHighlight(cell.X, cell.Y)
							m.getCanvas().ClearHighlight(cell.X, cell.Y)
							highlightedCells = append(highlightedCells, HighlightCell{
								X: cell.X, Y: cell.Y, Color: -1,
								HadColor: oldColor != -1, OldColor: oldColor,
							})
						}
					}
				}

				if len(highlightedCells) > 0 {
					inverseCells := make([]HighlightCell, len(highlightedCells))
					for i, cell := range highlightedCells {
						oldColorForInverse := cell.OldColor
						if oldColorForInverse < 0 {
							oldColorForInverse = -1
						}
						inverseCells[i] = HighlightCell{
							X: cell.X, Y: cell.Y,
							Color:    oldColorForInverse,
							HadColor: cell.HadColor, OldColor: cell.Color,
						}
					}
					m.recordAction(ActionHighlight, HighlightData{Cells: highlightedCells}, HighlightData{Cells: inverseCells})
				}
			}

		} else {

			m.highlightMode = true
		}
		return m, nil
	case "enter":
		if m.highlightMode {
			panX, panY := m.getPanOffset()
			worldX, worldY := m.cursorX+panX, m.cursorY+panY
			boxID := m.getCanvas().GetBoxAt(worldX, worldY)
			textID := m.getCanvas().GetTextAt(worldX, worldY)
			lineConnIdx, _, _ := m.getCanvas().FindNearestPointOnConnection(worldX, worldY)
			highlightedCells := make([]HighlightCell, 0)

			if boxID != -1 {

				contentTextCells := m.getCanvas().GetBoxContentTextCells(boxID)
				titleTextCells := m.getCanvas().GetBoxTitleTextCells(boxID)

				contentHighlighted := false
				titleHighlighted := false

				for _, cell := range contentTextCells {
					if m.getCanvas().GetHighlight(cell.X, cell.Y) != -1 {
						contentHighlighted = true
						break
					}
				}
				for _, cell := range titleTextCells {
					if m.getCanvas().GetHighlight(cell.X, cell.Y) != -1 {
						titleHighlighted = true
						break
					}
				}

				if !contentHighlighted && !titleHighlighted {

					for _, cell := range contentTextCells {
						oldColor := m.getCanvas().GetHighlight(cell.X, cell.Y)
						m.getCanvas().SetHighlight(cell.X, cell.Y, m.selectedColor)
						highlightedCells = append(highlightedCells, HighlightCell{
							X: cell.X, Y: cell.Y, Color: m.selectedColor,
							HadColor: oldColor != -1, OldColor: oldColor,
						})
					}
				} else if contentHighlighted && !titleHighlighted {

					for _, cell := range contentTextCells {
						oldColor := m.getCanvas().GetHighlight(cell.X, cell.Y)
						m.getCanvas().ClearHighlight(cell.X, cell.Y)
						highlightedCells = append(highlightedCells, HighlightCell{
							X: cell.X, Y: cell.Y, Color: -1,
							HadColor: oldColor != -1, OldColor: oldColor,
						})
					}
					for _, cell := range titleTextCells {
						oldColor := m.getCanvas().GetHighlight(cell.X, cell.Y)
						m.getCanvas().SetHighlight(cell.X, cell.Y, m.selectedColor)
						highlightedCells = append(highlightedCells, HighlightCell{
							X: cell.X, Y: cell.Y, Color: m.selectedColor,
							HadColor: oldColor != -1, OldColor: oldColor,
						})
					}
				} else if !contentHighlighted && titleHighlighted {

					for _, cell := range contentTextCells {
						oldColor := m.getCanvas().GetHighlight(cell.X, cell.Y)
						m.getCanvas().SetHighlight(cell.X, cell.Y, m.selectedColor)
						highlightedCells = append(highlightedCells, HighlightCell{
							X: cell.X, Y: cell.Y, Color: m.selectedColor,
							HadColor: oldColor != -1, OldColor: oldColor,
						})
					}
				} else {

					for _, cell := range contentTextCells {
						oldColor := m.getCanvas().GetHighlight(cell.X, cell.Y)
						m.getCanvas().ClearHighlight(cell.X, cell.Y)
						highlightedCells = append(highlightedCells, HighlightCell{
							X: cell.X, Y: cell.Y, Color: -1,
							HadColor: oldColor != -1, OldColor: oldColor,
						})
					}
					for _, cell := range titleTextCells {
						oldColor := m.getCanvas().GetHighlight(cell.X, cell.Y)
						m.getCanvas().ClearHighlight(cell.X, cell.Y)
						highlightedCells = append(highlightedCells, HighlightCell{
							X: cell.X, Y: cell.Y, Color: -1,
							HadColor: oldColor != -1, OldColor: oldColor,
						})
					}
				}
			} else if textID != -1 {

				for _, cell := range m.getCanvas().GetTextCells(textID) {
					oldColor := m.getCanvas().GetHighlight(cell.X, cell.Y)
					m.getCanvas().SetHighlight(cell.X, cell.Y, m.selectedColor)
					highlightedCells = append(highlightedCells, HighlightCell{
						X:        cell.X,
						Y:        cell.Y,
						Color:    m.selectedColor,
						HadColor: oldColor != -1,
						OldColor: oldColor,
					})
				}
			} else if lineConnIdx != -1 {

				for _, cell := range m.getCanvas().GetConnectionCells(lineConnIdx) {
					oldColor := m.getCanvas().GetHighlight(cell.X, cell.Y)
					m.getCanvas().SetHighlight(cell.X, cell.Y, m.selectedColor)
					highlightedCells = append(highlightedCells, HighlightCell{
						X:        cell.X,
						Y:        cell.Y,
						Color:    m.selectedColor,
						HadColor: oldColor != -1,
						OldColor: oldColor,
					})
				}
			}

			if len(highlightedCells) > 0 {
				inverseCells := make([]HighlightCell, len(highlightedCells))
				for i, cell := range highlightedCells {
					oldColorForInverse := cell.OldColor
					if oldColorForInverse < 0 {
						oldColorForInverse = -1
					}
					inverseCells[i] = HighlightCell{
						X:        cell.X,
						Y:        cell.Y,
						Color:    oldColorForInverse,
						HadColor: cell.HadColor,
						OldColor: cell.Color,
					}
				}
				m.recordAction(ActionHighlight, HighlightData{Cells: highlightedCells}, HighlightData{Cells: inverseCells})
			}
		}
		return m, nil
	}
	return m, nil
}
