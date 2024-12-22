package all

import (
	"log"

	"github.com/sago35/koebiten"
	"github.com/sago35/koebiten/games/blocks/blocks"
	"github.com/sago35/koebiten/games/flappygopher/flappygopher"
	"github.com/sago35/koebiten/games/jumpingopher/jumpingopher"
)

type Game struct {
	Title string
	Game  func()
}

type Menu struct {
	index int
	games []Game
}

func NewGame() *Menu {
	menu := &Menu{
		index: 0,
	}

	menu.AddGames([]Game{
		{
			Title: "Flappy Gopher",
			Game: func() {
				koebiten.SetRotation(koebiten.Rotation0)
				game := flappygopher.NewGame()
				if err := koebiten.RunGame(game); err != nil {
					log.Fatal(err)
				}
			},
		},
		{
			Title: "Blocks",
			Game: func() {
				koebiten.SetRotation(koebiten.Rotation90)
				game := blocks.NewGame()
				if err := koebiten.RunGame(game); err != nil {
					log.Fatal(err)
				}
			},
		},
		{
			Title: "Jumpin Gopher",
			Game: func() {
				koebiten.SetRotation(koebiten.Rotation0)
				game := jumpingopher.NewGame()
				if err := koebiten.RunGame(game); err != nil {
					log.Fatal(err)
				}
			},
		},
	})

	return menu
}

func (m *Menu) Update() error {
	if koebiten.IsKeyJustPressed(koebiten.KeyDown) || koebiten.IsKeyJustPressed(koebiten.KeyRotaryRight) {
		m.index = (m.index + 1) % len(m.games)
	} else if koebiten.IsKeyJustPressed(koebiten.KeyUp) || koebiten.IsKeyJustPressed(koebiten.KeyRotaryLeft) {
		m.index = (m.index - 1 + len(m.games)) % len(m.games)
	} else if len(koebiten.AppendJustPressedKeys(nil)) > 0 {
		return koebiten.Termination
	}
	return nil
}

// Screen size
func (m *Menu) Layout(outsideWidth, outsideHeight int) (w, h int) {
	return 128, 64
}

func (m *Menu) Draw(screen *koebiten.Image) {
	koebiten.Println("select game :")
	koebiten.Println(m.games[m.index].Title)
}

func (m *Menu) AddGames(game []Game) {
	m.games = append(m.games, game...)
}

func (m *Menu) RunCurrentGame() {
	m.games[m.index].Game()
}
