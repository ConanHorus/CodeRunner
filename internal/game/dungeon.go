package game

import (
	"image/color"
	"math"
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

	// dimLevel is the least of a tile's lit colour that is ever kept: what a
	// remembered tile keeps, and the floor the light falloff lands on. The rest
	// is given over to the dark the level sits on, so a dimmed tile reads as
	// the same shape seen through the dark instead of as a different tile.
	dimLevel = 0.45

	// exitPulseLow is how much of the exit's colour is kept at the low end of
	// its pulse, and exitPulseRate is how many full pulses it beats a second.
	exitPulseLow  = 0.55
	exitPulseRate = 0.7

	// lightFull is how far, in tiles, light holds at full strength before it
	// starts to fall off, and lightRange is how far it carries before it has
	// faded all the way down to dimLevel. Holding it for the first few tiles
	// puts a pool of proper light around the player instead of starting the
	// fade off from under their feet. Sight itself is not range limited; this
	// only shades what is already visible.
	lightFull  = 4
	lightRange = 16

	// maxRoomHeight and maxRoomWidth leave a one tile wall margin inside the
	// sector, which guarantees at least two solid tiles between the rooms of
	// neighbouring sectors. Those two tiles are the lanes the corridors turn
	// on, so no room floor can ever sit on a lane.
	maxRoomHeight = sectorHeight - 2
	maxRoomWidth  = sectorWidth - 2
	minRoomHeight = 4
	minRoomWidth  = 4
	sectorCount   = SectorCols * SectorRows
	sectorHeight  = Rows / SectorRows
	sectorWidth   = Cols / SectorCols

	// shimmerAlpha is the strongest a shimmer halo is laid over the level, and
	// shimmerRadius is how far, in pixels, it reaches from the middle of its
	// tile. Both are scaled by the shimmer level, so the halo swells as it
	// brightens and shrinks back as it fades.
	shimmerAlpha  = float32(0.4)
	shimmerRadius = float32(12)

	// tileSeam is how many pixels of the dark ground are left showing along
	// the right and bottom of every dug tile, which draws the grid the level
	// is laid out on and gives the floor some texture to move across. Walls
	// are painted whole, so they still read as solid rock.
	tileSeam = 1

	// wallEdgeHeight is how many pixels of the lit lip are painted along the
	// bottom of a wall that has dug ground below it. That is the one wall face
	// a level seen from above would show, and picking it out is what keeps a
	// room reading as a space rather than as a hole cut out of the dark.
	wallEdgeHeight = 3
)

var (
	corridorColor   = color.RGBA{R: 0x26, G: 0x26, B: 0x38, A: 0xFF}
	darkColor       = color.RGBA{R: 0x0B, G: 0x0B, B: 0x11, A: 0xFF}
	exitColor       = color.RGBA{R: 0xA6, G: 0xE3, B: 0xA1, A: 0xFF}
	floorColor      = color.RGBA{R: 0x31, G: 0x32, B: 0x44, A: 0xFF}
	keyholeColor    = color.RGBA{R: 0x11, G: 0x11, B: 0x1B, A: 0xFF}
	lockedDoorColor = color.RGBA{R: 0xF3, G: 0x8B, B: 0xA8, A: 0xFF}
	sealedExitColor = color.RGBA{R: 0x58, G: 0x5B, B: 0x70, A: 0xFF}
	wallColor       = color.RGBA{R: 0x18, G: 0x18, B: 0x25, A: 0xFF}
	wallEdgeColor   = color.RGBA{R: 0x45, G: 0x47, B: 0x5A, A: 0xFF}
)

// Dungeon is a generated level in the style of Rogue laid out as a maze: the
// map is divided into a SectorCols by SectorRows grid, every sector holds
// exactly one room, and the rooms are joined into a spanning tree, which
// leaves exactly one route between any two of them. One extra join is then
// added across the route from the entrance to the exit, opening a second way
// round; every other branch of the tree is a dead end. The tree spans every
// sector, so the level is connected by construction and no reachability pass
// is needed.
//
// The exit room doubles as the boss room. It is always a dead end with a
// single corridor into it, and every walkable tile on its wall ring starts
// out as a locked door, so nothing gets in or out until Unlock is called.
// The exit itself starts sealed and only opens when OpenExit is called, which
// the world does once the boss has fallen.
//
// The entrance room is the mirror of it: a sanctuary no monster may set foot
// in, which the player can always fall back to, and the tile they spawn on is
// set as far back from its doorways as the room allows. Between them the
// player never opens a run already cornered.
//
// Tiles in sight of the viewpoint are rendered lit, fading with distance from
// it, tiles that have been in sight of an earlier viewpoint are rendered dim,
// and everything else is dark. The viewpoint is set with Illuminate, and the
// level is spawn lit until then.
type Dungeon struct {
	entrance    int
	exit        int
	image       *ebiten.Image
	links       [][]int
	lockedDoors []Vector
	pulse       float32
	rooms       []Rect
	sight       *LineOfSight
	spawn       Vector
	tiles       [Cols][Rows]Tile
}

// edge is a join between two neighbouring rooms, held with the lower room
// index first so that the same join always compares equal.
type edge struct {
	from int
	to   int
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
	dungeon.sight = NewLineOfSight(dungeon.blocksSight)
	dungeon.placeRooms(random)
	dungeon.digCorridors(dungeon.planRoutes(random), random)
	dungeon.placeSpawn()
	dungeon.lockBossRoom()
	dungeon.placeExit()
	dungeon.Illuminate(dungeon.SpawnPoint())

	return dungeon
}

// newEdge pairs two rooms as a join.
//
// Parameters:
//   - a: one of the rooms.
//   - b: the other room.
//
// Returns:
//   - join: the pair, ordered so that from is always the lower index.
func newEdge(a int, b int) (join edge) {
	if a > b {
		a, b = b, a
	}

	return edge{from: a, to: b}
}

// BossRoom reports the room the boss guards, which is also the exit room.
//
// Returns:
//   - room: the extents of the boss room.
func (this *Dungeon) BossRoom() (room Rect) {
	return this.rooms[this.exit]
}

// Dispose releases the pre-rendered level image. The dungeon must not be
// drawn again afterwards.
func (this *Dungeon) Dispose() {
	if this.image == nil {
		return
	}

	this.image.Deallocate()
	this.image = nil
}

// Draw blits the pre-rendered level onto screen, then repaints the exit on top
// of it at this tick's pulse, under a halo that beats with it.
//
// Notes:
//   - the exit is the one tile painted per frame rather than baked, and the one
//     tile the light falloff is not applied to. It is the goal, so it is drawn
//     as something giving off light rather than as something lit.
//   - a sealed exit shimmers too, in its own barred colour and behind its seal,
//     so that the goal is worth spotting from across a room before the boss
//     falls as well as after.
//   - a remembered exit is left as the bake has it. Memory does not shimmer.
//
// Parameters:
//   - screen: the destination image for this frame.
func (this *Dungeon) Draw(screen *ebiten.Image) {
	screen.DrawImage(this.image, nil)

	exit := this.ExitPoint()
	if !this.sight.Visible(exit.X, exit.Y) {
		return
	}

	lit := exitColor
	if !this.ExitOpen() {
		lit = sealedExitColor
	}

	left := float32(exit.X * GridSize)
	top := float32(exit.Y * GridSize)
	level := this.pulseLevel()

	vector.DrawFilledRect(screen, left, top, GridSize-tileSeam, GridSize-tileSeam, blend(lit, level), false)

	if !this.ExitOpen() {
		renderSeal(screen, left, top)
	}

	renderShimmer(screen, left, top, lit, level)
}

// Entrance reports which room the player starts in.
//
// Returns:
//   - room: the index of the entrance room.
func (this *Dungeon) Entrance() (room int) {
	return this.entrance
}

// Exit reports which room holds the exit, which is the boss room.
//
// Returns:
//   - room: the index of the exit room.
func (this *Dungeon) Exit() (room int) {
	return this.exit
}

// ExitOpen reports whether the exit has been unsealed.
//
// Returns:
//   - open: true once OpenExit has been called.
func (this *Dungeon) ExitOpen() (open bool) {
	exit := this.ExitPoint()

	return this.tiles[exit.X][exit.Y] == TileExit
}

// ExitPoint reports the far end of the maze.
//
// Returns:
//   - exitPoint: the centre tile of the room furthest from the entrance.
func (this *Dungeon) ExitPoint() (exitPoint Vector) {
	return this.rooms[this.exit].Center()
}

// Illuminate moves the viewpoint the level is seen from, hiding whatever has
// fallen out of sight and revealing whatever has come into it.
//
// Notes:
//   - the level is re-rendered here rather than every frame, so a viewpoint
//     that has not moved costs nothing.
//
// Parameters:
//   - origin: the tile the level is seen from. Everything goes dark when it
//     lies off the map.
func (this *Dungeon) Illuminate(origin Vector) {
	if this.image != nil && origin == this.sight.Origin() {
		return
	}

	this.sight.LookFrom(origin)
	this.render()
}

// Locked reports whether the boss room is still sealed off.
//
// Returns:
//   - locked: true until Unlock has been called.
func (this *Dungeon) Locked() (locked bool) {
	return len(this.lockedDoors) > 0
}

// LockedDoors reports every tile currently holding a locked door.
//
// Returns:
//   - doors: a copy of the locked door positions, empty once unlocked.
func (this *Dungeon) LockedDoors() (doors []Vector) {
	return append([]Vector(nil), this.lockedDoors...)
}

// OpenExit unseals the exit so that the player can step onto it and win.
func (this *Dungeon) OpenExit() {
	exit := this.ExitPoint()
	this.tiles[exit.X][exit.Y] = TileExit
	this.render()
}

// Room reports the extents of one room.
//
// Parameters:
//   - index: the room index, in the range [0, RoomCount()).
//
// Returns:
//   - room: the room's extents.
func (this *Dungeon) Room(index int) (room Rect) {
	return this.rooms[index]
}

// RoomCount reports how many rooms the level holds.
//
// Returns:
//   - count: the number of rooms, one per sector.
func (this *Dungeon) RoomCount() (count int) {
	return len(this.rooms)
}

// RoomDistances measures how many corridors away every room is from one room.
//
// Parameters:
//   - room: the room to measure from.
//
// Returns:
//   - distances: for every room index, the fewest corridors walked to reach
//     it from room. Every room is reachable, so no entry is negative.
func (this *Dungeon) RoomDistances(room int) (distances []int) {
	return roomDistances(this.links, room)
}

// Sanctuary reports the ground no monster may set foot on: the room the player
// spawns in, grown by one tile so that the ring around it is warded too.
//
// Notes:
//   - the ring is what makes the room genuinely safe rather than merely hard
//     to reach. Every way into the room is a doorway on that ring, so warding
//     it stops a monster camping the threshold and clawing across it.
//
// Returns:
//   - sanctuary: the entrance room and its wall ring.
func (this *Dungeon) Sanctuary() (sanctuary Rect) {
	room := this.rooms[this.entrance]

	return Rect{X: room.X - 1, Y: room.Y - 1, Width: room.Width + 2, Height: room.Height + 2}
}

// SpawnPoint reports where a new player should start.
//
// Returns:
//   - spawnPoint: the tile of the entrance room that placeSpawn chose, which
//     is as far back from the room's doorways as its floor allows.
func (this *Dungeon) SpawnPoint() (spawnPoint Vector) {
	return this.spawn
}

// TileAt reports the contents of the tile at x, y.
//
// Parameters:
//   - x: the tile column.
//   - y: the tile row.
//
// Returns:
//   - tile: the tile, or TileWall for anything off the map.
func (this *Dungeon) TileAt(x int, y int) (tile Tile) {
	if x < 0 || x >= Cols || y < 0 || y >= Rows {
		return TileWall
	}

	return this.tiles[x][y]
}

// Unlock turns every locked door into an ordinary door, opening the boss
// room to the player and letting the boss out after them.
func (this *Dungeon) Unlock() {
	for _, door := range this.lockedDoors {
		this.tiles[door.X][door.Y] = TileDoor
	}

	this.lockedDoors = nil
	this.render()
}

// Update advances the exit beacon by a single tick. Nothing else about a level
// moves, so a dungeon that is never updated simply holds the beacon still.
func (this *Dungeon) Update() {
	this.pulse += 2 * math.Pi * exitPulseRate / float32(ebiten.TPS())
	if this.pulse >= 2*math.Pi {
		this.pulse -= 2 * math.Pi
	}
}

// Visible reports whether the tile at x, y is in sight of the viewpoint right
// now, which is what everything standing on the floor is drawn or held back on.
//
// Parameters:
//   - x: the tile column.
//   - y: the tile row.
//
// Returns:
//   - visible: false for a tile only remembered from an earlier viewpoint, for
//     one never seen at all, and for anything off the map.
func (this *Dungeon) Visible(x int, y int) (visible bool) {
	return this.sight.Visible(x, y)
}

// Walkable reports whether the tile at x, y can be stood on.
//
// Parameters:
//   - x: the tile column.
//   - y: the tile row.
//
// Returns:
//   - walkable: false for walls, locked doors, the sealed exit and anything
//     off the map.
func (this *Dungeon) Walkable(x int, y int) (walkable bool) {
	return this.TileAt(x, y).Walkable()
}

// blocksSight reports whether the tile at x, y stops sight passing through it.
//
// Parameters:
//   - x: the tile column.
//   - y: the tile row.
//
// Returns:
//   - blocksSight: true for walls, which are the only tiles sight cannot cross.
//     A wall is still seen itself; only what lies behind it is hidden.
func (this *Dungeon) blocksSight(x int, y int) (blocksSight bool) {
	return this.TileAt(x, y) == TileWall
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

// digCorridors digs a dogleg for every planned join and nothing else, so the
// corridors reproduce the planned maze exactly.
//
// Parameters:
//   - edges: the joins planRoutes decided on.
//   - random: the source every corridor turn is drawn from.
func (this *Dungeon) digCorridors(edges []edge, random *rand.Rand) {
	for _, join := range edges {
		if join.to == join.from+1 {
			this.digHorizontal(
				this.rooms[join.from],
				this.rooms[join.to],
				(join.from%SectorCols+1)*sectorWidth,
				random)

			continue
		}

		this.digVertical(
			this.rooms[join.from],
			this.rooms[join.to],
			(join.from/SectorCols+1)*sectorHeight,
			random)
	}
}

// digHorizontal joins two side by side rooms with a right, down, right dogleg.
//
// Notes:
//   - the vertical leg always turns on one of the two columns of the sector
//     boundary between the rooms. Room sizing keeps those columns clear of
//     floor, and no other corridor ever runs a leg along them, so two
//     corridors can only ever meet inside a room they both join. That is what
//     keeps the dug level the same maze that planRoutes laid out.
//
// Parameters:
//   - left: the room on the left.
//   - right: the room on the right.
//   - boundary: the first column of the right room's sector.
//   - random: the source the turn column is drawn from.
func (this *Dungeon) digHorizontal(left Rect, right Rect, boundary int, random *rand.Rand) {
	start := left.Center()
	end := right.Center()
	turn := boundary - 1 + random.IntN(2)

	this.carveRow(start.Y, start.X, turn)
	this.carveColumn(turn, start.Y, end.Y)
	this.carveRow(end.Y, turn, end.X)
}

// digVertical joins two stacked rooms with a down, across, down dogleg. The
// horizontal leg turns on one of the two rows of the sector boundary between
// the rooms, for the reasons digHorizontal gives.
//
// Parameters:
//   - top: the upper room.
//   - bottom: the lower room.
//   - boundary: the first row of the lower room's sector.
//   - random: the source the turn row is drawn from.
func (this *Dungeon) digVertical(top Rect, bottom Rect, boundary int, random *rand.Rand) {
	start := top.Center()
	end := bottom.Center()
	turn := boundary - 1 + random.IntN(2)

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

// lightLevel reports how brightly a tile in sight is painted. Light holds at
// full strength for lightFull tiles and then falls off with distance, so a room
// reads as lit from where the player stands rather than as a flat slab.
//
// Notes:
//   - the level never falls below dimLevel, so sight is never dimmer than
//     memory and the falloff never hides anything the player can see. Sight
//     stays unlimited in range; only its brightness is graded.
//
// Parameters:
//   - x: the tile column.
//   - y: the tile row.
//
// Returns:
//   - level: how much of the tile's lit colour to keep, in the range
//     [dimLevel, 1].
func (this *Dungeon) lightLevel(x int, y int) (level float32) {
	origin := this.sight.Origin()
	distance := float32(math.Hypot(float64(x-origin.X), float64(y-origin.Y)))

	if distance <= lightFull {
		return 1
	}

	level = 1 - (distance-lightFull)/(lightRange-lightFull)*(1-dimLevel)

	if level < dimLevel {
		return dimLevel
	}

	return level
}

// lockBossRoom turns every walkable tile on the boss room's wall ring into a
// locked door.
//
// Notes:
//   - the boss room has a single corridor, but that corridor can run along
//     the ring for a stretch when the room sits flush against its sector
//     lane, leaving a strip of door tiles rather than one. Locking the whole
//     ring rather than a single tile seals the room whichever shape the
//     opening took: every tile next to the room interior is on the ring.
func (this *Dungeon) lockBossRoom() {
	for _, doorway := range this.roomDoorways(this.rooms[this.exit]) {
		this.tiles[doorway.X][doorway.Y] = TileLockedDoor
		this.lockedDoors = append(this.lockedDoors, doorway)
	}
}

// placeExit marks the centre of the exit room as the sealed way out of the
// level. Corridors only ever dig through solid tiles, so the room interior is
// still floor when this runs, and the exit is reachable because the room is.
func (this *Dungeon) placeExit() {
	exit := this.ExitPoint()
	this.tiles[exit.X][exit.Y] = TileExitSealed
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

// placeSpawn picks the tile of the entrance room the player starts on: the one
// furthest from the nearest way into the room, so that they begin with the
// length of the room between them and whatever comes down a corridor rather
// than standing in a doorway with a monster already on the threshold.
//
// Notes:
//   - it must run once the corridors are dug, because the doorways it measures
//     from are only cut when a corridor pierces the room's wall ring.
//   - ties go to the tile nearest the middle of the room, so a long room puts
//     the player in the body of it rather than out in a corner.
//   - a room with no doorway at all cannot happen, but would leave every tile
//     tied, which the middle of the room then wins.
func (this *Dungeon) placeSpawn() {
	room := this.rooms[this.entrance]
	doorways := this.roomDoorways(room)
	middle := room.Center()

	this.spawn = middle
	best := -1

	for y := room.Y; y < room.Bottom(); y++ {
		for x := room.X; x < room.Right(); x++ {
			if this.tiles[x][y] != TileFloor {
				continue
			}

			candidate := Vector{X: x, Y: y}
			nearest := Cols + Rows

			for _, doorway := range doorways {
				nearest = min(nearest, candidate.Manhattan(doorway))
			}

			switch {
			case nearest < best:
				continue
			case nearest == best && candidate.Manhattan(middle) >= this.spawn.Manhattan(middle):
				continue
			}

			best, this.spawn = nearest, candidate
		}
	}
}

// planRoutes decides which rooms are joined, and sets the entrance and the
// exit.
//
// Notes:
//   - the spanning tree leaves exactly one route between any two rooms, so
//     every branch off the entrance to exit route is a dead end.
//   - the entrance and the exit are the two ends of the longest route in that
//     tree, and the extra join is drawn only from the joins whose cycle covers
//     part of that route, so it always opens a genuine second way from the
//     entrance to the exit rather than a loop off to one side. Two routes is
//     the most the level ever has.
//   - the extra join is never allowed to touch the exit room, so the exit
//     room keeps the single corridor the tree gave it. That is what lets a
//     lock on its wall ring seal it completely, and what guarantees that
//     every other room, the key's room included, can be reached without
//     passing through it.
//
// Parameters:
//   - random: the source the tree walk and the extra join are drawn from.
//
// Returns:
//   - edges: every pair of rooms a corridor should join.
func (this *Dungeon) planRoutes(random *rand.Rand) (edges []edge) {
	edges = spanningTree(random)
	links := adjacency(edges)

	this.entrance = farthest(links, 0)
	this.exit = farthest(links, this.entrance)

	route := routeEdges(links, this.entrance, this.exit)

	used := make(map[edge]bool, len(edges))
	for _, join := range edges {
		used[join] = true
	}

	var candidates []edge

	for _, candidate := range allEdges() {
		if used[candidate] || candidate.from == this.exit || candidate.to == this.exit {
			continue
		}

		if !crosses(routeEdges(links, candidate.from, candidate.to), route) {
			continue
		}

		candidates = append(candidates, candidate)
	}

	if len(candidates) > 0 {
		edges = append(edges, candidates[random.IntN(len(candidates))])
	}

	this.links = adjacency(edges)

	return edges
}

// pulseLevel reports how bright the exit beacon is on this tick.
//
// Returns:
//   - level: how much of the exit's colour to keep, swinging between
//     exitPulseLow and 1.
func (this *Dungeon) pulseLevel() (level float32) {
	wave := (1 + float32(math.Sin(float64(this.pulse)))) / 2

	return exitPulseLow + (1-exitPulseLow)*wave
}

// render bakes the tiles the viewpoint can see, and the dimmed tiles it
// remembers, into a single image over a dark ground, so that drawing a frame
// costs one blit instead of one filled rectangle per tile. The image is kept
// and repainted rather than replaced, so moving the viewpoint does not leave a
// dead image behind every step. It is run again whenever a door unlocks or the
// exit opens.
//
// Notes:
//   - a dug tile is painted a seam short of its cell so the dark ground shows
//     through as a grid, and a wall with dug ground below it gets a lit lip
//     along that edge.
//   - the keyhole on a locked door and the seal across a closed exit are
//     painted over the tile once it is down, so they ride whatever light level
//     the tile was painted at.
func (this *Dungeon) render() {
	if this.image == nil {
		this.image = ebiten.NewImage(ScreenWidth, ScreenHeight)
	}

	this.image.Fill(darkColor)

	for y := range Rows {
		for x := range Cols {
			var level float32

			switch {
			case this.sight.Visible(x, y):
				level = this.lightLevel(x, y)
			case this.sight.Explored(x, y):
				level = dimLevel
			default:
				continue
			}

			tile := this.tiles[x][y]
			left := float32(x * GridSize)
			top := float32(y * GridSize)
			size := float32(GridSize - tileSeam)

			if tile == TileWall {
				size = GridSize
			}

			vector.DrawFilledRect(
				this.image,
				left,
				top,
				size,
				size,
				blend(tileColor(tile), level),
				false)

			switch tile {
			case TileLockedDoor:
				renderKeyhole(this.image, left, top)
			case TileExitSealed:
				renderSeal(this.image, left, top)
			}

			if tile != TileWall || this.TileAt(x, y+1) == TileWall {
				continue
			}

			vector.DrawFilledRect(
				this.image,
				left,
				float32((y+1)*GridSize-wallEdgeHeight),
				GridSize,
				wallEdgeHeight,
				blend(wallEdgeColor, level),
				false)
		}
	}
}

// roomDoorways lists the ways into a room: every walkable tile on the one tile
// ring around it.
//
// Notes:
//   - a corridor that runs along the ring for a stretch leaves a strip of
//     doorways rather than a single one, so a room can have several tiles on
//     the same side. Every one of them is a way in.
//
// Parameters:
//   - room: the room to walk the ring of.
//
// Returns:
//   - doorways: the walkable ring tiles, in reading order.
func (this *Dungeon) roomDoorways(room Rect) (doorways []Vector) {
	for y := room.Y - 1; y <= room.Bottom(); y++ {
		for x := room.X - 1; x <= room.Right(); x++ {
			tile := Vector{X: x, Y: y}

			if room.Contains(tile) || !this.TileAt(x, y).Walkable() {
				continue
			}

			doorways = append(doorways, tile)
		}
	}

	return doorways
}

// adjacency turns a list of joins into a room by room neighbour lookup.
//
// Parameters:
//   - edges: the joins to index.
//
// Returns:
//   - links: for every room index, the rooms it is joined to.
func adjacency(edges []edge) (links [][]int) {
	links = make([][]int, sectorCount)

	for _, join := range edges {
		links[join.from] = append(links[join.from], join.to)
		links[join.to] = append(links[join.to], join.from)
	}

	return links
}

// allEdges lists every join the sector grid allows, whether used or not.
//
// Returns:
//   - edges: one join for every pair of neighbouring sectors.
func allEdges() (edges []edge) {
	for sectorY := range SectorRows {
		for sectorX := range SectorCols {
			room := sectorY*SectorCols + sectorX

			if sectorX+1 < SectorCols {
				edges = append(edges, newEdge(room, room+1))
			}

			if sectorY+1 < SectorRows {
				edges = append(edges, newEdge(room, room+SectorCols))
			}
		}
	}

	return edges
}

// blend fades a lit colour towards the dark the level sits on. It is what both
// the light falloff and the dimming of memory are painted with, so a tile only
// ever differs from its lit self by how much of the dark it has taken on.
//
// Parameters:
//   - lit: the colour the tile is painted at full brightness.
//   - level: how much of that colour to keep, from 0 for the bare dark through
//     to 1 for the lit colour untouched.
//
// Returns:
//   - blended: the colour to paint.
func blend(lit color.RGBA, level float32) (blended color.RGBA) {
	return color.RGBA{
		R: fade(lit.R, darkColor.R, level),
		G: fade(lit.G, darkColor.G, level),
		B: fade(lit.B, darkColor.B, level),
		A: lit.A,
	}
}

// crosses reports whether two sets of joins have a join in common.
//
// Parameters:
//   - cycle: the joins a candidate extra join would loop through.
//   - route: the joins on the entrance to exit route.
//
// Returns:
//   - crosses: true when the loop covers part of the route.
func crosses(cycle map[edge]bool, route map[edge]bool) (crosses bool) {
	for join := range cycle {
		if route[join] {
			return true
		}
	}

	return false
}

// farthest finds the room the most joins away from a starting room.
//
// Parameters:
//   - links: the neighbour lookup for the tree.
//   - from: the room to measure from.
//
// Returns:
//   - farthest: the index of the most distant room, which in a tree is always
//     a dead end room.
func farthest(links [][]int, from int) (farthest int) {
	distances := roomDistances(links, from)
	farthest = from

	for room, distance := range distances {
		if distance > distances[farthest] {
			farthest = room
		}
	}

	return farthest
}

// renderKeyhole draws the keyhole that marks a locked door tile.
//
// Parameters:
//   - image: the level image to draw on.
//   - left: the tile's left edge, in pixels.
//   - top: the tile's top edge, in pixels.
func renderKeyhole(image *ebiten.Image, left float32, top float32) {
	vector.DrawFilledCircle(image, left+GridSize/2, top+6, 2.5, keyholeColor, true)
	vector.DrawFilledRect(image, left+GridSize/2-1, top+7, 2, 5, keyholeColor, false)
}

// renderSeal draws the cross that marks the exit as still sealed.
//
// Parameters:
//   - image: the level image to draw on.
//   - left: the tile's left edge, in pixels.
//   - top: the tile's top edge, in pixels.
func renderSeal(image *ebiten.Image, left float32, top float32) {
	vector.StrokeLine(image, left+3, top+3, left+GridSize-3, top+GridSize-3, 2, keyholeColor, true)
	vector.StrokeLine(image, left+GridSize-3, top+3, left+3, top+GridSize-3, 2, keyholeColor, true)
}

// renderShimmer lays a soft halo of a thing's own colour over its tile, swelling
// and fading with the shimmer level, so that the thing reads as giving off
// light rather than as merely being lit. It is what marks out the two tiles
// worth crossing a level for: the exit and the key.
//
// Parameters:
//   - screen: the destination image for this frame.
//   - left: the tile's left edge, in pixels.
//   - top: the tile's top edge, in pixels.
//   - lit: the colour of the thing the halo surrounds.
//   - level: the shimmer level this tick, in the range [0, 1].
func renderShimmer(screen *ebiten.Image, left float32, top float32, lit color.RGBA, level float32) {
	vector.DrawFilledCircle(
		screen,
		left+GridSize/2,
		top+GridSize/2,
		shimmerRadius*level,
		tint(lit, shimmerAlpha*level),
		true)
}

// roomDistances measures how many joins away every room is from one room.
//
// Parameters:
//   - links: the neighbour lookup for the level.
//   - from: the room to measure from.
//
// Returns:
//   - distances: for every room index, the fewest joins walked to reach it
//     from the starting room, or -1 if it cannot be reached.
func roomDistances(links [][]int, from int) (distances []int) {
	distances = make([]int, len(links))
	for index := range distances {
		distances[index] = -1
	}

	distances[from] = 0
	queue := []int{from}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, next := range links[current] {
			if distances[next] >= 0 {
				continue
			}

			distances[next] = distances[current] + 1
			queue = append(queue, next)
		}
	}

	return distances
}

// fade mixes one channel of a lit colour into the same channel of the dark.
//
// Parameters:
//   - lit: the channel value of the lit colour.
//   - dark: the channel value of the dark the level sits on.
//   - level: how much of the lit channel to keep, in the range [0, 1].
//
// Returns:
//   - faded: the channel value level of the way from dark to lit, which always
//     lands between the two whichever of them is the brighter.
func fade(lit uint8, dark uint8, level float32) (faded uint8) {
	return uint8(float32(dark) + (float32(lit)-float32(dark))*level)
}

// routeEdges walks the one route a tree holds between two rooms.
//
// Parameters:
//   - links: the neighbour lookup for the tree.
//   - from: the room the route starts at.
//   - to: the room the route ends at.
//
// Returns:
//   - route: every join along the route.
func routeEdges(links [][]int, from int, to int) (route map[edge]bool) {
	parents := make([]int, len(links))
	for index := range parents {
		parents[index] = -1
	}

	parents[from] = from
	queue := []int{from}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, next := range links[current] {
			if parents[next] >= 0 {
				continue
			}

			parents[next] = current
			queue = append(queue, next)
		}
	}

	route = map[edge]bool{}

	for current := to; current != from; current = parents[current] {
		route[newEdge(current, parents[current])] = true
	}

	return route
}

// sectorNeighbors lists the sectors sharing a side with a sector.
//
// Parameters:
//   - room: the index of the sector.
//
// Returns:
//   - neighbors: the indices of its up, left, right and down neighbours, in
//     that order, leaving out any that fall off the grid.
func sectorNeighbors(room int) (neighbors []int) {
	sectorX := room % SectorCols
	sectorY := room / SectorCols

	if sectorY > 0 {
		neighbors = append(neighbors, room-SectorCols)
	}

	if sectorX > 0 {
		neighbors = append(neighbors, room-1)
	}

	if sectorX+1 < SectorCols {
		neighbors = append(neighbors, room+1)
	}

	if sectorY+1 < SectorRows {
		neighbors = append(neighbors, room+SectorCols)
	}

	return neighbors
}

// spanningTree walks the sector grid depth first, taking a random unvisited
// neighbour at every step, which is the recursive backtracker maze algorithm.
// Depth first is what gives the long winding routes and the long dead ends; a
// random join order would give a stubbier, bushier tree.
//
// Parameters:
//   - random: the source the start sector and every step are drawn from.
//
// Returns:
//   - edges: the sectorCount-1 joins of a tree touching every sector.
func spanningTree(random *rand.Rand) (edges []edge) {
	visited := make([]bool, sectorCount)
	start := random.IntN(sectorCount)
	visited[start] = true
	stack := []int{start}

	for len(stack) > 0 {
		current := stack[len(stack)-1]

		var unvisited []int

		for _, next := range sectorNeighbors(current) {
			if visited[next] {
				continue
			}

			unvisited = append(unvisited, next)
		}

		if len(unvisited) == 0 {
			stack = stack[:len(stack)-1]

			continue
		}

		next := unvisited[random.IntN(len(unvisited))]
		visited[next] = true
		edges = append(edges, newEdge(current, next))
		stack = append(stack, next)
	}

	return edges
}

// tileColor reports the colour a tile is painted when it can be seen.
//
// Parameters:
//   - tile: the tile to paint.
//
// Returns:
//   - tileColor: the colour for that tile. An ordinary door is painted as
//     floor, so a threshold reads as part of the room it opens onto rather
//     than as something laid across it, and only the tiles that matter to the
//     player are picked out: the exit, and the locked doors barring the way to
//     it.
func tileColor(tile Tile) (tileColor color.RGBA) {
	switch tile {
	case TileCorridor:
		return corridorColor
	case TileDoor:
		return floorColor
	case TileExit:
		return exitColor
	case TileExitSealed:
		return sealedExitColor
	case TileFloor:
		return floorColor
	case TileLockedDoor:
		return lockedDoorColor
	default:
		return wallColor
	}
}

// tint reduces a colour to a share of its strength, alpha included, giving a
// colour that lies over the level as a wash rather than painting over it.
//
// Notes:
//   - Ebiten works in alpha premultiplied colour, so scaling every channel by
//     the same share is all a translucent version of a colour takes.
//
// Parameters:
//   - lit: the colour to wash with.
//   - strength: how much of it to lay down, in the range [0, 1].
//
// Returns:
//   - tinted: the wash colour.
func tint(lit color.RGBA, strength float32) (tinted color.RGBA) {
	return color.RGBA{
		R: uint8(float32(lit.R) * strength),
		G: uint8(float32(lit.G) * strength),
		B: uint8(float32(lit.B) * strength),
		A: uint8(float32(lit.A) * strength),
	}
}
