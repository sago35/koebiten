//go:build tinygo && (zero_kb02 || conf2025badge || gopher_board_i2c)

package hardware

import (
	"tinygo.org/x/drivers/ssd1306"
)

// whiteBGDisplay wraps ssd1306.Device so that clearing the buffer fills the
// screen with white instead of black. koebiten draws ink in black on a white
// background.
type whiteBGDisplay struct {
	*ssd1306.Device
}

func (d whiteBGDisplay) ClearBuffer() {
	buf := d.GetBuffer()
	for i := range buf {
		buf[i] = 0xFF
	}
}

func (d whiteBGDisplay) ClearDisplay() {
	d.ClearBuffer()
	d.Display()
}
