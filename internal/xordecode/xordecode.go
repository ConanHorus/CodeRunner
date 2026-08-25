// Package xordecode recovers plaintext from a single-byte XOR-obfuscated
// source file.
//
// It tries every possible single-byte key (0-255), scores each candidate on
// how much it looks like real English/code text, and returns the
// highest-scoring decode. This lets the game load a source file without
// knowing in advance whether (or how) it was obfuscated.
package xordecode

import (
	"fmt"
	"sort"
	"strings"
)

// countBytes returns a frequency map of each byte value in data.
func countBytes(data []byte) map[byte]int {
	counts := make(map[byte]int)
	for _, b := range data {
		counts[b]++
	}
	return counts
}

// xorBytes XORs data with a repeating key.
func xorBytes(data []byte, key byte) []byte {
	out := make([]byte, len(data))
	for i, b := range data {
		out[i] = b ^ key
	}
	return out
}

// printableScore scores how much data looks like real English text, not
// just "technically printable" bytes. Rewards letters/spaces heavily,
// tolerates common punctuation, and penalizes control/non-printable bytes
// and rare symbols. Higher is more text-like.
func printableScore(data []byte) float64 {
	if len(data) == 0 {
		return 0.0
	}

	const punctuation = ".,!?'\"-:;\n\t"

	score := 0.0
	for _, b := range data {
		c := rune(b)
		switch {
		case isAlpha(c) || c == ' ':
			score += 1.0
		case strings.ContainsRune(punctuation, c):
			score += 0.3
		case c >= '0' && c <= '9':
			score += 0.2
		case isPrintable(b):
			score += 0.05
		default:
			score -= 1.0 // non-printable / control byte: strong penalty
		}
	}

	return score / float64(len(data))
}

func isAlpha(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isPrintable(b byte) bool {
	// Roughly mirrors Python's string.printable (32-126 plus common
	// whitespace control chars).
	if b >= 32 && b < 127 {
		return true
	}
	switch b {
	case '\t', '\n', '\r', '\v', '\f':
		return true
	}
	return false
}

// normChar renders a byte as a short readable token for the summary.
func normChar(b byte) string {
	switch b {
	case '\n':
		return "\\n"
	case '\t':
		return "\\t"
	case ' ':
		return "_"
	}
	if b >= 32 && b < 127 {
		return string(rune(b))
	}
	return "?"
}

// AutoDecode tries every single-byte XOR key against raw and returns the
// decode that scores best as real text, along with the key and score used.
// If raw is already plaintext, key 0x00 (a no-op XOR) typically wins.
func AutoDecode(raw []byte) (decoded []byte, key byte, score float64) {
	bestScore := -1.0
	var bestKey byte
	var bestDecoded []byte

	for keyVal := 0; keyVal < 256; keyVal++ {
		candidate := xorBytes(raw, byte(keyVal))
		candidateScore := printableScore(candidate)
		if candidateScore > bestScore {
			bestScore = candidateScore
			bestKey = byte(keyVal)
			bestDecoded = candidate
		}
	}

	return bestDecoded, bestKey, bestScore
}

// Summary condenses raw's size, its best-guess XOR key, and its character
// frequency table into a fixed-length, exactly-100-character fingerprint.
func Summary(raw []byte, key byte, score float64) string {
	counts := countBytes(raw)

	type freqItem struct {
		char  byte
		count int
	}
	items := make([]freqItem, 0, len(counts))
	for c, n := range counts {
		items = append(items, freqItem{c, n})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].count != items[j].count {
			return items[i].count > items[j].count
		}
		return items[i].char < items[j].char
	})

	tokens := make([]string, 0, len(items))
	for _, it := range items {
		tokens = append(tokens, fmt.Sprintf("%s%d", normChar(it.char), it.count))
	}
	freqStr := strings.Join(tokens, ",")

	header := fmt.Sprintf("len=%d|xor=0x%02x(%.0f%%)|", len(raw), key, score*100)
	summary := header + freqStr

	// Force to EXACTLY 100 characters: truncate if too long, pad with '.'
	// if too short.
	if len(summary) >= 100 {
		summary = summary[:100]
	} else {
		summary = summary + strings.Repeat(".", 100-len(summary))
	}

	return summary
}
