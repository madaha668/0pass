package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/madaha668/0pass/internal/generator"
	"github.com/madaha668/0pass/internal/vault"
	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:   "edit [query]",
	Short: "Edit an existing vault entry",
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

		// Name
		nameInput, err := readLine(fmt.Sprintf("Name [%s]: ", entry.Name))
		if err != nil {
			return err
		}
		if strings.TrimSpace(nameInput) != "" {
			entry.Name = strings.TrimSpace(nameInput)
		}

		// Username
		usernameInput, err := readLine(fmt.Sprintf("Username [%s]: ", entry.Username))
		if err != nil {
			return err
		}
		if strings.TrimSpace(usernameInput) != "" {
			entry.Username = strings.TrimSpace(usernameInput)
		}

		// URL
		urlInput, err := readLine(fmt.Sprintf("URL [%s]: ", entry.URL))
		if err != nil {
			return err
		}
		if strings.TrimSpace(urlInput) != "" {
			entry.URL = strings.TrimSpace(urlInput)
		}

		// Password
		pwInput, err := passwordReader("Password [leave empty to keep, g to generate]: ")
		if err != nil {
			return err
		}
		pwStr := strings.TrimSpace(string(pwInput))
		vault.ZeroBytes(pwInput)
		switch pwStr {
		case "g":
			opts := generator.DefaultOptions()
			generated, err := generator.Generate(opts)
			if err != nil {
				return fmt.Errorf("generating password: %w", err)
			}
			entry.Password = generated
			_, _ = fmt.Fprintf(stdout, "Generated password: %s\n", generated)
		case "":
			// keep existing
		default:
			entry.Password = pwStr
		}

		// Notes
		notesInput, err := readLine(fmt.Sprintf("Notes [%s]: ", entry.Notes))
		if err != nil {
			return err
		}
		if strings.TrimSpace(notesInput) != "" {
			entry.Notes = strings.TrimSpace(notesInput)
		}

		entry.UpdatedAt = time.Now()

		if err := v.Save(pw); err != nil {
			return err
		}

		_, _ = fmt.Fprintln(stdout, "Entry updated.")
		return nil
	},
}
