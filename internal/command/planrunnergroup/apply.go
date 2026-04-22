package planrunnergroup

import (
	"errors"
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	planrunnergroups "github.com/signadot/go-sdk/client/plan_runner_groups"
	"github.com/signadot/go-sdk/models"
	"github.com/spf13/cobra"
)

func newApply(prg *config.PlanRunnerGroup) *cobra.Command {
	cfg := &config.PlanRunnerGroupApply{PlanRunnerGroup: prg}

	cmd := &cobra.Command{
		Use:   "apply -f FILENAME [ --set var1=val1 --set var2=val2 ... ]",
		Short: "Create or update a plan runner group",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return apply(cfg, cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}

	cfg.AddFlags(cmd)

	return cmd
}

func apply(cfg *config.PlanRunnerGroupApply, out, log io.Writer) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	if cfg.Filename == "" {
		return errors.New("must specify plan runner group request file with '-f' flag")
	}
	req, err := loadPlanRunnerGroup(cfg.Filename, cfg.TemplateVals, false)
	if err != nil {
		return err
	}

	params := planrunnergroups.NewApplyPlanrunnergroupParams().
		WithOrgName(cfg.Org).
		WithPlanRunnerGroupName(req.Name).
		WithData(req)

	result, err := cfg.Client.PlanRunnerGroups.ApplyPlanrunnergroup(params, nil)
	if err != nil {
		return err
	}

	fmt.Fprintf(log, "Applied plan runner group %q\n\n", req.Name)

	return writeApplyOutput(cfg, out, result.Payload)
}

func writeApplyOutput(cfg *config.PlanRunnerGroupApply, out io.Writer, resp *models.PlanRunnerGroup) error {
	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return nil
	case config.OutputFormatJSON:
		return print.RawJSON(out, resp)
	case config.OutputFormatYAML:
		return print.RawYAML(out, resp)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
