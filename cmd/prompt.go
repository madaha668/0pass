package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/madaha668/0pass/internal/vault"
	"golang.org/x/term"
)

// readPassword reads a password from the terminal without echo.
func readPassword(prompt string) ([]byte, error) {
	fmt.Print(prompt)
	pw, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return nil, err
	}
	return pw, nil
}

// readLine reads a line of text from stdin.
func readLine(prompt string) (string, error) {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	return strings.TrimRight(line, "\r\n"), err
}

// selectEntry shows a numbered list and asks the user to pick one.
// Returns the chosen entry or an error.
func selectEntry(entries []*vault.Entry) (*vault.Entry, error) {
	for i, e := range entries {
		fmt.Printf("  [%d] %s (%s)\n", i+1, e.Name, e.URL)
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

// mustLoadVault prompts for master password and loads the vault.
// Returns the vault and the raw password bytes (caller must ZeroBytes when done).
func mustLoadVault() (*vault.Vault, []byte, error) {
	pw, err := readPassword("Master password: ")
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
