package config

import (
	"github.com/spf13/cobra"
)

type JobRunnerGroup struct {
	*API
}

type JobRunnerGroupApply struct {
	*JobRunnerGroup

	// Flags
	Filename     string
	TemplateVals TemplateVals
}

func (c *JobRunnerGroupApply) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&c.Filename, "filename", "f", "", "YAML or JSON file containing the jobrunnergroup creation request")
	cmd.MarkFlagRequired("filename")
	cmd.Flags().Var(&c.TemplateVals, "set", "--set var=val")
}

type JobRunnerGroupDelete struct {
	*JobRunnerGroup

	// Flags
	Filename     string
	TemplateVals TemplateVals
}

func (c *JobRunnerGroupDelete) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&c.Filename, "filename", "f", "", "optional YAML or JSON file containing the original routegroup creation request")
	cmd.Flags().Var(&c.TemplateVals, "set", "--set var=val")
}

type JobRunnerGroupGet struct {
	*JobRunnerGroup
}

type JobRunnerGroupList struct {
	*JobRunnerGroup
}
