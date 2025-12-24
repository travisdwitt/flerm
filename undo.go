package main

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
		for _, highlight := range inverse.Highlights {
			m.getCanvas().SetHighlight(highlight.X, highlight.Y, highlight.Color)
		}
	case ActionEditBox:
		data := action.Inverse.(EditBoxData)
		m.getCanvas().SetBoxText(data.ID, data.NewText)
	case ActionEditText:
		data := action.Inverse.(EditTextData)
		m.getCanvas().SetTextText(data.ID, data.NewText)
	case ActionDeleteText:
		data := action.Inverse.(AddTextData)
		m.getCanvas().AddTextWithID(data.X, data.Y, data.Text, data.ID)
		inverse := action.Data.(DeleteTextData)
		for _, highlight := range inverse.Highlights {
			m.getCanvas().SetHighlight(highlight.X, highlight.Y, highlight.Color)
		}
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
		cycleData := action.Inverse.(CycleArrowData)
		if cycleData.ConnIdx >= 0 && cycleData.ConnIdx < len(m.getCanvas().connections) {
			m.getCanvas().connections[cycleData.ConnIdx] = cycleData.OldConn
		}
	case ActionHighlight:
		data := action.Inverse.(HighlightData)
		for _, cell := range data.Cells {
			if cell.Color >= 0 {
				m.getCanvas().SetHighlight(cell.X, cell.Y, cell.Color)
			} else {
				m.getCanvas().ClearHighlight(cell.X, cell.Y)
			}
		}
	case ActionChangeBorderStyle:
		data := action.Inverse.(BorderStyleData)
		m.getCanvas().SetBorderStyle(data.BoxID, data.OldStyle)
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
	case ActionDeleteText:
		data := action.Data.(DeleteTextData)
		m.getCanvas().DeleteText(data.ID)
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
		cycleData := action.Data.(CycleArrowData)
		if cycleData.ConnIdx >= 0 && cycleData.ConnIdx < len(m.getCanvas().connections) {
			m.getCanvas().connections[cycleData.ConnIdx] = cycleData.NewConn
		}
	case ActionHighlight:
		data := action.Data.(HighlightData)
		for _, cell := range data.Cells {
			m.getCanvas().SetHighlight(cell.X, cell.Y, cell.Color)
		}
	case ActionChangeBorderStyle:
		data := action.Data.(BorderStyleData)
		m.getCanvas().SetBorderStyle(data.BoxID, data.NewStyle)
	}

	buf.undoStack = append(buf.undoStack, action)
}

