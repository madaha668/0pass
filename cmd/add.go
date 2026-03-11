package cmd

import (
	"crypto/rand"
	"fmt"
	"strings"
	"time"

	"github.com/madaha668/0pass/internal/generator"
	"github.com/madaha668/0pass/internal/vault"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new password entry",
	RunE: func(cmd *cobra.Command, args []string) error {
		v, pw, err := mustLoadVault()
		if err != nil {
			return err
		}
		defer vault.ZeroBytes(pw)

		// Name (required)
		var name string
		for {
			name, err = readLine("Name: ")
			if err != nil {
				return err
			}
			name = strings.TrimSpace(name)
			if name != "" {
				break
			}
			fmt.Fprintln(stdout, "Name is required.")
		}

		// Username
		username, err := readLine("Username: ")
		if err != nil {
			return err
		}
		username = strings.TrimSpace(username)

		// URL (required)
		var rawURL string
		for {
			rawURL, err = readLine("URL: ")
			if err != nil {
				return err
			}
			rawURL = strings.TrimSpace(rawURL)
			if rawURL != "" {
				break
			}
			fmt.Fprintln(stdout, "URL is required.")
		}

		// Attempt to fetch page info for notes
		var notes string
		pageInfo, fetchErr := pageInfoFetcher(rawURL)
		if fetchErr == nil && pageInfo != nil && (pageInfo.Title != "" || pageInfo.Description != "") {
			fmt.Fprintf(stdout, "Fetched: %s — %s\n", pageInfo.Title, pageInfo.Description)
			answer, err := readLine("Use as notes? [Y/n]: ")
			if err != nil {
				return err
			}
			answer = strings.TrimSpace(strings.ToLower(answer))
			if answer == "" || answer == "y" || answer == "yes" {
				parts := []string{}
				if pageInfo.Title != "" {
					parts = append(parts, pageInfo.Title)
				}
				if pageInfo.Description != "" {
					parts = append(parts, pageInfo.Description)
				}
				notes = strings.Join(parts, " — ")
			} else {
				notes, err = readLine("Notes: ")
				if err != nil {
					return err
				}
				notes = strings.TrimSpace(notes)
			}
		} else {
			notes, err = readLine("Notes: ")
			if err != nil {
				return err
			}
			notes = strings.TrimSpace(notes)
		}

		// Password (empty = generate)
		pwInput, err := passwordReader("Password (empty to generate): ")
		if err != nil {
			return err
		}

		var entryPassword string
		if len(pwInput) == 0 {
			opts := generator.DefaultOptions()
			generated, err := generator.Generate(opts)
			if err != nil {
				return fmt.Errorf("generating password: %w", err)
			}
			entryPassword = generated
			fmt.Fprintf(stdout, "Generated password: %s\n", entryPassword)
		} else {
			entryPassword = string(pwInput)
			vault.ZeroBytes(pwInput)
		}

		// Generate UUID from 16 random bytes
		var idBytes [16]byte
		if _, err := rand.Read(idBytes[:]); err != nil {
			return fmt.Errorf("generating ID: %w", err)
		}
		id := fmt.Sprintf("%x", idBytes)

		now := time.Now()
		entry := &vault.Entry{
			ID:        id,
			Name:      name,
			Username:  username,
			Password:  entryPassword,
			URL:       rawURL,
			Notes:     notes,
			CreatedAt: now,
			UpdatedAt: now,
		}

		v.Entries = append(v.Entries, entry)

		if err := v.Save(pw); err != nil {
			return err
		}

		fmt.Fprintln(stdout, "Entry added.")
		return nil
	},
}
