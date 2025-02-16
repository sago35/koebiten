package blocks

import (
	"math/rand"
	"strconv"
	"time"

	"github.com/sago35/koebiten"
	"tinygo.org/x/drivers/pixel"
	"tinygo.org/x/tinyfont"
)

var (
	backgroundColor = pixel.NewMonochrome(0x00, 0x00, 0x00)
	frameColor      = pixel.NewMonochrome(0xFF, 0xFF, 0xFF)
)

const (
	scale        = 1
	screenWidth  = 64 * scale
	screenHeight = 128 * scale
	gridSize     = 5 * scale
)

var (
	dropInterval   = time.Duration(1000 * time.Millisecond)
	lastDropTime   = time.Now()
	moveInterval   = time.Duration(100 * time.Millisecond)
	lastMoveTime   = time.Now()
	lastMoveAction = time.Duration(0)
)

type Game struct {
	board     [12][24]int // Game board
	tetromino Tetromino   // Current block
	score     int         // Score
	highScore int         // High Score
	scene     string
	next      Tetromino
}

type Tetromino struct {
	shapes   [][][]int // Holds 4 rotation states
	rotation int       // Holds the current rotation state
	x, y     int       // Position
}

var (
	// 4 rotation states for T-shaped tetromino
	tShapes = [][][]int{
		{
			{0, 1, 0},
			{1, 1, 1},
			{0, 0, 0},
		},
		{
			{0, 1, 0},
			{0, 1, 1},
			{0, 1, 0},
		},
		{
			{0, 0, 0},
			{1, 1, 1},
			{0, 1, 0},
		},
		{
			{0, 1, 0},
			{1, 1, 0},
			{0, 1, 0},
		},
	}

	// Other tetrominos also have rotation states
	// Examples: I-shaped, O-shaped, L-shaped, J-shaped, S-shaped, Z-shaped, etc.
	iShapes = [][][]int{
		{
			{0, 0, 0, 0},
			{1, 1, 1, 1},
			{0, 0, 0, 0},
			{0, 0, 0, 0},
		},
		{
			{0, 0, 1, 0},
			{0, 0, 1, 0},
			{0, 0, 1, 0},
			{0, 0, 1, 0},
		},
		{
			{0, 0, 0, 0},
			{0, 0, 0, 0},
			{1, 1, 1, 1},
			{0, 0, 0, 0},
		},
		{
			{0, 1, 0, 0},
			{0, 1, 0, 0},
			{0, 1, 0, 0},
			{0, 1, 0, 0},
		},
	}

	oShapes = [][][]int{
		{
			{1, 1},
			{1, 1},
		},
	}

	lShapes = [][][]int{
		{
			{0, 0, 0},
			{1, 1, 1},
			{1, 0, 0},
		},
		{
			{1, 1, 0},
			{0, 1, 0},
			{0, 1, 0},
		},
		{
			{0, 0, 1},
			{1, 1, 1},
			{0, 0, 0},
		},
		{
			{0, 1, 0},
			{0, 1, 0},
			{0, 1, 1},
		},
	}

	jShapes = [][][]int{
		{
			{0, 0, 0},
			{1, 1, 1},
			{0, 0, 1},
		},
		{
			{0, 1, 0},
			{0, 1, 0},
			{1, 1, 0},
		},
		{
			{1, 0, 0},
			{1, 1, 1},
			{0, 0, 0},
		},
		{
			{0, 1, 1},
			{0, 1, 0},
			{0, 1, 0},
		},
	}

	sShapes = [][][]int{
		{
			{0, 0, 0},
			{0, 1, 1},
			{1, 1, 0},
		},
		{
			{1, 0, 0},
			{1, 1, 0},
			{0, 1, 0},
		},
		{
			{0, 1, 1},
			{1, 1, 0},
			{0, 0, 0},
		},
		{
			{0, 1, 0},
			{0, 1, 1},
			{0, 0, 1},
		},
	}

	zShapes = [][][]int{
		{
			{0, 0, 0},
			{1, 1, 0},
			{0, 1, 1},
		},
		{
			{0, 1, 0},
			{1, 1, 0},
			{1, 0, 0},
		},
		{
			{1, 1, 0},
			{0, 1, 1},
			{0, 0, 0},
		},
		{
			{0, 0, 1},
			{0, 1, 1},
			{0, 1, 0},
		},
	}

	// Generate a tetromino randomly
	// Handle other tetrominos similarly
	randShapes = [][][][]int{
		tShapes,
		iShapes,
		oShapes,
		lShapes,
		jShapes,
		sShapes,
		zShapes,
	}
)

// Define T-shaped and other tetrominos including their rotation states in advance
func (g *Game) createNewTetromino() Tetromino {
	next := g.next
	g.next = g.generateNewTetromino()
	return next
}

func (g *Game) generateNewTetromino() Tetromino {
	choice := randShapes[rand.Intn(len(randShapes))]

	// Randomly select a tetromino and return it
	return Tetromino{
		shapes:   choice,
		rotation: 0,
		x:        4,
		y:        0,
	}
}

// Initialization
func NewGame() *Game {
	rand.Seed(time.Now().UnixNano())
	game := &Game{}
	game.next = game.generateNewTetromino()
	game.tetromino = game.createNewTetromino()
	game.scene = "title"
	return game
}

// Game update process
func (g *Game) Update() error {
	if g.scene != "game" {
		return nil
	}

	if time.Since(lastDropTime) > dropInterval {
		if g.isValidPosition(g.tetromino.x, g.tetromino.y+1, g.currentShape()) {
			g.tetromino.y++
		} else {
			// Lock the block and generate a new one
			g.lockTetromino()
			g.tetromino = g.createNewTetromino()
			if !g.isValidPosition(g.tetromino.x, g.tetromino.y, g.currentShape()) {
				g.scene = "gameover"
			}
		}
		lastDropTime = time.Now()
	}

	if time.Since(lastMoveTime) > moveInterval {
		// Move left
		if koebiten.KeyPressDuration(koebiten.KeyUp) > 0 {
			if g.isValidPosition(g.tetromino.x-1, g.tetromino.y, g.currentShape()) {
				g.tetromino.x--
			}
			if koebiten.KeyPressDuration(koebiten.KeyUp) < 10 {
				lastMoveTime = time.Now().Add(moveInterval * 2)
			} else {
				lastMoveTime = time.Now()
			}
		}
		// Move right
		if koebiten.KeyPressDuration(koebiten.KeyDown) > 0 {
			if g.isValidPosition(g.tetromino.x+1, g.tetromino.y, g.currentShape()) {
				g.tetromino.x++
			}
			if koebiten.KeyPressDuration(koebiten.KeyDown) < 10 {
				lastMoveTime = time.Now().Add(moveInterval * 2)
			} else {
				lastMoveTime = time.Now()
			}
		}
		// Move down
		if koebiten.KeyPressDuration(koebiten.KeyLeft) > 0 {
			if g.isValidPosition(g.tetromino.x, g.tetromino.y+1, g.currentShape()) {
				g.tetromino.y++
			} else {
				// Lock the block and generate a new one
				g.lockTetromino()
				g.tetromino = g.createNewTetromino()
				if !g.isValidPosition(g.tetromino.x, g.tetromino.y, g.currentShape()) {
					g.scene = "gameover"
				}
			}
			lastMoveTime = time.Now()
		}
	}

	// Move down faster
	if koebiten.IsKeyJustPressed(koebiten.KeyRight) || koebiten.IsKeyJustPressed(koebiten.Key9) {
		for g.isValidPosition(g.tetromino.x, g.tetromino.y+1, g.currentShape()) {
			g.tetromino.y++
		}
		// Lock the block and generate a new one
		g.lockTetromino()
		g.tetromino = g.createNewTetromino()
		if !g.isValidPosition(g.tetromino.x, g.tetromino.y, g.currentShape()) {
			g.scene = "gameover"
		}
	}

	if koebiten.KeyPressDuration(koebiten.KeyUp) == 0 &&
		koebiten.KeyPressDuration(koebiten.KeyDown) == 0 &&
		koebiten.KeyPressDuration(koebiten.KeyLeft) == 0 &&
		koebiten.KeyPressDuration(koebiten.KeyRight) == 0 {
		lastMoveTime = time.Time{}
	}

	// Rotate the block
	if koebiten.IsKeyJustPressed(koebiten.Key4) || koebiten.IsKeyJustPressed(koebiten.KeyRotaryLeft) || koebiten.IsKeyJustPressed(koebiten.Key1) {
		g.rotateTetromino(true)
	} else if koebiten.IsKeyJustPressed(koebiten.Key8) || koebiten.IsKeyJustPressed(koebiten.KeyRotaryRight) || koebiten.IsKeyJustPressed(koebiten.Key0) {
		g.rotateTetromino(false)
	}
	return nil
}

// Calculate how far the tetromino can fall
func (g *Game) calculateDropPosition() int {
	ghostY := g.tetromino.y
	for g.isValidPosition(g.tetromino.x, ghostY+1, g.currentShape()) {
		ghostY++
	}
	return ghostY
}

func (g *Game) isValidPosition(x, y int, shape [][]int) bool {
	for i, row := range shape {
		for j, cell := range row {
			if cell == 0 {
				continue
			}
			newX := x + j
			newY := y + i
			// Check if it goes out of the board range
			if newX < 0 || newX >= len(g.board) || newY >= len(g.board[0]) {
				return false
			}
			// Check if it collides with an existing block
			if g.board[newX][newY] == 1 {
				return false
			}
		}
	}
	return true
}

func (g *Game) lockTetromino() {
	for i, row := range g.currentShape() {
		for j, cell := range row {
			if cell == 1 {
				x := g.tetromino.x + j
				y := g.tetromino.y + i
				g.board[x][y] = 1
			}
		}
	}
	g.clearLines()
}

func (g *Game) clearLines() {
	cnt := 0
	for y := 0; y < len(g.board[0]); y++ {
		full := true
		for x := 0; x < len(g.board); x++ {
			if g.board[x][y] == 0 {
				full = false
				break
			}
		}
		if full {
			// Clear the line and move the upper blocks down
			for yy := y; yy > 0; yy-- {
				for x := 0; x < len(g.board); x++ {
					g.board[x][yy] = g.board[x][yy-1]
				}
			}
			// Clear the top row
			for x := 0; x < len(g.board); x++ {
				g.board[x][0] = 0
			}
			cnt++
		}
	}

	switch cnt {
	case 1:
		g.score += 1
	case 2:
		g.score += 2
	case 3:
		g.score += 4
	case 4:
		g.score += 10
	}
	if cnt > 0 {
		if (g.score % 10) == 0 {
			dropInterval = dropInterval * 900 / 1000
		}
	}
}

// Tetromino rotation process
func (g *Game) rotateTetromino(reverse bool) {
	// Calculate the next rotation state
	nextRotation := (g.tetromino.rotation + 1) % len(g.tetromino.shapes)
	if reverse {
		nextRotation = (g.tetromino.rotation + len(g.tetromino.shapes) - 1) % len(g.tetromino.shapes)
	}

	// Check if the rotated shape is valid
	if g.isValidPosition(g.tetromino.x, g.tetromino.y, g.tetromino.shapes[nextRotation]) {
		// If valid, apply the rotation
		g.tetromino.rotation = nextRotation
	} else {
		// If the rotated position is invalid, try wall kicks (move left or right and attempt to rotate)
		if g.isValidPosition(g.tetromino.x-1, g.tetromino.y, g.tetromino.shapes[nextRotation]) {
			// Move left and rotate
			g.tetromino.x--
			g.tetromino.rotation = nextRotation
		} else if g.isValidPosition(g.tetromino.x+1, g.tetromino.y, g.tetromino.shapes[nextRotation]) {
			// Move right and rotate
			g.tetromino.x++
			g.tetromino.rotation = nextRotation
		}
	}
}

// Function to get the current shape of the tetromino
func (g *Game) currentShape() [][]int {
	return g.tetromino.shapes[g.tetromino.rotation]
}

func (g *Game) Draw(screen *koebiten.Image) {
	switch g.scene {
	case "title":
		g.drawTitle(screen)
	case "game":
		g.drawGame(screen)
	case "gameover":
		g.drawGameover(screen)
	}
}

func (g *Game) drawTitle(screen *koebiten.Image) {
	koebiten.Println("click")
	koebiten.Println("to start")
	koebiten.Println("")
	koebiten.Println("high score")
	koebiten.Println("  ", g.highScore)
	if isAnyKeyJustPressed() {
		g.score = 0
		dropInterval = time.Duration(1000 * time.Millisecond)
		lastDropTime = time.Now()
		moveInterval = time.Duration(100 * time.Millisecond)
		lastMoveTime = time.Now()
		lastMoveAction = time.Duration(0)
		for i := range g.board {
			for j := range g.board[i] {
				g.board[i][j] = 0
			}
		}
		g.next = g.generateNewTetromino()
		g.tetromino = g.createNewTetromino()
		g.scene = "game"
	}
}

// Game drawing process
func (g *Game) drawGame(screen *koebiten.Image) {
	koebiten.DrawLine(screen, 1, 0, 1, gridSize*24, frameColor)
	koebiten.DrawLine(screen, gridSize*12+2, 0, gridSize*12+2, gridSize*24+1, frameColor)
	koebiten.DrawLine(screen, 1, gridSize*24+1, gridSize*12+2, gridSize*24+1, frameColor)

	// Calculate the falling position of the tetromino
	ghostY := g.calculateDropPosition()

	// Draw the guide
	for y, row := range g.currentShape() {
		for x, cell := range row {
			if cell == 1 {
				koebiten.DrawRect(screen, int((g.tetromino.x+x)*gridSize+1)+2, int((ghostY+y)*gridSize+1), int(gridSize-2), int(gridSize-2), frameColor)
			}
		}
	}

	// Draw the game board
	for y := 0; y < 24; y++ {
		for x := 0; x < 12; x++ {
			if g.board[x][y] == 1 {
				koebiten.DrawFilledRect(screen, int(x*gridSize)+2, int(y*gridSize), gridSize-1, gridSize-1, frameColor)
			}
		}
	}

	// Draw the tetromino
	for y, row := range g.currentShape() {
		for x, cell := range row {
			if cell == 1 {
				koebiten.DrawFilledRect(screen, int((g.tetromino.x+x)*gridSize)+2, int((g.tetromino.y+y)*gridSize), gridSize-1, gridSize-1, frameColor)
			}
		}
	}

	koebiten.DrawText(screen, "Score: "+strconv.Itoa(g.score), &tinyfont.Org01, 0, gridSize*24+6, frameColor)

	// Draw the next tetromino
	xoffset := 0
	if g.tetromino.x < 4 {
		xoffset = 8
	}
	for y, row := range g.next.shapes[0] {
		for x, cell := range row {
			if cell == 1 {
				koebiten.DrawRect(screen, int((x+xoffset)*gridSize+1)+2, int((y)*gridSize+1), int(gridSize-2), int(gridSize-2), frameColor)
			}
		}
	}
}

func (g *Game) drawGameover(screen *koebiten.Image) {
	koebiten.Println("gameover")
	koebiten.Println("score: ", g.score)
	if isAnyKeyJustPressed() {
		if g.highScore < g.score {
			g.highScore = g.score
		}
		g.scene = "title"
	}
}

// Screen size
func (g *Game) Layout(outsideWidth, outsideHeight int) (w, h int) {
	return screenWidth, screenHeight
}

func isAnyKeyJustPressed() bool {
	return len(koebiten.AppendJustPressedKeys(nil)) > 0
}
