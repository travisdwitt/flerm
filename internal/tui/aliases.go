package tui

import (
	cv "flerm/internal/canvas"
	"flerm/internal/config"
)

type (
	Canvas        = cv.Canvas
	Box           = cv.Box
	Text          = cv.Text
	Connection    = cv.Connection
	RenderResult  = cv.RenderResult
	HighlightCell = cv.HighlightCell
	BorderStyle   = cv.BorderStyle
	point         = cv.Point
	Config        = config.Config
)

var colorNames = cv.ColorNames

const (
	CoordUnset       = cv.CoordUnset
	numColors        = cv.NumColors
	colorMouseSelect = cv.ColorMouseSelect
	colorMenuSelect  = cv.ColorMenuSelect
	colorMenuBorder  = cv.ColorMenuBorder

	colorTooltipText    = cv.ColorTooltipText
	colorTooltipBorder  = cv.ColorTooltipBorder
	colorTooltipBracket = cv.ColorTooltipBracket

	BorderStyleASCII   = cv.BorderStyleASCII
	BorderStyleSingle  = cv.BorderStyleSingle
	BorderStyleDouble  = cv.BorderStyleDouble
	BorderStyleRounded = cv.BorderStyleRounded
)
