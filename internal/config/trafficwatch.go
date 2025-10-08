package config

import (
	"github.com/spf13/cobra"
)

type TrafficWatch struct {
	*Traffic

	// flags
	ToDir       string
	Sandbox     string
	Short       bool
	HeadersOnly bool
}

func (c *TrafficWatch) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&c.Sandbox, "sandbox", "", "sandbox whose traffic to watch")
	cmd.Flags().BoolVar(&c.Short, "short", false, "only watch request metadata")
	cmd.Flags().BoolVar(&c.HeadersOnly, "headers-only", false, "do not record request bodies")
	cmd.Flags().StringVar(&c.ToDir, "dir", "", "ouput to directory")
}
