package ringflight

import (
	"strconv"

	"github.com/chewxy/math32"
	"github.com/sago35/koebiten"
	"tinygo.org/x/drivers/pixel"
	"tinygo.org/x/tinyfont"
)

var black = pixel.NewMonochrome(0x00, 0x00, 0x00)

const (
	screenW = 128
	screenH = 64
	cx      = 64.0
	cy      = 32.0
	focal   = 64.0 // focal length (px)
	nearZ   = 0.5  // near clip plane

	dt = 0.032 // koebiten ticks every 32ms

	gridStep = 30.0  // ground grid spacing
	gridDist = 180.0 // grid draw distance
	tileSize = 60.0  // building tile size
	tileN    = 3     // tiles of buildings drawn in each direction
)

type Game struct {
	// aircraft state
	x, y, z          float32 // y is altitude
	yaw, pitch, roll float32
	speed            float32

	// next ring to pass through
	ringX, ringY, ringZ float32
	ringYaw             float32 // heading of the ring sequence
	ringN               int     // rings passed

	// state flow: title (demo) -> play -> result
	state     gameState
	ticksLeft int
	overTicks int // ticks on result screen (returns to title when idle)
	frame     uint32

	// demo: detect being stuck on the same ring
	demoPrevN     int
	demoStuckTick int

	// demo: committed lateral avoidance (avoid fully instead of small corrections)
	avoidDir   float32
	avoidTicks int

	// sin/cos of camera rotation (computed once per frame)
	sy, cyaw, sp, cp, sr, cr float32
}

const ringR = 8.0 // ring radius (world units)

const gameTicks = 60 * 1000 / 32    // 60 seconds (32ms tick)
const overLockTicks = 2 * 1000 / 32 // input lockout right after result
const rollMax = 1.0                 // max roll (rad)

// Button groups. A and B accept the rotary encoder and Key2/Key3 as well,
// since Key0/Key1 are awkward to reach on zero-kb02.
var (
	keysA = [...]koebiten.Key{koebiten.Key0, koebiten.Key2, koebiten.KeyRotaryLeft}
	keysB = [...]koebiten.Key{koebiten.Key1, koebiten.Key3, koebiten.KeyRotaryRight}
)

func anyPressed(keys []koebiten.Key) bool {
	for _, k := range keys {
		if koebiten.IsKeyPressed(k) {
			return true
		}
	}
	return false
}

func anyJustPressed(keys []koebiten.Key) bool {
	for _, k := range keys {
		if koebiten.IsKeyJustPressed(k) {
			return true
		}
	}
	return false
}

type gameState uint8

const (
	stateTitle gameState = iota // title: autopilot demo runs in the background
	statePlay
	stateOver
)

func NewGame() *Game {
	return &Game{
		y:         18,
		speed:     30,
		ringX:     0,
		ringY:     20,
		ringZ:     200,
		ticksLeft: gameTicks,
	}
}

func (g *Game) Update() error {
	g.frame++
	switch g.state {
	case stateTitle:
		g.autopilot()
		g.advance()
		if anyJustPressed(keysA[:]) || anyJustPressed(keysB[:]) {
			*g = *NewGame()
			g.state = statePlay
		}
		g.cacheTrig()
		return nil
	case stateOver:
		g.overTicks++
		// ignore keys for 2 seconds to prevent accidental presses
		if g.overTicks > overLockTicks && anyJustPressed(keysA[:]) {
			*g = *NewGame()
			g.state = statePlay
		} else if g.overTicks > overLockTicks && anyJustPressed(keysB[:]) {
			*g = *NewGame() // B returns to title
		} else if g.overTicks > 10*1000/32 { // return to title after 10 seconds idle
			*g = *NewGame()
		}
		g.cacheTrig()
		return nil
	}

	g.ticksLeft--
	if g.ticksLeft <= 0 {
		g.state = stateOver
	}

	// roll: left/right keys
	if koebiten.IsKeyPressed(koebiten.KeyArrowLeft) {
		g.roll -= 1.6 * dt
	} else if koebiten.IsKeyPressed(koebiten.KeyArrowRight) {
		g.roll += 1.6 * dt
	} else {
		g.roll *= 0.95 // level out when no input
	}
	g.roll = clamp(g.roll, -rollMax, rollMax)

	// elevator: up/down keys (down = pull up), applied in body frame:
	// while banked, pull splits into pitch by cos(roll) and turn by sin(roll)
	var ele float32
	if koebiten.IsKeyPressed(koebiten.KeyArrowDown) {
		ele = 1
	} else if koebiten.IsKeyPressed(koebiten.KeyArrowUp) {
		ele = -1
	}
	if ele != 0 {
		// visual bank is capped at +/-1.0rad (57 deg), but control mixing treats full
		// deflection as 90 deg, so full roll + pull up is a pure level turn
		sr, cr := math32.Sincos(g.roll * (math32.Pi / 2) / rollMax)
		g.pitch += ele * cr * 1.0 * dt
		g.yaw += ele * sr * 1.1 * dt
	} else {
		g.pitch *= 0.98
	}
	g.pitch = clamp(g.pitch, -0.7, 0.7)

	// throttle: A/B
	if anyPressed(keysA[:]) {
		g.speed += 15 * dt
	}
	if anyPressed(keysB[:]) {
		g.speed -= 15 * dt
	}
	g.speed = clamp(g.speed, 12, 60)

	// gentle auto-turn from bank (main turning is bank + pull up)
	g.yaw += g.roll * 0.6 * dt

	g.advance()
	g.cacheTrig()
	return nil
}

// move forward and check ring pass (shared by play and demo)
func (g *Game) advance() {
	cp := math32.Cos(g.pitch)
	g.x += math32.Sin(g.yaw) * cp * g.speed * dt
	g.z += math32.Cos(g.yaw) * cp * g.speed * dt
	g.y += math32.Sin(g.pitch) * g.speed * dt
	if g.y < 3 {
		g.y = 3
		if g.pitch < 0 {
			g.pitch = 0
		}
	}

	dx := g.ringX - g.x
	dy := g.ringY - g.y
	dz := g.ringZ - g.z
	if dx*dx+dy*dy+dz*dz < (ringR+2)*(ringR+2) {
		g.ringN++
		g.spawnRing()
	}
}

func (g *Game) cacheTrig() {
	g.sy, g.cyaw = math32.Sincos(g.yaw)
	g.sp, g.cp = math32.Sincos(g.pitch)
	g.sr, g.cr = math32.Sincos(g.roll)
}

// demo autopilot: track the ring center
func (g *Game) autopilot() {
	// skip the ring if not passed within 20 seconds (keeps the demo from looking stuck)
	if g.ringN == g.demoPrevN {
		g.demoStuckTick++
		if g.demoStuckTick > 20*1000/32 {
			g.spawnRing()
			g.demoStuckTick = 0
		}
	} else {
		g.demoPrevN = g.ringN
		g.demoStuckTick = 0
	}

	dx := g.ringX - g.x
	dy := g.ringY - g.y
	dz := g.ringZ - g.z
	horiz := math32.Sqrt(dx*dx + dz*dz)

	yawErr := wrapAngle(math32.Atan2(dx, dz) - g.yaw)
	if horiz < 40 && math32.Abs(yawErr) > 0.8 {
		// ring is inside the turn radius and behind: keep turning would circle forever,
		// so fly straight to gain distance before turning back
		yawErr = 0
	}

	// limit dive angle in demo so building avoidance has time to react
	pitchTgt := clamp(math32.Atan2(dy, horiz), -0.45, 0.7)
	pitchTgt, yawBias, emergency := g.avoidBuildings(pitchTgt, horiz)

	// lateral avoidance: always use full deflection while detected (no small corrections).
	// hold full deflection ~0.5s after detection ends to smooth the reversal.
	// reverse immediately if an obstacle appears on the other side
	const avoidHold = 16
	if yawBias != 0 {
		if yawBias > 0 {
			g.avoidDir = 1
		} else {
			g.avoidDir = -1
		}
		g.avoidTicks = avoidHold
	}

	yawCmd := clamp(yawErr*2, -1.2, 1.2)
	if g.avoidTicks > 0 {
		g.avoidTicks--
		// while avoiding, weaken ring tracking and steer hard away
		yawCmd = clamp(clamp(yawErr*2, -0.6, 0.6)+g.avoidDir*1.2, -1.4, 1.4)
	}
	g.yaw += yawCmd * dt

	rate := float32(1.0)
	if emergency {
		rate = 2.2 // pull up hard for a building right ahead
	}
	g.pitch += clamp((pitchTgt-g.pitch)*3, -rate, rate) * dt

	// visual bank (lean into the actual turn direction)
	rollTgt := clamp(yawCmd*0.9, -1, 1)
	g.roll += clamp(rollTgt-g.roll, -1.6*dt, 1.6*dt)
}

// If a look-ahead point hits a building, raise the pitch target to clear it.
// Buildings beyond the ring are ignored (the ring is passed first).
// Also probes +/-0.2rad to the sides at close range to catch turn arc drift.
func (g *Game) avoidBuildings(pitchTgt, ringDist float32) (float32, float32, bool) {
	yawBias := float32(0)
	emergency := false
	sp := math32.Sin(g.pitch)
	sy0, cy0 := math32.Sincos(g.yaw)
	maxD := ringDist + 12
	if maxD > 96 {
		maxD = 96
	}
	// near the ring, disable normal avoidance and rely on tight emergency avoidance,
	// so the final descent to a ring beside a building is not blocked
	if ringDist < 50 {
		maxD = 0
	}
	for _, dyaw := range [...]float32{-0.2, 0, 0.2} {
		sy, cy := math32.Sincos(g.yaw + dyaw)
		lim := maxD
		if dyaw != 0 && lim > 48 {
			lim = 48 // lateral arc drift only matters at close range
		}
		for d := float32(12); d <= lim; d += 12 {
			px := g.x + sy*d
			pz := g.z + cy*d
			py := g.y + sp*d
			ix := int32(math32.Floor(px / tileSize))
			iz := int32(math32.Floor(pz / tileSize))
			ok, bh, bw := buildingAt(ix, iz)
			if !ok {
				continue
			}
			bx := float32(ix)*tileSize + tileSize/2
			bz := float32(iz)*tileSize + tileSize/2
			if math32.Abs(px-bx) < bw+6 && math32.Abs(pz-bz) < bw+6 && py < bh+6 {
				// climb cost: extra pitch needed to clear
				climbCost := math32.Atan2(bh+10-g.y, d) - pitchTgt
				// steer cost: heading change to clear the building footprint
				lat := (bx-g.x)*cy0 - (bz-g.z)*sy0 // lateral offset to the right of heading
				miss := bw + 10 - math32.Abs(lat)
				steerCost := math32.Atan2(miss, d)
				if steerCost < climbCost {
					// steering is cheaper: bias heading away from the building
					dir := float32(-1) // building on the right -> go left
					if lat < 0 {
						dir = 1
					}
					b := dir * clamp(steerCost*2, 0.3, 1.0)
					if math32.Abs(b) > math32.Abs(yawBias) {
						yawBias = b
					}
				} else if need := math32.Atan2(bh+10-g.y, d); need > pitchTgt {
					pitchTgt = need
				}
			}
		}
	}
	// emergency avoidance: check within 40 ahead with a tight (size+2) test regardless of ring distance.
	// rings are never spawned beside buildings, so a valid descent never triggers this
	{
		sy, cy := math32.Sincos(g.yaw)
		for d := float32(8); d <= 40; d += 8 {
			px := g.x + sy*d
			pz := g.z + cy*d
			py := g.y + sp*d
			ix := int32(math32.Floor(px / tileSize))
			iz := int32(math32.Floor(pz / tileSize))
			ok, bh, bw := buildingAt(ix, iz)
			if !ok {
				continue
			}
			bx := float32(ix)*tileSize + tileSize/2
			bz := float32(iz)*tileSize + tileSize/2
			if math32.Abs(px-bx) < bw+2 && math32.Abs(pz-bz) < bw+2 && py < bh+2 {
				emergency = true
				need := math32.Atan2(bh+8-g.y, d)
				if need > pitchTgt {
					pitchTgt = need
				}
			}
		}
	}
	return clamp(pitchTgt, -0.7, 0.7), yawBias, emergency
}

func wrapAngle(a float32) float32 {
	for a > math32.Pi {
		a -= 2 * math32.Pi
	}
	for a < -math32.Pi {
		a += 2 * math32.Pi
	}
	return a
}

func (g *Game) Draw(screen *koebiten.Image) {
	g.drawHorizon(screen)
	g.drawGrid(screen)
	g.drawBuildings(screen)
	g.drawRing(screen)
	g.drawPlane(screen)
	g.drawHUD(screen)
}

// spawn the next ring in a pseudo-random direction
func (g *Game) spawnRing() {
	h := hash2(int32(g.ringN), 9973)
	g.ringYaw += (float32(h%201) - 100) / 100 * 0.6 // turn by up to +/-0.6rad
	dist := 150 + float32((h>>8)%100)
	g.ringX += math32.Sin(g.ringYaw) * dist
	g.ringZ += math32.Cos(g.ringYaw) * dist
	g.ringY = 12 + float32((h>>16)%45)

	// if the ring is inside or too close to a building, move it above the building.
	// check the surrounding 3x3 tiles since neighbors matter near tile edges
	ix0 := int32(math32.Floor(g.ringX / tileSize))
	iz0 := int32(math32.Floor(g.ringZ / tileSize))
	for iz := iz0 - 1; iz <= iz0+1; iz++ {
		for ix := ix0 - 1; ix <= ix0+1; ix++ {
			ok, bh, bw := buildingAt(ix, iz)
			if !ok {
				continue
			}
			bx := float32(ix)*tileSize + tileSize/2
			bz := float32(iz)*tileSize + tileSize/2
			if math32.Abs(g.ringX-bx) < bw+ringR+6 && math32.Abs(g.ringZ-bz) < bw+ringR+6 && g.ringY < bh+ringR+4 {
				g.ringY = bh + ringR + 6
			}
		}
	}
}

// ring (billboard circle) and direction indicator
func (g *Game) drawRing(screen *koebiten.Image) {
	x, y, z := g.toCamera(g.ringX, g.ringY, g.ringZ)

	onScreen := false
	if z > nearZ {
		sx := cx + focal*x/z
		sy := cy - focal*y/z
		r := focal * ringR / z
		if r < 1 {
			r = 1
		}
		if r > 90 {
			r = 90
		}
		if sx > -r && sx < screenW+r && sy > -r && sy < screenH+r {
			onScreen = sx >= 0 && sx < screenW && sy >= 0 && sy < screenH
			koebiten.DrawCircle(screen, int(sx), int(sy), int(r), black)
			if r > 6 { // double circle for thickness when close
				koebiten.DrawCircle(screen, int(sx), int(sy), int(r)-2, black)
			}
			if onScreen { // center marker (+)
				koebiten.DrawLine(screen, int(sx)-2, int(sy), int(sx)+2, int(sy), black)
				koebiten.DrawLine(screen, int(sx), int(sy)-2, int(sx), int(sy)+2, black)
			}
		}
	}
	if !onScreen {
		g.drawArrow(screen, x, y)
	}
}

// draw an arrow toward the ring at radius 26 from screen center
func (g *Game) drawArrow(screen *koebiten.Image, camX, camY float32) {
	dx := camX
	dy := -camY // screen y points down
	n := math32.Sqrt(dx*dx + dy*dy)
	if n < 0.001 {
		dx, dy, n = 0, -1, 1
	}
	dx /= n
	dy /= n
	tipX := cx + dx*26
	tipY := cy + dy*26
	// shaft
	koebiten.DrawLine(screen, int(tipX-dx*7), int(tipY-dy*7), int(tipX), int(tipY), black)
	// arrowhead (backward +/- perpendicular)
	px, py := -dy, dx
	koebiten.DrawLine(screen, int(tipX), int(tipY), int(tipX-dx*4+px*3), int(tipY-dy*4+py*3), black)
	koebiten.DrawLine(screen, int(tipX), int(tipY), int(tipX-dx*4-px*3), int(tipY-dy*4-py*3), black)
}

func (g *Game) Layout(w, h int) (int, int) { return screenW, screenH }

// ---- 3D transform ----

// world -> camera coordinates
func (g *Game) toCamera(wx, wy, wz float32) (x, y, z float32) {
	dx := wx - g.x
	dy := wy - g.y
	dz := wz - g.z
	// yaw (Y axis)
	x1 := g.cyaw*dx - g.sy*dz
	z1 := g.sy*dx + g.cyaw*dz
	// pitch (X axis)
	y2 := g.cp*dy - g.sp*z1
	z2 := g.sp*dy + g.cp*z1
	// roll (Z axis)
	x3 := g.cr*x1 - g.sr*y2
	y3 := g.sr*x1 + g.cr*y2
	return x3, y3, z2
}

// near-clip and draw a line in camera coordinates
func (g *Game) line3D(screen *koebiten.Image, x1, y1, z1, x2, y2, z2 float32) {
	if z1 < nearZ && z2 < nearZ {
		return
	}
	if z1 < nearZ {
		t := (nearZ - z1) / (z2 - z1)
		x1 += (x2 - x1) * t
		y1 += (y2 - y1) * t
		z1 = nearZ
	} else if z2 < nearZ {
		t := (nearZ - z2) / (z1 - z2)
		x2 += (x1 - x2) * t
		y2 += (y1 - y2) * t
		z2 = nearZ
	}
	sx1 := cx + focal*x1/z1
	sy1 := cy - focal*y1/z1
	sx2 := cx + focal*x2/z2
	sy2 := cy - focal*y2/z2
	drawClippedLine(screen, sx1, sy1, sx2, sy2)
}

func (g *Game) worldLine(screen *koebiten.Image, ax, ay, az, bx, by, bz float32) {
	x1, y1, z1 := g.toCamera(ax, ay, az)
	x2, y2, z2 := g.toCamera(bx, by, bz)
	g.line3D(screen, x1, y1, z1, x2, y2, z2)
}

// ---- scene ----

func (g *Game) drawHorizon(screen *koebiten.Image) {
	// connect eye-level points at distance 1000, +/-1.2rad from the view direction
	const d = 1000.0
	a := g.yaw - 1.2
	b := g.yaw + 1.2
	g.worldLine(screen,
		g.x+math32.Sin(a)*d, g.y, g.z+math32.Cos(a)*d,
		g.x+math32.Sin(b)*d, g.y, g.z+math32.Cos(b)*d)
}

func (g *Game) drawGrid(screen *koebiten.Image) {
	x0 := math32.Floor(g.x/gridStep) * gridStep
	z0 := math32.Floor(g.z/gridStep) * gridStep
	const crossDist = 100.0       // keep cross lines short; they collapse near the horizon
	n := int(gridDist / gridStep) // lines per side
	for i := -n; i <= n; i++ {
		fi := float32(i) * gridStep
		// lines along Z
		g.worldLine(screen, x0+fi, 0, g.z-gridDist, x0+fi, 0, g.z+gridDist)
		// lines along X (near only)
		if fi > -crossDist && fi < crossDist {
			g.worldLine(screen, g.x-crossDist, 0, z0+fi, g.x+crossDist, 0, z0+fi)
		}
	}
}

func (g *Game) drawBuildings(screen *koebiten.Image) {
	ix0 := int32(math32.Floor(g.x / tileSize))
	iz0 := int32(math32.Floor(g.z / tileSize))
	for iz := iz0 - tileN; iz <= iz0+tileN; iz++ {
		for ix := ix0 - tileN; ix <= ix0+tileN; ix++ {
			ok, bh, bw := buildingAt(ix, iz)
			if !ok {
				continue
			}
			bx := float32(ix)*tileSize + tileSize/2
			bz := float32(iz)*tileSize + tileSize/2
			g.drawBox(screen, bx, bz, bw, bh)
		}
	}
}

// box standing on the ground (center bx,bz / half width w / height h)
func (g *Game) drawBox(screen *koebiten.Image, bx, bz, w, h float32) {
	var px, py, pz [8]float32
	i := 0
	for k := 0; k < 2; k++ { // bottom, top
		y := float32(0)
		if k == 1 {
			y = h
		}
		for _, o := range [4][2]float32{{-w, -w}, {w, -w}, {w, w}, {-w, w}} {
			px[i], py[i], pz[i] = g.toCamera(bx+o[0], y, bz+o[1])
			i++
		}
	}
	// 4 top edges + 4 vertical edges (bottom edges blend into the grid, skipped for speed)
	for e := 0; e < 4; e++ {
		n := (e + 1) % 4
		g.line3D(screen, px[4+e], py[4+e], pz[4+e], px[4+n], py[4+n], pz[4+n]) // top
		g.line3D(screen, px[e], py[e], pz[e], px[4+e], py[4+e], pz[4+e])       // vertical
	}
}

// player aircraft (fixed at bottom of screen, tilts with roll)
func (g *Game) drawPlane(screen *koebiten.Image) {
	const ox, oy = 64.0, 52.0
	c, s := g.cr, g.sr
	rot := func(x, y float32) (float32, float32) {
		return ox + x*c - y*s, oy + x*s + y*c
	}
	pts := [...][4]float32{
		{-16, 0, 16, 0}, // wing
		{-16, 0, -16, -3},
		{16, 0, 16, -3}, // wingtip
		{0, -4, 0, 4},   // fuselage
		{0, 4, -6, 6},
		{0, 4, 6, 6}, // tail
	}
	for _, p := range pts {
		x1, y1 := rot(p[0], p[1])
		x2, y2 := rot(p[2], p[3])
		drawClippedLine(screen, x1, y1, x2, y2)
	}
}

var white = pixel.NewMonochrome(0xFF, 0xFF, 0xFF)

func (g *Game) drawHUD(screen *koebiten.Image) {
	if g.state == stateTitle {
		g.drawDemoCaption(screen)
		return
	}
	g.drawInstruments(screen)
	koebiten.DrawText(screen, "P"+strconv.Itoa(g.ringN), &tinyfont.Org01, 2, 62, black)

	switch g.state {
	case stateOver:
		koebiten.DrawFilledRect(screen, 16, 18, 96, 28, white)
		koebiten.DrawRect(screen, 16, 18, 96, 28, black)
		koebiten.DrawText(screen, "TIME UP", &tinyfont.Org01, 48, 26, black)
		koebiten.DrawText(screen, "PASS "+strconv.Itoa(g.ringN), &tinyfont.Org01, 48, 34, black)
		if g.overTicks > overLockTicks {
			koebiten.DrawText(screen, "A:RETRY B:TITLE", &tinyfont.Org01, 22, 42, black)
		}
	}
}

// top row: altitude, time left, ring distance, speed
func (g *Game) drawInstruments(screen *koebiten.Image) {
	dx := g.ringX - g.x
	dy := g.ringY - g.y
	dz := g.ringZ - g.z
	dist := int(math32.Sqrt(dx*dx + dy*dy + dz*dz))

	sec := (g.ticksLeft*32 + 999) / 1000
	if sec < 0 {
		sec = 0
	}

	koebiten.DrawText(screen, "A"+strconv.Itoa(int(g.y)), &tinyfont.Org01, 2, 6, black)
	// blink the timer in the last 10 seconds
	if sec > 10 || g.ticksLeft%16 < 10 {
		koebiten.DrawText(screen, "T"+strconv.Itoa(sec), &tinyfont.Org01, 40, 6, black)
	}
	koebiten.DrawText(screen, "R"+strconv.Itoa(dist), &tinyfont.Org01, 66, 6, black)
	koebiten.DrawText(screen, "S"+strconv.Itoa(int(g.speed)), &tinyfont.Org01, 104, 6, black)
}

// Title demo captions, shown in sequence like a tutorial video.
type demoScene struct {
	secs   int
	l1, l2 string
	hud    bool // show the in-game instruments above the caption
}

var demoScenes = [...]demoScene{
	{4, "RING FLIGHT", "", false},
	{4, "FLY THROUGH", "THE RINGS", false},
	{4, "LEFT/RIGHT", "ROLL", false},
	{4, "DOWN: PULL UP", "UP: NOSE DOWN", false},
	{4, "A/B", "THROTTLE", false},
	{8, "A ALT   T TIME", "R RING  S SPEED", true},
	{4, "60 SECONDS", "PASS MANY RINGS", false},
	{20, "", "", false},
}

func (g *Game) drawDemoCaption(screen *koebiten.Image) {
	total := 0
	for _, sc := range demoScenes {
		total += sc.secs
	}
	t := int(g.frame%uint32(total*1000/32)) * 32 / 1000
	var sc demoScene
	for _, sc = range demoScenes {
		if t < sc.secs {
			break
		}
		t -= sc.secs
	}
	top := 0
	if sc.hud {
		g.drawInstruments(screen)
		top = 9
	}
	lines := 0
	if sc.l1 != "" {
		lines++
	}
	if sc.l2 != "" {
		lines++
	}
	if lines == 0 {
		return
	}
	h := 3 + 8*lines
	koebiten.DrawFilledRect(screen, 0, top, 128, h, white)
	koebiten.DrawLine(screen, 0, top+h, 127, top+h, black)
	drawCentered(screen, sc.l1, int16(top+7))
	drawCentered(screen, sc.l2, int16(top+15))
}

func drawCentered(screen *koebiten.Image, str string, y int16) {
	_, w := tinyfont.LineWidth(&tinyfont.Org01, str)
	koebiten.DrawText(screen, str, &tinyfont.Org01, int16(screenW-int(w))/2, y, black)
}

// ---- utilities ----

// Clip to the screen rect with Cohen-Sutherland before drawing,
// so projected lines thousands of px long are not passed to Bresenham as is.
func drawClippedLine(screen *koebiten.Image, x1, y1, x2, y2 float32) {
	const xmin, ymin, xmax, ymax = 0.0, 0.0, 127.0, 63.0
	code := func(x, y float32) int {
		c := 0
		if x < xmin {
			c |= 1
		} else if x > xmax {
			c |= 2
		}
		if y < ymin {
			c |= 4
		} else if y > ymax {
			c |= 8
		}
		return c
	}
	c1, c2 := code(x1, y1), code(x2, y2)
	for {
		if c1|c2 == 0 {
			koebiten.DrawLine(screen, int(x1), int(y1), int(x2), int(y2), black)
			return
		}
		if c1&c2 != 0 {
			return
		}
		c := c1
		if c == 0 {
			c = c2
		}
		var x, y float32
		switch {
		case c&8 != 0:
			x = x1 + (x2-x1)*(ymax-y1)/(y2-y1)
			y = ymax
		case c&4 != 0:
			x = x1 + (x2-x1)*(ymin-y1)/(y2-y1)
			y = ymin
		case c&2 != 0:
			y = y1 + (y2-y1)*(xmax-x1)/(x2-x1)
			x = xmax
		default:
			y = y1 + (y2-y1)*(xmin-x1)/(x2-x1)
			x = xmin
		}
		if c == c1 {
			x1, y1 = x, y
			c1 = code(x1, y1)
		} else {
			x2, y2 = x, y
			c2 = code(x2, y2)
		}
	}
}

func clamp(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// building presence, height and half width for tile (ix, iz) (40% of tiles have one)
func buildingAt(ix, iz int32) (ok bool, bh, bw float32) {
	h := hash2(ix, iz)
	if h%5 >= 2 {
		return false, 0, 0
	}
	return true, 12 + float32((h>>4)%28), 7 + float32((h>>9)%7)
}

func hash2(x, y int32) uint32 {
	h := uint32(x)*374761393 + uint32(y)*668265263
	h ^= h >> 13
	h *= 1274126177
	h ^= h >> 16
	return h
}
