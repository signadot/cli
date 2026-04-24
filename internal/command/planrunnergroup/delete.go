package planrunnergroup

import (
	"errors"
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	planrunnergroups "github.com/signadot/go-sdk/client/plan_runner_groups"
	"github.com/spf13/cobra"
)

func newDelete(prg *config.PlanRunnerGroup) *cobra.Command {
	cfg := &config.PlanRunnerGroupDelete{PlanRunnerGroup: prg}

	cmd := &cobra.Command{
		Use:   "delete { NAME | -f FILENAME [ --set var1=val1 --set var2=val2 ... ] }",
		Short: "Delete plan runner group",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return prgDelete(cfg, cmd.ErrOrStderr(), args)
		},
	}

	cfg.AddFlags(cmd)

	return cmd
}

func prgDelete(cfg *config.PlanRunnerGroupDelete, log io.Writer, args []string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	var name string
	if cfg.Filename == "" {
		if len(args) == 0 {
			return errors.New("must specify filename (-f) or plan runner group name")
		}
		if len(cfg.TemplateVals) != 0 {
			return errors.New("must specify filename (-f) to use --set")
		}
		name = args[0]
	} else {
		if len(args) != 0 {
			return errors.New("must not provide args when filename (-f) specified")
		}
		prg, err := loadPlanRunnerGroup(cfg.Filename, cfg.TemplateVals, true)
		if err != nil {
			return err
		}
		name = prg.Name
	}

	if name == "" {
		return errors.New("plan runner group name is required")
	}

	params := planrunnergroups.NewDeletePlanrunnergroupParams().
		WithOrgName(cfg.Org).
		WithPlanRunnerGroupName(name)
	_, err := cfg.Client.PlanRunnerGroups.DeletePlanrunnergroup(params, nil)
	if err != nil {
		return err
	}

	fmt.Fprintf(log, "Deleted plan runner group %q.\n\n", name)

	return nil
}
