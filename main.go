package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
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

type Buffer struct {
	canvas    *Canvas
	undoStack []Action
	redoStack []Action
	filename  string
	panX      int
	panY      int
}

type model struct {
	width               int
	height              int
	cursorX             int
	cursorY             int
	zPanMode            bool
	buffers             []Buffer
	currentBufferIndex  int
	mode                Mode
	help                bool
	helpScroll          int
	selectedBox         int
	selectedText        int
	editText            string
	editCursorPos       int
	connectionFrom      int
	connectionFromX     int
	connectionFromY     int
	connectionFromLine  int
	connectionWaypoints []struct{ X, Y int }
	filename            string
	fileList            []string
	selectedFileIndex   int
	fileOp              FileOperation
	openInNewBuffer     bool
	createNewBuffer     bool
	confirmAction       ConfirmAction
	confirmBoxID        int
	confirmTextID       int
	confirmConnIdx      int
	originalMoveX       int
	originalMoveY       int
	originalTextMoveX   int
	originalTextMoveY   int
	originalWidth       int
	originalHeight      int
	textInputX          int
	textInputY          int
	textInputText       string
	textInputCursorPos  int
	errorMessage        string
	successMessage      string
	fromStartup         bool
	clipboard           *Box
}

type Mode int

const (
	ModeStartup Mode = iota
	ModeNormal
	ModeCreating
	ModeEditing
	ModeTextInput
	ModeResize
	ModeMove
	ModeFileInput
	ModeConfirm
)

type FileOperation int

const (
	FileOpSave FileOperation = iota
	FileOpSavePNG
	FileOpOpen
)

type ConfirmAction int

const (
	ConfirmDeleteBox ConfirmAction = iota
	ConfirmDeleteText
	ConfirmDeleteConnection
	ConfirmQuit
	ConfirmNewChart
	ConfirmCloseBuffer
	ConfirmOverwriteFile
)

type Action struct {
	Type    ActionType
	Data    interface{}
	Inverse interface{}
}

type ActionType int

const (
	ActionAddBox ActionType = iota
	ActionDeleteBox
	ActionEditBox
	ActionEditText
	ActionResizeBox
	ActionMoveBox
	ActionMoveText
	ActionAddConnection
	ActionDeleteConnection
	ActionCycleArrow
)

type AddBoxData struct {
	X, Y int
	Text string
	ID   int
}

type DeleteBoxData struct {
	Box         Box
	ID          int
	Connections []Connection
}

type EditBoxData struct {
	ID      int
	NewText string
	OldText string
}

type EditTextData struct {
	ID      int
	NewText string
	OldText string
}

type ResizeBoxData struct {
	ID          int
	DeltaWidth  int
	DeltaHeight int
}

type MoveBoxData struct {
	ID     int
	DeltaX int
	DeltaY int
}

type MoveTextData struct {
	ID     int
	DeltaX int
	DeltaY int
}

type OriginalBoxState struct {
	ID     int
	X      int
	Y      int
	Width  int
	Height int
}

type OriginalTextState struct {
	ID int
	X  int
	Y  int
}

type AddConnectionData struct {
	FromID     int
	ToID       int
	Connection Connection
}

func (m *model) getCurrentBuffer() *Buffer {
	if len(m.buffers) == 0 {
		return nil
	}
	return &m.buffers[m.currentBufferIndex]
}

func (m *model) getCanvas() *Canvas {
	buf := m.getCurrentBuffer()
	if buf == nil {
		return nil
	}
	return buf.canvas
}

func (m *model) addNewBuffer(canvas *Canvas, filename string) {
	buffer := Buffer{
		canvas:    canvas,
		undoStack: []Action{},
		redoStack: []Action{},
		filename:  filename,
		panX:      0,
		panY:      0,
	}
	m.buffers = append(m.buffers, buffer)
	m.currentBufferIndex = len(m.buffers) - 1
}

func (m *model) renderBufferBar(width int) string {
	// Only show buffer bar when there is more than one buffer
	if len(m.buffers) <= 1 {
		return strings.Repeat(" ", width)
	}

	var bar strings.Builder
	bar.WriteString("Open Charts: ")

	for i, buf := range m.buffers {
		if i > 0 {
			bar.WriteString(" | ")
		}

		// Get buffer name
		bufName := fmt.Sprintf("%d", i+1)
		if buf.filename != "" {
			// Show filename without extension
			name := buf.filename
			if strings.HasSuffix(strings.ToLower(name), ".txt") {
				name = name[:len(name)-4]
			}
			bufName = name
		} else {
			bufName = fmt.Sprintf("Buffer %d", i+1)
		}

		// Highlight current buffer
		if i == m.currentBufferIndex {
			bar.WriteString("[")
			bar.WriteString(bufName)
			bar.WriteString("]")
		} else {
			bar.WriteString(bufName)
		}
	}

	// Pad to width
	currentLen := bar.Len()
	if currentLen < width {
		bar.WriteString(strings.Repeat(" ", width-currentLen))
	} else {
		// Truncate if too long
		return bar.String()[:width]
	}

	return bar.String()
}

func (m *model) recordAction(actionType ActionType, data, inverse interface{}) {
	buf := m.getCurrentBuffer()
	if buf == nil {
		return
	}
	action := Action{
		Type:    actionType,
		Data:    data,
		Inverse: inverse,
	}
	buf.undoStack = append(buf.undoStack, action)
	buf.redoStack = buf.redoStack[:0]
}

func (m *model) undo() {
	buf := m.getCurrentBuffer()
	if buf == nil || len(buf.undoStack) == 0 {
		return
	}

	lastIndex := len(buf.undoStack) - 1
	action := buf.undoStack[lastIndex]
	buf.undoStack = buf.undoStack[:lastIndex]

	switch action.Type {
	case ActionAddBox:
		data := action.Inverse.(DeleteBoxData)
		m.getCanvas().DeleteBox(data.ID)
	case ActionDeleteBox:
		data := action.Inverse.(AddBoxData)
		m.getCanvas().AddBoxWithID(data.X, data.Y, data.Text, data.ID)
		inverse := action.Data.(DeleteBoxData)
		for _, connection := range inverse.Connections {
			m.getCanvas().RestoreConnection(connection)
		}
	case ActionEditBox:
		data := action.Inverse.(EditBoxData)
		m.getCanvas().SetBoxText(data.ID, data.NewText)
	case ActionEditText:
		data := action.Inverse.(EditTextData)
		m.getCanvas().SetTextText(data.ID, data.NewText)
	case ActionResizeBox:
		data := action.Inverse.(OriginalBoxState)
		m.getCanvas().SetBoxSize(data.ID, data.Width, data.Height)
	case ActionMoveBox:
		data := action.Inverse.(OriginalBoxState)
		m.getCanvas().SetBoxPosition(data.ID, data.X, data.Y)
	case ActionMoveText:
		data := action.Inverse.(OriginalTextState)
		m.getCanvas().SetTextPosition(data.ID, data.X, data.Y)
	case ActionAddConnection:
		data := action.Inverse.(AddConnectionData)
		m.getCanvas().RemoveSpecificConnection(data.Connection)
	case ActionDeleteConnection:
		data := action.Inverse.(AddConnectionData)
		m.getCanvas().RestoreConnection(data.Connection)
	case ActionCycleArrow:
		cycleData := action.Inverse.(struct {
			ConnIdx int
			OldConn Connection
			NewConn Connection
		})
		if cycleData.ConnIdx >= 0 && cycleData.ConnIdx < len(m.getCanvas().connections) {
			m.getCanvas().connections[cycleData.ConnIdx] = cycleData.OldConn
		}
	}

	buf.redoStack = append(buf.redoStack, action)
}

func (m *model) redo() {
	buf := m.getCurrentBuffer()
	if buf == nil || len(buf.redoStack) == 0 {
		return
	}

	lastIndex := len(buf.redoStack) - 1
	action := buf.redoStack[lastIndex]
	buf.redoStack = buf.redoStack[:lastIndex]

	switch action.Type {
	case ActionAddBox:
		data := action.Data.(AddBoxData)
		m.getCanvas().AddBoxWithID(data.X, data.Y, data.Text, data.ID)
	case ActionDeleteBox:
		data := action.Data.(DeleteBoxData)
		m.getCanvas().DeleteBox(data.ID)
	case ActionEditBox:
		data := action.Data.(EditBoxData)
		m.getCanvas().SetBoxText(data.ID, data.NewText)
	case ActionEditText:
		data := action.Data.(EditTextData)
		m.getCanvas().SetTextText(data.ID, data.NewText)
	case ActionResizeBox:
		data := action.Data.(ResizeBoxData)
		m.getCanvas().ResizeBox(data.ID, data.DeltaWidth, data.DeltaHeight)
	case ActionMoveBox:
		data := action.Data.(MoveBoxData)
		m.getCanvas().MoveBox(data.ID, data.DeltaX, data.DeltaY)
	case ActionMoveText:
		data := action.Data.(MoveTextData)
		m.getCanvas().MoveText(data.ID, data.DeltaX, data.DeltaY)
	case ActionAddConnection:
		data := action.Data.(AddConnectionData)
		m.getCanvas().RestoreConnection(data.Connection)
	case ActionDeleteConnection:
		data := action.Data.(AddConnectionData)
		m.getCanvas().RemoveSpecificConnection(data.Connection)
	case ActionCycleArrow:
		cycleData := action.Data.(struct {
			ConnIdx int
			OldConn Connection
			NewConn Connection
		})
		if cycleData.ConnIdx >= 0 && cycleData.ConnIdx < len(m.getCanvas().connections) {
			m.getCanvas().connections[cycleData.ConnIdx] = cycleData.NewConn
		}
	}

	buf.undoStack = append(buf.undoStack, action)
}

func initialModel() model {
	canvas := NewCanvas()
	canvas.AddBox(1, 1, "Welcome to Flerm!\nby Travis\n\n'n' New flowchart\n'o' Open existing chart\n'q' Quit")

	buffer := Buffer{
		canvas:    canvas,
		undoStack: []Action{},
		redoStack: []Action{},
		filename:  "",
		panX:      0,
		panY:      0,
	}

	return model{
		buffers:            []Buffer{buffer},
		currentBufferIndex: 0,
		mode:               ModeStartup,
		selectedBox:        -1,
		selectedText:       -1,
		connectionFrom:     -1,
		connectionFromLine: -1,
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
	// Leave room for status line
	maxY := m.height - 2
	if maxY < 0 {
		maxY = 0
	}
	if m.cursorY > maxY {
		m.cursorY = maxY
	}
}

func (m *model) scanTxtFiles() {
	m.fileList = []string{}

	// Get current directory
	dir, err := os.Getwd()
	if err != nil {
		m.selectedFileIndex = -1
		return
	}

	// Read directory
	entries, err := os.ReadDir(dir)
	if err != nil {
		m.selectedFileIndex = -1
		return
	}

	// Filter .txt files
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".txt") {
			m.fileList = append(m.fileList, entry.Name())
		}
	}

	// Sort files alphabetically
	sort.Strings(m.fileList)

	// Set initial selection
	if len(m.fileList) > 0 {
		m.selectedFileIndex = 0
		// Set filename to first file (without extension)
		firstFile := m.fileList[0]
		if strings.HasSuffix(strings.ToLower(firstFile), ".txt") {
			m.filename = firstFile[:len(firstFile)-4]
		} else {
			m.filename = firstFile
		}
	} else {
		m.selectedFileIndex = -1
	}
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
				helpLines := []string{
					"Fl(ow)(T)erm Help",
					"=================",
					"",
					"",
					"Navigation:",
					"-----------",
					"  h/←/j/↓/k/↑/l/→  Move cursor around the screen",
					"  Shift+h/j/k/l    Move cursor 2x faster (hold Shift with direction keys)",
					"",
					"Box Operations:",
					"---------------",
					"  b                Create new box at cursor position",
					"  e                Edit text in box under cursor",
					"  r                Resize box under cursor",
					"  m                Move box under cursor",
					"  d                Delete box under cursor",
					"  c                Copy box under cursor",
					"  p                Paste copied box at cursor position",
					"",
					"Text Operations:",
					"----------------",
					"  t                Enter text at cursor position",
					"  e                Edit text under cursor",
					"  m                Move text under cursor",
					"  d                Delete text under cursor",
					"",
					"Resize Mode:",
					"------------",
					"  h/←/j/↓/k/↑/l/→  Resize box",
					"  Shift+h/j/k/l    Resize box 2x faster",
					"  Enter            Confirm edit and return to normal mode",
					"  Esc   	        Canceland return to normal mode",
					"",
					"Move Mode:",
					"----------",
					"  h/←/j/↓/k/↑/l/→  Move object around the screen",
					"  Shift+h/j/k/l    Move object 2x faster",
					"  Enter            Finish moving and return to normal mode",
					"  Escape           Cancel move and return to normal mode",
					"",
					"Note: Selected boxes (being resized/moved) are highlighted with # borders",
					"Another note: Resizing/moving boxes with connections is a little wonky,",
					"              when in doubt, delete the lines and re-connect the boxes ¯\\_(ツ)_/¯",
					"",
					"Connection Operations:",
					"---------------------",
					"  a                Start/finish connection creation",
					"                   - Press 'a' on a box or line to start",
					"                   - Press 'a' on empty space to add waypoint",
					"                   - Press 'a' on a box or line to finish",
					"                   - Connections can start/end at boxes or lines",
					"  A                Toggle arrow state on connection line under cursor",
					"                   - Cycles through: no arrows → to arrow → from arrow → both arrows",
					"                   Note: Sometimes the arrows flip around. Redrawing the line fixes it.",
					"",
					"File Operations:",
					"----------------",
					"  s                Save flowchart",
					"  S                Export as PNG image (experimental and janky)",
					"  o                Load a saved flowchart in current buffer",
					"  O                Load a saved flowchart in new buffer",
					"",
					"Buffer Operations:",
					"------------------",
					"  {                Switch to previous buffer",
					"  }                Switch to next buffer",
					"  n                Create new chart in current buffer",
					"  N                Create new chart in new buffer",
					"  x                Close current buffer",
					"",
					"General:",
					"  u                Undo last action",
					"  U                Redo last undone action",
					"  Esc           	Clear selection/cancel current operation",
					"  ?                Toggle this help screen",
					"  q/Ctrl+C         Quit Flerm",
					"",
					"========== Thanks for trying Flerm! ==========",
				}
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
				m.mode = ModeConfirm
				m.confirmAction = ConfirmQuit
				return m, nil
			case "n":
				m.mode = ModeConfirm
				m.confirmAction = ConfirmNewChart
				m.createNewBuffer = false // Replace current buffer
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
			case "h", "left":
				if m.zPanMode {
					// Pan left (z+arrow or z+h)
					buf := m.getCurrentBuffer()
					if buf != nil {
						buf.panX++
					}
				} else {
					m.cursorX--
					m.ensureCursorInBounds()
				}
				return m, nil
			case "H", "shift+h":
				// Move cursor 2x faster left
				if m.zPanMode {
					// Pan left (z+H)
					buf := m.getCurrentBuffer()
					if buf != nil {
						buf.panX++
					}
				} else {
					m.cursorX -= 2
					m.ensureCursorInBounds()
				}
				return m, nil
			case "shift+left":
				// Move cursor 2x faster left
				if m.zPanMode {
					// Pan left (z+shift+left)
					buf := m.getCurrentBuffer()
					if buf != nil {
						buf.panX++
					}
				} else {
					m.cursorX -= 2
					m.ensureCursorInBounds()
				}
				return m, nil
			case "l", "right":
				if m.zPanMode {
					// Pan right (z+arrow or z+l)
					buf := m.getCurrentBuffer()
					if buf != nil {
						buf.panX--
					}
				} else {
					m.cursorX++
					m.ensureCursorInBounds()
				}
				return m, nil
			case "L", "shift+l":
				// Move cursor 2x faster right
				if m.zPanMode {
					// Pan right (z+L)
					buf := m.getCurrentBuffer()
					if buf != nil {
						buf.panX--
					}
				} else {
					m.cursorX += 2
					m.ensureCursorInBounds()
				}
				return m, nil
			case "shift+right":
				// Move cursor 2x faster right
				if m.zPanMode {
					// Pan right (z+shift+right)
					buf := m.getCurrentBuffer()
					if buf != nil {
						buf.panX--
					}
				} else {
					m.cursorX += 2
					m.ensureCursorInBounds()
				}
				return m, nil
			case "k", "up":
				if m.zPanMode {
					// Pan up (z+arrow or z+k)
					buf := m.getCurrentBuffer()
					if buf != nil {
						buf.panY++
					}
				} else {
					m.cursorY--
					m.ensureCursorInBounds()
				}
				return m, nil
			case "K", "shift+k":
				// Move cursor 2x faster up
				if m.zPanMode {
					// Pan up (z+K)
					buf := m.getCurrentBuffer()
					if buf != nil {
						buf.panY++
					}
				} else {
					m.cursorY -= 2
					m.ensureCursorInBounds()
				}
				return m, nil
			case "shift+up":
				// Move cursor 2x faster up
				if m.zPanMode {
					// Pan up (z+shift+up)
					buf := m.getCurrentBuffer()
					if buf != nil {
						buf.panY++
					}
				} else {
					m.cursorY -= 2
					m.ensureCursorInBounds()
				}
				return m, nil
			case "j", "down":
				if m.zPanMode {
					// Pan down (z+arrow or z+j)
					buf := m.getCurrentBuffer()
					if buf != nil {
						buf.panY--
					}
				} else {
					m.cursorY++
					m.ensureCursorInBounds()
				}
				return m, nil
			case "J", "shift+j":
				// Move cursor 2x faster down (same as shift+down)
				if m.zPanMode {
					// Pan down (z+J)
					buf := m.getCurrentBuffer()
					if buf != nil {
						buf.panY--
					}
				} else {
					m.cursorY += 2
					m.ensureCursorInBounds()
				}
				return m, nil
			case "shift+down":
				// Move cursor 2x faster down
				if m.zPanMode {
					// Pan down (z+shift+down)
					buf := m.getCurrentBuffer()
					if buf != nil {
						buf.panY--
					}
				} else {
					m.cursorY += 2
					m.ensureCursorInBounds()
				}
				return m, nil
			case "z":
				// Toggle z-pan mode (acts like holding z)
				m.zPanMode = !m.zPanMode
				return m, nil
			case "b":
				// Clear z-pan mode when other keys are pressed
				m.zPanMode = false
				boxID := len(m.getCanvas().boxes)
				buf := m.getCurrentBuffer()
				panX, panY := 0, 0
				if buf != nil {
					panX, panY = buf.panX, buf.panY
				}
				worldX := m.cursorX + panX
				worldY := m.cursorY + panY
				m.getCanvas().AddBox(worldX, worldY, "Box")
				addData := AddBoxData{X: worldX, Y: worldY, Text: "Box", ID: boxID}
				deleteData := DeleteBoxData{ID: boxID, Connections: nil}
				m.recordAction(ActionAddBox, addData, deleteData)
				m.successMessage = ""
				m.ensureCursorInBounds()
				return m, nil
			case "t":
				m.zPanMode = false
				m.mode = ModeTextInput
				m.textInputX = m.cursorX
				m.textInputY = m.cursorY
				m.textInputText = ""
				m.textInputCursorPos = 0
				return m, nil
			case "r":
				m.zPanMode = false
				buf := m.getCurrentBuffer()
				panX, panY := 0, 0
				if buf != nil {
					panX, panY = buf.panX, buf.panY
				}
				worldX := m.cursorX + panX
				worldY := m.cursorY + panY
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
				buf := m.getCurrentBuffer()
				panX, panY := 0, 0
				if buf != nil {
					panX, panY = buf.panX, buf.panY
				}
				worldX := m.cursorX + panX
				worldY := m.cursorY + panY
				boxID := m.getCanvas().GetBoxAt(worldX, worldY)
				textID := m.getCanvas().GetTextAt(worldX, worldY)
				if boxID != -1 {
					m.selectedBox = boxID
					m.selectedText = -1
					if boxID < len(m.getCanvas().boxes) {
						m.originalMoveX = m.getCanvas().boxes[boxID].X
						m.originalMoveY = m.getCanvas().boxes[boxID].Y
					}
					m.mode = ModeMove
				} else if textID != -1 {
					m.selectedText = textID
					m.selectedBox = -1
					if textID < len(m.getCanvas().texts) {
						m.originalTextMoveX = m.getCanvas().texts[textID].X
						m.originalTextMoveY = m.getCanvas().texts[textID].Y
					}
					m.mode = ModeMove
				}
				return m, nil
			case "e":
				buf := m.getCurrentBuffer()
				panX, panY := 0, 0
				if buf != nil {
					panX, panY = buf.panX, buf.panY
				}
				worldX := m.cursorX + panX
				worldY := m.cursorY + panY
				boxID := m.getCanvas().GetBoxAt(worldX, worldY)
				textID := m.getCanvas().GetTextAt(worldX, worldY)
				if boxID != -1 {
					m.selectedBox = boxID
					m.selectedText = -1
					m.mode = ModeEditing
					m.editText = m.getCanvas().GetBoxText(boxID)
					m.editCursorPos = len(m.editText)
				} else if textID != -1 {
					m.selectedText = textID
					m.selectedBox = -1
					m.mode = ModeEditing
					m.editText = m.getCanvas().GetTextText(textID)
					m.editCursorPos = len(m.editText)
				}
				return m, nil
			case "A":
				// Convert cursor position to world coordinates
				buf := m.getCurrentBuffer()
				panX, panY := 0, 0
				if buf != nil {
					panX, panY = buf.panX, buf.panY
				}
				worldX := m.cursorX + panX
				worldY := m.cursorY + panY
				lineConnIdx, _, _ := m.getCanvas().findNearestPointOnConnection(worldX, worldY)
				if lineConnIdx != -1 {
					oldConn := m.getCanvas().connections[lineConnIdx]
					m.getCanvas().CycleConnectionArrowState(lineConnIdx)
					newConn := m.getCanvas().connections[lineConnIdx]
					cycleData := struct {
						ConnIdx int
						OldConn Connection
						NewConn Connection
					}{lineConnIdx, oldConn, newConn}
					m.recordAction(ActionCycleArrow, cycleData, cycleData)
					m.successMessage = ""
				}
				return m, nil
			case "a":
				buf := m.getCurrentBuffer()
				panX, panY := 0, 0
				if buf != nil {
					panX, panY = buf.panX, buf.panY
				}
				worldX := m.cursorX + panX
				worldY := m.cursorY + panY
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
						m.connectionWaypoints = append(m.connectionWaypoints, struct{ X, Y int }{worldX, worldY})
					}
				}
				return m, nil
			case "d":
				buf := m.getCurrentBuffer()
				panX, panY := 0, 0
				if buf != nil {
					panX, panY = buf.panX, buf.panY
				}
				worldX := m.cursorX + panX
				worldY := m.cursorY + panY
				lineConnIdx, _, _ := m.getCanvas().findNearestPointOnConnection(worldX, worldY)
				if lineConnIdx != -1 {
					m.mode = ModeConfirm
					m.confirmAction = ConfirmDeleteConnection
					m.confirmConnIdx = lineConnIdx
				} else {
					boxID := m.getCanvas().GetBoxAt(worldX, worldY)
					textID := m.getCanvas().GetTextAt(worldX, worldY)

					if boxID != -1 {
						m.mode = ModeConfirm
						m.confirmAction = ConfirmDeleteBox
						m.confirmBoxID = boxID
					} else if textID != -1 {
						m.mode = ModeConfirm
						m.confirmAction = ConfirmDeleteText
						m.confirmTextID = textID
					}
				}
				return m, nil
			case "s":
				m.mode = ModeFileInput
				m.fileOp = FileOpSave
				// Auto-fill filename from buffer if it exists
				buf := m.getCurrentBuffer()
				if buf != nil && buf.filename != "" {
					// Remove .txt extension for display
					filename := buf.filename
					if strings.HasSuffix(strings.ToLower(filename), ".txt") {
						filename = filename[:len(filename)-4]
					}
					m.filename = filename
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
				m.mode = ModeFileInput
				m.fileOp = FileOpSavePNG
				m.filename = "flowchart"
				m.errorMessage = ""
				m.successMessage = ""
				m.fromStartup = false
				return m, nil
			case "x":
				// Close current buffer (with confirmation)
				if len(m.buffers) > 0 {
					m.mode = ModeConfirm
					m.confirmAction = ConfirmCloseBuffer
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
				buf := m.getCurrentBuffer()
				panX, panY := 0, 0
				if buf != nil {
					panX, panY = buf.panX, buf.panY
				}
				worldX := m.cursorX + panX
				worldY := m.cursorY + panY
				boxID := m.getCanvas().GetBoxAt(worldX, worldY)
				if boxID != -1 && boxID < len(m.getCanvas().boxes) {
					// Copy the box
					box := m.getCanvas().boxes[boxID]
					// Create a deep copy
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
					// Paste the box at cursor position
					boxID := len(m.getCanvas().boxes)
					text := m.clipboard.GetText()
					buf := m.getCurrentBuffer()
					panX, panY := 0, 0
					if buf != nil {
						panX, panY = buf.panX, buf.panY
					}
					worldX := m.cursorX + panX
					worldY := m.cursorY + panY
					m.getCanvas().AddBox(worldX, worldY, text)
					// Set the size to match the copied box (in case it was manually resized)
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
				m.connectionFrom = -1
				m.connectionFromLine = -1
				m.connectionFromX = 0
				m.connectionFromY = 0
				m.connectionWaypoints = nil
				m.selectedBox = -1
				return m, nil
			}

		case ModeEditing:
			switch {
			case msg.Type == tea.KeyEscape:
				m.mode = ModeNormal
				m.editText = ""
				m.editCursorPos = 0
				m.selectedBox = -1
				m.selectedText = -1
				return m, nil
			case msg.Type == tea.KeyCtrlS:
				if m.selectedBox != -1 {
					oldText := m.getCanvas().GetBoxText(m.selectedBox)
					m.getCanvas().SetBoxText(m.selectedBox, m.editText)
					editData := EditBoxData{ID: m.selectedBox, NewText: m.editText, OldText: oldText}
					inverseData := EditBoxData{ID: m.selectedBox, NewText: oldText, OldText: m.editText}
					m.recordAction(ActionEditBox, editData, inverseData)
				} else if m.selectedText != -1 {
					oldText := m.getCanvas().GetTextText(m.selectedText)
					m.getCanvas().SetTextText(m.selectedText, m.editText)
					editData := EditTextData{ID: m.selectedText, NewText: m.editText, OldText: oldText}
					inverseData := EditTextData{ID: m.selectedText, NewText: oldText, OldText: m.editText}
					m.recordAction(ActionEditText, editData, inverseData)
				}
				m.mode = ModeNormal
				m.editText = ""
				m.editCursorPos = 0
				m.selectedBox = -1
				m.selectedText = -1
				return m, nil
			case msg.String() == "left":
				if m.editCursorPos > 0 {
					m.editCursorPos--
				}
				return m, nil
			case msg.String() == "right":
				if m.editCursorPos < len(m.editText) {
					m.editCursorPos++
				}
				return m, nil
			case msg.Type == tea.KeyEnter:
				m.editText = m.editText[:m.editCursorPos] + "\n" + m.editText[m.editCursorPos:]
				m.editCursorPos++
				return m, nil
			case msg.Type == tea.KeyBackspace:
				if m.editCursorPos > 0 {
					m.editText = m.editText[:m.editCursorPos-1] + m.editText[m.editCursorPos:]
					m.editCursorPos--
				}
				return m, nil
			case msg.Type == tea.KeyDelete:
				if m.editCursorPos < len(m.editText) {
					m.editText = m.editText[:m.editCursorPos] + m.editText[m.editCursorPos+1:]
				}
				return m, nil
			default:
				keyStr := msg.String()
				if len(keyStr) == 1 {
					m.editText = m.editText[:m.editCursorPos] + keyStr + m.editText[m.editCursorPos:]
					m.editCursorPos++
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
					buf := m.getCurrentBuffer()
					panX, panY := 0, 0
					if buf != nil {
						panX, panY = buf.panX, buf.panY
					}
					worldX := m.textInputX + panX
					worldY := m.textInputY + panY
					m.getCanvas().AddText(worldX, worldY, m.textInputText)
				}
				m.mode = ModeNormal
				m.textInputText = ""
				m.textInputCursorPos = 0
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
			default:
				keyStr := msg.String()
				if len(keyStr) == 1 {
					m.textInputText = m.textInputText[:m.textInputCursorPos] + keyStr + m.textInputText[m.textInputCursorPos:]
					m.textInputCursorPos++
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

		case ModeMove:
			switch msg.String() {
			case "escape":
				m.mode = ModeNormal
				m.selectedBox = -1
				m.selectedText = -1
				return m, nil
			case "h", "left":
				if m.selectedBox != -1 {
					m.getCanvas().MoveBox(m.selectedBox, -1, 0)
					m.ensureCursorInBounds()
				} else if m.selectedText != -1 {
					m.getCanvas().MoveText(m.selectedText, -1, 0)
					m.ensureCursorInBounds()
				}
				return m, nil
			case "H", "shift+left":
				if m.selectedBox != -1 {
					m.getCanvas().MoveBox(m.selectedBox, -2, 0)
					m.ensureCursorInBounds()
				} else if m.selectedText != -1 {
					m.getCanvas().MoveText(m.selectedText, -2, 0)
					m.ensureCursorInBounds()
				}
				return m, nil
			case "l", "right":
				if m.selectedBox != -1 {
					m.getCanvas().MoveBox(m.selectedBox, 1, 0)
					m.ensureCursorInBounds()
				} else if m.selectedText != -1 {
					m.getCanvas().MoveText(m.selectedText, 1, 0)
					m.ensureCursorInBounds()
				}
				return m, nil
			case "L", "shift+right":
				if m.selectedBox != -1 {
					m.getCanvas().MoveBox(m.selectedBox, 2, 0)
					m.ensureCursorInBounds()
				} else if m.selectedText != -1 {
					m.getCanvas().MoveText(m.selectedText, 2, 0)
					m.ensureCursorInBounds()
				}
				return m, nil
			case "k", "up":
				if m.selectedBox != -1 {
					m.getCanvas().MoveBox(m.selectedBox, 0, -1)
					m.ensureCursorInBounds()
				} else if m.selectedText != -1 {
					m.getCanvas().MoveText(m.selectedText, 0, -1)
					m.ensureCursorInBounds()
				}
				return m, nil
			case "K", "shift+up":
				if m.selectedBox != -1 {
					m.getCanvas().MoveBox(m.selectedBox, 0, -2)
					m.ensureCursorInBounds()
				} else if m.selectedText != -1 {
					m.getCanvas().MoveText(m.selectedText, 0, -2)
					m.ensureCursorInBounds()
				}
				return m, nil
			case "j", "down":
				if m.selectedBox != -1 {
					m.getCanvas().MoveBox(m.selectedBox, 0, 1)
					m.ensureCursorInBounds()
				} else if m.selectedText != -1 {
					m.getCanvas().MoveText(m.selectedText, 0, 1)
					m.ensureCursorInBounds()
				}
				return m, nil
			case "J", "shift+down":
				if m.selectedBox != -1 {
					m.getCanvas().MoveBox(m.selectedBox, 0, 2)
					m.ensureCursorInBounds()
				} else if m.selectedText != -1 {
					m.getCanvas().MoveText(m.selectedText, 0, 2)
					m.ensureCursorInBounds()
				}
				return m, nil
			case "enter":
				// Record the move action when finishing move mode
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
				// Navigate file list (only for FileOpOpen, and only if not typing)
				if m.fileOp == FileOpOpen && len(m.fileList) > 0 {
					// Only navigate if filename matches a file in the list (user hasn't started typing)
					matchesFile := false
					if m.selectedFileIndex >= 0 && m.selectedFileIndex < len(m.fileList) {
						selectedFile := m.fileList[m.selectedFileIndex]
						fileDisplayName := selectedFile
						if strings.HasSuffix(strings.ToLower(selectedFile), ".txt") {
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
						// Update filename to match selected file (without .txt extension for display)
						selectedFile := m.fileList[m.selectedFileIndex]
						if strings.HasSuffix(strings.ToLower(selectedFile), ".txt") {
							m.filename = selectedFile[:len(selectedFile)-4]
						} else {
							m.filename = selectedFile
						}
						return m, nil
					}
				}
				// Fall through to treat as regular character if not navigating
			case msg.String() == "down":
				// Navigate file list (only for FileOpOpen, and only if not typing)
				if m.fileOp == FileOpOpen && len(m.fileList) > 0 {
					// Only navigate if filename matches a file in the list (user hasn't started typing)
					matchesFile := false
					if m.selectedFileIndex >= 0 && m.selectedFileIndex < len(m.fileList) {
						selectedFile := m.fileList[m.selectedFileIndex]
						fileDisplayName := selectedFile
						if strings.HasSuffix(strings.ToLower(selectedFile), ".txt") {
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
						// Update filename to match selected file (without .txt extension for display)
						selectedFile := m.fileList[m.selectedFileIndex]
						if strings.HasSuffix(strings.ToLower(selectedFile), ".txt") {
							m.filename = selectedFile[:len(selectedFile)-4]
						} else {
							m.filename = selectedFile
						}
						return m, nil
					}
				}
				// Fall through to treat as regular character if not navigating
			case msg.Type == tea.KeyEnter:
				// Execute the file operation with automatic extension
				filename := m.filename
				// If we have a selected file and filename is empty or matches, use selected file
				if m.fileOp == FileOpOpen && len(m.fileList) > 0 && m.selectedFileIndex >= 0 && m.selectedFileIndex < len(m.fileList) {
					selectedFile := m.fileList[m.selectedFileIndex]
					if filename == "" || (strings.HasSuffix(strings.ToLower(selectedFile), ".txt") && filename == selectedFile[:len(selectedFile)-4]) {
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
					if !strings.HasSuffix(strings.ToLower(filename), ".txt") {
						filename += ".txt"
					}
					if m.fileOp == FileOpSave {
						// Check if file exists and show confirmation if it does
						if _, err := os.Stat(filename); err == nil {
							// File exists, show confirmation
							m.mode = ModeConfirm
							m.confirmAction = ConfirmOverwriteFile
							// Store filename for confirmation handler
							m.filename = filename
							return m, nil
						}
						// File doesn't exist, save directly
						err := m.getCanvas().SaveToFile(filename)
						if err != nil {
							m.errorMessage = fmt.Sprintf("Error saving file: %s", err.Error())
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
					} else {
						// Load file into a buffer
						newCanvas := NewCanvas()
						err := newCanvas.LoadFromFile(filename)
						if err != nil {
							m.errorMessage = fmt.Sprintf("Error opening file: %s", err.Error())
							return m, nil
						} else {
							if m.fromStartup {
								// Replace startup buffer
								m.buffers[0] = Buffer{
									canvas:    newCanvas,
									undoStack: []Action{},
									redoStack: []Action{},
									filename:  filename,
									panX:      0,
									panY:      0,
								}
								m.currentBufferIndex = 0
								m.fromStartup = false
							} else if m.openInNewBuffer {
								// Add new buffer (capital O)
								m.addNewBuffer(newCanvas, filename)
								m.openInNewBuffer = false
							} else {
								// Replace current buffer (lowercase o)
								buf := m.getCurrentBuffer()
								if buf != nil {
									buf.canvas = newCanvas
									buf.filename = filename
									buf.undoStack = []Action{}
									buf.redoStack = []Action{}
								}
							}
							m.errorMessage = ""
						}
					}
				case FileOpSavePNG:
					if !strings.HasSuffix(strings.ToLower(filename), ".png") {
						filename += ".png"
					}
					err := m.getCanvas().ExportToPNG(filename, 800, 600)
					if err != nil {
						m.errorMessage = fmt.Sprintf("Error exporting PNG: %s", err.Error())
						return m, nil
					} else {
						absPath, _ := filepath.Abs(filename)
						m.successMessage = fmt.Sprintf("Exported to %s", absPath)
						m.errorMessage = ""
					}
				}
				m.mode = ModeNormal
				m.filename = ""
				return m, nil
			case msg.Type == tea.KeyBackspace:
				if len(m.filename) > 0 {
					m.filename = m.filename[:len(m.filename)-1]
					// Clear selection when typing
					m.selectedFileIndex = -1
				}
				return m, nil
			default:
				// Handle all other keys as regular characters (including hjkl/HJKL)
				keyStr := msg.String()
				// Only process single character keys, not special keys like "shift+left"
				if len(keyStr) == 1 {
					m.filename += keyStr
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
						canvas.AddBox(1, 1, "Welcome to Flerm!\nby Travis\n\n'n' New flowchart\n'o' Open existing chart\n'q' Quit")
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
					err := m.getCanvas().SaveToFile(filename)
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
				}
				m.mode = ModeNormal
				m.filename = ""
				return m, nil
			case "n", "N", "escape":
				// Cancel the action
				if m.confirmAction == ConfirmOverwriteFile {
					// Return to file input mode if canceling overwrite
					m.mode = ModeFileInput
					m.fileOp = FileOpSave
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
	var previewWaypoints []struct{ X, Y int }
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
	canvas := m.getCanvas().Render(renderWidth, renderHeight, selectedBox, previewFromX, previewFromY, previewWaypoints, previewToX, previewToY, panX, panY)

	// Ensure cursor is in bounds before rendering
	cursorX := m.cursorX
	cursorY := m.cursorY

	// Validate cursor position against actual canvas size
	if cursorY >= len(canvas) {
		cursorY = len(canvas) - 1
	}
	if cursorY < 0 {
		cursorY = 0
	}
	if cursorY < len(canvas) && cursorX >= len(canvas[cursorY]) {
		if len(canvas[cursorY]) > 0 {
			cursorX = len(canvas[cursorY]) - 1
		} else {
			cursorX = 0
		}
	}
	if cursorX < 0 {
		cursorX = 0
	}

	// Add cursor (except in startup mode and file input mode)
	if m.mode != ModeStartup && m.mode != ModeFileInput && cursorY < len(canvas) && cursorX < len(canvas[cursorY]) {
		line := []rune(canvas[cursorY])
		if cursorX < len(line) {
			line[cursorX] = '█'
			canvas[cursorY] = string(line)
		}
	}

	// Build result with proper newlines
	var result strings.Builder

	// Buffer bar at the top (only show when more than one buffer)
	if showBufferBar {
		bufferBar := m.renderBufferBar(renderWidth)
		result.WriteString(bufferBar)
		result.WriteString("\n")
	}

	// If in FileOpOpen mode, show file list instead of canvas
	if m.mode == ModeFileInput && m.fileOp == FileOpOpen {
		result.WriteString("Select a saved chart:\n")
		result.WriteString(strings.Repeat("─", renderWidth))
		result.WriteString("\n")

		if len(m.fileList) == 0 {
			result.WriteString("(No .txt files found in current directory)\n")
		} else {
			// Calculate how many files we can show
			maxFiles := renderHeight - 4 // Leave room for header, separator, input, and status
			if maxFiles < 1 {
				maxFiles = 1
			}

			// Determine start index for scrolling
			startIdx := 0
			if m.selectedFileIndex >= 0 && m.selectedFileIndex >= maxFiles {
				startIdx = m.selectedFileIndex - maxFiles + 1
			}
			endIdx := startIdx + maxFiles
			if endIdx > len(m.fileList) {
				endIdx = len(m.fileList)
			}

			// Display files
			for i := startIdx; i < endIdx; i++ {
				file := m.fileList[i]
				// Remove .txt extension for display
				displayName := file
				if strings.HasSuffix(strings.ToLower(file), ".txt") {
					displayName = file[:len(file)-4]
				}

				if i == m.selectedFileIndex && m.selectedFileIndex >= 0 {
					// Highlight selected file
					result.WriteString("> ")
					result.WriteString(displayName)
					result.WriteString(" <")
				} else {
					result.WriteString("  ")
					result.WriteString(displayName)
				}
				result.WriteString("\n")
			}
		}

		result.WriteString(strings.Repeat("─", renderWidth))
		result.WriteString("\n")
		result.WriteString("Filename: ")
		result.WriteString(m.filename)
		result.WriteString("█")
	} else {
		// Normal canvas display
		for i, line := range canvas {
			result.WriteString(line)
			if i < len(canvas)-1 {
				result.WriteString("\n")
			}
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
			statusLine = fmt.Sprintf("Mode: EDIT | Box %d | Text: %s | ←/→=move cursor, Enter=newline, Ctrl+S=save, Esc=cancel", m.selectedBox, cursorDisplay)
		} else if m.selectedText != -1 {
			statusLine = fmt.Sprintf("Mode: EDIT | Text %d | Text: %s | ←/→=move cursor, Enter=newline, Ctrl+S=save, Esc=cancel", m.selectedText, cursorDisplay)
		} else {
			statusLine = fmt.Sprintf("Mode: EDIT | Text: %s | ←/→=move cursor, Enter=newline, Ctrl+S=save, Esc=cancel", cursorDisplay)
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
		if m.selectedBox != -1 {
			statusLine = fmt.Sprintf("Mode: MOVE | Box %d | hjkl/arrows=move, Enter=finish, Esc=cancel", m.selectedBox)
		} else if m.selectedText != -1 {
			statusLine = fmt.Sprintf("Mode: MOVE | Text %d | hjkl/arrows=move, Enter=finish, Esc=cancel", m.selectedText)
		} else {
			statusLine = "Mode: MOVE | hjkl/arrows=move, Enter=finish, Esc=cancel"
		}
	case ModeFileInput:
		var opStr string
		switch m.fileOp {
		case FileOpSave:
			opStr = "Save"
		case FileOpOpen:
			opStr = "Open"
		case FileOpSavePNG:
			opStr = "Export PNG"
		}
		if m.errorMessage != "" {
			if m.fileOp == FileOpOpen {
				statusLine = fmt.Sprintf("Mode: FILE | ERROR: %s | %s filename: %s | ↑/↓=navigate, Enter=retry, Esc=cancel", m.errorMessage, opStr, m.filename)
			} else {
				statusLine = fmt.Sprintf("Mode: FILE | ERROR: %s | %s filename: %s | Enter=retry, Esc=cancel", m.errorMessage, opStr, m.filename)
			}
		} else {
			if m.fileOp == FileOpOpen {
				statusLine = fmt.Sprintf("Mode: FILE | %s filename: %s | ↑/↓=navigate list, Type=enter name, Enter=confirm, Esc=cancel", opStr, m.filename)
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
		case ConfirmQuit:
			message = "Quit Flerm? (y/n)"
		case ConfirmNewChart:
			message = "Create new chart? Unsaved changes will be lost. (y/n)"
		case ConfirmCloseBuffer:
			message = "Close current buffer? Unsaved changes will be lost. (y/n)"
		case ConfirmOverwriteFile:
			message = fmt.Sprintf("File %s already exists. Overwrite? (y/n)", m.filename)
		}
		statusLine = fmt.Sprintf("Mode: CONFIRM | %s", message)
	default:
		modeStr := m.modeString()
		if m.zPanMode {
			modeStr = "PAN"
		}
		status := fmt.Sprintf("Mode: %s | Cursor: (%d,%d)", modeStr, m.cursorX, m.cursorY)
		if m.connectionFrom != -1 {
			status += fmt.Sprintf(" | Connection from box %d (select target)", m.connectionFrom)
		} else if m.connectionFromLine != -1 {
			status += fmt.Sprintf(" | Connection from line (select target)")
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
	// Only add status line if not in startup mode
	if m.mode != ModeStartup {
		result.WriteString("\n")
		result.WriteString(statusLine)
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
	case ModeFileInput:
		return "FILE"
	case ModeConfirm:
		return "CONFIRM"
	default:
		return "UNKNOWN"
	}
}

func (m model) helpView() string {
	helpLines := []string{
		"Fl(ow)(T)erm Help",
		"=================",
		"",
		"",
		"Navigation:",
		"-----------",
		"  h/←/j/↓/k/↑/l/→  Move cursor around the screen",
		"  Shift+h/j/k/l    Move cursor 2x faster (hold Shift with direction keys)",
		"",
		"Box Operations:",
		"---------------",
		"  b                Create new box at cursor position",
		"  e                Edit text in box under cursor",
		"  r                Resize box under cursor",
		"  m                Move box under cursor",
		"  d                Delete box under cursor",
		"  c                Copy box under cursor",
		"  p                Paste copied box at cursor position",
		"",
		"Text Operations:",
		"----------------",
		"  t                Enter text at cursor position",
		"  e                Edit text under cursor",
		"  m                Move text under cursor",
		"  d                Delete text under cursor",
		"",
		"Resize Mode:",
		"------------",
		"  h/←/j/↓/k/↑/l/→  Resize box",
		"  Shift+h/j/k/l    Resize box 2x faster",
		"  Enter            Confirm edit and return to normal mode",
		"  Esc   	        Canceland return to normal mode",
		"",
		"Move Mode:",
		"----------",
		"  h/←/j/↓/k/↑/l/→  Move object around the screen",
		"  Shift+h/j/k/l    Move object 2x faster",
		"  Enter            Finish moving and return to normal mode",
		"  Escape           Cancel move and return to normal mode",
		"",
		"Note: Selected boxes (being resized/moved) are highlighted with # borders",
		"Another note: Resizing/moving boxes with connections is a little wonky,",
		"              when in doubt, delete the lines and re-connect the boxes ¯\\_(ツ)_/¯",
		"",
		"Connection Operations:",
		"---------------------",
		"  a                Start/finish connection creation",
		"                   - Press 'a' on a box or line to start",
		"                   - Press 'a' on empty space to add waypoint",
		"                   - Press 'a' on a box or line to finish",
		"                   - Connections can start/end at boxes or lines",
		"  A                Toggle arrow state on connection line under cursor",
		"                   - Cycles through: no arrows → to arrow → from arrow → both arrows",
		"                   Note: Sometimes the arrows flip around. Redrawing the line fixes it.",
		"",
		"File Operations:",
		"----------------",
		"  s                Save flowchart",
		"  S                Export as PNG image (experimental and janky)",
		"  o                Load a saved flowchart in current buffer",
		"  O                Load a saved flowchart in new buffer",
		"",
		"Buffer Operations:",
		"------------------",
		"  {                Switch to previous buffer",
		"  }                Switch to next buffer",
		"  n                Create new chart in current buffer",
		"  N                Create new chart in new buffer",
		"  x                Close current buffer",
		"",
		"General:",
		"  u                Undo last action",
		"  U                Redo last undone action",
		"  Esc           	Clear selection/cancel current operation",
		"  ?                Toggle this help screen",
		"  q/Ctrl+C         Quit Flerm",
		"",
		"========== Thanks for trying Flerm! ==========",
	}

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
