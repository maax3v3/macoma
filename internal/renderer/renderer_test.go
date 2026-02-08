package renderer

import (
	"image"
	"image/color"
	"testing"

	"github.com/maax3v3/macoma/internal/aggregation"
	mcol "github.com/maax3v3/macoma/internal/color"
	"github.com/maax3v3/macoma/internal/detection"
	"github.com/maax3v3/macoma/internal/zone"
)

func TestBitmapFont_MeasureString(t *testing.T) {
	bf := NewBitmapFont()

	tests := []struct {
		name       string
		text       string
		size       int
		wantW, wantH int
	}{
		{
			name: "empty string",
			text: "", size: 14,
			wantW: 0, wantH: 0,
		},
		{
			name: "single digit scale 1",
			text: "5", size: 7,
			wantW: 5, wantH: 7,
		},
		{
			name: "two digits scale 1",
			text: "12", size: 7,
			// 2 * (5*1) + (2-1)*1 = 11
			wantW: 11, wantH: 7,
		},
		{
			name: "single digit scale 2",
			text: "5", size: 14,
			wantW: 10, wantH: 14,
		},
		{
			name: "size smaller than glyph height uses scale 1",
			text: "0", size: 3,
			wantW: 5, wantH: 7,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, h := bf.MeasureString(tt.text, tt.size)
			if w != tt.wantW || h != tt.wantH {
				t.Errorf("MeasureString(%q, %d) = (%d, %d), want (%d, %d)",
					tt.text, tt.size, w, h, tt.wantW, tt.wantH)
			}
		})
	}
}

func TestBitmapFont_DrawString_WritesPixels(t *testing.T) {
	bf := NewBitmapFont()
	img := image.NewRGBA(image.Rect(0, 0, 50, 50))
	// Fill with white
	for y := 0; y < 50; y++ {
		for x := 0; x < 50; x++ {
			img.SetRGBA(x, y, color.RGBA{255, 255, 255, 255})
		}
	}

	bf.DrawString(img, "1", 25, 25, color.Black, 7)

	// At least some pixels should now be black
	blackCount := 0
	for y := 0; y < 50; y++ {
		for x := 0; x < 50; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			if r == 0 && g == 0 && b == 0 {
				blackCount++
			}
		}
	}
	if blackCount == 0 {
		t.Error("DrawString did not write any pixels")
	}
}

func TestBitmapFont_DrawString_UnknownGlyph(t *testing.T) {
	bf := NewBitmapFont()
	img := image.NewRGBA(image.Rect(0, 0, 50, 50))
	for y := 0; y < 50; y++ {
		for x := 0; x < 50; x++ {
			img.SetRGBA(x, y, color.RGBA{255, 255, 255, 255})
		}
	}

	// Drawing a character with no glyph should not panic
	bf.DrawString(img, "X", 25, 25, color.Black, 7)

	// No black pixels expected (unknown glyph is skipped)
	for y := 0; y < 50; y++ {
		for x := 0; x < 50; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			if r == 0 && g == 0 && b == 0 {
				t.Fatal("unexpected black pixel for unknown glyph")
			}
		}
	}
}

func TestBitmapFont_ImplementsFontRenderer(t *testing.T) {
	var _ FontRenderer = (*BitmapFont)(nil)
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.LegendPadding <= 0 || cfg.LegendCircleSize <= 0 ||
		cfg.LegendSpacing <= 0 || cfg.LegendMargin <= 0 {
		t.Errorf("default config has non-positive values: %+v", cfg)
	}
}

func TestRender_OutputDimensions(t *testing.T) {
	// Build a minimal 20x20 image with a vertical delimiter at x=10
	srcW, srcH := 20, 20
	src := image.NewRGBA(image.Rect(0, 0, srcW, srcH))
	delim := make([]bool, srcW*srcH)
	for y := 0; y < srcH; y++ {
		for x := 0; x < srcW; x++ {
			if x == 10 {
				src.SetRGBA(x, y, color.RGBA{0, 0, 0, 255})
				delim[y*srcW+x] = true
			} else if x < 10 {
				src.SetRGBA(x, y, color.RGBA{255, 0, 0, 255})
			} else {
				src.SetRGBA(x, y, color.RGBA{0, 0, 255, 255})
			}
		}
	}
	dm := &detection.Map{Width: srcW, Height: srcH, IsDelimiter: delim}
	zones, labels := zone.FindZones(dm)
	zc := zone.ComputeZoneColors(zones, src)
	cm := aggregation.ReduceColors(zc.Colors, 0)
	font := NewBitmapFont()
	cfg := DefaultConfig()

	out := Render(src, dm, zones, labels, cm, font, cfg)

	if out.Bounds().Dx() != srcW {
		t.Errorf("output width: got %d, want %d", out.Bounds().Dx(), srcW)
	}
	if out.Bounds().Dy() <= srcH {
		t.Errorf("output height should exceed source height (legend), got %d", out.Bounds().Dy())
	}
}

func TestRender_DelimiterPixelsPreserved(t *testing.T) {
	srcW, srcH := 10, 10
	src := image.NewRGBA(image.Rect(0, 0, srcW, srcH))
	delim := make([]bool, srcW*srcH)

	black := color.RGBA{0, 0, 0, 255}
	white := color.RGBA{255, 255, 255, 255}

	// Single delimiter pixel at (5,5)
	for y := 0; y < srcH; y++ {
		for x := 0; x < srcW; x++ {
			src.SetRGBA(x, y, white)
		}
	}
	src.SetRGBA(5, 5, black)
	delim[5*srcW+5] = true

	dm := &detection.Map{Width: srcW, Height: srcH, IsDelimiter: delim}
	zones, labels := zone.FindZones(dm)
	zc := zone.ComputeZoneColors(zones, src)
	cm := aggregation.ReduceColors(zc.Colors, 0)
	font := NewBitmapFont()
	cfg := DefaultConfig()

	out := Render(src, dm, zones, labels, cm, font, cfg)

	r, g, b, _ := out.At(5, 5).RGBA()
	if r != 0 || g != 0 || b != 0 {
		t.Errorf("delimiter pixel (5,5) not preserved: got (%d,%d,%d)", r>>8, g>>8, b>>8)
	}
}

func TestRender_FillerPixelsWhited(t *testing.T) {
	srcW, srcH := 10, 1
	src := image.NewRGBA(image.Rect(0, 0, srcW, srcH))
	delim := make([]bool, srcW*srcH)

	// All red filler, no delimiters
	for x := 0; x < srcW; x++ {
		src.SetRGBA(x, 0, color.RGBA{255, 0, 0, 255})
	}

	dm := &detection.Map{Width: srcW, Height: srcH, IsDelimiter: delim}
	zones, labels := zone.FindZones(dm)
	zc := zone.ComputeZoneColors(zones, src)
	cm := aggregation.ReduceColors(zc.Colors, 0)
	font := NewBitmapFont()
	cfg := DefaultConfig()

	out := Render(src, dm, zones, labels, cm, font, cfg)

	// Filler pixels in the drawing area (row 0) should be white (possibly
	// with number text drawn on top, but most should be white).
	whiteCount := 0
	for x := 0; x < srcW; x++ {
		r, g, b, _ := out.At(x, 0).RGBA()
		if r == 0xFFFF && g == 0xFFFF && b == 0xFFFF {
			whiteCount++
		}
	}
	if whiteCount == 0 {
		t.Error("expected at least some filler pixels to be white")
	}
}

func TestRender_NoZones(t *testing.T) {
	// All delimiter → no zones → should not panic, no legend
	srcW, srcH := 5, 5
	src := image.NewRGBA(image.Rect(0, 0, srcW, srcH))
	delim := make([]bool, srcW*srcH)
	for i := range delim {
		delim[i] = true
		src.SetRGBA(i%srcW, i/srcW, color.RGBA{0, 0, 0, 255})
	}

	dm := &detection.Map{Width: srcW, Height: srcH, IsDelimiter: delim}
	zones, labels := zone.FindZones(dm)
	cm := &aggregation.ColorMap{}
	font := NewBitmapFont()
	cfg := DefaultConfig()

	out := Render(src, dm, zones, labels, cm, font, cfg)

	// No legend → output height should equal source height
	if out.Bounds().Dy() != srcH {
		t.Errorf("expected height %d (no legend), got %d", srcH, out.Bounds().Dy())
	}
}

func TestComputeFontSize(t *testing.T) {
	tests := []struct {
		name     string
		w, h     int
		numZones int
	}{
		{"small image", 100, 100, 5},
		{"large image", 2000, 2000, 10},
		{"many zones", 500, 500, 300},
		{"tiny image", 30, 30, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size := computeFontSize(tt.w, tt.h, tt.numZones)
			if size < 7 {
				t.Errorf("font size %d below minimum 7", size)
			}
			if size > 40 {
				t.Errorf("font size %d above maximum 40", size)
			}
		})
	}
}

func TestCalculateLegendHeight_NoEntries(t *testing.T) {
	cm := &aggregation.ColorMap{}
	font := NewBitmapFont()
	cfg := DefaultConfig()
	h := calculateLegendHeight(cm, font, cfg, 200)
	if h != 0 {
		t.Errorf("expected 0 legend height for no entries, got %d", h)
	}
}

func TestCalculateLegendHeight_WithEntries(t *testing.T) {
	cm := &aggregation.ColorMap{
		Entries: []aggregation.ColorEntry{
			{Number: 1, Color: mcol.RGBA{255, 0, 0, 255}},
			{Number: 2, Color: mcol.RGBA{0, 255, 0, 255}},
		},
	}
	font := NewBitmapFont()
	cfg := DefaultConfig()
	h := calculateLegendHeight(cm, font, cfg, 200)
	if h <= 0 {
		t.Errorf("expected positive legend height, got %d", h)
	}
}
