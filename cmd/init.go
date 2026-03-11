package cmd

import (
	"bytes"
	"fmt"

	"github.com/madaha668/0pass/internal/vault"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new password vault",
	RunE: func(cmd *cobra.Command, args []string) error {
		pw1, err := passwordReader("Master password: ")
		if err != nil {
			return err
		}
		defer vault.ZeroBytes(pw1)

		pw2, err := passwordReader("Confirm master password: ")
		if err != nil {
			return err
		}
		defer vault.ZeroBytes(pw2)

		if !bytes.Equal(pw1, pw2) {
			return fmt.Errorf("passwords do not match")
		}

		if err := vault.Init(pw1); err != nil {
			return err
		}

		fmt.Fprintf(stdout, "Vault initialized at %s\n", vault.VaultPath())
		return nil
	},
}
