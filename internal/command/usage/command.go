package usage

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	usageclient "github.com/signadot/go-sdk/client/usage"
	"github.com/spf13/cobra"
)

func New(api *config.API) *cobra.Command {
	cfg := &config.Usage{API: api}

	cmd := &cobra.Command{
		Use:   "usage",
		Short: "Show plan quota and limit usage for the org",
		Long: `Show how much of each plan quota and limit the org has used, the same
numbers the Billing page and enforcement see.

Quotas (monthlySandboxes, monthlyJobs, monthlyTestInvocations) count activity in
the current UTC calendar month and reset at the window end; limits (numClusters,
numResourcePlugins, concurrentDevboxes) count what exists right now.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cfg, cmd.OutOrStdout())
		},
	}

	cfg.AddFlags(cmd)

	return cmd
}

func run(cfg *config.Usage, out io.Writer) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}

	params := usageclient.NewGetUsageParams().WithOrgName(cfg.Org)
	if len(cfg.IDs) > 0 {
		params = params.WithID(cfg.IDs)
	}
	if cfg.Month != "" {
		params = params.WithMonth(&cfg.Month)
	}

	resp, err := cfg.Client.Usage.GetUsage(params, nil)
	if err != nil {
		return err
	}

	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return printUsageTable(out, resp.Payload)
	case config.OutputFormatJSON:
		return print.RawJSON(out, resp.Payload)
	case config.OutputFormatYAML:
		return print.RawYAML(out, resp.Payload)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
