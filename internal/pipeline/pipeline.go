package pipeline

import (
	"fmt"
	"image"

	"github.com/maax3v3/macoma/internal/aggregation"
	"github.com/maax3v3/macoma/internal/cli"
	"github.com/maax3v3/macoma/internal/detection"
	"github.com/maax3v3/macoma/internal/imaging"
	"github.com/maax3v3/macoma/internal/renderer"
	"github.com/maax3v3/macoma/internal/zone"
)

// Run executes the full macoma pipeline with the given configuration.
func Run(cfg cli.Config, font renderer.FontRenderer) error {
	// Step 1: Load input image
	fmt.Printf("Loading image: %s\n", cfg.InPath)
	img, err := imaging.Load(cfg.InPath)
	if err != nil {
		return fmt.Errorf("loading image: %w", err)
	}
	fmt.Printf("Image loaded: %dx%d\n", img.Bounds().Dx(), img.Bounds().Dy())

	// Step 2: Detect delimiter pixels
	fmt.Println("Detecting delimiter pixels...")
	delim := delimiterFromConfig(cfg)
	dm := delim.Detect(img)
	delimCount := countDelimiters(dm)
	fmt.Printf("Delimiter pixels: %d / %d (%.1f%%)\n",
		delimCount, dm.Width*dm.Height,
		float64(delimCount)/float64(dm.Width*dm.Height)*100)

	// Step 3: Find zones via flood-fill
	fmt.Println("Finding zones...")
	zones, labels := zone.FindZones(dm)
	fmt.Printf("Zones found: %d\n", len(zones))

	// Step 4: Compute per-zone aggregated colors
	fmt.Println("Computing zone colors...")
	zoneColors := zone.ComputeZoneColors(zones, img)
	fmt.Printf("Zone colors computed\n")

	// Step 5: Reduce colors if necessary
	fmt.Println("Reducing colors...")
	cm := aggregation.ReduceColors(zoneColors.Colors, cfg.MaxColors)
	fmt.Printf("Distinct colors: %d\n", len(cm.Entries))

	// Step 6: Render output image
	fmt.Println("Rendering output...")
	rcfg := renderer.DefaultConfig()
	// Scale legend elements based on image size
	scaleLegendConfig(&rcfg, img.Bounds())
	output := renderer.Render(img, dm, zones, labels, cm, font, rcfg)

	// Step 7: Save output
	fmt.Printf("Saving output: %s\n", cfg.OutPath)
	if err := imaging.SavePNG(cfg.OutPath, output); err != nil {
		return fmt.Errorf("saving output: %w", err)
	}

	fmt.Println("Done!")
	return nil
}

// delimiterFromConfig builds the appropriate Delimiter from CLI config.
func delimiterFromConfig(cfg cli.Config) detection.Delimiter {
	if cfg.DelimiterStrategy == cli.StrategyBorder {
		return &detection.BorderDelimiter{
			Color:        cfg.BorderDelimiterColor,
			TolerancePct: cfg.BorderDelimiterTolerance,
		}
	}
	return &detection.ColorDelimiter{
		TolerancePct: cfg.ColorDelimiterTolerance,
	}
}

func countDelimiters(dm *detection.Map) int {
	count := 0
	for _, d := range dm.IsDelimiter {
		if d {
			count++
		}
	}
	return count
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
	// For small images, defaults are fine
}
