package canvas

import (
	"fmt"
	"image/color"
	"math"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
)

const (
	pngCharW = 8.0
	pngCharH = 16.0
)

func (c *Canvas) ExportToPNG(filename string) error {
	minX, minY, maxX, maxY := c.GetFullBounds()
	if minX > maxX || minY > maxY {
		return fmt.Errorf("nothing to export")
	}

	const padding = 2
	minX -= padding
	minY -= padding
	maxX += padding
	maxY += padding
	dc := gg.NewContext(int(float64(maxX-minX)*pngCharW), int(float64(maxY-minY)*pngCharH))
	dc.SetColor(color.White)
	dc.Clear()
	dc.SetColor(color.Black)
	ttfFont, err := truetype.Parse(gomono.TTF)
	if err != nil {
		return fmt.Errorf("failed to parse font: %v", err)
	}
	dc.SetFontFace(truetype.NewFace(ttfFont, &truetype.Options{
		Size:    12.0,
		DPI:     72,
		Hinting: font.HintingFull,
	}))
	px := func(x, y int) (float64, float64) {
		return float64(x-minX) * pngCharW, float64(y-minY) * pngCharH
	}
	dc.SetLineWidth(1.0)
	for _, conn := range c.connections {
		drawConnectionPNG(dc, conn, px)
	}
	for _, text := range c.texts {
		x, y := px(text.X, text.Y)
		dc.SetColor(pngColor(text.Color))
		for i, line := range text.Lines {
			dc.DrawString(line, x, y+float64(i)*pngCharH)
		}
	}
	for _, box := range c.boxes {
		x, y := px(box.X, box.Y)
		dc.SetColor(pngColor(box.Color))
		dc.DrawRectangle(x, y, float64(box.Width)*pngCharW, float64(box.Height)*pngCharH)
		dc.Stroke()
		dc.SetColor(color.Black)
		for i, line := range box.Lines {
			dc.DrawString(line, x+pngCharW, y+float64(i+1)*pngCharH)
		}
	}

	return dc.SavePNG(filename)
}

func pngColor(index int) color.Color {
	palette := []color.Color{
		color.RGBA{128, 128, 128, 255}, color.RGBA{205, 0, 0, 255},
		color.RGBA{0, 160, 0, 255}, color.RGBA{190, 160, 0, 255},
		color.RGBA{0, 0, 220, 255}, color.RGBA{190, 0, 190, 255},
		color.RGBA{0, 170, 170, 255}, color.RGBA{230, 230, 230, 255},
	}
	if index < 0 || index >= len(palette) {
		return color.Black
	}
	return palette[index]
}

func drawConnectionPNG(dc *gg.Context, conn Connection, px func(x, y int) (float64, float64)) {
	points := connPoints(conn)
	dc.SetColor(pngColor(conn.Color))
	for i := 0; i < len(points)-1; i++ {
		x1, y1 := px(points[i].X, points[i].Y)
		x2, y2 := px(points[i+1].X, points[i+1].Y)
		dc.DrawLine(x1, y1, x2, y2)
		dc.Stroke()
	}
	if conn.ArrowFrom {
		drawArrowPNG(dc, points[1], points[0], px)
	}
	if conn.ArrowTo {
		drawArrowPNG(dc, points[len(points)-2], points[len(points)-1], px)
	}
}

func drawArrowPNG(dc *gg.Context, from, to Point, px func(x, y int) (float64, float64)) {
	fx, fy := px(from.X, from.Y)
	tx, ty := px(to.X, to.Y)
	dx, dy := tx-fx, ty-fy
	length := math.Hypot(dx, dy)
	if length < 0.1 {
		return
	}
	dx /= length
	dy /= length
	const size, angle = 6.0, 0.5
	dc.MoveTo(tx, ty)
	dc.LineTo(tx-size*dx+size*dy*angle, ty-size*dy-size*dx*angle)
	dc.LineTo(tx-size*dx-size*dy*angle, ty-size*dy+size*dx*angle)
	dc.ClosePath()
	dc.Fill()
}
