package detection

import (
	"image"
	"image/color"
	"testing"

	mcol "github.com/maax3v3/macoma/internal/color"
)

// solidImage is a minimal image.Image for testing.
type solidImage struct {
	w, h int
	data []color.RGBA
}

func (s *solidImage) ColorModel() color.Model { return color.RGBAModel }
func (s *solidImage) Bounds() image.Rectangle { return image.Rect(0, 0, s.w, s.h) }
func (s *solidImage) At(x, y int) color.Color { return s.data[y*s.w+x] }

func newSolidImage(w, h int, fill color.RGBA) *solidImage {
	data := make([]color.RGBA, w*h)
	for i := range data {
		data[i] = fill
	}
	return &solidImage{w: w, h: h, data: data}
}

func TestDetect_AllDelimiter(t *testing.T) {
	img := newSolidImage(10, 10, color.RGBA{0, 0, 0, 255})
	dm := Detect(img, mcol.RGBA{R: 0, G: 0, B: 0, A: 255}, 1)

	if dm.Width != 10 || dm.Height != 10 {
		t.Fatalf("dimensions: got %dx%d, want 10x10", dm.Width, dm.Height)
	}
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			if !dm.At(x, y) {
				t.Errorf("pixel (%d,%d) should be delimiter", x, y)
			}
		}
	}
}

func TestDetect_NoDelimiter(t *testing.T) {
	img := newSolidImage(10, 10, color.RGBA{255, 255, 255, 255})
	dm := Detect(img, mcol.RGBA{R: 0, G: 0, B: 0, A: 255}, 1)

	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			if dm.At(x, y) {
				t.Errorf("pixel (%d,%d) should not be delimiter", x, y)
			}
		}
	}
}

func TestDetect_MixedWithCross(t *testing.T) {
	w, h := 10, 10
	img := newSolidImage(w, h, color.RGBA{255, 0, 0, 255})
	// Draw a black cross at row 5 and col 5
	for x := 0; x < w; x++ {
		img.data[5*w+x] = color.RGBA{0, 0, 0, 255}
	}
	for y := 0; y < h; y++ {
		img.data[y*w+5] = color.RGBA{0, 0, 0, 255}
	}

	dm := Detect(img, mcol.RGBA{R: 0, G: 0, B: 0, A: 255}, 1)

	// Cross pixels should be delimiters
	for x := 0; x < w; x++ {
		if !dm.At(x, 5) {
			t.Errorf("(%d,5) should be delimiter", x)
		}
	}
	for y := 0; y < h; y++ {
		if !dm.At(5, y) {
			t.Errorf("(5,%d) should be delimiter", y)
		}
	}
	// Non-cross red pixels should not be delimiters
	if dm.At(0, 0) {
		t.Error("(0,0) should not be delimiter")
	}
	if dm.At(9, 9) {
		t.Error("(9,9) should not be delimiter")
	}
}

func TestDetect_ToleranceExpands(t *testing.T) {
	// Near-black color should be detected with higher tolerance
	nearBlack := color.RGBA{20, 20, 20, 255}
	img := newSolidImage(5, 5, nearBlack)

	t.Run("low tolerance misses near-black", func(t *testing.T) {
		dm := Detect(img, mcol.RGBA{R: 0, G: 0, B: 0, A: 255}, 0)
		if dm.At(0, 0) {
			t.Error("should not detect near-black at 0% tolerance")
		}
	})

	t.Run("higher tolerance catches near-black", func(t *testing.T) {
		dm := Detect(img, mcol.RGBA{R: 0, G: 0, B: 0, A: 255}, 10)
		if !dm.At(0, 0) {
			t.Error("should detect near-black at 10% tolerance")
		}
	})
}

func TestBorderDelimiter_ImplementsInterface(t *testing.T) {
	var _ Delimiter = (*BorderDelimiter)(nil)
}

func TestColorDelimiter_ImplementsInterface(t *testing.T) {
	var _ Delimiter = (*ColorDelimiter)(nil)
}

func TestColorDelimiter_UniformImage(t *testing.T) {
	// A uniform-color image should have no delimiters at any tolerance > 0
	img := newSolidImage(10, 10, color.RGBA{100, 100, 100, 255})
	cd := &ColorDelimiter{TolerancePct: 5}
	dm := cd.Detect(img)

	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			if dm.At(x, y) {
				t.Errorf("pixel (%d,%d) should not be delimiter in uniform image", x, y)
			}
		}
	}
}

func TestColorDelimiter_TwoHalves(t *testing.T) {
	// Left half red, right half blue — boundary at x=20.
	// Image is wide enough (40px) so that deep-interior pixels survive
	// the extended detection radius (distance 3) + morphological closing.
	w, h := 40, 10
	img := newSolidImage(w, h, color.RGBA{255, 0, 0, 255})
	for y := 0; y < h; y++ {
		for x := 20; x < w; x++ {
			img.data[y*w+x] = color.RGBA{0, 0, 255, 255}
		}
	}

	cd := &ColorDelimiter{TolerancePct: 5}
	dm := cd.Detect(img)

	// Pixels at the boundary (x=19 red side, x=20 blue side) should be delimiters
	for y := 0; y < h; y++ {
		if !dm.At(19, y) {
			t.Errorf("pixel (19,%d) should be delimiter (red side of boundary)", y)
		}
		if !dm.At(20, y) {
			t.Errorf("pixel (20,%d) should be delimiter (blue side of boundary)", y)
		}
	}

	// Interior pixels far from boundary should not be delimiters
	if dm.At(0, 5) {
		t.Error("pixel (0,5) should not be delimiter (deep red interior)")
	}
	if dm.At(39, 5) {
		t.Error("pixel (39,5) should not be delimiter (deep blue interior)")
	}
}

func TestColorDelimiter_HighTolerance(t *testing.T) {
	// With very high tolerance (100%), even very different neighbors won't be delimiters
	w, h := 10, 1
	img := newSolidImage(w, h, color.RGBA{0, 0, 0, 255})
	img.data[5] = color.RGBA{255, 255, 255, 255}

	cd := &ColorDelimiter{TolerancePct: 100}
	dm := cd.Detect(img)

	for x := 0; x < w; x++ {
		if dm.At(x, 0) {
			t.Errorf("pixel (%d,0) should not be delimiter at 100%% tolerance", x)
		}
	}
}

func TestColorDelimiter_ZeroTolerance(t *testing.T) {
	// With 0% tolerance, any difference at all marks a delimiter.
	// Use a wide image so the morph close doesn't affect deep-interior pixels.
	w, h := 30, 1
	img := newSolidImage(w, h, color.RGBA{100, 100, 100, 255})
	img.data[15] = color.RGBA{101, 100, 100, 255} // tiny difference at center

	cd := &ColorDelimiter{TolerancePct: 0}
	dm := cd.Detect(img)

	// The different pixel and its neighbors should be delimiters
	if !dm.At(15, 0) {
		t.Error("pixel (15,0) should be delimiter at 0% tolerance")
	}
	if !dm.At(14, 0) {
		t.Error("pixel (14,0) should be delimiter at 0% tolerance")
	}
	// Far-away pixel should not be delimiter
	if dm.At(0, 0) {
		t.Error("pixel (0,0) should not be delimiter (far from difference)")
	}
}

func TestMap_At(t *testing.T) {
	dm := &Map{
		Width:  3,
		Height: 2,
		IsDelimiter: []bool{
			true, false, true,
			false, true, false,
		},
	}
	tests := []struct {
		x, y int
		want bool
	}{
		{0, 0, true}, {1, 0, false}, {2, 0, true},
		{0, 1, false}, {1, 1, true}, {2, 1, false},
	}
	for _, tt := range tests {
		if got := dm.At(tt.x, tt.y); got != tt.want {
			t.Errorf("At(%d,%d) = %v, want %v", tt.x, tt.y, got, tt.want)
		}
	}
}
