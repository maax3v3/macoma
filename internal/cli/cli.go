package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/maax3v3/macoma/v2/internal/color"
)

// Strategy constants for delimiter detection.
const (
	StrategyBorder   = "border"
	StrategyColor    = "color"
	StrategyGradient = "gradient"
)

// Config holds the parsed CLI arguments.
type Config struct {
	InPath                   string
	OutPath                  string
	DelimiterStrategy        string
	BorderDelimiterColor     color.RGBA
	BorderDelimiterTolerance float64
	ColorDelimiterTolerance  float64
	GradientBlurSigma        float64
	GradientLowThreshold     float64
	GradientHighThreshold    float64
	GradientCloseRadius      int
	GradientMinComponentSize int
	MaxColors                int
	LabelReadability         bool
}

// Parse parses CLI arguments and returns a validated Config.
func Parse() (Config, error) {
	inPath := flag.String("in", "", "Path to input image (required, supports PNG, JPEG, WEBP)")
	outPath := flag.String("out", "", "Path to generated output image (required, must be .png)")
	strategy := flag.String("delimiter-strategy", StrategyColor, "Delimitation strategy: \"border\" (explicit border color), \"color\" (neighbor color difference), or \"gradient\" (edge pipeline)")
	borderColor := flag.String("border-delimiter-color", "#000", "Hex color of the drawing delimiter lines (border strategy only, e.g. #000, #FF00FF)")
	borderTolerance := flag.Float64("border-delimiter-tolerance", 10, "Tolerance % for matching the border color, 0-100 (border strategy only)")
	colorTolerance := flag.Float64("color-delimiter-tolerance", 10, "Color difference threshold % from which neighbors are considered different sections, 0-100 (color strategy only)")
	gradientBlurSigma := flag.Float64("gradient-blur-sigma", 1.2, "Gaussian blur sigma for gradient strategy (>0)")
	gradientLowThreshold := flag.Float64("gradient-low-threshold", 0.08, "Weak edge threshold for gradient strategy, 0-1")
	gradientHighThreshold := flag.Float64("gradient-high-threshold", 0.20, "Strong edge threshold for gradient strategy, 0-1")
	gradientCloseRadius := flag.Int("gradient-close-radius", 1, "Morphological close radius in pixels for gradient strategy (>=0)")
	gradientMinComponentSize := flag.Int("gradient-min-component-size", 24, "Drop edge components smaller than this size for gradient strategy (>=0)")
	maxColors := flag.Int("max-colors", 10, "Maximum number of colors in the magic drawing (0 = unlimited)")
	labelReadability := flag.Bool("label-readability", true, "Enable readability enhancements for zone labels (white outline + safer placement)")

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
	if *strategy != StrategyBorder && *strategy != StrategyColor && *strategy != StrategyGradient {
		return Config{}, fmt.Errorf("--delimiter-strategy must be %q, %q or %q, got %q", StrategyBorder, StrategyColor, StrategyGradient, *strategy)
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
	if *gradientBlurSigma <= 0 {
		return Config{}, fmt.Errorf("--gradient-blur-sigma must be > 0, got %f", *gradientBlurSigma)
	}
	if *gradientLowThreshold < 0 || *gradientLowThreshold > 1 {
		return Config{}, fmt.Errorf("--gradient-low-threshold must be between 0 and 1, got %f", *gradientLowThreshold)
	}
	if *gradientHighThreshold < 0 || *gradientHighThreshold > 1 {
		return Config{}, fmt.Errorf("--gradient-high-threshold must be between 0 and 1, got %f", *gradientHighThreshold)
	}
	if *gradientLowThreshold > *gradientHighThreshold {
		return Config{}, fmt.Errorf("--gradient-low-threshold must be <= --gradient-high-threshold")
	}
	if *gradientCloseRadius < 0 {
		return Config{}, fmt.Errorf("--gradient-close-radius must be >= 0, got %d", *gradientCloseRadius)
	}
	if *gradientMinComponentSize < 0 {
		return Config{}, fmt.Errorf("--gradient-min-component-size must be >= 0, got %d", *gradientMinComponentSize)
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
		GradientBlurSigma:        *gradientBlurSigma,
		GradientLowThreshold:     *gradientLowThreshold,
		GradientHighThreshold:    *gradientHighThreshold,
		GradientCloseRadius:      *gradientCloseRadius,
		GradientMinComponentSize: *gradientMinComponentSize,
		MaxColors:                *maxColors,
		LabelReadability:         *labelReadability,
	}, nil
}
