package main

import (
	"embed"
	"math/rand/v2"

	"github.com/sago35/koebiten"
)

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
	x    = 50.0
	y    = 30.0
	vy   = 0.0
	g    = 0.05
	jump = -1.0

	frames      = 30
	interval    = 120
	cloudStartX = 200
	clouds      = []*cloud{}
	cloudX      = 20
	holeYMax    = 48
	cloudHeight = 8

	platforms = []*platform{
		{pY: 60},
	}

	scene = "title"
	score = 0

	isOnPlatform  = false
	isJustClicked = false
	isPrevClicked = false
)

func main() {
	koebiten.SetWindowSize(128, 64)
	koebiten.Run(draw)
}

func draw() {
	isJustClicked = koebiten.IsClicked() && !isPrevClicked
	isPrevClicked = koebiten.IsClicked()

	koebiten.DrawImageFS(fsys, "sky.png", 0, 0)

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

	if isJustClicked {
		scene = "game"
	}
}

func drawGame() {
	koebiten.DrawImageFS(fsys, "sky.png", 0, 0)
	koebiten.Println("Score", score)
	koebiten.DrawImageFS(fsys, "platform.png", 0, 60)

	// 高く上がりすぎてもゲームオーバー
	if y <= -10.0 {
		scene = "gameover"
	}

	for i, cloud := range clouds {
		if cloud.cloudX < int(x) {
			score = i + 1
		}
	}

	if koebiten.IsClicked() {
		vy = jump
		isOnPlatform = false
	}

	vy += g // 速度に加速度を足す
	y += vy // 位置に速度を足す

	for _, platform := range platforms {
		// プラットフォームとの当たり判定
		if hitPlatformRect(int(y), int(x), int(y)+22, int(x)+22, platform.pY, 0, platform.pY+22, 128) {
			isOnPlatform = true
		}
	}

	if isOnPlatform {
		vy = 0
		y = 33.5
	}

	frames += 1

	walkGopher(x, y, frames)

	if frames%interval == 0 {
		cloud := &cloud{cloudStartX, rand.N(holeYMax)}
		clouds = append(clouds, cloud)
	}
	for _, cloud := range clouds {
		cloud.cloudX -= 1 // 少しずつ左へ
	}
	for _, cloud := range clouds {
		drawWalls(cloud)

		aLeft := int(x)
		aTop := int(y)
		aRight := int(x) + 20
		aBottom := int(y) + 8

		bLeft := cloud.cloudX
		bTop := cloud.holeY - cloudHeight
		bRight := cloud.cloudX + cloudX
		bBottom := cloud.holeY + cloudHeight

		// 上の壁との当たり判定
		if hitRects(aLeft, aTop, aRight, aBottom, bLeft, bTop, bRight, bBottom) {
			scene = "gameover"
		}
	}
}

func drawGameover() {
	koebiten.Println("Game Over")
	koebiten.Println("Score:", score)

	if isJustClicked {
		scene = "title"

		x = 50.0
		y = 30.0
		vy = 0.0
		score = 0
		clouds = []*cloud{}
	}
}

func drawWalls(c *cloud) {
	// 上の壁の描画
	koebiten.DrawImageFS(fsys, "cloud.png", c.cloudX, c.holeY-cloudHeight)
}

func walkGopher(x, y float64, frames int) {
	if frames%2 == 0 {
		// 画像を切り替える
		koebiten.DrawImageFS(fsys, "gopher.png", int(x), int(y))
	} else {
		koebiten.DrawImageFS(fsys, "gopher_r.png", int(x), int(y))
	}
}

func hitRects(aLeft, aTop, aRight, aBottom, bLeft, bTop, bRight, bBottom int) bool {
	return aTop < bBottom &&
		bTop < aBottom &&
		aLeft < bRight &&
		bLeft < aRight
}

func hitPlatformRect(aTop, aLeft, aBottom, aRight, bTop, bLeft, bBottom, bRight int) bool {
	return aTop < bBottom &&
		bTop < aBottom &&
		aLeft < bRight &&
		bLeft < aRight
}
