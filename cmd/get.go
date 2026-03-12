package cmd

import (
	"fmt"
	"strings"

	"github.com/madaha668/0pass/internal/vault"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get [query]",
	Short: "Show a password entry",
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
			_, _ = fmt.Fprintln(stdout, "No entries found.")
			return nil
		case 1:
			entry = entries[0]
		default:
			entry, err = selectEntry(entries)
			if err != nil {
				return err
			}
		}

		_, _ = fmt.Fprintf(stdout, "Name:     %s\n", entry.Name)
		_, _ = fmt.Fprintf(stdout, "Username: %s\n", entry.Username)
		_, _ = fmt.Fprintf(stdout, "URL:      %s\n", entry.URL)
		_, _ = fmt.Fprintf(stdout, "Password: %s\n", entry.Password)
		if entry.Notes != "" {
			_, _ = fmt.Fprintf(stdout, "Notes:    %s\n", entry.Notes)
		}
		return nil
	},
}
