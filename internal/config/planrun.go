package config

import (
	"time"

	"github.com/spf13/cobra"
)

type PlanRun struct {
	*Plan

	// Flags
	Tag string
	// PlanTags are tag-name globs selecting the plans to run, taking the place
	// of the repository's configured list. WithTags narrows that selection and
	// WithoutTags removes from it.
	PlanTags    []string
	WithTags    []string
	WithoutTags []string
	Cluster     string
	Sandbox     string
	RouteGroup  string
	Params      TemplateVals
	Secrets     TemplateVals
	Wait        bool
	Attach      bool
	Timeout     time.Duration
	OutputDir   string
}

func (c *PlanRun) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&c.Tag, "tag", "", "run the plan referenced by this tag (alternative to plan ID argument)")
	cmd.Flags().StringArrayVar(&c.PlanTags, "tags", nil, "run every plan whose tag matches this glob (repeatable; replaces the repository's configured plans)")
	cmd.Flags().StringArrayVar(&c.WithTags, "with-tag", nil, "only run plans carrying a tag matching this glob (repeatable, and all must match)")
	cmd.Flags().StringArrayVar(&c.WithoutTags, "without-tag", nil, "skip plans carrying a tag matching this glob (repeatable)")
	cmd.MarkFlagsMutuallyExclusive("tag", "tags")
	cmd.Flags().StringVar(&c.Cluster, "cluster", "", "target cluster for the execution")
	cmd.Flags().StringVar(&c.Sandbox, "sandbox", "", "run in the context of a sandbox")
	cmd.Flags().StringVar(&c.RouteGroup, "route-group", "", "run in the context of a route group")
	cmd.MarkFlagsMutuallyExclusive("sandbox", "route-group")
	cmd.Flags().Var(&c.Params, "param", "parameter in key=value form (can be repeated)")
	cmd.Flags().Var(&c.Secrets, "param-secret", "bind a plan param to an org secret: param-name=secret-name (can be repeated)")
	cmd.Flags().BoolVar(&c.Wait, "wait", true, "wait for execution to complete")
	cmd.Flags().BoolVar(&c.Attach, "attach", false, "stream structured events (logs, outputs, result) to stdout")
	cmd.Flags().DurationVar(&c.Timeout, "timeout", 0, "timeout for waiting (0 means no timeout)")
	cmd.Flags().StringVar(&c.OutputDir, "output-dir", "", "directory to export all outputs to on completion")
}
