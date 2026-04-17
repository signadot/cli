package planrunnergroup

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	planrunnergroups "github.com/signadot/go-sdk/client/plan_runner_groups"
	"github.com/spf13/cobra"
)

func newImageDelete(prg *config.PlanRunnerGroup) *cobra.Command {
	cfg := &config.PlanRunnerGroupGet{PlanRunnerGroup: prg}

	cmd := &cobra.Command{
		Use:   "delete PRG_NAME REF_OR_DIGEST",
		Short: "Delete an image from a plan runner group",
		Long:  "Accepts a digest (sha256:...) or an image reference (alpine:3.19).",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return imageDelete(cfg, cmd.OutOrStdout(), args[0], args[1])
		},
	}
	return cmd
}

func imageDelete(cfg *config.PlanRunnerGroupGet, out io.Writer, prgName, refOrDigest string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	params := planrunnergroups.NewDeletePrgImageParams().
		WithOrgName(cfg.Org).
		WithPlanRunnerGroupName(prgName).
		WithImageDigest(refOrDigest)
	_, err := cfg.Client.PlanRunnerGroups.DeletePrgImage(params, nil)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "Deleted %s from %s\n", refOrDigest, prgName)
	return nil
}
