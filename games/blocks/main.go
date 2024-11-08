package main

import (
	"log"

	"github.com/sago35/koebiten"
	"github.com/sago35/koebiten/games/blocks/blocks"
	"github.com/sago35/koebiten/hardware"
)

func main() {
	koebiten.SetHardware(hardware.Device)
	koebiten.SetRotation(koebiten.Rotation90)
	koebiten.SetWindowSize(64, 128)
	koebiten.SetWindowTitle("Tetris in Go")

	game := blocks.NewGame()

	if err := koebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
