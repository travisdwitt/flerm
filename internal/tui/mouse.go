package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m *model) bufferBarOffset() int {
	if m.mode != ModeStartup && len(m.buffers) > 1 {
		return 1
	}
	return 0
}

func (m *model) mouseCanvasPos(msg tea.MouseMsg) (int, int) {
	return msg.X, max(msg.Y-m.bufferBarOffset(), 0)
}

func (m model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.mode {
	case ModeContextMenu:
		cmd = m.handleMenuMouse(msg)
	case ModeNormal:
		cmd = m.handleNormalMouse(msg)
	case ModeMultiSelect:
		m.handleMultiSelectMouse(msg)
	case ModeMove:
		m.handleMoveMouse(msg)
	case ModeEditing, ModeTextInput, ModeTitleEdit, ModeTooltipEdit:
		return m.handleTextMouse(msg)
	case ModeFileInput:
		return m.handleFileMouse(msg)
	}
	return m, cmd
}

func (m model) handleTextMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if msg.Action == tea.MouseActionPress {
		switch msg.Button {
		case tea.MouseButtonRight:
			if m.mode == ModeEditing && m.hasEditSelection() {
				return m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
			}
			return m, nil
		case tea.MouseButtonMiddle:
			return m.Update(tea.KeyMsg{Type: tea.KeyCtrlV})
		}
	}

	pos, ok := m.editPosAt(msg.X, msg.Y)
	if !ok {
		return m, nil
	}
	anchor := false
	switch {
	case msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft:
		anchor = true
	case msg.Action == tea.MouseActionMotion && msg.Button == tea.MouseButtonLeft,
		msg.Action == tea.MouseActionRelease:
	default:
		return m, nil
	}
	m.editCursorPos = pos

	if m.mode == ModeEditing {
		if anchor {
			m.editSelectionStart = pos
		}
		m.editSelectionEnd = pos
	}
	return m, nil
}

func (m model) handleFileMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.fileOp != FileOpOpen {
		return m, nil
	}
	if msg.Action == tea.MouseActionRelease {
		m.draggingFileScroll = false
		return m, nil
	}

	x, y, w, rows := m.fileMenuBounds()
	row := msg.Y - (y + 3)

	if m.draggingFileScroll {
		if msg.Action == tea.MouseActionMotion {
			m.dragFileScrollTo(row - m.fileThumbLen()/2)
		}
		return m, nil
	}

	switch msg.Button {
	case tea.MouseButtonWheelUp:
		m.scrollFileList(-3)
		return m, nil
	case tea.MouseButtonWheelDown:
		m.scrollFileList(3)
		return m, nil
	}

	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft || m.showingDeleteConfirm {
		return m, nil
	}
	if row < 0 || row >= rows || msg.X <= x || msg.X >= x+w-1 {
		return m, nil
	}

	if m.fileThumbLen() > 0 && msg.X == x+w-2 {
		m.draggingFileScroll = true
		m.dragFileScrollTo(row - m.fileThumbLen()/2)
		return m, nil
	}

	idx := m.fileScroll + row
	if idx >= len(m.fileList) {
		return m, nil
	}
	reopen := idx == m.selectedFileIndex
	m.selectFileIndex(idx)
	if reopen {
		return m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	}
	return m, nil
}

func (m *model) handleMoveMouse(msg tea.MouseMsg) {
	canvasX, canvasY := m.mouseCanvasPos(msg)
	switch {
	case msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft:
		m.draggingGroup = true
		m.groupLastX, m.groupLastY = canvasX, canvasY
	case msg.Action == tea.MouseActionMotion && m.draggingGroup:
		if dx, dy := canvasX-m.groupLastX, canvasY-m.groupLastY; dx != 0 || dy != 0 {
			m.handleMultiSelectMove(dx, dy)
			m.groupLastX, m.groupLastY = canvasX, canvasY
		}
	case msg.Action == tea.MouseActionRelease:
		m.draggingGroup = false
		m.commitMove()
	}
}

func (m *model) handleMultiSelectMouse(msg tea.MouseMsg) {
	canvasX, canvasY := m.mouseCanvasPos(msg)
	panX, panY := m.getPanOffset()
	if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonRight {
		m.mode = ModeNormal
		m.selectionStartX, m.selectionStartY = CoordUnset, CoordUnset
		m.selectedBoxes, m.selectedTexts = nil, nil
		return
	}
	m.cursorX, m.cursorY = canvasX, canvasY
	m.ensureCursorInBounds()
	switch {
	case msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft:
		m.selectionStartX, m.selectionStartY = canvasX+panX, canvasY+panY
	case msg.Action == tea.MouseActionRelease:
		m.finalizeMultiSelect(canvasX+panX, canvasY+panY)
	}
}

func (m *model) handleNormalMouse(msg tea.MouseMsg) tea.Cmd {
	if tea.MouseEvent(msg).IsWheel() {
		_, wheelY := m.mouseCanvasPos(msg)
		inSidebar := m.allTooltips && msg.X >= max(m.width, 1)-min(sidebarW, max(m.width, 1)) && !(m.minimap && wheelY < minimapH)
		buf := m.getCurrentBuffer()
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			if inSidebar {
				m.scrollTooltipSidebar(-3)
				return nil
			}
			buf.panY -= 2
		case tea.MouseButtonWheelDown:
			if inSidebar {
				m.scrollTooltipSidebar(3)
				return nil
			}
			buf.panY += 2
		case tea.MouseButtonWheelLeft:
			buf.panX -= 2
		case tea.MouseButtonWheelRight:
			buf.panX += 2
		}
		return nil
	}

	canvasX, canvasY := m.mouseCanvasPos(msg)
	panX, panY := m.getPanOffset()
	worldX, worldY := canvasX+panX, canvasY+panY

	if m.draggingBox {
		switch msg.Action {
		case tea.MouseActionMotion:
			m.dragMoveTo(canvasX, canvasY)
			return nil
		case tea.MouseActionRelease:
			m.finishBoxDrag()
			return nil
		default:
			m.finishBoxDrag()
		}
	}

	if m.draggingText {
		switch msg.Action {
		case tea.MouseActionMotion:
			m.dragTextMoveTo(canvasX, canvasY)
			return nil
		case tea.MouseActionRelease:
			m.finishTextDrag()
			return nil
		default:
			m.finishTextDrag()
		}
	}

	if m.panningView {
		switch msg.Action {
		case tea.MouseActionMotion:
			m.panViewTo(canvasX, canvasY)
			return nil
		case tea.MouseActionRelease:
			if !m.panMoved {
				m.selectAtMouse(worldX, worldY)
			}
			m.panningView = false
			return nil
		default:
			m.panningView = false
		}
	}

	if m.mouseLineDrawing && (m.connectionFrom != -1 || m.connectionFromLine != -1) {
		m.cursorX, m.cursorY = canvasX, canvasY
		m.ensureCursorInBounds()
		if msg.Action == tea.MouseActionPress {
			switch msg.Button {
			case tea.MouseButtonLeft:
				m.extendLine(m.worldCursor())
			case tea.MouseButtonRight:
				m.cancelMouseLine()
			}
		}
		return nil
	}

	if m.highlightMode {
		switch {
		case msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft:
			m.beginHighlightPaint(worldX, worldY, m.selectedColor)
			return nil
		case msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonRight:
			m.beginHighlightPaint(worldX, worldY, -1)
			return nil
		case msg.Action == tea.MouseActionMotion && m.paintingHighlight:
			m.paintHighlightTo(worldX, worldY)
			return nil
		case msg.Action == tea.MouseActionRelease && m.paintingHighlight:
			m.finishHighlightPaint()
			return nil
		}
	}

	m.hoverBoxTooltip(worldX, worldY, canvasX, canvasY)
	if msg.Action != tea.MouseActionPress {
		return nil
	}

	switch msg.Button {
	case tea.MouseButtonLeft:
		m.cursorX, m.cursorY = canvasX, canvasY
		m.ensureCursorInBounds()
		canvas := m.getCanvas()
		if boxID := canvas.GetBoxAt(worldX, worldY); boxID != -1 {
			m.beginBoxDrag(boxID, worldX, worldY)
			return nil
		}
		if textID := canvas.GetTextAt(worldX, worldY); textID != -1 {
			m.beginTextDrag(textID, worldX, worldY)
			return nil
		}
		m.panningView = true
		m.panLastX, m.panLastY = canvasX, canvasY
		m.panMoved = false
	case tea.MouseButtonRight:
		m.openContextMenu(canvasX, canvasY)
	}
	return nil
}

func (m *model) beginBoxDrag(boxID, worldX, worldY int) {
	canvas := m.getCanvas()
	box := canvas.Boxes()[boxID]
	m.draggingBox = true
	m.dragBoxID = boxID
	m.dragGrabOffsetX, m.dragGrabOffsetY = box.X-worldX, box.Y-worldY
	m.captureBoxMove(boxID)
	m.dragConnSnapshot = canvas.SnapshotConnections()
	m.selBox, m.selText, m.selConn = boxID, -1, -1
}

func (m *model) dragMoveTo(canvasX, canvasY int) {
	canvas := m.getCanvas()
	if m.dragBoxID < 0 || m.dragBoxID >= len(canvas.Boxes()) {
		m.draggingBox = false
		return
	}
	panX, panY := m.getPanOffset()
	desiredX := canvasX + panX + m.dragGrabOffsetX
	desiredY := canvasY + panY + m.dragGrabOffsetY

	canvas.RestoreConnectionsSnapshot(m.dragConnSnapshot)
	canvas.SetBoxPositionOnly(m.dragBoxID, m.originalMoveX, m.originalMoveY)
	canvas.MoveBox(m.dragBoxID, desiredX-m.originalMoveX, desiredY-m.originalMoveY)
	box := canvas.Boxes()[m.dragBoxID]
	m.moveCapturedHighlights(box.X-m.originalMoveX, box.Y-m.originalMoveY)

	m.cursorX, m.cursorY = canvasX, canvasY
	m.ensureCursorInBounds()
}

func (m *model) finishBoxDrag() {
	m.recordBoxMove(m.dragBoxID)
	m.draggingBox = false
	m.dragConnSnapshot = nil
	m.clearGroupSelection()
}

func (m *model) beginTextDrag(textID, worldX, worldY int) {
	text := m.getCanvas().Texts()[textID]
	m.draggingText = true
	m.dragTextID = textID
	m.dragGrabOffsetX, m.dragGrabOffsetY = text.X-worldX, text.Y-worldY
	m.captureTextMove(textID)
	m.selBox, m.selText, m.selConn = -1, textID, -1
}

func (m *model) dragTextMoveTo(canvasX, canvasY int) {
	canvas := m.getCanvas()
	if m.dragTextID < 0 || m.dragTextID >= len(canvas.Texts()) {
		m.draggingText = false
		return
	}
	panX, panY := m.getPanOffset()
	cur := canvas.Texts()[m.dragTextID]
	canvas.MoveText(m.dragTextID, canvasX+panX+m.dragGrabOffsetX-cur.X, canvasY+panY+m.dragGrabOffsetY-cur.Y)
	moved := canvas.Texts()[m.dragTextID]
	m.moveCapturedHighlights(moved.X-m.originalMoveX, moved.Y-m.originalMoveY)
	m.cursorX, m.cursorY = canvasX, canvasY
	m.ensureCursorInBounds()
}

func (m *model) finishTextDrag() {
	m.recordTextMove(m.dragTextID)
	m.draggingText = false
	m.clearGroupSelection()
}

func (m *model) panViewTo(canvasX, canvasY int) {
	buf := m.getCurrentBuffer()
	buf.panX -= canvasX - m.panLastX
	buf.panY -= canvasY - m.panLastY
	m.panLastX, m.panLastY = canvasX, canvasY
	m.panMoved = true
}

func (m *model) beginHighlightPaint(worldX, worldY, color int) {
	m.paintingHighlight = true
	m.paintColor = color
	m.paintedCells = nil
	m.paintedSeen = map[point]bool{}
	m.lastPaintX, m.lastPaintY = worldX, worldY
	m.paintHighlightCell(worldX, worldY)
}

func (m *model) paintHighlightCell(x, y int) {
	p := point{X: x, Y: y}
	if x < 0 || y < 0 || m.paintedSeen[p] {
		return
	}
	m.paintedSeen[p] = true
	m.paintedCells = append(m.paintedCells, m.paintCell(x, y, m.paintColor))
}

func (m *model) paintHighlightTo(worldX, worldY int) {
	x, y := m.lastPaintX, m.lastPaintY
	dx, dy := abs(worldX-x), -abs(worldY-y)
	sx, sy := sign(worldX-x), sign(worldY-y)
	err := dx + dy
	for {
		m.paintHighlightCell(x, y)
		if x == worldX && y == worldY {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x += sx
		}
		if e2 <= dx {
			err += dx
			y += sy
		}
	}
	m.lastPaintX, m.lastPaintY = worldX, worldY
}

func (m *model) finishHighlightPaint() {
	m.recordHighlights(m.paintedCells)
	m.paintingHighlight = false
	m.paintedCells = nil
	m.paintedSeen = nil
}

func (m *model) selectAtMouse(worldX, worldY int) {
	canvas := m.getCanvas()
	m.selBox, m.selText, m.selConn = -1, -1, -1
	if boxID := canvas.GetBoxAt(worldX, worldY); boxID != -1 {
		m.selBox = boxID
	} else if textID := canvas.GetTextAt(worldX, worldY); textID != -1 {
		m.selText = textID
	} else if connIdx, _, _ := canvas.FindNearestPointOnConnection(worldX, worldY); connIdx != -1 {
		m.selConn = connIdx
	}
}

func (m *model) openContextMenu(canvasX, canvasY int) {
	canvas := m.getCanvas()
	panX, panY := m.getPanOffset()
	worldX, worldY := canvasX+panX, canvasY+panY

	m.menuWorldX, m.menuWorldY = worldX, worldY
	m.menuTargetBox, m.menuTargetText, m.menuTargetConn = -1, -1, -1

	m.menuTargetBox = canvas.GetBoxAt(worldX, worldY)
	if m.menuTargetBox == -1 {
		m.menuTargetText = canvas.GetTextAt(worldX, worldY)
		if m.menuTargetText == -1 {
			m.menuTargetConn, _, _ = canvas.FindNearestPointOnConnection(worldX, worldY)
		}
	}

	m.menuItems = buildMenuItems(m.menuTargetBox, m.menuTargetText, m.menuTargetConn)
	m.menuIndex = firstSelectableMenuIndex(m.menuItems)
	m.menuStack = nil
	m.menuX, m.menuY = canvasX, canvasY
	m.mode = ModeContextMenu
}

func colorSubmenu() []MenuItem {
	items := []MenuItem{{Label: "None", Action: MenuSetColor, Arg: -1}}
	for i, n := range colorNames {
		items = append(items, MenuItem{Label: "■ " + n, Action: MenuSetColor, Arg: i})
	}
	return items
}

func borderStyleSubmenu() []MenuItem {
	return []MenuItem{
		{Label: "ASCII", Action: MenuSetBorderStyle, Arg: int(BorderStyleASCII)},
		{Label: "Single", Action: MenuSetBorderStyle, Arg: int(BorderStyleSingle)},
		{Label: "Double", Action: MenuSetBorderStyle, Arg: int(BorderStyleDouble)},
		{Label: "Rounded", Action: MenuSetBorderStyle, Arg: int(BorderStyleRounded)},
	}
}

func buildMenuItems(box, text, conn int) []MenuItem {
	var items []MenuItem
	switch {
	case box != -1:
		items = append(items,
			MenuItem{Label: "Edit Box", Action: MenuEditBox},
			MenuItem{Label: "Edit Title", Action: MenuEditTitle},
			MenuItem{Label: "Tooltip", Action: MenuEditTooltip},
			MenuItem{Label: "Border", Action: MenuSubmenu, Submenu: []MenuItem{
				{Label: "Style", Action: MenuSubmenu, Submenu: borderStyleSubmenu()},
				{Label: "Color", Action: MenuSubmenu, Submenu: colorSubmenu()},
			}},
			MenuItem{Label: "New Line", Action: MenuNewLine},
			MenuItem{Label: "Duplicate Box", Action: MenuDuplicate},
			MenuItem{Label: "Delete Box", Action: MenuDeleteBox},
			MenuItem{Separator: true},
		)
	case text != -1:
		items = append(items,
			MenuItem{Label: "Edit Text", Action: MenuEditText},
			MenuItem{Label: "Color", Action: MenuSubmenu, Submenu: colorSubmenu()},
			MenuItem{Label: "Duplicate Text", Action: MenuDuplicate},
			MenuItem{Label: "Delete Text", Action: MenuDeleteText},
			MenuItem{Separator: true},
		)
	case conn != -1:
		items = append(items,
			MenuItem{Label: "New Line", Action: MenuNewLine},
			MenuItem{Label: "Color", Action: MenuSubmenu, Submenu: colorSubmenu()},
			MenuItem{Label: "Delete Line", Action: MenuDeleteLine},
			MenuItem{Separator: true},
		)
	}
	return append(items,
		MenuItem{Label: "New Box", Action: MenuNewBox},
		MenuItem{Label: "New Text", Action: MenuNewText},
		MenuItem{Label: "Select Area", Action: MenuMultiSelect},
	)
}

func firstSelectableMenuIndex(items []MenuItem) int {
	for i, item := range items {
		if !item.Separator {
			return i
		}
	}
	return 0
}

func (m *model) allMenuLevels() []menuLevel {
	levels := []menuLevel{{items: m.menuItems, index: m.menuIndex, x: m.menuX, y: m.menuY}}
	return append(levels, m.menuStack...)
}

func (m *model) focusedLevel() *menuLevel {
	if n := len(m.menuStack); n > 0 {
		return &m.menuStack[n-1]
	}
	return nil
}

func (m *model) focusedItems() []MenuItem {
	if l := m.focusedLevel(); l != nil {
		return l.items
	}
	return m.menuItems
}

func (m *model) focusedIndex() int {
	if l := m.focusedLevel(); l != nil {
		return l.index
	}
	return m.menuIndex
}

func (m *model) setFocusedIndex(idx int) {
	if l := m.focusedLevel(); l != nil {
		l.index = idx
		return
	}
	m.menuIndex = idx
}

func (m *model) menuMoveSelection(dir int) {
	items := m.focusedItems()
	n := len(items)
	if n == 0 {
		return
	}
	idx := m.focusedIndex()
	for i := 0; i < n; i++ {
		idx = (idx + dir + n) % n
		if !items[idx].Separator {
			m.setFocusedIndex(idx)
			return
		}
	}
}

func (m *model) menuDescend() {
	items := m.focusedItems()
	idx := m.focusedIndex()
	if idx < 0 || idx >= len(items) || len(items[idx].Submenu) == 0 {
		return
	}
	levels := m.allMenuLevels()
	px, py, pw, _ := m.levelBounds(levels[len(levels)-1])
	m.menuStack = append(m.menuStack, menuLevel{
		items: items[idx].Submenu,
		index: firstSelectableMenuIndex(items[idx].Submenu),
		x:     px + pw,
		y:     py + 1 + idx,
	})
}

func (m *model) menuAscend() {
	if len(m.menuStack) > 0 {
		m.menuStack = m.menuStack[:len(m.menuStack)-1]
		return
	}
	m.closeContextMenu()
}

func (m *model) closeContextMenu() {
	m.mode = ModeNormal
	m.menuItems = nil
	m.menuStack = nil
}

func (m *model) handleMenuMouse(msg tea.MouseMsg) tea.Cmd {
	canvasX, canvasY := msg.X, msg.Y-m.bufferBarOffset()
	levelIdx, itemIdx, inside := m.menuHitTest(canvasX, canvasY)

	switch {
	case msg.Action == tea.MouseActionMotion:
		if inside && itemIdx >= 0 {
			m.focusMenuLevel(levelIdx, itemIdx)
		}
	case msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft:
		if !inside {
			m.closeContextMenu()
		} else if itemIdx >= 0 {
			m.focusMenuLevel(levelIdx, itemIdx)
			item := m.focusedItems()[m.focusedIndex()]
			if len(item.Submenu) > 0 {
				m.menuDescend()
			} else {
				m.activateMenuItem(item.Action, item.Arg)
			}
		}
	case msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonRight:
		m.closeContextMenu()
	}
	return nil
}

func (m *model) focusMenuLevel(levelIdx, itemIdx int) {
	if levelIdx < 0 {
		return
	}
	if levelIdx < len(m.menuStack)+1 {
		m.menuStack = m.menuStack[:levelIdx]
	}
	items := m.menuItems
	if levelIdx > 0 {
		items = m.menuStack[levelIdx-1].items
	}
	if itemIdx < 0 || itemIdx >= len(items) || items[itemIdx].Separator {
		return
	}
	if levelIdx == 0 {
		m.menuIndex = itemIdx
	} else {
		m.menuStack[levelIdx-1].index = itemIdx
	}
	if len(items[itemIdx].Submenu) > 0 {
		m.menuDescend()
	}
}

func (m *model) menuHitTest(canvasX, canvasY int) (int, int, bool) {
	levels := m.allMenuLevels()
	for li := len(levels) - 1; li >= 0; li-- {
		x, y, w, h := m.levelBounds(levels[li])
		if canvasX < x || canvasX >= x+w || canvasY < y || canvasY >= y+h {
			continue
		}
		itemRow := canvasY - (y + 1)
		if itemRow < 0 || itemRow >= len(levels[li].items) || levels[li].items[itemRow].Separator {
			return li, -1, true
		}
		return li, itemRow, true
	}
	return -1, -1, false
}

func (m *model) levelBounds(level menuLevel) (int, int, int, int) {
	w := menuInnerWidth(level.items) + 2
	h := len(level.items) + 2
	maxH := max(m.height-1-m.bufferBarOffset(), 1)
	x := max(min(level.x, m.width-w), 0)
	y := max(min(level.y, maxH-h), 0)
	return x, y, w, h
}

func menuInnerWidth(items []MenuItem) int {
	maxLabel := 0
	hasSubmenu := false
	for _, item := range items {
		if item.Separator {
			continue
		}
		maxLabel = max(maxLabel, runeLen(item.Label))
		hasSubmenu = hasSubmenu || len(item.Submenu) > 0
	}
	inner := maxLabel + 2
	if hasSubmenu {
		inner += 2
	}
	return inner
}

func (m *model) activateMenuItem(action MenuAction, arg int) {
	if action == MenuSubmenu {
		return
	}
	canvas := m.getCanvas()
	box, text, conn := m.menuTargetBox, m.menuTargetText, m.menuTargetConn
	m.closeContextMenu()

	switch action {
	case MenuNewBox:
		m.addBoxRecorded(m.menuWorldX, m.menuWorldY, "Box")
	case MenuNewText:
		m.textInputX, m.textInputY = m.menuWorldX, m.menuWorldY
		m.beginEdit(ModeTextInput, -1, -1, "")
	case MenuEditBox:
		if box >= 0 && box < len(canvas.Boxes()) {
			m.beginEdit(ModeEditing, box, -1, canvas.GetBoxText(box))
		}
	case MenuEditText:
		if text >= 0 && text < len(canvas.Texts()) {
			m.beginEdit(ModeEditing, -1, text, canvas.GetTextText(text))
		}
	case MenuNewLine:
		if m.beginLine(m.menuWorldX, m.menuWorldY) {
			m.mouseLineDrawing = true
			m.cursorX, m.cursorY = m.menuX, m.menuY
			m.ensureCursorInBounds()
		}
	case MenuMultiSelect:
		m.selectionStartX, m.selectionStartY = CoordUnset, CoordUnset
		m.selectedBoxes, m.selectedTexts, m.selectedConnections = nil, nil, nil
		m.cursorX, m.cursorY = m.menuX, m.menuY
		m.ensureCursorInBounds()
		m.mode = ModeMultiSelect
	case MenuDuplicate:
		if box != -1 {
			m.duplicateByID(false, box)
		} else {
			m.duplicateByID(true, text)
		}
	case MenuDeleteBox:
		m.deleteBoxByID(box)
		m.selBox, m.selText, m.selConn = -1, -1, -1
	case MenuDeleteText:
		m.deleteTextByID(text)
		m.selBox, m.selText, m.selConn = -1, -1, -1
	case MenuDeleteLine:
		m.deleteConnByIdx(conn)
		m.selBox, m.selText, m.selConn = -1, -1, -1
	case MenuEditTitle:
		if box >= 0 && box < len(canvas.Boxes()) {
			m.beginEdit(ModeTitleEdit, box, -1, canvas.Boxes()[box].Title)
		}
	case MenuEditTooltip:
		if box >= 0 && box < len(canvas.Boxes()) {
			m.beginTooltipEdit(box, m.menuX, m.menuY)
		}
	case MenuSetBorderStyle:
		if box >= 0 && box < len(canvas.Boxes()) {
			data := BorderStyleData{BoxID: box, OldStyle: canvas.Boxes()[box].BorderStyle, NewStyle: BorderStyle(arg)}
			canvas.SetBorderStyle(box, data.NewStyle)
			m.recordAction(ActionChangeBorderStyle, data, data)
		}
	case MenuSetColor:
		m.applyMenuColor(arg)
	}
}

func (m *model) applyMenuColor(color int) {
	canvas := m.getCanvas()
	var kind, id, old int
	switch {
	case m.menuTargetBox >= 0 && m.menuTargetBox < len(canvas.Boxes()):
		kind, id = ColorKindBox, m.menuTargetBox
		old = canvas.Boxes()[id].Color
	case m.menuTargetConn >= 0 && m.menuTargetConn < len(canvas.Connections()):
		kind, id = ColorKindLine, m.menuTargetConn
		old = canvas.Connections()[id].Color
	case m.menuTargetText >= 0 && m.menuTargetText < len(canvas.Texts()):
		kind, id = ColorKindText, m.menuTargetText
		old = canvas.Texts()[id].Color
	default:
		if color >= 0 {
			m.selectedColor = color
		}
		return
	}
	m.applyObjectColor(kind, id, color)
	if old != color {
		data := ColorData{Kind: kind, ID: id, OldColor: old, NewColor: color}
		m.recordAction(ActionSetColor, data, data)
	}
}

func (m model) overlaySelection(r *RenderResult, panX, panY int) {
	canvas := m.getCanvas()
	var cells, boxCells []point
	switch {
	case m.selBox >= 0 && m.selBox < len(canvas.Boxes()):
		boxCells = canvas.GetBoxBorderCells(m.selBox)
	case m.selText >= 0 && m.selText < len(canvas.Texts()):
		cells = canvas.GetTextCells(m.selText)
	case m.selConn >= 0 && m.selConn < len(canvas.Connections()):
		cells = canvas.GetConnectionCells(m.selConn)
	}
	for _, id := range m.selectedBoxes {
		boxCells = append(boxCells, canvas.GetBoxBorderCells(id)...)
	}
	for _, id := range m.selectedTexts {
		cells = append(cells, canvas.GetTextCells(id)...)
	}
	for _, id := range m.selectedConnections {
		cells = append(cells, canvas.GetConnectionCells(id)...)
	}
	for _, cell := range boxCells {
		markSelected(r, cell, panX, panY, '#')
	}
	for _, cell := range cells {
		markSelected(r, cell, panX, panY, 0)
	}
}

func markSelected(r *RenderResult, cell point, panX, panY int, ch rune) {
	sx, sy := cell.X-panX, cell.Y-panY
	if sy < 0 || sy >= len(r.ColorMap) || sx < 0 || sx >= len(r.ColorMap[sy]) {
		return
	}
	if ch != 0 {
		r.Canvas[sy][sx] = ch
	}
	r.ColorMap[sy][sx] = colorMouseSelect
}

func (m model) overlayContextMenu(r *RenderResult) {
	for _, level := range m.allMenuLevels() {
		m.drawMenuLevel(r, level)
	}
}

func (m model) drawMenuLevel(r *RenderResult, level menuLevel) {
	x, y, w, h := m.levelBounds(level)
	inner := w - 2

	setCell := func(px, py int, ch rune, colorIdx int) {
		if py < 0 || py >= len(r.Canvas) || px < 0 || px >= len(r.Canvas[py]) {
			return
		}
		r.Canvas[py][px] = ch
		r.ColorMap[py][px] = colorIdx
	}
	hline := func(py int, left, right rune) {
		setCell(x, py, left, colorMenuBorder)
		for i := 0; i < inner; i++ {
			setCell(x+1+i, py, '─', colorMenuBorder)
		}
		setCell(x+w-1, py, right, colorMenuBorder)
	}

	hline(y, '┌', '┐')
	hline(y+h-1, '└', '┘')
	for itemIdx, item := range level.items {
		py := y + 1 + itemIdx
		if item.Separator {
			hline(py, '├', '┤')
			continue
		}
		rowColor := -1
		if itemIdx == level.index {
			rowColor = colorMenuSelect
		}
		setCell(x, py, '│', colorMenuBorder)
		label := []rune(" " + item.Label)
		for i := 0; i < inner; i++ {
			ch := ' '
			if i < len(label) {
				ch = label[i]
			}
			setCell(x+1+i, py, ch, rowColor)
		}
		if len(item.Submenu) > 0 {
			setCell(x+w-2, py, '▸', rowColor)
		}
		if item.Action == MenuSetColor && item.Arg >= 0 {
			setCell(x+2, py, '■', item.Arg)
		}
		setCell(x+w-1, py, '│', colorMenuBorder)
	}
}
