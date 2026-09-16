package tui

import (
	"os"
	"path/filepath"
	"slices"
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

// fileListRows is how many saved charts the open dialog shows at once: as many
// as the terminal fits, since the dialog draws 6 rows of chrome around the
// list and wants a row of margin above and below it.
func (m model) fileListRows() int {
	rows := m.height - 8
	if rows > len(m.allFiles) {
		rows = len(m.allFiles)
	}
	if rows < 1 {
		rows = 1
	}
	return rows
}

// fileScrollMax is the largest fileScroll that still fills the list window.
func (m model) fileScrollMax() int {
	return max(len(m.fileList)-m.fileListRows(), 0)
}

// fileThumbLen is the scrollbar thumb's height in list rows, or 0 when the
// whole list fits and no scrollbar is drawn.
func (m model) fileThumbLen() int {
	rows := m.fileListRows()
	if len(m.fileList) <= rows {
		return 0
	}
	return max(rows*rows/len(m.fileList), 1)
}

// fileScrollThumb returns the first and one-past-last list row the scrollbar
// thumb covers, or 0, 0 when there is no scrollbar.
func (m model) fileScrollThumb() (int, int) {
	length := m.fileThumbLen()
	if length == 0 {
		return 0, 0
	}
	rows := m.fileListRows()
	start := min(m.fileScroll, m.fileScrollMax()) * (rows - length) / (len(m.fileList) - rows)
	return start, start + length
}

// dragFileScrollTo scrolls so the scrollbar thumb starts at list row row, the
// inverse of fileScrollThumb.
func (m *model) dragFileScrollTo(row int) {
	length := m.fileThumbLen()
	if length == 0 {
		return
	}
	rows := m.fileListRows()
	m.fileScroll = min(max(row, 0), rows-length) * (len(m.fileList) - rows) / (rows - length)
	m.scrollFileList(0)
}

// scrollFileList scrolls the window by delta rows without moving the
// highlight, the way a wheel or a scrollbar drag is expected to behave.
func (m *model) scrollFileList(delta int) {
	m.fileScroll = min(max(m.fileScroll+delta, 0), m.fileScrollMax())
}

// saveExt is the extension new charts are saved with; chartExts are all the
// extensions recognized when opening, so charts written by older versions
// under .sav keep working.
const saveExt = ".flerm"

var chartExts = []string{saveExt, ".sav"}

// hasChartExt reports whether file already carries a recognized chart
// extension.
func hasChartExt(file string) bool {
	return slices.Contains(chartExts, strings.ToLower(filepath.Ext(file)))
}

// chartDisplayName strips the extension used for saved charts.
func chartDisplayName(file string) string {
	return strings.TrimSuffix(file, filepath.Ext(file))
}

// resolveChartPath finds the chart a typed name refers to, looking in the
// configured save directory before the working directory and trying each
// recognized extension when the name carries none. It returns "" if no such
// chart exists.
func (m *model) resolveChartPath(name string) string {
	names := []string{name}
	if !hasChartExt(name) {
		names = nil
		for _, ext := range chartExts {
			names = append(names, name+ext)
		}
	}
	for _, n := range names {
		candidates := []string{n}
		if m.config != nil && m.config.SaveDirectory != "" {
			candidates = []string{m.config.GetSavePath(n), n}
		}
		for _, path := range candidates {
			if info, err := os.Stat(path); err == nil && !info.IsDir() {
				return path
			}
		}
	}
	return ""
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

// selectFileIndex highlights idx, keeping the typed filename and the scroll
// window in sync. Both the arrow keys and a mouse click land here.
func (m *model) selectFileIndex(idx int) {
	if idx < 0 || idx >= len(m.fileList) {
		return
	}
	m.selectedFileIndex = idx
	m.filename = chartDisplayName(m.fileList[idx])
	m.clampFileScroll()
}

// moveFileSelection moves the highlight by delta, wrapping at both ends.
func (m *model) moveFileSelection(delta int) {
	if len(m.fileList) == 0 {
		return
	}
	idx := m.selectedFileIndex
	if idx < 0 {
		if delta < 0 {
			idx = len(m.fileList) - 1
		} else {
			idx = 0
		}
	} else {
		idx = (idx + delta%len(m.fileList) + len(m.fileList)) % len(m.fileList)
	}
	m.selectFileIndex(idx)
}

// clampFileScroll scrolls the window just far enough to show the selection.
func (m *model) clampFileScroll() {
	rows := m.fileListRows()
	if m.selectedFileIndex >= 0 {
		if m.selectedFileIndex < m.fileScroll {
			m.fileScroll = m.selectedFileIndex
		} else if m.selectedFileIndex >= m.fileScroll+rows {
			m.fileScroll = m.selectedFileIndex - rows + 1
		}
	}
	m.scrollFileList(0)
}

func (m *model) scanTxtFiles() {
	m.allFiles = []string{}
	m.fileList = []string{}
	m.fileFilter = ""
	m.fileSearch = false
	m.fileScroll = 0
	m.draggingFileScroll = false
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
		if !entry.IsDir() && hasChartExt(entry.Name()) {
			m.allFiles = append(m.allFiles, entry.Name())
		}
	}
	sort.Strings(m.allFiles)
	if len(m.allFiles) > 0 {
		m.selectedFileIndex = 0
	}
	m.applyFileFilter()
}
