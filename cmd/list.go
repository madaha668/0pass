package cmd

import (
	"fmt"
	"os"

	"github.com/madaha668/0pass/internal/vault"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all vault entries",
	Run: func(cmd *cobra.Command, args []string) {
		v, pw, err := mustLoadVault()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		defer vault.ZeroBytes(pw)

		if len(v.Entries) == 0 {
			fmt.Println("Vault is empty.")
			return
		}

		fmt.Printf("%-20s %-20s %s\n", "NAME", "USERNAME", "URL")
		fmt.Println("────────────────────────────────────────────────────────")
		for _, e := range v.Entries {
			fmt.Printf("%-20s %-20s %s\n", e.Name, e.Username, e.URL)
		}
	},
}
