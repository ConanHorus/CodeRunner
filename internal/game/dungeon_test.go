package game

import "testing"

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
