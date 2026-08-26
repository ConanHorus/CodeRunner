package game

import "testing"

func TestOpenMapIsFullyVisible(t *testing.T) {
	sight := NewLineOfSight(func(x int, y int) bool { return false })

	for _, origin := range []Vector{{X: 0, Y: 0}, {X: Cols - 1, Y: Rows - 1}, {X: 7, Y: 11}} {
		sight.LookFrom(origin)

		for y := range Rows {
			for x := range Cols {
				if !sight.Visible(x, y) {
					t.Fatalf("from %v: tile %d, %d is not visible on an open map", origin, x, y)
				}
			}
		}
	}
}

func TestOriginIsVisibleEvenWhenWalledIn(t *testing.T) {
	origin := Vector{X: 12, Y: 9}
	sight := NewLineOfSight(func(x int, y int) bool { return Vector{X: x, Y: y} != origin })
	sight.LookFrom(origin)

	if !sight.Visible(origin.X, origin.Y) {
		t.Fatalf("origin %v is not visible", origin)
	}
}

func TestSightFromOffMapSeesNothing(t *testing.T) {
	sight := NewLineOfSight(func(x int, y int) bool { return false })

	for _, origin := range []Vector{{X: -1, Y: 5}, {X: 5, Y: -1}, {X: Cols, Y: 5}, {X: 5, Y: Rows}} {
		sight.LookFrom(origin)

		for y := range Rows {
			for x := range Cols {
				if sight.Visible(x, y) {
					t.Fatalf("from off map %v: tile %d, %d is visible", origin, x, y)
				}
			}
		}
	}
}

func TestSightHasNoRangeLimit(t *testing.T) {
	sight := NewLineOfSight(func(x int, y int) bool { return false })
	sight.LookFrom(Vector{X: 0, Y: 0})

	if !sight.Visible(Cols-1, Rows-1) {
		t.Fatalf("far corner %d, %d is not visible across an open map", Cols-1, Rows-1)
	}
}

func TestVisibleOffMapIsFalse(t *testing.T) {
	sight := NewLineOfSight(func(x int, y int) bool { return false })
	sight.LookFrom(Vector{X: 5, Y: 5})

	for _, point := range []Vector{{X: -1, Y: 5}, {X: 5, Y: -1}, {X: Cols, Y: 5}, {X: 5, Y: Rows}} {
		if sight.Visible(point.X, point.Y) {
			t.Errorf("Visible(%v) off the map is true", point)
		}
	}
}

func TestWallIsSeenButHidesWhatIsBehindIt(t *testing.T) {
	wall := Vector{X: 5, Y: 10}
	sight := NewLineOfSight(func(x int, y int) bool { return Vector{X: x, Y: y} == wall })
	sight.LookFrom(Vector{X: 0, Y: 10})

	if !sight.Visible(wall.X, wall.Y) {
		t.Fatalf("wall %v is not visible", wall)
	}

	for x := wall.X + 1; x < Cols; x++ {
		if sight.Visible(x, wall.Y) {
			t.Fatalf("tile %d, %d behind wall %v is visible", x, wall.Y, wall)
		}
	}
}

func TestSpawnRoomIsFullyVisibleFromSpawn(t *testing.T) {
	for seed := range uint64(200) {
		dungeon := NewDungeon(seed)
		room := dungeon.rooms[dungeon.entrance]

		for y := room.Y; y < room.Bottom(); y++ {
			for x := room.X; x < room.Right(); x++ {
				if !dungeon.sight.Visible(x, y) {
					t.Fatalf("seed %d: spawn room tile %d, %d is not visible from spawn %v",
						seed, x, y, dungeon.SpawnPoint())
				}
			}
		}
	}
}

func TestExploredKeepsWhatSightHasLeftBehind(t *testing.T) {
	divider := 5
	sight := NewLineOfSight(func(x int, y int) bool { return x == divider })
	near := Vector{X: 0, Y: 0}
	far := Vector{X: Cols - 1, Y: Rows - 1}

	sight.LookFrom(near)
	sight.LookFrom(far)

	if sight.Visible(near.X, near.Y) {
		t.Fatalf("tile %v is visible from %v through the dividing wall", near, far)
	}

	if !sight.Explored(near.X, near.Y) {
		t.Fatalf("tile %v was seen from %v but is not explored", near, near)
	}

	if !sight.Explored(far.X, far.Y) {
		t.Fatalf("tile %v is visible but not explored", far)
	}
}

func TestExploredIsFalseForTilesNeverSeen(t *testing.T) {
	origin := Vector{X: 12, Y: 9}
	sight := NewLineOfSight(func(x int, y int) bool { return Vector{X: x, Y: y} != origin })
	sight.LookFrom(origin)

	// A walled in origin still sees the ring of walls around it, and nothing
	// whatever beyond them.
	for y := range Rows {
		for x := range Cols {
			ringed := abs(x-origin.X) <= 1 && abs(y-origin.Y) <= 1

			if sight.Explored(x, y) == ringed {
				continue
			}

			t.Fatalf("tile %d, %d explored is %t, want %t for a walled in origin %v",
				x, y, sight.Explored(x, y), ringed, origin)
		}
	}
}

func abs(value int) (absolute int) {
	if value < 0 {
		return -value
	}

	return value
}

func TestExploredOffMapIsFalse(t *testing.T) {
	sight := NewLineOfSight(func(x int, y int) bool { return false })
	sight.LookFrom(Vector{X: 5, Y: 5})

	for _, point := range []Vector{{X: -1, Y: 5}, {X: 5, Y: -1}, {X: Cols, Y: 5}, {X: 5, Y: Rows}} {
		if sight.Explored(point.X, point.Y) {
			t.Errorf("Explored(%v) off the map is true", point)
		}
	}
}
