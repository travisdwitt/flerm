package canvas

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
	numZLevels       = 4
	NumColors        = 8
	colorEditSelect  = 100
	ColorMouseSelect = 101
	ColorMenuSelect  = 102
	ColorMenuBorder  = 103
)
