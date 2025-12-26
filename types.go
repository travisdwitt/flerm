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
	selectedBox           int
	selectedText          int
	editText              string
	editCursorPos         int
	editCursorRow         int
	editCursorCol         int
	originalEditText      string
	connectionFrom        int
	connectionFromX       int
	connectionFromY       int
	connectionFromLine    int
	connectionWaypoints   []point
	filename              string
	fileList              []string
	selectedFileIndex     int
	fileOp                FileOperation
	openInNewBuffer       bool
	createNewBuffer       bool
	showingDeleteConfirm  bool
	confirmAction         ConfirmAction
	confirmBoxID          int
	confirmTextID         int
	confirmConnIdx        int
	confirmHighlightX     int
	confirmHighlightY     int
	confirmFileIndex      int
	originalMoveX         int
	originalMoveY         int
	originalTextMoveX     int
	originalTextMoveY     int
	originalWidth         int
	originalHeight        int
	textInputX            int
	textInputY            int
	textInputText         string
	textInputCursorPos    int
	errorMessage          string
	successMessage        string
	fromStartup           bool
	clipboard             *Box
	config                *Config
	highlightMode         bool
	selectedColor         int
	selectionStartX       int
	selectionStartY       int
	selectedBoxes         []int
	selectedTexts         []int
	selectedConnections   []int
	originalBoxPositions  map[int]point
	originalTextPositions map[int]point
	originalConnections   map[int]Connection
	originalHighlights    map[point]int
	highlightMoveDelta    point
	boxJumpInput          string
	showTooltip           bool
	tooltipText           string
	tooltipX              int
	tooltipY              int
	tooltipBoxID          int // ID of the box being shown in tooltip, -1 if none

	// Konami Code Easter Egg
	konamiProgress  int            // How far through the code sequence
	easterEggActive bool           // Whether the falling animation is running
	fallingPieces   []FallingPiece // Pieces (boxes, lines) that are exploding
	particles       []Particle     // Trail particles
	piledChars      [][]rune       // Characters that have piled up at the bottom
	piledColors     [][]int        // Colors of piled characters
}

// FallingPiece represents a chunk of characters (box, line segment) exploding together
type FallingPiece struct {
	Chars  []PieceChar // Characters in this piece with relative positions
	X      float64     // Center X position
	Y      float64     // Center Y position
	VelX   float64     // Horizontal velocity
	VelY   float64     // Vertical velocity
	Rot    float64     // Rotation angle (for visual effect)
	RotVel float64     // Rotation velocity
	Color  int         // Piece color for trail
	Landed bool        // Whether this piece has landed
}

// PieceChar represents a character within a piece
type PieceChar struct {
	Char    rune
	OffsetX float64 // Offset from piece center
	OffsetY float64
	Color   int
}

// Particle represents a trail particle
type Particle struct {
	Char  rune
	X     float64
	Y     float64
	Life  int // Frames remaining
	Color int
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
	Cells []HighlightCell
}

type HighlightCell struct {
	X        int
	Y        int
	Color    int
	HadColor bool
	OldColor int
}

type BorderStyleData struct {
	BoxID    int
	OldStyle BorderStyle
	NewStyle BorderStyle
}
