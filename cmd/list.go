package cmd

import (
	"fmt"

	"github.com/madaha668/0pass/internal/vault"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all vault entries",
	RunE: func(cmd *cobra.Command, args []string) error {
		v, pw, err := mustLoadVault()
		if err != nil {
			return err
		}
		defer vault.ZeroBytes(pw)

		if len(v.Entries) == 0 {
			fmt.Fprintln(stdout, "Vault is empty.")
			return nil
		}

		fmt.Fprintf(stdout, "%-20s %-20s %s\n", "NAME", "USERNAME", "URL")
		fmt.Fprintln(stdout, "────────────────────────────────────────────────────────")
		for _, e := range v.Entries {
			fmt.Fprintf(stdout, "%-20s %-20s %s\n", e.Name, e.Username, e.URL)
		}
		return nil
	},
}
