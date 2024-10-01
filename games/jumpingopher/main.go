package main

import (
	"log"

	"github.com/sago35/koebiten"
	"github.com/sago35/koebiten/games/jumpingopher/jumpingopher"
)

func main() {
	koebiten.SetWindowSize(128, 64)
	koebiten.SetWindowTitle("Jumpin Gopher")
	game := jumpingopher.NewGame()

	if err := koebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
