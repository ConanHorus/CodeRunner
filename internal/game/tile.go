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

	// TileLockedDoor is a threshold into the boss room, sealed until the
	// player brings the key to it. Nothing can pass through it while locked,
	// so the boss stays penned in and the player stays out.
	TileLockedDoor

	// TileExitSealed is the exit before the boss has been defeated. It is
	// solid, so nothing can stand on it, and it becomes TileExit the moment
	// the boss falls.
	TileExitSealed
)

// Tile is the contents of a single dungeon cell.
type Tile uint8

// Walkable reports whether something may stand on this tile.
//
// Returns:
//   - walkable: false for walls, locked doors and the sealed exit, true for
//     everything else.
func (this Tile) Walkable() (walkable bool) {
	switch this {
	case TileWall, TileLockedDoor, TileExitSealed:
		return false
	default:
		return true
	}
}
