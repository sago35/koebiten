package koebiten

type Hardware interface {
	Init() error
	GetDisplay() Displayer
	KeyUpdate() error
}
