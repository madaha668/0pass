package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/madaha668/0pass/internal/vault"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get [query]",
	Short: "Copy a password to the clipboard",
	Run: func(cmd *cobra.Command, args []string) {
		var query string
		if len(args) > 0 {
			query = args[0]
		} else {
			var err error
			query, err = readLine("Search: ")
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			query = strings.TrimSpace(query)
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

		if err := clipboard.WriteAll(entry.Password); err != nil {
			fmt.Fprintln(os.Stderr, "writing to clipboard:", err)
			os.Exit(1)
		}

		fmt.Printf("Password for %q copied to clipboard.\n", entry.Name)
	},
}
