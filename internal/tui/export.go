package tui

import (
	"fmt"
	"os"
	"strings"
)

func (m *model) exportVisualTXT(filename string) error {
	canvas := m.getCanvas()
	minX, minY, maxX, maxY := canvas.GetFullBounds()
	if minX > maxX || minY > maxY {
		return fmt.Errorf("nothing to export")
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	const padding = 1
	minX -= padding
	minY -= padding
	width := maxX - minX + padding + 1
	height := maxY - minY + padding + 1

	r := canvas.RenderRaw(width, height, -1, CoordUnset, CoordUnset, nil, CoordUnset, CoordUnset, minX, minY, -1, -1, false, -1, -1, 0, "", CoordUnset, CoordUnset, CoordUnset, CoordUnset, CoordUnset, CoordUnset, false, -1, -1)
	for _, row := range r.Canvas {
		fmt.Fprintln(file, strings.TrimRight(string(row), " "))
	}
	return nil
}
