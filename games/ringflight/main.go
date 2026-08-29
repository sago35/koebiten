package main

import (
	"log"

	"github.com/sago35/koebiten"
	"github.com/sago35/koebiten/games/ringflight/ringflight"
	"github.com/sago35/koebiten/hardware"
)

func main() {
	koebiten.SetHardware(hardware.Device)
	koebiten.SetWindowSize(128, 64)
	koebiten.SetWindowTitle("Ring Flight")

	game := ringflight.NewGame()

	if err := koebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
