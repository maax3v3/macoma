package imaging

import (
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
)

func TestSavePNG_ThenLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.png")

	// Create a small test image
	src := image.NewRGBA(image.Rect(0, 0, 4, 4))
	src.SetRGBA(0, 0, color.RGBA{255, 0, 0, 255})
	src.SetRGBA(3, 3, color.RGBA{0, 0, 255, 255})

	if err := SavePNG(path, src); err != nil {
		t.Fatalf("SavePNG: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.Bounds().Dx() != 4 || loaded.Bounds().Dy() != 4 {
		t.Errorf("dimensions: got %dx%d, want 4x4", loaded.Bounds().Dx(), loaded.Bounds().Dy())
	}

	// Verify a pixel round-trips correctly
	r, g, b, _ := loaded.At(0, 0).RGBA()
	if r>>8 != 255 || g>>8 != 0 || b>>8 != 0 {
		t.Errorf("pixel (0,0): got (%d,%d,%d), want (255,0,0)", r>>8, g>>8, b>>8)
	}
}

func TestLoad_NonexistentFile(t *testing.T) {
	_, err := Load("/nonexistent/path/image.png")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestLoad_UnsupportedFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.bmp")
	if err := os.WriteFile(path, []byte("not a real image"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for unsupported format")
	}
}

func TestSavePNG_InvalidPath(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	err := SavePNG("/nonexistent/dir/out.png", img)
	if err == nil {
		t.Fatal("expected error for invalid path")
	}
}

func TestLoad_JPEG(t *testing.T) {
	dir := t.TempDir()
	jpgPath := filepath.Join(dir, "test.jpg")

	src := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			src.SetRGBA(x, y, color.RGBA{128, 128, 128, 255})
		}
	}

	f, err := os.Create(jpgPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := jpeg.Encode(f, src, nil); err != nil {
		f.Close()
		t.Fatal(err)
	}
	f.Close()

	loaded, err := Load(jpgPath)
	if err != nil {
		t.Fatalf("Load JPEG: %v", err)
	}
	if loaded.Bounds().Dx() != 10 || loaded.Bounds().Dy() != 10 {
		t.Errorf("dimensions: got %dx%d, want 10x10", loaded.Bounds().Dx(), loaded.Bounds().Dy())
	}
}

func TestLoad_CorruptPNG(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "corrupt.png")
	if err := os.WriteFile(path, []byte("not a real png"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for corrupt PNG")
	}
}
