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
	mode int
}

func (d *RotatedDisplay) Size() (x, y int16) {
	switch d.mode {
	case 0, 2:
		return x, y
	default:
	}
	return y, x
}

func (d *RotatedDisplay) SetPixel(x, y int16, c color.RGBA) {
	switch d.mode {
	case 0:
		d.Displayer.SetPixel(x, y, c)
	case 1:
		sx, _ := d.Displayer.Size()
		d.Displayer.SetPixel(sx-y, x, c)
	case 2:
		sx, sy := d.Displayer.Size()
		d.Displayer.SetPixel(sx-x, sy-y, c)
	case 3:
		_, sy := d.Displayer.Size()
		d.Displayer.SetPixel(y, sy-x, c)
	}
}
