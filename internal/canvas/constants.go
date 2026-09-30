package canvas

import "math"

type BorderStyle int

const (
	BorderStyleASCII BorderStyle = iota
	BorderStyleSingle
	BorderStyleDouble
	BorderStyleRounded
)

const (
	boxInsetX        = 2
	minBoxWidth      = 8
	minBoxHeight     = 3
	numZLevels       = 4
	NumColors        = 16
	colorEditSelect  = 100
	ColorMouseSelect = 101
	ColorMenuSelect  = 102
	ColorMenuBorder  = 103
)

const CoordUnset = math.MinInt32
