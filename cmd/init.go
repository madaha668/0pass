package cmd

import (
	"bytes"
	"fmt"
	"os"

	"github.com/madaha668/0pass/internal/vault"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new password vault",
	Run: func(cmd *cobra.Command, args []string) {
		pw1, err := readPassword("Master password: ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		defer vault.ZeroBytes(pw1)

		pw2, err := readPassword("Confirm master password: ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		defer vault.ZeroBytes(pw2)

		if !bytes.Equal(pw1, pw2) {
			fmt.Fprintln(os.Stderr, "passwords do not match")
			os.Exit(1)
		}

		if err := vault.Init(pw1); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		fmt.Printf("Vault initialized at %s\n", vault.VaultPath())
	},
}
