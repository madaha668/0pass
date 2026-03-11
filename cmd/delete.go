package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/madaha668/0pass/internal/vault"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete [query]",
	Short: "Delete a vault entry",
	Run: func(cmd *cobra.Command, args []string) {
		var query string
		if len(args) > 0 {
			query = args[0]
		}

		v, pw, err := mustLoadVault()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		defer vault.ZeroBytes(pw)

		entries := v.FindEntries(query)

		var entry *vault.Entry
		switch len(entries) {
		case 0:
			fmt.Println("No entries found.")
			return
		case 1:
			entry = entries[0]
		default:
			entry, err = selectEntry(entries)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}

		fmt.Printf("Name:     %s\n", entry.Name)
		fmt.Printf("Username: %s\n", entry.Username)
		fmt.Printf("URL:      %s\n", entry.URL)

		answer, err := readLine(fmt.Sprintf("Delete '%s'? [y/N]: ", entry.Name))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Println("Aborted.")
			return
		}

		// Remove entry from slice
		newEntries := make([]*vault.Entry, 0, len(v.Entries)-1)
		for _, e := range v.Entries {
			if e.ID != entry.ID {
				newEntries = append(newEntries, e)
			}
		}
		v.Entries = newEntries

		if err := v.Save(pw); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		fmt.Println("Deleted.")
	},
}
