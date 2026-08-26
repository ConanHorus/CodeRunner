package game

import (
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	// hudGap is the space, in pixels, between neighbouring elements of the
	// heads up display.
	hudGap = 10

	// hudMessageRise is how far, in pixels, above the health bar the current
	// announcement is drawn.
	hudMessageRise = 24
)

// drawHUD paints the heads up display along the bottom of the screen: the
// health bar, the weapon in hand, the key when it is held, and whatever the
// world last announced.
//
// Notes:
//   - the heads up display is drawn straight onto the screen, after the
//     world, so it stays still while the world shakes.
//
// Parameters:
//   - screen: the destination image for this frame.
//   - world: the world to read the player and the announcement from.
func drawHUD(screen *ebiten.Image, world *World) {
	player := world.Player()
	drawHealthBar(screen, player.Health(), PlayerMaxHealth)

	left := healthBarMargin + healthBarWidth + hudGap
	top := ScreenHeight - healthBarMargin - healthBarHeight

	label := player.Weapon().Name()
	ebitenutil.DebugPrintAt(screen, label, left, top)
	left += textWidth(label) + hudGap

	if player.HasKey() {
		drawKeyGlyph(screen, float32(left), float32(top))
		ebitenutil.DebugPrintAt(screen, "Key", left+GridSize+2, top)
	}

	if message := world.Message(); message != "" {
		ebitenutil.DebugPrintAt(screen, message, (ScreenWidth-textWidth(message))/2, top-hudMessageRise)
	}
}

// textWidth measures a line of text in the debug bitmap font.
//
// Parameters:
//   - line: the text to measure. It must not contain newlines.
//
// Returns:
//   - width: the width, in pixels, the text takes up.
func textWidth(line string) (width int) {
	return utf8.RuneCountInString(line)*glyphWidth + glyphInset
}
