package main

import (
	"io"
	"os"
	"testing"

	"github.com/madaha668/0pass/cmd"
)

// TestMain_Execute calls main() with --help to cover the main() statement
// without triggering os.Exit.
func TestMain_Execute(t *testing.T) {
	// Redirect stdout so help output doesn't pollute test output.
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd.SetArgs([]string{"--help"})
	// main() → cmd.Execute() → rootCmd.Execute() with --help → returns nil (no os.Exit)
	main()

	_ = w.Close()
	os.Stdout = old
	_, _ = io.Copy(io.Discard, r)
	_ = r.Close()
}
