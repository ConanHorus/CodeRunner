// Package sounds holds the game's audio assets, embedded into the binary so
// the game still ships as a single file with nothing to install beside it.
//
// Every asset here must be in the format ebitengine's wav decoder accepts:
// linear PCM (format tag 1), 8 or 16 bit little endian, 1 or 2 channels. The
// decoder does no resampling, so assets are stored at the sample rate the
// game's audio context runs at.
package sounds

import _ "embed"

// Cheering is the crowd cheer played when the player reaches the exit.
// Linear PCM, 16 bit little endian, 2 channels, 48000 Hz.
//
//go:embed cheering.wav
var Cheering []byte

// GunShot is the shot heard when the player looses an arrow.
// Linear PCM, 16 bit little endian, 2 channels, 48000 Hz.
//
//go:embed gun-shot.wav
var GunShot []byte
