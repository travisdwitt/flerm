package main

type Buffer struct {
	canvas    *Canvas
	undoStack []Action
	redoStack []Action
	filename  string
	panX      int
	panY      int
}

type model struct {
	width               int
	height              int
	cursorX             int
	cursorY             int
	zPanMode            bool
	buffers             []Buffer
	currentBufferIndex  int
	mode                Mode
	help                bool
	helpScroll          int
	selectedBox         int
	selectedText        int
	editText            string
	editCursorPos       int
	originalEditText    string
	connectionFrom      int
	connectionFromX     int
	connectionFromY     int
	connectionFromLine  int
	connectionWaypoints []point
	filename            string
	fileList            []string
	selectedFileIndex   int
	fileOp              FileOperation
	openInNewBuffer     bool
	createNewBuffer     bool
	confirmAction       ConfirmAction
	confirmBoxID        int
	confirmTextID       int
	confirmConnIdx      int
	originalMoveX       int
	originalMoveY       int
	originalTextMoveX   int
	originalTextMoveY   int
	originalWidth       int
	originalHeight      int
	textInputX          int
	textInputY          int
	textInputText       string
	textInputCursorPos  int
	errorMessage        string
	successMessage      string
	fromStartup         bool
	clipboard           *Box
	config              *Config
	highlightMode       bool
	selectedColor       int // 0-7 for 8 colors
}

type point struct {
	X, Y int
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

type DeleteBoxData struct {
	Box         Box
	ID          int
	Connections []Connection
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
	ID     int
	X      int
	Y      int
	Width  int
	Height int
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
	Cells []HighlightCell // List of cells that were highlighted
}

type HighlightCell struct {
	X         int
	Y         int
	Color     int
	HadColor  bool // Whether this cell had a color before (for undo)
	OldColor  int  // The previous color if it existed
}

