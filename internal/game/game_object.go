package game

import "github.com/hajimehoshi/ebiten/v2"

type GameObject interface {
	Position() Vector
	SetPosition(position Vector)
	Update()
	Draw(screen *ebiten.Image)
}
