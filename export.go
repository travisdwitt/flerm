package main

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

	// Calculate full canvas bounds to export ALL content, not just visible area
	minX, minY, maxX, maxY := canvas.GetFullBounds()
	if minX > maxX || minY > maxY {
		return fmt.Errorf("nothing to export")
	}

	// Add small padding around content
	padding := 1
	minX -= padding
	minY -= padding
	if minX < 0 {
		minX = 0
	}
	if minY < 0 {
		minY = 0
	}

	// Calculate dimensions needed to fit all content
	width := maxX - minX + padding + 1
	height := maxY - minY + padding + 1

	// Use RenderRaw with pan offset set to minX/minY to capture from the top-left of content
	// This renders the entire canvas content area
	renderResult := canvas.RenderRaw(width, height, -1, -1, -1, nil, -1, -1, minX, minY, -1, -1, false, -1, -1, 0, "", -1, -1, -1, -1, -1, -1, false, -1, -1, false, nil)

	// Convert rune canvas to plain text strings (no colors)
	// Trim trailing whitespace from each line for cleaner output
	for _, row := range renderResult.Canvas {
		line := strings.TrimRight(string(row), " ")
		fmt.Fprintln(file, line)
	}

	return nil
}

