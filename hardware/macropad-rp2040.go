//go:build tinygo && macropad_rp2040

package hardware

import (
	"machine"

	"github.com/sago35/koebiten"
	"tinygo.org/x/drivers/sh1106"
)

var Device = &device{}

type device struct {
	display  *sh1106.Device
	whiteBuf []byte
	gpioPins []machine.Pin
	state    []State
	cycle    []int
	keybuf   [1]koebiten.Key
}

// whiteBGDisplay wraps sh1106.Device so that clearing the buffer fills the
// screen with white instead of black. koebiten draws ink in black on a white
// background. sh1106 does not expose its buffer, so an all-white buffer is
// copied in via SetBuffer.
type whiteBGDisplay struct {
	*sh1106.Device
	whiteBuf []byte
}

func (d whiteBGDisplay) ClearBuffer() {
	d.SetBuffer(d.whiteBuf)
}

func (d whiteBGDisplay) ClearDisplay() {
	d.ClearBuffer()
	d.Display()
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
	err := machine.SPI1.Configure(machine.SPIConfig{
		Frequency: 48000000,
	})
	if err != nil {
		return err
	}

	d := sh1106.NewSPI(machine.SPI1, machine.OLED_DC, machine.OLED_RST, machine.OLED_CS)
	d.Configure(sh1106.Config{
		Width:  128,
		Height: 64,
	})
	d.ClearDisplay()
	z.display = &d

	z.whiteBuf = make([]byte, 128*64/8)
	for i := range z.whiteBuf {
		z.whiteBuf[i] = 0xFF
	}

	gpioPins := []machine.Pin{
		machine.KEY1,
		machine.KEY2,
		machine.KEY3,
		machine.KEY4,
		machine.KEY5,
		machine.KEY6,
		machine.KEY7,
		machine.KEY8,
		machine.KEY9,
		machine.KEY10,
		machine.KEY11,
		machine.KEY12,
	}

	for i := range gpioPins {
		gpioPins[i].Configure(machine.PinConfig{Mode: machine.PinInputPullup})
	}

	z.gpioPins = []machine.Pin{
		machine.KEY1,
		machine.KEY2,
		machine.KEY3,
		machine.KEY4,
		machine.KEY5,
		machine.KEY6,
		machine.NoPin,
		machine.NoPin,
		machine.NoPin,
		machine.NoPin,
		machine.NoPin,
		machine.KEY7,
		machine.NoPin,
		machine.NoPin,
		machine.NoPin,
		machine.NoPin,
		machine.KEY10,
		machine.KEY12,
		machine.KEY8,
		machine.KEY11,
	}

	z.state = make([]State, len(z.gpioPins))
	z.cycle = make([]int, len(z.gpioPins))
	return nil
}

func (z *device) GetDisplay() koebiten.Displayer {
	return whiteBGDisplay{z.display, z.whiteBuf}
}

func (z *device) KeyUpdate() error {
	buf := z.keybuf[:]
	for r := range z.gpioPins {
		current := !z.gpioPins[r].Get()
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
