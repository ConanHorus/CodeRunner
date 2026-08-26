package game

// maxSightDistance is how many rows out from the origin a scan walks before it
// gives up. Sight has no range limit here, so the only bound needed is the one
// that guarantees the scan has left the map: a scan row is that many tiles out
// along one axis, so past the longer side of the map no row can still hold an
// on-map tile.
const maxSightDistance = max(Cols, Rows)

// octants are the eight 45 degree wedges a scan is split into, held as the
// mapping from the scan's own axes onto tile space. Sweeping one wedge at a
// time is what lets a single pass work with one quadrant of slopes and treat
// every wedge exactly alike.
var octants = [8]octant{
	{xx: 1, xy: 0, yx: 0, yy: 1},
	{xx: 0, xy: 1, yx: 1, yy: 0},
	{xx: 0, xy: -1, yx: 1, yy: 0},
	{xx: -1, xy: 0, yx: 0, yy: 1},
	{xx: -1, xy: 0, yx: 0, yy: -1},
	{xx: 0, xy: -1, yx: -1, yy: 0},
	{xx: 0, xy: 1, yx: -1, yy: 0},
	{xx: 1, xy: 0, yx: 0, yy: -1},
}

// LineOfSight is the set of tiles visible from a single point, worked out by
// recursive shadowcasting: each octant is swept row by row outwards, and every
// blocking tile met narrows the wedge of slopes still in view, splitting it
// into the parts either side of the shadow the tile casts. Those parts are then
// swept in turn, so a tile is visible exactly when some part of it falls in a
// wedge no shadow has closed over.
//
// Sight is not range limited. A sweep only ever stops when the shadows have
// closed over the whole wedge or when its rows have walked off the map.
//
// Alongside what is visible now, a line of sight keeps every tile it has ever
// seen. That record only ever grows, so moving away from a tile leaves it
// explored rather than unknown.
type LineOfSight struct {
	blocks   func(x int, y int) (blocks bool)
	explored [Cols][Rows]bool
	origin   Vector
	visible  [Cols][Rows]bool
}

// octant is one of the eight wedges of a sweep. The four fields rotate and
// mirror the sweep's own axes onto tile space, so scan can work in one wedge
// and have it land in any of the eight.
type octant struct {
	xx int
	xy int
	yx int
	yy int
}

// NewLineOfSight creates a line of sight that has yet to see anything.
//
// Parameters:
//   - blocks: reports whether the tile at x, y stops sight passing through it.
//     It is asked only about on-map tiles.
//
// Returns:
//   - lineOfSight: a line of sight with nothing visible and nothing explored,
//     ready for LookFrom.
func NewLineOfSight(blocks func(x int, y int) (blocks bool)) (lineOfSight *LineOfSight) {
	return &LineOfSight{blocks: blocks}
}

// Explored reports whether the tile at x, y has ever been seen.
//
// Parameters:
//   - x: the tile column.
//   - y: the tile row.
//
// Returns:
//   - explored: true for a tile seen from this or any earlier origin, false
//     for anything off the map.
func (this *LineOfSight) Explored(x int, y int) (explored bool) {
	if x < 0 || x >= Cols || y < 0 || y >= Rows {
		return false
	}

	return this.explored[x][y]
}

// LookFrom throws out what was visible and works out what can be seen from
// origin instead. What is seen is added to the explored record, which nothing
// ever takes away from.
//
// Parameters:
//   - origin: the tile sight is taken from. Nothing at all is visible when it
//     lies off the map.
func (this *LineOfSight) LookFrom(origin Vector) {
	this.origin = origin
	this.visible = [Cols][Rows]bool{}

	if origin.X < 0 || origin.X >= Cols || origin.Y < 0 || origin.Y >= Rows {
		return
	}

	this.see(origin.X, origin.Y)

	for _, wedge := range octants {
		this.scan(wedge, 1, 1, 0)
	}
}

// Origin reports where sight is being taken from.
//
// Returns:
//   - origin: the tile passed to the last LookFrom, or the zero Vector when
//     LookFrom has yet to be called.
func (this *LineOfSight) Origin() (origin Vector) {
	return this.origin
}

// Visible reports whether the tile at x, y can be seen from the origin.
//
// Parameters:
//   - x: the tile column.
//   - y: the tile row.
//
// Returns:
//   - visible: false for anything off the map.
func (this *LineOfSight) Visible(x int, y int) (visible bool) {
	if x < 0 || x >= Cols || y < 0 || y >= Rows {
		return false
	}

	return this.visible[x][y]
}

// scan sweeps one wedge of one octant outwards, marking what it sees and
// recursing on the wedge left above every shadow it starts.
//
// Notes:
//   - a blocking tile is itself visible. Only what lies behind it is hidden,
//     which is what makes the wall of a lit room show up as its edge.
//
// Parameters:
//   - wedge: the octant to sweep.
//   - row: the first row to sweep, in tiles out from the origin.
//   - start: the slope bounding the side of the wedge the sweep runs from,
//     always the greater of the two bounds.
//   - end: the slope bounding the other side of the wedge.
func (this *LineOfSight) scan(wedge octant, row int, start float32, end float32) {
	if start < end {
		return
	}

	blocked := false

	var newStart float32

	for distance := row; distance <= maxSightDistance && !blocked; distance++ {
		deltaY := -distance

		for deltaX := -distance; deltaX <= 0; deltaX++ {
			x := this.origin.X + deltaX*wedge.xx + deltaY*wedge.xy
			y := this.origin.Y + deltaX*wedge.yx + deltaY*wedge.yy
			leftSlope := (float32(deltaX) - 0.5) / (float32(deltaY) + 0.5)
			rightSlope := (float32(deltaX) + 0.5) / (float32(deltaY) - 0.5)

			if x < 0 || x >= Cols || y < 0 || y >= Rows || start < rightSlope {
				continue
			}

			if end > leftSlope {
				break
			}

			this.see(x, y)

			if blocked {
				if this.blocks(x, y) {
					newStart = rightSlope

					continue
				}

				blocked = false
				start = newStart

				continue
			}

			if !this.blocks(x, y) {
				continue
			}

			blocked = true
			this.scan(wedge, distance+1, start, leftSlope)
			newStart = rightSlope
		}
	}
}

// see marks an on-map tile as visible now and as explored for good.
//
// Parameters:
//   - x: the tile column.
//   - y: the tile row.
func (this *LineOfSight) see(x int, y int) {
	this.visible[x][y] = true
	this.explored[x][y] = true
}
