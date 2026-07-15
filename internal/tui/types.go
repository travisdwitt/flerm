package tui

type Buffer struct {
	canvas    *Canvas
	undoStack []Action
	redoStack []Action
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
	selectedBox            int
	selectedText           int
	editText               string
	editCursorPos          int
	editCursorRow          int
	editCursorCol          int
	editSelectionStart     int
	editSelectionEnd       int
	originalEditText       string
	connectionFrom         int
	connectionFromX        int
	connectionFromY        int
	connectionFromLine     int
	connectionWaypoints    []point
	filename               string
	fileList               []string
	selectedFileIndex      int
	fileOp                 FileOperation
	openInNewBuffer        bool
	createNewBuffer        bool
	showingDeleteConfirm   bool
	confirmAction          ConfirmAction
	confirmBoxID           int
	confirmTextID          int
	confirmConnIdx         int
	confirmHighlightX      int
	confirmHighlightY      int
	confirmFileIndex       int
	originalMoveX          int
	originalMoveY          int
	originalTextMoveX      int
	originalTextMoveY      int
	originalWidth          int
	originalHeight         int
	textInputX             int
	textInputY             int
	textInputText          string
	textInputCursorPos     int
	errorMessage           string
	successMessage         string
	fromStartup            bool
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
	titleEditBoxID         int
	titleEditText          string
	titleEditCursorPos     int
	titleEditCursorRow     int
	titleEditCursorCol     int
	originalTitleText      string
	showTooltip            bool
	tooltipText            string
	tooltipX               int
	tooltipY               int
	tooltipBoxID           int

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

	panningView        bool
	panLastX, panLastY int
	panMoved           bool

	draggingGroup          bool
	groupLastX, groupLastY int

	paintingHighlight      bool
	paintedCells           []HighlightCell
	paintedSeen            map[point]bool
	lastPaintX, lastPaintY int

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
	Data    interface{}
	Inverse interface{}
}

type AddBoxData struct {
	X, Y int
	Text string
	ID   int
}

type AddTextData struct {
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

type EditBoxData struct {
	ID      int
	NewText string
	OldText string
}

type EditTextData struct {
	ID      int
	NewText string
	OldText string
}

type DeleteTextData struct {
	Text       Text
	ID         int
	Highlights []HighlightCell
}

type ResizeBoxData struct {
	ID          int
	DeltaWidth  int
	DeltaHeight int
}

type MoveBoxData struct {
	ID     int
	DeltaX int
	DeltaY int
}

type MoveTextData struct {
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

type OriginalTextState struct {
	ID int
	X  int
	Y  int
}

type AddConnectionData struct {
	FromID     int
	ToID       int
	Connection Connection
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

type EditTitleData struct {
	BoxID    int
	NewTitle string
	OldTitle string
}

const (
	ColorKindBox = iota
	ColorKindLine
	ColorKindText
)

type ColorData struct {
	Kind     int
	ID       int
	OldColor int
	NewColor int
}
