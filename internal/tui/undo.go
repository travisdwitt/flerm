package tui

func (m *model) applyHighlight(x, y, color int) {
	if color < 0 {
		m.getCanvas().ClearHighlight(x, y)
		return
	}
	m.getCanvas().SetHighlight(x, y, color)
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
		moveData := action.Data.(MoveBoxData)
		m.getCanvas().SetBoxPositionOnly(data.ID, data.X, data.Y)
		if len(data.Connections) > 0 {
			m.getCanvas().RestoreConnections(data.Connections)
		}
		m.moveRecordedHighlights(data.Highlights, moveData.DeltaX, moveData.DeltaY, 0, 0)
	case ActionMoveText:
		data := action.Inverse.(OriginalTextState)
		moveData := action.Data.(MoveTextData)
		m.getCanvas().SetTextPosition(data.ID, data.X, data.Y)
		m.moveRecordedHighlights(data.Highlights, moveData.DeltaX, moveData.DeltaY, 0, 0)
	case ActionAddConnection:
		data := action.Inverse.(AddConnectionData)
		m.getCanvas().RemoveSpecificConnection(data.Connection)
	case ActionDeleteConnection:
		data := action.Inverse.(AddConnectionData)
		m.getCanvas().RestoreConnection(data.Connection)
	case ActionCycleArrow:
		cycleData := action.Inverse.(CycleArrowData)
		if cycleData.ConnIdx >= 0 && cycleData.ConnIdx < len(m.getCanvas().Connections()) {
			m.getCanvas().Connections()[cycleData.ConnIdx] = cycleData.OldConn
		}
	case ActionHighlight:
		data := action.Inverse.(HighlightData)
		for _, cell := range data.Cells {
			m.applyHighlight(cell.X, cell.Y, cell.Color)
		}
	case ActionChangeBorderStyle:
		data := action.Inverse.(BorderStyleData)
		m.getCanvas().SetBorderStyle(data.BoxID, data.OldStyle)
	case ActionEditTitle:
		data := action.Inverse.(EditTitleData)
		if data.BoxID >= 0 && data.BoxID < len(m.getCanvas().Boxes()) {
			m.getCanvas().Boxes()[data.BoxID].Title = data.NewTitle
			m.getCanvas().Boxes()[data.BoxID].UpdateSize()
		}
	case ActionSetColor:
		data := action.Inverse.(ColorData)
		m.applyObjectColor(data.Kind, data.ID, data.OldColor)
	case ActionGroupMove:
		data := action.Inverse.(GroupMoveData)
		m.applyGroupMoveState(data.After, data.Before)
	case ActionDuplicate:
		data := action.Inverse.(DuplicateData)
		if data.IsText {
			m.getCanvas().DeleteText(data.NewID)
		} else {
			m.getCanvas().DeleteBox(data.NewID)
		}
	}

	buf.redoStack = append(buf.redoStack, action)
}

func (m *model) applyObjectColor(kind, id, color int) {
	switch kind {
	case ColorKindBox:
		m.getCanvas().SetBoxColor(id, color)
	case ColorKindLine:
		m.getCanvas().SetLineColor(id, color)
	case ColorKindText:
		m.getCanvas().SetTextColor(id, color)
	}
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
		m.moveRecordedHighlights(action.Inverse.(OriginalBoxState).Highlights, 0, 0, data.DeltaX, data.DeltaY)
	case ActionMoveText:
		data := action.Data.(MoveTextData)
		m.getCanvas().MoveText(data.ID, data.DeltaX, data.DeltaY)
		m.moveRecordedHighlights(action.Inverse.(OriginalTextState).Highlights, 0, 0, data.DeltaX, data.DeltaY)
	case ActionAddConnection:
		data := action.Data.(AddConnectionData)
		m.getCanvas().RestoreConnection(data.Connection)
	case ActionDeleteConnection:
		data := action.Data.(AddConnectionData)
		m.getCanvas().RemoveSpecificConnection(data.Connection)
	case ActionCycleArrow:
		cycleData := action.Data.(CycleArrowData)
		if cycleData.ConnIdx >= 0 && cycleData.ConnIdx < len(m.getCanvas().Connections()) {
			m.getCanvas().Connections()[cycleData.ConnIdx] = cycleData.NewConn
		}
	case ActionHighlight:
		data := action.Data.(HighlightData)
		for _, cell := range data.Cells {
			m.applyHighlight(cell.X, cell.Y, cell.Color)
		}
	case ActionChangeBorderStyle:
		data := action.Data.(BorderStyleData)
		m.getCanvas().SetBorderStyle(data.BoxID, data.NewStyle)
	case ActionEditTitle:
		data := action.Data.(EditTitleData)
		if data.BoxID >= 0 && data.BoxID < len(m.getCanvas().Boxes()) {
			m.getCanvas().Boxes()[data.BoxID].Title = data.NewTitle
			m.getCanvas().Boxes()[data.BoxID].UpdateSize()
		}
	case ActionSetColor:
		data := action.Data.(ColorData)
		m.applyObjectColor(data.Kind, data.ID, data.NewColor)
	case ActionGroupMove:
		data := action.Data.(GroupMoveData)
		m.applyGroupMoveState(data.Before, data.After)
	case ActionDuplicate:
		data := action.Data.(DuplicateData)
		if data.IsText {
			m.getCanvas().DuplicateText(data.SrcID, data.DX, data.DY)
		} else {
			m.getCanvas().DuplicateBox(data.SrcID, data.DX, data.DY)
		}
	}

	buf.undoStack = append(buf.undoStack, action)
}
