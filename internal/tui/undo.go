package tui

import (
	"slices"

	cv "flerm/internal/canvas"
)

const maxUndo = 100

func newBuffer(canvas *cv.Canvas) Buffer {
	return Buffer{canvas: canvas, last: canvas.Clone()}
}

func (m *model) applyHighlight(x, y, color int) {
	if color < 0 {
		m.getCanvas().ClearHighlight(x, y)
		return
	}
	m.getCanvas().SetHighlight(x, y, color)
}

func (m *model) recordAction() {
	buf := m.getCurrentBuffer()
	buf.undoStack = append(buf.undoStack, buf.last)
	if len(buf.undoStack) > maxUndo {
		buf.undoStack = slices.Delete(buf.undoStack, 0, 1)
		buf.savedAt--
	}
	buf.last = buf.canvas.Clone()
	buf.redoStack = nil
}

func (m *model) recordIf(changed bool) {
	if changed {
		m.recordAction()
	}
}

func (m *model) revert() {
	buf := m.getCurrentBuffer()
	buf.canvas = buf.last.Clone()
}

func (m *model) undo() {
	buf := m.getCurrentBuffer()
	if n := len(buf.undoStack); n > 0 {
		buf.redoStack = append(buf.redoStack, buf.last)
		buf.last = buf.undoStack[n-1]
		buf.undoStack = buf.undoStack[:n-1]
		buf.canvas = buf.last.Clone()
	}
}

func (m *model) redo() {
	buf := m.getCurrentBuffer()
	if n := len(buf.redoStack); n > 0 {
		buf.undoStack = append(buf.undoStack, buf.last)
		buf.last = buf.redoStack[n-1]
		buf.redoStack = buf.redoStack[:n-1]
		buf.canvas = buf.last.Clone()
	}
}
