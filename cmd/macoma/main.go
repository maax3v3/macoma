package main

import (
	"fmt"
	"os"

	"github.com/maax3v3/macoma/internal/cli"
	"github.com/maax3v3/macoma/internal/pipeline"
	"github.com/maax3v3/macoma/internal/renderer"
)

func main() {
	cfg, err := cli.Parse()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	font := renderer.NewBitmapFont()

	if err := pipeline.Run(cfg, font); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
