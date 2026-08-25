// Package game contains the core game loop implementation driven by Ebitengine.
package game

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	// GridSize is the width and height of one tile in pixels.
	GridSize = 16

	// ScreenHeight is the logical height of the game screen in pixels.
	ScreenHeight = 480

	// ScreenWidth is the logical width of the game screen in pixels.
	ScreenWidth = 640

	// UpdateTime is how long a held direction waits, in seconds, before the
	// player takes another step. It is the game speed dial.
	UpdateTime = float32(1) / 8

	// initialSeed is the seed of the first level. Regenerating counts up from
	// here, so a given level is reproducible across runs.
	initialSeed = 3
)

var GameObjects []GameObject

// Game holds all mutable game state and implements the ebiten.Game interface.
type Game struct {
	dungeon *Dungeon
	seed    uint64
	source  []byte
}

// New creates a Game holding a freshly generated level.
//
// Parameters:
//   - source: the contents of the source file to run.
//
// Returns:
//   - game: a ready-to-run Game.
func New(source []byte) (game *Game) {
	game = &Game{seed: initialSeed, source: source}
	game.generate()

	return game
}

// Draw renders the current frame onto screen.
//
// Parameters:
//   - screen: the destination image for this frame.
func (this *Game) Draw(screen *ebiten.Image) {
	this.dungeon.Draw(screen)

	for _, gameObject := range GameObjects {
		gameObject.Draw(screen)
	}

	ebitenutil.DebugPrint(
		screen,
		fmt.Sprintf(
			"CodeRunner\nFPS: %.1f  TPS: %.1f\nWASD/arrows to move, R to regenerate, esc to quit",
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

	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		this.seed++
		this.generate()

		return nil
	}

	for _, gameObject := range GameObjects {
		gameObject.Update()
	}

	return nil
}

// generate builds a level from the current seed and repopulates the world
// with the objects that live in it.
func (this *Game) generate() {
	this.dungeon = NewDungeon(this.seed)

	GameObjects = []GameObject{NewPlayer(this.dungeon)}
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
