//go:build tinygo && conf2025badge

package hardware

import (
	"machine"

	"github.com/sago35/koebiten"
	"tinygo.org/x/drivers"
	"tinygo.org/x/drivers/encoders"
	"tinygo.org/x/drivers/ssd1306"
)

var Device = CONF2025BADGE{}

type CONF2025BADGE struct {
}

func (z CONF2025BADGE) Init() error {
	return Init()
}

func (z CONF2025BADGE) GetDisplay() koebiten.Displayer {
	return Display
}

func (z CONF2025BADGE) KeyUpdate() error {
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
	keybuf           [1]koebiten.Key
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
	ax := machine.ADC{Pin: machine.GPIO27}
	ay := machine.ADC{Pin: machine.GPIO26}
	ax.Configure(machine.ADCConfig{})
	ay.Configure(machine.ADCConfig{})

	adcPins = []ADCDevice{
		{ADC: ax, PressedFunc: func() bool { return ax.Get() < 0x3000 }}, // left
		{ADC: ax, PressedFunc: func() bool { return 0xC800 < ax.Get() }}, // right
		{ADC: ay, PressedFunc: func() bool { return 0xC800 < ay.Get() }}, // up
		{ADC: ay, PressedFunc: func() bool { return ay.Get() < 0x3000 }}, // down
	}

	i2c := machine.I2C1
	i2c.Configure(machine.I2CConfig{
		Frequency: 2_800_000,
		SDA:       machine.GPIO6,
		SCL:       machine.GPIO7,
	})

	d := ssd1306.NewI2C(i2c)
	d.Configure(ssd1306.Config{
		Address: 0x3C,
		Width:   128,
		Height:  64,
	})
	d.SetRotation(drivers.Rotation0)
	d.ClearDisplay()
	Display = d

	gpioPins = []machine.Pin{
		machine.GPIO28,
		machine.GPIO29,
		machine.GPIO2, // rotary
	}

	for _, p := range gpioPins {
		p.Configure(machine.PinConfig{Mode: machine.PinInputPullup})
	}

	rotaryPins = []machine.Pin{
		machine.GPIO3,
		machine.GPIO4,
	}

	if invertRotaryPins {
		rotaryPins = []machine.Pin{
			machine.GPIO4,
			machine.GPIO3,
		}
	}
	enc = encoders.NewQuadratureViaInterrupt(rotaryPins[0], rotaryPins[1])

	enc.Configure(encoders.QuadratureConfig{
		Precision: 4,
	})

	state = make([]State, koebiten.KeyDown+1)
	cycle = make([]int, koebiten.KeyDown+1)
	duration = make([]int, koebiten.KeyDown+1)

	return nil
}

func keyUpdate() error {
	keyGpioUpdate()
	keyRotaryUpdate()
	keyJoystickUpdate()
	return nil
}

func keyGpioUpdate() {
	buf := keybuf[:]
	for r := range gpioPins {
		current := !gpioPins[r].Get()
		idx := r + int(koebiten.Key0)

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
			buf[0] = koebiten.Key(idx)
			koebiten.AppendJustPressedKeys(buf)
		case Press:
			buf[0] = koebiten.Key(idx)
			koebiten.AppendPressedKeys(buf)
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
			buf[0] = koebiten.Key(idx)
			koebiten.AppendJustReleasedKeys(buf)
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

	buf := keybuf[:]
	for c, current := range rot {
		idx := c + int(koebiten.KeyRotaryLeft)
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
			buf[0] = koebiten.Key(idx)
			koebiten.AppendJustPressedKeys(buf)
		case Press:
			buf[0] = koebiten.Key(idx)
			koebiten.AppendPressedKeys(buf)
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
			buf[0] = koebiten.Key(idx)
			koebiten.AppendJustReleasedKeys(buf)
		}
	}
}

func keyJoystickUpdate() {
	buf := keybuf[:]
	for r, p := range adcPins {
		current := p.Get()
		idx := r + int(koebiten.KeyLeft)

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
			buf[0] = koebiten.Key(idx)
			koebiten.AppendJustPressedKeys(buf)
		case Press:
			buf[0] = koebiten.Key(idx)
			koebiten.AppendPressedKeys(buf)
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
			buf[0] = koebiten.Key(idx)
			koebiten.AppendJustReleasedKeys(buf)
		}
	}
}
