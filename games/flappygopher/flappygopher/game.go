package flappygopher

import (
	"embed"
	"math/rand/v2"

	"github.com/sago35/koebiten"
	"tinygo.org/x/drivers/pixel"
)

type Game struct {
}

func NewGame() *Game {
	game := &Game{}

	for i := range wallsBuf {
		wallsBuf[i] = &wall{}
	}
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

type wall struct {
	wallX int
	holeY int
}

var (
	x    = float32(20.0)
	y    = float32(30.0)
	vy   = float32(0.0)  // Velocity of y (速度のy成分) の略
	g    = float32(0.05) // Gravity (重力加速度) の略
	jump = float32(-1.0) // ジャンプ力

	frames      = 0          // 経過フレーム数
	interval    = 120        // 壁の追加間隔
	intervalMin = 120        // 壁の追加間隔 (最小)
	wallStartX  = 200        // 壁の初期X座標
	walls       = []*wall{}  // 壁のX座標とY座標
	wallsBuf    = [8]*wall{} // 壁の実態
	wallWidth   = 7          // 壁の幅
	wallHeight  = 128        // 壁の高さ
	holeYMax    = 48         // 穴のY座標の最大値
	holeHeight  = 40         // 穴のサイズ（高さ）

	gopherWidth  = 20
	gopherHeight = 25

	scene = "title"
	score = 0
)

var (
	white = pixel.NewMonochrome(0xFF, 0xFF, 0xFF)
	black = pixel.NewMonochrome(0x00, 0x00, 0x00)
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
	koebiten.DrawImageFS(nil, fsys, "sky.png", 0, 0)
	koebiten.Println("click to start")
	koebiten.DrawImageFS(nil, fsys, "gopher.png", int(x), int(y))

	if isAnyKeyJustPressed() {
		scene = "game"
	}
}

func drawGame() {
	koebiten.DrawImageFS(nil, fsys, "sky.png", 0, 0)
	for _, wall := range walls {
		if wall.wallX == int(x) {
			score += 1
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
	interval--
	if interval == 0 {
		interval = intervalMin
		walls = wallsBuf[:len(walls)+1]
		walls[len(walls)-1].wallX = wallStartX
		walls[len(walls)-1].holeY = rand.N(holeYMax)
	}
	// 壁追加処理ここまで

	delete := false
	for _, wall := range walls {
		wall.wallX -= 1 // 少しずつ左へ
		if wall.wallX < 0 {
			delete = true
		}
	}
	if delete {
		for i := 1; i < len(walls); i++ {
			walls[i-1].wallX = walls[i].wallX
			walls[i-1].holeY = walls[i].holeY
		}
		walls = walls[:len(walls)-1]
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
			interval = intervalMin
		}

		// 下の壁を表す四角形を作る
		bLeft = wall.wallX
		bTop = wall.holeY + holeHeight
		bRight = wall.wallX + wallWidth
		bBottom = wall.holeY + holeHeight + wallHeight

		// 下の壁との当たり判定
		if hitTestRects(aLeft, aTop, aRight, aBottom, bLeft, bTop, bRight, bBottom) {
			scene = "gameover"
			interval = intervalMin
		}

		if y < 0 {
			scene = "gameover"
			interval = intervalMin
		}
		if 64 < y {
			scene = "gameover"
			interval = intervalMin
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
		interval = intervalMin
	}
}

func drawWalls(w *wall) {
	if w.wallX < 0-wallWidth || 128+wallWidth < w.wallX {
		return
	}
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
