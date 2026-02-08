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
func (s *solidImage) Bounds() image.Rectangle  { return image.Rect(0, 0, s.w, s.h) }
func (s *solidImage) At(x, y int) color.Color  { return s.data[y*s.w+x] }

func newSolidImage(w, h int, fill color.RGBA) *solidImage {
	data := make([]color.RGBA, w*h)
	for i := range data {
		data[i] = fill
	}
	return &solidImage{w: w, h: h, data: data}
}

func TestDetect_AllDelimiter(t *testing.T) {
	img := newSolidImage(10, 10, color.RGBA{0, 0, 0, 255})
	dm := Detect(img, mcol.RGBA{0, 0, 0, 255}, 1)

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
	dm := Detect(img, mcol.RGBA{0, 0, 0, 255}, 1)

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

	dm := Detect(img, mcol.RGBA{0, 0, 0, 255}, 1)

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
		dm := Detect(img, mcol.RGBA{0, 0, 0, 255}, 0)
		if dm.At(0, 0) {
			t.Error("should not detect near-black at 0% tolerance")
		}
	})

	t.Run("higher tolerance catches near-black", func(t *testing.T) {
		dm := Detect(img, mcol.RGBA{0, 0, 0, 255}, 10)
		if !dm.At(0, 0) {
			t.Error("should detect near-black at 10% tolerance")
		}
	})
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
