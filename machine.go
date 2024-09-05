//go:build tinygo && !macropad_rp2040

package koebiten

import (
	"machine"

	"tinygo.org/x/drivers"
	"tinygo.org/x/drivers/encoders"
	"tinygo.org/x/drivers/ssd1306"
)

var (
	colPins  []machine.Pin
	rowPins  []machine.Pin
	gpioPins []machine.Pin
	enc      *encoders.QuadratureDevice
	encOld   int
	state    []State
	cycle    []int
	duration []int
)

const (
	debounce = 2
)

type State uint8

const (
	None State = iota
	NoneToPress
	Press
	PressToRelease
)

func init() {
	btn = machine.GPIO2
	btn.Configure(machine.PinConfig{Mode: machine.PinInputPullup})

	i2c := machine.I2C0
	i2c.Configure(machine.I2CConfig{
		Frequency: 2_800_000,
		SDA:       machine.GPIO12,
		SCL:       machine.GPIO13,
	})

	d := ssd1306.NewI2C(i2c)
	d.Configure(ssd1306.Config{
		Address: 0x3C,
		Width:   128,
		Height:  64,
	})
	d.SetRotation(drivers.Rotation180)
	d.ClearDisplay()
	display = &d

	gpioPins = []machine.Pin{
		machine.GPIO2, // rotary
		machine.GPIO0, // joystick
	}

	for _, p := range gpioPins {
		p.Configure(machine.PinConfig{Mode: machine.PinInputPullup})
	}

	colPins = []machine.Pin{
		machine.GPIO5,
		machine.GPIO6,
		machine.GPIO7,
		machine.GPIO8,
	}

	rowPins = []machine.Pin{
		machine.GPIO9,
		machine.GPIO10,
		machine.GPIO11,
	}

	state = make([]State, len(colPins)*len(rowPins)+4)
	cycle = make([]int, len(colPins)*len(rowPins)+4)
	duration = make([]int, len(colPins)*len(rowPins)+4)

	for _, c := range colPins {
		c.Configure(machine.PinConfig{Mode: machine.PinOutput})
		c.Low()
	}

	for _, c := range rowPins {
		c.Configure(machine.PinConfig{Mode: machine.PinInputPulldown})
	}

	enc = encoders.NewQuadratureViaInterrupt(
		machine.GPIO3,
		machine.GPIO4,
	)

	enc.Configure(encoders.QuadratureConfig{
		Precision: 4,
	})
}

func keyUpdate() {
	keyGpioUpdate()
	keyRotaryUpdate()
	keyMatrixUpdate()
}

func keyGpioUpdate() {
	for r := range gpioPins {
		current := !gpioPins[r].Get()
		idx := r + len(colPins)*len(rowPins)

		switch state[idx] {
		case None:
			if current {
				if cycle[idx] >= debounce {
					state[idx] = NoneToPress
					cycle[idx] = 0
				} else {
					cycle[idx]++
				}
			} else {
				cycle[idx] = 0
			}
		case NoneToPress:
			state[idx] = Press
			theInputState.keyDurations[idx]++
			AppendJustPressedKeys([]Key{Key(idx)})
		case Press:
			AppendPressedKeys([]Key{Key(idx)})
			theInputState.keyDurations[idx]++
			if current {
				cycle[idx] = 0
				duration[idx]++
			} else {
				if cycle[idx] >= debounce {
					state[idx] = PressToRelease
					cycle[idx] = 0
					duration[idx] = 0
				} else {
					cycle[idx]++
				}
			}
		case PressToRelease:
			state[idx] = None
			AppendJustReleasedKeys([]Key{Key(idx)})
			theInputState.keyDurations[idx] = 0
		}
	}
}

func keyRotaryUpdate() {
	rot := []bool{false, false}
	if newValue := enc.Position(); newValue != encOld {
		if newValue < encOld {
			rot[0] = true
		} else {
			rot[1] = true
		}
		encOld = newValue
	}

	for c, current := range rot {
		idx := c + len(colPins)*len(rowPins) + 2
		switch state[idx] {
		case None:
			if current {
				state[idx] = NoneToPress
			} else {
			}
		case NoneToPress:
			if current {
				state[idx] = Press
			} else {
				state[idx] = PressToRelease
			}
			theInputState.keyDurations[idx]++
			AppendJustPressedKeys([]Key{Key(idx)})
		case Press:
			AppendPressedKeys([]Key{Key(idx)})
			theInputState.keyDurations[idx]++
			if current {
			} else {
				state[idx] = PressToRelease
			}
		case PressToRelease:
			if current {
				state[idx] = NoneToPress
			} else {
				state[idx] = None
			}
			AppendJustReleasedKeys([]Key{Key(idx)})
			theInputState.keyDurations[idx] = 0
		}
	}
}

func keyMatrixUpdate() {
	for c := range colPins {
		for r := range rowPins {
			colPins[c].Configure(machine.PinConfig{Mode: machine.PinOutput})
			colPins[c].High()
			current := rowPins[r].Get()
			idx := r*len(colPins) + c

			switch state[idx] {
			case None:
				if current {
					if cycle[idx] >= debounce {
						state[idx] = NoneToPress
						cycle[idx] = 0
					} else {
						cycle[idx]++
					}
				} else {
					cycle[idx] = 0
				}
			case NoneToPress:
				state[idx] = Press
				theInputState.keyDurations[idx]++
				AppendJustPressedKeys([]Key{Key(idx)})
			case Press:
				AppendPressedKeys([]Key{Key(idx)})
				theInputState.keyDurations[idx]++
				if current {
					cycle[idx] = 0
					duration[idx]++
				} else {
					if cycle[idx] >= debounce {
						state[idx] = PressToRelease
						cycle[idx] = 0
						duration[idx] = 0
					} else {
						cycle[idx]++
					}
				}
			case PressToRelease:
				state[idx] = None
				AppendJustReleasedKeys([]Key{Key(idx)})
				theInputState.keyDurations[idx] = 0
			}

			colPins[c].Low()
			colPins[c].Configure(machine.PinConfig{Mode: machine.PinInputPulldown})
		}
	}
}
