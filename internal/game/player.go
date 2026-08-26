package game

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	// PlayerMaxHealth is the health a freshly spawned player starts with, and
	// the value the health bar reads as full. Three hits end the run.
	PlayerMaxHealth = 3

	// playerBlinkTicks is how many ticks the player is shown, then hidden,
	// while they cannot be hurt after a hit.
	playerBlinkTicks = 4

	// playerEyeSize is the side, in pixels, of the square that shows which
	// way the player is facing, and playerEyeReach is how far from the centre
	// of the tile it sits.
	playerEyeReach = 5
	playerEyeSize  = 4

	// playerInvulnerability is how long, in seconds, the player cannot be
	// hurt again after taking a hit.
	playerInvulnerability = float32(1)

	// playerSlashDuration is how long, in seconds, a sword swing stays on
	// screen.
	playerSlashDuration = float32(0.12)
)

var (
	// attackKeys and swapKeys are the keys that swing the current weapon and
	// cycle to the next one owned.
	attackKeys = []ebiten.Key{ebiten.KeySpace, ebiten.KeyJ}
	swapKeys   = []ebiten.Key{ebiten.KeyQ, ebiten.KeyTab}

	// movementKeys is searched in order, so a direction listed earlier wins
	// when several keys are held at once.
	movementKeys = []directionKeys{
		{direction: Vector{X: 0, Y: -1}, keys: []ebiten.Key{ebiten.KeyUp, ebiten.KeyW}},
		{direction: Vector{X: 0, Y: 1}, keys: []ebiten.Key{ebiten.KeyDown, ebiten.KeyS}},
		{direction: Vector{X: -1, Y: 0}, keys: []ebiten.Key{ebiten.KeyLeft, ebiten.KeyA}},
		{direction: Vector{X: 1, Y: 0}, keys: []ebiten.Key{ebiten.KeyRight, ebiten.KeyD}},
	}

	playerColor    = color.RGBA{R: 0x89, G: 0xB4, B: 0xFA, A: 0xFF}
	playerEyeColor = color.RGBA{R: 0x11, G: 0x11, B: 0x1B, A: 0xFF}
	slashColor     = color.RGBA{R: 0x80, G: 0x86, B: 0x99, A: 0xA0}
)

// Player is the tile aligned avatar the user drives around the dungeon.
//
// Movement is a tap and repeat: a fresh direction steps one tile at once and
// arms the timer, and holding that direction steps again every UpdateTime
// seconds. Changing direction always steps immediately, so the controls stay
// responsive no matter how slow the repeat is set. A blocked step still turns
// the player to face that way, so walking into a monster lines up an attack.
//
// The player fights with whatever weapon they last picked up, swapping between
// the ones they own, and is briefly invulnerable after every hit so that a
// crowd cannot take all their health in one tick.
type Player struct {
	arsenal     []Weapon
	attackTimer float32
	direction   Vector
	facing      Vector
	hasKey      bool
	health      int
	hurtTimer   float32
	moveTimer   float32
	position    Vector
	slashTimer  float32
	weapon      Weapon
	world       *World
}

// directionKeys binds one movement direction to the keys that request it.
type directionKeys struct {
	direction Vector
	keys      []ebiten.Key
}

// NewPlayer creates a player standing on the dungeon spawn point, facing
// down, with the sword already in hand.
//
// Parameters:
//   - world: the world the player moves through and fights in.
//
// Returns:
//   - player: a ready-to-run Player.
func NewPlayer(world *World) (player *Player) {
	return &Player{
		arsenal:  []Weapon{WeaponSword},
		facing:   Vector{X: 0, Y: 1},
		health:   PlayerMaxHealth,
		position: world.Dungeon().SpawnPoint(),
		weapon:   WeaponSword,
		world:    world,
	}
}

// Damage takes health off the player, never dropping below zero.
//
// Parameters:
//   - amount: how much health to take off. A negative amount heals.
func (this *Player) Damage(amount int) {
	this.health = Clamp(this.health-amount, 0, PlayerMaxHealth)
}

// Draw renders the player onto screen, along with any sword swing in
// progress. The player blinks while they cannot be hurt.
//
// Parameters:
//   - screen: the destination image for this frame.
func (this *Player) Draw(screen *ebiten.Image) {
	if this.slashTimer > 0 {
		target := this.position.Add(this.facing)

		vector.DrawFilledRect(
			screen,
			float32(target.X*GridSize),
			float32(target.Y*GridSize),
			GridSize,
			GridSize,
			slashColor,
			false)
	}

	if this.hurtTimer > 0 && (this.world.Ticks()/playerBlinkTicks)%2 == 1 {
		return
	}

	left := float32(this.position.X * GridSize)
	top := float32(this.position.Y * GridSize)

	vector.DrawFilledRect(screen, left, top, GridSize, GridSize, playerColor, true)

	eyeLeft := left + GridSize/2 + float32(this.facing.X*playerEyeReach) - playerEyeSize/2
	eyeTop := top + GridSize/2 + float32(this.facing.Y*playerEyeReach) - playerEyeSize/2

	vector.DrawFilledRect(screen, eyeLeft, eyeTop, playerEyeSize, playerEyeSize, playerEyeColor, false)
}

// Facing reports which way the player is looking, and so which way they
// attack.
//
// Returns:
//   - facing: a unit direction.
func (this *Player) Facing() (facing Vector) {
	return this.facing
}

// GiveKey hands the player the key to the boss room.
func (this *Player) GiveKey() {
	this.hasKey = true
}

// GiveWeapon adds a weapon to the player's arsenal and puts it in their hand.
//
// Parameters:
//   - weapon: the weapon to give. WeaponNone is ignored.
func (this *Player) GiveWeapon(weapon Weapon) {
	if weapon == WeaponNone {
		return
	}

	owned := false

	for _, candidate := range this.arsenal {
		if candidate == weapon {
			owned = true

			break
		}
	}

	if !owned {
		this.arsenal = append(this.arsenal, weapon)
	}

	this.weapon = weapon
}

// HasKey reports whether the player is carrying the key to the boss room.
//
// Returns:
//   - hasKey: true from picking the key up until it is used.
func (this *Player) HasKey() (hasKey bool) {
	return this.hasKey
}

// Heal puts health back on the player, never rising above PlayerMaxHealth.
//
// Parameters:
//   - amount: how much health to put back. A negative amount damages.
func (this *Player) Heal(amount int) {
	this.Damage(-amount)
}

// Health reports the health the player has left.
//
// Returns:
//   - health: the health remaining, in the range [0, PlayerMaxHealth].
func (this *Player) Health() (health int) {
	return this.health
}

// Hurt lands a hit on the player, unless they are still recovering from the
// last one or are already dead.
//
// Parameters:
//   - amount: how much health the hit takes off.
//
// Returns:
//   - hurt: true when the hit connected and health came off.
func (this *Player) Hurt(amount int) (hurt bool) {
	if this.hurtTimer > 0 || this.health <= 0 {
		return false
	}

	this.Damage(amount)
	this.hurtTimer = playerInvulnerability

	return true
}

// Position reports where the player stands.
//
// Returns:
//   - position: the player position, in tiles.
func (this *Player) Position() (position Vector) {
	return this.position
}

// SetFacing turns the player without moving them.
//
// Parameters:
//   - facing: the unit direction to look in.
func (this *Player) SetFacing(facing Vector) {
	this.facing = facing
}

// SetPosition moves the player without checking the destination.
//
// Parameters:
//   - position: the new player position, in tiles.
func (this *Player) SetPosition(position Vector) {
	this.position = position
}

// Update reads the keys and moves, swaps weapons and attacks accordingly.
func (this *Player) Update() {
	seconds := tickSeconds()
	this.attackTimer -= seconds
	this.hurtTimer -= seconds
	this.slashTimer -= seconds

	this.move()

	if anyJustPressed(swapKeys) {
		this.swapWeapon()
	}

	if anyPressed(attackKeys) && this.attackTimer <= 0 {
		this.attack()
	}
}

// Weapon reports the weapon in the player's hand.
//
// Returns:
//   - weapon: the current weapon, the sword until something else is picked up
//     and equipped.
func (this *Player) Weapon() (weapon Weapon) {
	return this.weapon
}

// attack swings the current weapon the way the player is facing: the sword
// hits whatever monster stands on the next tile, the bow looses an arrow and
// sounds the shot. Empty hands do nothing.
func (this *Player) attack() {
	switch this.weapon {
	case WeaponSword:
		this.slashTimer = playerSlashDuration

		if monster := this.world.MonsterAt(this.position.Add(this.facing)); monster != nil {
			monster.Hurt(1)
		}
	case WeaponBow:
		this.world.SpawnProjectile(NewProjectile(this.world, ProjectileArrow, this.position, this.facing))
		this.world.Sounds().PlayShot()
	default:
		return
	}

	this.attackTimer = this.weapon.Cooldown()
}

// collect picks up whatever item is lying on the player's tile. A heart is
// left where it lies when the player has no health to regain, so it can be
// come back for.
func (this *Player) collect() {
	item := this.world.ItemAt(this.position)
	if item == nil {
		return
	}

	switch item.Kind() {
	case ItemKey:
		this.GiveKey()
	case ItemHeart:
		if this.health >= PlayerMaxHealth {
			this.world.Announce("Health is already full.")

			return
		}

		this.Heal(heartHealing)
	default:
		this.GiveWeapon(item.Kind().Weapon())
	}

	this.world.TakeItem(this.position)
	this.world.Announce("Picked up the %s", item.Kind().Name())
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
		if anyJustPressed(movement.keys) {
			return movement.direction, true
		}
	}

	for _, movement := range movementKeys {
		if anyPressed(movement.keys) {
			return movement.direction, true
		}
	}

	return Vector{}, false
}

// move reads the movement keys and steps the player at most one tile.
func (this *Player) move() {
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

	this.moveTimer -= tickSeconds()
	if this.moveTimer > 0 {
		return
	}

	this.moveTimer += UpdateTime
	this.step(direction)
}

// step turns the player to face a direction and moves them one tile that way,
// unless a wall or a monster is in the way. Walking into a locked door with
// the key in hand unlocks the boss room and steps through; without the key it
// only reminds the player what they are missing.
//
// Parameters:
//   - direction: the unit step to apply.
func (this *Player) step(direction Vector) {
	this.facing = direction
	target := this.position.Add(direction)
	dungeon := this.world.Dungeon()

	if dungeon.TileAt(target.X, target.Y) == TileLockedDoor {
		if !this.hasKey {
			this.world.Announce("Locked. Find the key.")

			return
		}

		this.hasKey = false
		this.world.Unlock()
	}

	if !dungeon.Walkable(target.X, target.Y) || this.world.MonsterAt(target) != nil {
		return
	}

	this.position = target
	this.collect()
}

// swapWeapon puts the next weapon the player owns in their hand, wrapping
// round to the first.
func (this *Player) swapWeapon() {
	if len(this.arsenal) < 2 {
		return
	}

	for index, candidate := range this.arsenal {
		if candidate == this.weapon {
			this.weapon = this.arsenal[(index+1)%len(this.arsenal)]

			return
		}
	}

	this.weapon = this.arsenal[0]
}

// anyJustPressed reports whether any of a set of keys went down this tick.
//
// Parameters:
//   - keys: the keys to test.
//
// Returns:
//   - pressed: true when at least one key was pressed this tick.
func anyJustPressed(keys []ebiten.Key) (pressed bool) {
	for _, key := range keys {
		if inpututil.IsKeyJustPressed(key) {
			return true
		}
	}

	return false
}

// anyPressed reports whether any of a set of keys is down.
//
// Parameters:
//   - keys: the keys to test.
//
// Returns:
//   - pressed: true when at least one key is held.
func anyPressed(keys []ebiten.Key) (pressed bool) {
	for _, key := range keys {
		if ebiten.IsKeyPressed(key) {
			return true
		}
	}

	return false
}
