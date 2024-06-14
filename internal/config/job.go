package config

import (
	"github.com/spf13/cobra"
	"time"
)

type Job struct {
	*API
}

type JobSubmit struct {
	*Job

	// Flags
	Filename     string
	Attach       string
	Timeout      time.Duration
	TemplateVals TemplateVals
}

func (c *JobSubmit) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&c.Filename, "filename", "f", "", "YAML or JSON file containing the jobs creation request")
	cmd.MarkFlagRequired("filename")
	cmd.Flags().Var(&c.TemplateVals, "set", "--set var=val")

	cmd.Flags().StringVarP(&c.Attach, "attach", "", "", "waits until the job is completed, displaying the selected stream (accepted values: stdout or stderr)")

	cmd.Flags().DurationVar(&c.Timeout, "timeout", 0, "timeout when waiting for the job to be started, if 0 is specified, no timeout will be applied and the command will wait until completion or cancellation of the job (default 0)")

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
