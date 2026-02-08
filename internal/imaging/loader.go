package imaging

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	_ "golang.org/x/image/webp"
)

// Load reads an image file from disk. Supports PNG, JPEG, and WEBP.
// The path is normalized: ~ is expanded to the user's home directory,
// and relative paths are resolved to absolute.
func Load(path string) (image.Image, error) {
	path = ExpandPath(path)
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
// The path is normalized: ~ is expanded and relative paths are resolved.
func SavePNG(path string, img image.Image) error {
	path = ExpandPath(path)
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

// ExpandPath normalizes a file path by expanding ~ to the user's home
// directory and resolving relative paths to absolute.
func ExpandPath(path string) string {
	if path == "" {
		return path
	}

	// Expand ~ and ~/ to home directory
	if path == "~" || strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			path = filepath.Join(home, path[1:])
		}
	}

	// On Windows, also handle ~\
	if runtime.GOOS == "windows" && strings.HasPrefix(path, "~\\") {
		if home, err := os.UserHomeDir(); err == nil {
			path = filepath.Join(home, path[2:])
		}
	}

	// Resolve relative paths to absolute
	if !filepath.IsAbs(path) {
		if abs, err := filepath.Abs(path); err == nil {
			path = abs
		}
	}

	return filepath.Clean(path)
}
