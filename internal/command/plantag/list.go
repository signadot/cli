package plantag

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	plantags "github.com/signadot/go-sdk/client/plan_tags"
	"github.com/spf13/cobra"
)

func newList(tag *config.PlanTag) *cobra.Command {
	cfg := &config.PlanTagList{PlanTag: tag}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List plan tags",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return listTags(cfg, cmd.OutOrStdout())
		},
	}

	return cmd
}

func listTags(cfg *config.PlanTagList, out io.Writer) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	resp, err := cfg.Client.PlanTags.ListPlanTags(
		plantags.NewListPlanTagsParams().WithOrgName(cfg.Org), nil)
	if err != nil {
		return err
	}

	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return printTagTable(out, resp.Payload)
	case config.OutputFormatJSON:
		return print.RawJSON(out, resp.Payload)
	case config.OutputFormatYAML:
		return print.RawYAML(out, resp.Payload)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
