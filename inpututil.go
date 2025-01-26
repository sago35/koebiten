package koebiten

import (
	"sync"
)

type Key int

const (
	KeyMax = 20
)

const (
	Key0 Key = iota
	Key1
	Key2
	Key3
	Key4
	Key5
	Key6
	Key7
	Key8
	Key9
	Key10
	Key11
	KeyRotaryButton
	KeyJoystick
	KeyRotaryLeft
	KeyRotaryRight
	KeyLeft
	KeyRight
	KeyUp
	KeyDown
)

const (
	KeyArrowLeft  = KeyLeft
	KeyArrowRight = KeyRight
	KeyArrowUp    = KeyUp
	KeyArrowDown  = KeyDown
)

type pos struct {
	x int
	y int
}

type inputState struct {
	keyDurations     []int
	prevKeyDurations []int
	state            []bool

	m sync.RWMutex
}

var theInputState = &inputState{
	keyDurations:     make([]int, KeyMax+1),
	prevKeyDurations: make([]int, KeyMax+1),
	state:            make([]bool, KeyMax+1),
}

var isKeyPressed = make([]bool, KeyMax)

func (i *inputState) update() {
	i.m.Lock()
	defer i.m.Unlock()

	// Keyboard
	copy(i.prevKeyDurations, i.keyDurations)
	for k := Key(0); k <= KeyMax; k++ {
		if i.state[k] {
			i.keyDurations[k]++
		} else {
			i.keyDurations[k] = 0
		}
	}
}

// AppendPressedKeys append currently pressed keyboard keys to keys and returns the extended buffer.
// Giving a slice that already has enough capacity works efficiently.
//
// AppendPressedKeys must be called in a game's Update, not Draw.
//
// AppendPressedKeys is concurrent safe.
func AppendPressedKeys(keys []Key) []Key {
	theInputState.m.Lock()
	defer theInputState.m.Unlock()

	for _, k := range keys {
		theInputState.state[k] = true
	}

	for i, d := range theInputState.keyDurations {
		if d == 0 {
			continue
		}
		keys = append(keys, Key(i))
	}
	return keys
}

// PressedKeys returns a set of currently pressed keyboard keys.
//
// PressedKeys must be called in a game's Update, not Draw.
//
// Deprecated: as of v2.2. Use AppendPressedKeys instead.
func PressedKeys() []Key {
	return AppendPressedKeys(nil)
}

// AppendJustPressedKeys append just pressed keyboard keys to keys and returns the extended buffer.
// Giving a slice that already has enough capacity works efficiently.
//
// AppendJustPressedKeys must be called in a game's Update, not Draw.
//
// AppendJustPressedKeys is concurrent safe.
func AppendJustPressedKeys(keys []Key) []Key {
	theInputState.m.Lock()
	defer theInputState.m.Unlock()

	for _, k := range keys {
		theInputState.state[k] = true
	}

	for i, d := range theInputState.keyDurations {
		if d != 1 {
			continue
		}
		keys = append(keys, Key(i))
	}
	return keys
}

// AppendJustReleasedKeys append just released keyboard keys to keys and returns the extended buffer.
// Giving a slice that already has enough capacity works efficiently.
//
// AppendJustReleasedKeys must be called in a game's Update, not Draw.
//
// AppendJustReleasedKeys is concurrent safe.
func AppendJustReleasedKeys(keys []Key) []Key {
	theInputState.m.Lock()
	defer theInputState.m.Unlock()

	for _, k := range keys {
		theInputState.state[k] = false
	}

	for k := Key(0); k <= KeyMax; k++ {
		if theInputState.keyDurations[k] != 0 {
			continue
		}
		if theInputState.prevKeyDurations[k] == 0 {
			continue
		}
		keys = append(keys, k)
	}
	return keys
}

// IsKeyJustPressed returns a boolean value indicating
// whether the given key is pressed just in the current tick.
//
// IsKeyJustPressed must be called in a game's Update, not Draw.
//
// IsKeyJustPressed is concurrent safe.
func IsKeyJustPressed(key Key) bool {
	return KeyPressDuration(key) == 1
}

func IsKeyPressed(key Key) bool {
	return KeyPressDuration(key) > 0
}

// IsKeyJustReleased returns a boolean value indicating
// whether the given key is released just in the current tick.
//
// IsKeyJustReleased must be called in a game's Update, not Draw.
//
// IsKeyJustReleased is concurrent safe.
func IsKeyJustReleased(key Key) bool {
	theInputState.m.RLock()
	r := theInputState.keyDurations[key] == 0 && theInputState.prevKeyDurations[key] > 0
	theInputState.m.RUnlock()
	return r
}

// KeyPressDuration returns how long the key is pressed in ticks (Update).
//
// KeyPressDuration must be called in a game's Update, not Draw.
//
// KeyPressDuration is concurrent safe.
func KeyPressDuration(key Key) int {
	theInputState.m.RLock()
	s := theInputState.keyDurations[key]
	theInputState.m.RUnlock()
	return s
}
