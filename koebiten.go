//go:build tinygo

package koebiten

import (
	"errors"
	"fmt"
	"image/color"
	"io/fs"
	"reflect"
	"strings"
	"time"

	"github.com/chewxy/math32"
	"tinygo.org/x/drivers"
	"tinygo.org/x/drivers/image/png"
	"tinygo.org/x/drivers/pixel"
	"tinygo.org/x/tinydraw"
	"tinygo.org/x/tinyfont"
)

// Displayer interface for display operations.
type Displayer interface {
	drivers.Displayer
	ClearDisplay()
	ClearBuffer()
}

var (
	display Displayer

	textY           int16
	ticks           uint32
	tickTimes       [32]uint32
	enableBenchmark bool
)

var (
	white = color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	black = color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xFF}
)

var keyUpdate = func() error { return nil }

func init() {
	pngBuffer = map[string]pixel.Image[pixel.Monochrome]{}
}

// Run starts the main loop for the application.
func Run(d func()) error {
	return RunGame(dummyGame(d))
}

func RunGame(game Game) error {
	tick := time.Tick(32 * time.Millisecond)
	for {
		<-tick
		ticks++
		if enableBenchmark && (ticks%32) == 0 {
			// print per 32 frame
			min := uint32(0xFFFFFFFF)
			max := uint32(0x00000000)
			for _, t := range tickTimes {
				if t < min {
					min = t
				}
				if max < t {
					max = t
				}
				fmt.Printf("%02d,", t/320)
			}
			fmt.Printf(" %3d %% - %3d %%\n", min/320, max/320)
		}
		s := time.Now().UnixMicro()

		keyUpdate()
		theInputState.update()
		textY = 0
		display.ClearBuffer()
		err := game.Update()
		if err != nil {
			if errors.Is(err, Termination) {
				return nil
			}
			return err
		}
		game.Draw(nil)
		display.Display()
		tickTimes[ticks%32] = uint32(time.Now().UnixMicro() - s)
	}
	return nil
}

// SetWindowSize sets the size of the display window.
func SetWindowSize(w, h int) {
}

// SetWindowTitle sets the title of the display window.
func SetWindowTitle(title string) {
}

func SetHardware(h Hardware) error {
	err := h.Init()
	if err != nil {
		return err
	}
	display = h.GetDisplay()
	keyUpdate = h.KeyUpdate
	return nil
}

// SetRotation sets the display rotation mode.
// If the display is already a RotatedDisplay, it updates the mode.
// Otherwise, it wraps the existing display in a new RotatedDisplay with the specified mode.
func SetRotation(mode int) {
	d, ok := display.(*RotatedDisplay)
	if ok {
		d.mode = mode
	} else {
		display = &RotatedDisplay{
			Displayer: display,
			mode:      mode,
		}
	}
}

// Println prints formatted output to the display.
func Println(args ...any) {
	str := []string{}
	for _, x := range args {
		s, ok := x.(string)
		if ok {
			str = append(str, s)
			continue
		}

		i, ok := x.(int)
		if ok {
			str = append(str, fmt.Sprintf("%d", i))
			continue
		}
	}

	textY += 8
	tinyfont.WriteLine(display, &tinyfont.Org01, 2, textY, strings.Join(str, " "), white)
}

// DrawText draws text on the display.
func DrawText(dst Displayer, str string, font tinyfont.Fonter, x, y int16, c pixel.BaseColor) {
	if isNil(dst) {
		dst = display
	}
	if font == nil {
		font = &tinyfont.Org01
	}
	tinyfont.WriteLine(dst, font, x, y, str, c.RGBA())
}

// DrawRect draws a rectangle on the display.
func DrawRect(dst Displayer, x, y, w, h int, c pixel.BaseColor) {
	if isNil(dst) {
		dst = display
	}
	tinydraw.Rectangle(dst, int16(x), int16(y), int16(w), int16(h), c.RGBA())
}

// DrawFilledRect draws a filled rectangle on the display.
func DrawFilledRect(dst Displayer, x, y, w, h int, c pixel.BaseColor) {
	if isNil(dst) {
		dst = display
	}
	tinydraw.FilledRectangle(dst, int16(x), int16(y), int16(w), int16(h), c.RGBA())
}

// DrawLine draws a line on the display.
func DrawLine(dst Displayer, x1, y1, x2, y2 int, c pixel.BaseColor) {
	if isNil(dst) {
		dst = display
	}
	tinydraw.Line(dst, int16(x1), int16(y1), int16(x2), int16(y2), c.RGBA())
}

// DrawCircle draws a circle on the display.
func DrawCircle(dst Displayer, x, y, r int, c pixel.BaseColor) {
	if isNil(dst) {
		dst = display
	}
	tinydraw.Circle(dst, int16(x), int16(y), int16(r), c.RGBA())
}

// DrawFilledCircle draws a filled circle on the display.
func DrawFilledCircle(dst Displayer, x, y, r int, c pixel.BaseColor) {
	if isNil(dst) {
		dst = display
	}
	tinydraw.FilledCircle(dst, int16(x), int16(y), int16(r), c.RGBA())
}

// DrawTriangle draws a triangle on the display.
func DrawTriangle(dst Displayer, x0, y0, x1, y1, x2, y2 int, c pixel.BaseColor) {
	if isNil(dst) {
		dst = display
	}
	tinydraw.Triangle(dst, int16(x0), int16(y0), int16(x1), int16(y1), int16(x2), int16(y2), c.RGBA())
}

// DrawFilledTriangle draws a filled triangle on the display.
func DrawFilledTriangle(dst Displayer, x0, y0, x1, y1, x2, y2 int, c pixel.BaseColor) {
	if isNil(dst) {
		dst = display
	}
	tinydraw.FilledTriangle(dst, int16(x0), int16(y0), int16(x1), int16(y1), int16(x2), int16(y2), c.RGBA())
}

var (
	buffer    [3 * 8 * 8 * 4]uint16
	pngBuffer map[string]pixel.Image[pixel.Monochrome]
)

type DrawImageFSOptions struct {
	GeoM GeoM
}

// DrawImageFS draws an image from the filesystem onto the display.
//
// Deprecated: Use Image and Image.DrawImage instead.
func DrawImageFS(dst Displayer, fsys fs.FS, path string, x, y int) {
	op := DrawImageFSOptions{}
	op.GeoM.Translate(float32(x), float32(y))
	DrawImageFSWithOptions(dst, fsys, path, op)
}

// DrawImageFSWithOptions draws an image from the filesystem onto the display with options.
//
// Deprecated: Use Image and Image.DrawImage instead.
func DrawImageFSWithOptions(dst Displayer, fsys fs.FS, path string, options DrawImageFSOptions) {
	if isNil(dst) {
		dst = display
	}
	img, ok := pngBuffer[path]
	if !ok {
		p, err := fsys.Open(path)
		if err != nil {
			return
		}

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

		_, err = png.Decode(p)
		if err != nil {
			return
		}
		pngBuffer[path] = img
	}

	geoM := options.GeoM
	if !geoM.IsInvertible() {
		return
	}

	w, h := img.Size()
	if geoM.a_1 == 0 && geoM.b == 0 && geoM.c == 0 && geoM.d_1 == 0 {
		tx, ty := geoM.Apply(0, 0)
		ox, oy := int(math32.Round(tx)), int(math32.Round(ty))
		for yy := 0; yy < h; yy++ {
			for xx := 0; xx < w; xx++ {
				if img.Get(xx, yy) == true {
					dst.SetPixel(int16(xx+ox), int16(yy+oy), white)
				}
			}
		}
	} else {
		for yy := 0; yy < h; yy++ {
			for xx := 0; xx < w; xx++ {
				if img.Get(xx, yy) == true {
					xxf, yyf := geoM.Apply(float32(xx), float32(yy))
					dst.SetPixel(int16(math32.Round(xxf)), int16(math32.Round(yyf)), white)
				}
			}
		}
	}
}

func isNil(d Displayer) bool {
	return d == nil || (reflect.ValueOf(d).Kind() == reflect.Ptr && reflect.ValueOf(d).IsNil())
}
