package cmd

import (
	"crypto/rand"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/madaha668/0pass/internal/fetch"
	"github.com/madaha668/0pass/internal/generator"
	"github.com/madaha668/0pass/internal/vault"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new password entry",
	Run: func(cmd *cobra.Command, args []string) {
		v, pw, err := mustLoadVault()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		defer vault.ZeroBytes(pw)

		// Prompt Name (required)
		var name string
		for {
			name, err = readLine("Name: ")
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			name = strings.TrimSpace(name)
			if name != "" {
				break
			}
			fmt.Println("Name is required.")
		}

		// Prompt Username
		username, err := readLine("Username: ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		username = strings.TrimSpace(username)

		// Prompt URL (required)
		var url string
		for {
			url, err = readLine("URL: ")
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			url = strings.TrimSpace(url)
			if url != "" {
				break
			}
			fmt.Println("URL is required.")
		}

		// Attempt to fetch page info
		var notes string
		pageInfo, fetchErr := fetch.FetchPageInfo(url)
		if fetchErr == nil && pageInfo != nil && (pageInfo.Title != "" || pageInfo.Description != "") {
			fmt.Printf("Fetched: %s — %s\n", pageInfo.Title, pageInfo.Description)
			answer, err := readLine("Use as notes? [Y/n]: ")
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
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
					fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}
				notes = strings.TrimSpace(notes)
			}
		} else {
			notes, err = readLine("Notes: ")
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			notes = strings.TrimSpace(notes)
		}

		// Prompt Password
		pwInput, err := readPassword("Password (empty to generate): ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		var entryPassword string
		if len(pwInput) == 0 {
			opts := generator.DefaultOptions()
			generated, err := generator.Generate(opts)
			if err != nil {
				fmt.Fprintln(os.Stderr, "generating password:", err)
				os.Exit(1)
			}
			entryPassword = generated
			fmt.Printf("Generated password: %s\n", entryPassword)
		} else {
			entryPassword = string(pwInput)
			vault.ZeroBytes(pwInput)
		}

		// Generate UUID from 16 random bytes
		var idBytes [16]byte
		if _, err := rand.Read(idBytes[:]); err != nil {
			fmt.Fprintln(os.Stderr, "generating ID:", err)
			os.Exit(1)
		}
		id := fmt.Sprintf("%x", idBytes)

		now := time.Now()
		entry := &vault.Entry{
			ID:        id,
			Name:      name,
			Username:  username,
			Password:  entryPassword,
			URL:       url,
			Notes:     notes,
			CreatedAt: now,
			UpdatedAt: now,
		}

		v.Entries = append(v.Entries, entry)

		if err := v.Save(pw); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		fmt.Println("Entry added.")
	},
}
