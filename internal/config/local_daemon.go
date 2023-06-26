package config

import (
	_ "github.com/signadot/libconnect/config"
	"github.com/spf13/cobra"
)

type LocalDaemon struct {
	*Local

	// Flags
	RunAsRoot bool
}

func (c *LocalDaemon) AddFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&c.RunAsRoot, "net", true, "enable local networking")
}
