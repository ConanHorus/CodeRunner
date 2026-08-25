package game

import (
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	// ShakeDuration is how long, in seconds, the shake started by Game on the
	// R key runs for.
	ShakeDuration = float32(0.35)

	// ShakeMagnitude is how far, in pixels, that same shake throws the world
	// from centre on its first tick.
	ShakeMagnitude = float32(8)
)

// ScreenShake is a decaying random offset used to punctuate an impact. It
// carries a single trauma value that Shake sets and Update bleeds back down to
// zero, and the offset is scaled by the square of that trauma, so a shake hits
// hard and then tails off quickly instead of stopping dead.
//
// Nothing inside ScreenShake touches the screen. It only reports how far the
// world should be drawn off centre this frame, and the caller applies it.
type ScreenShake struct {
	magnitude  float32
	offsetX    float32
	offsetY    float32
	trauma     float32
	traumaLoss float32
}

// NewScreenShake creates an idle ScreenShake.
//
// Returns:
//   - screenShake: a ready-to-run ScreenShake, reporting no offset until Shake
//     is called.
func NewScreenShake() (screenShake *ScreenShake) {
	return &ScreenShake{}
}

// Offset reports how far the world should be drawn off centre this frame.
//
// Returns:
//   - offsetX: the horizontal offset, in pixels.
//   - offsetY: the vertical offset, in pixels.
func (this *ScreenShake) Offset() (offsetX float32, offsetY float32) {
	return this.offsetX, this.offsetY
}

// Shake starts a shake, replacing any shake already running.
//
// Notes:
//   - a duration of zero or less is ignored, since it would ask for a shake
//     that bleeds out infinitely fast.
//
// Parameters:
//   - magnitude: how far, in pixels, to throw the world on the first tick.
//   - duration: how long, in seconds, the shake takes to bleed out.
func (this *ScreenShake) Shake(magnitude float32, duration float32) {
	if duration <= 0 {
		return
	}

	this.magnitude = magnitude
	this.trauma = 1
	this.traumaLoss = 1 / duration
}

// Update advances the shake by a single tick and picks this frame's offset.
func (this *ScreenShake) Update() {
	if this.trauma <= 0 {
		this.offsetX = 0
		this.offsetY = 0

		return
	}

	this.trauma -= this.traumaLoss / float32(ebiten.TPS())
	if this.trauma <= 0 {
		this.trauma = 0
		this.offsetX = 0
		this.offsetY = 0

		return
	}

	throw := this.magnitude * this.trauma * this.trauma
	this.offsetX = throw * randomSwing()
	this.offsetY = throw * randomSwing()
}

// randomSwing picks how far, and which way, one axis swings on this tick.
//
// Returns:
//   - swing: a value in the range [-1, 1).
func randomSwing() (swing float32) {
	return rand.Float32()*2 - 1
}
