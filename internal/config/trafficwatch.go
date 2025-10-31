package config

import (
	"time"

	"github.com/spf13/cobra"
)

type TrafficWatch struct {
	*Traffic

	// flags
	OutputDir    string
	OutputFile   string
	Sandbox      string
	Short        bool
	HeadersOnly  bool
	WaitTimeout  time.Duration
	NoInstrument bool
	Clean        bool

	TuiMode bool
}

func (c *TrafficWatch) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&c.Sandbox, "sandbox", "", "sandbox whose traffic to watch")
	cmd.Flags().BoolVar(&c.Short, "short", false, "only watch request metadata")
	cmd.Flags().BoolVar(&c.HeadersOnly, "headers-only", false, "do not record request and response bodies")
	cmd.Flags().StringVar(&c.OutputDir, "out-dir", "", "output to specified directory")
	cmd.Flags().StringVar(&c.OutputFile, "out-file", "", "output to specified file (only with --short)")
	cmd.Flags().DurationVar(&c.WaitTimeout, "wait-timeout", 30*time.Second, "time to wait for intial sandbox readiness")
	cmd.Flags().BoolVar(&c.NoInstrument, "no-instrument", false, "do not instrument sandbox")
	cmd.Flags().MarkHidden("no-instrument")
	cmd.Flags().BoolVar(&c.Clean, "clean", false, "remove old data from output directory first")
	cmd.Flags().BoolVar(&c.TuiMode, "inspect", false, "inspect traffic in TUI mode")
}
