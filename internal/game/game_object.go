package game

import "github.com/hajimehoshi/ebiten/v2"

// GameObject is anything that lives in the world: it stands somewhere, can be
// moved, advances by ticks and draws itself.
type GameObject interface {
	Position() Vector
	SetPosition(position Vector)
	Update()
	Draw(screen *ebiten.Image)
}

var (
	_ GameObject = (*Item)(nil)
	_ GameObject = (*Monster)(nil)
	_ GameObject = (*Player)(nil)
	_ GameObject = (*Projectile)(nil)
)
