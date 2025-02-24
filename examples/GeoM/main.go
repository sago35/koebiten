package main

import (
	"log"

	"github.com/sago35/koebiten"

	"github.com/sago35/koebiten/examples/GeoM/geom"
	"github.com/sago35/koebiten/hardware"
)

func main() {
	koebiten.SetHardware(hardware.Device)
	koebiten.SetWindowSize(128, 64)
	koebiten.SetWindowTitle("GeoM Gopher")

	game := geom.NewGame()

	if err := koebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
