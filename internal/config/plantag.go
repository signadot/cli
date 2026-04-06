package config

import "github.com/spf13/cobra"

type PlanTag struct {
	*Plan
}

type PlanTagApply struct {
	*PlanTag

	// Flags
	PlanID string
}

func (c *PlanTagApply) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&c.PlanID, "plan", "", "plan ID to tag")
	cmd.MarkFlagRequired("plan")
}

type PlanTagGet struct {
	*PlanTag
}

type PlanTagList struct {
	*PlanTag
}

type PlanTagDelete struct {
	*PlanTag
}
