package game

import (
	"image/color"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	// Cols is the width of the dungeon in tiles.
	Cols = ScreenWidth / GridSize

	// Rows is the height of the dungeon in tiles.
	Rows = ScreenHeight / GridSize

	// SectorCols is the number of sectors across the dungeon.
	SectorCols = 4

	// SectorRows is the number of sectors down the dungeon.
	SectorRows = 3

	// maxRoomHeight and maxRoomWidth leave a one tile wall margin inside the
	// sector, which guarantees at least two solid tiles between the rooms of
	// neighbouring sectors.
	maxRoomHeight = sectorHeight - 2
	maxRoomWidth  = sectorWidth - 2
	minRoomHeight = 4
	minRoomWidth  = 4
	sectorHeight  = Rows / SectorRows
	sectorWidth   = Cols / SectorCols
)

var (
	corridorColor = color.RGBA{R: 0x26, G: 0x26, B: 0x38, A: 0xFF}
	doorColor     = color.RGBA{R: 0xF9, G: 0xE2, B: 0xAF, A: 0xFF}
	floorColor    = color.RGBA{R: 0x31, G: 0x32, B: 0x44, A: 0xFF}
	wallColor     = color.RGBA{R: 0x18, G: 0x18, B: 0x25, A: 0xFF}
)

// Dungeon is a generated level in the style of Rogue: the map is divided into
// a SectorCols by SectorRows grid, every sector holds exactly one room, and
// each room is joined to its right and lower neighbour by a dogleg corridor.
// Connecting every neighbouring pair makes the level connected by
// construction, so no reachability pass is needed.
type Dungeon struct {
	image *ebiten.Image
	rooms []Rect
	tiles [Cols][Rows]Tile
}

// NewDungeon generates a complete level.
//
// Notes:
//   - the same seed always produces the same level. The generator only ever
//     reads from random, so swapping in a source backed by file bytes is
//     enough to make a level a pure function of that file.
//
// Parameters:
//   - seed: the value the level is generated from.
//
// Returns:
//   - dungeon: a fully dug and rendered level.
func NewDungeon(seed uint64) (dungeon *Dungeon) {
	random := rand.New(rand.NewPCG(seed, 0x9E3779B97F4A7C15))

	dungeon = &Dungeon{}
	dungeon.placeRooms(random)
	dungeon.digCorridors(random)
	dungeon.render()

	return dungeon
}

// Draw blits the pre-rendered level onto screen.
//
// Parameters:
//   - screen: the destination image for this frame.
func (this *Dungeon) Draw(screen *ebiten.Image) {
	screen.DrawImage(this.image, nil)
}

// SpawnPoint reports where a new player should start.
//
// Returns:
//   - spawnPoint: the centre tile of the first room.
func (this *Dungeon) SpawnPoint() (spawnPoint Vector) {
	return this.rooms[0].Center()
}

// Walkable reports whether the tile at x, y can be stood on.
//
// Parameters:
//   - x: the tile column.
//   - y: the tile row.
//
// Returns:
//   - walkable: false for walls and for anything off the map.
func (this *Dungeon) Walkable(x int, y int) (walkable bool) {
	if x < 0 || x >= Cols || y < 0 || y >= Rows {
		return false
	}

	return this.tiles[x][y].Walkable()
}

// carve turns a solid tile into a corridor, or into a door when it sits on the
// wall ring of a room. Tiles that are already dug are left alone, so a
// corridor crossing a room does not overwrite its floor.
//
// Parameters:
//   - x: the tile column.
//   - y: the tile row.
func (this *Dungeon) carve(x int, y int) {
	if this.tiles[x][y] != TileWall {
		return
	}

	if this.isRoomWall(x, y) {
		this.tiles[x][y] = TileDoor

		return
	}

	this.tiles[x][y] = TileCorridor
}

// carveColumn digs the vertical run of tiles between fromY and toY inclusive.
//
// Parameters:
//   - x: the column to dig down.
//   - fromY: one end of the run.
//   - toY: the other end of the run.
func (this *Dungeon) carveColumn(x int, fromY int, toY int) {
	if fromY > toY {
		fromY, toY = toY, fromY
	}

	for y := fromY; y <= toY; y++ {
		this.carve(x, y)
	}
}

// carveRow digs the horizontal run of tiles between fromX and toX inclusive.
//
// Parameters:
//   - y: the row to dig along.
//   - fromX: one end of the run.
//   - toX: the other end of the run.
func (this *Dungeon) carveRow(y int, fromX int, toX int) {
	if fromX > toX {
		fromX, toX = toX, fromX
	}

	for x := fromX; x <= toX; x++ {
		this.carve(x, y)
	}
}

// digCorridors joins every room to its right and lower neighbour.
//
// Parameters:
//   - random: the source every corridor turn is drawn from.
func (this *Dungeon) digCorridors(random *rand.Rand) {
	for sectorY := range SectorRows {
		for sectorX := range SectorCols {
			room := this.rooms[sectorY*SectorCols+sectorX]

			if sectorX+1 < SectorCols {
				this.digHorizontal(room, this.rooms[sectorY*SectorCols+sectorX+1], random)
			}

			if sectorY+1 < SectorRows {
				this.digVertical(room, this.rooms[(sectorY+1)*SectorCols+sectorX], random)
			}
		}
	}
}

// digHorizontal joins two side by side rooms with a right, down, right dogleg.
// The vertical leg is placed strictly in the gap between the two rooms, so it
// cannot cut through either of them, and both rooms sit in the same band of
// sector rows, so it cannot reach a room in another band either.
//
// Parameters:
//   - left: the room on the left.
//   - right: the room on the right.
//   - random: the source the turn column is drawn from.
func (this *Dungeon) digHorizontal(left Rect, right Rect, random *rand.Rand) {
	start := left.Center()
	end := right.Center()
	turn := left.Right() + random.IntN(right.X-left.Right())

	this.carveRow(start.Y, start.X, turn)
	this.carveColumn(turn, start.Y, end.Y)
	this.carveRow(end.Y, turn, end.X)
}

// digVertical joins two stacked rooms with a down, across, down dogleg. The
// horizontal leg is placed strictly in the gap between the two rooms, and both
// rooms sit in the same band of sector columns, so it cannot cut through any
// room.
//
// Parameters:
//   - top: the upper room.
//   - bottom: the lower room.
//   - random: the source the turn row is drawn from.
func (this *Dungeon) digVertical(top Rect, bottom Rect, random *rand.Rand) {
	start := top.Center()
	end := bottom.Center()
	turn := top.Bottom() + random.IntN(bottom.Y-top.Bottom())

	this.carveColumn(start.X, start.Y, turn)
	this.carveRow(turn, start.X, end.X)
	this.carveColumn(end.X, turn, end.Y)
}

// isRoomWall reports whether the tile at x, y lies on the one tile ring around
// a room. Only solid tiles are ever tested, so a tile inside the ring bounds
// is necessarily on the ring itself rather than in the room interior.
//
// Parameters:
//   - x: the tile column.
//   - y: the tile row.
//
// Returns:
//   - isRoomWall: true when the tile borders a room.
func (this *Dungeon) isRoomWall(x int, y int) (isRoomWall bool) {
	for _, room := range this.rooms {
		if x < room.X-1 || x > room.Right() || y < room.Y-1 || y > room.Bottom() {
			continue
		}

		return true
	}

	return false
}

// placeRooms puts one room in every sector, in reading order, and floors it.
//
// Parameters:
//   - random: the source room sizes and offsets are drawn from.
func (this *Dungeon) placeRooms(random *rand.Rand) {
	for sectorY := range SectorRows {
		for sectorX := range SectorCols {
			width := minRoomWidth + random.IntN(maxRoomWidth-minRoomWidth+1)
			height := minRoomHeight + random.IntN(maxRoomHeight-minRoomHeight+1)

			room := Rect{
				X:      sectorX*sectorWidth + 1 + random.IntN(maxRoomWidth-width+1),
				Y:      sectorY*sectorHeight + 1 + random.IntN(maxRoomHeight-height+1),
				Width:  width,
				Height: height,
			}

			this.rooms = append(this.rooms, room)

			for y := room.Y; y < room.Bottom(); y++ {
				for x := room.X; x < room.Right(); x++ {
					this.tiles[x][y] = TileFloor
				}
			}
		}
	}
}

// render bakes the tile grid into a single image so that drawing a frame costs
// one blit instead of one filled rectangle per tile.
func (this *Dungeon) render() {
	this.image = ebiten.NewImage(ScreenWidth, ScreenHeight)
	this.image.Fill(wallColor)

	for y := range Rows {
		for x := range Cols {
			var tileColor color.RGBA

			switch this.tiles[x][y] {
			case TileFloor:
				tileColor = floorColor
			case TileCorridor:
				tileColor = corridorColor
			case TileDoor:
				tileColor = doorColor
			default:
				continue
			}

			vector.DrawFilledRect(
				this.image,
				float32(x*GridSize),
				float32(y*GridSize),
				GridSize,
				GridSize,
				tileColor,
				false)
		}
	}
}
