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
	minX, minY := 0, 0
	maxX, maxY := 0, 0
	hasElements := false
	for _, box := range c.boxes {
		if !hasElements {
			minX, minY = box.X, box.Y
			maxX, maxY = box.X+box.Width, box.Y+box.Height
			hasElements = true
		} else {
			if box.X < minX {
				minX = box.X
			}
			if box.Y < minY {
				minY = box.Y
			}
			if box.X+box.Width > maxX {
				maxX = box.X + box.Width
			}
			if box.Y+box.Height > maxY {
				maxY = box.Y + box.Height
			}
		}
	}

	for _, conn := range c.connections {
		points := []Point{{conn.FromX, conn.FromY}}
		points = append(points, conn.Waypoints...)
		points = append(points, Point{conn.ToX, conn.ToY})

		for _, pt := range points {
			if !hasElements {
				minX, minY = pt.X, pt.Y
				maxX, maxY = pt.X, pt.Y
				hasElements = true
			} else {
				if pt.X < minX {
					minX = pt.X
				}
				if pt.Y < minY {
					minY = pt.Y
				}
				if pt.X > maxX {
					maxX = pt.X
				}
				if pt.Y > maxY {
					maxY = pt.Y
				}
			}
		}
	}

	for _, text := range c.texts {
		if !hasElements {
			minX, minY = text.X, text.Y
			maxX, maxY = text.X, text.Y
			hasElements = true
		} else {
			if text.X < minX {
				minX = text.X
			}
			if text.Y < minY {
				minY = text.Y
			}
			maxTextX := text.X
			for _, line := range text.Lines {
				if text.X+len(line) > maxTextX {
					maxTextX = text.X + len(line)
				}
			}
			if maxTextX > maxX {
				maxX = maxTextX
			}
			if text.Y+len(text.Lines) > maxY {
				maxY = text.Y + len(text.Lines)
			}
		}
	}

	if !hasElements {
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

func pngColor(index int) color.Color {
	switch index {
	case 0:
		return color.RGBA{128, 128, 128, 255}
	case 1:
		return color.RGBA{205, 0, 0, 255}
	case 2:
		return color.RGBA{0, 160, 0, 255}
	case 3:
		return color.RGBA{190, 160, 0, 255}
	case 4:
		return color.RGBA{0, 0, 220, 255}
	case 5:
		return color.RGBA{190, 0, 190, 255}
	case 6:
		return color.RGBA{0, 170, 170, 255}
	case 7:
		return color.RGBA{230, 230, 230, 255}
	default:
		return color.Black
	}
}

func (c *Canvas) drawConnectionPNG(dc *gg.Context, conn Connection, minX, minY int, charWidth, charHeight float64) {
	points := []Point{{conn.FromX, conn.FromY}}
	points = append(points, conn.Waypoints...)
	points = append(points, Point{conn.ToX, conn.ToY})
	if len(points) < 2 {
		return
	}
	dc.SetLineWidth(1.0)
	if conn.Color >= 0 {
		dc.SetColor(pngColor(conn.Color))
	} else {
		dc.SetColor(color.Black)
	}
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
	if box.Color >= 0 {
		dc.SetColor(pngColor(box.Color))
	} else {
		dc.SetColor(color.Black)
	}
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
	if text.Color >= 0 {
		dc.SetColor(pngColor(text.Color))
	} else {
		dc.SetColor(color.Black)
	}
	for i, line := range text.Lines {
		dc.DrawString(line, x, y+float64(i)*charHeight)
	}
}
