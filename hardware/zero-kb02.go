//go:build tinygo && zero_kb02

package hardware

import (
	"machine"

	"github.com/sago35/koebiten"
	"tinygo.org/x/drivers"
	"tinygo.org/x/drivers/encoders"
	"tinygo.org/x/drivers/ssd1306"
)

var Device = ZERO_KB02{}

type ZERO_KB02 struct {
}

func (z ZERO_KB02) Init() error {
	return Init()
}

func (z ZERO_KB02) GetDisplay() koebiten.Displayer {
	return Display
}

func (z ZERO_KB02) KeyUpdate() error {
	return keyUpdate()
}

var (
	Display *ssd1306.Device
)

var (
	colPins          []machine.Pin
	rowPins          []machine.Pin
	rotaryPins       []machine.Pin
	gpioPins         []machine.Pin
	adcPins          []ADCDevice
	enc              *encoders.QuadratureDevice
	encOld           int
	state            []State
	cycle            []int
	duration         []int
	invertRotaryPins = false
)

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

type ADCDevice struct {
	ADC         machine.ADC
	PressedFunc func() bool
}

func (a ADCDevice) Get() bool {
	return a.PressedFunc()
}

func Init() error {
	machine.InitADC()
	ax := machine.ADC{Pin: machine.GPIO29}
	ay := machine.ADC{Pin: machine.GPIO28}
	ax.Configure(machine.ADCConfig{})
	ay.Configure(machine.ADCConfig{})

	adcPins = []ADCDevice{
		{ADC: ax, PressedFunc: func() bool { return ax.Get() < 0x4800 }}, // left
		{ADC: ax, PressedFunc: func() bool { return 0xB800 < ax.Get() }}, // right
		{ADC: ay, PressedFunc: func() bool { return 0xB800 < ay.Get() }}, // up
		{ADC: ay, PressedFunc: func() bool { return ay.Get() < 0x4800 }}, // down
	}

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
	Display = &d

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

	for _, c := range colPins {
		c.Configure(machine.PinConfig{Mode: machine.PinOutput})
		c.Low()
	}

	for _, c := range rowPins {
		c.Configure(machine.PinConfig{Mode: machine.PinInputPulldown})
	}

	rotaryPins = []machine.Pin{
		machine.GPIO4,
		machine.GPIO3,
	}

	if invertRotaryPins {
		rotaryPins = []machine.Pin{
			machine.GPIO3,
			machine.GPIO4,
		}
	}
	enc = encoders.NewQuadratureViaInterrupt(rotaryPins[0], rotaryPins[1])

	enc.Configure(encoders.QuadratureConfig{
		Precision: 4,
	})

	state = make([]State, len(colPins)*len(rowPins)+len(gpioPins)+len(rotaryPins)+len(adcPins))
	cycle = make([]int, len(colPins)*len(rowPins)+len(gpioPins)+len(rotaryPins)+len(adcPins))
	duration = make([]int, len(colPins)*len(rowPins)+len(gpioPins)+len(rotaryPins)+len(adcPins))

	return nil
}

func keyUpdate() error {
	keyGpioUpdate()
	keyRotaryUpdate()
	keyMatrixUpdate()
	keyJoystickUpdate()
	return nil
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
			koebiten.AppendJustPressedKeys([]koebiten.Key{koebiten.Key(idx)})
		case Press:
			koebiten.AppendPressedKeys([]koebiten.Key{koebiten.Key(idx)})
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
			koebiten.AppendJustReleasedKeys([]koebiten.Key{koebiten.Key(idx)})
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
			koebiten.AppendJustPressedKeys([]koebiten.Key{koebiten.Key(idx)})
		case Press:
			koebiten.AppendPressedKeys([]koebiten.Key{koebiten.Key(idx)})
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
			koebiten.AppendJustReleasedKeys([]koebiten.Key{koebiten.Key(idx)})
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
				koebiten.AppendJustPressedKeys([]koebiten.Key{koebiten.Key(idx)})
			case Press:
				koebiten.AppendPressedKeys([]koebiten.Key{koebiten.Key(idx)})
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
				koebiten.AppendJustReleasedKeys([]koebiten.Key{koebiten.Key(idx)})
			}

			colPins[c].Low()
			colPins[c].Configure(machine.PinConfig{Mode: machine.PinInputPulldown})
		}
	}
}

func keyJoystickUpdate() {
	for r, p := range adcPins {
		current := p.Get()
		idx := r + len(colPins)*len(rowPins) + len(gpioPins) + len(rotaryPins)

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
			koebiten.AppendJustPressedKeys([]koebiten.Key{koebiten.Key(idx)})
		case Press:
			koebiten.AppendPressedKeys([]koebiten.Key{koebiten.Key(idx)})
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
			koebiten.AppendJustReleasedKeys([]koebiten.Key{koebiten.Key(idx)})
		}
	}
}
