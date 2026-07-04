//go:build tinygo && wasm

package hardware

import (
	"image/color"
	"syscall/js"

	"github.com/sago35/koebiten"
)

var (
	d        = NewDisplay(128, 64)
	Device   = &WasmDevice{}
	keyState = map[koebiten.Key]bool{}
	keysBuf  = [1]koebiten.Key{}
)

var allKeys = [...]koebiten.Key{
	koebiten.Key0,
	koebiten.Key1,
	koebiten.Key2,
	koebiten.Key3,
	koebiten.Key4,
	koebiten.Key5,
	koebiten.Key6,
	koebiten.Key7,
	koebiten.Key8,
	koebiten.Key9,
	koebiten.Key10,
	koebiten.Key11,
	koebiten.KeyRotaryButton,
	koebiten.KeyJoystick,
	koebiten.KeyRotaryLeft,
	koebiten.KeyRotaryRight,
	koebiten.KeyLeft,
	koebiten.KeyRight,
	koebiten.KeyUp,
	koebiten.KeyDown,
}

func init() {
	wasmKeyEvent := wasmKeyEvent()
	js.Global().Set("wasmKeyEvent", wasmKeyEvent)

}

func wasmKeyEvent() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) < 1 {
			return nil
		}
		key := args[0].String()

		switch key {
		case "ArrowUp", "e", "k":
			keyState[koebiten.KeyArrowUp] = true
		case "ArrowDown", "d", "j":
			keyState[koebiten.KeyArrowDown] = true
		case "ArrowLeft", "s", "h":
			keyState[koebiten.KeyArrowLeft] = true
		case "ArrowRight", "f", "l":
			keyState[koebiten.KeyArrowRight] = true
		case "z", "n", "0", " ", "Enter":
			keyState[koebiten.Key0] = true
		case "x", "m", "1":
			keyState[koebiten.Key1] = true
		case "c", ",", "2":
			keyState[koebiten.Key2] = true
		case "v", ".", "3":
			keyState[koebiten.Key3] = true
		default:
			//fmt.Printf("undefined key : %q\n", key)
		}
		return nil
	})
}

func NewDisplay(w, h int) *Display {
	return &Display{
		w:   int16(w),
		h:   int16(h),
		buf: make([]byte, w*h*4),
	}
}

type Display struct {
	w   int16
	h   int16
	buf []byte // RGBA framebuffer, transferred to JS in one call per frame

	jsBuf     js.Value // window.screenBuffer (Uint8Array)
	jsDisplay js.Value // window.display
}

func (d *Display) Size() (x, y int16) {
	return d.w, d.h
}

func (d *Display) SetPixel(x, y int16, c color.RGBA) {
	if x < 0 || x >= d.w || y < 0 || y >= d.h {
		return
	}
	i := (int(y)*int(d.w) + int(x)) * 4
	d.buf[i] = c.R
	d.buf[i+1] = c.G
	d.buf[i+2] = c.B
	d.buf[i+3] = c.A
}

func (d *Display) Display() error {
	js.CopyBytesToJS(d.jsBuf, d.buf)
	d.jsDisplay.Invoke()
	return nil
}

func (d *Display) ClearDisplay() {
	d.ClearBuffer()
	d.Display()
}

func (d *Display) ClearBuffer() {
	for i := range d.buf {
		d.buf[i] = 0xFF // white background
	}
}

type WasmDevice struct {
}

func (w *WasmDevice) GetDisplay() koebiten.Displayer {
	return d
}

func (w *WasmDevice) Init() error {
	g := js.Global()
	d.jsBuf = g.Get("screenBuffer")
	d.jsDisplay = g.Get("display")
	return nil
}

func (w *WasmDevice) KeyUpdate() error {
	for _, key := range allKeys {
		keysBuf[0] = key
		if _, ok := keyState[key]; ok {
			koebiten.AppendPressedKeys(keysBuf[:])
			delete(keyState, key)
		} else {
			koebiten.AppendJustReleasedKeys(keysBuf[:])
		}
	}
	return nil
}
