package plantag

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	plantags "github.com/signadot/go-sdk/client/plan_tags"
	"github.com/spf13/cobra"
)

func newDelete(tag *config.PlanTag) *cobra.Command {
	cfg := &config.PlanTagDelete{PlanTag: tag}

	cmd := &cobra.Command{
		Use:   "delete TAG_NAME",
		Short: "Delete a plan tag",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return deleteTag(cfg, cmd.ErrOrStderr(), args[0])
		},
	}

	return cmd
}

func deleteTag(cfg *config.PlanTagDelete, log io.Writer, tagName string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	params := plantags.NewDeletePlanTagParams().
		WithOrgName(cfg.Org).
		WithPlanTagName(tagName)
	_, err := cfg.Client.PlanTags.DeletePlanTag(params, nil)
	if err != nil {
		return err
	}
	fmt.Fprintf(log, "Deleted plan tag %q.\n", tagName)
	return nil
}
