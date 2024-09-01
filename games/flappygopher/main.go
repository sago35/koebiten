package main

import (
	"embed"
	"math/rand/v2"

	//"github.com/eihigh/miniten"
	miniten "github.com/sago35/koebiten"
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

	isPrevClicked = false // 前のフレームでクリックされていたか
	isJustClicked = false // 今のフレームでクリックされたか
)

func main() {
	miniten.SetWindowSize(128, 64)
	miniten.Run(draw)
}

func draw() {
	// 今のフレームでクリックされたか = 今のフレームでクリックされていて、前のフレームでクリックされていない
	isJustClicked = miniten.IsClicked() && !isPrevClicked
	// 次のフレームに備えて、クリックされたかを保存しておく
	isPrevClicked = miniten.IsClicked()

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
	miniten.DrawImageFS(fsys, "sky.png", 0, 0)
	miniten.Println("click to start")
	miniten.DrawImageFS(fsys, "gopher.png", int(x), int(y))

	//miniten.DrawRect(0, 20, 20, 20) // 左上座標、幅、高さ
	//miniten.DrawCircle(80, 32, 30)  // 中心座標、半径
	if isJustClicked {
		scene = "game"
	}
}

func drawGame() {
	miniten.DrawImageFS(fsys, "sky.png", 0, 0)
	for i, wall := range walls {
		if wall.wallX < int(x) {
			score = i + 1
		}
	}
	//miniten.DrawRect(10, 30, 20, 20) // 左上座標、幅、高さ
	//miniten.DrawCircle(80, 32, 20)   // 中心座標、半径

	miniten.Println("Score", score)

	//if isJustClicked {
	//	scene = "title"
	//}

	if miniten.IsClicked() {
		vy = jump
	}
	vy += g // 速度に加速度を足す
	y += vy // 位置に速度を足す
	miniten.DrawImageFS(fsys, "gopher.png", int(x), int(y))

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
	miniten.DrawImageFS(fsys, "sky.png", 0, 0)
	miniten.DrawImageFS(fsys, "gopher.png", int(x), int(y))

	for _, wall := range walls {
		drawWalls(wall)
	}

	miniten.Println("Game Over")
	miniten.Println("Score", score)

	if isJustClicked {
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
	miniten.DrawImageFS(fsys, "wall.png", w.wallX, w.holeY-wallHeight)

	// 下の壁の描画
	miniten.DrawImageFS(fsys, "wall.png", w.wallX, w.holeY+holeHeight)
}

func hitTestRects(aLeft, aTop, aRight, aBottom, bLeft, bTop, bRight, bBottom int) bool {
	return aLeft < bRight &&
		bLeft < aRight &&
		aTop < bBottom &&
		bTop < aBottom
}
