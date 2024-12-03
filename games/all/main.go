package main

import (
	"log"

	"github.com/sago35/koebiten"
	"github.com/sago35/koebiten/games/all/all"
	"github.com/sago35/koebiten/hardware"
)

func main() {
	koebiten.SetHardware(hardware.Device)
	koebiten.SetWindowSize(64, 128)
	koebiten.SetWindowTitle("All")

	game := all.NewGame()

	for {
		koebiten.SetRotation(0)
		if err := koebiten.RunGame(game); err != nil {
			log.Fatal(err)
		}

		game.RunCurrentGame()
	}
}
