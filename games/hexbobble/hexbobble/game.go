package hexbobble

import (
	"math"
	"math/rand/v2"

	"github.com/sago35/koebiten"
	"tinygo.org/x/drivers/pixel"
)

// パズルボブルを 90 度回転させたゲーム。
// 左の砲台から球を発射し、球は上下の壁で反射しながら右へ進む。
// 右壁または既存の球に当たると、不可視のヘックスグリッドに吸着する。
// 同種の球が 3 つ以上つながると消える。

const (
	screenW = 128
	screenH = 64

	maxCols     = 18 // 右壁からの列数(col 0 が最も右)
	maxRows     = 8  // 偶数列の行数(奇数列は 7)
	gameOverCol = 15 // この列以降に球が固定されたらゲームオーバー

	numTypes = 6

	colPitch = 7 // 列の x 間隔
	rowPitch = 8 // 行の y 間隔(奇数列は +4 オフセット)
	ballR    = 3 // 描画半径(直径 7)

	shooterX = 8
	shooterY = 32
	angleMax = 72 // 発射角の上限(度)

	speed = 3.0 // 球速(px/フレーム)
)

var (
	white = pixel.NewMonochrome(0xFF, 0xFF, 0xFF)
	black = pixel.NewMonochrome(0x00, 0x00, 0x00)
)

type Game struct {
	grid  [maxCols][maxRows]int8 // 球の種類。-1 は空
	angle int                    // 発射角(度)。0 で真右、負で上向き

	cur, next int8 // 現在装填中の球と次の球

	flying   bool
	fx, fy   float32
	fvx, fvy float32
	ftype    int8

	// マッチ消去エフェクト(拡大するリング)
	pops [maxCols * maxRows]popFx
	nPop int
	// 孤立球の落下エフェクト
	falls [maxCols * maxRows]fallFx
	nFall int

	score int
	scene string
}

type popFx struct {
	x, y  int16
	frame int8
}

type fallFx struct {
	t     int8
	x     int16
	y, vy float32
}

func NewGame() *Game {
	g := &Game{}
	g.reset()
	g.scene = "title"
	return g
}

func (g *Game) reset() {
	for c := range g.grid {
		for r := range g.grid[c] {
			g.grid[c][r] = -1
		}
	}
	// 初期配置: 右端 4 列を埋める
	for c := 0; c < 4; c++ {
		for r := 0; r < rowsInCol(c); r++ {
			g.grid[c][r] = int8(rand.N(numTypes))
		}
	}
	g.angle = 0
	g.cur = int8(rand.N(numTypes))
	g.next = int8(rand.N(numTypes))
	g.flying = false
	g.nPop = 0
	g.nFall = 0
	g.score = 0
}

// --- ヘックスグリッド ---

func rowsInCol(c int) int {
	if c%2 == 0 {
		return maxRows
	}
	return maxRows - 1
}

func colX(c int) int {
	return screenW - 4 - colPitch*c
}

func rowY(c, r int) int {
	if c%2 == 0 {
		return 4 + rowPitch*r
	}
	return 8 + rowPitch*r
}

func validCell(c, r int) bool {
	return 0 <= c && c < maxCols && 0 <= r && r < rowsInCol(c)
}

// neighbors は (c, r) に隣接するヘックスセルを返す(最大 6 個)
func neighbors(c, r int) ([6][2]int, int) {
	var nb [6][2]int
	n := 0
	add := func(nc, nr int) {
		if validCell(nc, nr) {
			nb[n] = [2]int{nc, nr}
			n++
		}
	}
	add(c, r-1)
	add(c, r+1)
	if c%2 == 0 {
		add(c-1, r-1)
		add(c-1, r)
		add(c+1, r-1)
		add(c+1, r)
	} else {
		add(c-1, r)
		add(c-1, r+1)
		add(c+1, r)
		add(c+1, r+1)
	}
	return nb, n
}

// --- 更新 ---

func (g *Game) Update() error {
	switch g.scene {
	case "title":
		if isFireJustPressed() {
			g.reset()
			g.scene = "game"
		}
	case "game":
		g.updateEffects()
		g.updateGame()
	case "gameover", "clear":
		g.updateEffects()
		if isFireJustPressed() {
			g.scene = "title"
		}
	}
	return nil
}

func (g *Game) updateGame() {
	// 照準(十字キーは押しっぱなしでゆっくり回り、ロータリーは 1 ノッチ 2° の微調整)
	if koebiten.IsKeyPressed(koebiten.KeyUp) {
		g.angle--
	}
	if koebiten.IsKeyPressed(koebiten.KeyDown) {
		g.angle++
	}
	if koebiten.IsKeyJustPressed(koebiten.KeyRotaryLeft) {
		g.angle -= 2
	}
	if koebiten.IsKeyJustPressed(koebiten.KeyRotaryRight) {
		g.angle += 2
	}
	if g.angle < -angleMax {
		g.angle = -angleMax
	}
	if g.angle > angleMax {
		g.angle = angleMax
	}

	// 発射
	if !g.flying && isFireJustPressed() {
		rad := float64(g.angle) * math.Pi / 180
		g.fx = shooterX
		g.fy = shooterY
		g.fvx = float32(math.Cos(rad)) * speed
		g.fvy = float32(math.Sin(rad)) * speed
		g.ftype = g.cur
		g.flying = true

		// 発射直後に次弾を装填する
		g.cur = g.next
		g.next = int8(rand.N(numTypes))
	}

	// 飛翔(トンネル防止のため 1px 刻みのサブステップで進める)
	if g.flying {
		for i := 0; i < 3; i++ {
			g.fx += g.fvx / 3
			g.fy += g.fvy / 3

			// 上下の壁で反射
			if g.fy < ballR+1 {
				g.fy = 2*(ballR+1) - g.fy
				g.fvy = -g.fvy
			}
			if g.fy > screenH-ballR-1 {
				g.fy = 2*(screenH-ballR-1) - g.fy
				g.fvy = -g.fvy
			}

			if g.hitAt(g.fx, g.fy) {
				g.land()
				break
			}
		}
	}
}

// hitAt は座標 (x, y) が右壁または既存の球に接触しているかを判定する
func (g *Game) hitAt(x, y float32) bool {
	if x >= float32(colX(0)) {
		return true
	}
	for c := 0; c < maxCols; c++ {
		dx := x - float32(colX(c))
		if dx < -7 || 7 < dx {
			continue
		}
		for r := 0; r < rowsInCol(c); r++ {
			if g.grid[c][r] < 0 {
				continue
			}
			dy := y - float32(rowY(c, r))
			if dx*dx+dy*dy < 42 { // (直径-0.5)^2 ≈ 6.5^2
				return true
			}
		}
	}
	return false
}

// land は飛翔中の球を最寄りの空きセルに吸着させる
func (g *Game) land() {
	g.flying = false

	bestC, bestR := -1, -1
	bestD := float32(150) // 約 12px 以内のみ候補
	for c := 0; c < maxCols; c++ {
		for r := 0; r < rowsInCol(c); r++ {
			if g.grid[c][r] >= 0 {
				continue
			}
			if c != 0 && !g.hasOccupiedNeighbor(c, r) {
				continue // 右壁(col 0)か既存の球に接するセルのみ
			}
			dx := g.fx - float32(colX(c))
			dy := g.fy - float32(rowY(c, r))
			d := dx*dx + dy*dy
			if d < bestD {
				bestD = d
				bestC, bestR = c, r
			}
		}
	}

	if bestC < 0 {
		g.scene = "gameover"
		return
	}

	g.grid[bestC][bestR] = g.ftype
	g.popMatches(bestC, bestR)
	g.removeOrphans()

	// 全消しでゲームクリア
	if g.isBoardEmpty() {
		g.scene = "clear"
		return
	}

	// ゲームオーバー判定
	for c := gameOverCol; c < maxCols; c++ {
		for r := 0; r < rowsInCol(c); r++ {
			if g.grid[c][r] >= 0 {
				g.scene = "gameover"
				return
			}
		}
	}
}

func (g *Game) isBoardEmpty() bool {
	for c := 0; c < maxCols; c++ {
		for r := 0; r < rowsInCol(c); r++ {
			if g.grid[c][r] >= 0 {
				return false
			}
		}
	}
	return true
}

func (g *Game) hasOccupiedNeighbor(c, r int) bool {
	nb, n := neighbors(c, r)
	for i := 0; i < n; i++ {
		if g.grid[nb[i][0]][nb[i][1]] >= 0 {
			return true
		}
	}
	return false
}

// popMatches は (c, r) と同種でつながる球が 3 つ以上なら消す
func (g *Game) popMatches(c, r int) {
	t := g.grid[c][r]
	var visited [maxCols][maxRows]bool
	var stack, found [maxCols * maxRows][2]int8
	sp, fp := 0, 0

	visited[c][r] = true
	stack[sp] = [2]int8{int8(c), int8(r)}
	sp++
	for sp > 0 {
		sp--
		cur := stack[sp]
		found[fp] = cur
		fp++
		nb, n := neighbors(int(cur[0]), int(cur[1]))
		for i := 0; i < n; i++ {
			nc, nr := nb[i][0], nb[i][1]
			if !visited[nc][nr] && g.grid[nc][nr] == t {
				visited[nc][nr] = true
				stack[sp] = [2]int8{int8(nc), int8(nr)}
				sp++
			}
		}
	}

	if fp < 3 {
		return
	}
	for i := 0; i < fp; i++ {
		c, r := int(found[i][0]), int(found[i][1])
		g.grid[c][r] = -1
		g.addPop(colX(c), rowY(c, r))
	}
	g.score += fp * 10
}

// removeOrphans は右壁(col 0)につながっていない球を落とす
func (g *Game) removeOrphans() {
	var reach [maxCols][maxRows]bool
	var stack [maxCols * maxRows][2]int8
	sp := 0

	for r := 0; r < rowsInCol(0); r++ {
		if g.grid[0][r] >= 0 {
			reach[0][r] = true
			stack[sp] = [2]int8{0, int8(r)}
			sp++
		}
	}
	for sp > 0 {
		sp--
		cur := stack[sp]
		nb, n := neighbors(int(cur[0]), int(cur[1]))
		for i := 0; i < n; i++ {
			nc, nr := nb[i][0], nb[i][1]
			if !reach[nc][nr] && g.grid[nc][nr] >= 0 {
				reach[nc][nr] = true
				stack[sp] = [2]int8{int8(nc), int8(nr)}
				sp++
			}
		}
	}

	removed := 0
	for c := 0; c < maxCols; c++ {
		for r := 0; r < rowsInCol(c); r++ {
			if g.grid[c][r] >= 0 && !reach[c][r] {
				g.addFall(colX(c), rowY(c, r), g.grid[c][r])
				g.grid[c][r] = -1
				removed++
			}
		}
	}
	g.score += removed * 20
}

// --- エフェクト ---

func (g *Game) addPop(x, y int) {
	if g.nPop >= len(g.pops) {
		return
	}
	g.pops[g.nPop] = popFx{x: int16(x), y: int16(y)}
	g.nPop++
}

func (g *Game) addFall(x, y int, t int8) {
	if g.nFall >= len(g.falls) {
		return
	}
	g.falls[g.nFall] = fallFx{t: t, x: int16(x), y: float32(y)}
	g.nFall++
}

func (g *Game) updateEffects() {
	// マッチ消去: リングを 1 フレームずつ拡大し、終わったら詰める
	for i := 0; i < g.nPop; {
		g.pops[i].frame++
		if g.pops[i].frame > 6 {
			g.pops[i] = g.pops[g.nPop-1]
			g.nPop--
		} else {
			i++
		}
	}
	// 孤立球: 重力で加速しながら落下し、画面外に出たら詰める
	for i := 0; i < g.nFall; {
		g.falls[i].vy += 0.35
		g.falls[i].y += g.falls[i].vy
		if g.falls[i].y > screenH+ballR+1 {
			g.falls[i] = g.falls[g.nFall-1]
			g.nFall--
		} else {
			i++
		}
	}
}

func (g *Game) drawEffects(screen *koebiten.Image) {
	for i := 0; i < g.nPop; i++ {
		p := g.pops[i]
		koebiten.DrawCircle(screen, int(p.x), int(p.y), ballR+int(p.frame), black)
	}
	for i := 0; i < g.nFall; i++ {
		f := g.falls[i]
		drawBall(screen, int(f.x), int(f.y), f.t)
	}
}

func isFireJustPressed() bool {
	keys := koebiten.AppendJustPressedKeys(nil)
	for _, k := range keys {
		switch k {
		case koebiten.KeyUp, koebiten.KeyDown,
			koebiten.KeyRotaryLeft, koebiten.KeyRotaryRight:
			// 照準操作は発射しない
		default:
			return true
		}
	}
	return false
}

// --- 描画 ---

func (g *Game) Draw(screen *koebiten.Image) {
	switch g.scene {
	case "title":
		g.drawTitle(screen)
	case "game":
		g.drawGame(screen)
	case "gameover":
		g.drawBoard(screen)
		g.drawEffects(screen)
		koebiten.Println("Game Over")
		koebiten.Println("Score", g.score)
	case "clear":
		g.drawClear(screen)
		g.drawEffects(screen)
	}
}

func (g *Game) drawClear(screen *koebiten.Image) {
	koebiten.Println("Game Clear!")
	koebiten.Println("Score", g.score)
	for t := 0; t < numTypes; t++ {
		drawBall(screen, 16+t*12, 40, int8(t))
		drawBall(screen, 22+t*12, 50, int8((t+3)%numTypes))
	}
}

func (g *Game) drawTitle(screen *koebiten.Image) {
	koebiten.Println("HEX BOBBLE")
	koebiten.Println("press any key")
	for t := 0; t < numTypes; t++ {
		drawBall(screen, 16+t*12, 40, int8(t))
	}
}

func (g *Game) drawGame(screen *koebiten.Image) {
	g.drawBoard(screen)
	g.drawEffects(screen)

	// 照準線と軌道ガイド、装填中の球
	g.drawAimGuide(screen)
	drawBall(screen, shooterX, shooterY, g.cur)

	// 次の球
	drawBall(screen, shooterX, shooterY+20, g.next)

	// 飛翔中の球
	if g.flying {
		drawBall(screen, int(g.fx), int(g.fy), g.ftype)
	}

	koebiten.Println("Score", g.score)
}

// drawAimGuide は照準線(球から少し離す)と、反射を含む軌道ガイドの点線を描く
func (g *Game) drawAimGuide(screen *koebiten.Image) {
	rad := float64(g.angle) * math.Pi / 180
	dx := float32(math.Cos(rad))
	dy := float32(math.Sin(rad))

	// 方向線: 装填中の球が見えるよう中心から 6px 離して描く
	x1 := shooterX + int(6*dx)
	y1 := shooterY + int(6*dy)
	x2 := shooterX + int(13*dx)
	y2 := shooterY + int(13*dy)
	koebiten.DrawLine(screen, x1, y1, x2, y2, black)

	// 軌道ガイド: 実際の飛翔と同じ反射計算で 2px 刻みに進め、5 点ごとに点を打つ
	px := float32(shooterX)
	py := float32(shooterY)
	vx := dx * 2
	vy := dy * 2
	for i := 1; i <= 200; i++ {
		px += vx
		py += vy
		if py < ballR+1 {
			py = 2*(ballR+1) - py
			vy = -vy
		}
		if py > screenH-ballR-1 {
			py = 2*(screenH-ballR-1) - py
			vy = -vy
		}
		if g.hitAt(px, py) {
			break
		}
		if i >= 8 && i%3 == 0 {
			koebiten.DrawFilledRect(screen, int(px), int(py), 1, 1, black)
		}
	}
}

func (g *Game) drawBoard(screen *koebiten.Image) {
	// 右壁
	koebiten.DrawLine(screen, screenW-1, 0, screenW-1, screenH-1, black)

	for c := 0; c < maxCols; c++ {
		for r := 0; r < rowsInCol(c); r++ {
			if g.grid[c][r] >= 0 {
				drawBall(screen, colX(c), rowY(c, r), g.grid[c][r])
			}
		}
	}
}

// drawBall は球の種類ごとに異なる模様で描く(モノクロのため模様で区別)
func drawBall(screen *koebiten.Image, x, y int, t int8) {
	switch t {
	case 0: // 塗りつぶし
		koebiten.DrawFilledCircle(screen, x, y, ballR, black)
	case 1: // 輪郭のみ
		koebiten.DrawCircle(screen, x, y, ballR, black)
	case 2: // 輪郭+中央ドット
		koebiten.DrawCircle(screen, x, y, ballR, black)
		koebiten.DrawFilledRect(screen, x-1, y-1, 3, 3, black)
	case 3: // 輪郭+横線
		koebiten.DrawCircle(screen, x, y, ballR, black)
		koebiten.DrawLine(screen, x-2, y, x+2, y, black)
	case 4: // 輪郭+縦線
		koebiten.DrawCircle(screen, x, y, ballR, black)
		koebiten.DrawLine(screen, x, y-2, x, y+2, black)
	case 5: // 輪郭+X
		koebiten.DrawCircle(screen, x, y, ballR, black)
		koebiten.DrawLine(screen, x-2, y-2, x+2, y+2, black)
		koebiten.DrawLine(screen, x-2, y+2, x+2, y-2, black)
	}
}

// Layout はスクリーンサイズを返す
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenW, screenH
}
