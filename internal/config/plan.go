package config

import "github.com/spf13/cobra"

type Plan struct {
	*API
}

type PlanCompile struct {
	*Plan

	// Flags
	Filename string
	Tag      string
}

func (c *PlanCompile) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&c.Filename, "filename", "f", "", "file containing the prompt to compile")
	cmd.MarkFlagRequired("filename")
	cmd.Flags().StringVar(&c.Tag, "tag", "", "tag the compiled plan with this name")
}

type PlanCreate struct {
	*Plan

	// Flags
	Filename     string
	TemplateVals TemplateVals
}

func (c *PlanCreate) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&c.Filename, "filename", "f", "", "YAML or JSON file containing the plan spec")
	cmd.MarkFlagRequired("filename")
	cmd.Flags().Var(&c.TemplateVals, "set", "--set var=val")
}

type PlanGet struct {
	*Plan
}

type PlanList struct {
	*Plan
}

type PlanDelete struct {
	*Plan
}
