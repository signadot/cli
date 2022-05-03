package main

import (
	"os"

	"github.com/signadot/cli/internal/signadot"
)

func main() {
	rootCmd := signadot.NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
