package game

import "testing"

func TestExitTileIsPlacedAtExitPoint(t *testing.T) {
	for seed := range uint64(500) {
		dungeon := NewDungeon(seed)
		exit := dungeon.ExitPoint()

		if tile := dungeon.TileAt(exit.X, exit.Y); tile != TileExit {
			t.Fatalf("seed %d: tile at exit point %v is %d, want TileExit", seed, exit, tile)
		}

		if !dungeon.Walkable(exit.X, exit.Y) {
			t.Fatalf("seed %d: exit point %v is not walkable", seed, exit)
		}

		if spawn := dungeon.SpawnPoint(); spawn == exit {
			t.Fatalf("seed %d: spawn point and exit point coincide at %v", seed, exit)
		}
	}
}

func TestExactlyOneExitTile(t *testing.T) {
	for seed := range uint64(500) {
		dungeon := NewDungeon(seed)
		count := 0

		for y := range Rows {
			for x := range Cols {
				if dungeon.TileAt(x, y) == TileExit {
					count++
				}
			}
		}

		if count != 1 {
			t.Fatalf("seed %d: found %d exit tiles, want 1", seed, count)
		}
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
