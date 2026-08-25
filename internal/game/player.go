package game

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

var (
	// movementKeys is searched in order, so a direction listed earlier wins
	// when several keys are held at once.
	movementKeys = []directionKeys{
		{direction: Vector{X: 0, Y: -1}, keys: []ebiten.Key{ebiten.KeyUp, ebiten.KeyW}},
		{direction: Vector{X: 0, Y: 1}, keys: []ebiten.Key{ebiten.KeyDown, ebiten.KeyS}},
		{direction: Vector{X: -1, Y: 0}, keys: []ebiten.Key{ebiten.KeyLeft, ebiten.KeyA}},
		{direction: Vector{X: 1, Y: 0}, keys: []ebiten.Key{ebiten.KeyRight, ebiten.KeyD}},
	}

	playerColor = color.RGBA{R: 0x89, G: 0xB4, B: 0xFA, A: 0xFF}
)

// Player is the tile aligned avatar the user drives around the dungeon.
//
// Movement is a tap and repeat: a fresh direction steps one tile at once and
// arms the timer, and holding that direction steps again every UpdateTime
// seconds. Changing direction always steps immediately, so the controls stay
// responsive no matter how slow the repeat is set.
type Player struct {
	direction Vector
	dungeon   *Dungeon
	moveTimer float32
	position  Vector
}

// directionKeys binds one movement direction to the keys that request it.
type directionKeys struct {
	direction Vector
	keys      []ebiten.Key
}

// NewPlayer creates a player standing on the dungeon spawn point.
//
// Parameters:
//   - dungeon: the level the player moves through and collides with.
//
// Returns:
//   - player: a ready-to-run Player.
func NewPlayer(dungeon *Dungeon) (player *Player) {
	return &Player{
		dungeon:  dungeon,
		position: dungeon.SpawnPoint(),
	}
}

// Draw renders the player onto screen.
//
// Parameters:
//   - screen: the destination image for this frame.
func (this *Player) Draw(screen *ebiten.Image) {
	vector.DrawFilledRect(
		screen,
		float32(this.position.X*GridSize),
		float32(this.position.Y*GridSize),
		GridSize,
		GridSize,
		playerColor,
		true)
}

// Position reports where the player stands.
//
// Returns:
//   - position: the player position, in tiles.
func (this *Player) Position() (position Vector) {
	return this.position
}

// SetPosition moves the player without checking the destination.
//
// Parameters:
//   - position: the new player position, in tiles.
func (this *Player) SetPosition(position Vector) {
	this.position = position
}

// Update reads the movement keys and steps the player at most one tile.
func (this *Player) Update() {
	direction, pressed := this.inputDirection()
	if !pressed {
		this.direction = Vector{}
		this.moveTimer = 0

		return
	}

	if direction != this.direction {
		this.direction = direction
		this.moveTimer = UpdateTime
		this.step(direction)

		return
	}

	this.moveTimer -= 1 / float32(ebiten.TPS())
	if this.moveTimer > 0 {
		return
	}

	this.moveTimer += UpdateTime
	this.step(direction)
}

// inputDirection reports the direction the user is asking for. A key pressed
// this tick beats one that is merely held, so tapping a new direction while
// another is still down turns immediately.
//
// Returns:
//   - direction: the requested direction, zero when no movement key is down.
//   - pressed: true when a movement key is down.
func (this *Player) inputDirection() (direction Vector, pressed bool) {
	for _, movement := range movementKeys {
		for _, key := range movement.keys {
			if inpututil.IsKeyJustPressed(key) {
				return movement.direction, true
			}
		}
	}

	for _, movement := range movementKeys {
		for _, key := range movement.keys {
			if ebiten.IsKeyPressed(key) {
				return movement.direction, true
			}
		}
	}

	return Vector{}, false
}

// step moves the player one tile, unless a wall is in the way.
//
// Parameters:
//   - direction: the unit step to apply.
func (this *Player) step(direction Vector) {
	target := Vector{X: this.position.X + direction.X, Y: this.position.Y + direction.Y}
	if !this.dungeon.Walkable(target.X, target.Y) {
		return
	}

	this.position = target
}
