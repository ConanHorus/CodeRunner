// char_count.go
//
// Reads an input file and (optionally) applies an XOR strategy to help
// understand the file's context:
//
//   - If you supply a key (-key), the file's bytes are XORed with that
//     key (repeating key XOR, like a classic stream/XOR cipher) and the
//     decoded content is printed.
//
//   - If you don't supply a key, the script XORs the file with every
//     possible single-byte key (0-255), picks the one whose decoded
//     output has the highest Shannon entropy, and prints a preview of it.
//
//   - -summary additionally prints a fixed 100-character fingerprint of
//     the file's size, best XOR key, and character frequency table.
//
// Usage:
//
//	go run char_count.go <path_to_file>
//	go run char_count.go <path_to_file> -key 42
//	go run char_count.go <path_to_file> -key "mysecret"
//	go run char_count.go <path_to_file> -summary
package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
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

// parseKey turns a -key argument into key bytes. Accepts a decimal or
// 0x-hex number (e.g. "42" or "0x2a") for a single-byte key, or falls
// back to treating the argument as a literal text key.
func parseKey(keyStr string) []byte {
	if value, err := strconv.ParseInt(keyStr, 0, 64); err == nil {
		if value >= 0 && value <= 255 {
			return []byte{byte(value)}
		}
	}
	return []byte(keyStr)
}

// xorBytes XORs data with a repeating key.
func xorBytes(data, key []byte) []byte {
	out := make([]byte, len(data))
	for i, b := range data {
		out[i] = b ^ key[i%len(key)]
	}
	return out
}

// shannonEntropy computes entropy in bits per byte. Higher = more
// random-looking (closer to 8.0); lower = more structured/predictable
// (plain English text is typically ~4.0-4.5).
func shannonEntropy(data []byte) float64 {
	if len(data) == 0 {
		return 0.0
	}
	counts := countBytes(data)
	length := float64(len(data))
	entropy := 0.0
	for _, count := range counts {
		p := float64(count) / length
		entropy -= p * math.Log2(p)
	}
	return entropy
}

// printableScore scores how much data looks like real English text,
// not just "technically printable" bytes. Rewards letters/spaces
// heavily, tolerates common punctuation, and penalizes control/
// non-printable bytes and rare symbols.
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
	// Roughly mirrors Python's string.printable (32-126 plus
	// common whitespace control chars).
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

// build100CharSummary condenses the file's analysis (size, best-guess
// XOR key, and top character frequencies) into a fixed-length,
// exactly-100-character summary string.
func build100CharSummary(rawBytes []byte, counts map[byte]int, bestKey byte, bestScore float64) string {
	total := len(rawBytes)

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

	header := fmt.Sprintf("len=%d|xor=0x%02x(%.0f%%)|", total, bestKey, bestScore*100)
	summary := header + freqStr

	// Force to EXACTLY 100 characters: truncate if too long,
	// pad with '.' if too short.
	if len(summary) >= 100 {
		summary = summary[:100]
	} else {
		summary = summary + strings.Repeat(".", 100-len(summary))
	}

	return summary
}

func main() {
	keyFlag := flag.String("key", "", "XOR key to decode the file with (text or number). "+
		"If omitted, tries all single-byte keys and shows the highest-entropy candidate.")
	summaryFlag := flag.Bool("summary", false, "Also print a fixed 100-character summary of the "+
		"file's frequency + XOR analysis (a compact fingerprint of its content).")
	flag.Parse()

	if flag.NArg() != 1 {
		fmt.Println("Usage: char_count <path_to_file> [-key KEY] [-summary]")
		os.Exit(1)
	}
	filepath := flag.Arg(0)

	rawBytes, err := os.ReadFile(filepath)
	if err != nil {
		fmt.Printf("Error: file '%s' not found.\n", filepath)
		os.Exit(1)
	}

	// --- XOR strategy ---
	byteCounts := countBytes(rawBytes) // kept for internal use, used by -summary
	var bestKey byte = 0x00
	bestScore := printableScore(rawBytes)

	if *keyFlag != "" {
		key := parseKey(*keyFlag)
		decoded := xorBytes(rawBytes, key)
		fmt.Printf("=== XOR-decoded with key '%s' ===\n", *keyFlag)
		fmt.Println(string(decoded))
	} else {
		bestEntropy := -1.0
		var bestDecoded []byte

		for keyVal := 0; keyVal < 256; keyVal++ {
			decoded := xorBytes(rawBytes, []byte{byte(keyVal)})
			entropy := shannonEntropy(decoded)
			if entropy > bestEntropy {
				bestEntropy = entropy
				bestKey = byte(keyVal)
				bestDecoded = decoded
			}
		}
		bestScore = printableScore(bestDecoded)

		preview := string(bestDecoded)
		if len(preview) > 200 {
			preview = preview[:200]
		}
		preview = strings.ReplaceAll(preview, "\n", "\\n")
		fmt.Printf("Preview: %s\n", preview)
	}

	// --- 100-character summary (only when explicitly requested) ---
	if *summaryFlag {
		summary := build100CharSummary(rawBytes, byteCounts, bestKey, bestScore)
		fmt.Println("\n=== 100-character summary ===")
		fmt.Println(summary)
		fmt.Printf("(length: %d)\n", len(summary))
	}
}
