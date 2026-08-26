package game

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	// glyphHeight and glyphWidth are the cell size, in pixels, of the fixed
	// width bitmap font that ebitenutil.DebugPrint draws with.
	glyphHeight = 16
	glyphWidth  = 6

	// glyphInset is the one pixel ebitenutil.DebugPrint shifts text right by.
	glyphInset = 1

	// endHintScale and endTitleScale are how many times the font is blown up
	// for each line of an end screen. Integer factors keep the pixels crisp.
	endHintScale  = 2
	endTitleScale = 6

	// endLineGap is the space, in pixels, between the title and the hint.
	endLineGap = 24

	deathTitle = "You Died"
	endHint    = "Press R to play again or Esc to quit"
	winTitle   = "You Win!"
)

var (
	endHintColor = color.RGBA{R: 0xA6, G: 0xAD, B: 0xC8, A: 0xFF}

	// endScreens holds each end screen, baked the first time it is needed, so
	// that showing one costs one blit per frame rather than re-rendering the
	// text each time.
	endScreens = map[string]*ebiten.Image{}
)

// drawEndScreen paints the screen shown once the run is over, one way or the
// other: a title in a colour that says which, with the two controls that still
// work underneath it.
//
// Parameters:
//   - screen: the destination image for this frame.
//   - title: the headline, winTitle or deathTitle.
//   - tint: the colour to draw the headline in.
func drawEndScreen(screen *ebiten.Image, title string, tint color.RGBA) {
	image, baked := endScreens[title]
	if !baked {
		image = renderEndScreen(title, tint)
		endScreens[title] = image
	}

	screen.DrawImage(image, nil)
}

// drawScaledText draws one line of text, horizontally centred on image, blown
// up from the debug bitmap font by an integer factor.
//
// Parameters:
//   - image: the destination image.
//   - line: the text to draw. It must not contain newlines.
//   - top: the row the text starts on.
//   - scale: the integer magnification to apply to the font.
//   - tint: the colour to draw the text in.
func drawScaledText(image *ebiten.Image, line string, top int, scale int, tint color.RGBA) {
	width := textWidth(line)
	buffer := ebiten.NewImage(width, glyphHeight)
	ebitenutil.DebugPrint(buffer, line)

	options := &ebiten.DrawImageOptions{}
	options.GeoM.Scale(float64(scale), float64(scale))
	options.GeoM.Translate(float64((image.Bounds().Dx()-width*scale)/2), float64(top))
	options.ColorScale.ScaleWithColor(tint)

	image.DrawImage(buffer, options)
	buffer.Deallocate()
}

// renderEndScreen bakes an end screen into a single image, with the title and
// hint stacked and centred as a block.
//
// Parameters:
//   - title: the headline.
//   - tint: the colour to draw the headline in.
//
// Returns:
//   - image: the finished screen.
func renderEndScreen(title string, tint color.RGBA) (image *ebiten.Image) {
	image = ebiten.NewImage(ScreenWidth, ScreenHeight)
	image.Fill(wallColor)

	titleHeight := glyphHeight * endTitleScale
	hintHeight := glyphHeight * endHintScale
	top := (ScreenHeight - titleHeight - endLineGap - hintHeight) / 2

	drawScaledText(image, title, top, endTitleScale, tint)
	drawScaledText(image, endHint, top+titleHeight+endLineGap, endHintScale, endHintColor)

	return image
}
