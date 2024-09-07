package main

import (
	"embed"
	"math/rand/v2"

	"github.com/sago35/koebiten"
	"tinygo.org/x/drivers/pixel"
)

//go:embed *.png
var fsys embed.FS

type wall struct {
	wallX int
	holeY int
}

var (
	x    = 20.0
	y    = 30.0
	vy   = 0.0  // Velocity of y (速度のy成分) の略
	g    = 0.05 // Gravity (重力加速度) の略
	jump = -1.0 // ジャンプ力

	frames     = 0         // 経過フレーム数
	interval   = 120       // 壁の追加間隔
	wallStartX = 200       // 壁の初期X座標
	walls      = []*wall{} // 壁のX座標とY座標
	wallWidth  = 7         // 壁の幅
	wallHeight = 128       // 壁の高さ
	holeYMax   = 48        // 穴のY座標の最大値
	holeHeight = 40        // 穴のサイズ（高さ）

	gopherWidth  = 20
	gopherHeight = 25

	scene = "title"
	score = 0
)

var (
	white = pixel.NewMonochrome(0xFF, 0xFF, 0xFF)
	black = pixel.NewMonochrome(0x00, 0x00, 0x00)
)

func main() {
	koebiten.SetWindowSize(128, 64)
	koebiten.Run(draw)
}

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
	koebiten.DrawImageFS(nil, fsys, "sky.png", 0, 0)
	koebiten.Println("click to start")
	koebiten.DrawImageFS(nil, fsys, "gopher.png", int(x), int(y))

	if isAnyKeyJustPressed() {
		scene = "game"
	}
}

func drawGame() {
	koebiten.DrawImageFS(nil, fsys, "sky.png", 0, 0)
	for i, wall := range walls {
		if wall.wallX < int(x) {
			score = i + 1
		}
	}
	koebiten.Println("Score", score)

	if isAnyKeyJustPressed() {
		vy = jump
	}
	vy += g // 速度に加速度を足す
	y += vy // 位置に速度を足す
	koebiten.DrawImageFS(nil, fsys, "gopher.png", int(x), int(y))

	// 壁追加処理ここから
	frames += 1
	if frames%interval == 0 {
		wall := &wall{wallStartX, rand.N(holeYMax)}
		walls = append(walls, wall)
	}
	// 壁追加処理ここまで

	for _, wall := range walls {
		wall.wallX -= 1 // 少しずつ左へ
	}
	for _, wall := range walls {
		drawWalls(wall)

		// gopherくんを表す四角形を作る
		aLeft := int(x)
		aTop := int(y)
		aRight := int(x) + gopherWidth
		aBottom := int(y) + gopherHeight

		// 上の壁を表す四角形を作る
		bLeft := wall.wallX
		bTop := wall.holeY - wallHeight
		bRight := wall.wallX + wallWidth
		bBottom := wall.holeY

		// 上の壁との当たり判定
		if hitTestRects(aLeft, aTop, aRight, aBottom, bLeft, bTop, bRight, bBottom) {
			scene = "gameover"
		}

		// 下の壁を表す四角形を作る
		bLeft = wall.wallX
		bTop = wall.holeY + holeHeight
		bRight = wall.wallX + wallWidth
		bBottom = wall.holeY + holeHeight + wallHeight

		// 下の壁との当たり判定
		if hitTestRects(aLeft, aTop, aRight, aBottom, bLeft, bTop, bRight, bBottom) {
			scene = "gameover"
		}

		if y < 0 {
			scene = "gameover"
		}
		if 64 < y {
			scene = "gameover"
		}
	}
}

func drawGameover() {
	// 背景、gopher、壁の描画はdrawGame関数のコピペ
	koebiten.DrawImageFS(nil, fsys, "sky.png", 0, 0)
	koebiten.DrawImageFS(nil, fsys, "gopher.png", int(x), int(y))

	for _, wall := range walls {
		drawWalls(wall)
	}

	koebiten.Println("Game Over")
	koebiten.Println("Score", score)

	if isAnyKeyJustPressed() {
		scene = "title"

		x = 20.0
		y = 30.0
		vy = 0.0
		frames = 0
		walls = []*wall{}
		score = 0
	}
}

func drawWalls(w *wall) {
	// 上の壁の描画
	koebiten.DrawImageFS(nil, fsys, "wall.png", w.wallX, w.holeY-wallHeight)

	// 下の壁の描画
	koebiten.DrawImageFS(nil, fsys, "wall.png", w.wallX, w.holeY+holeHeight)
}

func hitTestRects(aLeft, aTop, aRight, aBottom, bLeft, bTop, bRight, bBottom int) bool {
	return aLeft < bRight &&
		bLeft < aRight &&
		aTop < bBottom &&
		bTop < aBottom
}

func isAnyKeyJustPressed() bool {
	return len(koebiten.AppendJustPressedKeys(nil)) > 0
}
