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
	ModeContextMenu
)

type MenuAction int

const (
	MenuNewBox MenuAction = iota
	MenuNewText
	MenuEditBox
	MenuEditText
	MenuNewLine
	MenuDeleteBox
	MenuDeleteText
	MenuDeleteLine
	MenuEditTitle
	MenuSetBorderStyle // Arg = BorderStyle value
	MenuSetColor       // Arg = palette color index, or -1 for none
	MenuSubmenu        // opens Item.Submenu; fires no action
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
	ActionSetColor
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
	colorMouseSelect = 101 // Special color index for a mouse-selected element
	colorMenuSelect  = 102 // Special color index for the focused context-menu item
	colorMenuBorder  = 103 // Special color index for the context-menu border (green)
)

