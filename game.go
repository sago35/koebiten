package koebiten

// Game defines necessary functions for a game.
type Game interface {
	// Update updates a game by one tick. The given argument represents a screen image.
	//
	// Update updates only the game logic and Draw draws the screen.
	//
	// You can assume that Update is always called TPS-times per second (60 by default), and you can assume
	// that the time delta between two Updates is always 1 / TPS [s] (1/60[s] by default). As Ebitengine already
	// adjusts the number of Update calls, you don't have to measure time deltas in Update by e.g. OS timers.
	//
	// An actual TPS is available by ActualTPS(), and the result might slightly differ from your expected TPS,
	// but still, your game logic should stick to the fixed time delta and should not rely on ActualTPS() value.
	// This API is for just measurement and/or debugging. In the long run, the number of Update calls should be
	// adjusted based on the set TPS on average.
	//
	// An actual time delta between two Updates might be bigger than expected. In this case, your game's
	// Update or Draw takes longer than they should. In this case, there is nothing other than optimizing
	// your game implementation.
	//
	// In the first frame, it is ensured that Update is called at least once before Draw. You can use Update
	// to initialize the game state.
	//
	// After the first frame, Update might not be called or might be called once
	// or more for one frame. The frequency is determined by the current TPS (tick-per-second).
	//
	// If the error returned is nil, game execution proceeds normally.
	// If the error returned is Termination, game execution halts, but does not return an error from RunGame.
	// If the error returned is any other non-nil value, game execution halts and the error is returned from RunGame.
	Update() error

	// Draw draws the game screen by one frame.
	//
	// The give argument represents a screen image. The updated content is adopted as the game screen.
	//
	// The frequency of Draw calls depends on the user's environment, especially the monitors refresh rate.
	// For portability, you should not put your game logic in Draw in general.
	Draw(screen *Image)

	// Layout accepts a native outside size in device-independent pixels and returns the game's logical screen
	// size.
	//
	// On desktops, the outside is a window or a monitor (fullscreen mode). On browsers, the outside is a body
	// element. On mobiles, the outside is the view's size.
	//
	// Even though the outside size and the screen size differ, the rendering scale is automatically adjusted to
	// fit with the outside.
	//
	// Layout is called almost every frame.
	//
	// It is ensured that Layout is invoked before Update is called in the first frame.
	//
	// If Layout returns non-positive numbers, the caller can panic.
	//
	// You can return a fixed screen size if you don't care, or you can also return a calculated screen size
	// adjusted with the given outside size.
	//
	// If the game implements the interface LayoutFer, Layout is never called and LayoutF is called instead.
	Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int)
}
