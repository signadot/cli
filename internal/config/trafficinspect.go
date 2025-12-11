package config

import (
	"github.com/spf13/cobra"
)

type TrafficInspect struct {
	*Traffic

	// flags
	Directory string
	Wait      bool
}

func (c *TrafficInspect) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&c.Directory, "dir", "d", "", "directory containing traffic data to inspect")
	cmd.Flags().BoolVar(&c.Wait, "wait", false, "wait for directory to contain valid traffic data if it's empty")
}
