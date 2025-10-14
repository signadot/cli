package config

import (
	"time"

	"github.com/spf13/cobra"
)

type TrafficWatch struct {
	*Traffic

	// flags
	To           string
	Sandbox      string
	Short        bool
	HeadersOnly  bool
	WaitTimeout  time.Duration
	NoInstrument bool
}

func (c *TrafficWatch) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&c.Sandbox, "sandbox", "", "sandbox whose traffic to watch")
	cmd.Flags().BoolVar(&c.Short, "short", false, "only watch request metadata")
	cmd.Flags().BoolVar(&c.HeadersOnly, "headers-only", false, "do not record request and response bodies")
	cmd.Flags().StringVar(&c.To, "to", "", "output to specified file or directory as needed")
	cmd.Flags().DurationVar(&c.WaitTimeout, "wait-timeout", 30*time.Second, "time to wait for intial sandbox readiness")
	cmd.Flags().BoolVar(&c.NoInstrument, "no-instrument", false, "do not instrument sandbox")
}
