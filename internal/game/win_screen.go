package game

import (
	"image/color"
	"unicode/utf8"

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

	// winHintScale and winTitleScale are how many times the font is blown up
	// for each line of the win screen. Integer factors keep the pixels crisp.
	winHintScale  = 2
	winTitleScale = 6

	// winLineGap is the space, in pixels, between the title and the hint.
	winLineGap = 24

	winHint  = "Press Esc to quit"
	winTitle = "You Win!"
)

var (
	winHintColor = color.RGBA{R: 0xA6, G: 0xAD, B: 0xC8, A: 0xFF}

	// winScreen is baked the first time it is needed, so that showing it
	// costs one blit per frame rather than re-rendering the text each time.
	winScreen *ebiten.Image
)

// drawWinScreen paints the screen shown once the player has stepped onto the
// exit: the title in the exit tile's colour, with the one control that still
// works underneath it.
//
// Parameters:
//   - screen: the destination image for this frame.
func drawWinScreen(screen *ebiten.Image) {
	if winScreen == nil {
		winScreen = renderWinScreen()
	}

	screen.DrawImage(winScreen, nil)
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
	width := utf8.RuneCountInString(line)*glyphWidth + glyphInset
	buffer := ebiten.NewImage(width, glyphHeight)
	ebitenutil.DebugPrint(buffer, line)

	options := &ebiten.DrawImageOptions{}
	options.GeoM.Scale(float64(scale), float64(scale))
	options.GeoM.Translate(float64((image.Bounds().Dx()-width*scale)/2), float64(top))
	options.ColorScale.ScaleWithColor(tint)

	image.DrawImage(buffer, options)
	buffer.Deallocate()
}

// renderWinScreen bakes the win screen into a single image, with the title
// and hint stacked and centred as a block.
//
// Returns:
//   - image: the finished screen.
func renderWinScreen() (image *ebiten.Image) {
	image = ebiten.NewImage(ScreenWidth, ScreenHeight)
	image.Fill(wallColor)

	titleHeight := glyphHeight * winTitleScale
	hintHeight := glyphHeight * winHintScale
	top := (ScreenHeight - titleHeight - winLineGap - hintHeight) / 2

	drawScaledText(image, winTitle, top, winTitleScale, exitColor)
	drawScaledText(image, winHint, top+titleHeight+winLineGap, winHintScale, winHintColor)

	return image
}
