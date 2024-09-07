//go:build tinygo && macropad_rp2040

package koebiten

import (
	"machine"

	"tinygo.org/x/drivers/sh1106"
)

func init() {
	machine.SPI1.Configure(machine.SPIConfig{
		Frequency: 48000000,
	})

	d := sh1106.NewSPI(machine.SPI1, machine.OLED_DC, machine.OLED_RST, machine.OLED_CS)
	d.Configure(sh1106.Config{
		Width:  128,
		Height: 64,
	})
	d.ClearDisplay()
	display = &d
}
