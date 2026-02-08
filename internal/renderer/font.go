package renderer

import (
	"image"
	"image/color"
)

// FontRenderer is the interface for drawing text onto images.
// Implementations can be swapped (e.g., bitmap font, TTF font).
type FontRenderer interface {
	// DrawString draws the given text centered at (cx, cy) on the image
	// with the specified color and font size (approximate height in pixels).
	DrawString(img *image.RGBA, text string, cx, cy int, col color.Color, size int)

	// MeasureString returns the approximate width and height of the text
	// at the given font size.
	MeasureString(text string, size int) (width, height int)
}

// BitmapFont is a simple bitmap font renderer using hardcoded glyph data
// for digits 0-9 and a few extra characters.
type BitmapFont struct{}

// NewBitmapFont creates a new BitmapFont.
func NewBitmapFont() *BitmapFont {
	return &BitmapFont{}
}

// glyphs are 5x7 pixel bitmaps for digits 0-9.
var glyphs = map[rune][7]uint8{
	'0': {0x0E, 0x11, 0x13, 0x15, 0x19, 0x11, 0x0E},
	'1': {0x04, 0x0C, 0x04, 0x04, 0x04, 0x04, 0x0E},
	'2': {0x0E, 0x11, 0x01, 0x06, 0x08, 0x10, 0x1F},
	'3': {0x0E, 0x11, 0x01, 0x06, 0x01, 0x11, 0x0E},
	'4': {0x02, 0x06, 0x0A, 0x12, 0x1F, 0x02, 0x02},
	'5': {0x1F, 0x10, 0x1E, 0x01, 0x01, 0x11, 0x0E},
	'6': {0x06, 0x08, 0x10, 0x1E, 0x11, 0x11, 0x0E},
	'7': {0x1F, 0x01, 0x02, 0x04, 0x08, 0x08, 0x08},
	'8': {0x0E, 0x11, 0x11, 0x0E, 0x11, 0x11, 0x0E},
	'9': {0x0E, 0x11, 0x11, 0x0F, 0x01, 0x02, 0x0C},
}

const (
	glyphWidth  = 5
	glyphHeight = 7
)

func (bf *BitmapFont) DrawString(img *image.RGBA, text string, cx, cy int, col color.Color, size int) {
	scale := size / glyphHeight
	if scale < 1 {
		scale = 1
	}

	totalW, totalH := bf.MeasureString(text, size)
	startX := cx - totalW/2
	startY := cy - totalH/2

	curX := startX
	for _, ch := range text {
		glyph, ok := glyphs[ch]
		if !ok {
			curX += (glyphWidth + 1) * scale
			continue
		}
		for row := 0; row < glyphHeight; row++ {
			for col_bit := 0; col_bit < glyphWidth; col_bit++ {
				if glyph[row]&(1<<(glyphWidth-1-col_bit)) != 0 {
					// Draw a scale x scale block
					for dy := 0; dy < scale; dy++ {
						for dx := 0; dx < scale; dx++ {
							px := curX + col_bit*scale + dx
							py := startY + row*scale + dy
							if px >= 0 && px < img.Bounds().Dx() && py >= 0 && py < img.Bounds().Dy() {
								img.Set(px+img.Bounds().Min.X, py+img.Bounds().Min.Y, col)
							}
						}
					}
				}
			}
		}
		curX += (glyphWidth + 1) * scale
	}
}

func (bf *BitmapFont) MeasureString(text string, size int) (width, height int) {
	scale := size / glyphHeight
	if scale < 1 {
		scale = 1
	}
	n := len([]rune(text))
	if n == 0 {
		return 0, 0
	}
	w := n*(glyphWidth*scale) + (n-1)*scale
	h := glyphHeight * scale
	return w, h
}
