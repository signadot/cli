package subscription

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/signadot/cli/internal/utils"
	"github.com/signadot/go-sdk/models"
)

func printSubscriptionDetails(out io.Writer, sub *models.BillingSubscription) error {
	tw := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)

	fmt.Fprintf(tw, "Plan:\t%s\n", planName(sub.Plan))
	if sub.ManagedBy != "" {
		fmt.Fprintf(tw, "Managed by:\t%s\n", string(sub.ManagedBy))
	}
	if sub.Trial != nil && sub.Trial.EndsAt != "" {
		fmt.Fprintf(tw, "Trial ends:\t%s\n", utils.FormatTimestamp(sub.Trial.EndsAt))
	}
	if sub.CanceledAt != "" {
		fmt.Fprintf(tw, "Canceled:\t%s\n", utils.FormatTimestamp(sub.CanceledAt))
	}
	if sub.CreatedAt != "" {
		fmt.Fprintf(tw, "Created:\t%s\n", utils.FormatTimestamp(sub.CreatedAt))
	}
	if sub.UpdatedAt != "" {
		fmt.Fprintf(tw, "Updated:\t%s\n", utils.FormatTimestamp(sub.UpdatedAt))
	}

	return tw.Flush()
}

// planName renders the plan as "name (vN)", falling back cleanly when the plan
// or its version is absent.
func planName(plan *models.BillingPlan) string {
	if plan == nil || plan.Name == "" {
		return "-"
	}
	if plan.Version != 0 {
		return fmt.Sprintf("%s (v%d)", plan.Name, plan.Version)
	}
	return plan.Name
}
