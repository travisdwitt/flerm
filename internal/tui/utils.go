package tui

import (
	"os/exec"
	"runtime"

	"github.com/atotto/clipboard"
)

func (m *model) getCurrentBuffer() *Buffer {
	return &m.buffers[m.currentBufferIndex]
}

func (m *model) getCanvas() *Canvas {
	return m.getCurrentBuffer().canvas
}

func (m *model) getPanOffset() (int, int) {
	buf := m.getCurrentBuffer()
	return buf.panX, buf.panY
}

func (m *model) worldCursor() (int, int) {
	panX, panY := m.getPanOffset()
	return m.cursorX + panX, m.cursorY + panY
}

func (m *model) unsavedChanges() bool {
	for _, buf := range m.buffers {
		if len(buf.undoStack) != buf.savedAt {
			return true
		}
	}
	return false
}

func (m *model) addNewBuffer(buf Buffer) {
	m.buffers = append(m.buffers, buf)
	m.currentBufferIndex = len(m.buffers) - 1
}

func (m *model) recordAction(actionType ActionType, data, inverse any) {
	buf := m.getCurrentBuffer()
	buf.undoStack = append(buf.undoStack, Action{Type: actionType, Data: data, Inverse: inverse})
	buf.redoStack = buf.redoStack[:0]
}

func (m *model) confirm(action ConfirmAction, id int) {
	m.mode = ModeConfirm
	m.confirmAction = action
	m.confirmID = id
}

func (m *model) beginFileOp(op FileOperation, name string) {
	m.mode = ModeFileInput
	m.fileOp = op
	m.filename = name
	m.errorMessage, m.successMessage = "", ""
	m.fromStartup = false
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
