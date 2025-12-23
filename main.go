package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	p := tea.NewProgram(
		initialModel(),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

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
		bufName := fmt.Sprintf("%d", i+1)
		if buf.filename != "" {
			name := buf.filename
			if strings.HasSuffix(strings.ToLower(name), ".sav") {
				name = name[:len(name)-4]
			}
			bufName = name
		} else {
			bufName = fmt.Sprintf("Buffer %d", i+1)
		}
		if i == m.currentBufferIndex {
			bar.WriteString("[")
			bar.WriteString(bufName)
			bar.WriteString("]")
		} else {
			bar.WriteString(bufName)
		}
	}
	currentLen := bar.Len()
	if currentLen < width {
		bar.WriteString(strings.Repeat(" ", width-currentLen))
	} else {
		return bar.String()[:width]
	}
	return bar.String()
}

func initialModel() model {
	config := loadConfig()
	initialMode := ModeStartup
	if !config.StartMenu {
		initialMode = ModeNormal
	}
	buffer := Buffer{
		canvas:    NewCanvas(),
		undoStack: []Action{},
		redoStack: []Action{},
		filename:  "",
		panX:      0,
		panY:      0,
	}

	return model{
		buffers:               []Buffer{buffer},
		currentBufferIndex:    0,
		mode:                  initialMode,
		selectedBox:           -1,
		selectedText:          -1,
		connectionFrom:        -1,
		connectionFromLine:    -1,
		config:                config,
		highlightMode:         false,
		selectedColor:         0,
		selectionStartX:       -1,
		selectionStartY:       -1,
		selectedBoxes:         []int{},
		selectedTexts:         []int{},
		selectedConnections:   []int{},
		originalBoxPositions:  make(map[int]point),
		originalTextPositions: make(map[int]point),
		originalConnections:   make(map[int]Connection),
		originalHighlights:    make(map[point]int),
	}
}

func (m *model) ensureCursorInBounds() {
	if m.cursorX < 0 {
		m.cursorX = 0
	}
	if m.cursorY < 0 {
		m.cursorY = 0
	}
	if m.width > 0 && m.cursorX >= m.width {
		m.cursorX = m.width - 1
	}
	maxY := m.height - 2
	if maxY < 0 {
		maxY = 0
	}
	if m.cursorY > maxY {
		m.cursorY = maxY
	}
}

// linearToCursorPos converts linear cursor position to 2D row/column coordinates
func (m *model) linearToCursorPos(pos int, text string) (row, col int) {
	lines := strings.Split(text, "\n")
	currentPos := 0
	for lineIdx, line := range lines {
		lineLength := len([]rune(line))
		if pos <= currentPos+lineLength {
			return lineIdx, pos - currentPos
		}
		currentPos += lineLength + 1 // +1 for newline character
	}
	// If position is beyond text, place at end of last line
	if len(lines) > 0 {
		return len(lines) - 1, len([]rune(lines[len(lines)-1]))
	}
	return 0, 0
}

// cursorPosToLinear converts 2D row/column coordinates to linear cursor position
func (m *model) cursorPosToLinear(row, col int, text string) int {
	lines := strings.Split(text, "\n")
	if row < 0 {
		row = 0
	}
	if row >= len(lines) {
		// Position at end of text
		pos := 0
		for _, line := range lines {
			pos += len([]rune(line)) + 1 // +1 for newline
		}
		return pos - 1 // -1 to remove the last newline
	}

	pos := 0
	for i := 0; i < row; i++ {
		pos += len([]rune(lines[i])) + 1 // +1 for newline
	}

	// Clamp column to line length
	lineLength := len([]rune(lines[row]))
	if col < 0 {
		col = 0
	}
	if col > lineLength {
		col = lineLength
	}

	return pos + col
}

// syncCursorPositions keeps linear and 2D cursor positions synchronized
func (m *model) syncCursorPositions() {
	m.editCursorRow, m.editCursorCol = m.linearToCursorPos(m.editCursorPos, m.editText)
}

func (m *model) scanTxtFiles() {
	m.fileList = []string{}
	dir := ""
	if m.config != nil && m.config.SaveDirectory != "" {
		dir = m.config.SaveDirectory
	} else {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			m.selectedFileIndex = -1
			return
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		m.selectedFileIndex = -1
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".sav") {
			m.fileList = append(m.fileList, entry.Name())
		}
	}
	sort.Strings(m.fileList)
	if len(m.fileList) > 0 {
		m.selectedFileIndex = 0
		firstFile := m.fileList[0]
		if strings.HasSuffix(strings.ToLower(firstFile), ".sav") {
			m.filename = firstFile[:len(firstFile)-4]
		} else {
			m.filename = firstFile
		}
	} else {
		m.selectedFileIndex = -1
	}
}

func (m *model) moveHighlightsOnSelectedObjects(cumulativeDeltaX, cumulativeDeltaY int) point {
	if len(m.originalHighlights) == 0 {
		return m.highlightMoveDelta
	}
	for origPos := range m.originalHighlights {
		m.getCanvas().ClearHighlight(origPos.X+m.highlightMoveDelta.X, origPos.Y+m.highlightMoveDelta.Y)
	}
	for origPos, color := range m.originalHighlights {
		newX, newY := origPos.X+cumulativeDeltaX, origPos.Y+cumulativeDeltaY
		if newX >= 0 && newY >= 0 {
			m.getCanvas().SetHighlight(newX, newY, color)
		}
	}
	return point{X: cumulativeDeltaX, Y: cumulativeDeltaY}
}

func (m *model) moveContainedConnections(deltaX, deltaY int) {
	var cumulativeDeltaX, cumulativeDeltaY int
	if len(m.selectedBoxes) > 0 {
		boxID := m.selectedBoxes[0]
		if boxID >= 0 && boxID < len(m.getCanvas().boxes) {
			if originalPos, hasOriginal := m.originalBoxPositions[boxID]; hasOriginal {
				currentBox := m.getCanvas().boxes[boxID]
				cumulativeDeltaX, cumulativeDeltaY = currentBox.X-originalPos.X, currentBox.Y-originalPos.Y
			}
		}
	} else if len(m.selectedConnections) > 0 {
		connIdx := m.selectedConnections[0]
		if connIdx >= 0 && connIdx < len(m.getCanvas().connections) {
			conn := m.getCanvas().connections[connIdx]
			if originalConn, hasOriginal := m.originalConnections[connIdx]; hasOriginal {
				cumulativeDeltaX, cumulativeDeltaY = conn.FromX-originalConn.FromX+deltaX, conn.FromY-originalConn.FromY+deltaY
			} else {
				cumulativeDeltaX, cumulativeDeltaY = deltaX, deltaY
			}
		}
	}
	for _, connIdx := range m.selectedConnections {
		if connIdx >= 0 && connIdx < len(m.getCanvas().connections) {
			conn := &m.getCanvas().connections[connIdx]
			if conn.FromID == -1 {
				conn.FromX += cumulativeDeltaX
				conn.FromY += cumulativeDeltaY
			}
			if conn.ToID == -1 {
				conn.ToX += cumulativeDeltaX
				conn.ToY += cumulativeDeltaY
			}
			for i := range conn.Waypoints {
				conn.Waypoints[i].X += cumulativeDeltaX
				conn.Waypoints[i].Y += cumulativeDeltaY
			}
		}
	}
}

func (m *model) handleSingleElementMove(deltaX, deltaY int) {
	if m.selectedBox != -1 {
		m.getCanvas().MoveBox(m.selectedBox, deltaX, deltaY)
		if len(m.originalHighlights) > 0 {
			cumulativeDeltaX := m.getCanvas().boxes[m.selectedBox].X - m.originalMoveX
			cumulativeDeltaY := m.getCanvas().boxes[m.selectedBox].Y - m.originalMoveY
			m.highlightMoveDelta = m.moveHighlightsOnSelectedObjects(cumulativeDeltaX, cumulativeDeltaY)
		}
		m.ensureCursorInBounds()
	} else if m.selectedText != -1 {
		m.getCanvas().MoveText(m.selectedText, deltaX, deltaY)
		if len(m.originalHighlights) > 0 {
			cumulativeDeltaX := m.getCanvas().texts[m.selectedText].X - m.originalTextMoveX
			cumulativeDeltaY := m.getCanvas().texts[m.selectedText].Y - m.originalTextMoveY
			m.highlightMoveDelta = m.moveHighlightsOnSelectedObjects(cumulativeDeltaX, cumulativeDeltaY)
		}
		m.ensureCursorInBounds()
	}
}

func (m *model) handleMultiSelectMove(deltaX, deltaY int) {
	for _, boxID := range m.selectedBoxes {
		m.getCanvas().MoveBoxOnly(boxID, deltaX, deltaY)
	}
	for _, textID := range m.selectedTexts {
		m.getCanvas().MoveText(textID, deltaX, deltaY)
	}
	m.moveContainedConnections(deltaX, deltaY)
	var cumulativeDeltaX, cumulativeDeltaY int
	if len(m.selectedBoxes) > 0 {
		boxID := m.selectedBoxes[0]
		if boxID >= 0 && boxID < len(m.getCanvas().boxes) {
			if originalPos, hasOriginal := m.originalBoxPositions[boxID]; hasOriginal {
				currentBox := m.getCanvas().boxes[boxID]
				cumulativeDeltaX, cumulativeDeltaY = currentBox.X-originalPos.X, currentBox.Y-originalPos.Y
			}
		}
	} else if len(m.selectedTexts) > 0 {
		textID := m.selectedTexts[0]
		if textID >= 0 && textID < len(m.getCanvas().texts) {
			if originalPos, hasOriginal := m.originalTextPositions[textID]; hasOriginal {
				currentText := m.getCanvas().texts[textID]
				cumulativeDeltaX, cumulativeDeltaY = currentText.X-originalPos.X, currentText.Y-originalPos.Y
			}
		}
	} else if len(m.selectedConnections) > 0 {
		connIdx := m.selectedConnections[0]
		if connIdx >= 0 && connIdx < len(m.getCanvas().connections) {
			conn := m.getCanvas().connections[connIdx]
			if originalConn, hasOriginal := m.originalConnections[connIdx]; hasOriginal {
				cumulativeDeltaX, cumulativeDeltaY = conn.FromX-originalConn.FromX, conn.FromY-originalConn.FromY
			}
		}
	} else if len(m.originalHighlights) > 0 {
		cumulativeDeltaX, cumulativeDeltaY = m.highlightMoveDelta.X+deltaX, m.highlightMoveDelta.Y+deltaY
	}
	m.highlightMoveDelta = m.moveHighlightsOnSelectedObjects(cumulativeDeltaX, cumulativeDeltaY)
	m.ensureCursorInBounds()
}

func (m model) Init() tea.Cmd {
	return nil
}

func forceRefresh() tea.Msg {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ensureCursorInBounds()
		return m, nil

	case tea.KeyMsg:
		if m.help && m.mode != ModeStartup {
			switch msg.String() {
			case "escape", "q", "?":
				m.help = false
				m.helpScroll = 0
				return m, nil
			case "j", "down":
				helpLines := helpText
				totalLines := len(helpLines)
				visibleHeight := m.height - 1
				if visibleHeight < 1 {
					visibleHeight = 1
				}
				maxScroll := totalLines - visibleHeight
				if maxScroll < 0 {
					maxScroll = 0
				}
				if m.helpScroll < maxScroll {
					m.helpScroll++
				}
				return m, nil
			case "k", "up":
				if m.helpScroll > 0 {
					m.helpScroll--
				}
				return m, nil
			default:
				m.help = false
				m.helpScroll = 0
				return m, nil
			}
		}

		switch m.mode {
		case ModeStartup:
			switch msg.String() {
			case "n":
				// Replace startup buffer with new empty canvas
				m.buffers[0] = Buffer{
					canvas:    NewCanvas(),
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
				// Replace startup buffer when opening a file (will be replaced when file loads)
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

		case ModeNormal:
			if msg.Type == tea.KeyEscape {
				m.zPanMode = false
				m.highlightMode = false
				m.connectionFrom = -1
				m.connectionFromLine = -1
				m.connectionFromX = 0
				m.connectionFromY = 0
				m.selectedBox = -1
				m.selectedText = -1
				return m, nil
			}

			switch msg.String() {
			case "ctrl+c", "q":
				if m.config != nil && m.config.Confirmations {
					m.mode = ModeConfirm
					m.confirmAction = ConfirmQuit
					return m, nil
				}
				// Confirmations disabled, quit directly
				return m, tea.Quit
			case "n":
				if m.config != nil && m.config.Confirmations {
					m.mode = ModeConfirm
					m.confirmAction = ConfirmNewChart
					m.createNewBuffer = false // Replace current buffer
					return m, nil
				}
				// Confirmations disabled, replace current buffer directly
				buf := m.getCurrentBuffer()
				if buf != nil {
					buf.canvas = NewCanvas()
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
				// Create new buffer directly (no confirmation needed)
				m.addNewBuffer(NewCanvas(), "")
				m.cursorX = 0
				m.cursorY = 0
				m.errorMessage = ""
				m.successMessage = ""
				return m, nil
			case "{":
				// Go to previous buffer
				if len(m.buffers) > 1 {
					m.currentBufferIndex--
					if m.currentBufferIndex < 0 {
						m.currentBufferIndex = len(m.buffers) - 1
					}
				}
				return m, nil
			case "}":
				// Go to next buffer
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
				// Toggle z-pan mode (acts like holding z)
				m.zPanMode = !m.zPanMode
				return m, nil
			case "b":
				m.zPanMode = false
				boxID := len(m.getCanvas().boxes)
				panX, panY := m.getPanOffset()
				worldX, worldY := m.cursorX+panX, m.cursorY+panY
				m.getCanvas().AddBox(worldX, worldY, "Box")
				addData := AddBoxData{X: worldX, Y: worldY, Text: "Box", ID: boxID}
				deleteData := DeleteBoxData{ID: boxID, Connections: nil}
				m.recordAction(ActionAddBox, addData, deleteData)
				m.successMessage = ""
				m.ensureCursorInBounds()
				return m, nil
			case "B":
				// Enter box jump mode
				m.mode = ModeBoxJump
				m.boxJumpInput = ""
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
					if boxID < len(m.getCanvas().boxes) {
						m.originalWidth = m.getCanvas().boxes[boxID].Width
						m.originalHeight = m.getCanvas().boxes[boxID].Height
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
					m.highlightMoveDelta = point{X: 0, Y: 0}
					if boxID < len(m.getCanvas().boxes) {
						box := m.getCanvas().boxes[boxID]
						m.originalMoveX, m.originalMoveY = box.X, box.Y
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
					if textID < len(m.getCanvas().texts) {
						text := m.getCanvas().texts[textID]
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
					m.syncCursorPositions()
				} else if textID != -1 {
					m.selectedText = textID
					m.selectedBox = -1
					m.mode = ModeEditing
					m.editText = m.getCanvas().GetTextText(textID)
					m.originalEditText = m.editText
					m.editCursorPos = len(m.editText)
					m.syncCursorPositions()
				}
				return m, nil
			case "A":
				panX, panY := m.getPanOffset()
				worldX, worldY := m.cursorX+panX, m.cursorY+panY
				lineConnIdx, _, _ := m.getCanvas().findNearestPointOnConnection(worldX, worldY)
				if lineConnIdx != -1 {
					oldConn := m.getCanvas().connections[lineConnIdx]
					m.getCanvas().CycleConnectionArrowState(lineConnIdx)
					newConn := m.getCanvas().connections[lineConnIdx]
					cycleData := CycleArrowData{lineConnIdx, oldConn, newConn}
					m.recordAction(ActionCycleArrow, cycleData, cycleData)
					m.successMessage = ""
				}
				return m, nil
			case "a":
				panX, panY := m.getPanOffset()
				worldX, worldY := m.cursorX+panX, m.cursorY+panY
				boxID := m.getCanvas().GetBoxAt(worldX, worldY)
				lineConnIdx, lineX, lineY := m.getCanvas().findNearestPointOnConnection(worldX, worldY)

				if m.connectionFrom == -1 && m.connectionFromLine == -1 {
					if boxID != -1 {
						fromBox := m.getCanvas().boxes[boxID]
						m.connectionFrom = boxID
						m.connectionFromLine = -1
						m.connectionFromX, m.connectionFromY = m.getCanvas().findNearestEdgePoint(fromBox, worldX, worldY)
						m.connectionWaypoints = nil
					} else if lineConnIdx != -1 {
						m.connectionFrom = -1
						m.connectionFromLine = lineConnIdx
						m.connectionFromX, m.connectionFromY = lineX, lineY
						m.connectionWaypoints = nil
					}
				} else {
					if boxID != -1 {
						toBox := m.getCanvas().boxes[boxID]
						toX, toY := m.getCanvas().findNearestEdgePoint(toBox, worldX, worldY)

						connection := Connection{
							FromID:    m.connectionFrom,
							ToID:      boxID,
							FromX:     m.connectionFromX,
							FromY:     m.connectionFromY,
							ToX:       toX,
							ToY:       toY,
							Waypoints: m.connectionWaypoints,
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
						m.connectionWaypoints = append(m.connectionWaypoints, point{worldX, worldY})
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

				lineConnIdx, _, _ := m.getCanvas().findNearestPointOnConnection(worldX, worldY)
				if lineConnIdx != -1 {
					if m.config != nil && m.config.Confirmations {
						m.mode = ModeConfirm
						m.confirmAction = ConfirmDeleteConnection
						m.confirmConnIdx = lineConnIdx
						return m, nil
					}
					// Confirmations disabled, delete directly
					if lineConnIdx >= 0 && lineConnIdx < len(m.getCanvas().connections) {
						conn := m.getCanvas().connections[lineConnIdx]
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
						// Confirmations disabled, delete directly
						if boxID >= 0 && boxID < len(m.getCanvas().boxes) {
							box := m.getCanvas().boxes[boxID]
							connectedConnections := make([]Connection, 0)
							for _, connection := range m.getCanvas().connections {
								if connection.FromID == boxID || connection.ToID == boxID {
									connectedConnections = append(connectedConnections, connection)
								}
							}
							deleteData := DeleteBoxData{Box: box, ID: boxID, Connections: connectedConnections}
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
						// Confirmations disabled, delete directly
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
				lineConnIdx, _, _ := m.getCanvas().findNearestPointOnConnection(worldX, worldY)
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
				// Close current buffer (with confirmation if enabled)
				if len(m.buffers) > 0 {
					if m.config != nil && m.config.Confirmations {
						m.mode = ModeConfirm
						m.confirmAction = ConfirmCloseBuffer
						return m, nil
					}
					// Confirmations disabled, close directly
					if len(m.buffers) > 1 {
						newIndex := m.currentBufferIndex - 1
						if newIndex < 0 {
							newIndex = 0
						}
						m.buffers = append(m.buffers[:m.currentBufferIndex], m.buffers[m.currentBufferIndex+1:]...)
						m.currentBufferIndex = newIndex
					} else {
						// Last buffer - return to startup
						canvas := NewCanvas()
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
				if boxID != -1 && boxID < len(m.getCanvas().boxes) {
					box := m.getCanvas().boxes[boxID]
					copiedBox := Box{
						X:      box.X,
						Y:      box.Y,
						Width:  box.Width,
						Height: box.Height,
						ID:     box.ID,
						Lines:  make([]string, len(box.Lines)),
					}
					copy(copiedBox.Lines, box.Lines)
					m.clipboard = &copiedBox
				}
				return m, nil
			case "p":
				if m.clipboard != nil {
					boxID := len(m.getCanvas().boxes)
					text := m.clipboard.GetText()
					panX, panY := m.getPanOffset()
					worldX, worldY := m.cursorX+panX, m.cursorY+panY
					m.getCanvas().AddBox(worldX, worldY, text)
					if boxID < len(m.getCanvas().boxes) {
						m.getCanvas().SetBoxSize(boxID, m.clipboard.Width, m.clipboard.Height)
					}
					addData := AddBoxData{X: worldX, Y: worldY, Text: text, ID: boxID}
					deleteData := DeleteBoxData{ID: boxID, Connections: nil}
					m.recordAction(ActionAddBox, addData, deleteData)
					m.ensureCursorInBounds()
				}
				return m, nil
			case "escape":
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
				// Cycle through colors
				m.selectedColor = (m.selectedColor + 1) % numColors
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
				m.highlightMode = !m.highlightMode
				return m, nil
			case "enter":
				if m.highlightMode {
					panX, panY := m.getPanOffset()
					worldX, worldY := m.cursorX+panX, m.cursorY+panY
					boxID := m.getCanvas().GetBoxAt(worldX, worldY)
					textID := m.getCanvas().GetTextAt(worldX, worldY)
					lineConnIdx, _, _ := m.getCanvas().findNearestPointOnConnection(worldX, worldY)
					highlightedCells := make([]HighlightCell, 0)
					addHighlightCell := func(x, y int) {
						oldColor := m.getCanvas().GetHighlight(x, y)
						m.getCanvas().SetHighlight(x, y, m.selectedColor)
						highlightedCells = append(highlightedCells, HighlightCell{
							X:        x,
							Y:        y,
							Color:    m.selectedColor,
							HadColor: oldColor != -1,
							OldColor: oldColor,
						})
					}
					if boxID != -1 {
						for _, cell := range m.getCanvas().GetBoxCells(boxID) {
							addHighlightCell(cell.X, cell.Y)
						}
					} else if textID != -1 {
						for _, cell := range m.getCanvas().GetTextCells(textID) {
							addHighlightCell(cell.X, cell.Y)
						}
					} else if lineConnIdx != -1 {
						for _, cell := range m.getCanvas().GetConnectionCells(lineConnIdx) {
							addHighlightCell(cell.X, cell.Y)
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

		case ModeEditing:
			switch {
			case msg.Type == tea.KeyEscape:
				// Restore original text
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
				m.selectedBox = -1
				m.selectedText = -1
				return m, nil
			case msg.Type == tea.KeyCtrlS:
				if m.selectedBox != -1 {
					// Box text is already updated in real-time, just record the action
					editData := EditBoxData{ID: m.selectedBox, NewText: m.editText, OldText: m.originalEditText}
					inverseData := EditBoxData{ID: m.selectedBox, NewText: m.originalEditText, OldText: m.editText}
					m.recordAction(ActionEditBox, editData, inverseData)
				} else if m.selectedText != -1 {
					// Text is already updated in real-time, just record the action
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
				m.selectedBox = -1
				m.selectedText = -1
				return m, nil
			case msg.Type == tea.KeyCtrlV:
				// Paste clipboard content at cursor position
				clipText, err := readClipboardText()
				if err == nil && clipText != "" {
					// Insert clipboard text at cursor position
					m.editText = m.editText[:m.editCursorPos] + clipText + m.editText[m.editCursorPos:]
					m.editCursorPos += len([]rune(clipText)) // Use rune length for proper cursor positioning
					// Update box/text in real-time
					if m.selectedBox != -1 {
						m.getCanvas().SetBoxText(m.selectedBox, m.editText)
					} else if m.selectedText != -1 {
						m.getCanvas().SetTextText(m.selectedText, m.editText)
					}
				}
				return m, nil
			case msg.String() == "ctrl+v":
				// Alternative paste detection (some terminals send this)
				clipText, err := readClipboardText()
				if err == nil && clipText != "" {
					// Insert clipboard text at cursor position
					m.editText = m.editText[:m.editCursorPos] + clipText + m.editText[m.editCursorPos:]
					m.editCursorPos += len([]rune(clipText)) // Use rune length for proper cursor positioning
					// Update box/text in real-time
					if m.selectedBox != -1 {
						m.getCanvas().SetBoxText(m.selectedBox, m.editText)
					} else if m.selectedText != -1 {
						m.getCanvas().SetTextText(m.selectedText, m.editText)
					}
				}
				return m, nil
			case msg.String() == "left":
				if m.editCursorPos > 0 {
					m.editCursorPos--
				}
				m.syncCursorPositions()
				return m, nil
			case msg.String() == "right":
				if m.editCursorPos < len(m.editText) {
					m.editCursorPos++
				}
				m.syncCursorPositions()
				return m, nil
			case msg.String() == "up":
				// Move cursor up one line
				m.syncCursorPositions() // Ensure 2D position is current
				if m.editCursorRow > 0 {
					m.editCursorRow--
					m.editCursorPos = m.cursorPosToLinear(m.editCursorRow, m.editCursorCol, m.editText)
				}
				return m, nil
			case msg.String() == "down":
				// Move cursor down one line
				m.syncCursorPositions() // Ensure 2D position is current
				lines := strings.Split(m.editText, "\n")
				if m.editCursorRow < len(lines)-1 {
					m.editCursorRow++
					m.editCursorPos = m.cursorPosToLinear(m.editCursorRow, m.editCursorCol, m.editText)
				}
				return m, nil
			case msg.Type == tea.KeyEnter:
				m.editText = m.editText[:m.editCursorPos] + "\n" + m.editText[m.editCursorPos:]
				m.editCursorPos++
				// Update box/text in real-time
				if m.selectedBox != -1 {
					m.getCanvas().SetBoxText(m.selectedBox, m.editText)
				} else if m.selectedText != -1 {
					m.getCanvas().SetTextText(m.selectedText, m.editText)
				}
				return m, nil
			case msg.Type == tea.KeyBackspace:
				if m.editCursorPos > 0 {
					m.editText = m.editText[:m.editCursorPos-1] + m.editText[m.editCursorPos:]
					m.editCursorPos--
					// Update box/text in real-time
					if m.selectedBox != -1 {
						m.getCanvas().SetBoxText(m.selectedBox, m.editText)
					} else if m.selectedText != -1 {
						m.getCanvas().SetTextText(m.selectedText, m.editText)
					}
				}
				return m, nil
			case msg.Type == tea.KeyDelete:
				if m.editCursorPos < len(m.editText) {
					m.editText = m.editText[:m.editCursorPos] + m.editText[m.editCursorPos+1:]
					// Update box/text in real-time
					if m.selectedBox != -1 {
						m.getCanvas().SetBoxText(m.selectedBox, m.editText)
					} else if m.selectedText != -1 {
						m.getCanvas().SetTextText(m.selectedText, m.editText)
					}
				}
				return m, nil
			case msg.Type == tea.KeySpace:
				// Insert space character
				m.editText = m.editText[:m.editCursorPos] + " " + m.editText[m.editCursorPos:]
				m.editCursorPos++
				// Update box/text in real-time
				if m.selectedBox != -1 {
					m.getCanvas().SetBoxText(m.selectedBox, m.editText)
				} else if m.selectedText != -1 {
					m.getCanvas().SetTextText(m.selectedText, m.editText)
				}
				return m, nil
			default:
				// Handle typed characters - use msg.Runes for proper Unicode support
				// and to handle multi-character paste events
				if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 {
					runeStr := string(msg.Runes)
					m.editText = m.editText[:m.editCursorPos] + runeStr + m.editText[m.editCursorPos:]
					m.editCursorPos += len(msg.Runes)
					// Update box/text in real-time
					if m.selectedBox != -1 {
						m.getCanvas().SetBoxText(m.selectedBox, m.editText)
					} else if m.selectedText != -1 {
						m.getCanvas().SetTextText(m.selectedText, m.editText)
					}
				}
				return m, nil
			}

		case ModeTextInput:
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
				// Paste clipboard content at cursor position
				clipText, err := readClipboardText()
				if err == nil && clipText != "" {
					// Insert clipboard text at cursor position
					m.textInputText = m.textInputText[:m.textInputCursorPos] + clipText + m.textInputText[m.textInputCursorPos:]
					m.textInputCursorPos += len([]rune(clipText)) // Use rune length for proper cursor positioning
				}
				return m, nil
			case msg.String() == "ctrl+v":
				// Alternative paste detection (some terminals send this)
				clipText, err := readClipboardText()
				if err == nil && clipText != "" {
					// Insert clipboard text at cursor position
					m.textInputText = m.textInputText[:m.textInputCursorPos] + clipText + m.textInputText[m.textInputCursorPos:]
					m.textInputCursorPos += len([]rune(clipText)) // Use rune length for proper cursor positioning
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
				// Insert space character
				m.textInputText = m.textInputText[:m.textInputCursorPos] + " " + m.textInputText[m.textInputCursorPos:]
				m.textInputCursorPos++
				return m, nil
			default:
				// Handle typed characters - use msg.Runes for proper Unicode support
				// and to handle multi-character paste events
				if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 {
					runeStr := string(msg.Runes)
					m.textInputText = m.textInputText[:m.textInputCursorPos] + runeStr + m.textInputText[m.textInputCursorPos:]
					m.textInputCursorPos += len(msg.Runes)
				}
				return m, nil
			}

		case ModeBoxJump:
			switch {
			case msg.Type == tea.KeyEscape:
				m.mode = ModeNormal
				m.boxJumpInput = ""
				return m, nil
			case msg.Type == tea.KeyEnter:
				// Jump to the box with the entered number
				if m.boxJumpInput != "" {
					boxNum, err := strconv.Atoi(m.boxJumpInput)
					if err == nil && boxNum >= 0 && boxNum < len(m.getCanvas().boxes) {
						box := m.getCanvas().boxes[boxNum]
						panX, panY := m.getPanOffset()
						// Move cursor to the center of the box
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
				// Handle number input
				if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 {
					runeStr := string(msg.Runes)
					// Only allow digits
					if len(runeStr) == 1 && runeStr[0] >= '0' && runeStr[0] <= '9' {
						m.boxJumpInput += runeStr
					}
				}
				return m, nil
			}

		case ModeResize:
			switch msg.String() {
			case "escape":
				m.mode = ModeNormal
				m.selectedBox = -1
				return m, nil
			case "h", "left":
				if m.selectedBox != -1 {
					m.getCanvas().ResizeBox(m.selectedBox, -1, 0)
					m.ensureCursorInBounds()
				}
				return m, nil
			case "H", "shift+left":
				if m.selectedBox != -1 {
					m.getCanvas().ResizeBox(m.selectedBox, -2, 0)
					m.ensureCursorInBounds()
				}
				return m, nil
			case "l", "right":
				if m.selectedBox != -1 {
					m.getCanvas().ResizeBox(m.selectedBox, 1, 0)
					m.ensureCursorInBounds()
				}
				return m, nil
			case "L", "shift+right":
				if m.selectedBox != -1 {
					m.getCanvas().ResizeBox(m.selectedBox, 2, 0)
					m.ensureCursorInBounds()
				}
				return m, nil
			case "k", "up":
				if m.selectedBox != -1 {
					m.getCanvas().ResizeBox(m.selectedBox, 0, -1)
					m.ensureCursorInBounds()
				}
				return m, nil
			case "K", "shift+up":
				if m.selectedBox != -1 {
					m.getCanvas().ResizeBox(m.selectedBox, 0, -2)
					m.ensureCursorInBounds()
				}
				return m, nil
			case "j", "down":
				if m.selectedBox != -1 {
					m.getCanvas().ResizeBox(m.selectedBox, 0, 1)
					m.ensureCursorInBounds()
				}
				return m, nil
			case "J", "shift+down":
				if m.selectedBox != -1 {
					m.getCanvas().ResizeBox(m.selectedBox, 0, 2)
					m.ensureCursorInBounds()
				}
				return m, nil
			case "enter":
				// Record the resize action when finishing resize mode
				if m.selectedBox != -1 && m.selectedBox < len(m.getCanvas().boxes) {
					currentBox := m.getCanvas().boxes[m.selectedBox]
					// Calculate the total change from original size
					deltaWidth := currentBox.Width - m.originalWidth
					deltaHeight := currentBox.Height - m.originalHeight

					// Only record if there was an actual change
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

		case ModeMultiSelect:
			switch msg.String() {
			case "escape":
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
				selectionEndX, selectionEndY := m.cursorX+panX, m.cursorY+panY
				minX, maxX := m.selectionStartX, m.selectionStartX
				if selectionEndX < m.selectionStartX {
					minX = selectionEndX
				} else if selectionEndX > m.selectionStartX {
					maxX = selectionEndX
				}
				minY, maxY := m.selectionStartY, m.selectionStartY
				if selectionEndY < m.selectionStartY {
					minY = selectionEndY
				} else if selectionEndY > m.selectionStartY {
					maxY = selectionEndY
				}

				// Find all boxes and texts within the selection rectangle
				m.selectedBoxes = []int{}
				m.selectedTexts = []int{}
				m.selectedConnections = []int{}
				m.originalBoxPositions = make(map[int]point)
				m.originalTextPositions = make(map[int]point)
				m.originalConnections = make(map[int]Connection)

				for i, box := range m.getCanvas().boxes {
					boxRight, boxBottom := box.X+box.Width-1, box.Y+box.Height-1
					if !(boxRight < minX || box.X > maxX || boxBottom < minY || box.Y > maxY) {
						m.selectedBoxes = append(m.selectedBoxes, i)
						m.originalBoxPositions[i] = point{X: box.X, Y: box.Y}
					}
				}
				for i, text := range m.getCanvas().texts {
					textRight, textBottom := text.X, text.Y
					for _, line := range text.Lines {
						if text.X+len(line) > textRight {
							textRight = text.X + len(line)
						}
					}
					if len(text.Lines) > 0 {
						textBottom = text.Y + len(text.Lines) - 1
					}
					if !(textRight < minX || text.X > maxX || textBottom < minY || text.Y > maxY) {
						m.selectedTexts = append(m.selectedTexts, i)
						m.originalTextPositions[i] = point{X: text.X, Y: text.Y}
					}
				}
				selectedBoxSet := make(map[int]bool)
				for _, boxID := range m.selectedBoxes {
					selectedBoxSet[boxID] = true
				}
				pointInSelection := func(x, y int) bool {
					return x >= minX && x <= maxX && y >= minY && y <= maxY
				}
				shouldSelectConnection := func(conn Connection) bool {
					if conn.FromID >= 0 && conn.ToID >= 0 && selectedBoxSet[conn.FromID] && selectedBoxSet[conn.ToID] {
						return true
					}
					if conn.FromID >= 0 && selectedBoxSet[conn.FromID] && conn.ToID == -1 && pointInSelection(conn.ToX, conn.ToY) {
						return true
					}
					if conn.ToID >= 0 && selectedBoxSet[conn.ToID] && conn.FromID == -1 && pointInSelection(conn.FromX, conn.FromY) {
						return true
					}
					if conn.FromID == -1 && conn.ToID == -1 && pointInSelection(conn.FromX, conn.FromY) && pointInSelection(conn.ToX, conn.ToY) {
						return true
					}
					pointsInSelection, totalPoints := 0, 2+len(conn.Waypoints)
					if pointInSelection(conn.FromX, conn.FromY) {
						pointsInSelection++
					}
					if pointInSelection(conn.ToX, conn.ToY) {
						pointsInSelection++
					}
					for _, wp := range conn.Waypoints {
						if pointInSelection(wp.X, wp.Y) {
							pointsInSelection++
						}
					}
					return totalPoints > 0 && pointsInSelection*2 >= totalPoints
				}
				for i, conn := range m.getCanvas().connections {
					if shouldSelectConnection(conn) {
						m.selectedConnections = append(m.selectedConnections, i)
						connCopy := Connection{
							FromID:    conn.FromID,
							ToID:      conn.ToID,
							FromX:     conn.FromX,
							FromY:     conn.FromY,
							ToX:       conn.ToX,
							ToY:       conn.ToY,
							Waypoints: make([]point, len(conn.Waypoints)),
							ArrowFrom: conn.ArrowFrom,
							ArrowTo:   conn.ArrowTo,
						}
						copy(connCopy.Waypoints, conn.Waypoints)
						m.originalConnections[i] = connCopy
					}
				}
				m.originalHighlights = make(map[point]int)
				m.highlightMoveDelta = point{X: 0, Y: 0}
				for y := minY; y <= maxY; y++ {
					for x := minX; x <= maxX; x++ {
						if color := m.getCanvas().GetHighlight(x, y); color != -1 {
							m.originalHighlights[point{X: x, Y: y}] = color
						}
					}
				}

				// If we have selections (boxes, texts, connections, or highlights), enter move mode
				if len(m.selectedBoxes) > 0 || len(m.selectedTexts) > 0 || len(m.selectedConnections) > 0 || len(m.originalHighlights) > 0 || len(m.originalHighlights) > 0 {
					m.mode = ModeMove
					m.selectedBox = -1
					m.selectedText = -1
				} else {
					// No selections, return to normal mode
					m.mode = ModeNormal
					m.selectionStartX = -1
					m.selectionStartY = -1
				}
				return m, nil
			default:
				return m, nil
			}

		case ModeMove:
			switch msg.String() {
			case "escape":
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
			case "h", "left":
				if m.selectedBox != -1 || m.selectedText != -1 {
					m.handleSingleElementMove(-1, 0)
				} else if len(m.selectedBoxes) > 0 || len(m.selectedTexts) > 0 || len(m.selectedConnections) > 0 || len(m.originalHighlights) > 0 {
					m.handleMultiSelectMove(-1, 0)
				}
				return m, nil
			case "H", "shift+left":
				if m.selectedBox != -1 || m.selectedText != -1 {
					m.handleSingleElementMove(-2, 0)
				} else if len(m.selectedBoxes) > 0 || len(m.selectedTexts) > 0 || len(m.selectedConnections) > 0 || len(m.originalHighlights) > 0 {
					m.handleMultiSelectMove(-2, 0)
				}
				return m, nil
			case "l", "right":
				if m.selectedBox != -1 || m.selectedText != -1 {
					m.handleSingleElementMove(1, 0)
				} else if len(m.selectedBoxes) > 0 || len(m.selectedTexts) > 0 || len(m.selectedConnections) > 0 || len(m.originalHighlights) > 0 {
					m.handleMultiSelectMove(1, 0)
				}
				return m, nil
			case "L", "shift+right":
				if m.selectedBox != -1 || m.selectedText != -1 {
					m.handleSingleElementMove(2, 0)
				} else if len(m.selectedBoxes) > 0 || len(m.selectedTexts) > 0 || len(m.selectedConnections) > 0 || len(m.originalHighlights) > 0 {
					m.handleMultiSelectMove(2, 0)
				}
				return m, nil
			case "k", "up":
				// Check single-element moves FIRST (before multiselect)
				if m.selectedBox != -1 {
					m.getCanvas().MoveBox(m.selectedBox, 0, -1)
					if len(m.originalHighlights) > 0 {
						cumulativeDeltaX := m.getCanvas().boxes[m.selectedBox].X - m.originalMoveX
						cumulativeDeltaY := m.getCanvas().boxes[m.selectedBox].Y - m.originalMoveY
						m.highlightMoveDelta = m.moveHighlightsOnSelectedObjects(cumulativeDeltaX, cumulativeDeltaY)
					}
					m.ensureCursorInBounds()
				} else if m.selectedText != -1 {
					m.getCanvas().MoveText(m.selectedText, 0, -1)
					if len(m.originalHighlights) > 0 {
						cumulativeDeltaX := m.getCanvas().texts[m.selectedText].X - m.originalTextMoveX
						cumulativeDeltaY := m.getCanvas().texts[m.selectedText].Y - m.originalTextMoveY
						m.highlightMoveDelta = m.moveHighlightsOnSelectedObjects(cumulativeDeltaX, cumulativeDeltaY)
					}
					m.ensureCursorInBounds()
				} else if len(m.selectedBoxes) > 0 || len(m.selectedTexts) > 0 || len(m.selectedConnections) > 0 || len(m.originalHighlights) > 0 {
					// Multi-select move (or highlights-only move)
					deltaX, deltaY := 0, -1
					for _, boxID := range m.selectedBoxes {
						m.getCanvas().MoveBoxOnly(boxID, deltaX, deltaY)
					}
					for _, textID := range m.selectedTexts {
						m.getCanvas().MoveText(textID, deltaX, deltaY)
					}
					// Move contained connections as a unit
					m.moveContainedConnections(deltaX, deltaY)
					// Calculate cumulative delta for highlight movement
					var cumulativeDeltaX, cumulativeDeltaY int
					if len(m.selectedBoxes) > 0 {
						boxID := m.selectedBoxes[0]
						if boxID >= 0 && boxID < len(m.getCanvas().boxes) {
							currentBox := m.getCanvas().boxes[boxID]
							originalPos, hasOriginal := m.originalBoxPositions[boxID]
							if hasOriginal {
								cumulativeDeltaX = currentBox.X - originalPos.X
								cumulativeDeltaY = currentBox.Y - originalPos.Y
							}
						}
					} else if len(m.selectedTexts) > 0 {
						textID := m.selectedTexts[0]
						if textID >= 0 && textID < len(m.getCanvas().texts) {
							currentText := m.getCanvas().texts[textID]
							originalPos, hasOriginal := m.originalTextPositions[textID]
							if hasOriginal {
								cumulativeDeltaX = currentText.X - originalPos.X
								cumulativeDeltaY = currentText.Y - originalPos.Y
							}
						}
					} else if len(m.selectedConnections) > 0 {
						connIdx := m.selectedConnections[0]
						if connIdx >= 0 && connIdx < len(m.getCanvas().connections) {
							conn := m.getCanvas().connections[connIdx]
							originalConn, hasOriginal := m.originalConnections[connIdx]
							if hasOriginal {
								cumulativeDeltaX = conn.FromX - originalConn.FromX
								cumulativeDeltaY = conn.FromY - originalConn.FromY
							}
						}
					} else if len(m.originalHighlights) > 0 {
						// Only highlights selected, calculate cumulative delta from current position + incremental
						cumulativeDeltaX = m.highlightMoveDelta.X + deltaX
						cumulativeDeltaY = m.highlightMoveDelta.Y + deltaY
					}
					// Move highlights on selected objects (from original positions to new positions)
					m.highlightMoveDelta = m.moveHighlightsOnSelectedObjects(cumulativeDeltaX, cumulativeDeltaY)
					m.ensureCursorInBounds()
				}
				return m, nil
			case "K", "shift+up":
				// Check single-element moves FIRST (before multiselect)
				if m.selectedBox != -1 {
					m.getCanvas().MoveBox(m.selectedBox, 0, -2)
					if len(m.originalHighlights) > 0 {
						cumulativeDeltaX := m.getCanvas().boxes[m.selectedBox].X - m.originalMoveX
						cumulativeDeltaY := m.getCanvas().boxes[m.selectedBox].Y - m.originalMoveY
						m.highlightMoveDelta = m.moveHighlightsOnSelectedObjects(cumulativeDeltaX, cumulativeDeltaY)
					}
					m.ensureCursorInBounds()
				} else if m.selectedText != -1 {
					m.getCanvas().MoveText(m.selectedText, 0, -2)
					if len(m.originalHighlights) > 0 {
						cumulativeDeltaX := m.getCanvas().texts[m.selectedText].X - m.originalTextMoveX
						cumulativeDeltaY := m.getCanvas().texts[m.selectedText].Y - m.originalTextMoveY
						m.highlightMoveDelta = m.moveHighlightsOnSelectedObjects(cumulativeDeltaX, cumulativeDeltaY)
					}
					m.ensureCursorInBounds()
				} else if len(m.selectedBoxes) > 0 || len(m.selectedTexts) > 0 || len(m.selectedConnections) > 0 || len(m.originalHighlights) > 0 {
					// Multi-select move (or highlights-only move)
					deltaX, deltaY := 0, -2
					for _, boxID := range m.selectedBoxes {
						m.getCanvas().MoveBoxOnly(boxID, deltaX, deltaY)
					}
					for _, textID := range m.selectedTexts {
						m.getCanvas().MoveText(textID, deltaX, deltaY)
					}
					// Move contained connections as a unit
					m.moveContainedConnections(deltaX, deltaY)
					// Calculate cumulative delta for highlight movement
					var cumulativeDeltaX, cumulativeDeltaY int
					if len(m.selectedBoxes) > 0 {
						boxID := m.selectedBoxes[0]
						if boxID >= 0 && boxID < len(m.getCanvas().boxes) {
							currentBox := m.getCanvas().boxes[boxID]
							originalPos, hasOriginal := m.originalBoxPositions[boxID]
							if hasOriginal {
								cumulativeDeltaX = currentBox.X - originalPos.X
								cumulativeDeltaY = currentBox.Y - originalPos.Y
							}
						}
					} else if len(m.selectedTexts) > 0 {
						textID := m.selectedTexts[0]
						if textID >= 0 && textID < len(m.getCanvas().texts) {
							currentText := m.getCanvas().texts[textID]
							originalPos, hasOriginal := m.originalTextPositions[textID]
							if hasOriginal {
								cumulativeDeltaX = currentText.X - originalPos.X
								cumulativeDeltaY = currentText.Y - originalPos.Y
							}
						}
					} else if len(m.selectedConnections) > 0 {
						connIdx := m.selectedConnections[0]
						if connIdx >= 0 && connIdx < len(m.getCanvas().connections) {
							conn := m.getCanvas().connections[connIdx]
							originalConn, hasOriginal := m.originalConnections[connIdx]
							if hasOriginal {
								cumulativeDeltaX = conn.FromX - originalConn.FromX
								cumulativeDeltaY = conn.FromY - originalConn.FromY
							}
						}
					} else if len(m.originalHighlights) > 0 {
						// Only highlights selected, calculate cumulative delta from current position + incremental
						cumulativeDeltaX = m.highlightMoveDelta.X + deltaX
						cumulativeDeltaY = m.highlightMoveDelta.Y + deltaY
					}
					// Move highlights on selected objects (from original positions to new positions)
					m.highlightMoveDelta = m.moveHighlightsOnSelectedObjects(cumulativeDeltaX, cumulativeDeltaY)
					m.ensureCursorInBounds()
				}
				return m, nil
			case "j", "down":
				// Check single-element moves FIRST (before multiselect)
				if m.selectedBox != -1 {
					m.getCanvas().MoveBox(m.selectedBox, 0, 1)
					if len(m.originalHighlights) > 0 {
						cumulativeDeltaX := m.getCanvas().boxes[m.selectedBox].X - m.originalMoveX
						cumulativeDeltaY := m.getCanvas().boxes[m.selectedBox].Y - m.originalMoveY
						m.highlightMoveDelta = m.moveHighlightsOnSelectedObjects(cumulativeDeltaX, cumulativeDeltaY)
					}
					m.ensureCursorInBounds()
				} else if m.selectedText != -1 {
					m.getCanvas().MoveText(m.selectedText, 0, 1)
					if len(m.originalHighlights) > 0 {
						cumulativeDeltaX := m.getCanvas().texts[m.selectedText].X - m.originalTextMoveX
						cumulativeDeltaY := m.getCanvas().texts[m.selectedText].Y - m.originalTextMoveY
						m.highlightMoveDelta = m.moveHighlightsOnSelectedObjects(cumulativeDeltaX, cumulativeDeltaY)
					}
					m.ensureCursorInBounds()
				} else if len(m.selectedBoxes) > 0 || len(m.selectedTexts) > 0 || len(m.selectedConnections) > 0 || len(m.originalHighlights) > 0 {
					// Multi-select move (or highlights-only move)
					deltaX, deltaY := 0, 1
					for _, boxID := range m.selectedBoxes {
						m.getCanvas().MoveBoxOnly(boxID, deltaX, deltaY)
					}
					for _, textID := range m.selectedTexts {
						m.getCanvas().MoveText(textID, deltaX, deltaY)
					}
					// Move contained connections as a unit
					m.moveContainedConnections(deltaX, deltaY)
					// Calculate cumulative delta for highlight movement
					var cumulativeDeltaX, cumulativeDeltaY int
					if len(m.selectedBoxes) > 0 {
						boxID := m.selectedBoxes[0]
						if boxID >= 0 && boxID < len(m.getCanvas().boxes) {
							currentBox := m.getCanvas().boxes[boxID]
							originalPos, hasOriginal := m.originalBoxPositions[boxID]
							if hasOriginal {
								cumulativeDeltaX = currentBox.X - originalPos.X
								cumulativeDeltaY = currentBox.Y - originalPos.Y
							}
						}
					} else if len(m.selectedTexts) > 0 {
						textID := m.selectedTexts[0]
						if textID >= 0 && textID < len(m.getCanvas().texts) {
							currentText := m.getCanvas().texts[textID]
							originalPos, hasOriginal := m.originalTextPositions[textID]
							if hasOriginal {
								cumulativeDeltaX = currentText.X - originalPos.X
								cumulativeDeltaY = currentText.Y - originalPos.Y
							}
						}
					} else if len(m.selectedConnections) > 0 {
						connIdx := m.selectedConnections[0]
						if connIdx >= 0 && connIdx < len(m.getCanvas().connections) {
							conn := m.getCanvas().connections[connIdx]
							originalConn, hasOriginal := m.originalConnections[connIdx]
							if hasOriginal {
								cumulativeDeltaX = conn.FromX - originalConn.FromX
								cumulativeDeltaY = conn.FromY - originalConn.FromY
							}
						}
					} else if len(m.originalHighlights) > 0 {
						// Only highlights selected, calculate cumulative delta from current position + incremental
						cumulativeDeltaX = m.highlightMoveDelta.X + deltaX
						cumulativeDeltaY = m.highlightMoveDelta.Y + deltaY
					}
					// Move highlights on selected objects (from original positions to new positions)
					m.highlightMoveDelta = m.moveHighlightsOnSelectedObjects(cumulativeDeltaX, cumulativeDeltaY)
					m.ensureCursorInBounds()
				}
				return m, nil
			case "J", "shift+down":
				// Check single-element moves FIRST (before multiselect)
				if m.selectedBox != -1 {
					m.getCanvas().MoveBox(m.selectedBox, 0, 2)
					if len(m.originalHighlights) > 0 {
						cumulativeDeltaX := m.getCanvas().boxes[m.selectedBox].X - m.originalMoveX
						cumulativeDeltaY := m.getCanvas().boxes[m.selectedBox].Y - m.originalMoveY
						m.highlightMoveDelta = m.moveHighlightsOnSelectedObjects(cumulativeDeltaX, cumulativeDeltaY)
					}
					m.ensureCursorInBounds()
				} else if m.selectedText != -1 {
					m.getCanvas().MoveText(m.selectedText, 0, 2)
					if len(m.originalHighlights) > 0 {
						cumulativeDeltaX := m.getCanvas().texts[m.selectedText].X - m.originalTextMoveX
						cumulativeDeltaY := m.getCanvas().texts[m.selectedText].Y - m.originalTextMoveY
						m.highlightMoveDelta = m.moveHighlightsOnSelectedObjects(cumulativeDeltaX, cumulativeDeltaY)
					}
					m.ensureCursorInBounds()
				} else if len(m.selectedBoxes) > 0 || len(m.selectedTexts) > 0 || len(m.selectedConnections) > 0 || len(m.originalHighlights) > 0 {
					// Multi-select move (or highlights-only move)
					deltaX, deltaY := 0, 2
					for _, boxID := range m.selectedBoxes {
						m.getCanvas().MoveBoxOnly(boxID, deltaX, deltaY)
					}
					for _, textID := range m.selectedTexts {
						m.getCanvas().MoveText(textID, deltaX, deltaY)
					}
					// Move contained connections as a unit
					m.moveContainedConnections(deltaX, deltaY)
					// Calculate cumulative delta for highlight movement
					var cumulativeDeltaX, cumulativeDeltaY int
					if len(m.selectedBoxes) > 0 {
						boxID := m.selectedBoxes[0]
						if boxID >= 0 && boxID < len(m.getCanvas().boxes) {
							currentBox := m.getCanvas().boxes[boxID]
							originalPos, hasOriginal := m.originalBoxPositions[boxID]
							if hasOriginal {
								cumulativeDeltaX = currentBox.X - originalPos.X
								cumulativeDeltaY = currentBox.Y - originalPos.Y
							}
						}
					} else if len(m.selectedTexts) > 0 {
						textID := m.selectedTexts[0]
						if textID >= 0 && textID < len(m.getCanvas().texts) {
							currentText := m.getCanvas().texts[textID]
							originalPos, hasOriginal := m.originalTextPositions[textID]
							if hasOriginal {
								cumulativeDeltaX = currentText.X - originalPos.X
								cumulativeDeltaY = currentText.Y - originalPos.Y
							}
						}
					} else if len(m.selectedConnections) > 0 {
						connIdx := m.selectedConnections[0]
						if connIdx >= 0 && connIdx < len(m.getCanvas().connections) {
							conn := m.getCanvas().connections[connIdx]
							originalConn, hasOriginal := m.originalConnections[connIdx]
							if hasOriginal {
								cumulativeDeltaX = conn.FromX - originalConn.FromX
								cumulativeDeltaY = conn.FromY - originalConn.FromY
							}
						}
					} else if len(m.originalHighlights) > 0 {
						// Only highlights selected, calculate cumulative delta from current position + incremental
						cumulativeDeltaX = m.highlightMoveDelta.X + deltaX
						cumulativeDeltaY = m.highlightMoveDelta.Y + deltaY
					}
					// Move highlights on selected objects (from original positions to new positions)
					m.highlightMoveDelta = m.moveHighlightsOnSelectedObjects(cumulativeDeltaX, cumulativeDeltaY)
					m.ensureCursorInBounds()
				}
				return m, nil
			case "enter":
				// Record the move action when finishing move mode
				// Handle multi-select moves first
				if len(m.selectedBoxes) > 0 {
					for _, boxID := range m.selectedBoxes {
						if boxID >= 0 && boxID < len(m.getCanvas().boxes) {
							currentBox := m.getCanvas().boxes[boxID]
							originalPos, hasOriginal := m.originalBoxPositions[boxID]
							if hasOriginal {
								deltaX := currentBox.X - originalPos.X
								deltaY := currentBox.Y - originalPos.Y
								if deltaX != 0 || deltaY != 0 {
									moveData := MoveBoxData{ID: boxID, DeltaX: deltaX, DeltaY: deltaY}
									originalState := OriginalBoxState{ID: boxID, X: originalPos.X, Y: originalPos.Y, Width: currentBox.Width, Height: currentBox.Height}
									m.recordAction(ActionMoveBox, moveData, originalState)
								}
							}
						}
					}
				}
				if len(m.selectedTexts) > 0 {
					for _, textID := range m.selectedTexts {
						if textID >= 0 && textID < len(m.getCanvas().texts) {
							currentText := m.getCanvas().texts[textID]
							originalPos, hasOriginal := m.originalTextPositions[textID]
							if hasOriginal {
								deltaX := currentText.X - originalPos.X
								deltaY := currentText.Y - originalPos.Y
								if deltaX != 0 || deltaY != 0 {
									moveData := MoveTextData{ID: textID, DeltaX: deltaX, DeltaY: deltaY}
									originalState := OriginalTextState{ID: textID, X: originalPos.X, Y: originalPos.Y}
									m.recordAction(ActionMoveText, moveData, originalState)
								}
							}
						}
					}
				}
				// Handle single-item moves (backward compatibility)
				if m.selectedBox != -1 && m.selectedBox < len(m.getCanvas().boxes) {
					currentBox := m.getCanvas().boxes[m.selectedBox]
					// Calculate the total change from original position
					deltaX := currentBox.X - m.originalMoveX
					deltaY := currentBox.Y - m.originalMoveY

					// Only record if there was an actual change
					if deltaX != 0 || deltaY != 0 {
						moveData := MoveBoxData{ID: m.selectedBox, DeltaX: deltaX, DeltaY: deltaY}
						originalState := OriginalBoxState{ID: m.selectedBox, X: m.originalMoveX, Y: m.originalMoveY, Width: currentBox.Width, Height: currentBox.Height}
						m.recordAction(ActionMoveBox, moveData, originalState)
					}
				} else if m.selectedText != -1 && m.selectedText < len(m.getCanvas().texts) {
					currentText := m.getCanvas().texts[m.selectedText]
					// Calculate the total change from original position
					deltaX := currentText.X - m.originalTextMoveX
					deltaY := currentText.Y - m.originalTextMoveY

					// Only record if there was an actual change
					if deltaX != 0 || deltaY != 0 {
						moveData := MoveTextData{ID: m.selectedText, DeltaX: deltaX, DeltaY: deltaY}
						originalState := OriginalTextState{ID: m.selectedText, X: m.originalTextMoveX, Y: m.originalTextMoveY}
						m.recordAction(ActionMoveText, moveData, originalState)
					}
				}
				m.mode = ModeNormal
				m.selectedBox = -1
				m.selectedText = -1
				m.selectedBoxes = []int{}
				m.selectedTexts = []int{}
				m.selectedConnections = []int{}
				m.originalBoxPositions = make(map[int]point)
				m.originalTextPositions = make(map[int]point)
				m.originalConnections = make(map[int]Connection)
				return m, nil
			}

		case ModeFileInput:
			switch {
			case msg.Type == tea.KeyEscape:
				if m.fromStartup {
					// Return to startup mode if we came from there
					m.mode = ModeStartup
					m.fromStartup = false
				} else {
					m.mode = ModeNormal
				}
				m.filename = ""
				m.errorMessage = "" // Clear any error when canceling
				return m, nil
			case msg.String() == "up":
				// Navigate file list (only for FileOpOpen, and only if not typing and not showing delete confirmation)
				if m.fileOp == FileOpOpen && len(m.fileList) > 0 && !m.showingDeleteConfirm {
					// Only navigate if filename matches a file in the list (user hasn't started typing)
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
						// Update filename to match selected file (without .sav extension for display)
						selectedFile := m.fileList[m.selectedFileIndex]
						if strings.HasSuffix(strings.ToLower(selectedFile), ".sav") {
							m.filename = selectedFile[:len(selectedFile)-4]
						} else {
							m.filename = selectedFile
						}
						return m, nil
					}
				}
				// Fall through to treat as regular character if not navigating
			case msg.String() == "down":
				// Navigate file list (only for FileOpOpen, and only if not typing and not showing delete confirmation)
				if m.fileOp == FileOpOpen && len(m.fileList) > 0 && !m.showingDeleteConfirm {
					// Only navigate if filename matches a file in the list (user hasn't started typing)
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
						// Update filename to match selected file (without .sav extension for display)
						selectedFile := m.fileList[m.selectedFileIndex]
						if strings.HasSuffix(strings.ToLower(selectedFile), ".sav") {
							m.filename = selectedFile[:len(selectedFile)-4]
						} else {
							m.filename = selectedFile
						}
						return m, nil
					}
				}
				// Fall through to treat as regular character if not navigating
			case msg.String() == "d":
				// Delete selected chart (only for FileOpOpen, and only if a file is selected)
				if m.fileOp == FileOpOpen && len(m.fileList) > 0 && m.selectedFileIndex >= 0 && m.selectedFileIndex < len(m.fileList) && !m.showingDeleteConfirm {
					m.showingDeleteConfirm = true
					m.confirmFileIndex = m.selectedFileIndex
					return m, nil
				}
				// If not deleting, treat as regular character
				if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 {
					m.filename += string(msg.Runes)
					// Clear selection when typing
					m.selectedFileIndex = -1
				}
				return m, nil
			case msg.String() == "y" || msg.String() == "Y":
				// Confirm delete if we're showing delete confirmation
				if m.fileOp == FileOpOpen && m.showingDeleteConfirm {
					if m.confirmFileIndex >= 0 && m.confirmFileIndex < len(m.fileList) {
						filename := m.fileList[m.confirmFileIndex]
						// Apply save directory from config
						filepath := filename
						if m.config != nil {
							filepath = m.config.GetSavePath(filename)
						}
						err := os.Remove(filepath)
						if err != nil {
							m.errorMessage = fmt.Sprintf("Error deleting file: %s", err.Error())
						} else {
							// Remove file from the file list
							m.fileList = append(m.fileList[:m.confirmFileIndex], m.fileList[m.confirmFileIndex+1:]...)
							// Adjust selected index
							if m.selectedFileIndex >= len(m.fileList) {
								m.selectedFileIndex = len(m.fileList) - 1
							}
							if m.selectedFileIndex < 0 && len(m.fileList) > 0 {
								m.selectedFileIndex = 0
							}
							// Update filename to match new selection if any
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
				// If not showing delete confirmation, treat as regular character
				if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 {
					m.filename += string(msg.Runes)
					// Clear selection when typing
					m.selectedFileIndex = -1
				}
				return m, nil
			case msg.String() == "n" || msg.String() == "N":
				// Cancel delete if we're showing delete confirmation
				if m.fileOp == FileOpOpen && m.showingDeleteConfirm {
					m.showingDeleteConfirm = false
					return m, nil
				}
				// If not showing delete confirmation, treat as regular character
				if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 {
					m.filename += string(msg.Runes)
					// Clear selection when typing
					m.selectedFileIndex = -1
				}
				return m, nil
			case msg.Type == tea.KeyEscape:
				// Cancel delete confirmation if active, otherwise handle normal escape
				if m.fileOp == FileOpOpen && m.showingDeleteConfirm {
					m.showingDeleteConfirm = false
					return m, nil
				}
				// Normal escape handling for file input
				if m.fromStartup {
					m.mode = ModeStartup
					m.fromStartup = false
				} else {
					m.mode = ModeNormal
				}
				m.filename = ""
				m.errorMessage = "" // Clear any error when canceling
				return m, nil
			case msg.Type == tea.KeyEnter:
				// Don't allow file operations while showing delete confirmation
				if m.fileOp == FileOpOpen && m.showingDeleteConfirm {
					return m, nil
				}
				// Execute the file operation with automatic extension
				filename := m.filename
				// If we have a selected file and filename is empty or matches, use selected file
				if m.fileOp == FileOpOpen && len(m.fileList) > 0 && m.selectedFileIndex >= 0 && m.selectedFileIndex < len(m.fileList) {
					selectedFile := m.fileList[m.selectedFileIndex]
					if filename == "" || (strings.HasSuffix(strings.ToLower(selectedFile), ".sav") && filename == selectedFile[:len(selectedFile)-4]) {
						filename = selectedFile
					}
				}
				switch m.fileOp {
				case FileOpSave, FileOpOpen:
					if m.fileOp == FileOpSave {
						// Check if filename is empty before adding extension
						if strings.TrimSpace(m.filename) == "" {
							m.errorMessage = "Please enter a filename"
							return m, nil
						}
					}
					if !strings.HasSuffix(strings.ToLower(filename), ".sav") {
						filename += ".sav"
					}
					if m.fileOp == FileOpSave {
						// Apply save directory from config
						savePath := filename
						if m.config != nil {
							savePath = m.config.GetSavePath(filename)
						}
						// Check if file exists and show confirmation if it does (and confirmations are enabled)
						if _, err := os.Stat(savePath); err == nil {
							if m.config != nil && m.config.Confirmations {
								// File exists, show confirmation
								m.mode = ModeConfirm
								m.confirmAction = ConfirmOverwriteFile
								// Store filename for confirmation handler
								m.filename = savePath
								return m, nil
							}
							// Confirmations disabled, overwrite directly
						}
						// File doesn't exist or confirmations disabled, save directly
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
							// Update buffer filename
							buf := m.getCurrentBuffer()
							if buf != nil {
								buf.filename = savePath
							}
							absPath, _ := filepath.Abs(savePath)
							m.successMessage = fmt.Sprintf("Saved to %s", absPath)
							m.errorMessage = ""
						}
					} else {
						// Load file into a buffer
						// Check save directory first, then current directory
						loadPath := filename
						if m.config != nil && m.config.SaveDirectory != "" {
							saveDirPath := m.config.GetSavePath(filename)
							if _, err := os.Stat(saveDirPath); err == nil {
								loadPath = saveDirPath
							}
						}
						// Check if file exists
						if _, err := os.Stat(loadPath); os.IsNotExist(err) {
							m.errorMessage = fmt.Sprintf("File not found: %s", filename)
							return m, nil
						}
						newCanvas := NewCanvas()
						panX, panY, err := newCanvas.LoadFromFileWithPan(loadPath)
						if err != nil {
							m.errorMessage = fmt.Sprintf("Error opening file: %s", err.Error())
							return m, nil
						} else {
							// Update buffer filename with the actual path used
							if m.fromStartup {
								// Replace startup buffer
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
								// Add new buffer (capital O)
								m.addNewBufferWithPan(newCanvas, loadPath, panX, panY)
								m.openInNewBuffer = false
							} else {
								// Replace current buffer (lowercase o)
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
					// Extract just the base filename (in case user typed a path)
					baseFilename := filepath.Base(filename)
					if !strings.HasSuffix(strings.ToLower(baseFilename), ".png") {
						baseFilename += ".png"
					}
					// Apply save directory from config
					savePath := baseFilename
					if m.config != nil {
						savePath = m.config.GetSavePath(baseFilename)
					}
					// Get current buffer for pan offset and calculate render dimensions
					buf := m.getCurrentBuffer()
					panX, panY := 0, 0
					if buf != nil {
						panX, panY = buf.panX, buf.panY
					}
					// Calculate render dimensions - use current viewport size
					showBufferBar := m.mode != ModeStartup && len(m.buffers) > 1
					renderWidth := m.width
					if renderWidth < 1 {
						renderWidth = 80 // Default minimum width
					}
					var renderHeight int
					if showBufferBar {
						renderHeight = m.height - 2 // Leave room for buffer bar and status line
					} else {
						renderHeight = m.height - 1 // Leave room for status line only
					}
					if renderHeight < 1 {
						renderHeight = 24 // Default minimum height
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
					// Extract just the base filename (in case user typed a path)
					baseFilename := filepath.Base(filename)
					if !strings.HasSuffix(strings.ToLower(baseFilename), ".txt") {
						baseFilename += ".txt"
					}
					// Apply save directory from config
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
				// Don't allow editing while showing delete confirmation
				if m.fileOp == FileOpOpen && m.showingDeleteConfirm {
					return m, nil
				}
				if len(m.filename) > 0 {
					m.filename = m.filename[:len(m.filename)-1]
					// Clear selection when typing
					m.selectedFileIndex = -1
				}
				return m, nil
			case msg.Type == tea.KeySpace:
				// Don't allow editing while showing delete confirmation
				if m.fileOp == FileOpOpen && m.showingDeleteConfirm {
					return m, nil
				}
				// Insert space character
				m.filename += " "
				// Clear selection when typing
				m.selectedFileIndex = -1
				return m, nil
			default:
				// Don't allow typing while showing delete confirmation
				if m.fileOp == FileOpOpen && m.showingDeleteConfirm {
					return m, nil
				}
				// Handle all other keys as regular characters (including hjkl/HJKL)
				// Use msg.Runes for proper Unicode support and paste handling
				if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 {
					m.filename += string(msg.Runes)
					// Clear selection when typing
					m.selectedFileIndex = -1
				}
				return m, nil
			}

		case ModeConfirm:
			switch msg.String() {
			case "y", "Y":
				// Confirm the action
				switch m.confirmAction {
				case ConfirmDeleteBox:
					// Record the deletion for undo before actually deleting
					if m.confirmBoxID >= 0 && m.confirmBoxID < len(m.getCanvas().boxes) {
						box := m.getCanvas().boxes[m.confirmBoxID]
						// Find connections connected to this box
						connectedConnections := make([]Connection, 0)
						for _, connection := range m.getCanvas().connections {
							if connection.FromID == m.confirmBoxID || connection.ToID == m.confirmBoxID {
								connectedConnections = append(connectedConnections, connection)
							}
						}
						deleteData := DeleteBoxData{Box: box, ID: m.confirmBoxID, Connections: connectedConnections}
						addData := AddBoxData{X: box.X, Y: box.Y, Text: box.GetText(), ID: box.ID}
						m.recordAction(ActionDeleteBox, deleteData, addData)
					}
					m.getCanvas().DeleteBox(m.confirmBoxID)
					m.ensureCursorInBounds()
				case ConfirmDeleteText:
					// TODO: Add undo support for text deletion
					m.getCanvas().DeleteText(m.confirmTextID)
					m.ensureCursorInBounds()
				case ConfirmDeleteConnection:
					if m.confirmConnIdx >= 0 && m.confirmConnIdx < len(m.getCanvas().connections) {
						conn := m.getCanvas().connections[m.confirmConnIdx]
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
						// Record for undo
						highlightData := HighlightData{
							Cells: []HighlightCell{
								{
									X:        m.confirmHighlightX,
									Y:        m.confirmHighlightY,
									Color:    -1, // Removed
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
									Color:    oldColor, // Restore
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
						// Create new buffer (capital N)
						m.addNewBuffer(NewCanvas(), "")
					} else {
						// Replace current buffer (lowercase n)
						buf := m.getCurrentBuffer()
						if buf != nil {
							buf.canvas = NewCanvas()
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
					// Close current buffer
					if len(m.buffers) > 1 {
						// Determine which buffer to switch to (previous buffer)
						newIndex := m.currentBufferIndex - 1
						if newIndex < 0 {
							newIndex = 0 // If at first buffer, stay at first (which will be the next one after removal)
						}
						// Remove current buffer
						m.buffers = append(m.buffers[:m.currentBufferIndex], m.buffers[m.currentBufferIndex+1:]...)
						// Switch to the previous buffer
						m.currentBufferIndex = newIndex
					} else {
						// Last buffer - return to startup
						canvas := NewCanvas()
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
					// Overwrite existing file
					filename := m.filename
					// Apply save directory from config (already applied when setting filename, but ensure it's correct)
					if m.config != nil && m.config.SaveDirectory != "" {
						// Extract just the filename part if it's already a full path
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
						// Update buffer filename
						buf := m.getCurrentBuffer()
						if buf != nil {
							buf.filename = filename
						}
						absPath, _ := filepath.Abs(filename)
						m.successMessage = fmt.Sprintf("Saved to %s", absPath)
						m.errorMessage = ""
					}
				case ConfirmChooseExportType:
					// This case is handled by 'p' and 't' keys below
					return m, nil
				}
				m.mode = ModeNormal
				m.filename = ""
				return m, nil
			case "p", "P":
				// Export as PNG
				if m.confirmAction == ConfirmChooseExportType {
					m.mode = ModeFileInput
					m.fileOp = FileOpSavePNG
					// Auto-fill filename from buffer if it exists
					buf := m.getCurrentBuffer()
					if buf != nil && buf.filename != "" {
						// Extract just the base filename (without directory path)
						baseName := filepath.Base(buf.filename)
						// Remove .sav extension if present
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
				// Fall through for other confirmations
			case "t", "T":
				// Export as Visual TXT
				if m.confirmAction == ConfirmChooseExportType {
					m.mode = ModeFileInput
					m.fileOp = FileOpSaveVisualTXT
					// Auto-fill filename from buffer if it exists
					buf := m.getCurrentBuffer()
					if buf != nil && buf.filename != "" {
						// Extract just the base filename (without directory path)
						baseName := filepath.Base(buf.filename)
						// Remove .sav extension if present
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
				// Fall through for other confirmations
			case "n", "N", "escape":
				// Cancel the action
				if m.confirmAction == ConfirmOverwriteFile {
					// Return to file input mode if canceling overwrite
					m.mode = ModeFileInput
					m.fileOp = FileOpSave
				} else if m.confirmAction == ConfirmChooseExportType {
					// Return to normal mode if canceling export type selection
					m.mode = ModeNormal
				} else {
					m.mode = ModeNormal
				}
				return m, nil
			default:
				// Ignore other keys
				return m, nil
			}
		}
	}
	return m, nil
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

	// Ensure we have valid dimensions for rendering
	renderWidth := m.width
	if renderWidth < 1 {
		renderWidth = 1
	}

	// Calculate render height based on whether buffer bar will be shown
	// Buffer bar is shown when: not in startup mode AND more than one buffer
	showBufferBar := m.mode != ModeStartup && len(m.buffers) > 1
	var renderHeight int
	if showBufferBar {
		renderHeight = m.height - 2 // Leave room for buffer bar and status line
	} else {
		renderHeight = m.height - 1 // Leave room for status line only
	}
	if renderHeight < 1 {
		renderHeight = 1
	}

	// Prepare preview connection data if connection is in progress
	var previewFromX, previewFromY, previewToX, previewToY int = -1, -1, -1, -1
	var previewWaypoints []point
	if m.connectionFrom != -1 || m.connectionFromLine != -1 {
		previewFromX = m.connectionFromX
		previewFromY = m.connectionFromY
		previewWaypoints = m.connectionWaypoints
		// Convert cursor position to world coordinates for preview
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
	// Ensure cursor is in bounds before rendering
	cursorX := m.cursorX
	cursorY := m.cursorY

	// Validate cursor position against actual canvas size
	// Note: We validate against render dimensions, not canvas size
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

	// Determine if we should show the cursor
	// Determine if we should show the navigation cursor
	showCursor := (m.mode != ModeStartup && m.mode != ModeFileInput && m.mode != ModeEditing && m.mode != ModeTextInput)

	// Determine text editing cursor info
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
		// For text input, we need to show cursor at the input position
		editTextID = -1 // Text input creates new text, not editing existing
		editCursorPos = m.textInputCursorPos
		editText = m.textInputText
		editTextX = m.textInputX
		editTextY = m.textInputY
	}

	// Calculate selection rectangle parameters for rendering
	selectionStartX, selectionStartY := -1, -1
	selectionEndX, selectionEndY := -1, -1
	if m.mode == ModeMultiSelect && m.selectionStartX >= 0 && m.selectionStartY >= 0 {
		selectionStartX = m.selectionStartX
		selectionStartY = m.selectionStartY
		selectionEndX = m.cursorX + panX
		selectionEndY = m.cursorY + panY
	}

	showBoxNumbers := (m.mode == ModeBoxJump)
	canvas := m.getCanvas().Render(renderWidth, renderHeight, selectedBox, previewFromX, previewFromY, previewWaypoints, previewToX, previewToY, panX, panY, cursorX, cursorY, showCursor, editBoxID, editTextID, editCursorPos, editText, editTextX, editTextY, selectionStartX, selectionStartY, selectionEndX, selectionEndY, showBoxNumbers)

	// Build result with proper newlines
	var result strings.Builder

	// Buffer bar at the top (only show when more than one buffer)
	if showBufferBar {
		bufferBar := m.renderBufferBar(renderWidth)
		result.WriteString(bufferBar)
		result.WriteString("\n")
	}

	// Normal canvas display
	for i, line := range canvas {
		result.WriteString(line)
		if i < len(canvas)-1 {
			result.WriteString("\n")
		}
	}

	// Status line
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
		// Replace character at cursor position with cursor, don't insert
		var cursorDisplay string
		if len(displayText) == 0 {
			cursorDisplay = "█"
		} else if cursorPos >= len(displayText) {
			cursorDisplay = displayText + "█"
		} else {
			// Replace the character at cursor position with cursor
			runes := []rune(displayText)
			runes[cursorPos] = '█'
			cursorDisplay = string(runes)
		}
		if m.selectedBox != -1 {
			statusLine = fmt.Sprintf("Mode: EDIT | Box %d | Text: %s | ←/→/↑/↓=move cursor, Enter=newline, Ctrl+S=save, Esc=cancel", m.selectedBox, cursorDisplay)
		} else if m.selectedText != -1 {
			statusLine = fmt.Sprintf("Mode: EDIT | Text %d | Text: %s | ←/→/↑/↓=move cursor, Enter=newline, Ctrl+S=save, Esc=cancel", m.selectedText, cursorDisplay)
		} else {
			statusLine = fmt.Sprintf("Mode: EDIT | Text: %s | ←/→/↑/↓=move cursor, Enter=newline, Ctrl+S=save, Esc=cancel", cursorDisplay)
		}
	case ModeTextInput:
		displayText := strings.ReplaceAll(m.textInputText, "\n", " ")
		cursorPos := m.textInputCursorPos
		if cursorPos > len(displayText) {
			cursorPos = len(displayText)
		}
		// Replace character at cursor position with cursor, don't insert
		var cursorDisplay string
		if len(displayText) == 0 {
			cursorDisplay = "█"
		} else if cursorPos >= len(displayText) {
			cursorDisplay = displayText + "█"
		} else {
			// Replace the character at cursor position with cursor
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
	case ModeBoxJump:
		statusLine = fmt.Sprintf("Mode: BOX JUMP | Enter box number: %s | Enter=jump, Esc=cancel", m.boxJumpInput)
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
	// Only add status line if not in startup mode or centered file open mode
	if m.mode != ModeStartup && !(m.mode == ModeFileInput && m.fileOp == FileOpOpen) {
		result.WriteString("\n")
		result.WriteString(statusLine)
	}

	return result.String()
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
	case ModeCreating:
		return "CREATE"
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
	default:
		return "UNKNOWN"
	}
}

func (m model) helpView() string {
	helpLines := helpText

	// Calculate visible area
	visibleHeight := m.height - 1 // Leave room for status line
	if visibleHeight < 1 {
		visibleHeight = 1
	}

	// Apply scroll offset
	startLine := m.helpScroll
	endLine := startLine + visibleHeight

	// Ensure we don't scroll past the end
	if startLine >= len(helpLines) {
		startLine = len(helpLines) - visibleHeight
		if startLine < 0 {
			startLine = 0
		}
		m.helpScroll = startLine
		endLine = startLine + visibleHeight
	}

	if endLine > len(helpLines) {
		endLine = len(helpLines)
	}

	// Build visible help content
	var visibleLines []string
	if startLine < len(helpLines) {
		visibleLines = helpLines[startLine:endLine]
	}

	// Add scroll indicators if needed
	result := strings.Join(visibleLines, "\n")

	// Add status line showing scroll position
	statusLine := fmt.Sprintf("Help (%d-%d of %d lines) | j/k to scroll, Esc to close",
		startLine+1, endLine, len(helpLines))
	result += "\n" + statusLine

	return result
}
