package config

import (
	"time"

	"github.com/spf13/cobra"
)

type RouteGroup struct {
	*API
}

type RouteGroupApply struct {
	*RouteGroup

	// Flags
	Filename     string
	Wait         bool
	WaitTimeout  time.Duration
	TemplateVals TemplateVals
}

func (c *RouteGroupApply) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&c.Filename, "filename", "f", "", "YAML or JSON file containing the sandbox creation request")
	cmd.Flags().BoolVar(&c.Wait, "wait", true, "wait for the sandbox status to be Ready before returning")
	cmd.Flags().DurationVar(&c.WaitTimeout, "wait-timeout", 3*time.Minute, "timeout when waiting for the sandbox to be Ready")
	cmd.MarkFlagRequired("filename")
	cmd.Flags().Var(&c.TemplateVals, "set", "--set var=val")
}

type RouteGroupDelete struct {
	*RouteGroup

	// Flags
	Filename     string
	Wait         bool
	WaitTimeout  time.Duration
	TemplateVals TemplateVals
}

func (c *RouteGroupDelete) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&c.Filename, "filename", "f", "", "optional YAML or JSON file containing the original sandbox creation request")
	cmd.Flags().BoolVar(&c.Wait, "wait", true, "wait for the sandbox to finish terminating before returning")
	cmd.Flags().DurationVar(&c.WaitTimeout, "wait-timeout", 5*time.Minute, "timeout when waiting for the sandbox to finish terminating")
	cmd.Flags().Var(&c.TemplateVals, "set", "--set var=val")
}

type RouteGroupGet struct {
	*RouteGroup
}

type RouteGroupList struct {
	*RouteGroup
}
