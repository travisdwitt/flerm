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
	if n := len(buf.undoStack); n > 0 {
		action := buf.undoStack[n-1]
		buf.undoStack = buf.undoStack[:n-1]
		m.applyAction(action, false)
		buf.redoStack = append(buf.redoStack, action)
	}
}

func (m *model) redo() {
	buf := m.getCurrentBuffer()
	if n := len(buf.redoStack); n > 0 {
		action := buf.redoStack[n-1]
		buf.redoStack = buf.redoStack[:n-1]
		m.applyAction(action, true)
		buf.undoStack = append(buf.undoStack, action)
	}
}

func (m *model) applyAction(a Action, forward bool) {
	canvas := m.getCanvas()
	data, inv := a.Data, a.Inverse
	if !forward {
		data, inv = inv, data
	}
	switch a.Type {
	case ActionAddBox, ActionDeleteBox:
		if (a.Type == ActionAddBox) != forward {
			canvas.DeleteBox(data.(DeleteBoxData).ID)
			break
		}
		add, del := data.(AddData), inv.(DeleteBoxData)
		canvas.AddBoxWithID(add.X, add.Y, add.Text, add.ID)
		for _, conn := range del.Connections {
			canvas.RestoreConnection(conn)
		}
		for _, h := range del.Highlights {
			canvas.SetHighlight(h.X, h.Y, h.Color)
		}
	case ActionDeleteText:
		if forward {
			canvas.DeleteText(data.(DeleteTextData).ID)
			break
		}
		add, del := data.(AddData), inv.(DeleteTextData)
		canvas.AddTextWithID(add.X, add.Y, add.Text, add.ID)
		for _, h := range del.Highlights {
			canvas.SetHighlight(h.X, h.Y, h.Color)
		}
	case ActionEditBox:
		d := data.(EditData)
		canvas.SetBoxText(d.ID, d.NewText)
	case ActionEditText:
		d := data.(EditData)
		canvas.SetTextText(d.ID, d.NewText)
	case ActionEditTitle:
		d := data.(EditData)
		canvas.SetBoxTitle(d.ID, d.NewText)
	case ActionResizeBox:
		if forward {
			d := data.(ResizeBoxData)
			canvas.ResizeBox(d.ID, d.DeltaWidth, d.DeltaHeight)
		} else {
			d := data.(OriginalBoxState)
			canvas.SetBoxSize(d.ID, d.Width, d.Height)
		}
	case ActionMoveBox, ActionMoveText:
		isBox := a.Type == ActionMoveBox
		if forward {
			d := data.(MoveData)
			if isBox {
				canvas.MoveBox(d.ID, d.DeltaX, d.DeltaY)
			} else {
				canvas.MoveText(d.ID, d.DeltaX, d.DeltaY)
			}
			m.moveRecordedHighlights(inv.(OriginalBoxState).Highlights, 0, 0, d.DeltaX, d.DeltaY)
			break
		}
		d, mv := data.(OriginalBoxState), inv.(MoveData)
		if isBox {
			canvas.SetBoxPositionOnly(d.ID, d.X, d.Y)
			canvas.RestoreConnections(d.Connections)
		} else {
			canvas.SetTextPosition(d.ID, d.X, d.Y)
		}
		m.moveRecordedHighlights(d.Highlights, mv.DeltaX, mv.DeltaY, 0, 0)
	case ActionAddConnection, ActionDeleteConnection:
		conn := data.(Connection)
		if (a.Type == ActionAddConnection) == forward {
			canvas.RestoreConnection(conn)
		} else {
			canvas.RemoveSpecificConnection(conn)
		}
	case ActionCycleArrow:
		d := data.(CycleArrowData)
		conn := d.NewConn
		if !forward {
			conn = d.OldConn
		}
		if d.ConnIdx >= 0 && d.ConnIdx < len(canvas.Connections()) {
			canvas.Connections()[d.ConnIdx] = conn
		}
	case ActionHighlight:
		for _, c := range data.(HighlightData).Cells {
			m.applyHighlight(c.X, c.Y, c.Color)
		}
	case ActionChangeBorderStyle:
		d := data.(BorderStyleData)
		style := d.NewStyle
		if !forward {
			style = d.OldStyle
		}
		canvas.SetBorderStyle(d.BoxID, style)
	case ActionSetColor:
		d := data.(ColorData)
		color := d.NewColor
		if !forward {
			color = d.OldColor
		}
		m.applyObjectColor(d.Kind, d.ID, color)
	case ActionGroupMove:
		d := data.(GroupMoveData)
		if forward {
			m.applyGroupMoveState(d.Before, d.After)
		} else {
			m.applyGroupMoveState(d.After, d.Before)
		}
	case ActionDuplicate:
		d := data.(DuplicateData)
		switch {
		case forward && d.IsText:
			canvas.DuplicateText(d.SrcID, d.DX, d.DY)
		case forward:
			canvas.DuplicateBox(d.SrcID, d.DX, d.DY)
		case d.IsText:
			canvas.DeleteText(d.NewID)
		default:
			canvas.DeleteBox(d.NewID)
		}
	}
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
