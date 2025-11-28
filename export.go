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

	// Get current buffer for pan offset
	buf := m.getCurrentBuffer()
	panX, panY := 0, 0
	if buf != nil {
		panX, panY = buf.panX, buf.panY
	}

	// Calculate render dimensions - use current viewport size
	// Account for buffer bar if shown
	showBufferBar := m.mode != ModeStartup && len(m.buffers) > 1
	width := m.width
	if width < 1 {
		width = 80 // Default minimum width
	}
	
	var height int
	if showBufferBar {
		height = m.height - 2 // Leave room for buffer bar and status line
	} else {
		height = m.height - 1 // Leave room for status line only
	}
	if height < 1 {
		height = 24 // Default minimum height
	}

	// Render the canvas exactly as it appears (no selected box, no preview connection, no cursor, no editing cursor)
	rendered := canvas.Render(width, height, -1, -1, -1, nil, -1, -1, panX, panY, -1, -1, false, -1, -1, 0, "", -1, -1)

	// Write each line to the file
	for _, line := range rendered {
		fmt.Fprintln(file, line)
	}

	return nil
}

