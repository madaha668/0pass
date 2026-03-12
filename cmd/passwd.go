package cmd

import (
	"bytes"
	"fmt"

	"github.com/madaha668/0pass/internal/vault"
	"github.com/spf13/cobra"
)

var passwdCmd = &cobra.Command{
	Use:   "passwd",
	Short: "Change the master password",
	RunE: func(cmd *cobra.Command, args []string) error {
		currentPw, err := passwordReader("Current master password: ")
		if err != nil {
			return err
		}
		defer vault.ZeroBytes(currentPw)

		v, err := vault.Load(currentPw)
		if err != nil {
			return err
		}

		newPw1, err := passwordReader("New master password: ")
		if err != nil {
			return err
		}
		defer vault.ZeroBytes(newPw1)

		newPw2, err := passwordReader("Confirm new master password: ")
		if err != nil {
			return err
		}
		defer vault.ZeroBytes(newPw2)

		if !bytes.Equal(newPw1, newPw2) {
			return fmt.Errorf("passwords do not match")
		}

		if err := v.Save(newPw1); err != nil {
			return err
		}

		_, _ = fmt.Fprintln(stdout, "Master password updated.")
		return nil
	},
}
