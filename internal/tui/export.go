package tui

import (
	"io"
	"os"
)

func writeTo(filename string, write func(io.Writer) error) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	err = write(file)
	if cerr := file.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(filename)
	}
	return err
}

func (m *model) exportVisualTXT(filename string) error {
	return writeTo(filename, m.getCanvas().WriteText)
}

func (m *model) exportMermaid(filename string) error {
	return writeTo(filename, m.getCanvas().WriteMermaid)
}
