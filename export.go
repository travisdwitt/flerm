package main

import (
	"fmt"
	"os"
)

func (m *model) exportVisualTXT(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	canvas := m.getCanvas()
	if canvas == nil {
		return fmt.Errorf("no canvas available")
	}
	buf := m.getCurrentBuffer()
	panX, panY := 0, 0
	if buf != nil {
		panX, panY = buf.panX, buf.panY
	}
	showBufferBar := m.mode != ModeStartup && len(m.buffers) > 1
	width := m.width
	if width < 1 {
		width = 80
	}
	height := m.height - 1
	if showBufferBar {
		height = m.height - 2
	}
	if height < 1 {
		height = 24
	}
	rendered := canvas.Render(width, height, -1, -1, -1, nil, -1, -1, panX, panY, -1, -1, false, -1, -1, 0, "", -1, -1, -1, -1, -1, -1, false)
	for _, line := range rendered {
		fmt.Fprintln(file, line)
	}

	return nil
}

