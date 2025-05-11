package goradius

import (
	"embed"
	"math/rand/v2"
	"slices"

	"github.com/sago35/koebiten"
	"tinygo.org/x/drivers/pixel"
)

//go:embed *.png
var fsys embed.FS

const (
	gameStateStart = iota
	gameStatePlaying
	gameStateGameOver

	width  = 128
	height = 64

	gopherWidth  = 20
	gopherHeight = 25

	beamMax      = 1
	beamCooldown = 60
)

type Game struct {
	gopher            *koebiten.Image
	x, y              int
	scale             float32
	theta             float32
	score             int
	beamEnergy        int
	beamActive        bool // ビームが有効かどうかのフラグ
	beamCooldownTimer int  // ビームのクールダウンタイマー

	gameState int // ゲームの状態を管理する変数
}

type enemy struct {
	enemyX int
	enemyY int
	speed  int // 敵の移動速度
}

var (
	white = pixel.NewMonochrome(0xFF, 0xFF, 0xFF)
	black = pixel.NewMonochrome(0x00, 0x00, 0x00)

	x = float32(20.0)
	y = float32(30.0)

	frames   = 30         // フレーム数
	interval = 120        // 敵の追加間隔
	enemie   = []*enemy{} // 敵のX座標とY座標
)

func NewGame() *Game {
	game := &Game{
		gopher: koebiten.NewImageFromFS(fsys, "gopher.png"),
		x:      width / 2,
		y:      height / 2,
		scale:  1,
	}
	return game
}

// Game update process
func (g *Game) Update() error {
	ds := float32(0.05)
	dt := float32(0.2)
	speed := 1
	dx := 1 * speed
	dy := 1 * speed

	// スタート画面からゲームプレイ画面に遷移
	if koebiten.IsKeyPressed(koebiten.KeyRotaryButton) {
		g.gameState = gameStatePlaying
	}

	// rotary buttonを回すとgopherが回転する
	// キーボードを押しながら回すと拡大縮小する
	if koebiten.IsKeyPressed(koebiten.KeyRotaryRight) {
		if isAnyKeyboardKeyPressed() {
			g.scale += ds
		} else {
			g.theta += dt
		}
	}
	if koebiten.IsKeyPressed(koebiten.KeyRotaryLeft) {
		if isAnyKeyboardKeyPressed() {
			g.scale -= ds
		} else {
			g.theta -= dt
		}
	}

	// joystickを倒すとgopherが移動する
	if koebiten.IsKeyPressed(koebiten.KeyArrowRight) {
		if g.x < width {
			g.x += dx
		}
	}
	if koebiten.IsKeyPressed(koebiten.KeyArrowLeft) {
		if g.x > -5 {
			g.x -= dx
		}
	}
	if koebiten.IsKeyPressed(koebiten.KeyArrowDown) {
		if g.y <= height {
			g.y += dy
		}
	}
	if koebiten.IsKeyPressed(koebiten.KeyArrowUp) {
		if g.y > -5 {
			g.y -= dy
		}
	}

	// key0を押すとデバッグ情報を表示する
	if koebiten.IsKeyPressed(koebiten.Key0) {
		koebiten.Println("Goradius")
		koebiten.Println("x:", g.x, "y:", g.y)
		koebiten.Println("beam:", g.beamEnergy)
		koebiten.Println("Score:", g.score)
	}

	// ビームを発射する
	if g.beamCooldownTimer > 0 {
		// クールダウン中はタイマーを減らす
		g.beamCooldownTimer--
		g.beamActive = false
	} else if koebiten.IsKeyPressed(koebiten.Key1) {
		// クールダウンが終わっていて、キーが押されていればビーム発射
		g.beamEnergy++
		if g.beamEnergy <= beamMax {
			g.beamActive = true
		} else {
			// ビームエネルギーが上限に達したらクールダウン開始
			g.beamActive = false
			g.beamEnergy = 0
			g.beamCooldownTimer = beamCooldown
		}
	} else {
		// Key1が押されていない場合はビームを無効化
		g.beamActive = false
		g.beamEnergy = 0
	}

	// 敵の移動
	for i := 0; i < len(enemie); i++ {
		if i < len(enemie) { // 境界チェック
			enemie[i].enemyX -= 1 // 少しずつ左へ

			// 画面外に出た敵を削除
			if enemie[i].enemyX < -gopherWidth {
				enemie = append(enemie[:i], enemie[i+1:]...)
				i--
			}
		}
	}

	return nil
}

func (g *Game) drawEnemy(e *enemy) {
	// 敵を描画する
	koebiten.DrawImageFS(nil, fsys, "enemy.png", e.enemyX, e.enemyY)
	// 自機との当たり判定
	if hitRects(g.x, g.y, g.x+gopherWidth, g.y+gopherHeight,
		e.enemyX, e.enemyY, e.enemyX+gopherWidth, e.enemyY+gopherHeight) {
		// 当たった場合、ゲームオーバー
		g.gameState = gameStateGameOver
	}
}

func (g *Game) drawTitle() {
	// タイトル画面を描画する
	koebiten.Println("Goradius")
	koebiten.Println("Press any key to start")
}

func (g *Game) drawGameOver() {
	// ゲームオーバー画面を描画する
	koebiten.Println("Game Over")
	koebiten.Println("Score:", g.score)
	koebiten.Println("Press any key to restart")
}

func (g *Game) drawGame(screen *koebiten.Image) {
	// 自機描画
	op := koebiten.DrawImageOptions{}
	op.GeoM.Translate(-float32(gopherWidth)/2, -float32(gopherHeight)/2)
	op.GeoM.Scale(g.scale, g.scale)
	op.GeoM.Rotate(g.theta)
	op.GeoM.Translate(float32(g.x), float32(g.y))
	g.gopher.DrawImage(screen, op)

	// 一定間隔で敵を追加
	frames++
	if rand.N(interval) < 3 {
		enemyY := rand.N(height - gopherHeight)
		// スピードもランダムに（1〜3の範囲）
		enemySpeed := rand.N(3) + 1
		enemy := &enemy{width, enemyY, enemySpeed}
		enemie = append(enemie, enemy)
	}

	// 敵の描画
	for i := 0; i < len(enemie); i++ {
		if i < len(enemie) { // 境界チェック
			g.drawEnemy(enemie[i])
		}
	}

	// ビームを描画（ビームが有効な場合）
	if g.beamActive && g.beamEnergy <= beamMax {
		for i := 0; i < 10; i++ {
			beamX := g.x + (i * 10)
			beamY := g.y
			koebiten.DrawImageFS(nil, fsys, "beam.png", beamX, beamY)

			// 各ビームセグメントと敵の当たり判定
			for j := 0; j < len(enemie); j++ {
				if j < len(enemie) { // 配列の境界チェック
					if hitBeam(beamX, beamY, beamX+gopherWidth, beamY+gopherHeight,
						enemie[j].enemyX, enemie[j].enemyY, enemie[j].enemyX+gopherWidth, enemie[j].enemyY+gopherHeight) {

						// 敵が当たった場合、敵を削除して得点加算
						g.score++

						// 敵をスライスから削除
						enemie = append(enemie[:j], enemie[j+1:]...)
						j-- // 削除したので、インデックスを1つ戻す
						break
					}
				}
			}
		}
	}
}

// Screen size
func (g *Game) Layout(outsideWidth, outsideHeight int) (w, h int) {
	return width, height
}

func (g *Game) Draw(screen *koebiten.Image) {
	switch g.gameState {
	case gameStateStart:
		g.drawTitle()
	case gameStatePlaying:
		g.drawGame(screen)
	case gameStateGameOver:
		g.drawGameOver()
	default:
		g.drawTitle()
	}
}

// isAnyKeyboardKeyPressed returns true if any keyboard key is pressed
//
// keyboard key are koebiten.Key0 to koebiten.Key11
func isAnyKeyboardKeyPressed() bool {
	return slices.ContainsFunc(koebiten.AppendPressedKeys(nil), func(k koebiten.Key) bool {
		switch k {
		case
			koebiten.Key0,
			koebiten.Key1,
			koebiten.Key2,
			koebiten.Key3,
			koebiten.Key4,
			koebiten.Key5,
			koebiten.Key6,
			koebiten.Key7,
			koebiten.Key8,
			koebiten.Key9,
			koebiten.Key10,
			koebiten.Key11:
			return true
		default:
			return false
		}
	})
}

// キャラと敵の当たり判定で、場所が重なっているかどうかを判定する
func hitRects(aLeft, aTop, aRight, aBottom, bLeft, bTop, bRight, bBottom int) bool {
	return aTop < bBottom && bTop < aBottom && aLeft < bRight && bLeft < aRight
}

func hitBeam(aLeft, aTop, aRight, aBottom, bLeft, bTop, bRight, bBottom int) bool {
	return aTop < bBottom && bTop < aBottom && aLeft < bRight && bLeft < aRight
}
