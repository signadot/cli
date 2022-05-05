package main

import (
	"os"

	"github.com/signadot/cli/internal/command"
)

func main() {
	cmd := command.New()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
