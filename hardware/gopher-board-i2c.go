//go:build tinygo && gopher_board_i2c

package hardware

import (
	"machine"

	"github.com/sago35/koebiten"
	"tinygo.org/x/drivers/ssd1306"
)

var (
	Device   = &device{}
	Display  *ssd1306.Device
	gpioPins []machine.Pin
)

type device struct {
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

func (z *device) GetDisplay() koebiten.Displayer {
	return Display
}

func (z *device) Init() error {
	i2c := machine.I2C0
	i2c.Configure(machine.I2CConfig{
		Frequency: 2_800_000,
		SDA:       machine.GPIO0,
		SCL:       machine.GPIO1,
	})

	d := ssd1306.NewI2C(i2c)
	d.Configure(ssd1306.Config{
		Address: 0x3C,
		Width:   128,
		Height:  64,
	})
	d.ClearDisplay()
	Display = &d

	gpioPins = []machine.Pin{
		machine.GPIO4,  // up
		machine.GPIO5,  // left
		machine.GPIO6,  // down
		machine.GPIO7,  // right
		machine.GPIO27, // A
		machine.GPIO28, // B
	}

	for _, p := range gpioPins {
		p.Configure(machine.PinConfig{Mode: machine.PinInputPullup})
	}

	z.gpioPins = []machine.Pin{
		machine.GPIO27, // A
		machine.GPIO28, // B
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
		machine.GPIO5, // left
		machine.GPIO7, // right
		machine.GPIO4, // up
		machine.GPIO6, // down
	}

	z.state = make([]State, len(z.gpioPins))
	z.cycle = make([]int, len(z.gpioPins))
	return nil
}

func (z *device) KeyUpdate() error {
	buf := z.keybuf[:]
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
