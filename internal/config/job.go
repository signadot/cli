package config

import (
	"github.com/spf13/cobra"
)

type Job struct {
	*API
}

type JobSubmit struct {
	*Job

	// Flags
	Filename     string
	TemplateVals TemplateVals
}

func (c *JobSubmit) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&c.Filename, "filename", "f", "", "YAML or JSON file containing the jobs creation request")
	cmd.MarkFlagRequired("filename")
	cmd.Flags().Var(&c.TemplateVals, "set", "--set var=val")
}

type JobDelete struct {
	*Job
}

type JobGet struct {
	*Job
}

type JobList struct {
	*Job

	// Flags
	ShowCompleted bool
}

func (c *JobList) AddFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&c.ShowCompleted, "show-completed", "", false, "List also jobs with completed status")
}
