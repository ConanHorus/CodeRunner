package game

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	// ProjectileArrow is loosed by the player's bow and kills the first
	// monster it reaches.
	ProjectileArrow ProjectileKind = iota

	// ProjectileFireball is spat by demons and the boss and burns the player
	// if it reaches them.
	ProjectileFireball
)

const (
	// arrowSpeed and fireballSpeed are how fast each projectile flies, in
	// tiles per second. Arrows outpace the player; fireballs are slower than
	// the player so that a sidestep dodges them.
	arrowSpeed    = float32(16)
	fireballSpeed = float32(7)
)

var (
	arrowHeadColor    = color.RGBA{R: 0x89, G: 0xDC, B: 0xEB, A: 0xFF}
	arrowShaftColor   = color.RGBA{R: 0xCD, G: 0xD6, B: 0xF4, A: 0xFF}
	fireballColor     = color.RGBA{R: 0xFA, G: 0xB3, B: 0x87, A: 0xFF}
	fireballCoreColor = color.RGBA{R: 0xF9, G: 0xE2, B: 0xAF, A: 0xFF}
)

// Projectile is something in flight along a row or a column. It moves
// smoothly in pixels rather than stepping tile to tile, but it collides by
// the tile its centre is over: it dies on the first solid tile it enters and
// on the first target it shares a tile with.
type Projectile struct {
	alive     bool
	direction Vector
	kind      ProjectileKind
	world     *World
	x         float32
	y         float32
}

// ProjectileKind is what a projectile is and therefore what it hurts.
type ProjectileKind uint8

// NewProjectile launches a projectile from the centre of a tile.
//
// Parameters:
//   - world: the world the projectile flies through.
//   - kind: what is being fired.
//   - origin: the tile it starts from, normally the shooter's own tile.
//   - direction: the unit step it flies along.
//
// Returns:
//   - projectile: a projectile in flight.
func NewProjectile(world *World, kind ProjectileKind, origin Vector, direction Vector) (projectile *Projectile) {
	projectile = &Projectile{alive: true, direction: direction, kind: kind, world: world}
	projectile.SetPosition(origin)

	return projectile
}

// Alive reports whether the projectile is still in flight.
//
// Returns:
//   - alive: false once it has hit something.
func (this *Projectile) Alive() (alive bool) {
	return this.alive
}

// Draw renders the projectile onto screen.
//
// Parameters:
//   - screen: the destination image for this frame.
func (this *Projectile) Draw(screen *ebiten.Image) {
	if !this.alive {
		return
	}

	switch this.kind {
	case ProjectileArrow:
		tailX := this.x - float32(this.direction.X)*5
		tailY := this.y - float32(this.direction.Y)*5
		headX := this.x + float32(this.direction.X)*5
		headY := this.y + float32(this.direction.Y)*5

		vector.StrokeLine(screen, tailX, tailY, headX, headY, 2, arrowShaftColor, true)
		vector.DrawFilledCircle(screen, headX, headY, 2, arrowHeadColor, true)
	case ProjectileFireball:
		vector.DrawFilledCircle(screen, this.x, this.y, 4.5, fireballColor, true)
		vector.DrawFilledCircle(screen, this.x, this.y, 2, fireballCoreColor, true)
	}
}

// Kind reports what the projectile is.
//
// Returns:
//   - kind: the projectile kind.
func (this *Projectile) Kind() (kind ProjectileKind) {
	return this.kind
}

// Position reports the tile the projectile's centre is over.
//
// Returns:
//   - position: the tile, which may be off the map once it flies off an edge.
func (this *Projectile) Position() (position Vector) {
	return Vector{
		X: int(math.Floor(float64(this.x) / GridSize)),
		Y: int(math.Floor(float64(this.y) / GridSize)),
	}
}

// SetPosition moves the projectile to the centre of a tile.
//
// Parameters:
//   - position: the tile to move to.
func (this *Projectile) SetPosition(position Vector) {
	this.x = float32(position.X*GridSize) + GridSize/2
	this.y = float32(position.Y*GridSize) + GridSize/2
}

// Update flies the projectile forward one tick and resolves anything it hits.
func (this *Projectile) Update() {
	if !this.alive {
		return
	}

	travel := this.speed() * GridSize * tickSeconds()
	this.x += float32(this.direction.X) * travel
	this.y += float32(this.direction.Y) * travel

	tile := this.Position()
	if !this.world.Dungeon().Walkable(tile.X, tile.Y) {
		this.alive = false

		return
	}

	switch this.kind {
	case ProjectileArrow:
		if monster := this.world.MonsterAt(tile); monster != nil {
			monster.Hurt(1)
			this.alive = false
		}
	case ProjectileFireball:
		if this.world.Player().Position() == tile {
			this.world.HurtPlayer(1)
			this.alive = false
		}
	}
}

// speed reports how fast this kind of projectile flies.
//
// Returns:
//   - speed: the speed, in tiles per second.
func (this *Projectile) speed() (speed float32) {
	if this.kind == ProjectileArrow {
		return arrowSpeed
	}

	return fireballSpeed
}
