package game

import (
	"encoding/binary"
	"math"
	"testing"
)

func TestRenderNotesLength(t *testing.T) {
	notes := []note{{frequency: 440, seconds: 0.25}, {frequency: 880, seconds: 0.5}}

	pcm := renderNotes(notes, 1)

	want := (SampleRate/4 + SampleRate/2) * channels * bytesPerSample
	if len(pcm) != want {
		t.Fatalf("renderNotes gave %d bytes, want %d", len(pcm), want)
	}
}

func TestRenderNotesIsStereo(t *testing.T) {
	pcm := renderNotes([]note{{frequency: 440, seconds: 0.05}}, 1)

	for offset := 0; offset < len(pcm); offset += channels * bytesPerSample {
		left := sampleAt(pcm, offset)
		right := sampleAt(pcm, offset+bytesPerSample)

		if left != right {
			t.Fatalf("sample at byte %d: left %v and right %v differ", offset, left, right)
		}
	}
}

func TestRenderNotesStaysInRangeAndMakesSound(t *testing.T) {
	gain := float32(0.3)
	pcm := renderNotes(winChime, gain)
	peak := float32(0)

	for offset := 0; offset < len(pcm); offset += bytesPerSample {
		sample := sampleAt(pcm, offset)

		if math.IsNaN(float64(sample)) || sample < -gain || sample > gain {
			t.Fatalf("sample at byte %d is %v, want within [-%v, %v]", offset, sample, gain, gain)
		}

		peak = max(peak, float32(math.Abs(float64(sample))))
	}

	if peak < gain/2 {
		t.Fatalf("peak sample is %v, want a chime that is audibly close to its %v gain", peak, gain)
	}
}

func TestRenderNotesStartsAndEndsSilent(t *testing.T) {
	pcm := renderNotes(winChime, 0.3)

	if first := sampleAt(pcm, 0); first != 0 {
		t.Errorf("first sample is %v, want silence", first)
	}

	if last := sampleAt(pcm, len(pcm)-bytesPerSample); last != 0 {
		t.Errorf("last sample is %v, want silence", last)
	}
}

func TestRenderNotesOfNothing(t *testing.T) {
	if pcm := renderNotes(nil, 1); len(pcm) != 0 {
		t.Fatalf("renderNotes(nil) gave %d bytes, want none", len(pcm))
	}

	if pcm := renderNotes([]note{{frequency: 440, seconds: -1}}, 1); len(pcm) != 0 {
		t.Fatalf("renderNotes of a negative length note gave %d bytes, want none", len(pcm))
	}
}

func TestEnvelope(t *testing.T) {
	for _, test := range []struct {
		progress  float32
		amplitude float32
	}{
		{progress: -1, amplitude: 0},
		{progress: 0, amplitude: 0},
		{progress: envelopeAttack, amplitude: 1},
		{progress: 1, amplitude: 0},
		{progress: 2, amplitude: 0},
	} {
		if amplitude := envelope(test.progress); amplitude != test.amplitude {
			t.Errorf("envelope(%v) = %v, want %v", test.progress, amplitude, test.amplitude)
		}
	}

	for progress := float32(0.01); progress < 1; progress += 0.01 {
		if amplitude := envelope(progress); amplitude <= 0 || amplitude > 1 {
			t.Fatalf("envelope(%v) = %v, want within (0, 1]", progress, amplitude)
		}
	}
}

// sampleAt reads the sample stored at one byte offset into rendered audio.
//
// Parameters:
//   - pcm: the rendered audio to read from.
//   - offset: the byte the sample starts at.
//
// Returns:
//   - sample: the sample read.
func sampleAt(pcm []byte, offset int) (sample float32) {
	return math.Float32frombits(binary.LittleEndian.Uint32(pcm[offset : offset+bytesPerSample]))
}
