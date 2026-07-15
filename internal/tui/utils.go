package tui

import (
	"os/exec"
	"runtime"

	"github.com/atotto/clipboard"
)

func (m *model) getCurrentBuffer() *Buffer {
	if len(m.buffers) == 0 {
		return nil
	}
	return &m.buffers[m.currentBufferIndex]
}

func (m *model) getCanvas() *Canvas {
	if buf := m.getCurrentBuffer(); buf != nil {
		return buf.canvas
	}
	return nil
}

func (m *model) getPanOffset() (int, int) {
	if buf := m.getCurrentBuffer(); buf != nil {
		return buf.panX, buf.panY
	}
	return 0, 0
}

func (m *model) addNewBuffer(canvas *Canvas, filename string) {
	m.addNewBufferWithPan(canvas, filename, 0, 0)
}

func (m *model) addNewBufferWithPan(canvas *Canvas, filename string, panX, panY int) {
	buffer := Buffer{
		canvas:    canvas,
		undoStack: []Action{},
		redoStack: []Action{},
		filename:  filename,
		panX:      panX,
		panY:      panY,
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

func readClipboardText() (string, error) {
	if runtime.GOOS == "darwin" {
		if output, err := exec.Command("pbpaste", "-Prefer", "txt").Output(); err == nil {
			return string(output), nil
		}
		if output, err := exec.Command("pbpaste").Output(); err == nil {
			return string(output), nil
		}
	}
	return clipboard.ReadAll()
}
