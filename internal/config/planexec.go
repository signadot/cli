package config

import "github.com/spf13/cobra"

type PlanExecution struct {
	*Plan
}

type PlanExecGet struct {
	*PlanExecution
}

type PlanExecCancel struct {
	*PlanExecution
}

type PlanExecOutputs struct {
	*PlanExecution
}

type PlanExecGetOutput struct {
	*PlanExecution

	// Flags
	All      bool
	Dir      string
	Metadata bool
}

type PlanExecList struct {
	*PlanExecution

	// Flags
	PlanID string
	Tag    string
	Phase  string
}

func (c *PlanExecList) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&c.PlanID, "plan", "", "filter by plan ID")
	cmd.Flags().StringVar(&c.Tag, "tag", "", "filter by plan tag name")
	cmd.Flags().StringVar(&c.Phase, "phase", "", "filter by execution phase")
}

type PlanExecLogs struct {
	*PlanExecution

	// Flags
	Stream    string
	TailLines uint
}

func (c *PlanExecLogs) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&c.Stream, "stream", "s", "stdout", "stream type (stdout or stderr), only used with a step ID")
	cmd.Flags().UintVarP(&c.TailLines, "tail", "t", 0, "number of lines from the end to show (0 = all)")
}
