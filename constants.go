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
	ActionResizeBox
	ActionMoveBox
	ActionMoveText
	ActionAddConnection
	ActionDeleteConnection
	ActionCycleArrow
	ActionHighlight
)

const (
	minBoxWidth  = 8
	minBoxHeight = 3
	numColors    = 8 // Number of highlight colors
)

