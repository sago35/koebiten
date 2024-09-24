package koebiten

import (
	"image/color"
)

// RGBATo565 converts a color.RGBA to uint16
func RGBATo565(c color.RGBA) uint16 {
	r, g, b, _ := c.RGBA()
	return uint16((r & 0xF800) +
		((g & 0xFC00) >> 5) +
		((b & 0xF800) >> 11))
}

// C565toRGBA converts a uint16 color to color.RGBA
func C565toRGBA(c uint16) color.RGBA {
	r := ((c & 0xF800) >> 11) << 3
	g := ((c & 0x07E0) >> 5) << 2
	b := ((c & 0x001F) >> 0) << 3
	return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 0xFF}
}

type RotatedDisplay struct {
	Displayer
}

func (d *RotatedDisplay) Size() (x, y int16) {
	return y, x
}

func (d *RotatedDisplay) SetPixel(x, y int16, c color.RGBA) {
	sx, _ := d.Displayer.Size()
	d.Displayer.SetPixel(sx-y, x, c)
}
