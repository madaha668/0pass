package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/madaha668/0pass/internal/generator"
	"github.com/madaha668/0pass/internal/vault"
	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:   "edit [query]",
	Short: "Edit an existing vault entry",
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

		// Name
		nameInput, err := readLine(fmt.Sprintf("Name [%s]: ", entry.Name))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		nameInput = strings.TrimSpace(nameInput)
		if nameInput != "" {
			entry.Name = nameInput
		}

		// Username
		usernameInput, err := readLine(fmt.Sprintf("Username [%s]: ", entry.Username))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		usernameInput = strings.TrimSpace(usernameInput)
		if usernameInput != "" {
			entry.Username = usernameInput
		}

		// URL
		urlInput, err := readLine(fmt.Sprintf("URL [%s]: ", entry.URL))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		urlInput = strings.TrimSpace(urlInput)
		if urlInput != "" {
			entry.URL = urlInput
		}

		// Password
		pwInput, err := readPassword("Password [leave empty to keep, g to generate]: ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		pwStr := strings.TrimSpace(string(pwInput))
		vault.ZeroBytes(pwInput)
		if pwStr == "g" {
			opts := generator.DefaultOptions()
			generated, err := generator.Generate(opts)
			if err != nil {
				fmt.Fprintln(os.Stderr, "generating password:", err)
				os.Exit(1)
			}
			entry.Password = generated
			fmt.Printf("Generated password: %s\n", generated)
		} else if pwStr != "" {
			entry.Password = pwStr
		}

		// Notes
		notesInput, err := readLine(fmt.Sprintf("Notes [%s]: ", entry.Notes))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		notesInput = strings.TrimSpace(notesInput)
		if notesInput != "" {
			entry.Notes = notesInput
		}

		entry.UpdatedAt = time.Now()

		if err := v.Save(pw); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		fmt.Println("Entry updated.")
	},
}
