package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/maax3v3/macoma/internal/color"
)

// Strategy constants for delimiter detection.
const (
	StrategyBorder = "border"
	StrategyColor  = "color"
)

// Config holds the parsed CLI arguments.
type Config struct {
	InPath                   string
	OutPath                  string
	DelimiterStrategy        string
	BorderDelimiterColor     color.RGBA
	BorderDelimiterTolerance float64
	ColorDelimiterTolerance  float64
	MaxColors                int
}

// Parse parses CLI arguments and returns a validated Config.
func Parse() (Config, error) {
	inPath := flag.String("in", "", "Path to input image (required, supports PNG, JPEG, WEBP)")
	outPath := flag.String("out", "", "Path to generated output image (required, must be .png)")
	strategy := flag.String("delimiter-strategy", StrategyColor, "Delimitation strategy: \"border\" (explicit border color) or \"color\" (neighbor color difference)")
	borderColor := flag.String("border-delimiter-color", "#000", "Hex color of the drawing delimiter lines (border strategy only, e.g. #000, #FF00FF)")
	borderTolerance := flag.Float64("border-delimiter-tolerance", 10, "Tolerance % for matching the border color, 0-100 (border strategy only)")
	colorTolerance := flag.Float64("color-delimiter-tolerance", 10, "Color difference threshold % from which neighbors are considered different sections, 0-100 (color strategy only)")
	maxColors := flag.Int("max-colors", 10, "Maximum number of colors in the magic drawing (0 = unlimited)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: macoma [options]\n\nOptions:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n  macoma --in=drawing.png --out=coloring.png --delimiter-strategy=color --color-delimiter-tolerance=10 --max-colors=15\n")
	}

	flag.Parse()

	if *inPath == "" {
		return Config{}, fmt.Errorf("--in is required")
	}
	if *outPath == "" {
		return Config{}, fmt.Errorf("--out is required")
	}
	if ext := strings.ToLower(filepath.Ext(*outPath)); ext != ".png" {
		return Config{}, fmt.Errorf("--out must be a .png file, got %q", ext)
	}
	if *strategy != StrategyBorder && *strategy != StrategyColor {
		return Config{}, fmt.Errorf("--delimiter-strategy must be %q or %q, got %q", StrategyBorder, StrategyColor, *strategy)
	}
	if *borderTolerance < 0 || *borderTolerance > 100 {
		return Config{}, fmt.Errorf("--border-delimiter-tolerance must be between 0 and 100, got %f", *borderTolerance)
	}
	if *colorTolerance < 0 || *colorTolerance > 100 {
		return Config{}, fmt.Errorf("--color-delimiter-tolerance must be between 0 and 100, got %f", *colorTolerance)
	}
	if *maxColors < 0 {
		return Config{}, fmt.Errorf("--max-colors must be >= 0, got %d", *maxColors)
	}

	dc, err := color.ParseHex(*borderColor)
	if err != nil {
		return Config{}, fmt.Errorf("--border-delimiter-color: %w", err)
	}

	return Config{
		InPath:                   *inPath,
		OutPath:                  *outPath,
		DelimiterStrategy:        *strategy,
		BorderDelimiterColor:     dc,
		BorderDelimiterTolerance: *borderTolerance,
		ColorDelimiterTolerance:  *colorTolerance,
		MaxColors:                *maxColors,
	}, nil
}
