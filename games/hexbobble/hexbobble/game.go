package hexbobble

import (
	"math"
	"math/rand/v2"

	"github.com/sago35/koebiten"
	"tinygo.org/x/drivers/pixel"
	"tinygo.org/x/tinyfont"
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

	numTypes = 5

	colPitch = 7 // 列の x 間隔
	rowPitch = 8 // 行の y 間隔(奇数列は +4 オフセット)
	ballR    = 3 // 描画半径(直径 7)

	shooterX = 8
	shooterY = 32
	angleMax = 72 // 発射角の上限(度)

	speed = 3.0 // 球速(px/フレーム)

	reloadFrames = 8 // 次弾装填アニメーションの長さ(フレーム数)

	cardinalFastFrames = 6 // 上下キーがこのフレーム数以上押され続けたら粗調整(±2)に切り替わる

	shakeThresholdPercent = 15 // せり上がりまでの残り時間がこの割合以下になったら固定球がプルプル震える
	shakeUpdatePeriod     = 3  // プルプルの位相をこのフレーム数ごとにしか更新しない(控えめにするため)

	allClearMessageFrames = 90 // 全消しメッセージを表示し、入力を止めておくフレーム数
)

// せり上がりパラメータ。一定間隔で右から新しい球が 1 列ぶんせり出し、
// 既存の球を左へ押し出す。せり上がるたびに間隔が riseIntervalStep ずつ
// 短くなり(riseIntervalMin が下限)、徐々にテンポが上がっていく。
var (
	riseIntervalInit = 600 // 最初のせり上がり間隔(フレーム数。32ms/フレームなので約 19.2 秒)
	riseIntervalMin  = 180 // せり上がり間隔の下限(フレーム数。約 5.8 秒)
	riseIntervalStep = 10  // せり上がりが起きるたびに間隔を短縮するフレーム数
	riseFillPercent  = 60  // せり出す新しい列の各セルが埋まる確率(%)。100 未満なので列が全部埋まるとは限らない
)

// allClearBonus は盤面を全消しした際に加算されるボーナス得点
var allClearBonus = 200

var (
	white = pixel.NewMonochrome(0xFF, 0xFF, 0xFF)
	black = pixel.NewMonochrome(0x00, 0x00, 0x00)
)

type Game struct {
	grid  [maxCols][maxRows]int8 // 球の種類。-1 は空
	angle int                    // 発射角(度)。0 で真右、負で上向き

	cur, next int8 // 現在装填中の球と次の球

	reloadFrame int  // 装填アニメーションの経過フレーム。0 ならアニメーション中でない
	reloadBall  int8 // アニメーション中に next 位置から cur 位置へ動く球の種類

	flying   bool
	fx, fy   float32
	fvx, fvy float32
	ftype    int8

	riseTimer    int // 次のせり上がりまでのフレーム数
	riseInterval int // 現在のせり上がり間隔(フレーム数。徐々に短くなる)
	pushCount    int // これまでのせり上がり(1 列押し出し)回数。列の偶奇の基準をずらすために使う
	tick         int // プルプル演出のアニメーション位相用フレームカウンタ

	allClearTimer     int // 全消しメッセージを表示中の残りフレーム数。0 なら通常状態
	allClearBonusLeft int // ボーナス得点のうち、まだスコアに加算していない残り

	// マッチ消去エフェクト(拡大するリング)
	pops [maxCols * maxRows]popFx
	nPop int
	// 孤立球の落下エフェクト
	falls [maxCols * maxRows]fallFx
	nFall int

	score     int
	highScore int // reset() では初期化しない(電源を入れ直すまでプレイ間で保持される)
	scene     string

	// BFS 用のワークバッファ。ローカル変数にすると 288 バイトでスタック確保上限
	// (256 バイト)を超え、reachableFromWall/popMatches を呼ぶたびにヒープ確保
	// が発生してしまうため、フィールドとして使い回す(--print-allocs で確認済み)。
	bfsStack           [maxCols * maxRows][2]int8
	popStack, popFound [maxCols * maxRows][2]int8
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
	g.pushCount = 0
	// 初期配置: 右端 4 列を埋める
	for c := 0; c < 4; c++ {
		for r := 0; r < g.rowsInCol(c); r++ {
			g.grid[c][r] = int8(rand.N(numTypes))
		}
	}
	g.angle = 0
	g.cur = int8(rand.N(numTypes))
	g.next = int8(rand.N(numTypes))
	g.reloadFrame = 0
	g.flying = false
	g.nPop = 0
	g.nFall = 0
	g.riseInterval = riseIntervalInit
	g.riseTimer = g.riseInterval
	g.tick = 0
	g.allClearTimer = 0
	g.allClearBonusLeft = 0
	g.score = 0
}

// --- ヘックスグリッド ---
//
// 列 c の行数・y オフセットは (pushCount + c) の偶奇で決まる(pushCount は
// これまでのせり上がり回数)。せり上がりで列の中身を 1 列分シフトすると
// 同時に pushCount が 1 増えるため、あるボールについて「シフト後の列番号 +
// シフト後の pushCount」は常に元の値と偶奇が一致する(差が常に 2 になるた
// め)。つまりボール 1 個 1 個の書式(8 行 or 7 行、オフセット)はせり上がり
// を跨いでも変化しない。列の物理的な位置(生成された瞬間の偶奇)と結び付け
// ず、pushCount 込みで毎回計算し直すことで、1 列ずつの押し出しでも書式の
// 不整合(行が消える・Y 座標が飛ぶ)が起きない。

func (g *Game) rowsInCol(c int) int {
	if (g.pushCount+c)%2 == 0 {
		return maxRows
	}
	return maxRows - 1
}

func colX(c int) int {
	return screenW - 4 - colPitch*c
}

func (g *Game) rowY(c, r int) int {
	if (g.pushCount+c)%2 == 0 {
		return 4 + rowPitch*r
	}
	return 8 + rowPitch*r
}

func (g *Game) validCell(c, r int) bool {
	return 0 <= c && c < maxCols && 0 <= r && r < g.rowsInCol(c)
}

// neighbors は (c, r) に隣接するヘックスセルを返す(最大 6 個)。
// クロージャ経由の追加はゲーム中に毎回ヒープへエスケープしてしまうため、
// 素朴な if の並びで書く(--print-allocs で確認済み)。
func (g *Game) neighbors(c, r int) ([6][2]int, int) {
	var nb [6][2]int
	n := 0
	if g.validCell(c, r-1) {
		nb[n] = [2]int{c, r - 1}
		n++
	}
	if g.validCell(c, r+1) {
		nb[n] = [2]int{c, r + 1}
		n++
	}
	if (g.pushCount+c)%2 == 0 {
		if g.validCell(c-1, r-1) {
			nb[n] = [2]int{c - 1, r - 1}
			n++
		}
		if g.validCell(c-1, r) {
			nb[n] = [2]int{c - 1, r}
			n++
		}
		if g.validCell(c+1, r-1) {
			nb[n] = [2]int{c + 1, r - 1}
			n++
		}
		if g.validCell(c+1, r) {
			nb[n] = [2]int{c + 1, r}
			n++
		}
	} else {
		if g.validCell(c-1, r) {
			nb[n] = [2]int{c - 1, r}
			n++
		}
		if g.validCell(c-1, r+1) {
			nb[n] = [2]int{c - 1, r + 1}
			n++
		}
		if g.validCell(c+1, r) {
			nb[n] = [2]int{c + 1, r}
			n++
		}
		if g.validCell(c+1, r+1) {
			nb[n] = [2]int{c + 1, r + 1}
			n++
		}
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
	case "gameover":
		g.updateEffects()
		// 球が落ち切るまではキー入力を無視する。直後にボタンを押して
		// スコアが見えなくなるのを防ぐため。
		if g.nFall == 0 && isFireJustPressed() {
			g.scene = "title"
		}
	}
	return nil
}

func (g *Game) updateGame() {
	g.tick++

	// 照準: 上下単体は押し始め(cardinalFastFrames フレーム未満)は微調整
	// ±1、それ以上押し続けると粗調整 ±2 に切り替わる(コンコンと短く叩けば
	// 微調整、押しっぱなしにすると速く動く)。左を同時押しした斜め(左斜め
	// 上/左斜め下)は速度が変わらず毎フレーム ±1 の微調整。右(押した瞬間)
	// でセンター(0°)に戻す。ロータリーは 1 ノッチ ±1 の微調整。
	// 全消しメッセージ表示中も照準だけは変更できるようにするため、
	// 発射・せり上がりより先に(早期 return の前に)処理する。
	upDur := koebiten.KeyPressDuration(koebiten.KeyUp)
	downDur := koebiten.KeyPressDuration(koebiten.KeyDown)
	up := upDur > 0
	down := downDur > 0
	left := koebiten.IsKeyPressed(koebiten.KeyLeft)
	switch {
	case up && left:
		g.angle--
	case down && left:
		g.angle++
	case up:
		if upDur >= cardinalFastFrames {
			g.angle -= 2
		} else {
			g.angle--
		}
	case down:
		if downDur >= cardinalFastFrames {
			g.angle += 2
		} else {
			g.angle++
		}
	}
	if koebiten.IsKeyJustPressed(koebiten.KeyRight) {
		g.angle = 0
	}
	if koebiten.IsKeyJustPressed(koebiten.KeyRotaryLeft) {
		g.angle--
	}
	if koebiten.IsKeyJustPressed(koebiten.KeyRotaryRight) {
		g.angle++
	}
	if g.angle < -angleMax {
		g.angle = -angleMax
	}
	if g.angle > angleMax {
		g.angle = angleMax
	}

	// 全消しメッセージの表示中は照準以外の入力・せり上がりを止めておき、
	// 表示が終わったら 1 段詰めて(せり上がらせて)新しい球を供給する。
	// ボーナス得点は一括加算ではなく、残り表示フレーム数で均等に割った分だけ
	// (端数は前倒しで)毎フレーム加算し、最終フレームでちょうど 0 になる
	// ようにする(スコアがカウントアップしていくように見せるため)。
	if g.allClearTimer > 0 {
		add := (g.allClearBonusLeft + g.allClearTimer - 1) / g.allClearTimer
		g.score += add
		g.allClearBonusLeft -= add
		g.updateHighScore()

		g.allClearTimer--
		if g.allClearTimer == 0 {
			g.performRise()
		}
		return
	}

	// 発射してから固定されるまではせり上がりを止める
	if !g.flying {
		g.updateRise()
	}
	if g.scene != "game" {
		return
	}

	// 装填アニメーションの進行。次弾が撃てるようになるより前に必ず終わらせる
	// (着弾が早い場合に備えて、飛翔が終わったら強制的に完了させる)。
	if g.reloadFrame > 0 {
		g.reloadFrame++
		if g.reloadFrame > reloadFrames || !g.flying {
			g.cur = g.reloadBall
			g.reloadFrame = 0
		}
	}

	// 左キー単体(上下との同時押しでない)は一列詰める(せり上がりを手動で
	// 前倒しする)。ゲームテンポを自分で速められる。飛翔中は着弾の判定と
	// 盤面がずれてしまうため無効。
	if !g.flying && !up && !down && koebiten.IsKeyJustPressed(koebiten.KeyLeft) {
		g.performRise()
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

		// 発射直後に次弾の装填アニメーションを開始する(cur は
		// アニメーション完了時に更新される)
		g.reloadBall = g.next
		g.reloadFrame = 1
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

// updateRise はタイマーを消化し、0 になったらせり上がりを実行する
func (g *Game) updateRise() {
	if g.riseTimer > 0 {
		g.riseTimer--
		return
	}
	g.performRise()
}

// performRise はせり上がりを 1 回実行し、次回に向けて間隔を短縮してタイマー
// をリセットする。自動(タイマー満了)・手動(左キー)の両方から呼ばれる。
func (g *Game) performRise() {
	g.doRise()
	if g.scene != "game" {
		return // せり上がりでゲームオーバーになった
	}

	g.riseInterval -= riseIntervalStep
	if g.riseInterval < riseIntervalMin {
		g.riseInterval = riseIntervalMin
	}
	g.riseTimer = g.riseInterval
}

// doRise は右から新しい球を 1 列ぶんせり出させ、既存の球を左へ押し出す。
// 列の中身を 1 つ隣の列インデックスへずらし、pushCount を 1 増やす。
// rowsInCol/rowY が pushCount を織り込んで書式を決めるため、シフトされた
// 側のボールの書式(行数・Y オフセット)は変化しない([[ヘックスグリッド]]
// の説明を参照)。
func (g *Game) doRise() {
	for c := maxCols - 1; c >= 1; c-- {
		g.grid[c] = g.grid[c-1]
	}
	g.pushCount++

	// 列 0 の書式(有効な行数)はせり上がりのたびに偶奇が入れ替わる。
	// 無効な行を -1 のままにしておかないと、2 回後のせり上がりで
	// 書式が元に戻った際に前回以前の古いデータが復活してしまうため、
	// maxRows 全体を必ず塗り直す。
	for r := 0; r < maxRows; r++ {
		if r < g.rowsInCol(0) && rand.N(100) < riseFillPercent {
			g.grid[0][r] = int8(rand.N(numTypes))
		} else {
			g.grid[0][r] = -1
		}
	}

	// ゲームオーバー判定(せり出しで押し出された球が危険域に達した場合)
	for c := gameOverCol; c < maxCols; c++ {
		for r := 0; r < g.rowsInCol(c); r++ {
			if g.grid[c][r] >= 0 {
				g.triggerGameOver()
				return
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
		for r := 0; r < g.rowsInCol(c); r++ {
			if g.grid[c][r] < 0 {
				continue
			}
			dy := y - float32(g.rowY(c, r))
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
		for r := 0; r < g.rowsInCol(c); r++ {
			if g.grid[c][r] >= 0 {
				continue
			}
			if c != 0 && !g.hasOccupiedNeighbor(c, r) {
				continue // 右壁(col 0)か既存の球に接するセルのみ
			}
			dx := g.fx - float32(colX(c))
			dy := g.fy - float32(g.rowY(c, r))
			d := dx*dx + dy*dy
			if d < bestD {
				bestD = d
				bestC, bestR = c, r
			}
		}
	}

	if bestC < 0 {
		g.triggerGameOver()
		return
	}

	g.grid[bestC][bestR] = g.ftype

	// 消去判定はあくまで自分が撃った球によるマッチが起点(せり上がりそのもの
	// では何も消えない)。マッチが成立したら、その時点で壁(col 0)と非連結に
	// なっている球を全て落とす。以前は「このマッチで新たに非連結になった球」
	// だけを対象にしていたが、それだと既にせり上がりで浮いていた塊の内部で
	// マッチが起きても、その塊自体は元々壁と非連結なので永久に浮いたまま残っ
	// てしまう不具合があった。マッチのたびに壁との連結を再判定することで、
	// 浮いた球がプレイを続けても解消されずに残り続ける状況を防ぐ。
	if g.popMatches(bestC, bestR) > 0 {
		reach := g.reachableFromWall()
		for c := 0; c < maxCols; c++ {
			for r := 0; r < g.rowsInCol(c); r++ {
				if g.grid[c][r] >= 0 && !reach[c][r] {
					g.addFall(colX(c), g.rowY(c, r), g.grid[c][r])
					g.grid[c][r] = -1
					g.score += 2
				}
			}
		}
	}

	// 全消しでボーナス得点。スコアには即座に加算せず、メッセージを表示している
	// 間に毎フレーム少しずつ加算していく(updateGame 側の allClearTimer 処理)。
	// 表示が終わったら 1 段詰めて新しい球を供給する。
	if g.isBoardEmpty() {
		g.allClearBonusLeft += allClearBonus
		g.allClearTimer = allClearMessageFrames
	}

	// ゲームオーバー判定
	for c := gameOverCol; c < maxCols; c++ {
		for r := 0; r < g.rowsInCol(c); r++ {
			if g.grid[c][r] >= 0 {
				g.triggerGameOver()
				return
			}
		}
	}
}

// triggerGameOver はゲームオーバーへ遷移し、盤面上の全ての球を落下エフェクト
// に変える。球が落ち切るまでは Update() 側でキー入力を無視するため、直後に
// ボタンを押してもスコアが見えなくなることはない。
func (g *Game) triggerGameOver() {
	g.scene = "gameover"
	for c := 0; c < maxCols; c++ {
		for r := 0; r < g.rowsInCol(c); r++ {
			if g.grid[c][r] >= 0 {
				g.addFall(colX(c), g.rowY(c, r), g.grid[c][r])
				g.grid[c][r] = -1
			}
		}
	}
	g.updateHighScore()
}

func (g *Game) updateHighScore() {
	if g.score > g.highScore {
		g.highScore = g.score
	}
}

func (g *Game) isBoardEmpty() bool {
	for c := 0; c < maxCols; c++ {
		for r := 0; r < g.rowsInCol(c); r++ {
			if g.grid[c][r] >= 0 {
				return false
			}
		}
	}
	return true
}

// reachableFromWall は現在の盤面において右壁(col 0)から連結している
// セルを BFS で求める
func (g *Game) reachableFromWall() [maxCols][maxRows]bool {
	var reach [maxCols][maxRows]bool
	stack := &g.bfsStack
	sp := 0

	for r := 0; r < g.rowsInCol(0); r++ {
		if g.grid[0][r] >= 0 {
			reach[0][r] = true
			stack[sp] = [2]int8{0, int8(r)}
			sp++
		}
	}
	for sp > 0 {
		sp--
		cur := stack[sp]
		nb, n := g.neighbors(int(cur[0]), int(cur[1]))
		for i := 0; i < n; i++ {
			nc, nr := nb[i][0], nb[i][1]
			if !reach[nc][nr] && g.grid[nc][nr] >= 0 {
				reach[nc][nr] = true
				stack[sp] = [2]int8{int8(nc), int8(nr)}
				sp++
			}
		}
	}
	return reach
}

func (g *Game) hasOccupiedNeighbor(c, r int) bool {
	nb, n := g.neighbors(c, r)
	for i := 0; i < n; i++ {
		if g.grid[nb[i][0]][nb[i][1]] >= 0 {
			return true
		}
	}
	return false
}

// popMatches は (c, r) と同種でつながる球が 3 つ以上なら消し、消した数を返す
func (g *Game) popMatches(c, r int) int {
	t := g.grid[c][r]
	var visited [maxCols][maxRows]bool
	stack, found := &g.popStack, &g.popFound
	sp, fp := 0, 0

	visited[c][r] = true
	stack[sp] = [2]int8{int8(c), int8(r)}
	sp++
	for sp > 0 {
		sp--
		cur := stack[sp]
		found[fp] = cur
		fp++
		nb, n := g.neighbors(int(cur[0]), int(cur[1]))
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
		return 0
	}
	for i := 0; i < fp; i++ {
		c, r := int(found[i][0]), int(found[i][1])
		g.grid[c][r] = -1
		g.addPop(colX(c), g.rowY(c, r))
	}
	g.score += fp
	return fp
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

// fireKeysBuf は isFireJustPressed が毎フレーム呼ぶ AppendJustPressedKeys 用の
// 使い回しバッファ。nil を渡すとキーが押された瞬間に毎回ヒープ確保が走るため、
// 外部で確保済みの配列を渡して確保を避ける。
var fireKeysBuf [4]koebiten.Key

func isFireJustPressed() bool {
	keys := koebiten.AppendJustPressedKeys(fireKeysBuf[:0])
	for _, k := range keys {
		switch k {
		case koebiten.KeyUp, koebiten.KeyDown, koebiten.KeyLeft, koebiten.KeyRight,
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
		g.drawEffects(screen)
		koebiten.Println("Game Over")
		koebiten.Println(g.score)
	}
}

func (g *Game) drawTitle(screen *koebiten.Image) {
	koebiten.Println("HEX BOBBLE")
	koebiten.Println("High Score", g.highScore)
	koebiten.Println("press any key")
	for t := 0; t < numTypes; t++ {
		drawBall(screen, 16+t*12, 40, int8(t))
	}
}

func (g *Game) drawGame(screen *koebiten.Image) {
	g.drawBoard(screen)
	g.drawEffects(screen)
	g.drawRiseTimer(screen)

	// 照準線と軌道ガイド、装填中の球
	g.drawAimGuide(screen)
	if g.reloadFrame > 0 {
		// 装填アニメーション: next だった球が下から上へスライドしてくる
		t := float32(g.reloadFrame) / float32(reloadFrames)
		if t > 1 {
			t = 1
		}
		y := shooterY + int(20*(1-t))
		drawBall(screen, shooterX, y, g.reloadBall)
	} else {
		drawBall(screen, shooterX, shooterY, g.cur)
	}

	// 次の球
	drawBall(screen, shooterX, shooterY+20, g.next)

	// 飛翔中の球
	if g.flying {
		drawBall(screen, int(g.fx), int(g.fy), g.ftype)
	}

	if g.allClearTimer > 0 {
		drawCenteredText(screen, "ALL CLEAR!")
	}
	koebiten.Println(g.score)
}

// drawCenteredText は画面中央に文字列を描画する
func drawCenteredText(screen *koebiten.Image, s string) {
	w, _ := tinyfont.LineWidth(&tinyfont.Org01, s)
	x := (screenW - int(w)) / 2
	koebiten.DrawText(screen, s, &tinyfont.Org01, int16(x), screenH/2+2, black)
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

	shake := g.shaking()
	phase := g.tick / shakeUpdatePeriod
	for c := 0; c < maxCols; c++ {
		for r := 0; r < g.rowsInCol(c); r++ {
			if g.grid[c][r] >= 0 {
				x, y := colX(c), g.rowY(c, r)
				if shake {
					x += shakeOffset(phase, c, r)
					y += shakeOffset(phase+11, c, r)
				}
				drawBall(screen, x, y, g.grid[c][r])
			}
		}
	}
}

// shaking はせり上がりまでの残り時間が閾値以下かどうかを返す。true の間、
// 固定された球をプルプル震わせて「もうすぐ詰める」ことを警告する。
func (g *Game) shaking() bool {
	if g.riseInterval <= 0 {
		return false
	}
	return g.riseTimer*100/g.riseInterval <= shakeThresholdPercent
}

// shakeOffset はプルプル演出用の -1/0/1 の疑似乱数オフセットを返す(控えめに
// なるよう半分以上は 0 になる)。math/rand を消費せず、seed(shakeUpdatePeriod
// フレームごとに変化)とセル座標から毎回計算するハッシュなので、球ごとに
// 震え方がずれて見える。
func shakeOffset(seed, c, r int) int {
	h := uint32(seed)*2654435761 + uint32(c)*40503 + uint32(r)*2246822519
	switch h % 4 {
	case 2:
		return -1
	case 3:
		return 1
	default:
		return 0
	}
}

// drawRiseTimer は次のせり上がりまでの残り時間を、画面一番左の 1 ピクセル列
// (x=0)に縦棒で表示する。せり上がり(自動でタイマーが 0 になる、または
// 左キーで手動発生)のたびにタイマーがリセットされ、棒は画面いっぱいに戻る。
func (g *Game) drawRiseTimer(screen *koebiten.Image) {
	if g.riseInterval <= 0 {
		return
	}
	h := g.riseTimer * screenH / g.riseInterval
	if h > 0 {
		koebiten.DrawLine(screen, 0, 0, 0, h-1, black)
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
	case 3: // 輪郭+縦線
		koebiten.DrawCircle(screen, x, y, ballR, black)
		koebiten.DrawLine(screen, x, y-2, x, y+2, black)
	case 4: // 輪郭+X
		koebiten.DrawCircle(screen, x, y, ballR, black)
		koebiten.DrawLine(screen, x-2, y-2, x+2, y+2, black)
		koebiten.DrawLine(screen, x-2, y+2, x+2, y-2, black)
	}
}

// Layout はスクリーンサイズを返す
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenW, screenH
}
