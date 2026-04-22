package plantag

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	plantags "github.com/signadot/go-sdk/client/plan_tags"
	"github.com/spf13/cobra"
)

func newGet(tag *config.PlanTag) *cobra.Command {
	cfg := &config.PlanTagGet{PlanTag: tag}

	cmd := &cobra.Command{
		Use:   "get TAG_NAME",
		Short: "Get a plan tag",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return getTag(cfg, cmd.OutOrStdout(), args[0])
		},
	}

	return cmd
}

func getTag(cfg *config.PlanTagGet, out io.Writer, tagName string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	params := plantags.NewGetPlanTagParams().
		WithOrgName(cfg.Org).
		WithPlanTagName(tagName)
	resp, err := cfg.Client.PlanTags.GetPlanTag(params, nil)
	if err != nil {
		return err
	}

	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return printTagDetails(out, resp.Payload)
	case config.OutputFormatJSON:
		return print.RawJSON(out, resp.Payload)
	case config.OutputFormatYAML:
		return print.RawYAML(out, resp.Payload)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
