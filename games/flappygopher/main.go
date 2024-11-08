package main

import (
	"log"

	"github.com/sago35/koebiten"
	"github.com/sago35/koebiten/games/flappygopher/flappygopher"
	"github.com/sago35/koebiten/hardware"
)

func main() {
	koebiten.SetHardware(hardware.Device)
	koebiten.SetWindowSize(128, 64)
	koebiten.SetWindowTitle("Flappy Gopher")

	game := flappygopher.NewGame()

	if err := koebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
