package drawing

import (
	"github.com/sago35/koebiten"
	"tinygo.org/x/drivers/pixel"
)

var (
	white = pixel.NewMonochrome(0xFF, 0xFF, 0xFF)
	black = pixel.NewMonochrome(0x00, 0x00, 0x00)
)

type Pointer struct {
	x, y int
}

type Game struct {
	pointer Pointer
	thick   int
	canvas  *koebiten.Image
}

func NewGame() *Game {
	return &Game{
		pointer: Pointer{64, 32},
		canvas:  koebiten.NewImage(128, 64),
	}
}

func (g *Game) Update() error {
	if koebiten.IsKeyPressed(koebiten.KeyArrowLeft) {
		g.pointer.x--
	}
	if koebiten.IsKeyPressed(koebiten.KeyArrowRight) {
		g.pointer.x++
	}
	if koebiten.IsKeyPressed(koebiten.KeyArrowUp) {
		g.pointer.y--
	}
	if koebiten.IsKeyPressed(koebiten.KeyArrowDown) {
		g.pointer.y++
	}

	if koebiten.IsKeyPressed(koebiten.KeyRotaryLeft) {
		g.thick--
		if g.thick < 0 {
			g.thick = 0
		}
	}
	if koebiten.IsKeyPressed(koebiten.KeyRotaryRight) {
		g.thick++
		if g.thick > 10 {
			g.thick = 10
		}
	}

	if koebiten.IsKeyPressed(koebiten.Key0) {
		g.draw(g.canvas, g.pointer.x, g.pointer.y, black)
	}
	if koebiten.IsKeyPressed(koebiten.Key1) || koebiten.IsKeyPressed(koebiten.Key2) ||
		koebiten.IsKeyPressed(koebiten.Key3) || koebiten.IsKeyPressed(koebiten.Key4) ||
		koebiten.IsKeyPressed(koebiten.Key5) || koebiten.IsKeyPressed(koebiten.Key6) ||
		koebiten.IsKeyPressed(koebiten.Key7) || koebiten.IsKeyPressed(koebiten.Key8) ||
		koebiten.IsKeyPressed(koebiten.Key9) || koebiten.IsKeyPressed(koebiten.Key10) ||
		koebiten.IsKeyPressed(koebiten.Key11) {
		g.draw(g.canvas, g.pointer.x, g.pointer.y, white)
	}

	return nil
}

func (g *Game) Draw(screen *koebiten.Image) {
	g.canvas.DrawImage(screen, koebiten.DrawImageOptions{})
	koebiten.DrawFilledCircle(screen, g.pointer.x, g.pointer.y, g.thick+1, white)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 128, 64
}

func (g *Game) draw(canvas *koebiten.Image, x, y int, color pixel.BaseColor) {
	koebiten.DrawFilledCircle(g.canvas, x, y, g.thick, color)
}

func isAnyKeyPressed() bool {
	keys := []koebiten.Key{
		koebiten.Key0, koebiten.Key1, koebiten.Key2, koebiten.Key3,
		koebiten.Key4, koebiten.Key5, koebiten.Key6, koebiten.Key7,
		koebiten.Key8, koebiten.Key9, koebiten.Key10, koebiten.Key11,
	}
	for _, key := range keys {
		if koebiten.IsKeyPressed(key) {
			return true
		}
	}
	return false
}
