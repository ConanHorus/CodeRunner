package game

import (
	"bytes"
	"encoding/binary"
	"math"

	assets "github.com/ConanHorus/CodeRunner/sounds"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

const (
	// SampleRate is how many samples a second every sound in the game is
	// generated and played at.
	SampleRate = 48000

	// bytesPerSample is the width, in bytes, of one 32 bit float sample.
	bytesPerSample = 4

	// channels is how many channels the generated audio carries. Ebitengine
	// only plays stereo, so every sample is written twice.
	channels = 2

	// envelopeAttack is the fraction of a note spent rising to full volume.
	// The rest of the note falls back to silence, so notes neither click on
	// nor cut off.
	envelopeAttack = float32(0.02)

	// winChimeGain is how loud the win chime's notes peak, as a fraction of
	// full scale.
	winChimeGain = float32(0.3)
)

// winChime is the arpeggio played when the player steps onto the exit: a C
// major triad climbing to the octave above, with the top note held.
var winChime = []note{
	{frequency: 523.25, seconds: 0.09},
	{frequency: 659.25, seconds: 0.09},
	{frequency: 783.99, seconds: 0.09},
	{frequency: 1046.50, seconds: 0.34},
}

// note is a single tone in a generated sound.
type note struct {
	// frequency is the pitch of the tone, in hertz.
	frequency float32

	// seconds is how long the tone sounds for.
	seconds float32
}

// Sounds holds the game's sound effects, prepared once up front and ready to
// play. Nothing is read off disk: effects are either synthesised or embedded
// into the binary, so the game stays a single file with no assets beside it.
type Sounds struct {
	shot *audio.Player
	win  *audio.Player
}

// NewSounds generates the game's sound effects.
//
// Returns:
//   - sounds: ready-to-play sounds.
func NewSounds() (sounds *Sounds) {
	return &Sounds{
		shot: wavPlayer(assets.GunShot),
		win:  wavPlayer(assets.Cheering),
	}
}

// wavPlayer prepares one of the embedded wav assets for playing.
//
// Notes:
//   - an asset that will not decode leaves the player nil and that one effect
//     silent, rather than taking the game down over a sound effect. play skips
//     a nil player.
//   - the decoder does no resampling, so an asset recorded at anything other
//     than SampleRate would play at the wrong pitch. The sounds package keeps
//     every asset at SampleRate, and sound_test.go holds it to that.
//
// Parameters:
//   - asset: the embedded wav bytes to decode.
//
// Returns:
//   - player: the sound, ready to play, or nil if it could not be prepared.
func wavPlayer(asset []byte) (player *audio.Player) {
	stream, err := wav.DecodeF32(bytes.NewReader(asset))
	if err != nil {
		return nil
	}

	player, err = audioContext().NewPlayerF32(stream)
	if err != nil {
		return nil
	}

	return player
}

// PlayShot plays the shot the player's bow looses, from the top, cutting off
// any shot still sounding so that quick fire keeps up with the bow.
//
// Notes:
//   - a nil Sounds is silent, so a world built without sound, as the tests
//     build it, shoots quietly rather than panicking.
func (this *Sounds) PlayShot() {
	if this == nil {
		return
	}

	play(this.shot)
}

// PlayWin plays the win cheer from the top, cutting off any copy of it still
// sounding.
//
// Notes:
//   - a nil Sounds is silent, matching PlayShot.
func (this *Sounds) PlayWin() {
	if this == nil {
		return
	}

	play(this.win)
}

// play starts a sound over from the top, cutting off any copy of it still
// sounding.
//
// Notes:
//   - a sound that would not prepare, or that will not rewind, is skipped
//     rather than played from wherever it happens to sit. Losing an effect is
//     not worth taking the game down for.
//
// Parameters:
//   - player: the sound to play, which may be nil.
func play(player *audio.Player) {
	if player == nil {
		return
	}

	player.Pause()

	if err := player.Rewind(); err != nil {
		return
	}

	player.Play()
}

// audioContext reports the process wide audio context, creating it on first
// use.
//
// Notes:
//   - audio.NewContext panics once a context exists, so an existing one is
//     handed back instead. Every context in this game is made here, and so is
//     made at SampleRate.
//
// Returns:
//   - context: the audio context every player is created from.
func audioContext() (context *audio.Context) {
	if context = audio.CurrentContext(); context != nil {
		return context
	}

	return audio.NewContext(SampleRate)
}

// envelope reports how loud a note is at one point along its length, shaping
// it so that it rises out of and falls back into silence.
//
// Parameters:
//   - progress: how far through the note this point is, in the range [0, 1].
//
// Returns:
//   - amplitude: the fraction of the note's peak volume to sound, in the
//     range [0, 1].
func envelope(progress float32) (amplitude float32) {
	if progress <= 0 || progress >= 1 {
		return 0
	}

	if progress < envelopeAttack {
		return progress / envelopeAttack
	}

	return (1 - progress) / (1 - envelopeAttack)
}

// renderNotes synthesises notes into audio Ebitengine can play: linear PCM,
// 32 bit little endian floats, two channels, at SampleRate.
//
// Notes:
//   - every note starts and ends at silence, so they can be laid end to end
//     without a click at the seams.
//
// Parameters:
//   - notes: the tones to sound, one after another.
//   - gain: how loud the notes peak, as a fraction of full scale.
//
// Returns:
//   - pcm: the rendered audio.
func renderNotes(notes []note, gain float32) (pcm []byte) {
	pcm = make([]byte, 0, sampleCount(notes)*channels*bytesPerSample)
	frame := make([]byte, bytesPerSample)

	for _, current := range notes {
		samples := noteSampleCount(current)

		for sample := range samples {
			seconds := float64(sample) / float64(SampleRate)
			amplitude := gain * envelope(noteProgress(sample, samples))
			value := amplitude * float32(math.Sin(2*math.Pi*float64(current.frequency)*seconds))

			binary.LittleEndian.PutUint32(frame, math.Float32bits(value))

			for range channels {
				pcm = append(pcm, frame...)
			}
		}
	}

	return pcm
}

// noteProgress reports how far through a note one of its samples sits, with
// the first sample at the very start of the note and the last at the very
// end, so that the note's envelope opens and closes on silence.
//
// Parameters:
//   - sample: the index of the sample within the note.
//   - samples: how many samples the whole note is rendered into.
//
// Returns:
//   - progress: the position within the note, in the range [0, 1].
func noteProgress(sample int, samples int) (progress float32) {
	if samples < 2 {
		return 0
	}

	return float32(sample) / float32(samples-1)
}

// noteSampleCount reports how many samples one note is rendered into.
//
// Parameters:
//   - current: the note to measure.
//
// Returns:
//   - samples: the sample count, never less than zero.
func noteSampleCount(current note) (samples int) {
	return max(int(current.seconds*SampleRate), 0)
}

// sampleCount reports how many samples a run of notes is rendered into.
//
// Parameters:
//   - notes: the notes to measure.
//
// Returns:
//   - samples: the total sample count.
func sampleCount(notes []note) (samples int) {
	for _, current := range notes {
		samples += noteSampleCount(current)
	}

	return samples
}
