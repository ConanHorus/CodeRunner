// Package game contains the core game loop implementation driven by Ebitengine.
package game

import (
	"fmt"
	"hash/fnv"

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
)

var GameObjects []GameObject

// Game holds all mutable game state and implements the ebiten.Game interface.
type Game struct {
	dungeon *Dungeon
	player  *Player
	seed    uint64
	shake   *ScreenShake
	source  []byte
	won     bool
	world   *ebiten.Image
}

// New creates a Game holding a freshly generated level.
//
// Parameters:
//   - source: the contents of the source file to run. The first level's seed
//     is derived from these bytes, so a given source always opens onto the
//     same level.
//
// Returns:
//   - game: a ready-to-run Game.
func New(source []byte) (game *Game) {
	seed := seedFromSource(source)
	dungeon := NewDungeon(seed)

	game = &Game{
		dungeon: dungeon,
		player:  NewPlayer(dungeon),
		seed:    seed,
		shake:   NewScreenShake(),
		source:  source,
		world:   ebiten.NewImage(ScreenWidth, ScreenHeight),
	}

	GameObjects = []GameObject{game.player}

	return game
}

// seedFromSource hashes source down to a seed, so that the first level is a
// pure function of the source file's (decoded) bytes.
//
// Parameters:
//   - source: the bytes to derive a seed from.
//
// Returns:
//   - seed: the derived seed.
func seedFromSource(source []byte) (seed uint64) {
	hash := fnv.New64a()
	hash.Write(source)

	return hash.Sum64()
}

// Draw renders the current frame onto screen.
//
// Notes:
//   - the tiles the player can see are drawn lit, fading off with distance,
//     and the ones already walked past are drawn dim. Everything else is dark,
//     including the gutter the screen shake opens up at the edges.
//   - the world is drawn into an offscreen buffer and then blitted across at
//     the screen shake's offset, so the shake moves the level as one piece.
//     The heads up display is drawn straight onto the screen afterwards and so
//     stays still while the world shakes.
//
// Parameters:
//   - screen: the destination image for this frame.
func (this *Game) Draw(screen *ebiten.Image) {
	if this.won {
		drawWinScreen(screen)

		return
	}

	this.drawWorld()

	offsetX, offsetY := this.shake.Offset()
	options := &ebiten.DrawImageOptions{}
	options.GeoM.Translate(float64(offsetX), float64(offsetY))

	screen.Fill(darkColor)
	screen.DrawImage(this.world, options)

	drawHealthBar(screen, this.player.Health(), PlayerMaxHealth)

	ebitenutil.DebugPrint(
		screen,
		fmt.Sprintf(
			"CodeRunner\nFPS: %.1f  TPS: %.1f\nWASD/arrows to move, r to shake, esc to quit\nReach the green exit to win",
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
// Notes:
//   - once the player has stepped onto the exit the level is won and the
//     world freezes: only Escape is read after that.
//   - what the player can see is worked out here, from wherever the step left
//     them, so a frame never has to pay for it.
//
// Returns:
//   - err: ebiten.Termination when the player quits, otherwise nil.
func (this *Game) Update() (err error) {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return ebiten.Termination
	}

	this.shake.Update()

	if this.won {
		return nil
	}

	this.dungeon.Update()

	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		this.shake.Shake(ShakeMagnitude, ShakeDuration)
	}

	for _, gameObject := range GameObjects {
		gameObject.Update()
	}

	position := this.player.Position()
	this.dungeon.Illuminate(position)

	if this.dungeon.TileAt(position.X, position.Y) == TileExit {
		this.won = true
	}

	return nil
}

// drawWorld paints the dungeon and everything living in it into the offscreen
// world buffer, ready to be blitted at the shake offset.
func (this *Game) drawWorld() {
	this.world.Clear()
	this.dungeon.Draw(this.world)

	for _, gameObject := range GameObjects {
		gameObject.Draw(this.world)
	}
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
