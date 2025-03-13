package snakegame

import (
	"math/rand"
	"time"

	"github.com/sago35/koebiten"
	"tinygo.org/x/drivers/pixel"
)

var (
	white = pixel.NewMonochrome(0xFF, 0xFF, 0xFF)
	black = pixel.NewMonochrome(0x00, 0x00, 0x00)

	gridSize     = 4
	width        = 128 / gridSize
	height       = 64 / gridSize
	initialSpeed = 100 * time.Millisecond
)

type Point struct {
	x, y int
}

type GameState int

const (
	StateOpening GameState = iota
	StatePlaying
	StateGameOver
)

type Game struct {
	snake      []Point
	snakeBuf   [128 / 4 * 64 / 4]Point
	dir        Point
	food       Point
	alive      bool
	speed      time.Duration
	lastMove   time.Time
	pendingDir Point
	score      int
	state      GameState
	waitCnt    int
}

func NewGame() *Game {
	game := &Game{}
	game.Init()
	return game
}

func (g *Game) Init() {
	g.snake = g.snakeBuf[:1]
	g.snake[0] = Point{width / 2, height / 2}
	g.dir = Point{1, 0}
	g.pendingDir = g.dir
	g.spawnFood()
	g.alive = true
	g.speed = initialSpeed
	g.lastMove = time.Now()
	g.score = 0
}

func (g *Game) spawnFood() {
	g.food = Point{rand.Intn(width), rand.Intn(height)}
}

func (g *Game) Update() error {
	if g.state == StateOpening {
		if isAnyKeyJustPressed() {
			g.Init()
			g.state = StatePlaying
			g.waitCnt = 32
		}
		return nil
	}
	if g.state == StateGameOver {
		if g.waitCnt == 0 {
			if isAnyKeyJustPressed() {
				g.Init()
				g.state = StatePlaying
				g.waitCnt = 32
			}
		} else if g.waitCnt > 0 {
			g.waitCnt--
		}
		return nil
	}

	if !g.alive {
		g.state = StateGameOver
		return nil
	}

	if koebiten.IsKeyPressed(koebiten.KeyArrowUp) && g.dir.y == 0 {
		g.pendingDir = Point{0, -1}
	}
	if koebiten.IsKeyPressed(koebiten.KeyArrowDown) && g.dir.y == 0 {
		g.pendingDir = Point{0, 1}
	}
	if koebiten.IsKeyPressed(koebiten.KeyArrowRight) && g.dir.x == 0 {
		g.pendingDir = Point{1, 0}
	}
	if koebiten.IsKeyPressed(koebiten.KeyArrowLeft) && g.dir.x == 0 {
		g.pendingDir = Point{-1, 0}
	}
	if koebiten.IsKeyPressed(koebiten.KeyRotaryRight) {
		if g.dir.x == 1 {
			g.pendingDir = Point{0, 1}
		} else if g.dir.y == 1 {
			g.pendingDir = Point{-1, 0}
		} else if g.dir.x == -1 {
			g.pendingDir = Point{0, -1}
		} else if g.dir.y == -1 {
			g.pendingDir = Point{1, 0}
		}
	}
	if koebiten.IsKeyPressed(koebiten.KeyRotaryLeft) {
		if g.dir.x == 1 {
			g.pendingDir = Point{0, -1}
		} else if g.dir.y == 1 {
			g.pendingDir = Point{1, 0}
		} else if g.dir.x == -1 {
			g.pendingDir = Point{0, 1}
		} else if g.dir.y == -1 {
			g.pendingDir = Point{-1, 0}
		}
	}

	if time.Since(g.lastMove) < g.speed {
		return nil
	}
	g.lastMove = time.Now()
	g.dir = g.pendingDir

	next := Point{(g.snake[0].x + g.dir.x + width) % width, (g.snake[0].y + g.dir.y + height) % height}
	for _, s := range g.snake {
		if s == next {
			g.alive = false
			g.state = StateGameOver
			return nil
		}
	}

	g.snake = g.snakeBuf[:len(g.snake)+1]
	for i := len(g.snake) - 1; i > 0; i-- {
		g.snake[i] = g.snake[i-1]
	}
	g.snake[0] = next
	if next == g.food {
		g.spawnFood()
		g.score = len(g.snake) - 1
		g.speed = time.Duration(float32(g.speed) * 0.95)
	} else {
		g.snake = g.snake[:len(g.snake)-1]
	}

	return nil
}

func (g *Game) Draw(screen *koebiten.Image) {
	if g.state == StateOpening {
		koebiten.Println("Press Button to Start")
		return
	}
	if g.state == StateGameOver {
		koebiten.Println("Game Over")
		koebiten.Println("Score:", g.score)
		if g.waitCnt == 0 {
			koebiten.Println("Press Button to Restart")
		}
		return
	}

	for _, s := range g.snake {
		koebiten.DrawFilledRect(screen, s.x*gridSize, s.y*gridSize, gridSize, gridSize, white)
	}
	koebiten.DrawFilledRect(screen, g.food.x*gridSize, g.food.y*gridSize, gridSize, gridSize, white)
	koebiten.Println("Score:", g.score)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 128, 64
}

func isAnyKeyJustPressed() bool {
	return len(koebiten.AppendJustPressedKeys(nil)) > 0
}
