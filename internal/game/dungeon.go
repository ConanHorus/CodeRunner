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
	// neighbouring sectors. Those two tiles are the lanes the corridors turn
	// on, so no room floor can ever sit on a lane.
	maxRoomHeight = sectorHeight - 2
	maxRoomWidth  = sectorWidth - 2
	minRoomHeight = 4
	minRoomWidth  = 4
	sectorCount   = SectorCols * SectorRows
	sectorHeight  = Rows / SectorRows
	sectorWidth   = Cols / SectorCols
)

var (
	corridorColor   = color.RGBA{R: 0x26, G: 0x26, B: 0x38, A: 0xFF}
	doorColor       = color.RGBA{R: 0xF9, G: 0xE2, B: 0xAF, A: 0xFF}
	exitColor       = color.RGBA{R: 0xA6, G: 0xE3, B: 0xA1, A: 0xFF}
	floorColor      = color.RGBA{R: 0x31, G: 0x32, B: 0x44, A: 0xFF}
	keyholeColor    = color.RGBA{R: 0x11, G: 0x11, B: 0x1B, A: 0xFF}
	lockedDoorColor = color.RGBA{R: 0xF3, G: 0x8B, B: 0xA8, A: 0xFF}
	sealedExitColor = color.RGBA{R: 0x58, G: 0x5B, B: 0x70, A: 0xFF}
	wallColor       = color.RGBA{R: 0x18, G: 0x18, B: 0x25, A: 0xFF}
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
type Dungeon struct {
	entrance    int
	exit        int
	image       *ebiten.Image
	links       [][]int
	lockedDoors []Vector
	rooms       []Rect
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
	dungeon.placeRooms(random)
	dungeon.digCorridors(dungeon.planRoutes(random), random)
	dungeon.lockBossRoom()
	dungeon.placeExit()
	dungeon.render()

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

// Draw blits the pre-rendered level onto screen.
//
// Parameters:
//   - screen: the destination image for this frame.
func (this *Dungeon) Draw(screen *ebiten.Image) {
	screen.DrawImage(this.image, nil)
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

// SpawnPoint reports where a new player should start.
//
// Returns:
//   - spawnPoint: the centre tile of the entrance room, which is one end of
//     the longest route through the maze.
func (this *Dungeon) SpawnPoint() (spawnPoint Vector) {
	return this.rooms[this.entrance].Center()
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
	room := this.rooms[this.exit]

	for y := room.Y - 1; y <= room.Bottom(); y++ {
		for x := room.X - 1; x <= room.Right(); x++ {
			if room.Contains(Vector{X: x, Y: y}) || !this.tiles[x][y].Walkable() {
				continue
			}

			this.tiles[x][y] = TileLockedDoor
			this.lockedDoors = append(this.lockedDoors, Vector{X: x, Y: y})
		}
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

// render bakes the tile grid into a single image so that drawing a frame costs
// one blit instead of one filled rectangle per tile. It is run again whenever
// a door unlocks or the exit opens, reusing the same image.
func (this *Dungeon) render() {
	if this.image == nil {
		this.image = ebiten.NewImage(ScreenWidth, ScreenHeight)
	}

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
			case TileExit:
				tileColor = exitColor
			case TileLockedDoor:
				tileColor = lockedDoorColor
			case TileExitSealed:
				tileColor = sealedExitColor
			default:
				continue
			}

			left := float32(x * GridSize)
			top := float32(y * GridSize)

			vector.DrawFilledRect(this.image, left, top, GridSize, GridSize, tileColor, false)

			switch this.tiles[x][y] {
			case TileLockedDoor:
				renderKeyhole(this.image, left, top)
			case TileExitSealed:
				renderSeal(this.image, left, top)
			}
		}
	}
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
