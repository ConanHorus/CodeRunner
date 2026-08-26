package game

import (
	"fmt"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	// messageDuration is how long, in seconds, an announcement stays on the
	// heads up display.
	messageDuration = float32(3)

	// populateSalt is mixed into the level seed to give the monsters and items
	// a random stream of their own, so that changing how a level is populated
	// never reshapes its rooms.
	populateSalt = 0x5851F42D4C957F2D
)

// World is everything that lives in a level: the dungeon and the player, the
// monsters hunting them, the items waiting on the floor and the projectiles in
// flight. It owns the update order and the draw order of all of them, keeps
// the distance field the monsters navigate by, and relays the events that tie
// them together: hits shake the screen, the key unlocks the boss room and the
// boss's death opens the exit.
type World struct {
	distances    [Cols][Rows]int
	dungeon      *Dungeon
	items        []*Item
	message      string
	messageTimer float32
	monsters     []*Monster
	player       *Player
	projectiles  []*Projectile
	random       *rand.Rand
	shake        *ScreenShake
	sounds       *Sounds
	ticks        int
}

// NewWorld generates a level and populates it.
//
// Parameters:
//   - seed: the value the level and its population are generated from.
//
// Returns:
//   - world: a ready-to-run World with the player on the spawn point.
func NewWorld(seed uint64) (world *World) {
	world = &World{
		dungeon: NewDungeon(seed),
		random:  rand.New(rand.NewPCG(seed, populateSalt)),
		shake:   NewScreenShake(),
	}

	world.player = NewPlayer(world)
	populate(world, world.random)
	world.computeDistances()

	return world
}

// Announce shows a message on the heads up display for a few seconds,
// replacing any message already showing.
//
// Parameters:
//   - format: a fmt format string.
//   - arguments: the values the format string refers to.
func (this *World) Announce(format string, arguments ...any) {
	this.message = fmt.Sprintf(format, arguments...)
	this.messageTimer = messageDuration
}

// BossAlive reports whether the boss is still standing.
//
// Returns:
//   - alive: true while a living boss is in the world.
func (this *World) BossAlive() (alive bool) {
	for _, monster := range this.monsters {
		if monster.IsBoss() && monster.Alive() {
			return true
		}
	}

	return false
}

// Dispose releases the resources the world holds. It must not be drawn again
// afterwards.
func (this *World) Dispose() {
	this.dungeon.Dispose()
}

// Distance reports how many steps a tile is from the player, walking around
// walls and through open doors.
//
// Parameters:
//   - position: the tile to measure.
//
// Returns:
//   - distance: the step count, or -1 if the tile cannot be reached from the
//     player, which includes every tile behind a locked door.
func (this *World) Distance(position Vector) (distance int) {
	if position.X < 0 || position.X >= Cols || position.Y < 0 || position.Y >= Rows {
		return -1
	}

	return this.distances[position.X][position.Y]
}

// Draw paints the level and everything living in it onto screen, floor first
// and the player last so that they are never hidden.
//
// Parameters:
//   - screen: the destination image for this frame.
func (this *World) Draw(screen *ebiten.Image) {
	this.dungeon.Draw(screen)

	for _, item := range this.items {
		item.Draw(screen)
	}

	for _, monster := range this.monsters {
		monster.Draw(screen)
	}

	for _, projectile := range this.projectiles {
		projectile.Draw(screen)
	}

	this.player.Draw(screen)
}

// Dungeon reports the level the world is set in.
//
// Returns:
//   - dungeon: the level.
func (this *World) Dungeon() (dungeon *Dungeon) {
	return this.dungeon
}

// HurtPlayer lands a hit on the player and shakes the screen if it connects.
//
// Parameters:
//   - amount: how much health the hit takes off.
func (this *World) HurtPlayer(amount int) {
	if this.player.Hurt(amount) {
		this.shake.Shake(ShakeMagnitude, ShakeDuration)
	}
}

// ItemAt finds the item lying on a tile.
//
// Parameters:
//   - position: the tile to look at.
//
// Returns:
//   - item: the item there, or nil if the tile is bare.
func (this *World) ItemAt(position Vector) (item *Item) {
	for _, candidate := range this.items {
		if candidate.Position() == position {
			return candidate
		}
	}

	return nil
}

// Items reports every item still lying on the floor.
//
// Returns:
//   - items: the items, in placement order.
func (this *World) Items() (items []*Item) {
	return this.items
}

// Message reports the announcement currently showing.
//
// Returns:
//   - message: the text, or an empty string when nothing is showing.
func (this *World) Message() (message string) {
	return this.message
}

// MonsterAt finds the living monster standing on a tile.
//
// Parameters:
//   - position: the tile to look at.
//
// Returns:
//   - monster: the monster there, or nil if none is standing on it.
func (this *World) MonsterAt(position Vector) (monster *Monster) {
	for _, candidate := range this.monsters {
		if candidate.Alive() && candidate.Position() == position {
			return candidate
		}
	}

	return nil
}

// Monsters reports every monster still in the world.
//
// Returns:
//   - monsters: the monsters, in placement order.
func (this *World) Monsters() (monsters []*Monster) {
	return this.monsters
}

// Occupied reports whether something is standing on a tile.
//
// Parameters:
//   - position: the tile to test.
//
// Returns:
//   - occupied: true when the player or a living monster is on the tile.
func (this *World) Occupied(position Vector) (occupied bool) {
	return this.player.Position() == position || this.MonsterAt(position) != nil
}

// Player reports the player.
//
// Returns:
//   - player: the player.
func (this *World) Player() (player *Player) {
	return this.player
}

// Projectiles reports every projectile in flight.
//
// Returns:
//   - projectiles: the projectiles, in launch order.
func (this *World) Projectiles() (projectiles []*Projectile) {
	return this.projectiles
}

// Random reports the world's random source, which everything that needs a
// random choice at play time draws from.
//
// Returns:
//   - random: the source.
func (this *World) Random() (random *rand.Rand) {
	return this.random
}

// SetSounds hands the world the sound effects to play as things happen in it.
//
// Notes:
//   - the world is generated before anything is handed to it, and the tests
//     never hand it sound at all, so the sounds stay nil until the game sets
//     them. Every play goes through *Sounds methods, which are silent on a nil
//     Sounds.
//
// Parameters:
//   - sounds: the game's sound effects.
func (this *World) SetSounds(sounds *Sounds) {
	this.sounds = sounds
}

// Shake reports the screen shake the world drives.
//
// Returns:
//   - shake: the shake, for the caller to read the offset from.
func (this *World) Shake() (shake *ScreenShake) {
	return this.shake
}

// Sounds reports the sound effects to play as things happen in the world.
//
// Returns:
//   - sounds: the effects, or nil when the world was built without sound. The
//     nil is safe to play from.
func (this *World) Sounds() (sounds *Sounds) {
	return this.sounds
}

// SpawnProjectile launches a projectile into the world.
//
// Parameters:
//   - projectile: the projectile to launch.
func (this *World) SpawnProjectile(projectile *Projectile) {
	this.projectiles = append(this.projectiles, projectile)
}

// TakeItem lifts the item lying on a tile out of the world.
//
// Parameters:
//   - position: the tile to take from.
//
// Returns:
//   - item: the item that was there, or nil if the tile was bare.
func (this *World) TakeItem(position Vector) (item *Item) {
	for index, candidate := range this.items {
		if candidate.Position() != position {
			continue
		}

		this.items = append(this.items[:index], this.items[index+1:]...)

		return candidate
	}

	return nil
}

// Ticks reports how many updates the world has run, for animations to key
// off.
//
// Returns:
//   - ticks: the update count.
func (this *World) Ticks() (ticks int) {
	return this.ticks
}

// Unlock opens the boss room.
func (this *World) Unlock() {
	this.dungeon.Unlock()
	this.Announce("The key turns. The boss room is open!")
}

// Update advances the world by a single tick: the player acts first, then the
// monsters, then the projectiles, and whatever died along the way is swept up.
// The level is then relit from wherever the step left the player, so that a
// frame never has to work out what can be seen, and the screen shake is
// advanced last so that a hit landed this tick shows up in this tick's frame.
func (this *World) Update() {
	this.ticks++

	this.messageTimer -= tickSeconds()
	if this.messageTimer <= 0 {
		this.message = ""
	}

	this.computeDistances()
	this.player.Update()

	for _, monster := range this.monsters {
		monster.Update()
	}

	for _, projectile := range this.projectiles {
		projectile.Update()
	}

	this.sweep()
	this.dungeon.Update()
	this.dungeon.Illuminate(this.player.Position())
	this.shake.Update()
}

// addItem lays an item on the floor.
//
// Parameters:
//   - item: the item to add.
func (this *World) addItem(item *Item) {
	this.items = append(this.items, item)
}

// addMonster lets a monster loose in the world.
//
// Parameters:
//   - monster: the monster to add.
func (this *World) addMonster(monster *Monster) {
	this.monsters = append(this.monsters, monster)
}

// computeDistances floods the walkable tiles outward from the player so that
// every monster can read how far it is from them, and which way is closer,
// with a single lookup.
func (this *World) computeDistances() {
	for x := range Cols {
		for y := range Rows {
			this.distances[x][y] = -1
		}
	}

	start := this.player.Position()
	this.distances[start.X][start.Y] = 0
	queue := []Vector{start}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, direction := range cardinalDirections {
			next := current.Add(direction)

			if !this.dungeon.Walkable(next.X, next.Y) || this.distances[next.X][next.Y] >= 0 {
				continue
			}

			this.distances[next.X][next.Y] = this.distances[current.X][current.Y] + 1
			queue = append(queue, next)
		}
	}
}

// sweep drops dead monsters and spent projectiles, and opens the exit when
// the boss is among the dead.
func (this *World) sweep() {
	monsters := this.monsters[:0]

	for _, monster := range this.monsters {
		if monster.Alive() {
			monsters = append(monsters, monster)

			continue
		}

		if monster.IsBoss() && !this.dungeon.ExitOpen() {
			this.dungeon.OpenExit()
			this.Announce("The %s falls! The exit is open.", monster.Name())
		}
	}

	this.monsters = monsters

	projectiles := this.projectiles[:0]

	for _, projectile := range this.projectiles {
		if projectile.Alive() {
			projectiles = append(projectiles, projectile)
		}
	}

	this.projectiles = projectiles
}
