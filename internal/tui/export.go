package tui

import (
	"fmt"
	"os"
	"strings"
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

	minX, minY, maxX, maxY := canvas.GetFullBounds()
	if minX > maxX || minY > maxY {
		return fmt.Errorf("nothing to export")
	}

	padding := 1
	minX -= padding
	minY -= padding
	if minX < 0 {
		minX = 0
	}
	if minY < 0 {
		minY = 0
	}

	width := maxX - minX + padding + 1
	height := maxY - minY + padding + 1

	renderResult := canvas.RenderRaw(width, height, -1, -1, -1, nil, -1, -1, minX, minY, -1, -1, false, -1, -1, 0, "", -1, -1, -1, -1, -1, -1, false, -1, -1)

	for _, row := range renderResult.Canvas {
		line := strings.TrimRight(string(row), " ")
		fmt.Fprintln(file, line)
	}

	return nil
}
