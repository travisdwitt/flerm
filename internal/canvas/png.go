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

func (c *Canvas) ExportToPNG(filename string, renderWidth, renderHeight int, panX, panY int) error {
	if len(c.boxes) == 0 && len(c.connections) == 0 && len(c.texts) == 0 {
		return fmt.Errorf("nothing to export")
	}

	charWidth := 8.0
	charHeight := 16.0
	minX, minY, maxX, maxY := c.GetFullBounds()
	if minX > maxX || minY > maxY {
		return fmt.Errorf("nothing to export")
	}

	padding := 2
	minX -= padding
	minY -= padding
	maxX += padding
	maxY += padding
	imageWidth := int(float64(maxX-minX) * charWidth)
	imageHeight := int(float64(maxY-minY) * charHeight)
	dc := gg.NewContext(imageWidth, imageHeight)
	dc.SetColor(color.White)
	dc.Clear()
	dc.SetColor(color.Black)
	fontData := gomono.TTF
	ttfFont, err := truetype.Parse(fontData)
	if err != nil {
		return fmt.Errorf("failed to parse font: %v", err)
	}
	face := truetype.NewFace(ttfFont, &truetype.Options{
		Size:    12.0,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	dc.SetFontFace(face)
	for _, conn := range c.connections {
		c.drawConnectionPNG(dc, conn, minX, minY, charWidth, charHeight)
	}
	for _, text := range c.texts {
		c.drawTextPNG(dc, text, minX, minY, charWidth, charHeight)
	}
	for _, box := range c.boxes {
		c.drawBoxPNG(dc, box, minX, minY, charWidth, charHeight)
	}

	return dc.SavePNG(filename)
}

// pngColor maps a palette index to ink; anything unset draws black.
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

func (c *Canvas) drawConnectionPNG(dc *gg.Context, conn Connection, minX, minY int, charWidth, charHeight float64) {
	points := connPoints(conn)
	if len(points) < 2 {
		return
	}
	dc.SetLineWidth(1.0)
	dc.SetColor(pngColor(conn.Color))
	for i := 0; i < len(points)-1; i++ {
		x1 := float64(points[i].X-minX) * charWidth
		y1 := float64(points[i].Y-minY) * charHeight
		x2 := float64(points[i+1].X-minX) * charWidth
		y2 := float64(points[i+1].Y-minY) * charHeight
		dc.DrawLine(x1, y1, x2, y2)
		dc.Stroke()
	}
	if conn.ArrowFrom && len(points) > 1 {
		c.drawArrowPNG(dc, points[1].X, points[1].Y, points[0].X, points[0].Y, minX, minY, charWidth, charHeight)
	}
	if conn.ArrowTo && len(points) > 1 {
		c.drawArrowPNG(dc, points[len(points)-2].X, points[len(points)-2].Y, points[len(points)-1].X, points[len(points)-1].Y, minX, minY, charWidth, charHeight)
	}
}

func (c *Canvas) drawArrowPNG(dc *gg.Context, fromX, fromY, toX, toY, minX, minY int, charWidth, charHeight float64) {
	fx := float64(fromX-minX) * charWidth
	fy := float64(fromY-minY) * charHeight
	tx := float64(toX-minX) * charWidth
	ty := float64(toY-minY) * charHeight
	dx := tx - fx
	dy := ty - fy
	length := math.Sqrt(dx*dx + dy*dy)
	if length < 0.1 {
		return
	}
	dx /= length
	dy /= length
	arrowSize := 6.0
	arrowAngle := 0.5
	tipX, tipY := tx, ty
	baseX1 := tx - arrowSize*dx + arrowSize*dy*arrowAngle
	baseY1 := ty - arrowSize*dy - arrowSize*dx*arrowAngle
	baseX2 := tx - arrowSize*dx - arrowSize*dy*arrowAngle
	baseY2 := ty - arrowSize*dy + arrowSize*dx*arrowAngle
	dc.MoveTo(tipX, tipY)
	dc.LineTo(baseX1, baseY1)
	dc.LineTo(baseX2, baseY2)
	dc.ClosePath()
	dc.Fill()
}

func (c *Canvas) drawBoxPNG(dc *gg.Context, box Box, minX, minY int, charWidth, charHeight float64) {
	x := float64(box.X-minX) * charWidth
	y := float64(box.Y-minY) * charHeight
	width := float64(box.Width) * charWidth
	height := float64(box.Height) * charHeight
	dc.SetLineWidth(1.0)
	dc.SetColor(pngColor(box.Color))
	dc.DrawRectangle(x, y, width, height)
	dc.Stroke()

	dc.SetColor(color.Black)
	textY := y + charHeight
	for i, line := range box.Lines {
		dc.DrawString(line, x+charWidth, textY+float64(i)*charHeight)
	}
}

func (c *Canvas) drawTextPNG(dc *gg.Context, text Text, minX, minY int, charWidth, charHeight float64) {
	x := float64(text.X-minX) * charWidth
	y := float64(text.Y-minY) * charHeight
	dc.SetColor(pngColor(text.Color))
	for i, line := range text.Lines {
		dc.DrawString(line, x, y+float64(i)*charHeight)
	}
}
