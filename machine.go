//go:build tinygo && !macropad_rp2040

package koebiten

import (
	"machine"

	"tinygo.org/x/drivers"
	"tinygo.org/x/drivers/ssd1306"
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
}
