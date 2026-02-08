package imaging

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	_ "golang.org/x/image/webp"
)

// Load reads an image file from disk. Supports PNG, JPEG, and WEBP.
func Load(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening image: %w", err)
	}
	defer f.Close()

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".png":
		return png.Decode(f)
	case ".jpg", ".jpeg":
		return jpeg.Decode(f)
	case ".webp":
		// Decoded via the blank import of golang.org/x/image/webp
		img, _, err := image.Decode(f)
		return img, err
	default:
		return nil, fmt.Errorf("unsupported image format %q (supported: png, jpg, jpeg, webp)", ext)
	}
}

// SavePNG writes an image to disk as PNG.
func SavePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		return fmt.Errorf("encoding PNG: %w", err)
	}
	return nil
}
