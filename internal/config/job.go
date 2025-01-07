package config

import (
	"time"

	"github.com/spf13/cobra"
)

type Job struct {
	*API
}

type JobSubmit struct {
	*Job

	// Flags
	Filename     string
	Attach       bool
	Timeout      time.Duration
	TemplateVals TemplateVals
	Wait         bool
}

func (c *JobSubmit) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&c.Filename, "filename", "f", "", "YAML or JSON file containing the jobs creation request")
	cmd.MarkFlagRequired("filename")
	cmd.Flags().Var(&c.TemplateVals, "set", "--set var=val")
	cmd.Flags().BoolVar(&c.Attach, "attach", false, "waits until the job is completed, displaying the stdout and stderr streams")
	cmd.Flags().DurationVar(&c.Timeout, "timeout", 0, "timeout when waiting for the job, if 0 is specified, no timeout will be applied and the command will wait until completion or cancellation of the job (default 0)")
	cmd.Flags().BoolVar(&c.Wait, "wait", false, "waits until the job is completed")
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
