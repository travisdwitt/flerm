package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	cv "flerm/internal/canvas"
)

func chromeBG() string    { return cv.AnsiTier("\033[48;5;236m", "\033[100m") }
func chromeReset() string { return cv.Ansi("\033[0m") }
func chromeFG() string    { return cv.Ansi("\033[32m") }
func chromeFGOff() string { return cv.Ansi("\033[39m") }

func displayLen(s string) int {
	n, inEscape := 0, false
	for _, r := range s {
		switch {
		case inEscape:
			inEscape = r != 'm'
		case r == '\033':
			inEscape = true
		default:
			n++
		}
	}
	return n
}

func clipDisplay(s string, width int) string {
	var out strings.Builder
	n, inEscape := 0, false
	for _, r := range s {
		if inEscape {
			inEscape = r != 'm'
			out.WriteRune(r)
			continue
		}
		if r == '\033' {
			inEscape = true
			out.WriteRune(r)
			continue
		}
		if n == width {
			break
		}
		out.WriteRune(r)
		n++
	}
	return out.String()
}

func chromeLine(s string, width int) string {
	switch pad := width - displayLen(s); {
	case pad > 0:
		s += strings.Repeat(" ", pad)
	case pad < 0:
		s = clipDisplay(s, width)
	}
	return chromeBG() + s + chromeReset()
}

func (m *model) renderBufferBar() string {
	var bar strings.Builder
	write := func(s string) { bar.WriteString(s) }
	bracket := func(s string) {
		bar.WriteString(chromeFG())
		write(s)
		bar.WriteString(chromeFGOff())
	}
	write("Open Charts: ")
	for i, buf := range m.buffers {
		if i > 0 {
			write(" | ")
		}
		bufName := fmt.Sprintf("Buffer %d", i+1)
		if buf.filename != "" {
			bufName = chartDisplayName(filepath.Base(buf.filename))
		}
		if i == m.currentBufferIndex {
			bracket("[")
			write(bufName)
			bracket("]")
		} else {
			write(" " + bufName + " ")
		}
	}
	return bar.String()
}

func (m *model) hoverBoxTooltip(worldX, worldY, screenX, screenY int) {
	canvas := m.getCanvas()
	if boxID := canvas.GetBoxAt(worldX, worldY); boxID != -1 && canvas.Boxes()[boxID].Tooltip != "" {
		m.showTooltip, m.tooltipStyled = true, true
		m.tooltipText, m.tooltipBoxID = canvas.Boxes()[boxID].Tooltip, -1
		m.tooltipX, m.tooltipY = screenX, screenY
		return
	}
	if m.tooltipStyled {
		m.showTooltip, m.tooltipStyled = false, false
	}
}

func (m *model) updateTooltip() {
	m.showTooltip = false
	m.tooltipStyled = false
	m.tooltipBoxID = -1
	if m.mode != ModeNormal {
		return
	}
	worldX, worldY := m.worldCursor()
	canvas := m.getCanvas()
	boxID := canvas.GetBoxAt(worldX, worldY)
	if boxID == -1 {
		return
	}
	if x, y, ok := canvas.TooltipMarkPos(boxID); ok && x == worldX && y == worldY {
		m.showTooltip, m.tooltipStyled = true, true
		m.tooltipText, m.tooltipBoxID = canvas.Boxes()[boxID].Tooltip, -1
		m.tooltipX, m.tooltipY = m.cursorX, m.cursorY
		return
	}
	if box := &canvas.Boxes()[boxID]; box.IsTextTruncated() {
		m.showTooltip = true
		m.tooltipText = box.GetText()
		m.tooltipX, m.tooltipY = m.cursorX, m.cursorY
		m.tooltipBoxID = boxID
	}
}

func (m model) View() string {
	if m.help && m.mode != ModeStartup {
		return m.helpView()
	}
	if m.mode == ModeStartup {
		return m.renderStartupMenu()
	}
	if m.mode == ModeFileInput && m.fileOp == FileOpOpen {
		return m.renderFileMenu()
	}

	selectedBox := -1
	if m.mode == ModeResize || m.mode == ModeMove {
		selectedBox = m.selectedBox
	}

	renderWidth := max(m.width, 1)
	showBufferBar := m.bufferBarOffset() == 1
	renderHeight := max(m.height-1-m.bufferBarOffset(), 1)
	panX, panY := m.getPanOffset()

	previewFromX, previewFromY, previewToX, previewToY := CoordUnset, CoordUnset, CoordUnset, CoordUnset
	var previewWaypoints []point
	if m.connectionFrom != -1 || m.connectionFromLine != -1 {
		previewFromX, previewFromY = m.connectionFromX, m.connectionFromY
		previewWaypoints = m.connectionWaypoints
		previewToX, previewToY = m.cursorX+panX, m.cursorY+panY
	}

	cursorX := max(0, min(m.cursorX, renderWidth-1))
	cursorY := max(0, min(m.cursorY, renderHeight-1))
	showCursor := m.mode != ModeFileInput && m.mode != ModeEditing && m.mode != ModeTextInput && m.mode != ModeTitleEdit && m.mode != ModeTooltipEdit

	editBoxID, editTextID, editTextX, editTextY := m.editTarget()

	selectionStartX, selectionStartY := CoordUnset, CoordUnset
	selectionEndX, selectionEndY := CoordUnset, CoordUnset
	if m.mode == ModeMultiSelect && m.selectionStartX != CoordUnset && m.selectionStartY != CoordUnset {
		selectionStartX, selectionStartY = m.selectionStartX, m.selectionStartY
		selectionEndX, selectionEndY = m.cursorX+panX, m.cursorY+panY
	}

	editSelStart, editSelEnd := -1, -1
	if m.mode == ModeEditing && m.hasEditSelection() {
		editSelStart, editSelEnd = m.editSelectionStart, m.editSelectionEnd
	}

	r := m.getCanvas().RenderRaw(renderWidth, renderHeight, selectedBox, previewFromX, previewFromY, previewWaypoints, previewToX, previewToY, panX, panY, cursorX, cursorY, showCursor, editBoxID, editTextID, m.editCursorPos, m.editText, editTextX, editTextY, selectionStartX, selectionStartY, selectionEndX, selectionEndY, m.mode == ModeBoxJump || m.allTooltips, editSelStart, editSelEnd)

	m.overlaySelection(r, panX, panY)
	if m.mode == ModeTooltipEdit {
		m.showTooltip, m.tooltipStyled, m.tooltipBoxID = true, true, -1
		m.tooltipText = editCaret(m.editText, m.editCursorPos, m.editSelectionStart, m.editSelectionEnd)
	}
	if m.showTooltip && m.tooltipText != "" {
		m.overlayTooltipOnRenderResult(r)
	}
	if m.mode == ModeContextMenu {
		m.overlayContextMenu(r)
	}
	if m.minimap {
		m.overlayMinimap(r, panX, panY)
	}
	if m.allTooltips {
		m.overlayTooltipSidebar(r)
	}

	var result strings.Builder
	if showBufferBar {
		result.WriteString(chromeLine(m.renderBufferBar(), renderWidth))
		result.WriteString("\n")
	}
	result.WriteString(strings.Join(r.ApplyColors(), "\n"))
	result.WriteString("\n")
	result.WriteString(chromeLine(m.statusLine(), renderWidth))
	return result.String()
}

const minimapW, minimapH = 32, 12

const minimapLabel = "minimap"

// '+'/'-'/'|' outline, blank inside; a box only 1-2 cells wide or tall is all edge
func minimapFrame(relX, relY, w, h int) rune {
	edgeX, edgeY := relX == 0 || relX == w-1, relY == 0 || relY == h-1
	switch {
	case edgeX && edgeY:
		return '+'
	case edgeY:
		return '-'
	case edgeX:
		return '|'
	}
	return ' '
}

func minimapArrow(dx, dy int) rune {
	switch {
	case abs(dx) > 2*abs(dy):
		if dx < 0 {
			return '←'
		}
		return '→'
	case abs(dy) > 2*abs(dx):
		if dy < 0 {
			return '↑'
		}
		return '↓'
	case dx < 0 && dy < 0:
		return '↖'
	case dx < 0:
		return '↙'
	case dy < 0:
		return '↗'
	}
	return '↘'
}

func (m model) overlayMinimap(r *RenderResult, panX, panY int) {
	w, h := min(minimapW, r.Width), min(minimapH, r.Height)
	if w < 5 || h < 5 {
		return
	}
	innerW, innerH := w-2, h-2
	originX, originY := r.Width-w, 0

	cell := func(px, py int, ch rune, color int) {
		if py < 0 || py >= len(r.Canvas) || px < 0 || px >= len(r.Canvas[py]) {
			return
		}
		r.Canvas[py][px] = ch
		r.ColorMap[py][px] = color
	}
	hline := func(py int, left, right rune) {
		cell(originX, py, left, colorMenuBorder)
		for i := 0; i < innerW; i++ {
			cell(originX+1+i, py, '─', colorMenuBorder)
		}
		cell(originX+w-1, py, right, colorMenuBorder)
	}

	hline(originY, '┌', '┐')
	hline(originY+h-1, '└', '┘')
	for i := 0; i < innerH; i++ {
		py := originY + 1 + i
		cell(originX, py, '│', colorMenuBorder)
		for j := 0; j < innerW; j++ {
			cell(originX+1+j, py, ' ', -1)
		}
		cell(originX+w-1, py, '│', colorMenuBorder)
	}

	for i, ch := range minimapLabel {
		cell(originX+w-len(minimapLabel)+i, originY+h, ch, 7)
	}

	canvas := m.getCanvas()
	minX, minY, maxX, maxY := canvas.GetFullBounds()
	if maxX < minX || maxY < minY {
		return
	}
	vx0, vy0 := panX, panY
	vx1, vy1 := panX+r.Width-1, panY+r.Height-1

	if maxX < vx0 || minX > vx1 || maxY < vy0 || minY > vy1 {
		ch := minimapArrow((minX+maxX)/2-(vx0+vx1)/2, (minY+maxY)/2-(vy0+vy1)/2)
		py := originY + 1 + innerH/2
		for i := -1; i <= 1; i++ {
			cell(originX+1+innerW/2+i, py, ch, colorMouseSelect)
		}
		return
	}

	bx0, by0 := min(minX, vx0), min(minY, vy0)
	bx1, by1 := max(maxX, vx1), max(maxY, vy1)
	spanX, spanY := bx1-bx0+1, by1-by0+1

	scale := max((spanX+innerW-1)/innerW, (spanY+innerH-1)/innerH, 1)
	useW, useH := (spanX+scale-1)/scale, (spanY+scale-1)/scale
	offX, offY := (innerW-useW)/2, (innerH-useH)/2
	mapX := func(wx int) int { return offX + min(max((wx-bx0)/scale, 0), useW-1) }
	mapY := func(wy int) int { return offY + min(max((wy-by0)/scale, 0), useH-1) }

	plot := func(cells []point, ch rune) {
		for _, c := range cells {
			cell(originX+1+mapX(c.X), originY+1+mapY(c.Y), ch, -1)
		}
	}
	for i := range canvas.Connections() {
		plot(canvas.GetConnectionCells(i), '.')
	}
	for i := range canvas.Texts() {
		plot(canvas.GetTextCells(i), '.')
	}
	for _, box := range canvas.Boxes() {
		x0, y0 := mapX(box.X), mapY(box.Y)
		x1, y1 := mapX(box.X+box.Width-1), mapY(box.Y+box.Height-1)
		for y := y0; y <= y1; y++ {
			for x := x0; x <= x1; x++ {
				cell(originX+1+x, originY+1+y, minimapFrame(x-x0, y-y0, x1-x0+1, y1-y0+1), -1)
			}
		}
	}

	x0, y0 := mapX(vx0), mapY(vy0)
	x1, y1 := mapX(vx1), mapY(vy1)
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			if ch := minimapFrame(x-x0, y-y0, x1-x0+1, y1-y0+1); ch != ' ' {
				cell(originX+1+x, originY+1+y, ch, colorMouseSelect)
			}
		}
	}
}

func editStatus(text string, pos, selStart, selEnd int) string {
	return editCaret(strings.ReplaceAll(text, "\n", " "), pos, selStart, selEnd)
}

func editCaret(text string, pos, selStart, selEnd int) string {
	r := []rune(text)
	if selStart >= 0 && selEnd >= 0 && selStart != selEnd {
		s, e := min(min(selStart, selEnd), len(r)), min(max(selStart, selEnd), len(r))
		return string(r[:s]) + "[" + string(r[s:e]) + "]" + string(r[e:])
	}
	if pos >= len(r) {
		return string(r) + "█"
	}
	r[pos] = '█'
	return string(r)
}

func colorLabel(color int) string {
	if color < 0 || color >= numColors {
		return ""
	}
	name := " " + colorNames[color]
	swatch := cv.ColorFG(color)
	if swatch == "" {
		return name
	}
	return " " + swatch + "■" + chromeFGOff() + name
}

func (m model) cursorObject() string {
	canvas := m.getCanvas()
	worldX, worldY := m.worldCursor()
	switch boxID, textID := canvas.GetBoxAt(worldX, worldY), canvas.GetTextAt(worldX, worldY); {
	case boxID != -1:
		return fmt.Sprintf("Box %d%s", boxID, colorLabel(canvas.Boxes()[boxID].Color))
	case textID != -1:
		return fmt.Sprintf("Text %d%s", textID, colorLabel(canvas.Texts()[textID].Color))
	default:
		idx, _, _ := canvas.FindNearestPointOnConnection(worldX, worldY)
		if idx == -1 {
			return ""
		}
		conn := canvas.Connections()[idx]
		return fmt.Sprintf("Connection %s to %s%s", connEndpoint(conn.FromID), connEndpoint(conn.ToID), colorLabel(conn.Color))
	}
}

func connEndpoint(id int) string {
	if id < 0 {
		return "line"
	}
	return fmt.Sprintf("Box %d", id)
}

func (m model) statusLine() string {
	switch m.mode {
	case ModeEditing:
		hint := ""
		if m.hasEditSelection() {
			start, end := m.editSelectionBounds()
			hint = fmt.Sprintf(" | %d chars selected", end-start)
		}
		target := ""
		if m.selectedBox != -1 {
			target = fmt.Sprintf("Box %d | ", m.selectedBox)
		} else if m.selectedText != -1 {
			target = fmt.Sprintf("Text %d | ", m.selectedText)
		}
		return fmt.Sprintf("Mode: EDIT | %sText: %s%s | Home/End, Shift+←→↑↓/Home/End=select, y=copy, Ctrl+S/Esc=save", target, editStatus(m.editText, m.editCursorPos, m.editSelectionStart, m.editSelectionEnd), hint)
	case ModeTextInput:
		return fmt.Sprintf("Mode: TEXT | Text: %s | ←/→=move cursor, Enter=newline, Ctrl+S/Esc=save", editStatus(m.editText, m.editCursorPos, -1, -1))
	case ModeTitleEdit:
		return fmt.Sprintf("Mode: TITLE EDIT | Title: %s | ←/→/↑/↓=move cursor, Enter=newline, Ctrl+S/Esc=save", editStatus(m.editText, m.editCursorPos, -1, -1))
	case ModeResize:
		return fmt.Sprintf("Mode: RESIZE | Box %d | hjkl/arrows=resize, Enter=finish, Esc=cancel", m.selectedBox)
	case ModeMove:
		const keys = "hjkl/arrows=move, Enter=finish, Esc=cancel"
		switch {
		case m.hasGroupSelection():
			var parts []string
			for _, c := range []struct {
				n    int
				name string
			}{{len(m.selectedBoxes), "boxes"}, {len(m.selectedTexts), "texts"}, {len(m.selectedConnections), "connections"}} {
				if c.n > 0 {
					parts = append(parts, fmt.Sprintf("%d %s", c.n, c.name))
				}
			}
			return fmt.Sprintf("Mode: MOVE | %s | %s", strings.Join(parts, ", "), keys)
		case m.selectedBox != -1:
			return fmt.Sprintf("Mode: MOVE | Box %d | %s", m.selectedBox, keys)
		case m.selectedText != -1:
			return fmt.Sprintf("Mode: MOVE | Text %d | %s", m.selectedText, keys)
		}
		return "Mode: MOVE | " + keys
	case ModeMultiSelect:
		if m.selectionStartX == CoordUnset || m.selectionStartY == CoordUnset {
			return "Mode: MULTI-SELECT | drag to select, or Enter to start a selection at the cursor, Esc=cancel"
		}
		return "Mode: MULTI-SELECT | hjkl/arrows=draw selection, Enter=select and move, Esc=cancel"
	case ModeFileInput:
		opStr := map[FileOperation]string{FileOpSave: "Save", FileOpOpen: "Open", FileOpSavePNG: "Export PNG", FileOpSaveVisualTXT: "Export Visual TXT"}[m.fileOp]
		if m.errorMessage != "" {
			return fmt.Sprintf("Mode: FILE | ERROR: %s | %s filename: %s | Enter=retry, Esc=cancel", m.errorMessage, opStr, m.filename)
		}
		return fmt.Sprintf("Mode: FILE | %s filename: %s | Enter=confirm, Esc=cancel", opStr, m.filename)
	case ModeConfirm:
		var message string
		switch m.confirmAction {
		case ConfirmDeleteBox:
			message = "Delete this box? (y/n)"
		case ConfirmDeleteText:
			message = "Delete this text? (y/n)"
		case ConfirmDeleteConnection:
			message = "Delete this connection? (y/n)"
		case ConfirmQuit:
			message = "Quit Flerm? (y/n)"
			if m.unsavedChanges() {
				message = "Quit Flerm? Unsaved changes will be lost. (y/n)"
			}
		case ConfirmNewChart:
			message = "Create new chart? Unsaved changes will be lost. (y/n)"
		case ConfirmCloseBuffer:
			message = "Close current buffer? Unsaved changes will be lost. (y/n)"
		case ConfirmOverwriteFile:
			message = fmt.Sprintf("File %s already exists. Overwrite? (y/n)", m.filename)
		case ConfirmChooseExportType:
			message = "Export as PNG (p) or Visual TXT (t)? Press Esc to cancel"
		case ConfirmDeleteBoxOrTooltip:
			message = "Delete the box (b) or just its tooltip (t)? Press Esc to cancel"
		}
		return "Mode: CONFIRM | " + message
	case ModeContextMenu:
		return "Mode: MENU | ↑/↓ or hover=navigate, →/Enter=open submenu, ←=back, click=select, Esc/right-click=cancel"
	case ModeTooltipEdit:
		return fmt.Sprintf("Mode: TOOLTIP | Box %d | Ctrl+S/Esc=save, Enter=newline", m.selectedBox)
	case ModeBoxJump:
		return fmt.Sprintf("Mode: BOX JUMP | Enter box number: %s | Enter=jump, Esc=cancel", m.boxJumpInput)
	}

	modeStr := "NORMAL"
	if m.zPanMode {
		modeStr = "PAN"
	}
	if m.highlightMode {
		modeStr = "HIGHLIGHT"
	}
	status := fmt.Sprintf("Mode: %s | Cursor: (%d,%d)", modeStr, m.cursorX, m.cursorY)
	if obj := m.cursorObject(); obj != "" {
		status += " | " + obj
	}
	if m.highlightMode {
		status += fmt.Sprintf(" | Color:%s (%d/%d)", colorLabel(m.selectedColor), m.selectedColor+1, numColors)
	}
	if m.connectionFrom != -1 {
		status += fmt.Sprintf(" | Connection from box %d (select target)", m.connectionFrom)
	} else if m.connectionFromLine != -1 {
		status += " | Connection from line (select target)"
	}
	if m.selectedBox != -1 {
		status += fmt.Sprintf(" | Selected: Box %d", m.selectedBox)
	}
	if m.successMessage != "" {
		status += " | " + m.successMessage
	}
	if m.errorMessage != "" {
		status += " | ERROR: " + m.errorMessage
	} else if m.successMessage == "" {
		status += " | ? for help | q to quit"
	}
	return status
}

// wrap only what a box would not fit either; below that the window grows with the text
const (
	tooltipWrapLimit  = 41
	tooltipMinContent = 11
)

type tooltipChar struct {
	char        rune
	origCharIdx int
}

const (
	sidebarW     = minimapW
	sidebarLabel = "tooltips"
)

func wrapWords(text string, width int) []string {
	var lines []string
	for _, paragraph := range strings.Split(text, "\n") {
		cur := ""
		for _, word := range strings.Fields(paragraph) {
			switch {
			case cur == "":
				cur = word
			case runeLen(cur)+1+runeLen(word) <= width:
				cur += " " + word
			default:
				lines = append(lines, cur)
				cur = word
			}
		}
		lines = append(lines, cur)
	}
	return lines
}

func (m model) tooltipSidebarRows(boxW int) []string {
	var rows []string
	for id, box := range m.getCanvas().Boxes() {
		if box.Tooltip == "" {
			continue
		}
		rows = append(rows, "┌"+strings.Repeat("─", boxW-2)+"┐")
		for _, line := range append([]string{fmt.Sprintf("Box [%d]:", id)}, wrapWords(box.Tooltip, boxW-4)...) {
			if runeLen(line) > boxW-4 {
				line = string([]rune(line)[:boxW-4])
			}
			rows = append(rows, "│ "+line+strings.Repeat(" ", boxW-4-runeLen(line))+" │")
		}
		rows = append(rows, "└"+strings.Repeat("─", boxW-2)+"┘")
	}
	if rows == nil {
		return []string{"(no tooltips yet)", "", "t on a box adds one"}
	}
	return rows
}

func (m model) sidebarInnerH() int {
	h := max(m.height-1-m.bufferBarOffset(), 1)
	if m.minimap {
		h -= min(minimapH, h)
	}
	return max(h-2, 1)
}

func (m *model) scrollTooltipSidebar(delta int) {
	rows := len(m.tooltipSidebarRows(min(sidebarW, max(m.width, 1)) - 4))
	m.tooltipScroll = max(0, min(m.tooltipScroll+delta, max(0, rows-m.sidebarInnerH())))
}

func (m model) overlayTooltipSidebar(r *RenderResult) {
	w := min(sidebarW, r.Width)
	top := 0
	if m.minimap {
		top = min(minimapH, r.Height)
	}
	h := r.Height - top
	if w < 12 || h < 3 {
		return
	}
	innerW, innerH := w-2, h-2
	originX := r.Width - w

	cell := func(px, py int, ch rune, color int) {
		if py < 0 || py >= len(r.Canvas) || px < 0 || px >= len(r.Canvas[py]) {
			return
		}
		r.Canvas[py][px] = ch
		r.ColorMap[py][px] = color
	}
	hline := func(py int, left, right rune) {
		cell(originX, py, left, colorTooltipBorder)
		for i := 0; i < innerW; i++ {
			cell(originX+1+i, py, '─', colorTooltipBorder)
		}
		cell(originX+w-1, py, right, colorTooltipBorder)
	}

	rows := m.tooltipSidebarRows(innerW - 2)
	scroll := min(m.tooltipScroll, max(0, len(rows)-innerH))

	hline(top, '┌', '┐')
	hline(top+h-1, '└', '┘')
	for i := 0; i < innerH; i++ {
		py := top + 1 + i
		cell(originX, py, '│', colorTooltipBorder)
		row := ""
		if idx := scroll + i; idx < len(rows) {
			row = rows[idx]
		}
		for j := 0; j < innerW; j++ {
			cell(originX+1+j, py, ' ', colorTooltipText)
		}
		for j, ch := range []rune(row) {
			if j+1 < innerW {
				color := colorTooltipText
				if (ch == '[' || ch == ']') && strings.HasPrefix(row, "Box [") {
					color = colorTooltipBracket
				}
				cell(originX+2+j, py, ch, color)
			}
		}
		cell(originX+w-1, py, '│', colorTooltipBorder)
	}
	if len(rows) > innerH {
		thumb := max(1, innerH*innerH/len(rows))
		start := scroll * (innerH - thumb) / (len(rows) - innerH)
		for i := 0; i < thumb; i++ {
			cell(originX+w-1, top+1+start+i, '█', colorTooltipBorder)
		}
	}
	for i, ch := range sidebarLabel {
		cell(originX+w-len(sidebarLabel)-1+i, top, ch, colorTooltipBorder)
	}
}

func (m model) overlayTooltipOnRenderResult(r *RenderResult) {
	charHighlights := map[int]int{}
	if m.tooltipBoxID >= 0 {
		charHighlights = m.getCanvas().GetBoxContentHighlights(m.tooltipBoxID)
	}

	type word struct {
		text     string
		startIdx int
		line     int
	}
	var words []word
	text := m.tooltipText
	start, line := -1, 0
	for i, ch := range text + " " {
		if strings.ContainsRune(" \t\n\r", ch) {
			if start >= 0 {
				words = append(words, word{text[start:i], start, line})
				start = -1
			}
			if ch == '\n' {
				line++
			}
		} else if start < 0 {
			start = i
		}
	}
	if len(words) == 0 {
		return
	}

	plain := func(s string) []tooltipChar {
		out := make([]tooltipChar, 0, len(s))
		for _, ch := range s {
			out = append(out, tooltipChar{ch, -1})
		}
		return out
	}
	var contentLines [][]tooltipChar
	var cur []tooltipChar
	curLine := 0
	for _, w := range words {
		for curLine < w.line {
			contentLines = append(contentLines, cur)
			cur, curLine = nil, curLine+1
		}
		wordLen := runeLen(w.text)
		if len(cur) > 0 && len(cur)+wordLen+1 > tooltipWrapLimit {
			contentLines = append(contentLines, cur)
			cur = nil
		}
		if len(cur) > 0 {
			cur = append(cur, tooltipChar{' ', -1})
		}
		for i, ch := range w.text {
			cur = append(cur, tooltipChar{ch, w.startIdx + i})
		}
	}
	contentLines = append(contentLines, cur)

	contentWidth := tooltipMinContent
	for _, lc := range contentLines {
		contentWidth = max(contentWidth, len(lc))
	}
	tooltipWidth := contentWidth + 4

	border := func(left, right string) []tooltipChar {
		return plain(left + strings.Repeat("─", tooltipWidth-2) + right)
	}

	lines := [][]tooltipChar{border("┌", "┐")}
	for _, lc := range contentLines {
		line := append(plain("│ "), lc...)
		line = append(line, plain(strings.Repeat(" ", contentWidth-len(lc))+" │")...)
		lines = append(lines, line)
	}
	lines = append(lines, border("└", "┘"))

	tooltipX := m.tooltipX + 6
	if tooltipX+tooltipWidth >= r.Width {
		tooltipX = max(m.tooltipX-tooltipWidth-3, 1)
	}
	tooltipY := m.tooltipY
	if tooltipY+len(lines) >= r.Height {
		tooltipY = max(m.tooltipY-len(lines)-1, 0)
	}

	for i, line := range lines {
		y := tooltipY + i
		if y < 0 || y >= r.Height {
			continue
		}
		for j, c := range line {
			x := tooltipX + j
			if x < 0 || x >= r.Width {
				continue
			}
			r.Canvas[y][x] = c.char
			r.ColorMap[y][x] = -1
			color, highlighted := charHighlights[c.origCharIdx]
			switch {
			case m.tooltipStyled && (i == 0 || i == len(lines)-1 || j == 0 || j == len(line)-1):
				r.ColorMap[y][x] = colorTooltipBorder
			case m.tooltipStyled:
				r.ColorMap[y][x] = colorTooltipText
			case highlighted && c.origCharIdx >= 0:
				r.ColorMap[y][x] = color
			}
		}
	}
}

func frameCell(relX, relY, w, h int) (string, bool) {
	switch {
	case relY == 0 || relY == h-1:
		corners := "┌┐"
		if relY != 0 {
			corners = "└┘"
		}
		switch relX {
		case 0:
			return string([]rune(corners)[0]), true
		case w - 1:
			return string([]rune(corners)[1]), true
		}
		return "─", true
	case relX == 0 || relX == w-1:
		return "│", true
	}
	return "", false
}

type startupLayout struct {
	logo      []string
	logoInk   map[point]string
	logoBase  string
	menuItems []string
	logoW     int
	contentW  int
	boxW      int
	boxH      int
	boxX      int
	boxY      int
}

func (l startupLayout) logoOrigin() (int, int) {
	return l.boxX + 1 + (l.contentW-l.logoW)/2, l.boxY + 2
}

func (m model) startupLayout() startupLayout {
	l := startupLayout{
		logo: []string{
			"    ___ __                      ",
			"  .'  _|  |.-----.----.--------.",
			"  |   _|  ||  -__|   _|        |",
			"  |__| |__||_____|__| |__|__|__|",
		},
		menuItems: []string{
			"  n: New",
			"  o: Open",
		},
		logoBase: logoGreen,
	}
	switch m.effect {
	case effectSnow:
		l.logo, l.logoInk = christmasLogo, christmasLogoInk
	case effectBlood:
		l.logo, l.logoInk = halloweenLogo, halloweenLogoInk
	case effectFireworks:
		l.logo, l.logoInk = newYearLogo, newYearLogoInk
	case effectFireworksUSA:
		l.logo, l.logoInk, l.logoBase = julyFourthLogo, julyFourthLogoInk, ansiBlue
	}
	if !cv.ColorEnabled() {
		l.logoInk, l.logoBase = nil, ""
	}
	if m.lastFile != "" {
		name := chartDisplayName(filepath.Base(m.lastFile))
		if r := []rune(name); len(r) > 24 {
			name = string(r[:23]) + "…"
		}
		l.menuItems = append(l.menuItems, "  r: Resume "+name)
	}
	l.menuItems = append(l.menuItems, "  q: Quit")

	l.logoW = len(l.logo[0])
	menuWidth := 0
	for _, item := range l.menuItems {
		menuWidth = max(menuWidth, runeLen(item))
	}
	l.contentW = max(l.logoW, menuWidth)
	l.boxW = l.contentW + 4
	l.boxH = len(l.logo) + len(l.menuItems) + 6
	l.boxX = m.width/2 - l.boxW/2
	l.boxY = m.height/2 - l.boxH/2
	return l
}

func (m model) renderStartupMenu() string {
	l := m.startupLayout()
	particles := m.particleOverlay()
	bleeds := m.effect == effectBlood

	var result strings.Builder
	for y := 0; y < m.height; y++ {
		for x := 0; x < m.width; x++ {
			cell := " "
			relX, relY := x-l.boxX, y-l.boxY
			inBox := relX >= 0 && relX < l.boxW && relY >= 0 && relY < l.boxH
			inside := !inBox
			if inBox {
				var isFrame bool
				cell, isFrame = frameCell(relX, relY, l.boxW, l.boxH)
				switch {
				case isFrame:
				case relY >= 2 && relY < 2+len(l.logo):
					line := l.logo[relY-2]
					logoX := relX - 1 - (l.contentW-l.logoW)/2
					cell = " "
					if logoX >= 0 && logoX < len(line) {
						ink := l.logoBase
						if c, ok := l.logoInk[point{X: logoX, Y: relY - 2}]; ok {
							ink = c
						}
						cell = ink + string(line[logoX]) + cv.Ansi(ansiReset)
					}
				case relY >= 4+len(l.logo) && relY < 4+len(l.logo)+len(l.menuItems):
					item := []rune(l.menuItems[relY-4-len(l.logo)])
					cell = " "
					if menuX := relX - 1; menuX >= 0 && menuX < len(item) {
						cell = string(item[menuX])
					} else {
						inside = true
					}
				default:
					cell = " "
					inside = true
				}
			}
			if inside && (!inBox || bleeds) {
				if glyph, ok := particles[point{X: x, Y: y}]; ok {
					cell = glyph
				}
			}
			result.WriteString(cell)
		}
		if y < m.height-1 {
			result.WriteString("\n")
		}
	}
	return result.String()
}

const (
	fileOpenHint    = "  Enter: Open  d: Delete  ?: Search  Esc: Cancel"
	fileSearchHint  = "  Enter: Open  Esc: Exit search"
	fileEmptyHint   = "  Esc: Cancel"
	fileEmptyList   = "  (No charts found in current directory)"
	fileNoMatch     = "  (No charts match)"
	fileSearchLabel = "  Search: "

	fileConfirmPrefix  = "  Are you sure you want to delete "
	fileConfirmSuffix  = "? (Y/N)"
	fileMenuMinContent = 62
)

func (m model) fileMenuTitle() string {
	n := len(m.allFiles)
	if n == 0 {
		return "Select a saved chart:"
	}
	if total := len(m.fileList); total != n {
		return fmt.Sprintf("Select a saved chart (%d of %d):", total, n)
	}
	return fmt.Sprintf("Select a saved chart (%d):", n)
}

func (m model) fileConfirmLine(width int) string {
	if !m.showingDeleteConfirm || m.confirmFileIndex < 0 || m.confirmFileIndex >= len(m.fileList) {
		return ""
	}
	name := []rune(chartDisplayName(m.fileList[m.confirmFileIndex]))
	fits := max(width-len(fileConfirmPrefix)-len(fileConfirmSuffix), 1)
	if len(name) > fits {
		name = append(name[:fits-1], '\u2026')
	}
	return fileConfirmPrefix + string(name) + fileConfirmSuffix
}

func (m model) fileMenuBounds() (x, y, w, rows int) {
	rows = m.fileListRows()

	contentWidth := runeLen(m.fileMenuTitle())
	if n := len(m.allFiles); n > 0 {
		contentWidth = len(fmt.Sprintf("Select a saved chart (%d of %d):", n, n))
	}
	for _, f := range m.allFiles {
		contentWidth = max(contentWidth, runeLen(chartDisplayName(f))+2)
	}
	for _, s := range []string{fileOpenHint, fileSearchHint, fileEmptyList} {
		contentWidth = max(contentWidth, runeLen(s))
	}
	contentWidth = max(contentWidth, fileMenuMinContent)

	w = contentWidth + 4
	return m.width/2 - w/2, m.height/2 - (rows+6)/2, w, rows
}

func (m model) renderFileMenu() string {
	title := m.fileMenuTitle()
	boxX, boxY, boxWidth, listRows := m.fileMenuBounds()
	contentWidth := boxWidth - 4
	boxHeight := listRows + 6

	total := len(m.fileList)
	var menuItems []string
	switch {
	case len(m.allFiles) == 0:
		menuItems = append(menuItems, fileEmptyList)
	case total == 0:
		menuItems = append(menuItems, fileNoMatch)
	default:
		for i := m.fileScroll; i < min(m.fileScroll+listRows, total); i++ {
			row := "  "
			if i == m.selectedFileIndex {
				row = "> "
			}
			menuItems = append(menuItems, row+chartDisplayName(m.fileList[i]))
		}
	}
	for len(menuItems) < listRows {
		menuItems = append(menuItems, "")
	}

	if m.fileSearch {
		filter := []rune(m.fileFilter)
		if fits := contentWidth - len(fileSearchLabel) - 1; len(filter) > fits && fits > 0 {
			filter = filter[len(filter)-fits:]
		}
		menuItems = append(menuItems, fileSearchLabel+string(filter)+"█")
	} else {
		menuItems = append(menuItems, "")
	}

	switch {
	case m.fileConfirmLine(contentWidth) != "":
		menuItems = append(menuItems, m.fileConfirmLine(contentWidth))
	case m.fileSearch:
		menuItems = append(menuItems, fileSearchHint)
	case len(m.allFiles) == 0:
		menuItems = append(menuItems, fileEmptyHint)
	default:
		menuItems = append(menuItems, fileOpenHint)
	}

	thumbStart, thumbEnd := m.fileScrollThumb()

	var result strings.Builder
	for y := 0; y < m.height; y++ {
		for x := 0; x < m.width; x++ {
			relX, relY := x-boxX, y-boxY
			if relX < 0 || relX >= boxWidth || relY < 0 || relY >= boxHeight {
				result.WriteString(" ")
				continue
			}
			cell, isFrame := frameCell(relX, relY, boxWidth, boxHeight)
			switch {
			case isFrame:
			case relY == 1:
				cell = " "
				if titleX := relX - 1 - (contentWidth-len(title))/2; titleX >= 0 && titleX < len(title) {
					cell = string(title[titleX])
				}
			case relY == 2:
				cell = "─"
			case thumbEnd > 0 && relX == boxWidth-2 && relY >= 3 && relY < 3+listRows:
				cell = "░"
				if i := relY - 3; i >= thumbStart && i < thumbEnd {
					cell = "█"
				}
			case relY >= 3 && relY < 3+len(menuItems):
				item := []rune(menuItems[relY-3])
				cell = " "
				if itemX := relX - 1; itemX >= 0 && itemX < len(item) {
					cell = string(item[itemX])
				}
			default:
				cell = " "
			}
			result.WriteString(cell)
		}
		if y < m.height-1 {
			result.WriteString("\n")
		}
	}
	return result.String()
}

func (m model) helpView() string {
	visible := m.helpPageHeight()
	start := min(m.helpScroll, max(len(helpText)-visible, 0))
	end := min(start+visible, len(helpText))
	return strings.Join(helpText[start:end], "\n") +
		fmt.Sprintf("\nHelp (%d-%d of %d lines) | j/k or PgUp/PgDn to scroll, Esc or q to close", start+1, end, len(helpText))
}
