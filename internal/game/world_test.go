package game

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// worldTestSeeds is how many populated levels the placement tests sweep. The
// worlds are heavier than bare dungeons, so fewer are swept.
const worldTestSeeds = 200

func TestPopulationIsPlacedOnFreeFloor(t *testing.T) {
	for seed := range uint64(worldTestSeeds) {
		world := NewWorld(seed)
		dungeon := world.Dungeon()
		taken := map[Vector]string{world.Player().Position(): "player"}

		claim := func(position Vector, what string) {
			if !dungeon.Walkable(position.X, position.Y) {
				t.Fatalf("seed %d: %s placed on solid tile %v", seed, what, position)
			}

			if dungeon.TileAt(position.X, position.Y) != TileFloor {
				t.Fatalf("seed %d: %s placed off room floor at %v", seed, what, position)
			}

			if other, clash := taken[position]; clash {
				t.Fatalf("seed %d: %s and %s share tile %v", seed, what, other, position)
			}

			taken[position] = what
		}

		for _, item := range world.Items() {
			claim(item.Position(), item.Kind().Name())
		}

		for _, monster := range world.Monsters() {
			claim(monster.Position(), monster.Name())
		}

		world.Dispose()
	}
}

func TestItemsAreReachableBeforeUnlock(t *testing.T) {
	for seed := range uint64(worldTestSeeds) {
		world := NewWorld(seed)
		dungeon := world.Dungeon()
		reachable := reachableTiles(dungeon, dungeon.SpawnPoint())
		counts := map[ItemKind]int{}
		heartRooms := map[Rect]bool{}
		antechamber := dungeon.Room(bossAntechamber(dungeon))
		heartOutsideBoss := false

		for _, item := range world.Items() {
			kind := item.Kind()
			counts[kind]++

			if !reachable[item.Position()] {
				t.Fatalf("seed %d: %s at %v unreachable before unlock", seed, kind.Name(), item.Position())
			}

			if dungeon.BossRoom().Contains(item.Position()) {
				t.Fatalf("seed %d: %s at %v is inside the boss room", seed, kind.Name(), item.Position())
			}

			if kind != ItemHeart {
				continue
			}

			room := roomContaining(dungeon, item.Position())
			if heartRooms[room] {
				t.Fatalf("seed %d: two hearts in room %v", seed, room)
			}

			heartRooms[room] = true

			if room == antechamber {
				heartOutsideBoss = true
			}
		}

		if counts[ItemKey] != 1 || counts[ItemBow] != 1 {
			t.Fatalf("seed %d: %d keys and %d bows, want 1 of each", seed, counts[ItemKey], counts[ItemBow])
		}

		if counts[ItemHeart] != heartCount {
			t.Fatalf("seed %d: %d hearts, want %d", seed, counts[ItemHeart], heartCount)
		}

		if !heartOutsideBoss {
			t.Fatalf("seed %d: no heart in the boss room's antechamber %v", seed, antechamber)
		}

		world.Dispose()
	}
}

func TestPlayerStartsArmedWithTheSword(t *testing.T) {
	world := NewWorld(3)
	defer world.Dispose()

	player := world.Player()

	if player.Weapon() != WeaponSword {
		t.Fatalf("player starts holding %s, want Sword", player.Weapon().Name())
	}

	if player.Health() != PlayerMaxHealth {
		t.Fatalf("player starts with %d health, want %d", player.Health(), PlayerMaxHealth)
	}

	player.swapWeapon()

	if player.Weapon() != WeaponSword {
		t.Fatalf("swapping with only a sword should keep the sword, got %s", player.Weapon().Name())
	}
}

func TestBossAloneGuardsTheBossRoom(t *testing.T) {
	for seed := range uint64(worldTestSeeds) {
		world := NewWorld(seed)
		dungeon := world.Dungeon()
		entrance := dungeon.Room(dungeon.Entrance())
		bosses := 0

		for _, monster := range world.Monsters() {
			inBossRoom := dungeon.BossRoom().Contains(monster.Position())

			if monster.IsBoss() {
				bosses++

				if !inBossRoom {
					t.Fatalf("seed %d: boss at %v is outside the boss room", seed, monster.Position())
				}

				if monster.Health() != BossMaxHealth {
					t.Fatalf("seed %d: boss has %d health, want %d", seed, monster.Health(), BossMaxHealth)
				}

				continue
			}

			if inBossRoom {
				t.Fatalf("seed %d: %s at %v is inside the boss room", seed, monster.Name(), monster.Position())
			}

			if entrance.Contains(monster.Position()) {
				t.Fatalf("seed %d: %s at %v is inside the entrance room", seed, monster.Name(), monster.Position())
			}

			if monster.Health() != 1 {
				t.Fatalf("seed %d: %s has %d health, want 1", seed, monster.Name(), monster.Health())
			}
		}

		if bosses != 1 {
			t.Fatalf("seed %d: found %d bosses, want 1", seed, bosses)
		}

		if len(world.Monsters()) < 1+dungeon.RoomCount()-2 {
			t.Fatalf("seed %d: only %d monsters for %d rooms", seed, len(world.Monsters()), dungeon.RoomCount())
		}

		world.Dispose()
	}
}

func TestBossDeathOpensTheExit(t *testing.T) {
	world := NewWorld(7)
	defer world.Dispose()

	dungeon := world.Dungeon()
	boss := bossOf(world)

	boss.Hurt(1)
	boss.Hurt(1)
	world.Update()

	if !boss.Alive() || dungeon.ExitOpen() {
		t.Fatalf("boss should survive two hits with the exit still sealed")
	}

	boss.Hurt(1)
	world.Update()

	if boss.Alive() || world.BossAlive() {
		t.Fatalf("boss should die on the third hit")
	}

	if !dungeon.ExitOpen() {
		t.Fatalf("exit should open when the boss dies")
	}

	if world.Message() == "" {
		t.Fatalf("boss death should be announced")
	}

	for _, monster := range world.Monsters() {
		if monster == boss {
			t.Fatalf("dead boss should be swept out of the world")
		}
	}
}

func TestKeyUnlocksTheBossRoom(t *testing.T) {
	for seed := range uint64(worldTestSeeds) {
		world := NewWorld(seed)
		dungeon := world.Dungeon()
		player := world.Player()

		outside, direction, found := thresholdOutsideBossRoom(dungeon)
		if !found {
			t.Fatalf("seed %d: no corridor tile leads to a locked door", seed)
		}

		player.SetPosition(outside)
		player.step(direction)

		if !dungeon.Locked() || player.Position() != outside {
			t.Fatalf("seed %d: locked door opened without the key", seed)
		}

		player.GiveKey()
		player.step(direction)

		if dungeon.Locked() {
			t.Fatalf("seed %d: locked door did not open for the key", seed)
		}

		if player.HasKey() {
			t.Fatalf("seed %d: key was not used up", seed)
		}

		if player.Position() != outside.Add(direction) {
			t.Fatalf("seed %d: player did not step through the unlocked door", seed)
		}

		world.Dispose()
	}
}

func TestPickingUpTheBowEquipsIt(t *testing.T) {
	world := NewWorld(3)
	defer world.Dispose()

	player := world.Player()
	bow := itemOfKind(world, ItemBow)

	player.SetPosition(bow.Position())
	player.collect()

	if player.Weapon() != WeaponBow {
		t.Fatalf("player holds %s after picking up the bow, want Bow", player.Weapon().Name())
	}

	if world.ItemAt(bow.Position()) != nil {
		t.Fatalf("bow should be gone from the floor once picked up")
	}

	player.swapWeapon()

	if player.Weapon() != WeaponSword {
		t.Fatalf("swap should cycle back to the sword, got %s", player.Weapon().Name())
	}

	player.swapWeapon()

	if player.Weapon() != WeaponBow {
		t.Fatalf("swap should cycle on to the bow, got %s", player.Weapon().Name())
	}
}

func TestHeartHealsOnlyWhenHurt(t *testing.T) {
	world := NewWorld(3)
	defer world.Dispose()

	player := world.Player()
	heart := itemOfKind(world, ItemHeart)

	player.SetPosition(heart.Position())
	player.collect()

	if world.ItemAt(heart.Position()) == nil {
		t.Fatalf("heart should stay on the floor when health is already full")
	}

	if player.Health() != PlayerMaxHealth {
		t.Fatalf("player has %d health, want %d", player.Health(), PlayerMaxHealth)
	}

	player.Damage(2)
	player.collect()

	if world.ItemAt(heart.Position()) != nil {
		t.Fatalf("heart should be picked up when hurt")
	}

	if player.Health() != PlayerMaxHealth-2+heartHealing {
		t.Fatalf("player has %d health after a heart, want %d", player.Health(), PlayerMaxHealth-2+heartHealing)
	}

	if world.Message() == "" {
		t.Fatalf("picking up a heart should be announced")
	}
}

func TestSwordKillsAnAdjacentMonster(t *testing.T) {
	world := NewWorld(11)
	defer world.Dispose()

	player := world.Player()
	monster, standing, found := monsterWithFreeNeighbour(world)
	if !found {
		t.Fatalf("no ordinary monster has a free neighbouring tile")
	}

	player.SetPosition(standing)
	player.SetFacing(monster.Position().Sub(standing))
	player.GiveWeapon(WeaponSword)

	before := len(world.Monsters())
	player.attack()
	world.Update()

	if monster.Alive() {
		t.Fatalf("%s survived a sword blow", monster.Name())
	}

	if len(world.Monsters()) != before-1 {
		t.Fatalf("dead %s was not swept: %d monsters, want %d", monster.Name(), len(world.Monsters()), before-1)
	}
}

func TestArrowKillsTheMonsterItReaches(t *testing.T) {
	world := NewWorld(11)
	defer world.Dispose()

	player := world.Player()
	monster, standing, found := monsterWithFreeNeighbour(world)
	if !found {
		t.Fatalf("no ordinary monster has a free neighbouring tile")
	}

	player.SetPosition(standing)
	player.SetFacing(monster.Position().Sub(standing))
	player.GiveWeapon(WeaponBow)
	player.attack()

	if len(world.Projectiles()) != 1 {
		t.Fatalf("bow loosed %d projectiles, want 1", len(world.Projectiles()))
	}

	for tick := 0; tick < ebiten.TPS() && monster.Alive(); tick++ {
		world.Update()
	}

	if monster.Alive() {
		t.Fatalf("%s survived an arrow", monster.Name())
	}

	if len(world.Projectiles()) != 0 {
		t.Fatalf("spent arrow was not swept")
	}
}

func TestFireballHurtsAndShakes(t *testing.T) {
	world := NewWorld(5)
	defer world.Dispose()

	player := world.Player()
	origin := player.Position().Add(Vector{X: -1})
	world.SpawnProjectile(NewProjectile(world, ProjectileFireball, origin, Vector{X: 1}))

	for tick := 0; tick < ebiten.TPS() && player.Health() == PlayerMaxHealth; tick++ {
		world.Update()
	}

	if player.Health() != PlayerMaxHealth-1 {
		t.Fatalf("player has %d health after a fireball, want %d", player.Health(), PlayerMaxHealth-1)
	}

	if offsetX, offsetY := world.Shake().Offset(); offsetX == 0 && offsetY == 0 {
		t.Fatalf("a landed hit should shake the screen")
	}

	if world.HurtPlayer(1); player.Health() != PlayerMaxHealth-1 {
		t.Fatalf("player should be invulnerable right after a hit")
	}
}

func TestAdjacentMonsterClawsThePlayer(t *testing.T) {
	world := NewWorld(11)
	defer world.Dispose()

	player := world.Player()
	monster, standing, found := monsterWithFreeNeighbour(world)
	if !found {
		t.Fatalf("no ordinary monster has a free neighbouring tile")
	}

	player.SetPosition(standing)

	for tick := 0; tick < 2*ebiten.TPS() && player.Health() == PlayerMaxHealth; tick++ {
		world.Update()
	}

	if player.Health() != PlayerMaxHealth-1 {
		t.Fatalf("player has %d health beside a %s, want %d", player.Health(), monster.Name(), PlayerMaxHealth-1)
	}
}

func TestMonstersChaseAlongTheDistanceField(t *testing.T) {
	world := NewWorld(11)
	defer world.Dispose()

	player := world.Player()
	monster, _, found := monsterWithFreeNeighbour(world)
	if !found {
		t.Fatalf("no ordinary monster has a free neighbouring tile")
	}

	room := roomContaining(world.Dungeon(), monster.Position())
	corner := Vector{X: room.X, Y: room.Y}
	if corner == monster.Position() || world.Occupied(corner) {
		corner = Vector{X: room.Right() - 1, Y: room.Bottom() - 1}
	}

	player.SetPosition(corner)
	start := monster.Position().Manhattan(corner)

	for tick := 0; tick < 3*ebiten.TPS() && monster.Position().Manhattan(corner) > 1; tick++ {
		world.Update()
	}

	if after := monster.Position().Manhattan(corner); after >= start && after > 1 {
		t.Fatalf("%s did not close in: %d tiles away, was %d", monster.Name(), after, start)
	}
}

// bossAntechamber finds the one room joined to the boss room.
//
// Parameters:
//   - dungeon: the level.
//
// Returns:
//   - room: the index of the room a corridor away from the boss room.
func bossAntechamber(dungeon *Dungeon) (room int) {
	for index, distance := range dungeon.RoomDistances(dungeon.Exit()) {
		if distance == 1 {
			return index
		}
	}

	return -1
}

// bossOf finds the boss in a world.
//
// Parameters:
//   - world: the world to search.
//
// Returns:
//   - boss: the boss monster.
func bossOf(world *World) (boss *Monster) {
	for _, monster := range world.Monsters() {
		if monster.IsBoss() {
			return monster
		}
	}

	return nil
}

// itemOfKind finds the item of one kind lying in a world.
//
// Parameters:
//   - world: the world to search.
//   - kind: the item kind wanted.
//
// Returns:
//   - item: the item, or nil if none of that kind is lying about.
func itemOfKind(world *World, kind ItemKind) (item *Item) {
	for _, candidate := range world.Items() {
		if candidate.Kind() == kind {
			return candidate
		}
	}

	return nil
}

// monsterWithFreeNeighbour finds an ordinary monster with an empty floor tile
// beside it for the player to stand on.
//
// Parameters:
//   - world: the world to search.
//
// Returns:
//   - monster: the monster.
//   - standing: the free tile beside it.
//   - found: false if no such monster exists.
func monsterWithFreeNeighbour(world *World) (monster *Monster, standing Vector, found bool) {
	dungeon := world.Dungeon()

	for _, candidate := range world.Monsters() {
		if candidate.IsBoss() {
			continue
		}

		for _, direction := range cardinalDirections {
			neighbour := candidate.Position().Add(direction)

			if dungeon.TileAt(neighbour.X, neighbour.Y) == TileFloor && !world.Occupied(neighbour) {
				return candidate, neighbour, true
			}
		}
	}

	return nil, Vector{}, false
}

// roomContaining finds the room a tile lies in.
//
// Parameters:
//   - dungeon: the level.
//   - position: the tile.
//
// Returns:
//   - room: the room holding the tile, or the zero Rect if none does.
func roomContaining(dungeon *Dungeon, position Vector) (room Rect) {
	for index := range dungeon.RoomCount() {
		if candidate := dungeon.Room(index); candidate.Contains(position) {
			return candidate
		}
	}

	return Rect{}
}

// thresholdOutsideBossRoom finds a walkable tile just outside a locked door.
//
// Parameters:
//   - dungeon: the level.
//
// Returns:
//   - outside: the corridor tile next to the door.
//   - direction: the unit step from outside onto the door.
//   - found: false if no locked door has a walkable tile outside it.
func thresholdOutsideBossRoom(dungeon *Dungeon) (outside Vector, direction Vector, found bool) {
	boss := dungeon.BossRoom()

	for _, door := range dungeon.LockedDoors() {
		for _, step := range cardinalDirections {
			neighbour := door.Add(step)

			if boss.Contains(neighbour) || !dungeon.Walkable(neighbour.X, neighbour.Y) {
				continue
			}

			return neighbour, Vector{X: -step.X, Y: -step.Y}, true
		}
	}

	return Vector{}, Vector{}, false
}
