package detection

import (
	"image"
	"image/color"
	"testing"
)

func benchmarkGradientImage(w, h int, mode string) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	switch mode {
	case "flat":
		fill := color.RGBA{120, 120, 120, 255}
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				img.SetRGBA(x, y, fill)
			}
		}
	case "seam":
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				if x < w/2 {
					img.SetRGBA(x, y, color.RGBA{240, 70, 70, 255})
				} else {
					img.SetRGBA(x, y, color.RGBA{60, 60, 230, 255})
				}
			}
		}
	case "noisy":
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				v := uint8((x*31 + y*17 + (x*y)%29) % 256)
				img.SetRGBA(x, y, color.RGBA{v, 255 - v, v / 2, 255})
			}
		}
	default:
		panic("unknown mode")
	}
	return img
}

func BenchmarkGradientDelimiter_MediumSeam(b *testing.B) {
	img := benchmarkGradientImage(1200, 800, "seam")
	d := &GradientDelimiter{
		BlurSigma:        0.9,
		LowThreshold:     0.14,
		HighThreshold:    0.30,
		CloseRadius:      0,
		MinComponentSize: 24,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = d.Detect(img)
	}
}

func BenchmarkGradientDelimiter_MediumNoisy(b *testing.B) {
	img := benchmarkGradientImage(1200, 800, "noisy")
	d := &GradientDelimiter{
		BlurSigma:        0.9,
		LowThreshold:     0.14,
		HighThreshold:    0.30,
		CloseRadius:      0,
		MinComponentSize: 24,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = d.Detect(img)
	}
}
