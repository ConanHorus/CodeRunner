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

const (
	// statePlaying is the run in progress.
	statePlaying gameState = iota

	// stateWon is the run over with the player on the exit.
	stateWon

	// stateDead is the run over with the player out of health.
	stateDead
)

// Game holds all mutable game state and implements the ebiten.Game interface.
type Game struct {
	buffer *ebiten.Image
	seed   uint64
	sounds *Sounds
	source []byte
	state  gameState
	world  *World
}

// gameState is which screen the game is showing.
type gameState uint8

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
	sounds := NewSounds()

	game = &Game{
		buffer: ebiten.NewImage(ScreenWidth, ScreenHeight),
		seed:   seed,
		sounds: sounds,
		source: source,
		world:  NewWorld(seed),
	}

	game.world.SetSounds(sounds)

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
	switch this.state {
	case stateWon:
		drawEndScreen(screen, winTitle, exitColor)

		return
	case stateDead:
		drawEndScreen(screen, deathTitle, healthBarLowColor)

		return
	}

	this.buffer.Clear()
	this.world.Draw(this.buffer)

	offsetX, offsetY := this.world.Shake().Offset()
	options := &ebiten.DrawImageOptions{}
	options.GeoM.Translate(float64(offsetX), float64(offsetY))

	screen.Fill(darkColor)
	screen.DrawImage(this.buffer, options)

	drawHUD(screen, this.world)

	ebitenutil.DebugPrint(
		screen,
		fmt.Sprintf(
			"CodeRunner\nFPS: %.1f  TPS: %.1f\n"+
				"WASD/arrows move, Space attack, Q swap weapon, Esc quit\n"+
				"Find the key, open the boss room, slay the boss, take the exit",
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
//   - once the player has stepped onto the open exit, or run out of health,
//     the run is over and the world freezes: only Escape and R, which starts
//     the same level over, are read after that. The win chime is started on
//     the tick the level is won, and plays on over the win screen.
//   - what the player can see is worked out as the world steps, from wherever
//     the step left the player, so a frame never has to pay for it.
//
// Returns:
//   - err: ebiten.Termination when the player quits, otherwise nil.
func (this *Game) Update() (err error) {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return ebiten.Termination
	}

	if this.state != statePlaying {
		if inpututil.IsKeyJustPressed(ebiten.KeyR) {
			this.restart()
		}

		return nil
	}

	this.world.Update()

	player := this.world.Player()
	position := player.Position()

	switch {
	case player.Health() <= 0:
		this.state = stateDead
	case this.world.Dungeon().TileAt(position.X, position.Y) == TileExit:
		this.state = stateWon

		this.sounds.PlayWin()
	}

	return nil
}

// World reports the level currently being played.
//
// Returns:
//   - world: the live world, for tooling and tests to inspect or steer.
func (this *Game) World() (world *World) {
	return this.world
}

// restart throws the current world away and starts the same level over.
func (this *Game) restart() {
	this.world.Dispose()
	this.world = NewWorld(this.seed)
	this.world.SetSounds(this.sounds)
	this.state = statePlaying
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

// tickSeconds reports how much game time one update covers.
//
// Returns:
//   - seconds: the length of a tick, in seconds.
func tickSeconds() (seconds float32) {
	return 1 / float32(ebiten.TPS())
}
