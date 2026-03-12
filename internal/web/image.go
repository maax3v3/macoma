package web

import (
	"image"
	"io"
	"math"

	_ "image/jpeg"
	_ "image/png"

	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

func decodeImage(r io.Reader) (image.Image, error) {
	img, _, err := image.Decode(r)
	return img, err
}

func scaleDown(img image.Image, maxDim int) image.Image {
	if img == nil || maxDim <= 0 {
		return img
	}

	b := img.Bounds()
	w := b.Dx()
	h := b.Dy()
	if w <= maxDim && h <= maxDim {
		return img
	}

	var nw, nh int
	if w >= h {
		nw = maxDim
		nh = int(math.Round(float64(h) * float64(maxDim) / float64(w)))
	} else {
		nh = maxDim
		nw = int(math.Round(float64(w) * float64(maxDim) / float64(h)))
	}

	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, b, xdraw.Over, nil)
	return dst
}
