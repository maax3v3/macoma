package detection

import (
	"image"
	"sync"

	"github.com/maax3v3/macoma/internal/color"
)

// Map holds a boolean grid where true means the pixel is a delimiter pixel.
type Map struct {
	Width, Height int
	IsDelimiter   []bool // row-major: index = y*Width + x
}

// At returns whether the pixel at (x, y) is a delimiter.
func (m *Map) At(x, y int) bool {
	return m.IsDelimiter[y*m.Width+x]
}

// Detect classifies every pixel in the image as delimiter or filler.
// A pixel is a delimiter if its color distance (in RGB space, as a percentage
// of the maximum distance) to delimiterColor is within tolerancePct.
func Detect(img image.Image, delimiterColor color.RGBA, tolerancePct float64) *Map {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	threshold := (tolerancePct / 100.0) * color.MaxRGBDistance

	dm := &Map{
		Width:       w,
		Height:      h,
		IsDelimiter: make([]bool, w*h),
	}

	// Parallelize row-by-row
	var wg sync.WaitGroup
	numWorkers := 8
	rowsPerWorker := (h + numWorkers - 1) / numWorkers

	for worker := 0; worker < numWorkers; worker++ {
		startY := worker * rowsPerWorker
		endY := startY + rowsPerWorker
		if endY > h {
			endY = h
		}
		if startY >= h {
			break
		}
		wg.Add(1)
		go func(sy, ey int) {
			defer wg.Done()
			for y := sy; y < ey; y++ {
				for x := 0; x < w; x++ {
					px := color.FromStdColor(img.At(bounds.Min.X+x, bounds.Min.Y+y))
					dist := color.DistanceRGB(px, delimiterColor)
					if dist <= threshold {
						dm.IsDelimiter[y*w+x] = true
					}
				}
			}
		}(startY, endY)
	}

	wg.Wait()
	return dm
}
