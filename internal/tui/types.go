package tui

import (
	cv "flerm/internal/canvas"
	"flerm/internal/config"
)

type Buffer struct {
	canvas    *cv.Canvas
	last      *cv.Canvas
	undoStack []*cv.Canvas
	redoStack []*cv.Canvas
	savedAt   int
	filename  string
	panX      int
	panY      int
}

type model struct {
	width                 int
	height                int
	cursorX               int
	cursorY               int
	zPanMode              bool
	buffers               []Buffer
	currentBufferIndex    int
	mode                  Mode
	help                  bool
	helpScroll            int
	minimap               bool
	selectedBox           int
	selectedText          int
	editText              string
	editCursorPos         int
	editSelectionStart    int
	editSelectionEnd      int
	originalEditText      string
	connectionFrom        int
	connectionFromX       int
	connectionFromY       int
	connectionFromLine    int
	connectionWaypoints   []cv.Point
	filename              string
	allFiles              []string
	fileList              []string
	fileFilter            string
	fileSearch            bool
	fileScroll            int
	selectedFileIndex     int
	fileOp                FileOperation
	openInNewBuffer       bool
	showingDeleteConfirm  bool
	confirmAction         ConfirmAction
	confirmID             int
	confirmFileIndex      int
	overwriteOK           bool
	originalMoveX         int
	originalMoveY         int
	originalWidth         int
	originalHeight        int
	textInputX            int
	textInputY            int
	errorMessage          string
	successMessage        string
	fromStartup           bool
	lastFile              string
	clipboard             *cv.Box
	config                *config.Config
	highlightMode         bool
	selectedColor         int
	selectionStartX       int
	selectionStartY       int
	selectedBoxes         []int
	selectedTexts         []int
	selectedConnections   []int
	originalBoxPositions  map[int]cv.Point
	originalTextPositions map[int]cv.Point
	originalConnections   map[int]cv.Connection
	originalHighlights    map[cv.Point]int
	highlightMoveDelta    cv.Point
	boxJumpInput          string
	showTooltip           bool
	tooltipText           string
	tooltipX              int
	tooltipY              int
	tooltipBoxID          int
	tooltipStyled         bool
	allTooltips           bool
	tooltipScroll         int

	selBox  int
	selText int
	selConn int

	mouseLineDrawing bool

	draggingBox      bool
	dragBoxID        int
	dragGrabOffsetX  int
	dragGrabOffsetY  int
	dragConnSnapshot []cv.Connection

	draggingText bool
	dragTextID   int

	draggingFileScroll bool

	panningView        bool
	panLastX, panLastY int
	panMoved           bool

	draggingGroup          bool
	groupLastX, groupLastY int

	paintingHighlight      bool
	paintColor             int
	paintedAny             bool
	paintedSeen            map[cv.Point]bool
	lastPaintX, lastPaintY int

	effect      effectKind
	effectFrame int
	particles   []particle

	menuItems      []MenuItem
	menuIndex      int
	menuX          int
	menuY          int
	menuTargetBox  int
	menuTargetText int
	menuTargetConn int
	menuWorldX     int
	menuWorldY     int
	menuStack      []menuLevel
}

type MenuItem struct {
	Label     string
	Action    MenuAction
	Separator bool
	Submenu   []MenuItem
	Arg       int
}

type menuLevel struct {
	items []MenuItem
	index int
	x, y  int
}
