package cmd

import (
	"fmt"
	"strings"

	"github.com/madaha668/0pass/internal/vault"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete [query]",
	Short: "Delete a vault entry",
	RunE: func(cmd *cobra.Command, args []string) error {
		var query string
		if len(args) > 0 {
			query = args[0]
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

		fmt.Fprintf(stdout, "Name:     %s\n", entry.Name)
		fmt.Fprintf(stdout, "Username: %s\n", entry.Username)
		fmt.Fprintf(stdout, "URL:      %s\n", entry.URL)

		answer, err := readLine(fmt.Sprintf("Delete '%s'? [y/N]: ", entry.Name))
		if err != nil {
			return err
		}
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Fprintln(stdout, "Aborted.")
			return nil
		}

		newEntries := make([]*vault.Entry, 0, len(v.Entries)-1)
		for _, e := range v.Entries {
			if e.ID != entry.ID {
				newEntries = append(newEntries, e)
			}
		}
		v.Entries = newEntries

		if err := v.Save(pw); err != nil {
			return err
		}

		fmt.Fprintln(stdout, "Deleted.")
		return nil
	},
}
