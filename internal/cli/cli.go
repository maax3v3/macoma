package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/maax3v3/macoma/internal/color"
)

// Config holds the parsed CLI arguments.
type Config struct {
	InPath             string
	OutPath            string
	DelimiterColor     color.RGBA
	DelimiterTolerance float64
	MaxColors          int
}

// Parse parses CLI arguments and returns a validated Config.
func Parse() (Config, error) {
	inPath := flag.String("in", "", "Path to input image (required, supports PNG, JPEG, WEBP)")
	outPath := flag.String("out", "", "Path to generated output image (required, must be .png)")
	delimiterColor := flag.String("delimiter-color", "#000", "Hex color of the drawing delimiter lines (e.g. #000, #FF00FF)")
	delimiterTolerance := flag.Float64("delimiter-tolerance", 10, "Tolerance percentage for the delimiter color (0-100)")
	maxColors := flag.Int("max-colors", 10, "Maximum number of colors in the magic drawing (0 = unlimited)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: macoma [options]\n\nOptions:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n  macoma --in=drawing.png --out=coloring.png --delimiter-color=#000 --delimiter-tolerance=10 --max-colors=15\n")
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
	if *delimiterTolerance < 0 || *delimiterTolerance > 100 {
		return Config{}, fmt.Errorf("--delimiter-tolerance must be between 0 and 100, got %f", *delimiterTolerance)
	}
	if *maxColors < 0 {
		return Config{}, fmt.Errorf("--max-colors must be >= 0, got %d", *maxColors)
	}

	dc, err := color.ParseHex(*delimiterColor)
	if err != nil {
		return Config{}, fmt.Errorf("--delimiter-color: %w", err)
	}

	return Config{
		InPath:             *inPath,
		OutPath:            *outPath,
		DelimiterColor:     dc,
		DelimiterTolerance: *delimiterTolerance,
		MaxColors:          *maxColors,
	}, nil
}
