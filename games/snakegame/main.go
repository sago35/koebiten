package main

import (
	"math/rand"
	"time"

	"github.com/sago35/koebiten"
	"github.com/sago35/koebiten/games/snakegame/snakegame"
	"github.com/sago35/koebiten/hardware"
	"tinygo.org/x/drivers/pixel"
)

var (
	white = pixel.NewMonochrome(0xFF, 0xFF, 0xFF)
	black = pixel.NewMonochrome(0x00, 0x00, 0x00)

	gridSize     = 4
	width        = 128 / gridSize
	height       = 64 / gridSize
	initialSpeed = 100 * time.Millisecond
)

func main() {
	rand.Seed(time.Now().UnixNano())
	koebiten.SetHardware(hardware.Device)
	koebiten.SetWindowSize(128, 64)
	koebiten.SetWindowTitle("Snake Game")

	game := snakegame.NewGame()
	koebiten.RunGame(game)
}
