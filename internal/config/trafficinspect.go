package config

import (
	"github.com/spf13/cobra"
)

type TrafficInspect struct {
	*Traffic

	// flags
	Directory string
}

func (c *TrafficInspect) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&c.Directory, "directory", "d", "", "directory containing traffic data to inspect")
	cmd.MarkFlagRequired("directory")
}
