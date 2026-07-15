package canvas

type Point struct {
	X, Y int
}

type HighlightCell struct {
	X        int
	Y        int
	Color    int
	HadColor bool
	OldColor int
}
