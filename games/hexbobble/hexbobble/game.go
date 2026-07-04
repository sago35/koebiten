package hexbobble

import (
	"math"
	"math/rand/v2"

	"github.com/sago35/koebiten"
	"tinygo.org/x/drivers/pixel"
	"tinygo.org/x/tinyfont"
)

// A game like Puzzle Bobble rotated 90 degrees.
// Balls are fired from the cannon on the left and travel right while bouncing
// off the top and bottom walls. When a ball hits the right wall or an existing
// ball, it snaps to an invisible hex grid.
// When 3 or more balls of the same type connect, they pop.

const (
	screenW = 128
	screenH = 64

	maxCols     = 18 // number of columns from the right wall (col 0 is rightmost)
	maxRows     = 8  // number of rows in even columns (odd columns have 7)
	gameOverCol = 15 // game over if a ball is fixed at or past this column

	numTypes = 5

	colPitch = 7 // x spacing between columns
	rowPitch = 8 // y spacing between rows (odd columns are offset by +4)
	ballR    = 3 // draw radius (diameter 7)

	shooterX = 8
	shooterY = 32
	angleMax = 72 // maximum firing angle (degrees)

	speed = 3.0 // ball speed (px/frame)

	reloadFrames = 8 // length of the next-ball reload animation (frames)

	cardinalFastFrames = 6 // holding up/down starts fast repeat (±2/frame) once held this many frames; before that only the initial tap (±1) moves

	shakeThresholdPercent = 15 // fixed balls shake when the time left until the rise drops to this percentage or below
	shakeUpdatePeriod     = 3  // update the shake phase only once every this many frames (to keep it subtle)

	allClearMessageFrames = 90 // frames to show the all-clear message and hold input
)

// Rise parameters. At a fixed interval, a new column of balls pushes in from
// the right and shoves the existing balls to the left. Each rise shortens the
// interval by riseIntervalStep (down to a floor of riseIntervalMin), so the
// tempo gradually speeds up.
var (
	riseIntervalInit = 600 // initial rise interval (frames; at 32ms/frame, about 19.2 seconds)
	riseIntervalMin  = 180 // lower bound of the rise interval (frames; about 5.8 seconds)
	riseIntervalStep = 10  // frames to shorten the interval by on each rise
	riseFillPercent  = 60  // probability (%) that each cell of a new pushed-in column is filled; below 100, so a column is not always full
)

// allClearBonus is the bonus score added when the board is fully cleared.
var allClearBonus = 200

var (
	white = pixel.NewMonochrome(0xFF, 0xFF, 0xFF)
	black = pixel.NewMonochrome(0x00, 0x00, 0x00)
)

type Game struct {
	grid  [maxCols][maxRows]int8 // ball type; -1 is empty
	angle int                    // firing angle (degrees); 0 is straight right, negative is upward

	cur, next int8 // currently loaded ball and the next ball

	reloadFrame int  // elapsed frames of the reload animation; 0 means not animating
	reloadBall  int8 // type of the ball moving from the next position to the cur position during the animation

	flying   bool
	fx, fy   float32
	fvx, fvy float32
	ftype    int8

	riseTimer    int // frames until the next rise
	riseInterval int // current rise interval (frames; gradually shortens)
	pushCount    int // number of rises (single-column pushes) so far; used to shift the column parity baseline
	tick         int // frame counter for the shake animation phase

	allClearTimer     int // frames left while showing the all-clear message; 0 means normal state
	allClearBonusLeft int // remaining bonus score not yet added to the score

	// match-pop effects (expanding rings)
	pops [maxCols * maxRows]popFx
	nPop int
	// falling effects for orphaned balls
	falls [maxCols * maxRows]fallFx
	nFall int

	score     int
	highScore int // not reset by reset() (kept across plays until the power is cycled)
	scene     string

	// Work buffers for BFS. As local variables they are 288 bytes, which exceeds
	// the stack allocation limit (256 bytes) and causes a heap allocation on
	// every call to reachableFromWall/popMatches, so reuse them as fields
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
	// initial layout: fill the rightmost 4 columns
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

// --- hex grid ---
//
// A column c's row count and y offset are determined by the parity of
// (pushCount + c), where pushCount is the number of rises so far. When a rise
// shifts a column's contents by one column, pushCount also increases by 1, so
// for any given ball "shifted column index + shifted pushCount" always has the
// same parity as the original value (the difference is always 2). In other
// words, each individual ball's format (8 rows or 7 rows, and offset) does not
// change across rises. By recomputing everything with pushCount included each
// time, rather than tying it to a column's physical position (the parity at the
// moment it was generated), single-column pushes never cause format
// inconsistencies (rows disappearing or Y coordinates jumping).

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

// neighbors returns the hex cells adjacent to (c, r) (up to 6).
// Adding via a closure escapes to the heap on every call during the game, so it
// is written as a plain sequence of ifs.
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

// --- update ---

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
		// Ignore key input until the balls have finished falling. This prevents
		// pressing a button right away and losing sight of the score.
		if g.nFall == 0 && isFireJustPressed() {
			g.scene = "title"
		}
	}
	return nil
}

func (g *Game) updateGame() {
	g.tick++

	// Aim: a quick tap of up/down alone nudges the angle by exactly ±1 (only on
	// the first frame of the press) for fine adjustment. Keep holding and, after
	// a short dead zone (frames 2 up to cardinalFastFrames), it repeats at a
	// coarse ±2 per frame for fast movement. So a tap fine-tunes and a hold moves
	// quickly, with nothing in between. Diagonals with left held (upper-left/
	// lower-left) keep a fine ±1 per frame. Right (on press) recenters to 0°.
	// The rotary is a fine ±1 per notch.
	// Process this before firing/rise (before the early return) so that aiming
	// alone can still be changed while the all-clear message is showing.
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
		if upDur == 1 {
			g.angle--
		} else if upDur >= cardinalFastFrames {
			g.angle -= 2
		}
	case down:
		if downDur == 1 {
			g.angle++
		} else if downDur >= cardinalFastFrames {
			g.angle += 2
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

	// While the all-clear message is showing, hold all input except aiming and
	// the rise; when it finishes, push in one column (a rise) to supply new
	// balls. The bonus score is not added all at once: each frame adds the
	// remaining bonus divided evenly by the remaining frames (rounding up so the
	// remainder is front-loaded), reaching exactly 0 on the final frame (to make
	// the score appear to count up).
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

	// stop the rise from occurring between firing and the ball being fixed
	if !g.flying {
		g.updateRise()
	}
	if g.scene != "game" {
		return
	}

	// Advance the reload animation. Always finish it before the next ball can be
	// fired (in case landing is quick, force it to complete once the flight ends).
	if g.reloadFrame > 0 {
		g.reloadFrame++
		if g.reloadFrame > reloadFrames || !g.flying {
			g.cur = g.reloadBall
			g.reloadFrame = 0
		}
	}

	// Left alone (not combined with up/down) pushes in one column (manually
	// triggering a rise early). This lets the player speed up the game tempo.
	// Disabled while a ball is in flight, since it would misalign the landing
	// check with the board.
	if !g.flying && !up && !down && koebiten.IsKeyJustPressed(koebiten.KeyLeft) {
		g.performRise()
	}

	// fire
	if !g.flying && isFireJustPressed() {
		rad := float64(g.angle) * math.Pi / 180
		g.fx = shooterX
		g.fy = shooterY
		g.fvx = float32(math.Cos(rad)) * speed
		g.fvy = float32(math.Sin(rad)) * speed
		g.ftype = g.cur
		g.flying = true

		// start the next-ball reload animation right after firing (cur is
		// updated when the animation completes)
		g.reloadBall = g.next
		g.reloadFrame = 1
		g.next = int8(rand.N(numTypes))
	}

	// flight (advance in 1px sub-steps to prevent tunneling)
	if g.flying {
		for i := 0; i < 3; i++ {
			g.fx += g.fvx / 3
			g.fy += g.fvy / 3

			// bounce off the top and bottom walls
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

// updateRise ticks down the timer and performs a rise when it reaches 0.
func (g *Game) updateRise() {
	if g.riseTimer > 0 {
		g.riseTimer--
		return
	}
	g.performRise()
}

// performRise executes one rise, then shortens the interval for next time and
// resets the timer. Called from both the automatic (timer expiry) and manual
// (left key) paths.
func (g *Game) performRise() {
	g.doRise()
	if g.scene != "game" {
		return // the rise caused a game over
	}

	g.riseInterval -= riseIntervalStep
	if g.riseInterval < riseIntervalMin {
		g.riseInterval = riseIntervalMin
	}
	g.riseTimer = g.riseInterval
}

// doRise pushes in a new column of balls from the right and shoves the existing
// balls to the left. It shifts each column's contents to the next column index
// and increments pushCount by 1. Because rowsInCol/rowY factor pushCount into
// the format, the format (row count, Y offset) of the shifted balls does not
// change (see the "hex grid" explanation).
func (g *Game) doRise() {
	for c := maxCols - 1; c >= 1; c-- {
		g.grid[c] = g.grid[c-1]
	}
	g.pushCount++

	// Column 0's format (its valid row count) flips parity on every rise. Unless
	// invalid rows are left as -1, old data from before could resurface when the
	// format flips back two rises later, so always repaint the entire maxRows.
	for r := 0; r < maxRows; r++ {
		if r < g.rowsInCol(0) && rand.N(100) < riseFillPercent {
			g.grid[0][r] = int8(rand.N(numTypes))
		} else {
			g.grid[0][r] = -1
		}
	}

	// Prevent complete floating islands. Before the rise every connected
	// component of the board touches col 0 (an invariant maintained here and by
	// the post-match sweep in land), so after the shift each component contains
	// at least one cell in col 1. If the randomly generated col 0 happens to
	// leave a component disconnected from the wall, add one connecting ball in
	// col 0 next to a col-1 cell of that component. Without this, repeated rises
	// with riseFillPercent < 100 can slowly grow a large fully-floating island
	// that all falls at once when the player pops anything inside it.
	for i := 0; i <= maxRows; i++ {
		reach := g.reachableFromWall()
		fixed := false
		for r := 0; r < g.rowsInCol(1) && !fixed; r++ {
			if g.grid[1][r] < 0 || reach[1][r] {
				continue
			}
			// An unreachable col-1 cell always has an empty col-0 neighbor:
			// if one were occupied it would be a wall anchor, making this
			// cell reachable.
			nb, n := g.neighbors(1, r)
			for j := 0; j < n; j++ {
				nc, nr := nb[j][0], nb[j][1]
				if nc == 0 && g.grid[nc][nr] < 0 {
					g.grid[nc][nr] = int8(rand.N(numTypes))
					fixed = true
					break
				}
			}
		}
		if !fixed {
			break // all components are connected to the wall
		}
	}

	// game over check (if a ball pushed out by the rise reaches the danger zone)
	for c := gameOverCol; c < maxCols; c++ {
		for r := 0; r < g.rowsInCol(c); r++ {
			if g.grid[c][r] >= 0 {
				g.triggerGameOver()
				return
			}
		}
	}
}

// hitAt reports whether the coordinate (x, y) is touching the right wall or an existing ball.
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
			if dx*dx+dy*dy < 42 { // (diameter-0.5)^2 ≈ 6.5^2
				return true
			}
		}
	}
	return false
}

// land snaps the ball in flight to the nearest empty cell.
func (g *Game) land() {
	g.flying = false

	bestC, bestR := -1, -1
	bestD := float32(150) // only cells within about 12px are candidates
	for c := 0; c < maxCols; c++ {
		for r := 0; r < g.rowsInCol(c); r++ {
			if g.grid[c][r] >= 0 {
				continue
			}
			if c != 0 && !g.hasOccupiedNeighbor(c, r) {
				continue // only cells touching the right wall (col 0) or an existing ball
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

	// Pop detection is always triggered by a match from the player's own shot
	// (the rise itself pops nothing). Once a match occurs, drop every ball that
	// is disconnected from the wall (col 0) at that point. Previously only balls
	// "newly disconnected by this match" were targeted, but with that approach a
	// match occurring inside a cluster that was already floating from a rise
	// would leave the cluster itself floating forever, since it was disconnected
	// from the wall to begin with. Re-checking wall connectivity on every match
	// prevents floating balls from lingering unresolved as play continues.
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

	// Bonus score on a full clear. It is not added to the score immediately;
	// instead it is added a little each frame while the message is showing
	// (handled by the allClearTimer logic in updateGame). When the message
	// finishes, push in one column to supply new balls.
	if g.isBoardEmpty() {
		g.allClearBonusLeft += allClearBonus
		g.allClearTimer = allClearMessageFrames
	}

	// game over check
	for c := gameOverCol; c < maxCols; c++ {
		for r := 0; r < g.rowsInCol(c); r++ {
			if g.grid[c][r] >= 0 {
				g.triggerGameOver()
				return
			}
		}
	}
}

// triggerGameOver transitions to game over and turns every ball on the board
// into a falling effect. Update() ignores key input until the balls finish
// falling, so pressing a button right away won't hide the score.
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

// reachableFromWall computes, via BFS, the cells connected to the right wall
// (col 0) in the current board.
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

// popMatches pops the group of same-type balls connected to (c, r) if it has 3
// or more, and returns the number popped.
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

// --- effects ---

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
	// match pop: expand the ring one frame at a time, then compact when done
	for i := 0; i < g.nPop; {
		g.pops[i].frame++
		if g.pops[i].frame > 6 {
			g.pops[i] = g.pops[g.nPop-1]
			g.nPop--
		} else {
			i++
		}
	}
	// orphaned balls: fall while accelerating under gravity, then compact when off-screen
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

// fireKeysBuf is a reusable buffer for the AppendJustPressedKeys call that
// isFireJustPressed makes every frame. Passing nil would heap-allocate every
// time a key is pressed, so pass a pre-allocated array to avoid the allocation.
var fireKeysBuf [4]koebiten.Key

func isFireJustPressed() bool {
	keys := koebiten.AppendJustPressedKeys(fireKeysBuf[:0])
	for _, k := range keys {
		switch k {
		case koebiten.KeyUp, koebiten.KeyDown, koebiten.KeyLeft, koebiten.KeyRight,
			koebiten.KeyRotaryLeft, koebiten.KeyRotaryRight:
			// aiming controls do not fire
		default:
			return true
		}
	}
	return false
}

// --- drawing ---

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

	// aim line, trajectory guide, and the loaded ball
	g.drawAimGuide(screen)
	if g.reloadFrame > 0 {
		// reload animation: the ball that was "next" slides up from below
		t := float32(g.reloadFrame) / float32(reloadFrames)
		if t > 1 {
			t = 1
		}
		y := shooterY + int(20*(1-t))
		drawBall(screen, shooterX, y, g.reloadBall)
	} else {
		drawBall(screen, shooterX, shooterY, g.cur)
	}

	// next ball
	drawBall(screen, shooterX, shooterY+20, g.next)

	// ball in flight
	if g.flying {
		drawBall(screen, int(g.fx), int(g.fy), g.ftype)
	}

	if g.allClearTimer > 0 {
		drawCenteredText(screen, "ALL CLEAR!")
	}
	koebiten.Println(g.score)
}

// drawCenteredText draws a string centered on the screen.
func drawCenteredText(screen *koebiten.Image, s string) {
	w, _ := tinyfont.LineWidth(&tinyfont.Org01, s)
	x := (screenW - int(w)) / 2
	koebiten.DrawText(screen, s, &tinyfont.Org01, int16(x), screenH/2+2, black)
}

// drawAimGuide draws the aim line (offset a little from the ball) and the dotted trajectory guide including bounces.
func (g *Game) drawAimGuide(screen *koebiten.Image) {
	rad := float64(g.angle) * math.Pi / 180
	dx := float32(math.Cos(rad))
	dy := float32(math.Sin(rad))

	// direction line: drawn 6px away from the center so the loaded ball stays visible
	x1 := shooterX + int(6*dx)
	y1 := shooterY + int(6*dy)
	x2 := shooterX + int(13*dx)
	y2 := shooterY + int(13*dy)
	koebiten.DrawLine(screen, x1, y1, x2, y2, black)

	// trajectory guide: advance in 2px steps using the same bounce math as actual flight, plotting a dot every 5 points
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
	// right wall
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

// shaking reports whether the time left until the rise is at or below the
// threshold. While true, fixed balls shake to warn that a rise is imminent.
func (g *Game) shaking() bool {
	if g.riseInterval <= 0 {
		return false
	}
	return g.riseTimer*100/g.riseInterval <= shakeThresholdPercent
}

// shakeOffset returns a pseudo-random -1/0/1 offset for the shake effect (more
// than half the time it is 0, to keep it subtle). It does not consume
// math/rand; it is a hash computed each time from the seed (which changes every
// shakeUpdatePeriod frames) and the cell coordinates, so each ball appears to
// shake out of phase with the others.
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

// drawRiseTimer shows the time left until the next rise as a vertical bar in
// the leftmost pixel column (x=0) of the screen. On every rise (the timer
// automatically reaching 0, or a manual trigger with the left key) the timer
// resets and the bar returns to full height.
func (g *Game) drawRiseTimer(screen *koebiten.Image) {
	if g.riseInterval <= 0 {
		return
	}
	h := g.riseTimer * screenH / g.riseInterval
	if h > 0 {
		koebiten.DrawLine(screen, 0, 0, 0, h-1, black)
	}
}

// drawBall draws each ball type with a different pattern (patterns distinguish types since the display is monochrome).
func drawBall(screen *koebiten.Image, x, y int, t int8) {
	switch t {
	case 0: // filled
		koebiten.DrawFilledCircle(screen, x, y, ballR, black)
	case 1: // outline only
		koebiten.DrawCircle(screen, x, y, ballR, black)
	case 2: // outline + center dot
		koebiten.DrawCircle(screen, x, y, ballR, black)
		koebiten.DrawFilledRect(screen, x-1, y-1, 3, 3, black)
	case 3: // outline + vertical line
		koebiten.DrawCircle(screen, x, y, ballR, black)
		koebiten.DrawLine(screen, x, y-2, x, y+2, black)
	case 4: // outline + X
		koebiten.DrawCircle(screen, x, y, ballR, black)
		koebiten.DrawLine(screen, x-2, y-2, x+2, y+2, black)
		koebiten.DrawLine(screen, x-2, y+2, x+2, y-2, black)
	}
}

// Layout returns the screen size.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenW, screenH
}
