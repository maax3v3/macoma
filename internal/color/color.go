package color

import (
	"fmt"
	"image/color"
	"math"
	"strings"
)

// RGBA represents a color with 8-bit RGBA components.
type RGBA struct {
	R, G, B, A uint8
}

// FromStdColor converts a standard library color to RGBA.
func FromStdColor(c color.Color) RGBA {
	r, g, b, a := c.RGBA()
	return RGBA{
		R: uint8(r >> 8),
		G: uint8(g >> 8),
		B: uint8(b >> 8),
		A: uint8(a >> 8),
	}
}

// ToStdColor converts RGBA to a standard library color.
func (c RGBA) ToStdColor() color.RGBA {
	return color.RGBA{R: c.R, G: c.G, B: c.B, A: c.A}
}

// ParseHex parses a hex color string like "#000", "#000000", "#FF00FF".
func ParseHex(s string) (RGBA, error) {
	s = strings.TrimPrefix(s, "#")
	var r, g, b uint8
	switch len(s) {
	case 3:
		_, err := fmt.Sscanf(s, "%1x%1x%1x", &r, &g, &b)
		if err != nil {
			return RGBA{}, fmt.Errorf("invalid hex color %q: %w", s, err)
		}
		r = r*16 + r
		g = g*16 + g
		b = b*16 + b
	case 6:
		_, err := fmt.Sscanf(s, "%02x%02x%02x", &r, &g, &b)
		if err != nil {
			return RGBA{}, fmt.Errorf("invalid hex color %q: %w", s, err)
		}
	default:
		return RGBA{}, fmt.Errorf("invalid hex color %q: must be 3 or 6 hex digits", s)
	}
	return RGBA{R: r, G: g, B: b, A: 255}, nil
}

// LAB represents a color in the CIELAB color space.
type LAB struct {
	L, A, B float64
}

// ToLAB converts an RGBA color to CIELAB.
func (c RGBA) ToLAB() LAB {
	// RGB to linear sRGB
	rLin := srgbToLinear(float64(c.R) / 255.0)
	gLin := srgbToLinear(float64(c.G) / 255.0)
	bLin := srgbToLinear(float64(c.B) / 255.0)

	// Linear sRGB to XYZ (D65 illuminant)
	x := 0.4124564*rLin + 0.3575761*gLin + 0.1804375*bLin
	y := 0.2126729*rLin + 0.7151522*gLin + 0.0721750*bLin
	z := 0.0193339*rLin + 0.1191920*gLin + 0.9503041*bLin

	// D65 reference white
	const xn, yn, zn = 0.95047, 1.00000, 1.08883

	// XYZ to LAB
	fx := labF(x / xn)
	fy := labF(y / yn)
	fz := labF(z / zn)

	l := 116.0*fy - 16.0
	a := 500.0 * (fx - fy)
	b := 200.0 * (fy - fz)

	return LAB{L: l, A: a, B: b}
}

func srgbToLinear(v float64) float64 {
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

func labF(t float64) float64 {
	const delta = 6.0 / 29.0
	if t > delta*delta*delta {
		return math.Cbrt(t)
	}
	return t/(3.0*delta*delta) + 4.0/29.0
}

// DistanceLAB computes the Euclidean distance in CIELAB space between two colors.
func DistanceLAB(a, b RGBA) float64 {
	la := a.ToLAB()
	lb := b.ToLAB()
	dl := la.L - lb.L
	da := la.A - lb.A
	db := la.B - lb.B
	return math.Sqrt(dl*dl + da*da + db*db)
}

// DistanceRGB computes the Euclidean distance in RGB space between two colors.
func DistanceRGB(a, b RGBA) float64 {
	dr := float64(a.R) - float64(b.R)
	dg := float64(a.G) - float64(b.G)
	db := float64(a.B) - float64(b.B)
	return math.Sqrt(dr*dr + dg*dg + db*db)
}

// WeightedMean computes the weighted mean of a set of colors.
// weights[i] corresponds to colors[i]. If weights is nil, equal weights are used.
func WeightedMean(colors []RGBA, weights []int) RGBA {
	if len(colors) == 0 {
		return RGBA{}
	}
	var totalR, totalG, totalB, totalA float64
	var totalW float64
	for i, c := range colors {
		w := 1.0
		if weights != nil {
			w = float64(weights[i])
		}
		totalR += float64(c.R) * w
		totalG += float64(c.G) * w
		totalB += float64(c.B) * w
		totalA += float64(c.A) * w
		totalW += w
	}
	if totalW == 0 {
		return RGBA{}
	}
	return RGBA{
		R: uint8(math.Round(totalR / totalW)),
		G: uint8(math.Round(totalG / totalW)),
		B: uint8(math.Round(totalB / totalW)),
		A: uint8(math.Round(totalA / totalW)),
	}
}

// IsLight returns true if the color is perceptually light (luminance > 0.5).
func (c RGBA) IsLight() bool {
	// Relative luminance formula
	rLin := srgbToLinear(float64(c.R) / 255.0)
	gLin := srgbToLinear(float64(c.G) / 255.0)
	bLin := srgbToLinear(float64(c.B) / 255.0)
	luminance := 0.2126*rLin + 0.7152*gLin + 0.0722*bLin
	return luminance > 0.5
}

// MaxRGBDistance is the maximum possible Euclidean distance in RGB space.
var MaxRGBDistance = math.Sqrt(255*255*3)
