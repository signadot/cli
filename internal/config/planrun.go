package config

import (
	"time"

	"github.com/spf13/cobra"
)

type PlanRun struct {
	*Plan

	// Flags
	Tag        string
	Cluster    string
	Sandbox    string
	RouteGroup string
	Params     TemplateVals
	Wait       bool
	Attach    bool
	Timeout   time.Duration
	OutputDir string
}

func (c *PlanRun) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&c.Tag, "tag", "", "run the plan referenced by this tag (alternative to plan ID argument)")
	cmd.Flags().StringVar(&c.Cluster, "cluster", "", "target cluster for the execution")
	cmd.Flags().StringVar(&c.Sandbox, "sandbox", "", "run in the context of a sandbox")
	cmd.Flags().StringVar(&c.RouteGroup, "route-group", "", "run in the context of a route group")
	cmd.MarkFlagsMutuallyExclusive("sandbox", "route-group")
	cmd.Flags().Var(&c.Params, "param", "parameter in key=value form (can be repeated)")
	cmd.Flags().BoolVar(&c.Wait, "wait", true, "wait for execution to complete")
	cmd.Flags().BoolVar(&c.Attach, "attach", false, "stream structured events (logs, outputs, result) to stdout")
	cmd.Flags().DurationVar(&c.Timeout, "timeout", 0, "timeout for waiting (0 means no timeout)")
	cmd.Flags().StringVar(&c.OutputDir, "output-dir", "", "directory to export all outputs to on completion")
}
