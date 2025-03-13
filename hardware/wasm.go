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
		w: int16(w),
		h: int16(h),
	}
}

type Display struct {
	w int16
	h int16
}

func (d *Display) Size() (x, y int16) {
	return d.w, d.h
}

func (d *Display) SetPixel(x, y int16, c color.RGBA) {
	js.Global().Call("setPixel", x, y, c.R, c.G, c.B, c.A)
}

func (d *Display) Display() error {
	js.Global().Call("display")
	return nil
}

func (d *Display) ClearDisplay() {
	js.Global().Call("clearScreen")
}

func (d *Display) ClearBuffer() {
	js.Global().Call("clearScreen")
}

type WasmDevice struct {
}

func (w *WasmDevice) GetDisplay() koebiten.Displayer {
	return d
}

func (w *WasmDevice) Init() error {
	return nil
}

func (w *WasmDevice) KeyUpdate() error {
	keys := []koebiten.Key{
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

	for _, key := range keys {
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
