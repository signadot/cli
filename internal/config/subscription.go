package config

import (
	"github.com/spf13/cobra"
)

type Subscription struct {
	*API
}

type SubscriptionStatus struct {
	*Subscription
}

type SubscriptionUsage struct {
	*Subscription

	// Flags
	IDs   []string
	Month string
}

func (c *SubscriptionUsage) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringArrayVar(&c.IDs, "id", nil,
		"only return these quota or limit ids, e.g. --id monthlySandboxes --id numClusters (repeatable)")
	cmd.Flags().StringVar(&c.Month, "month", "",
		"usage for a past UTC calendar month, as YYYY-MM (default: the current month)")
}
