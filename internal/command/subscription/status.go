package subscription

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	subscriptionclient "github.com/signadot/go-sdk/client/subscription"
	"github.com/spf13/cobra"
)

func newStatus(sub *config.Subscription) *cobra.Command {
	cfg := &config.SubscriptionStatus{Subscription: sub}

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show the org's subscription plan, trial, and management status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(cfg, cmd.OutOrStdout())
		},
	}

	return cmd
}

func runStatus(cfg *config.SubscriptionStatus, out io.Writer) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}

	params := subscriptionclient.NewGetSubscriptionParams().WithOrgName(cfg.Org)

	resp, err := cfg.Client.Subscription.GetSubscription(params, nil)
	if err != nil {
		return err
	}

	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return printSubscriptionDetails(out, resp.Payload)
	case config.OutputFormatJSON:
		return print.RawJSON(out, resp.Payload)
	case config.OutputFormatYAML:
		return print.RawYAML(out, resp.Payload)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
