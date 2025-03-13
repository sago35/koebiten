package flappygopher

import (
	"embed"
	"math/rand/v2"

	"github.com/sago35/koebiten"
	"tinygo.org/x/drivers/pixel"
)

type Game struct {
	imageGopher *koebiten.Image
	imageWall   *koebiten.Image
}

func NewGame() *Game {
	game := &Game{}

	game.imageGopher = koebiten.NewImageFromFS(fsys, "gopher.png")
	game.imageWall = koebiten.NewImageFromFS(fsys, "wall.png")

	for i := range wallsBuf {
		wallsBuf[i] = &wall{}
	}
	return game
}

// Game update process
func (game *Game) Update() error {
	return nil
}

// Screen size
func (game *Game) Layout(outsideWidth, outsideHeight int) (w, h int) {
	return 128, 64
}

func (game *Game) Draw(screen *koebiten.Image) {
	game.draw()
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

	interval      = 100        // 壁の追加間隔
	intervalMin   = 50         // 壁の追加間隔
	intervalMax   = 100        // 壁の追加間隔
	wallStartX    = 130        // 壁の初期X座標
	walls         = []*wall{}  // 壁のX座標とY座標
	wallsBuf      = [8]*wall{} // 壁の実態
	wallWidth     = 7          // 壁の幅
	wallHeight    = 128        // 壁の高さ
	holeYMax      = 48         // 穴のY座標の最大値
	holeHeight    = 45         // 穴のサイズ（高さ）
	holeHeightMin = 40         // 穴のサイズ（高さ）

	gopherWidth  = 18
	gopherHeight = 23

	scene = "title"
	score = 0
)

var (
	white = pixel.NewMonochrome(0xFF, 0xFF, 0xFF)
	black = pixel.NewMonochrome(0x00, 0x00, 0x00)
)

func (game *Game) draw() {
	switch scene {
	case "title":
		game.drawTitle()
	case "game":
		game.drawGame()
	case "gameover":
		game.drawGameover()
	}
}

func (game *Game) drawTitle() {
	koebiten.Println("click to start")
	op := koebiten.DrawImageOptions{}
	op.GeoM.Translate(float32(x), float32(y))
	game.imageGopher.DrawImage(nil, op)

	if isAnyKeyJustPressed() {
		scene = "game"
	}
}

func (game *Game) drawGame() {
	for _, wall := range walls {
		if wall.wallX == int(x) {
			score += 1

			if score%5 == 0 {
				if holeHeightMin <= holeHeight {
					holeHeight -= 1
				}
			}
		}
	}
	koebiten.Println("Score", score)

	if isAnyKeyJustPressed() {
		vy = jump
	}
	vy += g // 速度に加速度を足す
	y += vy // 位置に速度を足す
	op := koebiten.DrawImageOptions{}
	op.GeoM.Translate(float32(x), float32(y))
	game.imageGopher.DrawImage(nil, op)

	// 壁追加処理ここから
	interval--
	if interval == 0 {
		interval = intervalMin + rand.N(intervalMax-intervalMin)
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
		game.drawWalls(wall)

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

func (game *Game) drawGameover() {
	if interval > 0 {
		interval--
		op := koebiten.DrawImageOptions{}
		op.GeoM.Translate(float32(x), float32(y))
		game.imageGopher.DrawImage(nil, op)

		for _, wall := range walls {
			game.drawWalls(wall)
		}
	}

	koebiten.Println("Game Over")
	koebiten.Println("Score", score)

	if interval == 0 && isAnyKeyJustPressed() {
		scene = "title"

		x = 20.0
		y = 30.0
		vy = 0.0
		walls = wallsBuf[:0]
		score = 0
		interval = intervalMax
	}
}

func (game *Game) drawWalls(w *wall) {
	if w.wallX < 0-wallWidth || 128+wallWidth < w.wallX {
		return
	}
	// upper wall
	op1 := koebiten.DrawImageOptions{}
	op1.GeoM.Translate(float32(w.wallX), float32(w.holeY-wallHeight))
	game.imageWall.DrawImage(nil, op1)

	// lower wall
	op2 := koebiten.DrawImageOptions{}
	op2.GeoM.Translate(float32(w.wallX), float32(w.holeY+holeHeight))
	game.imageWall.DrawImage(nil, op2)
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
