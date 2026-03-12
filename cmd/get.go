package cmd

import (
	"fmt"
	"strings"

	"github.com/madaha668/0pass/internal/vault"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get [query]",
	Short: "Copy a password to the clipboard",
	RunE: func(cmd *cobra.Command, args []string) error {
		var query string
		if len(args) > 0 {
			query = args[0]
		} else {
			var err error
			query, err = readLine("Search: ")
			if err != nil {
				return err
			}
			query = strings.TrimSpace(query)
		}

		v, pw, err := mustLoadVault()
		if err != nil {
			return err
		}
		defer vault.ZeroBytes(pw)

		entries := v.FindEntries(query)

		var entry *vault.Entry
		switch len(entries) {
		case 0:
			fmt.Fprintln(stdout, "No entries found.")
			return nil
		case 1:
			entry = entries[0]
		default:
			entry, err = selectEntry(entries)
			if err != nil {
				return err
			}
		}

		if err := clipboardWriter(entry.Password); err != nil {
			// Clipboard unavailable (e.g. headless Linux without a display server).
			// Fall back to printing the password so the app remains usable.
			fmt.Fprintf(stderr, "Warning: clipboard unavailable (%v)\n", err)
			fmt.Fprintf(stdout, "Password: %s\n", entry.Password)
			return nil
		}

		fmt.Fprintf(stdout, "Password for %q copied to clipboard.\n", entry.Name)
		return nil
	},
}
