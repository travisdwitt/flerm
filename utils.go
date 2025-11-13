package main

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

func (m *model) getPanOffset() (int, int) {
	buf := m.getCurrentBuffer()
	if buf == nil {
		return 0, 0
	}
	return buf.panX, buf.panY
}

func (m *model) worldCoords() (int, int) {
	panX, panY := m.getPanOffset()
	return m.cursorX + panX, m.cursorY + panY
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

