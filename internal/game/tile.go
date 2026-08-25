package game

const (
	// TileWall is solid rock. The player cannot enter it.
	TileWall Tile = iota

	// TileFloor is the open interior of a room.
	TileFloor

	// TileCorridor is a passage dug between two rooms.
	TileCorridor

	// TileDoor is the threshold where a corridor pierces a room wall.
	TileDoor

	// TileExit is the way out of the dungeon. Stepping onto it wins the game.
	TileExit
)

// Tile is the contents of a single dungeon cell.
type Tile uint8

// Walkable reports whether the player may stand on this tile.
//
// Returns:
//   - walkable: true for every tile except TileWall.
func (this Tile) Walkable() (walkable bool) {
	return this != TileWall
}
