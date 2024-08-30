package main

import (
	"embed"
	"math/rand/v2"

	"github.com/eihigh/miniten"
)

//go:embed *.png
var fsys embed.FS

var (
	x    = 20.0
	y    = 30.0
	vy   = 0.0  // Velocity of y (速度のy成分) の略
	g    = 0.05 // Gravity (重力加速度) の略
	jump = -1.0 // ジャンプ力

	frames     = 0       // 経過フレーム数
	interval   = 120     // 壁の追加間隔
	wallStartX = 200     // 壁の初期X座標
	wallXs     = []int{} // 壁のX座標
	wallWidth  = 7       // 壁の幅
	wallHeight = 128     // 壁の高さ
	holeYs     = []int{} // 穴のY座標
	holeYMax   = 48      // 穴のY座標の最大値
	holeHeight = 40      // 穴のサイズ（高さ）

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
	//miniten.DrawImageFS(fsys, "sky.png", 0, 0)
	miniten.Println("click to start")
	miniten.DrawImageFS(fsys, "gopher.png", int(x), int(y))

	//miniten.DrawRect(0, 20, 20, 20) // 左上座標、幅、高さ
	//miniten.DrawCircle(80, 32, 30)  // 中心座標、半径
	if isJustClicked {
		scene = "game"
	}
}

func drawGame() {
	//miniten.DrawImageFS(fsys, "sky.png", 0, 0)
	for i, wallX := range wallXs {
		if wallX < int(x) {
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
		wallXs = append(wallXs, wallStartX)
		holeYs = append(holeYs, rand.N(holeYMax))
	}
	// 壁追加処理ここまで

	for i := range wallXs {
		wallXs[i] -= 1 // 少しずつ左へ
	}
	for i := range wallXs {
		// 上の壁の描画
		wallX := wallXs[i]
		holeY := holeYs[i]
		miniten.DrawImageFS(fsys, "wall.png", wallX, holeY-wallHeight)

		// 下の壁の描画
		miniten.DrawImageFS(fsys, "wall.png", wallX, holeY+holeHeight)

		// gopherくんを表す四角形を作る
		aLeft := int(x)
		aTop := int(y)
		aRight := int(x) + gopherWidth
		aBottom := int(y) + gopherHeight

		// 上の壁を表す四角形を作る
		bLeft := wallX
		bTop := holeY - wallHeight
		bRight := wallX + wallWidth
		bBottom := holeY

		// 上の壁との当たり判定
		if aLeft < bRight &&
			bLeft < aRight &&
			aTop < bBottom &&
			bTop < aBottom {
			scene = "gameover"
		}

		// 下の壁を表す四角形を作る
		bLeft = wallX
		bTop = holeY + holeHeight
		bRight = wallX + wallWidth
		bBottom = holeY + holeHeight + wallHeight

		// 下の壁との当たり判定
		if aLeft < bRight &&
			bLeft < aRight &&
			aTop < bBottom &&
			bTop < aBottom {
			scene = "gameover"
		}

		if y < 0 {
			scene = "gameover"
		}
		if 240 < y {
			scene = "gameover"
		}
	}
}

func drawGameover() {
	// 背景、gopher、壁の描画はdrawGame関数のコピペ
	//miniten.DrawImageFS(fsys, "sky.png", 0, 0)
	miniten.DrawImageFS(fsys, "gopher.png", int(x), int(y))

	for i := range wallXs {
		// 上の壁の描画
		wallX := wallXs[i]
		holeY := holeYs[i]
		miniten.DrawImageFS(fsys, "wall.png", wallX, holeY-wallHeight)

		// 下の壁の描画
		miniten.DrawImageFS(fsys, "wall.png", wallX, holeY+holeHeight)
	}

	miniten.Println("Game Over")
	miniten.Println("Score", score)

	if isJustClicked {
		scene = "title"

		x = 20.0
		y = 30.0
		vy = 0.0
		frames = 0
		wallXs = []int{}
		holeYs = []int{}
		score = 0
	}
}
