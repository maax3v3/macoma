package zone

import (
	"image"
	"image/color"
	"testing"

	mcol "github.com/maax3v3/macoma/internal/color"
	"github.com/maax3v3/macoma/internal/detection"
)

func TestCentroid(t *testing.T) {
	tests := []struct {
		name   string
		pixels []image.Point
		want   image.Point
	}{
		{
			name:   "empty zone",
			pixels: nil,
			want:   image.Point{},
		},
		{
			name:   "single pixel",
			pixels: []image.Point{{5, 10}},
			want:   image.Point{5, 10},
		},
		{
			name:   "symmetric square",
			pixels: []image.Point{{0, 0}, {2, 0}, {0, 2}, {2, 2}},
			want:   image.Point{1, 1},
		},
		{
			name:   "horizontal line",
			pixels: []image.Point{{0, 0}, {1, 0}, {2, 0}, {3, 0}, {4, 0}},
			want:   image.Point{2, 0},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			z := &Zone{ID: 0, Pixels: tt.pixels}
			got := z.Centroid()
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInteriorPoint_EmptyZone(t *testing.T) {
	z := &Zone{ID: 0}
	got := z.InteriorPoint()
	if got != (image.Point{}) {
		t.Errorf("expected zero point for empty zone, got %v", got)
	}
}

func TestInteriorPoint_ConvexZone(t *testing.T) {
	// A filled 50x50 square: centroid is (24,24), well inside with margin
	var pixels []image.Point
	for y := 0; y < 50; y++ {
		for x := 0; x < 50; x++ {
			pixels = append(pixels, image.Point{X: x, Y: y})
		}
	}
	z := &Zone{ID: 0, Pixels: pixels}
	pt := z.InteriorPoint()

	// Must be a zone pixel
	found := false
	for _, p := range pixels {
		if p == pt {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("InteriorPoint %v is not a zone pixel", pt)
	}
}

func TestInteriorPoint_ConcaveZone(t *testing.T) {
	// Build an L-shaped zone. The centroid of an L is often outside the shape.
	// L shape: bottom row (y=20..39, x=0..39) + left column (y=0..19, x=0..19)
	members := make(map[image.Point]struct{})
	var pixels []image.Point

	// Left column
	for y := 0; y < 20; y++ {
		for x := 0; x < 20; x++ {
			p := image.Point{X: x, Y: y}
			if _, ok := members[p]; !ok {
				members[p] = struct{}{}
				pixels = append(pixels, p)
			}
		}
	}
	// Bottom row
	for y := 20; y < 40; y++ {
		for x := 0; x < 40; x++ {
			p := image.Point{X: x, Y: y}
			if _, ok := members[p]; !ok {
				members[p] = struct{}{}
				pixels = append(pixels, p)
			}
		}
	}

	z := &Zone{ID: 0, Pixels: pixels}
	pt := z.InteriorPoint()

	// The returned point must be inside the zone
	if _, ok := members[pt]; !ok {
		t.Fatalf("InteriorPoint %v is not inside the zone", pt)
	}
}

func TestInteriorPoint_ThinZone(t *testing.T) {
	// A 1-pixel-wide horizontal line — no pixel can have margin, should still return a valid point
	var pixels []image.Point
	for x := 0; x < 30; x++ {
		pixels = append(pixels, image.Point{X: x, Y: 0})
	}
	z := &Zone{ID: 0, Pixels: pixels}
	pt := z.InteriorPoint()

	members := make(map[image.Point]struct{}, len(pixels))
	for _, p := range pixels {
		members[p] = struct{}{}
	}
	if _, ok := members[pt]; !ok {
		t.Fatalf("InteriorPoint %v is not inside the thin zone", pt)
	}
}

func TestFindZones_SingleZone(t *testing.T) {
	// 5x5 grid with no delimiters → one zone with 25 pixels
	dm := &detection.Map{
		Width:       5,
		Height:      5,
		IsDelimiter: make([]bool, 25),
	}
	zones, labels := FindZones(dm)

	if len(zones) != 1 {
		t.Fatalf("expected 1 zone, got %d", len(zones))
	}
	if len(zones[0].Pixels) != 25 {
		t.Errorf("expected 25 pixels in zone, got %d", len(zones[0].Pixels))
	}
	// All labels should be 0
	for i, l := range labels {
		if l != 0 {
			t.Errorf("label[%d] = %d, want 0", i, l)
		}
	}
}

func TestFindZones_FourQuadrants(t *testing.T) {
	// 5x5 grid split by a cross of delimiters at row 2 and col 2
	w, h := 5, 5
	delim := make([]bool, w*h)
	for x := 0; x < w; x++ {
		delim[2*w+x] = true // row 2
	}
	for y := 0; y < h; y++ {
		delim[y*w+2] = true // col 2
	}
	dm := &detection.Map{Width: w, Height: h, IsDelimiter: delim}

	zones, labels := FindZones(dm)

	if len(zones) != 4 {
		t.Fatalf("expected 4 zones, got %d", len(zones))
	}

	// Each zone should have 4 pixels (2x2 corners)
	for i, z := range zones {
		if len(z.Pixels) != 4 {
			t.Errorf("zone %d: expected 4 pixels, got %d", i, len(z.Pixels))
		}
	}

	// Delimiter pixels should have label -1
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			idx := y*w + x
			if delim[idx] && labels[idx] != -1 {
				t.Errorf("delimiter pixel (%d,%d) has label %d, want -1", x, y, labels[idx])
			}
		}
	}
}

func TestFindZones_AllDelimiter(t *testing.T) {
	w, h := 3, 3
	delim := make([]bool, w*h)
	for i := range delim {
		delim[i] = true
	}
	dm := &detection.Map{Width: w, Height: h, IsDelimiter: delim}

	zones, labels := FindZones(dm)

	if len(zones) != 0 {
		t.Errorf("expected 0 zones, got %d", len(zones))
	}
	for i, l := range labels {
		if l != -1 {
			t.Errorf("label[%d] = %d, want -1", i, l)
		}
	}
}

func TestFindZones_DiagonalNotConnected(t *testing.T) {
	// 3x3 grid, delimiter everywhere except (0,0) and (2,2)
	// Since we use 4-connectivity, these are separate zones.
	w, h := 3, 3
	delim := make([]bool, w*h)
	for i := range delim {
		delim[i] = true
	}
	delim[0*w+0] = false // (0,0)
	delim[2*w+2] = false // (2,2)

	dm := &detection.Map{Width: w, Height: h, IsDelimiter: delim}
	zones, _ := FindZones(dm)

	if len(zones) != 2 {
		t.Fatalf("expected 2 zones (diagonal pixels not 4-connected), got %d", len(zones))
	}
}

// testImage implements image.Image for ComputeZoneColors testing.
type testImage struct {
	w, h int
	data map[image.Point]color.RGBA
}

func (ti *testImage) ColorModel() color.Model { return color.RGBAModel }
func (ti *testImage) Bounds() image.Rectangle  { return image.Rect(0, 0, ti.w, ti.h) }
func (ti *testImage) At(x, y int) color.Color {
	if c, ok := ti.data[image.Point{X: x, Y: y}]; ok {
		return c
	}
	return color.RGBA{0, 0, 0, 255}
}

func TestComputeZoneColors(t *testing.T) {
	// Two zones: zone 0 is all red, zone 1 is all blue
	zones := []Zone{
		{ID: 0, Pixels: []image.Point{{0, 0}, {1, 0}}},
		{ID: 1, Pixels: []image.Point{{3, 0}, {4, 0}}},
	}
	img := &testImage{
		w: 5, h: 1,
		data: map[image.Point]color.RGBA{
			{0, 0}: {255, 0, 0, 255},
			{1, 0}: {255, 0, 0, 255},
			{3, 0}: {0, 0, 255, 255},
			{4, 0}: {0, 0, 255, 255},
		},
	}

	zc := ComputeZoneColors(zones, img)

	if len(zc.Colors) != 2 {
		t.Fatalf("expected 2 colors, got %d", len(zc.Colors))
	}
	if zc.Colors[0] != (mcol.RGBA{255, 0, 0, 255}) {
		t.Errorf("zone 0 color: got %+v, want red", zc.Colors[0])
	}
	if zc.Colors[1] != (mcol.RGBA{0, 0, 255, 255}) {
		t.Errorf("zone 1 color: got %+v, want blue", zc.Colors[1])
	}
}

func TestComputeZoneColors_MixedPixels(t *testing.T) {
	// Zone with black (0,0,0) and white (255,255,255) pixels → mean is (128,128,128)
	zones := []Zone{
		{ID: 0, Pixels: []image.Point{{0, 0}, {1, 0}}},
	}
	img := &testImage{
		w: 2, h: 1,
		data: map[image.Point]color.RGBA{
			{0, 0}: {1, 1, 1, 255},
			{1, 0}: {255, 255, 255, 255},
		},
	}

	zc := ComputeZoneColors(zones, img)
	c := zc.Colors[0]
	if c.R != 128 || c.G != 128 || c.B != 128 {
		t.Errorf("expected ~{128,128,128}, got %+v", c)
	}
}
