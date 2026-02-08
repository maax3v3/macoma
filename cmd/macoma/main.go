package main

import (
	"fmt"
	"os"

	"github.com/maax3v3/macoma"
	"github.com/maax3v3/macoma/internal/cli"
)

func main() {
	cfg, err := cli.Parse()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	opts := macoma.Options{
		DelimiterStrategy: cfg.DelimiterStrategy,
		BorderDelimiterColor: macoma.Color{
			R: cfg.BorderDelimiterColor.R,
			G: cfg.BorderDelimiterColor.G,
			B: cfg.BorderDelimiterColor.B,
			A: cfg.BorderDelimiterColor.A,
		},
		BorderDelimiterTolerance: cfg.BorderDelimiterTolerance,
		ColorDelimiterTolerance:  cfg.ColorDelimiterTolerance,
		MaxColors:                cfg.MaxColors,
	}

	fmt.Printf("Loading image: %s\n", cfg.InPath)
	img, err := macoma.LoadImage(cfg.InPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Image loaded: %dx%d\n", img.Bounds().Dx(), img.Bounds().Dy())

	fmt.Printf("Converting (strategy=%s)...\n", opts.DelimiterStrategy)
	result, err := macoma.Convert(img, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Saving output: %s\n", cfg.OutPath)
	if err := macoma.SavePNG(cfg.OutPath, result); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Done!")
}
