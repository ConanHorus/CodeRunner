package game

// Vector is a position on the tile grid, or a direction across it.
type Vector struct {
	X int
	Y int
}

// cardinalDirections lists the four unit steps: up, down, left and right.
var cardinalDirections = []Vector{
	{X: 0, Y: -1},
	{X: 0, Y: 1},
	{X: -1, Y: 0},
	{X: 1, Y: 0},
}

// Add sums two vectors.
//
// Parameters:
//   - other: the vector to add.
//
// Returns:
//   - sum: this plus other, component by component.
func (this Vector) Add(other Vector) (sum Vector) {
	return Vector{X: this.X + other.X, Y: this.Y + other.Y}
}

// Manhattan reports how many cardinal steps apart two positions are, ignoring
// walls.
//
// Parameters:
//   - other: the position to measure to.
//
// Returns:
//   - distance: the sum of the horizontal and vertical gaps.
func (this Vector) Manhattan(other Vector) (distance int) {
	return abs(this.X-other.X) + abs(this.Y-other.Y)
}

// Sign reduces each component to -1, 0 or 1, which turns the gap between two
// tiles on the same row or column into a unit direction.
//
// Returns:
//   - unit: the component-wise sign of this.
func (this Vector) Sign() (unit Vector) {
	return Vector{X: sign(this.X), Y: sign(this.Y)}
}

// Sub takes one vector from another.
//
// Parameters:
//   - other: the vector to take away.
//
// Returns:
//   - difference: this minus other, component by component.
func (this Vector) Sub(other Vector) (difference Vector) {
	return Vector{X: this.X - other.X, Y: this.Y - other.Y}
}

// abs reports the magnitude of an integer.
//
// Parameters:
//   - value: the integer to strip the sign from.
//
// Returns:
//   - magnitude: value without its sign.
func abs(value int) (magnitude int) {
	if value < 0 {
		return -value
	}

	return value
}

// sign reports which side of zero an integer lies on.
//
// Parameters:
//   - value: the integer to classify.
//
// Returns:
//   - sign: -1 for negatives, 1 for positives and 0 for zero.
func sign(value int) (sign int) {
	switch {
	case value < 0:
		return -1
	case value > 0:
		return 1
	default:
		return 0
	}
}
