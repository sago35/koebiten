//go:build tinygo && gopher_badge

package hardware

import (
	"image/color"
	"machine"

	"github.com/sago35/koebiten"
	"tinygo.org/x/drivers/pixel"
	"tinygo.org/x/drivers/st7789"
	"tinygo.org/x/tinydraw"
)

var Device = &device{}

type device struct {
	display  *Display
	gpioPins []machine.Pin
	state    []State
	cycle    []int
}

const (
	debounce = 0
)

type State uint8

const (
	None State = iota
	NoneToPress
	Press
	PressToRelease
)

func (z *device) Init() error {
	machine.SPI0.Configure(machine.SPIConfig{
		Frequency: 32 * machine.MHz,
		Mode:      0,
	})

	d := st7789.New(machine.SPI0,
		machine.TFT_RST,       // TFT_RESET
		machine.TFT_WRX,       // TFT_DC
		machine.TFT_CS,        // TFT_CS
		machine.TFT_BACKLIGHT) // TFT_LITE

	d.Configure(st7789.Config{
		Rotation: st7789.ROTATION_270,
		Height:   320,
	})
	//d.ClearDisplay()
	z.display = InitDisplay(&d, 128, 64)

	gpioPins := []machine.Pin{
		machine.BUTTON_A,
		machine.BUTTON_B,
		machine.BUTTON_UP,
		machine.BUTTON_LEFT,
		machine.BUTTON_DOWN,
		machine.BUTTON_RIGHT,
	}

	for i := range gpioPins {
		gpioPins[i].Configure(machine.PinConfig{Mode: machine.PinInputPullup})
	}

	z.gpioPins = []machine.Pin{
		machine.BUTTON_A,
		machine.BUTTON_B,
		machine.NoPin,
		machine.NoPin,
		machine.NoPin,
		machine.NoPin,
		machine.NoPin,
		machine.NoPin,
		machine.NoPin,
		machine.NoPin,
		machine.NoPin,
		machine.NoPin,
		machine.NoPin,
		machine.NoPin,
		machine.NoPin,
		machine.NoPin,
		machine.BUTTON_LEFT,
		machine.BUTTON_RIGHT,
		machine.BUTTON_UP,
		machine.BUTTON_DOWN,
	}

	z.state = make([]State, len(z.gpioPins))
	z.cycle = make([]int, len(z.gpioPins))
	return nil
}

func (z *device) GetDisplay() koebiten.Displayer {
	return z.display
}

func (z *device) KeyUpdate() error {
	for r := range z.gpioPins {
		current := !z.gpioPins[r].Get()
		if z.gpioPins[r] == machine.NoPin {
			current = false
		}
		idx := r

		switch z.state[idx] {
		case None:
			if current {
				if z.cycle[idx] >= debounce {
					z.state[idx] = NoneToPress
					z.cycle[idx] = 0
				} else {
					z.cycle[idx]++
				}
			} else {
				z.cycle[idx] = 0
			}
		case NoneToPress:
			z.state[idx] = Press
			koebiten.AppendJustPressedKeys([]koebiten.Key{koebiten.Key(idx)})
		case Press:
			koebiten.AppendPressedKeys([]koebiten.Key{koebiten.Key(idx)})
			if current {
				z.cycle[idx] = 0
			} else {
				if z.cycle[idx] >= debounce {
					z.state[idx] = PressToRelease
					z.cycle[idx] = 0
				} else {
					z.cycle[idx]++
				}
			}
		case PressToRelease:
			z.state[idx] = None
			koebiten.AppendJustReleasedKeys([]koebiten.Key{koebiten.Key(idx)})
		}
	}
	return nil
}

type Display struct {
	d   *st7789.Device
	img pixel.Image[pixel.RGB565BE]
}

func InitDisplay(dev *st7789.Device, width, height int) *Display {
	d := &Display{
		d:   dev,
		img: pixel.NewImage[pixel.RGB565BE](width*2+1, height*2+1),
	}

	ox, oy := d.getImageTopLeftForCentering()
	w, h := d.img.Size()
	tinydraw.Rectangle(dev, ox-1, oy-1, int16(w)+2, int16(h)+2, white)

	return d
}

func (d *Display) Size() (x, y int16) {
	return 128, 64
}

func (d *Display) SetPixel(x, y int16, c color.RGBA) {
	mx, my := d.Size()
	if 0 <= x && x < int16(mx) && 0 <= y && y < int16(my) {
		d.img.Set(int(x*2+0), int(y*2+0), pixelWhite)
		d.img.Set(int(x*2+1), int(y*2+0), pixelWhite)
		d.img.Set(int(x*2+0), int(y*2+1), pixelWhite)
		d.img.Set(int(x*2+1), int(y*2+1), pixelWhite)
	}
	return
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
		d.img.Set(int(x), int(y), pixelWhite)
	} else {
		d.img.Set(int(x), int(y), pixelBlack)
	}
	//d.d.SetPixel(x, y, c)
}

func (d *Display) Display() error {
	ox, oy := d.getImageTopLeftForCentering()
	return d.d.DrawBitmap(int16(ox), int16(oy), d.img)
}

func (d *Display) ClearBuffer() {
	d.img.FillSolidColor(pixelBlack)
}

func (d *Display) ClearDisplay() {
}

func (d *Display) getImageTopLeftForCentering() (int16, int16) {
	mx, my := d.img.Size()
	ox := (320 - mx) / 2
	oy := (240 - my) / 2
	return int16(ox), int16(oy)
}

var (
	white = color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	black = color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xFF}

	pixelWhite = pixel.NewColor[pixel.RGB565BE](0xFF, 0xFF, 0xFF)
	pixelBlack = pixel.NewColor[pixel.RGB565BE](0x00, 0x00, 0x00)
)
