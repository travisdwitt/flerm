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

const (
	numColors        = cv.NumColors
	colorMouseSelect = cv.ColorMouseSelect
	colorMenuSelect  = cv.ColorMenuSelect
	colorMenuBorder  = cv.ColorMenuBorder

	BorderStyleASCII   = cv.BorderStyleASCII
	BorderStyleSingle  = cv.BorderStyleSingle
	BorderStyleDouble  = cv.BorderStyleDouble
	BorderStyleRounded = cv.BorderStyleRounded
)
