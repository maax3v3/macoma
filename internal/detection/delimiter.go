package detection

import (
	"image"
	"math"
	"runtime"
	"sync"

	"github.com/maax3v3/macoma/v2/internal/color"
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

// GradientDelimiter classifies pixels as delimiters from gradient edges.
// The output favors closed, continuous contours robust to gradients/shading.
type GradientDelimiter struct {
	BlurSigma        float64
	LowThreshold     float64 // normalized 0..1
	HighThreshold    float64 // normalized 0..1
	CloseRadius      int
	MinComponentSize int
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

// Detect applies a classical edge pipeline:
// 1) luminance + Gaussian blur, 2) Sobel gradient magnitude,
// 3) hysteresis thresholding, 4) morphology close, 5) tiny-component removal.
func (d *GradientDelimiter) Detect(img image.Image) *Map {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	dm := &Map{
		Width:       w,
		Height:      h,
		IsDelimiter: make([]bool, w*h),
	}
	if w == 0 || h == 0 {
		return dm
	}

	sigma := d.BlurSigma
	if sigma <= 0 {
		sigma = 1.2
	}
	low := clamp01(d.LowThreshold)
	high := clamp01(d.HighThreshold)
	if low > high {
		low = high
	}

	luma := make([]float32, w*h)
	parallelRows(h, func(sy, ey int) {
		for y := sy; y < ey; y++ {
			off := y * w
			for x := 0; x < w; x++ {
				c := color.FromStdColor(img.At(bounds.Min.X+x, bounds.Min.Y+y))
				luma[off+x] = 0.2126*float32(c.R) + 0.7152*float32(c.G) + 0.0722*float32(c.B)
			}
		}
	})

	blurred := gaussianBlurSeparable(luma, w, h, sigma)
	gradMag := sobelMagnitudeSq(blurred, w, h)
	var maxMag float32
	for _, v := range gradMag {
		if v > maxMag {
			maxMag = v
		}
	}
	if maxMag <= 0 {
		return dm
	}

	strongT := float32(high) * maxMag
	weakT := float32(low) * maxMag
	isEdge := hysteresisEdges(gradMag, w, h, weakT, strongT)

	if d.CloseRadius > 0 {
		isEdge = morphClose(isEdge, w, h, d.CloseRadius)
	}
	if d.MinComponentSize > 1 {
		isEdge = removeSmallComponents(isEdge, w, h, d.MinComponentSize)
	}
	dm.IsDelimiter = isEdge
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
	numWorkers := runtime.GOMAXPROCS(0)
	if numWorkers < 1 {
		numWorkers = 1
	}
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

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func gaussianKernel1D(sigma float64) []float32 {
	radius := int(math.Ceil(3 * sigma))
	if radius < 1 {
		radius = 1
	}
	k := make([]float32, 2*radius+1)
	sum := 0.0
	for i := -radius; i <= radius; i++ {
		v := math.Exp(-float64(i*i) / (2 * sigma * sigma))
		k[i+radius] = float32(v)
		sum += v
	}
	for i := range k {
		k[i] /= float32(sum)
	}
	return k
}

func gaussianBlurSeparable(src []float32, w, h int, sigma float64) []float32 {
	k := gaussianKernel1D(sigma)
	r := len(k) / 2
	tmp := make([]float32, len(src))
	dst := make([]float32, len(src))

	parallelRows(h, func(sy, ey int) {
		for y := sy; y < ey; y++ {
			off := y * w
			for x := 0; x < w; x++ {
				var sum float32
				for dx := -r; dx <= r; dx++ {
					nx := x + dx
					if nx < 0 {
						nx = 0
					} else if nx >= w {
						nx = w - 1
					}
					sum += src[off+nx] * k[dx+r]
				}
				tmp[off+x] = sum
			}
		}
	})

	parallelRows(h, func(sy, ey int) {
		for y := sy; y < ey; y++ {
			off := y * w
			for x := 0; x < w; x++ {
				var sum float32
				for dy := -r; dy <= r; dy++ {
					ny := y + dy
					if ny < 0 {
						ny = 0
					} else if ny >= h {
						ny = h - 1
					}
					sum += tmp[ny*w+x] * k[dy+r]
				}
				dst[off+x] = sum
			}
		}
	})
	return dst
}

func sobelMagnitudeSq(src []float32, w, h int) []float32 {
	dst := make([]float32, w*h)
	parallelRows(h, func(sy, ey int) {
		for y := sy; y < ey; y++ {
			for x := 0; x < w; x++ {
				xm1 := x - 1
				if xm1 < 0 {
					xm1 = 0
				}
				xp1 := x + 1
				if xp1 >= w {
					xp1 = w - 1
				}
				ym1 := y - 1
				if ym1 < 0 {
					ym1 = 0
				}
				yp1 := y + 1
				if yp1 >= h {
					yp1 = h - 1
				}
				a := src[ym1*w+xm1]
				b := src[ym1*w+x]
				c := src[ym1*w+xp1]
				d := src[y*w+xm1]
				f := src[y*w+xp1]
				g := src[yp1*w+xm1]
				hv := src[yp1*w+x]
				i := src[yp1*w+xp1]

				gx := -a + c - 2*d + 2*f - g + i
				gy := -a - 2*b - c + g + 2*hv + i
				dst[y*w+x] = gx*gx + gy*gy
			}
		}
	})
	return dst
}

func hysteresisEdges(grad []float32, w, h int, weakT, strongT float32) []bool {
	n := w * h
	out := make([]bool, n)
	// 0 = none, 1 = weak, 2 = strong (and eventually visited).
	mask := make([]uint8, n)
	queue := make([]int, 0, n/16)

	for i, v := range grad {
		if v >= strongT {
			mask[i] = 2
			out[i] = true
			queue = append(queue, i)
			continue
		}
		if v >= weakT {
			mask[i] = 1
		}
	}
	if len(queue) == 0 {
		return out
	}
	for head := 0; head < len(queue); head++ {
		i := queue[head]
		x := i % w
		y := i / w
		for dy := -1; dy <= 1; dy++ {
			ny := y + dy
			if ny < 0 || ny >= h {
				continue
			}
			for dx := -1; dx <= 1; dx++ {
				if dx == 0 && dy == 0 {
					continue
				}
				nx := x + dx
				if nx < 0 || nx >= w {
					continue
				}
				ni := ny*w + nx
				if mask[ni] != 1 {
					continue
				}
				mask[ni] = 2
				out[ni] = true
				queue = append(queue, ni)
			}
		}
	}
	return out
}

func morphClose(src []bool, w, h, radius int) []bool {
	if radius <= 0 {
		out := make([]bool, len(src))
		copy(out, src)
		return out
	}
	return erodeBool(dilateBool(src, w, h, radius), w, h, radius)
}

func dilateBool(src []bool, w, h, radius int) []bool {
	dst := make([]bool, len(src))
	for y := 0; y < h; y++ {
		y0 := y - radius
		if y0 < 0 {
			y0 = 0
		}
		y1 := y + radius
		if y1 >= h {
			y1 = h - 1
		}
		for x := 0; x < w; x++ {
			x0 := x - radius
			if x0 < 0 {
				x0 = 0
			}
			x1 := x + radius
			if x1 >= w {
				x1 = w - 1
			}
			found := false
			for ny := y0; ny <= y1 && !found; ny++ {
				off := ny * w
				for nx := x0; nx <= x1; nx++ {
					if src[off+nx] {
						found = true
						break
					}
				}
			}
			dst[y*w+x] = found
		}
	}
	return dst
}

func erodeBool(src []bool, w, h, radius int) []bool {
	dst := make([]bool, len(src))
	for y := 0; y < h; y++ {
		y0 := y - radius
		if y0 < 0 {
			y0 = 0
		}
		y1 := y + radius
		if y1 >= h {
			y1 = h - 1
		}
		for x := 0; x < w; x++ {
			x0 := x - radius
			if x0 < 0 {
				x0 = 0
			}
			x1 := x + radius
			if x1 >= w {
				x1 = w - 1
			}
			allSet := true
			for ny := y0; ny <= y1 && allSet; ny++ {
				off := ny * w
				for nx := x0; nx <= x1; nx++ {
					if !src[off+nx] {
						allSet = false
						break
					}
				}
			}
			dst[y*w+x] = allSet
		}
	}
	return dst
}

func removeSmallComponents(src []bool, w, h, minSize int) []bool {
	if minSize <= 1 {
		out := make([]bool, len(src))
		copy(out, src)
		return out
	}
	out := make([]bool, len(src))
	copy(out, src)
	seen := make([]uint8, len(src))
	queue := make([]int, 0, 256)
	component := make([]int, 0, 256)
	for i := range out {
		if !out[i] || seen[i] == 1 {
			continue
		}
		queue = queue[:0]
		component = component[:0]
		queue = append(queue, i)
		seen[i] = 1
		for head := 0; head < len(queue); head++ {
			p := queue[head]
			component = append(component, p)
			x := p % w
			y := p / w
			for dy := -1; dy <= 1; dy++ {
				ny := y + dy
				if ny < 0 || ny >= h {
					continue
				}
				for dx := -1; dx <= 1; dx++ {
					if dx == 0 && dy == 0 {
						continue
					}
					nx := x + dx
					if nx < 0 || nx >= w {
						continue
					}
					ni := ny*w + nx
					if seen[ni] == 1 || !out[ni] {
						continue
					}
					seen[ni] = 1
					queue = append(queue, ni)
				}
			}
		}
		if len(component) < minSize {
			for _, p := range component {
				out[p] = false
			}
		}
	}
	return out
}
