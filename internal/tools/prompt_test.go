package tools

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPromptForFileContents(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "real.txt")
	if err := os.WriteFile(existing, []byte("file contents"), 0600); err != nil {
		t.Fatal(err)
	}

	t.Run("returns the contents", func(t *testing.T) {
		out := new(strings.Builder)
		contents, err := promptForFileContents(strings.NewReader(existing+"\n"), out)
		if err != nil {
			t.Fatalf("got error: %v", err)
		}
		if string(contents) != "file contents" {
			t.Errorf("got %q, want %q", contents, "file contents")
		}
		if prompts := strings.Count(out.String(), "Enter a file name"); prompts != 1 {
			t.Errorf("got %d prompts, want 1", prompts)
		}
	})

	t.Run("empty file yields empty contents", func(t *testing.T) {
		empty := filepath.Join(dir, "empty.txt")
		if err := os.WriteFile(empty, nil, 0600); err != nil {
			t.Fatal(err)
		}

		contents, err := promptForFileContents(strings.NewReader(empty+"\n"), new(strings.Builder))
		if err != nil {
			t.Fatalf("got error: %v", err)
		}
		if len(contents) != 0 {
			t.Errorf("got %q, want empty", contents)
		}
	})

	t.Run("reprompts after a missing file", func(t *testing.T) {
		out := new(strings.Builder)
		in := strings.NewReader("nope.txt\n" + dir + "\n" + existing + "\n")

		contents, err := promptForFileContents(in, out)
		if err != nil {
			t.Fatalf("got error: %v", err)
		}
		if string(contents) != "file contents" {
			t.Errorf("got %q, want %q", contents, "file contents")
		}
		if prompts := strings.Count(out.String(), "Enter a file name"); prompts != 3 {
			t.Errorf("got %d prompts, want 3", prompts)
		}
		if !strings.Contains(out.String(), "no such file") {
			t.Errorf("missing open error, got %q", out.String())
		}
		if !strings.Contains(out.String(), "is a directory") {
			t.Errorf("missing directory error, got %q", out.String())
		}
	})

	t.Run("blank line cancels", func(t *testing.T) {
		contents, err := promptForFileContents(strings.NewReader("   \n"), new(strings.Builder))
		if !errors.Is(err, ErrCanceled) {
			t.Errorf("got %v, want ErrCanceled", err)
		}
		if contents != nil {
			t.Errorf("got %q, want nil contents on cancel", contents)
		}
	})

	t.Run("cancels after a failed attempt", func(t *testing.T) {
		out := new(strings.Builder)
		contents, err := promptForFileContents(strings.NewReader("nope.txt\n\n"), out)
		if !errors.Is(err, ErrCanceled) {
			t.Errorf("got %v, want ErrCanceled", err)
		}
		if contents != nil {
			t.Errorf("got %q, want nil contents on cancel", contents)
		}
		if prompts := strings.Count(out.String(), "Enter a file name"); prompts != 2 {
			t.Errorf("got %d prompts, want 2", prompts)
		}
	})

	t.Run("closed input cancels", func(t *testing.T) {
		_, err := promptForFileContents(strings.NewReader(""), new(strings.Builder))
		if !errors.Is(err, ErrCanceled) {
			t.Errorf("got %v, want ErrCanceled", err)
		}
	})

	t.Run("closed input after a failed attempt cancels", func(t *testing.T) {
		_, err := promptForFileContents(strings.NewReader("nope.txt"), new(strings.Builder))
		if !errors.Is(err, ErrCanceled) {
			t.Errorf("got %v, want ErrCanceled", err)
		}
	})

	t.Run("no trailing newline", func(t *testing.T) {
		contents, err := promptForFileContents(strings.NewReader(existing), new(strings.Builder))
		if err != nil {
			t.Fatalf("got error: %v", err)
		}
		if string(contents) != "file contents" {
			t.Errorf("got %q, want %q", contents, "file contents")
		}
	})
}
