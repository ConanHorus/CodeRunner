package tools

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// ErrCanceled is returned when the user cancels the prompt with a blank line
// or by closing the input stream.
var ErrCanceled = errors.New("file prompt canceled")

// PromptForFileContents prompts on stdout for a file name and returns the
// contents of that file, reporting any failure and prompting again until it
// succeeds. A blank line or a closed stdin cancels the prompt and returns
// ErrCanceled.
func PromptForFileContents() ([]byte, error) {
	return promptForFileContents(os.Stdin, os.Stdout)
}

func promptForFileContents(in io.Reader, out io.Writer) ([]byte, error) {
	reader := bufio.NewReader(in)

	for {
		if _, err := fmt.Fprint(out, "Enter a file name (blank to cancel): "); err != nil {
			return nil, err
		}

		line, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, err
		}

		name := strings.TrimSpace(line)
		if name == "" {
			return nil, ErrCanceled
		}

		contents, err := readFile(name)
		if err == nil {
			return contents, nil
		}

		if _, err := fmt.Fprintf(out, "%v\n", err); err != nil {
			return nil, err
		}
	}
}

// readFile reads all of name, rejecting anything that is not a file.
func readFile(name string) ([]byte, error) {
	file, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("%q is a directory, not a file", name)
	}

	contents, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("reading %q: %w", name, err)
	}

	return contents, nil
}
