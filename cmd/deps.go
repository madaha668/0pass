package cmd

import (
	"io"
	"os"

	"github.com/atotto/clipboard"
	"github.com/madaha668/0pass/internal/fetch"
	"github.com/madaha668/0pass/internal/vault"
	"golang.org/x/term"
)

// Overridable I/O — replaced in tests to avoid TTY requirements and capture output.
var (
	stdin  io.Reader = os.Stdin
	stdout io.Writer = os.Stdout
	stderr io.Writer = os.Stderr
)

// passwordReader reads a password without echo. Replaced in tests.
var passwordReader = func(prompt string) ([]byte, error) {
	// Write prompt to stdout (not using fmt to avoid import cycle risk)
	_, _ = io.WriteString(stdout, prompt)
	pw, err := term.ReadPassword(int(os.Stdin.Fd())) //nolint:gosec // G115: uintptr→int is safe for fd
	_, _ = io.WriteString(stdout, "\n")
	return pw, err
}

// clipboardWriter writes text to the system clipboard. Replaced in tests.
var clipboardWriter = func(text string) error {
	return clipboard.WriteAll(text)
}

// pageInfoFetcher fetches page metadata from a URL. Replaced in tests.
var pageInfoFetcher = func(url string) (*fetch.PageInfo, error) {
	return fetch.FetchPageInfo(url)
}

// mustLoadVault prompts for master password and loads the vault.
// Returns vault, raw password bytes (caller must ZeroBytes), and any error.
func mustLoadVault() (*vault.Vault, []byte, error) {
	pw, err := passwordReader("Master password: ")
	if err != nil {
		return nil, nil, err
	}
	v, err := vault.Load(pw)
	if err != nil {
		vault.ZeroBytes(pw)
		return nil, nil, err
	}
	return v, pw, nil
}
