package main

type Mode int

const (
	ModeStartup Mode = iota
	ModeNormal
	ModeCreating
	ModeEditing
	ModeTextInput
	ModeResize
	ModeMove
	ModeMultiSelect
	ModeFileInput
	ModeConfirm
	ModeBoxJump
	ModeTitleEdit
)

type FileOperation int

const (
	FileOpSave FileOperation = iota
	FileOpSavePNG
	FileOpSaveVisualTXT
	FileOpOpen
)

type ConfirmAction int

const (
	ConfirmDeleteBox ConfirmAction = iota
	ConfirmDeleteText
	ConfirmDeleteConnection
	ConfirmDeleteHighlight
	ConfirmQuit
	ConfirmNewChart
	ConfirmCloseBuffer
	ConfirmOverwriteFile
	ConfirmChooseExportType
	ConfirmDeleteChart
)

type ActionType int

const (
	ActionAddBox ActionType = iota
	ActionDeleteBox
	ActionEditBox
	ActionEditText
	ActionDeleteText
	ActionResizeBox
	ActionMoveBox
	ActionMoveText
	ActionAddConnection
	ActionDeleteConnection
	ActionCycleArrow
	ActionHighlight
	ActionChangeBorderStyle
	ActionEditTitle
)

type BorderStyle int

const (
	BorderStyleASCII BorderStyle = iota
	BorderStyleSingle
	BorderStyleDouble
	BorderStyleRounded
)

const (
	minBoxWidth      = 8
	minBoxHeight     = 3
	numColors        = 8
	colorEditSelect  = 100 // Special color index for edit text selection
)

