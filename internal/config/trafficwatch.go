package config

import (
	"github.com/spf13/cobra"
)

type TrafficWatch struct {
	*Traffic

	// flags
	MetaOnly bool
	ToDir    string
	Sandbox  string
}

func (c *TrafficWatch) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&c.Sandbox, "sandbox", "", "sandbox whose traffic to watch")
	cmd.Flags().BoolVar(&c.MetaOnly, "meta-only", false, "only watch request metadata")
	cmd.Flags().StringVar(&c.ToDir, "dir", "", "ouput to directory")
}
