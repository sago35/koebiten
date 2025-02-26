package koebiten

import (
	"image/color"
	"io/fs"
	"math"

	"tinygo.org/x/drivers/image/png"
	"tinygo.org/x/drivers/pixel"
)

var _ Displayer = (*Image)(nil)

// Image is a Wrapper for the pixel.Image type.
//
// Image implements the Displayer interface.
type Image struct {
	img pixel.Image[pixel.Monochrome]
}

// Size returns the width and height of the image.
//
// It implements the Displayer interface.
func (i *Image) Size() (int16, int16) {
	x, y := i.img.Size()
	return int16(x), int16(y)
}

// SetPixel sets the pixel at the given x and y coordinates to the given color.
// The color is converted to a pixel.Monochrome color.
//
// It implements the Displayer interface.
func (i *Image) SetPixel(x, y int16, c color.RGBA) {
	i.img.Set(int(x), int(y), pixel.NewMonochrome(c.R, c.G, c.B))
}

// Display does nothing.
//
// It implements the Displayer interface.
func (i *Image) Display() error { return nil }

// ClearDisplay clears the display.
//
// It implements the Displayer interface.
func (i *Image) ClearDisplay() {}

// ClearBuffer clears the buffer.
//
// It implements the Displayer interface.
func (i *Image) ClearBuffer() {}

// NewImage creates a new Image with the given width and height.
//
// It returns a pointer to the new Image.
func NewImage(width, height int16) *Image {
	return &Image{
		img: pixel.NewImage[pixel.Monochrome](int(width), int(height)),
	}
}

// NewImageFromFS creates a new Image from the filesystem.
func NewImageFromFS(fsys fs.FS, path string) *Image {
	img, err := loadImageFromFS(fsys, path)
	if err != nil {
		panic(err)
	}
	return &Image{img: img}
}

// loadImageFromFS loads an image from the filesystem.
func loadImageFromFS(fsys fs.FS, path string) (pixel.Image[pixel.Monochrome], error) {
	var buffer [3 * 8 * 8 * 4]uint16
	p, err := fsys.Open(path)
	if err != nil {
		return pixel.Image[pixel.Monochrome]{}, err
	}

	var img pixel.Image[pixel.Monochrome]
	png.SetCallback(buffer[:], func(data []uint16, x, y, w, h, width, height int16) {
		if img.Len() == 0 {
			img = pixel.NewImage[pixel.Monochrome](int(width), int(height))
		}

		for yy := int16(0); yy < h; yy++ {
			for xx := int16(0); xx < w; xx++ {
				c := C565toRGBA(data[yy*w+xx])
				cnt := 0
				if c.R < 0x80 {
					cnt++
				}
				if c.G < 0x80 {
					cnt++
				}
				if c.B < 0x80 {
					cnt++
				}
				if cnt >= 2 {
					img.Set(int(x+xx), int(y+yy), true)
				}
			}
		}
	})

	if _, err = png.Decode(p); err != nil {
		return pixel.Image[pixel.Monochrome]{}, err
	}

	return img, nil
}

// Fill fills the image with the given color.
func (i *Image) Fill(clr color.Color) {
	r, g, b, _ := clr.RGBA()
	w, h := i.img.Size()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i.img.Set(x, y, pixel.NewMonochrome(uint8(r), uint8(g), uint8(b)))
		}
	}
}

type DrawImageOptions struct {
	GeoM GeoM
}

// DrawImage draws an image onto the display.
func (i *Image) DrawImage(dst Displayer, options DrawImageOptions) {
	if isNil(dst) {
		dst = display
	}

	geoM := options.GeoM
	if !geoM.IsInvertible() {
		return
	}

	w, h := i.img.Size()
	dw, dh := dst.Size()
	for yy := 0; yy < min(h, int(dh)); yy++ {
		for xx := 0; xx < min(w, int(dw)); xx++ {
			if i.img.Get(xx, yy) == true {
				xxf, yyf := geoM.Apply(float64(xx), float64(yy))
				dst.SetPixel(int16(math.Round(float64(xxf))), int16(math.Round(float64(yyf))), white)
			}
		}
	}
}
