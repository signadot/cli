package config

import (
	"github.com/spf13/cobra"
)

type LocalDaemon struct {
	*Local

	// Flags
	RunUnpriveleged bool
}

func (c *LocalDaemon) AddFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&c.RunUnpriveleged, "sandbox-manager", false, "run without root priveleges")
}
