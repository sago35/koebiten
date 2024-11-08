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
	return z.display
}

func (z *device) KeyUpdate() error {
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
