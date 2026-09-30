package tui

type Mode int

const (
	ModeStartup Mode = iota
	ModeNormal
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
	ModeTooltipEdit
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
	MenuEditTooltip
	MenuSetBorderStyle
	MenuSetColor
	MenuDuplicate
	MenuMultiSelect
	MenuSubmenu
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
	ConfirmQuit
	ConfirmNewChart
	ConfirmCloseBuffer
	ConfirmOverwriteFile
	ConfirmChooseExportType
	ConfirmDeleteBoxOrTooltip
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
	ActionEditTooltip
	ActionSetColor
	ActionDuplicate
	ActionGroupMove
)
