package geom

import (
	"embed"
	"slices"

	"github.com/sago35/koebiten"
	"tinygo.org/x/drivers/pixel"
)

//go:embed *.png
var fsys embed.FS

const (
	width  = 128
	height = 64

	gopherWidth  = 20
	gopherHeight = 25
)

type Game struct {
	x, y int

	scale float64
	theta float64
}

var (
	white = pixel.NewMonochrome(0xFF, 0xFF, 0xFF)
	black = pixel.NewMonochrome(0x00, 0x00, 0x00)
)

func NewGame() *Game {
	game := &Game{
		x:     width / 2,
		y:     height / 2,
		scale: 1,
	}
	return game
}

// Game update process
func (g *Game) Update() error {
	ds := 0.05
	dt := 0.2
	dx := 1
	dy := 1

	if koebiten.IsKeyPressed(koebiten.KeyRotaryRight) {
		if isAnyKeyboardKeyPressed() {
			g.scale += ds
		} else {
			g.theta += dt
		}
	}
	if koebiten.IsKeyPressed(koebiten.KeyRotaryLeft) {
		if isAnyKeyboardKeyPressed() {
			g.scale -= ds
		} else {
			g.theta -= dt
		}
	}

	if koebiten.IsKeyPressed(koebiten.KeyArrowRight) {
		g.x += dx
	}
	if koebiten.IsKeyPressed(koebiten.KeyArrowLeft) {
		g.x -= dx
	}
	if koebiten.IsKeyPressed(koebiten.KeyArrowDown) {
		g.y += dy
	}
	if koebiten.IsKeyPressed(koebiten.KeyArrowUp) {
		g.y -= dy
	}

	return nil
}

// Screen size
func (g *Game) Layout(outsideWidth, outsideHeight int) (w, h int) {
	return width, height
}

func (g *Game) Draw(screen *koebiten.Image) {
	op := koebiten.DrawImageFSOptions{}
	op.GeoM.Translate(-float64(gopherWidth)/2, -float64(gopherHeight)/2)
	op.GeoM.Scale(g.scale, g.scale)
	op.GeoM.Rotate(g.theta)
	op.GeoM.Translate(float64(g.x), float64(g.y))
	koebiten.DrawImageFSWithOptions(nil, fsys, "gopher.png", op)
}

func isAnyKeyboardKeyPressed() bool {
	return slices.ContainsFunc(koebiten.AppendPressedKeys(nil), func(k koebiten.Key) bool {
		switch k {
		case
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
			koebiten.Key11:
			return true
		default:
			return false
		}
	})
}
