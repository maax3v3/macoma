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

// Delimiter detects which pixels in an image are delimiters (zone boundaries).
type Delimiter interface {
	Detect(img image.Image) *Map
}

// BorderDelimiter classifies pixels as delimiters if their color matches a
// specific border color within a tolerance.
type BorderDelimiter struct {
	Color        color.RGBA
	TolerancePct float64
}

// Detect classifies every pixel as delimiter or filler based on color distance
// to the configured border color.
func (d *BorderDelimiter) Detect(img image.Image) *Map {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	threshold := (d.TolerancePct / 100.0) * color.MaxRGBDistance

	dm := &Map{
		Width:       w,
		Height:      h,
		IsDelimiter: make([]bool, w*h),
	}

	parallelRows(h, func(sy, ey int) {
		for y := sy; y < ey; y++ {
			for x := 0; x < w; x++ {
				px := color.FromStdColor(img.At(bounds.Min.X+x, bounds.Min.Y+y))
				dist := color.DistanceRGB(px, d.Color)
				if dist <= threshold {
					dm.IsDelimiter[y*w+x] = true
				}
			}
		}
	})

	return dm
}

// ColorDelimiter classifies pixels as delimiters using a local range filter.
// For each pixel, it examines a 5×5 neighborhood and checks whether the
// color range (max − min per channel) exceeds the tolerance. This reliably
// detects edges even through anti-aliased transitions because the window
// spans both sides of the boundary.
type ColorDelimiter struct {
	TolerancePct float64
}

// Detect marks every pixel whose 5×5 neighborhood contains colors that
// differ by more than the tolerance.
//
// Performance notes:
//   - Precomputes a flat RGB buffer to avoid repeated interface dispatch.
//   - Uses squared integer RGB distance (no sqrt, no float per pixel).
//   - Parallelized across row bands — each worker only writes its own rows.
func (d *ColorDelimiter) Detect(img image.Image) *Map {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	// Precompute flat RGB buffer to avoid repeated img.At interface dispatch.
	buf := make([]color.RGBA, w*h)
	parallelRows(h, func(sy, ey int) {
		for y := sy; y < ey; y++ {
			for x := 0; x < w; x++ {
				buf[y*w+x] = color.FromStdColor(img.At(bounds.Min.X+x, bounds.Min.Y+y))
			}
		}
	})

	// Chebyshev threshold: max per-channel difference.
	// More sensitive than Euclidean to single-channel differences (e.g.
	// dark green vs black where only the green channel diverges).
	threshold := int(d.TolerancePct / 100.0 * 255.0)

	dm := &Map{
		Width:       w,
		Height:      h,
		IsDelimiter: make([]bool, w*h),
	}

	// Local range filter: for each pixel, compute the min/max of each
	// channel in its 5×5 neighborhood (radius 2). If the largest
	// per-channel range exceeds the threshold the pixel sits at a
	// color boundary.
	const radius = 2
	parallelRows(h, func(sy, ey int) {
		for y := sy; y < ey; y++ {
			for x := 0; x < w; x++ {
				var minR, minG, minB int = 255, 255, 255
				var maxR, maxG, maxB int

				y0 := y - radius
				if y0 < 0 {
					y0 = 0
				}
				y1 := y + radius
				if y1 >= h {
					y1 = h - 1
				}
				x0 := x - radius
				if x0 < 0 {
					x0 = 0
				}
				x1 := x + radius
				if x1 >= w {
					x1 = w - 1
				}

				for ny := y0; ny <= y1; ny++ {
					off := ny * w
					for nx := x0; nx <= x1; nx++ {
						c := buf[off+nx]
						r, g, b := int(c.R), int(c.G), int(c.B)
						if r < minR {
							minR = r
						}
						if r > maxR {
							maxR = r
						}
						if g < minG {
							minG = g
						}
						if g > maxG {
							maxG = g
						}
						if b < minB {
							minB = b
						}
						if b > maxB {
							maxB = b
						}
					}
				}

				dr := maxR - minR
				dg := maxG - minG
				db := maxB - minB
				maxDiff := dr
				if dg > maxDiff {
					maxDiff = dg
				}
				if db > maxDiff {
					maxDiff = db
				}
				if maxDiff > threshold {
					dm.IsDelimiter[y*w+x] = true
				}
			}
		}
	})

	return dm
}

// Detect is a convenience wrapper that creates a BorderDelimiter.
// Retained for backward compatibility.
func Detect(img image.Image, delimiterColor color.RGBA, tolerancePct float64) *Map {
	d := &BorderDelimiter{Color: delimiterColor, TolerancePct: tolerancePct}
	return d.Detect(img)
}

// parallelRows runs fn across row bands using multiple goroutines.
func parallelRows(h int, fn func(startY, endY int)) {
	numWorkers := 8
	rowsPerWorker := (h + numWorkers - 1) / numWorkers
	var wg sync.WaitGroup
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
			fn(sy, ey)
		}(startY, endY)
	}
	wg.Wait()
}
