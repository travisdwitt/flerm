package tui

type Buffer struct {
	canvas    *Canvas
	undoStack []Action
	redoStack []Action
	savedAt   int
	filename  string
	panX      int
	panY      int
}

type model struct {
	width                  int
	height                 int
	cursorX                int
	cursorY                int
	zPanMode               bool
	buffers                []Buffer
	currentBufferIndex     int
	mode                   Mode
	help                   bool
	helpScroll             int
	minimap                bool
	selectedBox            int
	selectedText           int
	editText               string
	editCursorPos          int
	editSelectionStart     int
	editSelectionEnd       int
	originalEditText       string
	connectionFrom         int
	connectionFromX        int
	connectionFromY        int
	connectionFromLine     int
	connectionWaypoints    []point
	filename               string
	allFiles               []string
	fileList               []string
	fileFilter             string
	fileSearch             bool
	fileScroll             int
	selectedFileIndex      int
	fileOp                 FileOperation
	openInNewBuffer        bool
	showingDeleteConfirm   bool
	confirmAction          ConfirmAction
	confirmID              int
	confirmFileIndex       int
	originalMoveX          int
	originalMoveY          int
	originalWidth          int
	originalHeight         int
	textInputX             int
	textInputY             int
	errorMessage           string
	successMessage         string
	fromStartup            bool
	lastFile               string
	clipboard              *Box
	config                 *Config
	highlightMode          bool
	selectedColor          int
	selectionStartX        int
	selectionStartY        int
	selectedBoxes          []int
	selectedTexts          []int
	selectedConnections    []int
	originalBoxPositions   map[int]point
	originalTextPositions  map[int]point
	originalConnections    map[int]Connection
	originalHighlights     map[point]int
	highlightMoveDelta     point
	originalBoxConnections map[int][]Connection
	boxJumpInput           string
	showTooltip            bool
	tooltipText            string
	tooltipX               int
	tooltipY               int
	tooltipBoxID           int
	tooltipStyled          bool
	allTooltips            bool
	tooltipScroll          int

	selBox  int
	selText int
	selConn int

	mouseLineDrawing bool

	draggingBox      bool
	dragBoxID        int
	dragGrabOffsetX  int
	dragGrabOffsetY  int
	dragConnSnapshot []Connection

	draggingText bool
	dragTextID   int

	draggingFileScroll bool

	panningView        bool
	panLastX, panLastY int
	panMoved           bool

	draggingGroup          bool
	groupLastX, groupLastY int
	groupConnSnapshot      []Connection

	paintingHighlight      bool
	paintColor             int
	paintedCells           []HighlightCell
	paintedSeen            map[point]bool
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

type Action struct {
	Type    ActionType
	Data    any
	Inverse any
}

type AddData struct {
	X, Y int
	Text string
	ID   int
}

type DeleteBoxData struct {
	Box         Box
	ID          int
	Connections []Connection
	Highlights  []HighlightCell
}

type DeleteTextData struct {
	Text       Text
	ID         int
	Highlights []HighlightCell
}

type EditData struct {
	ID      int
	NewText string
	OldText string
}

type ResizeBoxData struct {
	ID          int
	DeltaWidth  int
	DeltaHeight int
}

type MoveData struct {
	ID     int
	DeltaX int
	DeltaY int
}

type OriginalBoxState struct {
	ID          int
	X           int
	Y           int
	Width       int
	Height      int
	Connections []Connection
	Highlights  []HighlightCell
}

type GroupMoveState struct {
	BoxPositions  map[int]point
	TextPositions map[int]point
	Connections   []Connection
	Highlights    []HighlightCell
}

type GroupMoveData struct {
	Before GroupMoveState
	After  GroupMoveState
}

type CycleArrowData struct {
	ConnIdx int
	OldConn Connection
	NewConn Connection
}

type HighlightData struct {
	Cells []HighlightCell
}

type BorderStyleData struct {
	BoxID    int
	OldStyle BorderStyle
	NewStyle BorderStyle
}

const (
	ColorKindBox = iota
	ColorKindLine
	ColorKindText
)

type DuplicateData struct {
	IsText bool
	SrcID  int
	NewID  int
	DX, DY int
}

type ColorData struct {
	Kind     int
	ID       int
	OldColor int
	NewColor int
}
