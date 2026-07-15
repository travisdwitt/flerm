package tui

import (
	"os"
	"sort"
	"strings"

	cv "flerm/internal/canvas"
	"flerm/internal/config"

	tea "github.com/charmbracelet/bubbletea"
)

func Run() error {
	p := tea.NewProgram(
		initialModel(),
		tea.WithAltScreen(),

		tea.WithMouseAllMotion(),
	)
	_, err := p.Run()
	return err
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func initialModel() model {
	cfg := config.Load()
	initialMode := ModeStartup
	if !cfg.StartMenu {
		initialMode = ModeNormal
	}
	buffer := Buffer{
		canvas:    cv.NewCanvas(),
		undoStack: []Action{},
		redoStack: []Action{},
		filename:  "",
		panX:      0,
		panY:      0,
	}

	return model{
		buffers:                []Buffer{buffer},
		currentBufferIndex:     0,
		mode:                   initialMode,
		selectedBox:            -1,
		selectedText:           -1,
		connectionFrom:         -1,
		connectionFromLine:     -1,
		config:                 cfg,
		highlightMode:          false,
		selectedColor:          0,
		selectionStartX:        -1,
		selectionStartY:        -1,
		selectedBoxes:          []int{},
		selectedTexts:          []int{},
		selectedConnections:    []int{},
		originalBoxPositions:   make(map[int]point),
		originalTextPositions:  make(map[int]point),
		originalConnections:    make(map[int]Connection),
		originalHighlights:     make(map[point]int),
		originalBoxConnections: make(map[int][]Connection),
		selBox:                 -1,
		selText:                -1,
		selConn:                -1,
		menuTargetBox:          -1,
		menuTargetText:         -1,
		menuTargetConn:         -1,
	}
}

func (m *model) ensureCursorInBounds() {
	if m.cursorX < 0 {
		m.cursorX = 0
	}
	if m.cursorY < 0 {
		m.cursorY = 0
	}
	if m.width > 0 && m.cursorX >= m.width {
		m.cursorX = m.width - 1
	}
	maxY := m.height - 2
	if maxY < 0 {
		maxY = 0
	}
	if m.cursorY > maxY {
		m.cursorY = maxY
	}
}

func (m *model) scanTxtFiles() {
	m.fileList = []string{}
	dir := ""
	if m.config != nil && m.config.SaveDirectory != "" {
		dir = m.config.SaveDirectory
	} else {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			m.selectedFileIndex = -1
			return
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		m.selectedFileIndex = -1
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".sav") {
			m.fileList = append(m.fileList, entry.Name())
		}
	}
	sort.Strings(m.fileList)
	if len(m.fileList) > 0 {
		m.selectedFileIndex = 0
		firstFile := m.fileList[0]
		if strings.HasSuffix(strings.ToLower(firstFile), ".sav") {
			m.filename = firstFile[:len(firstFile)-4]
		} else {
			m.filename = firstFile
		}
	} else {
		m.selectedFileIndex = -1
	}
}
