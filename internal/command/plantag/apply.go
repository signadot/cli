package plantag

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	plantags "github.com/signadot/go-sdk/client/plan_tags"
	"github.com/signadot/go-sdk/models"
	"github.com/spf13/cobra"
)

func newApply(tag *config.PlanTag) *cobra.Command {
	cfg := &config.PlanTagApply{PlanTag: tag}

	cmd := &cobra.Command{
		Use:   "apply TAG_NAME --plan PLAN_ID",
		Short: "Create or move a plan tag",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return applyTag(cfg, cmd.OutOrStdout(), args[0])
		},
	}

	cfg.AddFlags(cmd)
	return cmd
}

// ApplyTag creates or moves a plan tag. Caller must have already called InitAPIConfig.
func ApplyTag(cfg *config.Plan, planID, tagName string) (*models.PlanTag, error) {
	params := plantags.NewPutPlanTagParams().
		WithOrgName(cfg.Org).
		WithPlanTagName(tagName).
		WithData(&models.PlanTagSpec{
			PlanID: planID,
		})
	resp, err := cfg.Client.PlanTags.PutPlanTag(params, nil)
	if err != nil {
		return nil, err
	}
	return resp.Payload, nil
}

func applyTag(cfg *config.PlanTagApply, out io.Writer, tagName string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	tag, err := ApplyTag(cfg.Plan, cfg.PlanID, tagName)
	if err != nil {
		return err
	}

	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return printTagDetails(out, tag)
	case config.OutputFormatJSON:
		return print.RawJSON(out, tag)
	case config.OutputFormatYAML:
		return print.RawYAML(out, tag)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
