package cmd

import (
	"bytes"
	"fmt"
	"os"

	"github.com/madaha668/0pass/internal/vault"
	"github.com/spf13/cobra"
)

var passwdCmd = &cobra.Command{
	Use:   "passwd",
	Short: "Change the master password",
	Run: func(cmd *cobra.Command, args []string) {
		// Prompt for current password and verify by loading vault
		currentPw, err := readPassword("Current master password: ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		defer vault.ZeroBytes(currentPw)

		v, err := vault.Load(currentPw)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		// Prompt for new password twice
		newPw1, err := readPassword("New master password: ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		defer vault.ZeroBytes(newPw1)

		newPw2, err := readPassword("Confirm new master password: ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		defer vault.ZeroBytes(newPw2)

		if !bytes.Equal(newPw1, newPw2) {
			fmt.Fprintln(os.Stderr, "passwords do not match")
			os.Exit(1)
		}

		if err := v.Save(newPw1); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		fmt.Println("Master password updated.")
	},
}
