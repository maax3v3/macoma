// Package macoma converts colored drawings into magic colorings (color-by-number images).
//
// A magic coloring is a drawing stripped of its colors, with a number replacing
// each color zone, and a legend mapping each number to its original color.
//
// Usage as a library:
//
//	img, _ := macoma.LoadImage("drawing.png")
//	result, _ := macoma.Convert(img, macoma.DefaultOptions())
//	macoma.SavePNG("coloring.png", result)
//
// Or use the file-based convenience:
//
//	err := macoma.ConvertFile("drawing.png", "coloring.png", macoma.DefaultOptions())
package macoma

import (
	"fmt"
	"image"
	stdcolor "image/color"

	"github.com/maax3v3/macoma/internal/aggregation"
	"github.com/maax3v3/macoma/internal/color"
	"github.com/maax3v3/macoma/internal/detection"
	"github.com/maax3v3/macoma/internal/imaging"
	"github.com/maax3v3/macoma/internal/renderer"
	"github.com/maax3v3/macoma/internal/zone"
)

// Delimiter strategy constants.
const (
	StrategyBorder = "border" // Detect borders by matching a specific color.
	StrategyColor  = "color"  // Detect borders by color differences between neighbors.
)

// Options configures the magic coloring conversion.
type Options struct {
	// DelimiterStrategy selects how zones are delimited.
	// "border" matches a specific border color; "color" uses neighbor color
	// differences. Default: "color".
	DelimiterStrategy string

	// BorderDelimiterColor is the color of the delimiter lines.
	// Only used when DelimiterStrategy is "border".
	// Default: black (#000000).
	BorderDelimiterColor Color

	// BorderDelimiterTolerance is the tolerance percentage (0–100) for
	// matching the border color. Only used when DelimiterStrategy is "border".
	// Default: 10.
	BorderDelimiterTolerance float64

	// ColorDelimiterTolerance is the color difference threshold percentage
	// (0–100) from which two neighboring pixels are considered different
	// sections. Only used when DelimiterStrategy is "color".
	// Default: 10.
	ColorDelimiterTolerance float64

	// MaxColors is the maximum number of distinct colors in the output.
	// 0 means unlimited.
	// Default: 10.
	MaxColors int

	// Font is the font renderer used to draw numbers on the output image.
	// If nil, a built-in bitmap font is used.
	Font FontRenderer
}

// Color represents an RGBA color with 8-bit components.
type Color struct {
	R, G, B, A uint8
}

// FontRenderer is the interface for drawing text onto images.
// Implement this to provide a custom font (e.g., TTF rendering).
type FontRenderer interface {
	// DrawString draws text centered at (cx, cy) on the image with the
	// specified color and approximate height in pixels.
	DrawString(img *image.RGBA, text string, cx, cy int, col stdcolor.Color, size int)

	// MeasureString returns the approximate width and height of the text
	// at the given font size.
	MeasureString(text string, size int) (width, height int)
}

// DefaultOptions returns Options with sensible defaults.
func DefaultOptions() Options {
	return Options{
		DelimiterStrategy:        StrategyColor,
		BorderDelimiterColor:     Color{0, 0, 0, 255},
		BorderDelimiterTolerance: 10,
		ColorDelimiterTolerance:  10,
		MaxColors:                10,
	}
}

// ParseHexColor parses a hex color string like "#000", "#FF00FF".
func ParseHexColor(hex string) (Color, error) {
	c, err := color.ParseHex(hex)
	if err != nil {
		return Color{}, err
	}
	return Color{R: c.R, G: c.G, B: c.B, A: c.A}, nil
}

// LoadImage reads an image from disk. Supports PNG, JPEG, and WEBP.
func LoadImage(path string) (image.Image, error) {
	return imaging.Load(path)
}

// SavePNG writes an image to disk as PNG.
func SavePNG(path string, img image.Image) error {
	return imaging.SavePNG(path, img)
}

// Convert takes an input image and produces a magic coloring image.
// The returned image has the coloring zones with numbers and a legend
// appended at the bottom.
func Convert(img image.Image, opts Options) (*image.RGBA, error) {
	if img == nil {
		return nil, fmt.Errorf("input image is nil")
	}

	// Build the appropriate delimiter strategy
	delim := delimiterFromOpts(opts)

	// Detect delimiter pixels
	dm := delim.Detect(img)

	// Find zones via flood-fill
	zones, labels := zone.FindZones(dm)

	// Compute per-zone aggregated colors
	zoneColors := zone.ComputeZoneColors(zones, img)

	// Reduce colors if necessary
	cm := aggregation.ReduceColors(zoneColors.Colors, opts.MaxColors)

	// Resolve font
	font := resolveFont(opts.Font)

	// Render output image
	rcfg := renderer.DefaultConfig()
	scaleLegendConfig(&rcfg, img.Bounds())
	output := renderer.Render(img, dm, zones, labels, cm, font, rcfg)

	return output, nil
}

// ConvertFile is a convenience that loads an image from inPath, converts it,
// and saves the result as PNG to outPath.
func ConvertFile(inPath, outPath string, opts Options) error {
	img, err := LoadImage(inPath)
	if err != nil {
		return fmt.Errorf("loading image: %w", err)
	}

	result, err := Convert(img, opts)
	if err != nil {
		return fmt.Errorf("converting: %w", err)
	}

	if err := SavePNG(outPath, result); err != nil {
		return fmt.Errorf("saving output: %w", err)
	}

	return nil
}

// resolveFont returns a renderer.FontRenderer, using the built-in bitmap font
// if the user did not provide one.
func resolveFont(f FontRenderer) renderer.FontRenderer {
	if f != nil {
		return &fontAdapter{f}
	}
	return renderer.NewBitmapFont()
}

// fontAdapter adapts the public FontRenderer interface to the internal one.
type fontAdapter struct {
	f FontRenderer
}

func (a *fontAdapter) DrawString(img *image.RGBA, text string, cx, cy int, col stdcolor.Color, size int) {
	a.f.DrawString(img, text, cx, cy, col, size)
}

func (a *fontAdapter) MeasureString(text string, size int) (int, int) {
	return a.f.MeasureString(text, size)
}

// delimiterFromOpts builds the appropriate Delimiter from public Options.
func delimiterFromOpts(opts Options) detection.Delimiter {
	if opts.DelimiterStrategy == StrategyBorder {
		return &detection.BorderDelimiter{
			Color: color.RGBA{
				R: opts.BorderDelimiterColor.R,
				G: opts.BorderDelimiterColor.G,
				B: opts.BorderDelimiterColor.B,
				A: opts.BorderDelimiterColor.A,
			},
			TolerancePct: opts.BorderDelimiterTolerance,
		}
	}
	return &detection.ColorDelimiter{
		TolerancePct: opts.ColorDelimiterTolerance,
	}
}

func scaleLegendConfig(cfg *renderer.Config, bounds image.Rectangle) {
	w := bounds.Dx()
	if w > 1000 {
		cfg.LegendCircleSize = 50
		cfg.LegendSpacing = 25
		cfg.LegendPadding = 30
		cfg.LegendMargin = 30
	} else if w > 500 {
		cfg.LegendCircleSize = 36
		cfg.LegendSpacing = 18
		cfg.LegendPadding = 24
		cfg.LegendMargin = 24
	}
}
