//go:build !tinygo

package koebiten

import (
	"github.com/eihigh/miniten"
)

var (
	Run           = miniten.Run
	SetWindowSize = miniten.SetWindowSize
	IsClicked     = miniten.IsClicked
	CursorPos     = miniten.CursorPos
	Println       = miniten.Println
	DrawRect      = miniten.DrawRect
	DrawCircle    = miniten.DrawCircle
	DrawImageFS   = miniten.DrawImageFS
	DrawImage     = miniten.DrawImage
	HitTestRects  = miniten.HitTestRects
)
