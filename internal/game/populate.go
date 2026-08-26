package game

import "math/rand/v2"

const (
	// heartCount is how many hearts a level holds. One always waits in the
	// room outside the boss room, so the player can top up before the fight;
	// the rest are scattered.
	heartCount = 3
)

// populate stocks a freshly generated level with everything the player will
// meet in it.
//
// Notes:
//   - the entrance room holds no monsters and the player arrives armed.
//   - the key goes in the room that makes the longest detour off the way to
//     the boss, so that finding it means exploring. The boss room is a dead
//     end behind its locked doors, so every other room, and so the key, can
//     be reached without going through it.
//   - the bow waits in one of the remaining rooms, and every room other than
//     the entrance and the boss room gets one to three ordinary monsters.
//   - hearts go one to a room: the first in the antechamber of the boss room,
//     the others wherever the dice land, entrance and boss room excepted.
//   - the boss stands in the boss room, as far from its doors as it can get.
//
// Parameters:
//   - world: the world to populate. Its dungeon must already be generated and
//     its player already placed.
//   - random: the source every placement is drawn from.
func populate(world *World, random *rand.Rand) {
	dungeon := world.Dungeon()
	occupied := map[Vector]bool{world.Player().Position(): true}

	keyRoom := chooseKeyRoom(dungeon, random)
	placeItem(world, ItemKey, pickTile(openTiles(dungeon, dungeon.Room(keyRoom), occupied), random), occupied)

	bowRoom := chooseBowRoom(dungeon, keyRoom, random)
	placeItem(world, ItemBow, pickTile(openTiles(dungeon, dungeon.Room(bowRoom), occupied), random), occupied)

	for _, heartRoom := range chooseHeartRooms(dungeon, random) {
		placeItem(world, ItemHeart, pickTile(openTiles(dungeon, dungeon.Room(heartRoom), occupied), random), occupied)
	}

	for index := range dungeon.RoomCount() {
		if index == dungeon.Entrance() || index == dungeon.Exit() {
			continue
		}

		for range rollMonsterCount(random) {
			tiles := openTiles(dungeon, dungeon.Room(index), occupied)
			if len(tiles) == 0 {
				break
			}

			placeMonster(world, rollMonsterKind(random), pickTile(tiles, random), occupied)
		}
	}

	placeMonster(world, MonsterBoss, chooseBossTile(dungeon, random, occupied), occupied)
}

// chooseBossTile picks where the boss stands: the free tile of the boss room
// furthest from its nearest door, so the player gets a look at it before it
// closes in.
//
// Parameters:
//   - dungeon: the level.
//   - random: the source ties are broken with.
//   - occupied: the tiles already taken.
//
// Returns:
//   - tile: the chosen tile.
func chooseBossTile(dungeon *Dungeon, random *rand.Rand, occupied map[Vector]bool) (tile Vector) {
	doors := dungeon.LockedDoors()
	best := -1

	var candidates []Vector

	for _, candidate := range openTiles(dungeon, dungeon.BossRoom(), occupied) {
		nearest := Cols + Rows

		for _, door := range doors {
			nearest = min(nearest, candidate.Manhattan(door))
		}

		switch {
		case nearest > best:
			best, candidates = nearest, []Vector{candidate}
		case nearest == best:
			candidates = append(candidates, candidate)
		}
	}

	return pickTile(candidates, random)
}

// chooseBowRoom picks the room the bow waits in: any room other than the
// entrance, the boss room and the key's room.
//
// Parameters:
//   - dungeon: the level.
//   - keyRoom: the index of the room holding the key.
//   - random: the source the room is drawn from.
//
// Returns:
//   - room: the index of the chosen room.
func chooseBowRoom(dungeon *Dungeon, keyRoom int, random *rand.Rand) (room int) {
	var candidates []int

	for index := range dungeon.RoomCount() {
		if index == dungeon.Entrance() || index == dungeon.Exit() || index == keyRoom {
			continue
		}

		candidates = append(candidates, index)
	}

	return candidates[random.IntN(len(candidates))]
}

// chooseHeartRooms picks the rooms the hearts wait in: the antechamber of the
// boss room first, then distinct rooms drawn at random from the rest, never
// the entrance or the boss room.
//
// Parameters:
//   - dungeon: the level.
//   - random: the source the rooms are drawn from.
//
// Returns:
//   - rooms: heartCount room indices, antechamber first, or fewer if the level
//     has fewer eligible rooms.
func chooseHeartRooms(dungeon *Dungeon, random *rand.Rand) (rooms []int) {
	fromExit := dungeon.RoomDistances(dungeon.Exit())

	var candidates []int

	for index := range dungeon.RoomCount() {
		switch {
		case index == dungeon.Entrance() || index == dungeon.Exit():
			continue
		case fromExit[index] == 1:
			rooms = append(rooms, index)
		default:
			candidates = append(candidates, index)
		}
	}

	random.Shuffle(len(candidates), func(a int, b int) {
		candidates[a], candidates[b] = candidates[b], candidates[a]
	})

	for _, candidate := range candidates {
		if len(rooms) >= heartCount {
			break
		}

		rooms = append(rooms, candidate)
	}

	return rooms
}

// chooseKeyRoom picks the room the key waits in: the one that adds the most
// walking to the trip from the entrance to the boss room, which is the deepest
// dead end off that route. When every room lies on the route, so that no
// detour exists, any room along it will do.
//
// Parameters:
//   - dungeon: the level.
//   - random: the source ties are broken with.
//
// Returns:
//   - room: the index of the chosen room.
func chooseKeyRoom(dungeon *Dungeon, random *rand.Rand) (room int) {
	fromEntrance := dungeon.RoomDistances(dungeon.Entrance())
	fromExit := dungeon.RoomDistances(dungeon.Exit())
	best := -1

	var candidates []int

	for index := range dungeon.RoomCount() {
		if index == dungeon.Entrance() || index == dungeon.Exit() {
			continue
		}

		detour := fromEntrance[index] + fromExit[index]

		switch {
		case detour > best:
			best, candidates = detour, []int{index}
		case detour == best:
			candidates = append(candidates, index)
		}
	}

	return candidates[random.IntN(len(candidates))]
}

// openTiles lists the floor tiles of a room that nothing has been placed on.
// Tiles right inside a door are left out when the room has any others, so
// that nothing waits to ambush the player on the threshold.
//
// Parameters:
//   - dungeon: the level.
//   - room: the room to search.
//   - occupied: the tiles already taken.
//
// Returns:
//   - tiles: the free tiles, in reading order.
func openTiles(dungeon *Dungeon, room Rect, occupied map[Vector]bool) (tiles []Vector) {
	var thresholds []Vector

	for y := room.Y; y < room.Bottom(); y++ {
		for x := room.X; x < room.Right(); x++ {
			tile := Vector{X: x, Y: y}

			if dungeon.TileAt(x, y) != TileFloor || occupied[tile] {
				continue
			}

			if bordersDoor(dungeon, tile) {
				thresholds = append(thresholds, tile)

				continue
			}

			tiles = append(tiles, tile)
		}
	}

	if len(tiles) == 0 {
		return thresholds
	}

	return tiles
}

// bordersDoor reports whether a tile sits right inside a doorway.
//
// Parameters:
//   - dungeon: the level.
//   - tile: the tile to test.
//
// Returns:
//   - borders: true when a door, locked or not, is on a neighbouring tile.
func bordersDoor(dungeon *Dungeon, tile Vector) (borders bool) {
	for _, direction := range cardinalDirections {
		neighbour := tile.Add(direction)

		switch dungeon.TileAt(neighbour.X, neighbour.Y) {
		case TileDoor, TileLockedDoor:
			return true
		}
	}

	return false
}

// pickTile draws one tile from a list.
//
// Parameters:
//   - tiles: the tiles to choose from. It must not be empty.
//   - random: the source the choice is drawn from.
//
// Returns:
//   - tile: the chosen tile.
func pickTile(tiles []Vector, random *rand.Rand) (tile Vector) {
	return tiles[random.IntN(len(tiles))]
}

// placeItem lays an item on a tile and marks the tile taken.
//
// Parameters:
//   - world: the world to add the item to.
//   - kind: which item to lay down.
//   - tile: where to lay it.
//   - occupied: the tiles already taken, updated with tile.
func placeItem(world *World, kind ItemKind, tile Vector, occupied map[Vector]bool) {
	world.addItem(NewItem(world, kind, tile))
	occupied[tile] = true
}

// placeMonster stands a monster on a tile and marks the tile taken.
//
// Parameters:
//   - world: the world to add the monster to.
//   - kind: which monster to stand there.
//   - tile: where to stand it.
//   - occupied: the tiles already taken, updated with tile.
func placeMonster(world *World, kind MonsterKind, tile Vector, occupied map[Vector]bool) {
	world.addMonster(NewMonster(world, kind, tile))
	occupied[tile] = true
}

// rollMonsterCount decides how many ordinary monsters a room gets.
//
// Parameters:
//   - random: the source the roll is drawn from.
//
// Returns:
//   - count: one or two monsters most of the time, three now and then.
func rollMonsterCount(random *rand.Rand) (count int) {
	switch roll := random.IntN(10); {
	case roll < 4:
		return 1
	case roll < 8:
		return 2
	default:
		return 3
	}
}

// rollMonsterKind decides which ordinary monster to place.
//
// Parameters:
//   - random: the source the roll is drawn from.
//
// Returns:
//   - kind: a ghost most often, then a bat, then a demon.
func rollMonsterKind(random *rand.Rand) (kind MonsterKind) {
	switch roll := random.IntN(20); {
	case roll < 9:
		return MonsterGhost
	case roll < 15:
		return MonsterBat
	default:
		return MonsterDemon
	}
}
