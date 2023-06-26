package config

import (
	_ "github.com/signadot/libconnect/config"
	"github.com/spf13/cobra"
)

type Local struct {
	*API
}

type LocalConnect struct {
	*Local

	// Flags
	NonInteractive bool
}

func (c *LocalConnect) AddFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&c.NonInteractive, "non-interactive", false, "run in background")
}

type LocalDisconnect struct {
	*Local
}
