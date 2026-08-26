package game

// Rect is a tile-aligned rectangle covering the tiles
// [X, X+Width) horizontally and [Y, Y+Height) vertically.
type Rect struct {
	X      int
	Y      int
	Width  int
	Height int
}

// Bottom reports the first row past the rectangle.
//
// Returns:
//   - bottom: the row immediately below the rectangle.
func (this Rect) Bottom() (bottom int) {
	return this.Y + this.Height
}

// Center reports the tile at the middle of the rectangle.
//
// Returns:
//   - center: the middle tile, biased down and right on even extents.
func (this Rect) Center() (center Vector) {
	return Vector{X: this.X + this.Width/2, Y: this.Y + this.Height/2}
}

// Contains reports whether a tile lies inside the rectangle.
//
// Parameters:
//   - position: the tile to test.
//
// Returns:
//   - contains: true when the tile is within the rectangle's extents.
func (this Rect) Contains(position Vector) (contains bool) {
	return position.X >= this.X && position.X < this.Right() &&
		position.Y >= this.Y && position.Y < this.Bottom()
}

// Right reports the first column past the rectangle.
//
// Returns:
//   - right: the column immediately right of the rectangle.
func (this Rect) Right() (right int) {
	return this.X + this.Width
}
