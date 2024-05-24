package jobrunnergroup

import (
	"errors"
	"fmt"
	"github.com/signadot/cli/internal/config"
	runnergroups "github.com/signadot/go-sdk/client/runner_groups"
	"github.com/spf13/cobra"
	"io"
)

func newDelete(jobrunnergroup *config.JobRunnerGroup) *cobra.Command {
	cfg := &config.JobRunnerGroupDelete{JobRunnerGroup: jobrunnergroup}

	cmd := &cobra.Command{
		Use:   "delete { NAME | -f FILENAME [ --set var1=val1 --set var2=val2 ... ] }",
		Short: "Delete jobrunnergroup",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return rgDelete(cfg, cmd.ErrOrStderr(), args)
		},
	}

	cfg.AddFlags(cmd)

	return cmd
}

func rgDelete(cfg *config.JobRunnerGroupDelete, log io.Writer, args []string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	// Get the name either from a file or from the command line.
	var name string
	if cfg.Filename == "" {
		if len(args) == 0 {
			return errors.New("must specify filename (-f) or jobrunnergroup name")
		}
		if len(cfg.TemplateVals) != 0 {
			return errors.New("must specify filename (-f) to use --set")
		}
		name = args[0]
	} else {
		if len(args) != 0 {
			return errors.New("must not provide args when filename (-f) specified")
		}
		rg, err := loadRunnerGroup(cfg.Filename, cfg.TemplateVals, true /* forDelete */)
		if err != nil {
			return err
		}
		name = rg.Name
	}

	if name == "" {
		return errors.New("jobrunnergroup name is required")
	}

	// Delete the jobrunnergroup.
	params := runnergroups.NewDeleteRunnergroupParams().
		WithOrgName(cfg.Org).
		WithRunnergroupName(name)
	_, err := cfg.Client.RunnerGroups.DeleteRunnergroup(params, nil)
	if err != nil {
		return err
	}

	fmt.Fprintf(log, "Deleted jobrunnergroup %q.\n\n", name)

	return nil
}
