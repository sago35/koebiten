package jumpingopher

import (
	"embed"
	"math/rand/v2"

	"github.com/sago35/koebiten"
)

type Game struct {
}

func NewGame() *Game {
	game := &Game{}
	return game
}

// Game update process
func (g *Game) Update() error {
	return nil
}

// Screen size
func (g *Game) Layout(outsideWidth, outsideHeight int) (w, h int) {
	return 128, 64
}

func (g *Game) Draw(screen *koebiten.Image) {
	draw()
}

//go:embed *.png
var fsys embed.FS

type cloud struct {
	cloudX int
	holeY  int
}

type platform struct {
	pY int
}

var (
	x            = 50.0
	y            = 30.0
	vy           = 0.0
	g            = 0.05
	jump         = -1.0
	frames       = 30
	interval     = 120
	cloudStartX  = 200
	clouds       = []*cloud{}
	cloudX       = 20
	holeYMax     = 48
	cloudHeight  = 8
	platforms    = []*platform{{pY: 60}}
	scene        = "title"
	score        = 0
	isOnPlatform = false
)

func draw() {
	switch scene {
	case "title":
		drawTitle()
	case "game":
		drawGame()
	case "gameover":
		drawGameover()
	}
}

func drawTitle() {
	koebiten.Println("click to start")
	if isAnyKeyJustPressed() {
		scene = "game"
	}
}

func drawGame() {
	koebiten.DrawImageFS(nil, fsys, "sky.png", 0, 0)
	koebiten.Println("Score", score)
	koebiten.DrawImageFS(nil, fsys, "platform.png", 0, 60)

	if y <= -10.0 {
		scene = "gameover"
	}

	for i, cloud := range clouds {
		if cloud.cloudX < int(x) {
			score = i + 1
		}
	}

	if isAnyKeyJustPressed() {
		vy = jump
		isOnPlatform = false
	}

	vy += g // 速度に加速度を足す
	y += vy // 位置に速度を足す

	for _, platform := range platforms {
		if hitPlatformRect(int(y), int(x), int(y)+22, int(x)+22, platform.pY, 0, platform.pY+22, 128) {
			isOnPlatform = true
		}
	}

	if isOnPlatform {
		vy = 0
		y = 33.5
	}

	frames++
	walkGopher(x, y, frames)

	if frames%interval == 0 {
		cloud := &cloud{cloudStartX, rand.N(holeYMax)}
		clouds = append(clouds, cloud)
	}

	for _, cloud := range clouds {
		cloud.cloudX -= 1 // 少しずつ左へ
		drawWalls(cloud)

		if hitRects(int(x), int(y), int(x)+20, int(y)+8, cloud.cloudX, cloud.holeY-cloudHeight, cloud.cloudX+cloudX, cloud.holeY+cloudHeight) {
			scene = "gameover"
		}
	}
}

func drawGameover() {
	koebiten.Println("Game Over")
	koebiten.Println("Score:", score)

	if isAnyKeyJustPressed() {
		scene = "title"
		x, y, vy, score = 50.0, 30.0, 0.0, 0
		clouds = []*cloud{}
	}
}

func drawWalls(c *cloud) {
	koebiten.DrawImageFS(nil, fsys, "cloud.png", c.cloudX, c.holeY-cloudHeight)
}

func walkGopher(x, y float64, frames int) {
	img := "gopher.png"
	if frames%2 != 0 {
		img = "gopher_r.png"
	}
	koebiten.DrawImageFS(nil, fsys, img, int(x), int(y))
}

func hitRects(aLeft, aTop, aRight, aBottom, bLeft, bTop, bRight, bBottom int) bool {
	return aTop < bBottom && bTop < aBottom && aLeft < bRight && bLeft < aRight
}

func hitPlatformRect(aTop, aLeft, aBottom, aRight, bTop, bLeft, bBottom, bRight int) bool {
	return aTop < bBottom && bTop < aBottom && aLeft < bRight && bLeft < aRight
}

func isAnyKeyJustPressed() bool {
	return len(koebiten.AppendJustPressedKeys(nil)) > 0
}
