package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// osExit is injectable for tests.
var osExit = os.Exit

var rootCmd = &cobra.Command{
	Use:   "0pass",
	Short: "0pass is a secure command-line password vault",
	Long:  `0pass stores your passwords encrypted with AES-256-GCM, protected by an Argon2id-derived master password.`,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		osExit(1)
	}
}

// SetArgs sets the command arguments (used in tests and main_test.go).
func SetArgs(args []string) {
	rootCmd.SetArgs(args)
}

// SetVersion injects the build-time version string into the root command.
func SetVersion(v string) {
	rootCmd.Version = v
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(editCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(passwdCmd)
}
