package tui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	cv "flerm/internal/canvas"
	"flerm/internal/config"

	tea "github.com/charmbracelet/bubbletea"
)

func Run(noResume bool) error {
	m := initialModel()
	if noResume {
		m.lastFile = ""
	}
	p := tea.NewProgram(
		m,
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
		lastFile:               cfg.LastFile(),
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

// replaceBuffer0 makes the given chart the only chart being edited, used by
// both the start menu and opening a chart from the start menu.
func (m *model) replaceBuffer0(canvas *Canvas, filename string, panX, panY int) {
	m.buffers[0] = Buffer{
		canvas:    canvas,
		undoStack: []Action{},
		redoStack: []Action{},
		filename:  filename,
		panX:      panX,
		panY:      panY,
	}
	m.currentBufferIndex = 0
}

// rememberChart records a chart as the most recently opened one, for the start
// menu's resume option.
func (m *model) rememberChart(path string) {
	if m.config == nil {
		return
	}
	m.config.RememberLastFile(path)
	m.lastFile = path
}

// fileListWindow is the number of saved charts shown at once in the open
// dialog; longer lists scroll with a scrollbar.
const fileListWindow = 10

// chartDisplayName strips the .sav extension used for saved charts.
func chartDisplayName(file string) string {
	return strings.TrimSuffix(file, filepath.Ext(file))
}

// fuzzyMatch reports whether pattern's runes appear in order (not necessarily
// adjacent) in s, case-insensitively.
// ponytail: subsequence filter only, keeps the alphabetical order; add
// relevance scoring if picking from many similar names gets annoying.
func fuzzyMatch(s, pattern string) bool {
	want := []rune(strings.ToLower(pattern))
	i := 0
	for _, r := range strings.ToLower(s) {
		if i < len(want) && r == want[i] {
			i++
		}
	}
	return i == len(want)
}

// applyFileFilter rebuilds the displayed chart list from allFiles and the
// current fuzzy filter, keeping the selection and scroll window valid.
func (m *model) applyFileFilter() {
	if m.fileFilter == "" {
		m.fileList = m.allFiles
	} else {
		m.fileList = nil
		for _, f := range m.allFiles {
			if fuzzyMatch(chartDisplayName(f), m.fileFilter) {
				m.fileList = append(m.fileList, f)
			}
		}
	}
	if m.selectedFileIndex >= len(m.fileList) {
		m.selectedFileIndex = len(m.fileList) - 1
	}
	if m.selectedFileIndex < 0 && len(m.fileList) > 0 {
		m.selectedFileIndex = 0
	}
	if m.selectedFileIndex >= 0 {
		m.filename = chartDisplayName(m.fileList[m.selectedFileIndex])
	}
	m.clampFileScroll()
}

// fileSelectionCurrent reports whether the typed filename still matches the
// highlighted entry, i.e. the user has not started typing a name of their own.
func (m *model) fileSelectionCurrent() bool {
	if m.filename == "" {
		return true
	}
	if m.selectedFileIndex < 0 || m.selectedFileIndex >= len(m.fileList) {
		return false
	}
	return m.filename == chartDisplayName(m.fileList[m.selectedFileIndex])
}

// refilterFromTop re-applies the fuzzy filter and highlights the first match,
// the way a search prompt is expected to behave as you type.
func (m *model) refilterFromTop() {
	m.selectedFileIndex = 0
	m.fileScroll = 0
	m.applyFileFilter()
}

// moveFileSelection moves the highlight by delta (wrapping at both ends) and
// keeps the typed filename and the scroll window in sync.
func (m *model) moveFileSelection(delta int) {
	if len(m.fileList) == 0 {
		return
	}
	if m.selectedFileIndex < 0 {
		if delta < 0 {
			m.selectedFileIndex = len(m.fileList) - 1
		} else {
			m.selectedFileIndex = 0
		}
	} else {
		m.selectedFileIndex = (m.selectedFileIndex + delta%len(m.fileList) + len(m.fileList)) % len(m.fileList)
	}
	m.filename = chartDisplayName(m.fileList[m.selectedFileIndex])
	m.clampFileScroll()
}

// clampFileScroll scrolls the window just far enough to show the selection.
func (m *model) clampFileScroll() {
	if m.selectedFileIndex >= 0 {
		if m.selectedFileIndex < m.fileScroll {
			m.fileScroll = m.selectedFileIndex
		} else if m.selectedFileIndex >= m.fileScroll+fileListWindow {
			m.fileScroll = m.selectedFileIndex - fileListWindow + 1
		}
	}
	if max := len(m.fileList) - fileListWindow; m.fileScroll > max {
		m.fileScroll = max
	}
	if m.fileScroll < 0 {
		m.fileScroll = 0
	}
}

func (m *model) scanTxtFiles() {
	m.allFiles = []string{}
	m.fileList = []string{}
	m.fileFilter = ""
	m.fileSearch = false
	m.fileScroll = 0
	m.selectedFileIndex = -1
	dir := ""
	if m.config != nil && m.config.SaveDirectory != "" {
		dir = m.config.SaveDirectory
	} else {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".sav") {
			m.allFiles = append(m.allFiles, entry.Name())
		}
	}
	sort.Strings(m.allFiles)
	if len(m.allFiles) > 0 {
		m.selectedFileIndex = 0
	}
	m.applyFileFilter()
}
