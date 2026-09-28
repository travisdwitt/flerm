package tui

import (
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	cv "flerm/internal/canvas"
	"flerm/internal/config"

	tea "github.com/charmbracelet/bubbletea"
)

func Run(noResume bool, dateOverride string) error {
	m := initialModel()
	if noResume {
		m.lastFile = ""
	}
	if dateOverride != "" {
		m.effect = effectForOverride(dateOverride)
	} else {
		m.effect = effectForDate(time.Now())
	}
	_, err := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseAllMotion()).Run()
	return err
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func sign(n int) int {
	switch {
	case n > 0:
		return 1
	case n < 0:
		return -1
	}
	return 0
}

func initialModel() model {
	cfg := config.Load()
	m := model{
		buffers:            []Buffer{{canvas: cv.NewCanvas()}},
		lastFile:           cfg.LastFile(),
		mode:               ModeNormal,
		selectedBox:        -1,
		selectedText:       -1,
		connectionFrom:     -1,
		connectionFromLine: -1,
		config:             cfg,
		selectionStartX:    CoordUnset,
		selectionStartY:    CoordUnset,
		selBox:             -1,
		selText:            -1,
		selConn:            -1,
		menuTargetBox:      -1,
		menuTargetText:     -1,
		menuTargetConn:     -1,
	}
	if cfg.StartMenu {
		m.mode = ModeStartup
	}
	m.clearGroupSelection()
	return m
}

func (m *model) ensureCursorInBounds() {
	m.cursorX = max(0, min(m.cursorX, max(m.width-1, 0)))
	m.cursorY = max(0, min(m.cursorY, max(m.height-2, 0)))
}

func (m *model) resetView() {
	m.cursorX, m.cursorY = 0, 0
	m.errorMessage, m.successMessage = "", ""
}

func (m *model) newChart() {
	*m.getCurrentBuffer() = Buffer{canvas: cv.NewCanvas()}
	m.resetView()
}

func (m *model) closeBuffer() {
	if len(m.buffers) > 1 {
		m.buffers = slices.Delete(m.buffers, m.currentBufferIndex, m.currentBufferIndex+1)
		m.currentBufferIndex = max(m.currentBufferIndex-1, 0)
	} else {
		m.buffers = []Buffer{{canvas: cv.NewCanvas()}}
		m.currentBufferIndex = 0
		m.mode = ModeStartup
	}
	m.resetView()
}

func (m *model) rememberChart(path string) {
	m.config.RememberLastFile(path)
	m.lastFile = path
}

func (m *model) openChart(path string) error {
	canvas := cv.NewCanvas()
	panX, panY, err := canvas.LoadFromFileWithPan(path)
	if err != nil {
		return err
	}
	m.rememberChart(path)
	buf := Buffer{canvas: canvas, filename: path, panX: panX, panY: panY}
	switch {
	case m.fromStartup:
		m.buffers[0] = buf
		m.currentBufferIndex = 0
		m.fromStartup = false
	case m.openInNewBuffer:
		m.addNewBuffer(buf)
		m.openInNewBuffer = false
	default:
		*m.getCurrentBuffer() = buf
	}
	m.errorMessage = ""
	return nil
}

func (m *model) saveChart(path string) bool {
	buf := m.getCurrentBuffer()
	if err := buf.canvas.SaveToFileWithPan(path, buf.panX, buf.panY); err != nil {
		m.errorMessage = "Error saving file: " + err.Error()
		return false
	}
	buf.filename = path
	buf.savedAt = len(buf.undoStack)
	m.rememberChart(path)
	m.reportWritten("Saved", path)
	return true
}

func (m *model) export(name, ext, label string, write func(string) error) bool {
	base := filepath.Base(name)
	if !strings.HasSuffix(strings.ToLower(base), ext) {
		base += ext
	}
	path := m.config.GetSavePath(base)
	if err := write(path); err != nil {
		m.errorMessage = "Error exporting " + label + ": " + err.Error()
		return false
	}
	m.reportWritten("Exported", path)
	return true
}

func (m *model) reportWritten(verb, path string) {
	absPath, _ := filepath.Abs(path)
	m.successMessage = verb + " to " + absPath
	m.errorMessage = ""
}

func (m *model) currentChartName() string {
	if buf := m.getCurrentBuffer(); buf.filename != "" {
		return chartDisplayName(filepath.Base(buf.filename))
	}
	return ""
}

func (m model) fileListRows() int {
	return max(min(m.height-8, len(m.allFiles)), 1)
}

func (m model) fileScrollMax() int {
	return max(len(m.fileList)-m.fileListRows(), 0)
}

func (m model) fileThumbLen() int {
	rows := m.fileListRows()
	if len(m.fileList) <= rows {
		return 0
	}
	return max(rows*rows/len(m.fileList), 1)
}

func (m model) fileScrollThumb() (int, int) {
	length := m.fileThumbLen()
	if length == 0 {
		return 0, 0
	}
	rows := m.fileListRows()
	start := min(m.fileScroll, m.fileScrollMax()) * (rows - length) / (len(m.fileList) - rows)
	return start, start + length
}

func (m *model) dragFileScrollTo(row int) {
	length := m.fileThumbLen()
	if length == 0 {
		return
	}
	rows := m.fileListRows()
	m.fileScroll = min(max(row, 0), rows-length) * (len(m.fileList) - rows) / (rows - length)
	m.scrollFileList(0)
}

func (m *model) scrollFileList(delta int) {
	m.fileScroll = min(max(m.fileScroll+delta, 0), m.fileScrollMax())
}

const saveExt = ".flerm"

var chartExts = []string{saveExt, ".sav"}

func hasChartExt(file string) bool {
	return slices.Contains(chartExts, strings.ToLower(filepath.Ext(file)))
}

func chartDisplayName(file string) string {
	return strings.TrimSuffix(file, filepath.Ext(file))
}

func (m *model) resolveChartPath(name string) string {
	names := []string{name}
	if !hasChartExt(name) {
		names = nil
		for _, ext := range chartExts {
			names = append(names, name+ext)
		}
	}
	for _, n := range names {
		for _, path := range []string{m.config.GetSavePath(n), n} {
			if info, err := os.Stat(path); err == nil && !info.IsDir() {
				return path
			}
		}
	}
	return ""
}

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
	m.selectedFileIndex = min(m.selectedFileIndex, len(m.fileList)-1)
	if m.selectedFileIndex < 0 && len(m.fileList) > 0 {
		m.selectedFileIndex = 0
	}
	if m.selectedFileIndex >= 0 {
		m.filename = chartDisplayName(m.fileList[m.selectedFileIndex])
	}
	m.clampFileScroll()
}

func (m *model) fileSelectionCurrent() bool {
	if m.filename == "" {
		return true
	}
	if m.selectedFileIndex < 0 || m.selectedFileIndex >= len(m.fileList) {
		return false
	}
	return m.filename == chartDisplayName(m.fileList[m.selectedFileIndex])
}

func (m *model) refilterFromTop() {
	m.selectedFileIndex = 0
	m.fileScroll = 0
	m.applyFileFilter()
}

func (m *model) selectFileIndex(idx int) {
	if idx < 0 || idx >= len(m.fileList) {
		return
	}
	m.selectedFileIndex = idx
	m.filename = chartDisplayName(m.fileList[idx])
	m.clampFileScroll()
}

func (m *model) moveFileSelection(delta int) {
	n := len(m.fileList)
	if n == 0 {
		return
	}
	idx := m.selectedFileIndex
	switch {
	case idx >= 0:
		idx = (idx + delta%n + n) % n
	case delta < 0:
		idx = n - 1
	default:
		idx = 0
	}
	m.selectFileIndex(idx)
}

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

func (m *model) deleteSelectedFile() {
	if m.confirmFileIndex < 0 || m.confirmFileIndex >= len(m.fileList) {
		return
	}
	filename := m.fileList[m.confirmFileIndex]
	if err := os.Remove(m.config.GetSavePath(filename)); err != nil {
		m.errorMessage = "Error deleting file: " + err.Error()
		return
	}
	if i := slices.Index(m.allFiles, filename); i >= 0 {
		m.allFiles = slices.Delete(slices.Clone(m.allFiles), i, i+1)
	}
	m.applyFileFilter()
	if m.selectedFileIndex < 0 {
		m.filename = ""
	}
	m.successMessage = "Deleted " + chartDisplayName(filename)
}

func (m *model) scanTxtFiles() {
	m.allFiles = nil
	m.fileList = nil
	m.fileFilter = ""
	m.fileSearch = false
	m.fileScroll = 0
	m.draggingFileScroll = false
	m.selectedFileIndex = -1
	dir := m.config.SaveDirectory
	if dir == "" {
		var err error
		if dir, err = os.Getwd(); err != nil {
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
