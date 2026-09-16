package canvas

type BorderStyle int

const (
	BorderStyleASCII BorderStyle = iota
	BorderStyleSingle
	BorderStyleDouble
	BorderStyleRounded
)

const (
	// boxInsetX is the columns between a box's left edge and its text: the
	// border plus one blank padding column. Same on the right.
	boxInsetX        = 2
	minBoxWidth      = 8
	minBoxHeight     = 3
	numZLevels       = 4
	NumColors        = 8
	colorEditSelect  = 100
	ColorMouseSelect = 101
	ColorMenuSelect  = 102
	ColorMenuBorder  = 103
)
