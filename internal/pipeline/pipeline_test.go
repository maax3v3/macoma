package pipeline

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/maax3v3/macoma/internal/cli"
	mcol "github.com/maax3v3/macoma/internal/color"
	"github.com/maax3v3/macoma/internal/renderer"
)

func createTestImage(t *testing.T, path string) {
	t.Helper()
	w, h := 200, 200
	img := image.NewRGBA(image.Rect(0, 0, w, h))

	red := color.RGBA{255, 0, 0, 255}
	green := color.RGBA{0, 200, 0, 255}
	blue := color.RGBA{0, 0, 255, 255}
	yellow := color.RGBA{255, 255, 0, 255}
	black := color.RGBA{0, 0, 0, 255}

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			switch {
			case x < 100 && y < 100:
				img.Set(x, y, red)
			case x >= 100 && y < 100:
				img.Set(x, y, green)
			case x < 100 && y >= 100:
				img.Set(x, y, blue)
			default:
				img.Set(x, y, yellow)
			}
		}
	}

	// Delimiter lines
	for y := 0; y < h; y++ {
		for dx := 0; dx < 3; dx++ {
			img.Set(99+dx, y, black)
		}
	}
	for x := 0; x < w; x++ {
		for dy := 0; dy < 3; dy++ {
			img.Set(x, 99+dy, black)
		}
	}
	for x := 0; x < w; x++ {
		for d := 0; d < 2; d++ {
			img.Set(x, d, black)
			img.Set(x, h-1-d, black)
		}
	}
	for y := 0; y < h; y++ {
		for d := 0; d < 2; d++ {
			img.Set(d, y, black)
			img.Set(w-1-d, y, black)
		}
	}

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

func TestPipelineEndToEnd(t *testing.T) {
	tmpDir := t.TempDir()
	inPath := filepath.Join(tmpDir, "input.png")
	outPath := filepath.Join(tmpDir, "output.png")

	createTestImage(t, inPath)

	cfg := cli.Config{
		InPath:             inPath,
		OutPath:            outPath,
		DelimiterColor:     mcol.RGBA{R: 0, G: 0, B: 0, A: 255},
		DelimiterTolerance: 1,
		MaxColors:          0,
	}

	font := renderer.NewBitmapFont()
	if err := Run(cfg, font); err != nil {
		t.Fatalf("pipeline failed: %v", err)
	}

	// Verify output file exists and is a valid PNG
	f, err := os.Open(outPath)
	if err != nil {
		t.Fatalf("output file not found: %v", err)
	}
	defer f.Close()

	outImg, err := png.Decode(f)
	if err != nil {
		t.Fatalf("output is not valid PNG: %v", err)
	}

	// Output should be wider or same width, taller (legend added)
	if outImg.Bounds().Dx() != 200 {
		t.Errorf("expected output width 200, got %d", outImg.Bounds().Dx())
	}
	if outImg.Bounds().Dy() <= 200 {
		t.Errorf("expected output height > 200 (legend), got %d", outImg.Bounds().Dy())
	}
}

func TestPipelineWithMaxColors(t *testing.T) {
	tmpDir := t.TempDir()
	inPath := filepath.Join(tmpDir, "input.png")
	outPath := filepath.Join(tmpDir, "output.png")

	createTestImage(t, inPath)

	cfg := cli.Config{
		InPath:             inPath,
		OutPath:            outPath,
		DelimiterColor:     mcol.RGBA{R: 0, G: 0, B: 0, A: 255},
		DelimiterTolerance: 1,
		MaxColors:          2,
	}

	font := renderer.NewBitmapFont()
	if err := Run(cfg, font); err != nil {
		t.Fatalf("pipeline failed: %v", err)
	}

	f, err := os.Open(outPath)
	if err != nil {
		t.Fatalf("output file not found: %v", err)
	}
	defer f.Close()

	_, err = png.Decode(f)
	if err != nil {
		t.Fatalf("output is not valid PNG: %v", err)
	}
}
