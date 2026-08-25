// Package game contains the core game loop implementation driven by Ebitengine.
package game

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	// ScreenWidth is the logical width of the game screen in pixels.
	ScreenWidth = 640

	// ScreenHeight is the logical height of the game screen in pixels.
	ScreenHeight = 480

	GridSize = 16

	UpdateTime = float32(1) / 8
)

var (
	GameObjects []GameObject

	backgroundColor = color.RGBA{R: 0x1E, G: 0x1E, B: 0x2E, A: 0xFF}
	playerColor     = color.RGBA{R: 0x89, G: 0xB4, B: 0xFA, A: 0xFF}
)

// Game holds all mutable game state and implements the ebiten.Game interface.
type Game struct {
}

// New creates a Game with the player centered on screen.
//
// Returns:
//   - game: a ready-to-run Game.
func New() (game *Game) {
	player := &Player{}
	player.SetPosition(Vector{X: (ScreenWidth - GridSize) / 2, Y: (ScreenHeight - GridSize) / 2})
	GameObjects = append(GameObjects, &Player{})

	return &Game{}
}

// Draw renders the current frame onto screen.
//
// Parameters:
//   - screen: the destination image for this frame.
func (this *Game) Draw(screen *ebiten.Image) {
	screen.Fill(backgroundColor)

	for _, gameObject := range GameObjects {
		gameObject.Draw(screen)
	}

	ebitenutil.DebugPrint(
		screen,
		fmt.Sprintf(
			"CodeRunner\nFPS: %.1f  TPS: %.1f\nWASD/arrows to move, space to pause, esc to quit",
			ebiten.ActualFPS(),
			ebiten.ActualTPS()))
}

// Layout reports the logical screen size, independent of the outside window size.
//
// Parameters:
//   - outsideWidth: the window width in device-independent pixels.
//   - outsideHeight: the window height in device-independent pixels.
//
// Returns:
//   - screenWidth: the logical screen width.
//   - screenHeight: the logical screen height.
func (this *Game) Layout(outsideWidth int, outsideHeight int) (screenWidth int, screenHeight int) {
	return ScreenWidth, ScreenHeight
}

// Update advances the game state by a single tick.
//
// Returns:
//   - err: ebiten.Termination when the player quits, otherwise nil.
func (this *Game) Update() (err error) {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return ebiten.Termination
	}

	for _, gameObject := range GameObjects {
		gameObject.Update()
	}

	return nil
}

// Clamp constrains value to the inclusive range [min, max].
//
// Parameters:
//   - value: the value to constrain.
//   - min: the lower bound.
//   - max: the upper bound.
//
// Returns:
//   - result: the constrained value.
func Clamp(value int, min int, max int) (result int) {
	if value < min {
		return min
	}

	if value > max {
		return max
	}

	return value
}
