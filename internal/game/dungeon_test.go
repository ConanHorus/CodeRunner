package game

import (
	"math"
	"testing"
)

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
