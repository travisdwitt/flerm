package canvas

import (
	"fmt"
	"image/color"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
)

const (
	pngCharW   = 14.0
	pngCharH   = 30.0
	pngPadding = 2
	pngLineW   = 1.7
	pngRailGap = pngCharW * 0.18
	pngRadius  = pngCharW * 0.45
)

var (
	pngBG        = color.RGBA{0x16, 0x18, 0x1d, 0xff}
	pngDefaultFG = color.RGBA{0xdc, 0xdf, 0xe4, 0xff}
)

func (c *Canvas) ExportToPNG(filename string) error {
	minX, minY, maxX, maxY := c.GetFullBounds()
	if minX > maxX || minY > maxY {
		return fmt.Errorf("nothing to export")
	}
	minX, minY = minX-pngPadding, minY-pngPadding
	cols := maxX - minX + pngPadding + 1
	rows := maxY - minY + pngPadding + 1

	r := c.RenderRaw(cols, rows, -1, CoordUnset, CoordUnset, nil, CoordUnset, CoordUnset, minX, minY, -1, -1, false, -1, -1, 0, "", CoordUnset, CoordUnset, CoordUnset, CoordUnset, CoordUnset, CoordUnset, false, -1, -1)

	dc := gg.NewContext(int(float64(cols)*pngCharW), int(float64(rows)*pngCharH))
	dc.SetColor(pngBG)
	dc.Clear()

	ttfFont, err := truetype.Parse(gomono.TTF)
	if err != nil {
		return fmt.Errorf("failed to parse font: %v", err)
	}
	dc.SetFontFace(truetype.NewFace(ttfFont, &truetype.Options{
		Size:    pngCharW / 0.6,
		DPI:     72,
		Hinting: font.HintingNone,
	}))
	advance, _ := dc.MeasureString("M")
	dc.SetLineWidth(pngLineW)

	for y, row := range r.Canvas {
		runColor, runBlank := -1, false
		for x, ch := range row {
			colorIndex := -1
			if y < len(r.ColorMap) && x < len(r.ColorMap[y]) {
				colorIndex = r.ColorMap[y][x]
			}
			if colorIndex != runColor {
				runColor, runBlank = colorIndex, ch == ' '
			}
			left, top := float64(x)*pngCharW, float64(y)*pngCharH
			fg := pngColor(colorIndex)
			asBG := colorIndex >= 0 && runBlank
			if asBG {
				dc.SetColor(fg)
				dc.DrawRectangle(left, top, pngCharW, pngCharH)
				dc.Fill()
				fg = pngBG
			}
			if ch == ' ' {
				continue
			}
			dc.SetColor(fg)
			if drawCellRune(dc, ch, fg, left, top) {
				continue
			}
			dc.DrawString(string(ch), left+(pngCharW-advance)/2, top+pngCharH*0.78)
		}
	}
	return dc.SavePNG(filename)
}

func pngColor(index int) color.Color {
	palette := []color.Color{
		color.RGBA{0x70, 0x78, 0x85, 255}, color.RGBA{0xe0, 0x52, 0x52, 255},
		color.RGBA{0x4c, 0xb0, 0x5e, 255}, color.RGBA{0xd4, 0xa0, 0x37, 255},
		color.RGBA{0x54, 0x8a, 0xe8, 255}, color.RGBA{0xb4, 0x6c, 0xe0, 255},
		color.RGBA{0x3d, 0xb0, 0xb8, 255}, color.RGBA{0xc5, 0xca, 0xd3, 255},
		color.RGBA{0x4a, 0x4f, 0x5a, 255}, color.RGBA{0xff, 0x7b, 0x72, 255},
		color.RGBA{0x74, 0xdd, 0x84, 255}, color.RGBA{0xf5, 0xc7, 0x5e, 255},
		color.RGBA{0x7d, 0xb0, 0xff, 255}, color.RGBA{0xdb, 0x9a, 0xf5, 255},
		color.RGBA{0x62, 0xdc, 0xe4, 255}, color.RGBA{0xff, 0xff, 0xff, 255},
	}
	if index < 0 || index >= len(palette) {
		return pngDefaultFG
	}
	return palette[index]
}

const (
	dirL = 1 << iota
	dirR
	dirU
	dirD
)

const (
	styleSingle = iota
	styleDouble
	styleRounded
)

var pngBoxRunes = map[rune]struct {
	dirs  int
	style int
}{
	'─': {dirL | dirR, styleSingle}, '│': {dirU | dirD, styleSingle},
	'┌': {dirR | dirD, styleSingle}, '┐': {dirL | dirD, styleSingle},
	'└': {dirR | dirU, styleSingle}, '┘': {dirL | dirU, styleSingle},
	'├': {dirR | dirU | dirD, styleSingle}, '┤': {dirL | dirU | dirD, styleSingle},
	'┬': {dirL | dirR | dirD, styleSingle}, '┴': {dirL | dirR | dirU, styleSingle},
	'┼': {dirL | dirR | dirU | dirD, styleSingle},
	'═': {dirL | dirR, styleDouble}, '║': {dirU | dirD, styleDouble},
	'╔': {dirR | dirD, styleDouble}, '╗': {dirL | dirD, styleDouble},
	'╚': {dirR | dirU, styleDouble}, '╝': {dirL | dirU, styleDouble},
	'╭': {dirR | dirD, styleRounded}, '╮': {dirL | dirD, styleRounded},
	'╰': {dirR | dirU, styleRounded}, '╯': {dirL | dirU, styleRounded},
}

func drawCellRune(dc *gg.Context, ch rune, fg color.Color, left, top float64) bool {
	cx, cy := left+pngCharW/2, top+pngCharH/2
	right, bottom := left+pngCharW, top+pngCharH
	hw, hh := pngCharW*0.36, pngCharH*0.28

	switch ch {
	case '█':
		dc.DrawRectangle(left, top, pngCharW, pngCharH)
		dc.Fill()
		return true
	case '░', '▒', '▓':
		r, g, b, _ := fg.RGBA()
		alpha := map[rune]float64{'░': 0.25, '▒': 0.5, '▓': 0.75}[ch]
		dc.SetRGBA(float64(r>>8)/255, float64(g>>8)/255, float64(b>>8)/255, alpha)
		dc.DrawRectangle(left, top, pngCharW, pngCharH)
		dc.Fill()
		return true
	case '▶', '◀', '▲', '▼':
		switch ch {
		case '▶':
			dc.MoveTo(cx-hw, cy-hh)
			dc.LineTo(cx+hw, cy)
			dc.LineTo(cx-hw, cy+hh)
		case '◀':
			dc.MoveTo(cx+hw, cy-hh)
			dc.LineTo(cx-hw, cy)
			dc.LineTo(cx+hw, cy+hh)
		case '▲':
			dc.MoveTo(cx, cy-hh)
			dc.LineTo(cx+hw, cy+hh)
			dc.LineTo(cx-hw, cy+hh)
		default:
			dc.MoveTo(cx, cy+hh)
			dc.LineTo(cx+hw, cy-hh)
			dc.LineTo(cx-hw, cy-hh)
		}
		dc.ClosePath()
		dc.Fill()
		return true
	}

	glyph, ok := pngBoxRunes[ch]
	if !ok {
		return false
	}
	hasH := glyph.dirs&(dirL|dirR) != 0
	hasV := glyph.dirs&(dirU|dirD) != 0
	hEdge, vEdge := right, bottom
	if glyph.dirs&dirL != 0 {
		hEdge = left
	}
	if glyph.dirs&dirU != 0 {
		vEdge = top
	}

	switch {
	case glyph.style == styleRounded:
		dc.MoveTo(hEdge, cy)
		dc.LineTo(cx+sign(hEdge-cx)*pngRadius, cy)
		dc.QuadraticTo(cx, cy, cx, cy+sign(vEdge-cy)*pngRadius)
		dc.LineTo(cx, vEdge)

	case glyph.style == styleDouble && hasH && hasV:
		h, v := sign(hEdge-cx), sign(vEdge-cy)
		for _, s := range []float64{-1, 1} {
			dc.MoveTo(cx+s*h*pngRailGap, cy+s*v*pngRailGap)
			dc.LineTo(hEdge, cy+s*v*pngRailGap)
			dc.MoveTo(cx+s*h*pngRailGap, cy+s*v*pngRailGap)
			dc.LineTo(cx+s*h*pngRailGap, vEdge)
		}

	case glyph.style == styleDouble:
		for _, s := range []float64{-pngRailGap, pngRailGap} {
			if hasH {
				dc.MoveTo(left, cy+s)
				dc.LineTo(right, cy+s)
			} else {
				dc.MoveTo(cx+s, top)
				dc.LineTo(cx+s, bottom)
			}
		}

	default:
		if glyph.dirs&dirL != 0 {
			dc.MoveTo(left, cy)
			dc.LineTo(cx, cy)
		}
		if glyph.dirs&dirR != 0 {
			dc.MoveTo(cx, cy)
			dc.LineTo(right, cy)
		}
		if glyph.dirs&dirU != 0 {
			dc.MoveTo(cx, top)
			dc.LineTo(cx, cy)
		}
		if glyph.dirs&dirD != 0 {
			dc.MoveTo(cx, cy)
			dc.LineTo(cx, bottom)
		}
	}
	dc.Stroke()
	return true
}

func sign(v float64) float64 {
	if v < 0 {
		return -1
	}
	return 1
}
