package game

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	// MonsterGhost drifts straight at the player and claws them when adjacent.
	MonsterGhost MonsterKind = iota

	// MonsterBat is fast but flighty: half its moves are random, so it
	// zigzags toward the player rather than beelining.
	MonsterBat

	// MonsterDemon keeps its distance and spits fireballs down any clear row
	// or column the player is on.
	MonsterDemon

	// MonsterBoss guards the exit. It takes BossMaxHealth hits, chases and
	// claws like a ghost, spits fireballs like a demon and speeds up as it is
	// wounded.
	MonsterBoss
)

const (
	// BossMaxHealth is how many hits the boss takes to kill. Every other
	// monster dies to one.
	BossMaxHealth = 3

	// bossWoundedSpeedup is the share of its step time the boss sheds by the
	// time it is down to its last hit: at 0.5 it moves twice as fast.
	bossWoundedSpeedup = float32(0.5)

	// monsterHitFlash is how long, in seconds, a monster flashes white after
	// being hit.
	monsterHitFlash = float32(0.15)

	// monsterShootPause is the least a monster stands still, in seconds,
	// after spitting a fireball, so that it does not fire and lunge at once.
	monsterShootPause = float32(0.3)

	// monsterWanderChance is the odds a monster with no target in range takes
	// an idle step on any given move tick.
	monsterWanderChance = float32(0.3)

	// bossHealthPipGap, bossHealthPipHeight and bossHealthPipWidth size the
	// health pips drawn over the boss.
	bossHealthPipGap    = 2
	bossHealthPipHeight = 3
	bossHealthPipWidth  = 4
)

var (
	batColor          = color.RGBA{R: 0xCB, G: 0xA6, B: 0xF7, A: 0xFF}
	bossColor         = color.RGBA{R: 0xF3, G: 0x8B, B: 0xA8, A: 0xFF}
	bossTrimColor     = color.RGBA{R: 0xF9, G: 0xE2, B: 0xAF, A: 0xFF}
	demonColor        = color.RGBA{R: 0xFA, G: 0xB3, B: 0x87, A: 0xFF}
	demonHornColor    = color.RGBA{R: 0xF3, G: 0x8B, B: 0xA8, A: 0xFF}
	ghostColor        = color.RGBA{R: 0xCD, G: 0xD6, B: 0xF4, A: 0xFF}
	monsterEyeColor   = color.RGBA{R: 0x11, G: 0x11, B: 0x1B, A: 0xFF}
	monsterFlashColor = color.RGBA{R: 0xC0, G: 0xC0, B: 0xC0, A: 0xC0}
	monsterGlowColor  = color.RGBA{R: 0xF9, G: 0xE2, B: 0xAF, A: 0xFF}
	pipEmptyColor     = color.RGBA{R: 0x45, G: 0x47, B: 0x5A, A: 0xFF}

	// monsterKinds holds the tuning for every monster kind, indexed by kind.
	monsterKinds = [...]monsterStats{
		MonsterGhost: {
			aggroRange: 9,
			health:     1,
			name:       "Ghost",
			stepTime:   0.4,
		},
		MonsterBat: {
			aggroRange: 7,
			erratic:    0.5,
			health:     1,
			name:       "Bat",
			stepTime:   0.16,
		},
		MonsterDemon: {
			aggroRange:    8,
			health:        1,
			keepsDistance: 2,
			name:          "Demon",
			shootCooldown: 1.8,
			shootRange:    7,
			stepTime:      0.7,
		},
		MonsterBoss: {
			aggroRange:    12,
			boss:          true,
			health:        BossMaxHealth,
			name:          "Boss",
			shootCooldown: 1.2,
			shootRange:    8,
			stepTime:      0.4,
		},
	}
)

// Monster is an enemy that hunts the player through the dungeon. Every
// monster navigates by the world's distance field, stepping to whichever
// neighbouring tile is fewest steps from the player, which walks them around
// walls and through doors without any pathfinding of their own. A monster
// standing next to the player claws them instead of stepping.
type Monster struct {
	health     int
	hitTimer   float32
	kind       MonsterKind
	moveTimer  float32
	position   Vector
	shootTimer float32
	world      *World
}

// MonsterKind is which sort of monster one is.
type MonsterKind uint8

// monsterStats is the tuning that tells one monster kind from another.
type monsterStats struct {
	// aggroRange is how many steps away the player must be, walking around
	// walls, before the monster gives chase.
	aggroRange int

	// boss marks the one monster whose death opens the exit.
	boss bool

	// erratic is the odds a chasing step goes in a random direction instead.
	erratic float32

	// health is how many hits the monster takes to kill.
	health int

	// keepsDistance, when positive, is how close the player must get before
	// the monster backs away rather than closing in.
	keepsDistance int

	// name is the label announcements use.
	name string

	// shootCooldown is the wait, in seconds, between fireballs.
	shootCooldown float32

	// shootRange is how far, in tiles, the monster can spit a fireball. Zero
	// means it never shoots.
	shootRange int

	// stepTime is the wait, in seconds, between steps.
	stepTime float32
}

// NewMonster creates a monster standing on a tile.
//
// Parameters:
//   - world: the world the monster hunts through.
//   - kind: which sort of monster it is.
//   - position: the tile it starts on.
//
// Returns:
//   - monster: a ready-to-run Monster at full health.
func NewMonster(world *World, kind MonsterKind, position Vector) (monster *Monster) {
	return &Monster{
		health:    monsterKinds[kind].health,
		kind:      kind,
		moveTimer: monsterKinds[kind].stepTime,
		position:  position,
		world:     world,
	}
}

// Alive reports whether the monster still has health left.
//
// Returns:
//   - alive: true while health is above zero.
func (this *Monster) Alive() (alive bool) {
	return this.health > 0
}

// Draw renders the monster onto screen.
//
// Parameters:
//   - screen: the destination image for this frame.
func (this *Monster) Draw(screen *ebiten.Image) {
	if !this.Alive() {
		return
	}

	left := float32(this.position.X * GridSize)
	top := float32(this.position.Y * GridSize)

	switch this.kind {
	case MonsterGhost:
		this.drawGhost(screen, left, top)
	case MonsterBat:
		this.drawBat(screen, left, top)
	case MonsterDemon:
		this.drawDemon(screen, left, top)
	case MonsterBoss:
		this.drawBoss(screen, left, top)
	}

	if this.hitTimer > 0 {
		vector.DrawFilledRect(screen, left, top, GridSize, GridSize, monsterFlashColor, false)
	}
}

// Health reports how many more hits the monster can take.
//
// Returns:
//   - health: the health remaining.
func (this *Monster) Health() (health int) {
	return this.health
}

// Hurt takes health off the monster and starts its hit flash.
//
// Parameters:
//   - amount: how many hits to land.
func (this *Monster) Hurt(amount int) {
	this.health = Clamp(this.health-amount, 0, this.stats().health)
	this.hitTimer = monsterHitFlash
}

// IsBoss reports whether this is the monster guarding the exit.
//
// Returns:
//   - isBoss: true for MonsterBoss.
func (this *Monster) IsBoss() (isBoss bool) {
	return this.stats().boss
}

// Kind reports which sort of monster this is.
//
// Returns:
//   - kind: the monster kind.
func (this *Monster) Kind() (kind MonsterKind) {
	return this.kind
}

// Name reports the label announcements use for this monster.
//
// Returns:
//   - name: the human readable monster name.
func (this *Monster) Name() (name string) {
	return this.stats().name
}

// Position reports where the monster stands.
//
// Returns:
//   - position: the monster position, in tiles.
func (this *Monster) Position() (position Vector) {
	return this.position
}

// SetPosition moves the monster without checking the destination.
//
// Parameters:
//   - position: the new monster position, in tiles.
func (this *Monster) SetPosition(position Vector) {
	this.position = position
}

// Update advances the monster by a single tick: it spits a fireball if it can,
// and otherwise, when its step timer runs out, wanders, backs off, chases or
// claws depending on its kind and how close the player is.
func (this *Monster) Update() {
	if !this.Alive() {
		return
	}

	seconds := tickSeconds()
	this.hitTimer -= seconds
	this.moveTimer -= seconds
	this.shootTimer -= seconds

	stats := this.stats()
	distance := this.world.Distance(this.position)
	aggro := distance >= 0 && distance <= stats.aggroRange

	if aggro && stats.shootRange > 0 && this.shootTimer <= 0 {
		if direction, clear := this.lineToPlayer(); clear {
			this.world.SpawnProjectile(NewProjectile(this.world, ProjectileFireball, this.position, direction))
			this.shootTimer = stats.shootCooldown
			this.moveTimer = max(this.moveTimer, monsterShootPause)

			return
		}
	}

	if this.moveTimer > 0 {
		return
	}

	this.moveTimer = this.stepTime()

	switch {
	case !aggro:
		this.wander()
	case stats.erratic > 0 && this.world.Random().Float32() < stats.erratic:
		this.stray()
	case stats.keepsDistance > 0 && distance <= stats.keepsDistance:
		if !this.retreat() {
			this.clawIfAdjacent()
		}
	default:
		this.chase()
	}
}

// canMoveTo reports whether the monster may step onto a tile.
//
// Parameters:
//   - target: the tile to test.
//
// Returns:
//   - canMove: false for solid tiles, the exit, and tiles something else is
//     standing on.
func (this *Monster) canMoveTo(target Vector) (canMove bool) {
	dungeon := this.world.Dungeon()

	return dungeon.Walkable(target.X, target.Y) &&
		dungeon.TileAt(target.X, target.Y) != TileExit &&
		!this.world.Occupied(target)
}

// chase steps to the neighbouring tile fewest steps from the player, or claws
// the player if they are already adjacent.
//
// Notes:
//   - a step that leaves the distance unchanged is allowed, so a monster
//     stuck behind another shuffles sideways instead of freezing.
//   - ties are broken at random so a pack does not stack up in one lane.
func (this *Monster) chase() {
	if this.clawIfAdjacent() {
		return
	}

	current := this.world.Distance(this.position)
	best := this.position
	bestDistance := -1
	ties := 0

	for _, direction := range cardinalDirections {
		target := this.position.Add(direction)
		distance := this.world.Distance(target)

		if distance < 0 || distance > current || !this.canMoveTo(target) {
			continue
		}

		switch {
		case bestDistance < 0 || distance < bestDistance:
			best, bestDistance, ties = target, distance, 1
		case distance == bestDistance:
			ties++
			if this.world.Random().IntN(ties) == 0 {
				best = target
			}
		}
	}

	this.position = best
}

// clawIfAdjacent hurts the player if they stand on a neighbouring tile.
//
// Returns:
//   - clawed: true when the player was adjacent and was struck at.
func (this *Monster) clawIfAdjacent() (clawed bool) {
	if this.position.Manhattan(this.world.Player().Position()) != 1 {
		return false
	}

	this.world.HurtPlayer(1)

	return true
}

// drawBat draws a bat: a small body with a wing either side, flapping with
// the world clock.
//
// Parameters:
//   - screen: the destination image.
//   - left: the tile's left edge, in pixels.
//   - top: the tile's top edge, in pixels.
func (this *Monster) drawBat(screen *ebiten.Image, left float32, top float32) {
	flap := float32((this.world.Ticks() / 6) % 2)

	vector.DrawFilledRect(screen, left+1, top+6-flap, 5, 3, batColor, false)
	vector.DrawFilledRect(screen, left+10, top+6-flap, 5, 3, batColor, false)
	vector.DrawFilledCircle(screen, left+8, top+8, 3.5, batColor, true)
	vector.DrawFilledRect(screen, left+6, top+7, 1, 1, monsterEyeColor, false)
	vector.DrawFilledRect(screen, left+9, top+7, 1, 1, monsterEyeColor, false)
}

// drawBoss draws the boss: a horned brute spilling over its tile with a gold
// trim, and its remaining health as pips above it.
//
// Parameters:
//   - screen: the destination image.
//   - left: the tile's left edge, in pixels.
//   - top: the tile's top edge, in pixels.
func (this *Monster) drawBoss(screen *ebiten.Image, left float32, top float32) {
	vector.StrokeRect(screen, left-1, top+1, GridSize+2, GridSize-1, 2, bossTrimColor, false)
	vector.DrawFilledRect(screen, left+1, top+3, GridSize-2, GridSize-4, bossColor, false)
	vector.DrawFilledRect(screen, left+1, top-2, 3, 5, bossTrimColor, false)
	vector.DrawFilledRect(screen, left+GridSize-4, top-2, 3, 5, bossTrimColor, false)
	vector.DrawFilledRect(screen, left+4, top+7, 3, 3, monsterGlowColor, false)
	vector.DrawFilledRect(screen, left+9, top+7, 3, 3, monsterGlowColor, false)
	vector.DrawFilledRect(screen, left+5, top+12, 6, 1, monsterEyeColor, false)

	pipsWidth := BossMaxHealth*bossHealthPipWidth + (BossMaxHealth-1)*bossHealthPipGap
	pipLeft := left + (GridSize-float32(pipsWidth))/2
	pipTop := top - 7

	for pip := range BossMaxHealth {
		pipColor := pipEmptyColor
		if pip < this.health {
			pipColor = bossColor
		}

		vector.DrawFilledRect(
			screen,
			pipLeft+float32(pip*(bossHealthPipWidth+bossHealthPipGap)),
			pipTop,
			bossHealthPipWidth,
			bossHealthPipHeight,
			pipColor,
			false)
	}
}

// drawDemon draws a demon: a squat body with two horns and glowing eyes.
//
// Parameters:
//   - screen: the destination image.
//   - left: the tile's left edge, in pixels.
//   - top: the tile's top edge, in pixels.
func (this *Monster) drawDemon(screen *ebiten.Image, left float32, top float32) {
	vector.DrawFilledRect(screen, left+3, top+4, 10, 11, demonColor, false)
	vector.DrawFilledRect(screen, left+3, top+1, 2, 4, demonHornColor, false)
	vector.DrawFilledRect(screen, left+11, top+1, 2, 4, demonHornColor, false)
	vector.DrawFilledRect(screen, left+5, top+7, 2, 2, monsterGlowColor, false)
	vector.DrawFilledRect(screen, left+9, top+7, 2, 2, monsterGlowColor, false)
}

// drawGhost draws a ghost: a rounded head over a trailing body, with two
// dark eyes.
//
// Parameters:
//   - screen: the destination image.
//   - left: the tile's left edge, in pixels.
//   - top: the tile's top edge, in pixels.
func (this *Monster) drawGhost(screen *ebiten.Image, left float32, top float32) {
	vector.DrawFilledCircle(screen, left+8, top+7, 6, ghostColor, true)
	vector.DrawFilledRect(screen, left+2, top+7, 12, 7, ghostColor, false)
	vector.DrawFilledRect(screen, left+5, top+5, 2, 3, monsterEyeColor, false)
	vector.DrawFilledRect(screen, left+9, top+5, 2, 3, monsterEyeColor, false)
}

// lineToPlayer checks for a clear shot along the monster's row or column.
//
// Returns:
//   - direction: the unit step toward the player.
//   - clear: true when the player is on the same row or column, at least two
//     tiles away, within shooting range and with nothing solid in between.
func (this *Monster) lineToPlayer() (direction Vector, clear bool) {
	player := this.world.Player().Position()
	gap := player.Sub(this.position)

	if gap.X != 0 && gap.Y != 0 {
		return Vector{}, false
	}

	distance := abs(gap.X) + abs(gap.Y)
	if distance < 2 || distance > this.stats().shootRange {
		return Vector{}, false
	}

	direction = gap.Sign()
	dungeon := this.world.Dungeon()

	for cursor := this.position.Add(direction); cursor != player; cursor = cursor.Add(direction) {
		if !dungeon.Walkable(cursor.X, cursor.Y) {
			return direction, false
		}
	}

	return direction, true
}

// retreat steps to the neighbouring tile that puts the most distance between
// the monster and the player.
//
// Returns:
//   - retreated: false when no neighbouring tile is further from the player.
func (this *Monster) retreat() (retreated bool) {
	current := this.world.Distance(this.position)
	best := this.position
	bestDistance := current

	for _, direction := range cardinalDirections {
		target := this.position.Add(direction)
		distance := this.world.Distance(target)

		if distance <= bestDistance || !this.canMoveTo(target) {
			continue
		}

		best, bestDistance = target, distance
	}

	if best == this.position {
		return false
	}

	this.position = best

	return true
}

// stats reports the tuning for this monster's kind.
//
// Returns:
//   - stats: the kind's tuning.
func (this *Monster) stats() (stats monsterStats) {
	return monsterKinds[this.kind]
}

// stepTime reports how long the monster waits before its next step. The boss
// grows faster as it is wounded; everything else keeps a steady pace.
//
// Returns:
//   - stepTime: the wait, in seconds.
func (this *Monster) stepTime() (stepTime float32) {
	stats := this.stats()
	if !stats.boss {
		return stats.stepTime
	}

	lost := float32(stats.health-this.health) / float32(stats.health)

	return stats.stepTime * (1 - bossWoundedSpeedup*lost)
}

// stray steps in a random direction onto any free tile.
func (this *Monster) stray() {
	direction := cardinalDirections[this.world.Random().IntN(len(cardinalDirections))]
	target := this.position.Add(direction)

	if this.canMoveTo(target) {
		this.position = target
	}
}

// wander sometimes takes an idle step, but only onto room floor, so monsters
// with nobody to chase pace their room rather than drifting off down the
// corridors.
func (this *Monster) wander() {
	if this.world.Random().Float32() >= monsterWanderChance {
		return
	}

	direction := cardinalDirections[this.world.Random().IntN(len(cardinalDirections))]
	target := this.position.Add(direction)

	if this.world.Dungeon().TileAt(target.X, target.Y) == TileFloor && this.canMoveTo(target) {
		this.position = target
	}
}
