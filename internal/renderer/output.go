package renderer

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"sync"

	"github.com/maax3v3/macoma/internal/aggregation"
	"github.com/maax3v3/macoma/internal/detection"
	"github.com/maax3v3/macoma/internal/zone"
)

// Config holds rendering configuration.
type Config struct {
	LegendPadding    int // vertical padding above the legend
	LegendCircleSize int // diameter of legend color circles
	LegendSpacing    int // horizontal spacing between legend items
	LegendMargin     int // left/right margin for the legend area
}

// DefaultConfig returns sensible default rendering configuration.
func DefaultConfig() Config {
	return Config{
		LegendPadding:    20,
		LegendCircleSize: 30,
		LegendSpacing:    15,
		LegendMargin:     20,
	}
}

// Render produces the final magic coloring image.
func Render(
	srcImg image.Image,
	dm *detection.Map,
	zones []zone.Zone,
	labels []int,
	cm *aggregation.ColorMap,
	font FontRenderer,
	cfg Config,
) *image.RGBA {
	bounds := srcImg.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()

	// Calculate legend dimensions
	legendHeight := calculateLegendHeight(cm, cfg, srcW)
	totalH := srcH + legendHeight

	out := image.NewRGBA(image.Rect(0, 0, srcW, totalH))

	// Fill entire image with white
	for y := 0; y < totalH; y++ {
		for x := 0; x < srcW; x++ {
			out.SetRGBA(x, y, color.RGBA{255, 255, 255, 255})
		}
	}

	// Draw delimiter pixels in their original color (typically black)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for y := 0; y < srcH; y++ {
			for x := 0; x < srcW; x++ {
				if dm.At(x, y) {
					c := srcImg.At(bounds.Min.X+x, bounds.Min.Y+y)
					out.Set(x, y, c)
				}
			}
		}
	}()
	wg.Wait()

	// Compute font size based on image size (small for in-drawing labels)
	fontSize := computeFontSize(srcW, srcH, len(zones)) / 4
	if fontSize < 7 {
		fontSize = 7
	}

	// Draw zone numbers at centroids (parallelized)
	wg.Add(len(zones))
	for i := range zones {
		go func(zIdx int) {
			defer wg.Done()
			z := &zones[zIdx]
			entryIdx := cm.ZoneMap[zIdx]
			entry := cm.Entries[entryIdx]
			pos := z.InteriorPoint()

			numStr := fmt.Sprintf("%d", entry.Number)
			font.DrawString(out, numStr, pos.X, pos.Y, color.Black, fontSize)
		}(i)
	}
	wg.Wait()

	// Draw legend
	drawLegend(out, cm, font, cfg, srcW, srcH)

	return out
}

func computeFontSize(imgW, imgH, numZones int) int {
	// Heuristic: font size proportional to image size, scaled down with more zones
	base := math.Min(float64(imgW), float64(imgH)) / 30.0
	if numZones > 50 {
		base *= 0.7
	}
	if numZones > 200 {
		base *= 0.5
	}
	size := int(math.Round(base))
	if size < 7 {
		size = 7
	}
	if size > 40 {
		size = 40
	}
	return size
}

func calculateLegendHeight(cm *aggregation.ColorMap, cfg Config, imgW int) int {
	if len(cm.Entries) == 0 {
		return 0
	}
	// Calculate how many rows we need
	itemWidth := cfg.LegendCircleSize + cfg.LegendSpacing
	availableW := imgW - 2*cfg.LegendMargin
	itemsPerRow := availableW / itemWidth
	if itemsPerRow < 1 {
		itemsPerRow = 1
	}
	numRows := (len(cm.Entries) + itemsPerRow - 1) / itemsPerRow
	rowHeight := cfg.LegendCircleSize + cfg.LegendSpacing
	return cfg.LegendPadding + numRows*rowHeight + cfg.LegendPadding
}

func drawLegend(img *image.RGBA, cm *aggregation.ColorMap, font FontRenderer, cfg Config, imgW, drawingH int) {
	if len(cm.Entries) == 0 {
		return
	}

	// Draw a thin separator line
	separatorY := drawingH + cfg.LegendPadding/2
	for x := cfg.LegendMargin; x < imgW-cfg.LegendMargin; x++ {
		img.SetRGBA(x, separatorY, color.RGBA{200, 200, 200, 255})
	}

	itemWidth := cfg.LegendCircleSize + cfg.LegendSpacing
	availableW := imgW - 2*cfg.LegendMargin
	itemsPerRow := availableW / itemWidth
	if itemsPerRow < 1 {
		itemsPerRow = 1
	}

	fontSize := cfg.LegendCircleSize * 2 / 3
	radius := cfg.LegendCircleSize / 2

	for i, entry := range cm.Entries {
		row := i / itemsPerRow
		col := i % itemsPerRow

		// Center items in each row
		rowItemCount := itemsPerRow
		remaining := len(cm.Entries) - row*itemsPerRow
		if remaining < itemsPerRow {
			rowItemCount = remaining
		}
		rowWidth := rowItemCount * itemWidth
		rowStartX := cfg.LegendMargin + (availableW-rowWidth)/2

		cx := rowStartX + col*itemWidth + radius
		cy := drawingH + cfg.LegendPadding + row*(cfg.LegendCircleSize+cfg.LegendSpacing) + radius

		// Draw filled circle
		fillColor := entry.Color.ToStdColor()
		drawFilledCircle(img, cx, cy, radius, fillColor)

		// Draw circle border
		drawCircleBorder(img, cx, cy, radius, color.RGBA{100, 100, 100, 255})

		// Draw number text
		textColor := color.Color(color.Black)
		if !entry.Color.IsLight() {
			textColor = color.White
		}
		numStr := fmt.Sprintf("%d", entry.Number)
		font.DrawString(img, numStr, cx, cy, textColor, fontSize)
	}
}

func drawFilledCircle(img *image.RGBA, cx, cy, radius int, col color.RGBA) {
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			if dx*dx+dy*dy <= radius*radius {
				px, py := cx+dx, cy+dy
				if px >= 0 && px < img.Bounds().Dx() && py >= 0 && py < img.Bounds().Dy() {
					img.SetRGBA(px, py, col)
				}
			}
		}
	}
}

func drawCircleBorder(img *image.RGBA, cx, cy, radius int, col color.RGBA) {
	for angle := 0.0; angle < 2*math.Pi; angle += 0.01 {
		px := cx + int(math.Round(float64(radius)*math.Cos(angle)))
		py := cy + int(math.Round(float64(radius)*math.Sin(angle)))
		if px >= 0 && px < img.Bounds().Dx() && py >= 0 && py < img.Bounds().Dy() {
			img.SetRGBA(px, py, col)
		}
	}
}
