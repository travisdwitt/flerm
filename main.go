package main

import (
	"fmt"
	"log"
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

type model struct {
	width        int
	height       int
	cursorX      int
	cursorY      int
	canvas       *Canvas
	mode         Mode
	help         bool
	helpScroll   int
	selectedBox  int
	editText     string
	arrowFrom    int
	filename     string
	fileOp       FileOperation
	confirmAction ConfirmAction
	confirmBoxID int
	confirmTextID int
	undoStack    []Action
	redoStack    []Action
	// Track original state for move/resize operations
	originalMoveX  int
	originalMoveY  int
	originalWidth  int
	originalHeight int
	// Text input state
	textInputX    int
	textInputY    int
	textInputText string
	// Error message state
	errorMessage string
	// Track if we came from startup mode
	fromStartup bool
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
	ConfirmQuit
)

type Action struct {
	Type    ActionType
	Data    interface{}
	Inverse interface{} // Data needed to reverse the action
}

type ActionType int

const (
	ActionAddBox ActionType = iota
	ActionDeleteBox
	ActionEditBox
	ActionResizeBox
	ActionMoveBox
	ActionAddArrow
)

type AddBoxData struct {
	X, Y int
	Text string
	ID   int
}

type DeleteBoxData struct {
	Box   Box
	ID    int
	Arrows []Arrow // Arrows that were connected to this box
}

type EditBoxData struct {
	ID      int
	NewText string
	OldText string
}

type ResizeBoxData struct {
	ID           int
	DeltaWidth   int
	DeltaHeight  int
}

type MoveBoxData struct {
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

type AddArrowData struct {
	FromID int
	ToID   int
	Arrow  Arrow
}

func (m *model) recordAction(actionType ActionType, data, inverse interface{}) {
	action := Action{
		Type:    actionType,
		Data:    data,
		Inverse: inverse,
	}
	m.undoStack = append(m.undoStack, action)
	// Clear redo stack when new action is performed
	m.redoStack = m.redoStack[:0]
}

func (m *model) undo() {
	if len(m.undoStack) == 0 {
		return
	}

	// Pop the last action
	lastIndex := len(m.undoStack) - 1
	action := m.undoStack[lastIndex]
	m.undoStack = m.undoStack[:lastIndex]

	// Perform the inverse action
	switch action.Type {
	case ActionAddBox:
		data := action.Inverse.(DeleteBoxData)
		m.canvas.DeleteBox(data.ID)
	case ActionDeleteBox:
		data := action.Inverse.(AddBoxData)
		m.canvas.AddBoxWithID(data.X, data.Y, data.Text, data.ID)
		// Restore arrows that were connected to this box
		inverse := action.Data.(DeleteBoxData)
		for _, arrow := range inverse.Arrows {
			m.canvas.RestoreArrow(arrow)
		}
	case ActionEditBox:
		data := action.Inverse.(EditBoxData)
		m.canvas.SetBoxText(data.ID, data.NewText)
	case ActionResizeBox:
		data := action.Inverse.(OriginalBoxState)
		m.canvas.SetBoxSize(data.ID, data.Width, data.Height)
	case ActionMoveBox:
		data := action.Inverse.(OriginalBoxState)
		m.canvas.SetBoxPosition(data.ID, data.X, data.Y)
	case ActionAddArrow:
		data := action.Inverse.(AddArrowData)
		m.canvas.RemoveArrow(data.FromID, data.ToID)
	}

	// Move action to redo stack
	m.redoStack = append(m.redoStack, action)
}

func (m *model) redo() {
	if len(m.redoStack) == 0 {
		return
	}

	// Pop the last action from redo stack
	lastIndex := len(m.redoStack) - 1
	action := m.redoStack[lastIndex]
	m.redoStack = m.redoStack[:lastIndex]

	// Perform the action again
	switch action.Type {
	case ActionAddBox:
		data := action.Data.(AddBoxData)
		m.canvas.AddBoxWithID(data.X, data.Y, data.Text, data.ID)
	case ActionDeleteBox:
		data := action.Data.(DeleteBoxData)
		m.canvas.DeleteBox(data.ID)
	case ActionEditBox:
		data := action.Data.(EditBoxData)
		m.canvas.SetBoxText(data.ID, data.NewText)
	case ActionResizeBox:
		data := action.Data.(ResizeBoxData)
		m.canvas.ResizeBox(data.ID, data.DeltaWidth, data.DeltaHeight)
	case ActionMoveBox:
		data := action.Data.(MoveBoxData)
		m.canvas.MoveBox(data.ID, data.DeltaX, data.DeltaY)
	case ActionAddArrow:
		data := action.Data.(AddArrowData)
		m.canvas.RestoreArrow(data.Arrow)
	}

	// Move action back to undo stack
	m.undoStack = append(m.undoStack, action)
}

func initialModel() model {
	canvas := NewCanvas()
	// Add welcome box to top-left corner with 1 row/column padding
	canvas.AddBox(1, 1, "Welcome to Flerm!\n\nby Travis\n\n'n' New flowchart\n'o' Open existing chart\n'q' Quit")

	return model{
		canvas:      canvas,
		mode:        ModeStartup,
		selectedBox: -1,
		arrowFrom:   -1,
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

func (m model) Init() tea.Cmd {
	return nil
}

// forceRefresh is a command that forces a screen refresh
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
		// Handle help screen with scrolling (but not in startup mode)
		if m.help && m.mode != ModeStartup {
			switch msg.String() {
			case "escape", "q", "?":
				m.help = false
				m.helpScroll = 0
				return m, nil
			case "j", "down":
				// Calculate max scroll to prevent scrolling past end
				totalLines := 54 // Total lines in help text (approximate)
				maxScroll := totalLines - (m.height - 1)
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
				// Any other key closes help for compatibility
				m.help = false
				m.helpScroll = 0
				return m, nil
			}
		}

		switch m.mode {
		case ModeStartup:
			switch msg.String() {
			case "n":
				// New flowchart - clear canvas and switch to normal mode
				m.canvas = NewCanvas()
				m.mode = ModeNormal
				m.cursorX = 0
				m.cursorY = 0
				m.errorMessage = "" // Clear any previous error
				return m, nil
			case "o":
				// Open existing chart - switch to file input mode
				m.mode = ModeFileInput
				m.fileOp = FileOpOpen
				m.filename = "flowchart"
				m.errorMessage = "" // Clear any previous error
				m.fromStartup = true // Track that we came from startup
				return m, nil
			case "q", "ctrl+c":
				// Quit application
				return m, tea.Quit
			default:
				// Ignore other keys in startup mode
				return m, nil
			}

		case ModeNormal:
			switch msg.String() {
			case "ctrl+c", "q":
				m.mode = ModeConfirm
				m.confirmAction = ConfirmQuit
				return m, nil
			case "?":
				m.help = !m.help
				return m, nil
			case "h", "left":
				m.cursorX--
				m.ensureCursorInBounds()
				return m, nil
			case "H", "shift+left":
				m.cursorX -= 2
				m.ensureCursorInBounds()
				return m, nil
			case "l", "right":
				m.cursorX++
				m.ensureCursorInBounds()
				return m, nil
			case "L", "shift+right":
				m.cursorX += 2
				m.ensureCursorInBounds()
				return m, nil
			case "k", "up":
				m.cursorY--
				m.ensureCursorInBounds()
				return m, nil
			case "K", "shift+up":
				m.cursorY -= 2
				m.ensureCursorInBounds()
				return m, nil
			case "j", "down":
				m.cursorY++
				m.ensureCursorInBounds()
				return m, nil
			case "J", "shift+down":
				m.cursorY += 2
				m.ensureCursorInBounds()
				return m, nil
			case "b":
				boxID := len(m.canvas.boxes) // Get ID before adding
				m.canvas.AddBox(m.cursorX, m.cursorY, "Box")
				// Record the action for undo
				addData := AddBoxData{X: m.cursorX, Y: m.cursorY, Text: "Box", ID: boxID}
				deleteData := DeleteBoxData{ID: boxID, Arrows: nil} // No arrows initially
				m.recordAction(ActionAddBox, addData, deleteData)
				m.ensureCursorInBounds()
				return m, nil
			case "t":
				// Enter text input mode at cursor position
				m.mode = ModeTextInput
				m.textInputX = m.cursorX
				m.textInputY = m.cursorY
				m.textInputText = ""
				return m, nil
			case "r":
				boxID := m.canvas.GetBoxAt(m.cursorX, m.cursorY)
				if boxID != -1 {
					m.selectedBox = boxID
					// Store original size for undo
					if boxID < len(m.canvas.boxes) {
						m.originalWidth = m.canvas.boxes[boxID].Width
						m.originalHeight = m.canvas.boxes[boxID].Height
					}
					m.mode = ModeResize
				}
				return m, nil
			case "m":
				boxID := m.canvas.GetBoxAt(m.cursorX, m.cursorY)
				if boxID != -1 {
					m.selectedBox = boxID
					// Store original position for undo
					if boxID < len(m.canvas.boxes) {
						m.originalMoveX = m.canvas.boxes[boxID].X
						m.originalMoveY = m.canvas.boxes[boxID].Y
					}
					m.mode = ModeMove
				}
				return m, nil
			case "e":
				boxID := m.canvas.GetBoxAt(m.cursorX, m.cursorY)
				if boxID != -1 {
					m.selectedBox = boxID
					m.mode = ModeEditing
					m.editText = m.canvas.GetBoxText(boxID)
				}
				return m, nil
			case "a":
				boxID := m.canvas.GetBoxAt(m.cursorX, m.cursorY)
				if boxID != -1 {
					if m.arrowFrom == -1 {
						m.arrowFrom = boxID
					} else {
						// Get the arrow that will be created
						fromBox := m.canvas.boxes[m.arrowFrom]
						toBox := m.canvas.boxes[boxID]
						var fromX, fromY, toX, toY int

						// Calculate connection points (same logic as AddArrow)
						fromCenterX := fromBox.X + fromBox.Width/2
						fromCenterY := fromBox.Y + fromBox.Height/2
						toCenterX := toBox.X + toBox.Width/2
						toCenterY := toBox.Y + toBox.Height/2

						if abs(fromCenterX-toCenterX) > abs(fromCenterY-toCenterY) {
							if fromCenterX < toCenterX {
								fromX = fromBox.X + fromBox.Width
								fromY = fromCenterY
								toX = toBox.X - 1
								toY = toCenterY
							} else {
								fromX = fromBox.X - 1
								fromY = fromCenterY
								toX = toBox.X + toBox.Width
								toY = toCenterY
							}
						} else {
							if fromCenterY < toCenterY {
								fromX = fromCenterX
								fromY = fromBox.Y + fromBox.Height
								toX = toCenterX
								toY = toBox.Y - 1
							} else {
								fromX = fromCenterX
								fromY = fromBox.Y - 1
								toX = toCenterX
								toY = toBox.Y + toBox.Height
							}
						}

						arrow := Arrow{
							FromID: m.arrowFrom,
							ToID:   boxID,
							FromX:  fromX,
							FromY:  fromY,
							ToX:    toX,
							ToY:    toY,
						}

						m.canvas.AddArrow(m.arrowFrom, boxID)
						// Record the action for undo
						addArrowData := AddArrowData{FromID: m.arrowFrom, ToID: boxID, Arrow: arrow}
						inverseArrowData := AddArrowData{FromID: m.arrowFrom, ToID: boxID, Arrow: arrow}
						m.recordAction(ActionAddArrow, addArrowData, inverseArrowData)
						m.arrowFrom = -1
					}
				}
				return m, nil
			case "d":
				boxID := m.canvas.GetBoxAt(m.cursorX, m.cursorY)
				textID := m.canvas.GetTextAt(m.cursorX, m.cursorY)

				if boxID != -1 {
					m.mode = ModeConfirm
					m.confirmAction = ConfirmDeleteBox
					m.confirmBoxID = boxID
				} else if textID != -1 {
					m.mode = ModeConfirm
					m.confirmAction = ConfirmDeleteText
					m.confirmTextID = textID
				}
				return m, nil
			case "s":
				m.mode = ModeFileInput
				m.fileOp = FileOpSave
				m.filename = "flowchart"
				m.errorMessage = "" // Clear any previous error
				m.fromStartup = false // Clear startup flag
				return m, nil
			case "o":
				m.mode = ModeFileInput
				m.fileOp = FileOpOpen
				m.filename = "flowchart"
				m.errorMessage = "" // Clear any previous error
				m.fromStartup = false // Clear startup flag
				return m, nil
			case "x":
				m.mode = ModeFileInput
				m.fileOp = FileOpSavePNG
				m.filename = "flowchart"
				m.errorMessage = "" // Clear any previous error
				m.fromStartup = false // Clear startup flag
				return m, nil
			case "u":
				m.undo()
				return m, nil
			case "U":
				m.redo()
				return m, nil
			case "escape":
				m.arrowFrom = -1
				m.selectedBox = -1
				return m, nil
			}

		case ModeEditing:
			switch {
			case msg.Type == tea.KeyEscape:
				m.mode = ModeNormal
				m.editText = ""
				m.selectedBox = -1
				return m, nil
			case msg.Type == tea.KeyCtrlS:
				// Ctrl+S - save and exit
				if m.selectedBox != -1 {
					oldText := m.canvas.GetBoxText(m.selectedBox)
					m.canvas.SetBoxText(m.selectedBox, m.editText)
					// Record the edit for undo
					editData := EditBoxData{ID: m.selectedBox, NewText: m.editText, OldText: oldText}
					inverseData := EditBoxData{ID: m.selectedBox, NewText: oldText, OldText: m.editText}
					m.recordAction(ActionEditBox, editData, inverseData)
				}
				m.mode = ModeNormal
				m.editText = ""
				m.selectedBox = -1
				return m, nil
			case msg.Type == tea.KeyEnter:
				// Enter - add newline
				m.editText += "\n"
				return m, nil
			case msg.Type == tea.KeyBackspace:
				if len(m.editText) > 0 {
					m.editText = m.editText[:len(m.editText)-1]
				}
				return m, nil
			default:
				// Handle regular character input
				keyStr := msg.String()
				if len(keyStr) == 1 {
					m.editText += keyStr
				}
				return m, nil
			}

		case ModeTextInput:
			switch {
			case msg.Type == tea.KeyEscape:
				m.mode = ModeNormal
				m.textInputText = ""
				return m, nil
			case msg.Type == tea.KeyCtrlS:
				// Ctrl+S - save and create text
				if m.textInputText != "" {
					m.canvas.AddText(m.textInputX, m.textInputY, m.textInputText)
					// TODO: Add undo support for text creation
				}
				m.mode = ModeNormal
				m.textInputText = ""
				return m, nil
			case msg.Type == tea.KeyEnter:
				// Enter - add newline
				m.textInputText += "\n"
				return m, nil
			case msg.Type == tea.KeyBackspace:
				if len(m.textInputText) > 0 {
					m.textInputText = m.textInputText[:len(m.textInputText)-1]
				}
				return m, nil
			default:
				// Handle regular character input
				keyStr := msg.String()
				if len(keyStr) == 1 {
					m.textInputText += keyStr
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
					m.canvas.ResizeBox(m.selectedBox, -1, 0)
					m.ensureCursorInBounds()
				}
				return m, nil
			case "H", "shift+left":
				if m.selectedBox != -1 {
					m.canvas.ResizeBox(m.selectedBox, -2, 0)
					m.ensureCursorInBounds()
				}
				return m, nil
			case "l", "right":
				if m.selectedBox != -1 {
					m.canvas.ResizeBox(m.selectedBox, 1, 0)
					m.ensureCursorInBounds()
				}
				return m, nil
			case "L", "shift+right":
				if m.selectedBox != -1 {
					m.canvas.ResizeBox(m.selectedBox, 2, 0)
					m.ensureCursorInBounds()
				}
				return m, nil
			case "k", "up":
				if m.selectedBox != -1 {
					m.canvas.ResizeBox(m.selectedBox, 0, -1)
					m.ensureCursorInBounds()
				}
				return m, nil
			case "K", "shift+up":
				if m.selectedBox != -1 {
					m.canvas.ResizeBox(m.selectedBox, 0, -2)
					m.ensureCursorInBounds()
				}
				return m, nil
			case "j", "down":
				if m.selectedBox != -1 {
					m.canvas.ResizeBox(m.selectedBox, 0, 1)
					m.ensureCursorInBounds()
				}
				return m, nil
			case "J", "shift+down":
				if m.selectedBox != -1 {
					m.canvas.ResizeBox(m.selectedBox, 0, 2)
					m.ensureCursorInBounds()
				}
				return m, nil
			case "enter":
				// Record the resize action when finishing resize mode
				if m.selectedBox != -1 && m.selectedBox < len(m.canvas.boxes) {
					currentBox := m.canvas.boxes[m.selectedBox]
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
				return m, nil
			case "h", "left":
				if m.selectedBox != -1 {
					m.canvas.MoveBox(m.selectedBox, -1, 0)
					m.ensureCursorInBounds()
				}
				return m, nil
			case "H", "shift+left":
				if m.selectedBox != -1 {
					m.canvas.MoveBox(m.selectedBox, -2, 0)
					m.ensureCursorInBounds()
				}
				return m, nil
			case "l", "right":
				if m.selectedBox != -1 {
					m.canvas.MoveBox(m.selectedBox, 1, 0)
					m.ensureCursorInBounds()
				}
				return m, nil
			case "L", "shift+right":
				if m.selectedBox != -1 {
					m.canvas.MoveBox(m.selectedBox, 2, 0)
					m.ensureCursorInBounds()
				}
				return m, nil
			case "k", "up":
				if m.selectedBox != -1 {
					m.canvas.MoveBox(m.selectedBox, 0, -1)
					m.ensureCursorInBounds()
				}
				return m, nil
			case "K", "shift+up":
				if m.selectedBox != -1 {
					m.canvas.MoveBox(m.selectedBox, 0, -2)
					m.ensureCursorInBounds()
				}
				return m, nil
			case "j", "down":
				if m.selectedBox != -1 {
					m.canvas.MoveBox(m.selectedBox, 0, 1)
					m.ensureCursorInBounds()
				}
				return m, nil
			case "J", "shift+down":
				if m.selectedBox != -1 {
					m.canvas.MoveBox(m.selectedBox, 0, 2)
					m.ensureCursorInBounds()
				}
				return m, nil
			case "enter":
				// Record the move action when finishing move mode
				if m.selectedBox != -1 && m.selectedBox < len(m.canvas.boxes) {
					currentBox := m.canvas.boxes[m.selectedBox]
					// Calculate the total change from original position
					deltaX := currentBox.X - m.originalMoveX
					deltaY := currentBox.Y - m.originalMoveY

					// Only record if there was an actual change
					if deltaX != 0 || deltaY != 0 {
						moveData := MoveBoxData{ID: m.selectedBox, DeltaX: deltaX, DeltaY: deltaY}
						originalState := OriginalBoxState{ID: m.selectedBox, X: m.originalMoveX, Y: m.originalMoveY, Width: currentBox.Width, Height: currentBox.Height}
						m.recordAction(ActionMoveBox, moveData, originalState)
					}
				}
				m.mode = ModeNormal
				m.selectedBox = -1
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
			case msg.Type == tea.KeyEnter:
				// Execute the file operation with automatic extension
				filename := m.filename
				switch m.fileOp {
				case FileOpSave, FileOpOpen:
					if !strings.HasSuffix(strings.ToLower(filename), ".txt") {
						filename += ".txt"
					}
					if m.fileOp == FileOpSave {
						err := m.canvas.SaveToFile(filename)
						if err != nil {
							// Could show error in status, for now just ignore
						}
					} else {
						err := m.canvas.LoadFromFile(filename)
						if err != nil {
							m.errorMessage = fmt.Sprintf("Error opening file: %s", err.Error())
							// Stay in file input mode so user can try again or cancel
							return m, nil
						} else {
							m.errorMessage = "" // Clear any previous error
							m.fromStartup = false // Clear startup flag on successful load
						}
					}
				case FileOpSavePNG:
					if !strings.HasSuffix(strings.ToLower(filename), ".png") {
						filename += ".png"
					}
					err := m.canvas.ExportToPNG(filename, 800, 600)
					if err != nil {
						// Could show error in status, for now just ignore
					}
				}
				m.mode = ModeNormal
				m.filename = ""
				return m, nil
			case msg.Type == tea.KeyBackspace:
				if len(m.filename) > 0 {
					m.filename = m.filename[:len(m.filename)-1]
				}
				return m, nil
			default:
				// Handle regular character input for filename
				keyStr := msg.String()
				if len(keyStr) == 1 {
					m.filename += keyStr
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
					if m.confirmBoxID >= 0 && m.confirmBoxID < len(m.canvas.boxes) {
						box := m.canvas.boxes[m.confirmBoxID]
						// Find arrows connected to this box
						connectedArrows := make([]Arrow, 0)
						for _, arrow := range m.canvas.arrows {
							if arrow.FromID == m.confirmBoxID || arrow.ToID == m.confirmBoxID {
								connectedArrows = append(connectedArrows, arrow)
							}
						}
						deleteData := DeleteBoxData{Box: box, ID: m.confirmBoxID, Arrows: connectedArrows}
						addData := AddBoxData{X: box.X, Y: box.Y, Text: box.GetText(), ID: box.ID}
						m.recordAction(ActionDeleteBox, deleteData, addData)
					}
					m.canvas.DeleteBox(m.confirmBoxID)
					m.ensureCursorInBounds()
				case ConfirmDeleteText:
					// TODO: Add undo support for text deletion
					m.canvas.DeleteText(m.confirmTextID)
					m.ensureCursorInBounds()
				case ConfirmQuit:
					return m, tea.Quit
				}
				m.mode = ModeNormal
				return m, nil
			case "n", "N", "escape":
				// Cancel the action
				m.mode = ModeNormal
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
	renderHeight := m.height - 1 // Leave room for status line
	if renderHeight < 1 {
		renderHeight = 1
	}
	renderWidth := m.width
	if renderWidth < 1 {
		renderWidth = 1
	}

	canvas := m.canvas.Render(renderWidth, renderHeight, selectedBox)

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

	// Add cursor (except in startup mode)
	if m.mode != ModeStartup && cursorY < len(canvas) && cursorX < len(canvas[cursorY]) {
		line := []rune(canvas[cursorY])
		if cursorX < len(line) {
			line[cursorX] = '█'
			canvas[cursorY] = string(line)
		}
	}

	// Build result with proper newlines
	var result strings.Builder
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
		// Show text with visual newline indicators
		displayText := strings.ReplaceAll(m.editText, "\n", "↵")
		statusLine = fmt.Sprintf("Mode: EDIT | Text: %s | Enter=newline, Ctrl+S=save, Esc=cancel", displayText)
	case ModeTextInput:
		// Show text with visual newline indicators
		displayText := strings.ReplaceAll(m.textInputText, "\n", "↵")
		statusLine = fmt.Sprintf("Mode: TEXT | Text: %s | Enter=newline, Ctrl+S=save, Esc=cancel", displayText)
	case ModeResize:
		statusLine = fmt.Sprintf("Mode: RESIZE | Box %d | hjkl/arrows=resize, Enter=finish, Esc=cancel", m.selectedBox)
	case ModeMove:
		statusLine = fmt.Sprintf("Mode: MOVE | Box %d | hjkl/arrows=move, Enter=finish, Esc=cancel", m.selectedBox)
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
			statusLine = fmt.Sprintf("Mode: FILE | ERROR: %s | %s filename: %s | Enter=retry, Esc=cancel", m.errorMessage, opStr, m.filename)
		} else {
			statusLine = fmt.Sprintf("Mode: FILE | %s filename: %s | Enter=confirm, Esc=cancel", opStr, m.filename)
		}
	case ModeConfirm:
		var message string
		switch m.confirmAction {
		case ConfirmDeleteBox:
			message = "Delete this box? (y/n)"
		case ConfirmDeleteText:
			message = "Delete this text? (y/n)"
		case ConfirmQuit:
			message = "Quit Flerm? (y/n)"
		}
		statusLine = fmt.Sprintf("Mode: CONFIRM | %s", message)
	default:
		status := fmt.Sprintf("Mode: %s | Cursor: (%d,%d)", m.modeString(), m.cursorX, m.cursorY)
		if m.arrowFrom != -1 {
			status += fmt.Sprintf(" | Arrow from Box %d (select target)", m.arrowFrom)
		}
		if m.selectedBox != -1 {
			status += fmt.Sprintf(" | Selected: Box %d", m.selectedBox)
		}
		if m.errorMessage != "" {
			status += fmt.Sprintf(" | ERROR: %s", m.errorMessage)
		} else {
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
		"",
		"Navigation:",
		"  h/←/j/↓/k/↑/l/→  Move cursor around the screen",
		"  Shift+h/j/k/l    Move cursor 2x faster (hold Shift with direction keys)",
		"",
		"Box Operations:",
		"  b                Create new box at cursor position",
		"  e                Edit text in box under cursor",
		"  r                Resize box under cursor (enters resize mode)",
		"  m                Move box under cursor (enters move mode)",
		"  d                Delete box under cursor (shows confirmation)",
		"",
		"Text Operations:",
		"  t                Enter text mode at cursor position (plain text, no borders)",
		"  d                Delete text under cursor (shows confirmation)",
		"",
		"Resize Mode (after pressing 'r' on a box):",
		"  h/←/j/↓/k/↑/l/→  Resize box (shrink/expand width/height)",
		"  Shift+h/j/k/l    Resize box 2x faster",
		"  Enter            Finish resizing and return to normal mode",
		"  Escape           Cancel resize and return to normal mode",
		"",
		"Move Mode (after pressing 'm' on a box):",
		"  h/←/j/↓/k/↑/l/→  Move box around the screen",
		"  Shift+h/j/k/l    Move box 2x faster",
		"  Enter            Finish moving and return to normal mode",
		"  Escape           Cancel move and return to normal mode",
		"",
		"Note: Active boxes (being resized/moved) are highlighted with # borders",
		"",
		"Arrow Operations:",
		"  a                Start/finish arrow creation between boxes",
		"                   (press 'a' on source box, then 'a' on target box)",
		"",
		"File Operations:",
		"  s                Save flowchart (prompts for filename, adds .txt if missing)",
		"  o                Open flowchart (prompts for filename, adds .txt if missing)",
		"  x                Export as PNG image (prompts for filename, adds .png if missing)",
		"",
		"File Input Mode (after pressing s/o/x):",
		"  Type             Enter filename (extensions added automatically)",
		"  Backspace        Delete last character",
		"  Enter            Confirm and execute operation",
		"  Escape           Cancel file operation",
		"",
		"Editing Mode:",
		"  Type             Add text to box",
		"  Enter            Add new line to box text",
		"  Backspace        Delete last character",
		"  Ctrl+S           Save changes and return to normal mode",
		"  Escape           Cancel changes and return to normal mode",
		"",
		"Text Mode (after pressing 't'):",
		"  Type             Add plain text at cursor position",
		"  Enter            Add new line to text",
		"  Backspace        Delete last character",
		"  Ctrl+S           Save text and return to normal mode",
		"  Escape           Cancel and return to normal mode",
		"",
		"Note: Boxes automatically resize to fit text content",
		"",
		"General:",
		"  u                Undo last action",
		"  U                Redo last undone action",
		"  Escape           Clear selection/cancel current operation",
		"  ?                Toggle this help screen",
		"  q/Ctrl+C         Quit application (shows confirmation)",
		"",
		"Confirmation Dialogues:",
		"  Y/y              Confirm action (delete box or quit)",
		"  N/n/Escape       Cancel action and return to normal mode",
		"",
		"Help Navigation:",
		"  j/↓/k/↑          Scroll help text up and down",
		"  Escape/?/q       Close help screen",
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