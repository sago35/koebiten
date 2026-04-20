//go:build tinygo && m5stack

package hardware

import (
	"image/color"
	"machine"

	"github.com/sago35/koebiten"
	"tinygo.org/x/drivers/ili9341"
	"tinygo.org/x/drivers/pixel"
	"tinygo.org/x/tinydraw"
)

var Device = &device{}

type device struct {
	display  *Display
	gpioPins []machine.Pin
	state    []State
	cycle    []int
	keybuf   [1]koebiten.Key
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
	machine.SPI2.Configure(machine.SPIConfig{
		SCK:       machine.SPI0_SCK_PIN,
		SDO:       machine.SPI0_SDO_PIN,
		SDI:       machine.SPI0_SDI_PIN,
		Frequency: 40e6,
	})

	// configure backlight
	backlight := machine.LCD_BL_PIN
	backlight.Configure(machine.PinConfig{machine.PinOutput})

	display := ili9341.NewSPI(
		machine.SPI2,
		machine.LCD_DC_PIN,
		machine.LCD_SS_PIN,
		machine.LCD_RST_PIN,
	)

	// configure display
	display.Configure(ili9341.Config{
		Width:            320,
		Height:           240,
		DisplayInversion: true,
	})

	backlight.Low()

	display.SetRotation(ili9341.Rotation0Mirror)
	display.FillRectangle(0, 0, 320, 240, black)

	backlight.High()

	z.display = InitDisplay(display, 128, 64)

	gpioPins := []machine.Pin{
		machine.BUTTON_A,
		machine.BUTTON_B,
		machine.BUTTON_C,
	}

	for i := range gpioPins {
		gpioPins[i].Configure(machine.PinConfig{Mode: machine.PinInputPullup})
	}

	z.gpioPins = []machine.Pin{
		machine.BUTTON_A,
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
		machine.BUTTON_B,
		machine.BUTTON_C,
		machine.NoPin,
		machine.NoPin,
		machine.NoPin,
		machine.NoPin,
	}

	z.state = make([]State, len(z.gpioPins))
	z.cycle = make([]int, len(z.gpioPins))
	return nil
}

func (z *device) GetDisplay() koebiten.Displayer {
	return z.display
}

func (z *device) KeyUpdate() error {
	buf := z.keybuf[:]
	for r := range z.gpioPins {
		current := false
		if z.gpioPins[r] != machine.NoPin {
			current = !z.gpioPins[r].Get()
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
			buf[0] = koebiten.Key(idx)
			koebiten.AppendJustPressedKeys(buf)
		case Press:
			buf[0] = koebiten.Key(idx)
			koebiten.AppendPressedKeys(buf)
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
			buf[0] = koebiten.Key(idx)
			koebiten.AppendJustReleasedKeys(buf)
		}
	}
	return nil
}

type Display struct {
	d   *ili9341.Device
	img pixel.Image[pixel.RGB565BE]
}

func InitDisplay(dev *ili9341.Device, width, height int) *Display {
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
