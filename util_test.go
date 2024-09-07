package koebiten

import (
	"image/color"
	"testing"
)

func TestRGBATo565(t *testing.T) {
	c := color.RGBA{0xFF, 0xFF, 0xFF, 0xFF}
	c5 := RGBATo565(c)
	if g, e := c5, uint16(0xFFFF); g != e {
		t.Errorf("got %04X want %04X", g, e)
	}
}

func TestC565toRGBA(t *testing.T) {
	c := uint16(0xFFFF)
	rgba := C565toRGBA(c)
	if g, e := rgba.R, uint8(0xF8); g != e {
		t.Errorf("got %04X want %04X", g, e)
	}
	if g, e := rgba.G, uint8(0xFC); g != e {
		t.Errorf("got %04X want %04X", g, e)
	}
	if g, e := rgba.B, uint8(0xF8); g != e {
		t.Errorf("got %04X want %04X", g, e)
	}
	if g, e := rgba.A, uint8(0xFF); g != e {
		t.Errorf("got %04X want %04X", g, e)
	}
}
