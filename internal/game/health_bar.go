package game

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	// healthBarBorder is how thick, in pixels, the frame around the bar is.
	healthBarBorder = 2

	// healthBarHeight and healthBarWidth are the size, in pixels, of the whole
	// bar, frame included.
	healthBarHeight = 16
	healthBarWidth  = 160

	// healthBarLowFraction is the share of health at or below which the fill
	// turns to healthBarLowColor as a warning.
	healthBarLowFraction = float32(0.3)

	// healthBarMargin is how far, in pixels, the bar sits from the bottom left
	// corner of the screen.
	healthBarMargin = 12
)

var (
	healthBarBorderColor = color.RGBA{R: 0x18, G: 0x18, B: 0x25, A: 0xFF}
	healthBarEmptyColor  = color.RGBA{R: 0x45, G: 0x47, B: 0x5A, A: 0xFF}
	healthBarFillColor   = color.RGBA{R: 0xA6, G: 0xE3, B: 0xA1, A: 0xFF}
	healthBarLowColor    = color.RGBA{R: 0xF3, G: 0x8B, B: 0xA8, A: 0xFF}
)

// drawHealthBar paints the health bar in the bottom left corner of the screen.
//
// Notes:
//   - the bar is heads up display, not world, so it is drawn straight onto the
//     screen and is deliberately left out of the screen shake.
//   - a maxHealth of zero or less draws an empty bar rather than dividing by
//     zero.
//
// Parameters:
//   - screen: the destination image for this frame.
//   - health: the health remaining.
//   - maxHealth: the health the bar is full at.
func drawHealthBar(screen *ebiten.Image, health int, maxHealth int) {
	left := float32(healthBarMargin)
	top := float32(ScreenHeight - healthBarMargin - healthBarHeight)

	vector.DrawFilledRect(screen, left, top, healthBarWidth, healthBarHeight, healthBarBorderColor, false)

	innerLeft := left + healthBarBorder
	innerTop := top + healthBarBorder
	innerWidth := float32(healthBarWidth - healthBarBorder*2)
	innerHeight := float32(healthBarHeight - healthBarBorder*2)

	vector.DrawFilledRect(screen, innerLeft, innerTop, innerWidth, innerHeight, healthBarEmptyColor, false)

	fraction := healthFraction(health, maxHealth)
	if fraction <= 0 {
		return
	}

	fill := healthBarFillColor
	if fraction <= healthBarLowFraction {
		fill = healthBarLowColor
	}

	vector.DrawFilledRect(screen, innerLeft, innerTop, innerWidth*fraction, innerHeight, fill, false)
}

// healthFraction reports how full the bar should be drawn.
//
// Parameters:
//   - health: the health remaining.
//   - maxHealth: the health the bar is full at.
//
// Returns:
//   - fraction: how full the bar is, in the range [0, 1]. It is zero when
//     maxHealth is zero or less.
func healthFraction(health int, maxHealth int) (fraction float32) {
	if maxHealth <= 0 {
		return 0
	}

	return float32(Clamp(health, 0, maxHealth)) / float32(maxHealth)
}
