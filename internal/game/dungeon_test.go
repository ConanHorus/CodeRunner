package game

import (
	"image/color"
	"math"
	"testing"
)

// testSeeds is how many generated levels the structural tests sweep.
const testSeeds = 500

func TestExitIsSealedUntilOpened(t *testing.T) {
	for seed := range uint64(testSeeds) {
		dungeon := NewDungeon(seed)
		exit := dungeon.ExitPoint()

		if tile := dungeon.TileAt(exit.X, exit.Y); tile != TileExitSealed {
			t.Fatalf("seed %d: tile at exit point %v is %d, want TileExitSealed", seed, exit, tile)
		}

		if dungeon.Walkable(exit.X, exit.Y) || dungeon.ExitOpen() {
			t.Fatalf("seed %d: sealed exit %v should be neither walkable nor open", seed, exit)
		}

		dungeon.OpenExit()

		if tile := dungeon.TileAt(exit.X, exit.Y); tile != TileExit {
			t.Fatalf("seed %d: tile at exit point %v is %d after OpenExit, want TileExit", seed, exit, tile)
		}

		if !dungeon.Walkable(exit.X, exit.Y) || !dungeon.ExitOpen() {
			t.Fatalf("seed %d: open exit %v should be walkable and open", seed, exit)
		}

		if spawn := dungeon.SpawnPoint(); spawn == exit {
			t.Fatalf("seed %d: spawn point and exit point coincide at %v", seed, exit)
		}

		dungeon.Dispose()
	}
}

func TestExactlyOneExitTile(t *testing.T) {
	for seed := range uint64(testSeeds) {
		dungeon := NewDungeon(seed)
		count := 0

		for y := range Rows {
			for x := range Cols {
				switch dungeon.TileAt(x, y) {
				case TileExit, TileExitSealed:
					count++
				}
			}
		}

		if count != 1 {
			t.Fatalf("seed %d: found %d exit tiles, want 1", seed, count)
		}

		dungeon.Dispose()
	}
}

func TestBossRoomHasSingleCorridor(t *testing.T) {
	for seed := range uint64(testSeeds) {
		dungeon := NewDungeon(seed)

		if joins := len(dungeon.links[dungeon.exit]); joins != 1 {
			t.Fatalf("seed %d: boss room has %d corridors, want 1", seed, joins)
		}

		if dungeon.exit == dungeon.entrance {
			t.Fatalf("seed %d: boss room is the entrance room", seed)
		}

		dungeon.Dispose()
	}
}

func TestBossRoomIsSealedUntilUnlocked(t *testing.T) {
	for seed := range uint64(testSeeds) {
		dungeon := NewDungeon(seed)
		boss := dungeon.BossRoom()

		if !dungeon.Locked() || len(dungeon.LockedDoors()) == 0 {
			t.Fatalf("seed %d: boss room starts unlocked", seed)
		}

		reachable := reachableTiles(dungeon, dungeon.SpawnPoint())

		for y := boss.Y; y < boss.Bottom(); y++ {
			for x := boss.X; x < boss.Right(); x++ {
				if reachable[Vector{X: x, Y: y}] {
					t.Fatalf("seed %d: boss room tile %v reachable before unlock", seed, Vector{X: x, Y: y})
				}
			}
		}

		for room := range dungeon.RoomCount() {
			if room == dungeon.Exit() {
				continue
			}

			if center := dungeon.Room(room).Center(); !reachable[center] {
				t.Fatalf("seed %d: room %d centre %v unreachable before unlock", seed, room, center)
			}
		}

		dungeon.Unlock()

		if dungeon.Locked() {
			t.Fatalf("seed %d: boss room still locked after Unlock", seed)
		}

		for y := range Rows {
			for x := range Cols {
				if dungeon.TileAt(x, y) == TileLockedDoor {
					t.Fatalf("seed %d: locked door left at %v after Unlock", seed, Vector{X: x, Y: y})
				}
			}
		}

		if reachable = reachableTiles(dungeon, dungeon.SpawnPoint()); !reachable[boss.Center().Add(Vector{X: 1})] {
			t.Fatalf("seed %d: boss room unreachable after unlock", seed)
		}

		dungeon.Dispose()
	}
}

func TestRoomDistancesCoverEveryRoom(t *testing.T) {
	for seed := range uint64(testSeeds) {
		dungeon := NewDungeon(seed)
		distances := dungeon.RoomDistances(dungeon.Entrance())

		for room, distance := range distances {
			if distance < 0 {
				t.Fatalf("seed %d: room %d unreachable from the entrance", seed, room)
			}
		}

		if distances[dungeon.Entrance()] != 0 {
			t.Fatalf("seed %d: entrance is %d from itself", seed, distances[dungeon.Entrance()])
		}

		dungeon.Dispose()
	}
}

func TestTileAtOffMapIsWall(t *testing.T) {
	dungeon := NewDungeon(1)

	for _, point := range []Vector{{X: -1, Y: 0}, {X: 0, Y: -1}, {X: Cols, Y: 0}, {X: 0, Y: Rows}} {
		if tile := dungeon.TileAt(point.X, point.Y); tile != TileWall {
			t.Errorf("TileAt(%v) = %d, want TileWall", point, tile)
		}
	}
}

func TestBlendSpansTheDarkToTheLitColor(t *testing.T) {
	if blended := blend(floorColor, 1); blended != floorColor {
		t.Errorf("blend(floorColor, 1) = %v, want the lit colour %v", blended, floorColor)
	}

	if blended := blend(floorColor, 0); blended != darkColor {
		t.Errorf("blend(floorColor, 0) = %v, want the dark %v", blended, darkColor)
	}
}

func TestLightLevelIsFullNearTheOriginAndNeverBelowDim(t *testing.T) {
	dungeon := NewDungeon(3)
	spawn := dungeon.SpawnPoint()

	if level := dungeon.lightLevel(spawn.X, spawn.Y); level != 1 {
		t.Errorf("lightLevel at the origin %v = %v, want 1", spawn, level)
	}

	if level := dungeon.lightLevel(spawn.X+lightFull, spawn.Y); level != 1 {
		t.Errorf("lightLevel %d tiles out = %v, want 1", lightFull, level)
	}

	if level := dungeon.lightLevel(spawn.X, spawn.Y+lightRange); level != dimLevel {
		t.Errorf("lightLevel %d tiles out = %v, want dimLevel %v", lightRange, level, dimLevel)
	}

	for y := range Rows {
		for x := range Cols {
			level := dungeon.lightLevel(x, y)

			if level < dimLevel || level > 1 {
				t.Fatalf("lightLevel at %d, %d = %v, want it within [%v, 1]", x, y, level, dimLevel)
			}
		}
	}
}

func TestPulseLevelStaysWithinItsSwing(t *testing.T) {
	dungeon := NewDungeon(4)

	for tick := range 600 {
		dungeon.Update()

		level := dungeon.pulseLevel()
		if level < exitPulseLow || level > 1 {
			t.Fatalf("tick %d: pulseLevel = %v, want it within [%v, 1]", tick, level, exitPulseLow)
		}

		if dungeon.pulse < 0 || dungeon.pulse >= 2*math.Pi {
			t.Fatalf("tick %d: pulse = %v, want it wrapped into [0, 2pi)", tick, dungeon.pulse)
		}
	}
}

func TestSpawnPointStandsAsFarBackFromTheDoorwaysAsItCan(t *testing.T) {
	for seed := range uint64(testSeeds) {
		dungeon := NewDungeon(seed)
		room := dungeon.Room(dungeon.Entrance())
		doorways := dungeon.roomDoorways(room)
		spawn := dungeon.SpawnPoint()

		if !room.Contains(spawn) || dungeon.TileAt(spawn.X, spawn.Y) != TileFloor {
			t.Fatalf("seed %d: spawn %v is not floor inside the entrance room %v", seed, spawn, room)
		}

		reach := nearestOf(doorways, spawn)

		for y := room.Y; y < room.Bottom(); y++ {
			for x := room.X; x < room.Right(); x++ {
				candidate := Vector{X: x, Y: y}

				if nearestOf(doorways, candidate) > reach {
					t.Fatalf("seed %d: spawn %v is %d from the nearest doorway, but %v is %d",
						seed, spawn, reach, candidate, nearestOf(doorways, candidate))
				}
			}
		}

		dungeon.Dispose()
	}
}

func TestSanctuaryCoversTheEntranceRoomAndItsRing(t *testing.T) {
	for seed := range uint64(testSeeds) {
		dungeon := NewDungeon(seed)
		room := dungeon.Room(dungeon.Entrance())
		sanctuary := dungeon.Sanctuary()

		if !sanctuary.Contains(dungeon.SpawnPoint()) {
			t.Fatalf("seed %d: the spawn point is outside the sanctuary", seed)
		}

		for _, doorway := range dungeon.roomDoorways(room) {
			if !sanctuary.Contains(doorway) {
				t.Fatalf("seed %d: doorway %v is outside the sanctuary %v", seed, doorway, sanctuary)
			}
		}

		if sanctuary.Contains(dungeon.ExitPoint()) {
			t.Fatalf("seed %d: the exit lies inside the sanctuary", seed)
		}

		dungeon.Dispose()
	}
}

func TestOrdinaryDoorsArePaintedAsFloor(t *testing.T) {
	if painted := tileColor(TileDoor); painted != floorColor {
		t.Errorf("tileColor(TileDoor) = %v, want the floor colour %v", painted, floorColor)
	}

	if painted := tileColor(TileLockedDoor); painted == floorColor {
		t.Errorf("a locked door should still stand out from the floor")
	}
}

func TestTintScalesEveryChannelIncludingAlpha(t *testing.T) {
	if tinted := tint(exitColor, 1); tinted != exitColor {
		t.Errorf("tint(exitColor, 1) = %v, want the colour untouched %v", tinted, exitColor)
	}

	if tinted := tint(exitColor, 0); tinted != (color.RGBA{}) {
		t.Errorf("tint(exitColor, 0) = %v, want it fully transparent", tinted)
	}

	tinted := tint(exitColor, 0.5)
	if tinted.R > tinted.A || tinted.G > tinted.A || tinted.B > tinted.A {
		t.Errorf("tint(exitColor, 0.5) = %v, want no channel above its alpha", tinted)
	}
}

// nearestOf measures how far a tile is from the closest of a set of tiles.
//
// Parameters:
//   - tiles: the tiles to measure to. An empty set reads as unreachably far.
//   - from: the tile to measure from.
//
// Returns:
//   - distance: the fewest cardinal steps to any of them, ignoring walls.
func nearestOf(tiles []Vector, from Vector) (distance int) {
	distance = Cols + Rows

	for _, tile := range tiles {
		distance = min(distance, from.Manhattan(tile))
	}

	return distance
}

// reachableTiles floods the walkable tiles out from a start tile.
//
// Parameters:
//   - dungeon: the level to walk.
//   - from: the tile to start from.
//
// Returns:
//   - reachable: every tile that can be walked to from the start.
func reachableTiles(dungeon *Dungeon, from Vector) (reachable map[Vector]bool) {
	reachable = map[Vector]bool{from: true}
	queue := []Vector{from}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, direction := range cardinalDirections {
			next := current.Add(direction)

			if reachable[next] || !dungeon.Walkable(next.X, next.Y) {
				continue
			}

			reachable[next] = true
			queue = append(queue, next)
		}
	}

	return reachable
}
