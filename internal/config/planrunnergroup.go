package config

import (
	"github.com/spf13/cobra"
)

type PlanRunnerGroup struct {
	*API
}

type PlanRunnerGroupApply struct {
	*PlanRunnerGroup

	// Flags
	Filename     string
	TemplateVals TemplateVals
}

func (c *PlanRunnerGroupApply) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&c.Filename, "filename", "f", "", "YAML or JSON file containing the plan runner group definition")
	cmd.MarkFlagRequired("filename")
	cmd.Flags().Var(&c.TemplateVals, "set", "--set var=val")
}

type PlanRunnerGroupDelete struct {
	*PlanRunnerGroup

	// Flags
	Filename     string
	TemplateVals TemplateVals
}

func (c *PlanRunnerGroupDelete) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&c.Filename, "filename", "f", "", "optional YAML or JSON file containing the plan runner group definition")
	cmd.Flags().Var(&c.TemplateVals, "set", "--set var=val")
}

type PlanRunnerGroupGet struct {
	*PlanRunnerGroup
}

type PlanRunnerGroupList struct {
	*PlanRunnerGroup
}
