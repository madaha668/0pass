package cmd

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/madaha668/0pass/internal/vault"
)

// readLine reads a line of text from stdin, one byte at a time to avoid
// buffering issues when stdin is replaced in tests.
func readLine(prompt string) (string, error) {
	_, _ = fmt.Fprint(stdout, prompt)
	var result []byte
	buf := make([]byte, 1)
	for {
		n, err := stdin.Read(buf)
		if n > 0 {
			if buf[0] == '\n' {
				break
			}
			result = append(result, buf[0])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
	}
	return strings.TrimRight(string(result), "\r"), nil
}

// selectEntry shows a numbered list and prompts the user to pick one.
func selectEntry(entries []*vault.Entry) (*vault.Entry, error) {
	for i, e := range entries {
		_, _ = fmt.Fprintf(stdout, "  [%d] %s (%s)\n", i+1, e.Name, e.URL)
	}
	line, err := readLine("Select entry: ")
	if err != nil {
		return nil, err
	}
	n, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil || n < 1 || n > len(entries) {
		return nil, fmt.Errorf("invalid selection")
	}
	return entries[n-1], nil
}
