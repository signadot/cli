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
	Attach       string
	TemplateVals TemplateVals
}

func (c *JobSubmit) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&c.Filename, "filename", "f", "", "YAML or JSON file containing the jobs creation request")
	cmd.MarkFlagRequired("filename")
	cmd.Flags().Var(&c.TemplateVals, "set", "--set var=val")

	cmd.Flags().StringVarP(&c.Attach, "attach", "", "", "Waits until the job runs exits. Accept stdout or stderr as param")

	cmd.Flags().Lookup("attach").NoOptDefVal = "stdout"
}

func (c *JobSubmit) ValidateAttachFlag(cmd *cobra.Command) bool {
	attach := cmd.Flags().Lookup("attach")

	if !attach.Changed {
		return true
	}

	switch attach.Value.String() {
	case "stdout", "stderr":
		return true
	}

	return false
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
	ShowAll bool
}

func (c *JobList) AddFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&c.ShowAll, "all", "", false, "List all jobs")
}
