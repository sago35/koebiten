//go:build tinygo && pybadge

package hardware

import (
	"image/color"
	"machine"

	"github.com/sago35/koebiten"
	"tinygo.org/x/drivers/pixel"
	"tinygo.org/x/drivers/shifter"
	"tinygo.org/x/drivers/st7735"
	"tinygo.org/x/tinydraw"
)

var Device = &device{}

type device struct {
	display  *Display
	gpioPins []uint8
	buttons  shifter.Device
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
	machine.SPI1.Configure(machine.SPIConfig{
		SCK:       machine.SPI1_SCK_PIN,
		SDO:       machine.SPI1_SDO_PIN,
		SDI:       machine.SPI1_SDI_PIN,
		Frequency: 8 * machine.MHz,
	})

	d := st7735.New(machine.SPI1, machine.TFT_RST, machine.TFT_DC, machine.TFT_CS, machine.TFT_LITE)
	d.Configure(st7735.Config{
		Rotation: st7735.ROTATION_90,
	})

	d.FillScreen(color.RGBA{0, 0, 0, 255})
	z.display = InitDisplay(&d, 128, 64)

	z.buttons = shifter.NewButtons()
	z.buttons.Configure()

	z.gpioPins = []uint8{
		shifter.BUTTON_A,
		shifter.BUTTON_B,
		shifter.BUTTON_SELECT,
		shifter.BUTTON_START,
		0xFF,
		0xFF,
		0xFF,
		0xFF,
		0xFF,
		0xFF,
		0xFF,
		0xFF,
		0xFF,
		0xFF,
		0xFF,
		0xFF,
		shifter.BUTTON_LEFT,
		shifter.BUTTON_RIGHT,
		shifter.BUTTON_UP,
		shifter.BUTTON_DOWN,
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
	z.buttons.ReadInput()
	for i, r := range z.gpioPins {
		current := false
		if r != 0xFF {
			current = z.buttons.Pins[r].Get()
		}
		idx := i

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
	d   *st7735.Device
	img pixel.Image[pixel.RGB565BE]
}

func InitDisplay(dev *st7735.Device, width, height int) *Display {
	d := &Display{
		d:   dev,
		img: pixel.NewImage[pixel.RGB565BE](width, height),
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
		d.img.Set(int(x), int(y), pixelWhite)
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
	ox := (160 - mx) / 2
	oy := (128 - my) / 2
	return int16(ox), int16(oy)
}

var (
	white = color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	black = color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xFF}

	pixelWhite = pixel.NewColor[pixel.RGB565BE](0xFF, 0xFF, 0xFF)
	pixelBlack = pixel.NewColor[pixel.RGB565BE](0x00, 0x00, 0x00)
)
