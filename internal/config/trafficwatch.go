package config

import (
	"time"

	"github.com/spf13/cobra"
)

type TrafficWatch struct {
	*Traffic

	// flags
	MetaOnly    bool
	ToDir       string
	Sandbox     string
	WaitTimeout time.Duration
}

func (c *TrafficWatch) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&c.Sandbox, "sandbox", "", "sandbox whose traffic to watch")
	cmd.Flags().BoolVar(&c.MetaOnly, "meta-only", false, "only watch request metadata")
	cmd.Flags().StringVar(&c.ToDir, "dir", "", "ouput to directory")
	cmd.Flags().DurationVar(&c.WaitTimeout, "wait-timeout", 10*time.Second, "time to wait for sandbox readiness")
}
