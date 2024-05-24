package jobrunnergroup

import (
	"errors"
	"fmt"
	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	runnergroups "github.com/signadot/go-sdk/client/runner_groups"
	"github.com/signadot/go-sdk/models"
	"github.com/spf13/cobra"
	"io"
)

func newApply(jobrunnergroup *config.JobRunnerGroup) *cobra.Command {
	cfg := &config.JobRunnerGroupApply{JobRunnerGroup: jobrunnergroup}

	cmd := &cobra.Command{
		Use:   "apply -f FILENAME [ --set var1=val1 --set var2=val2 ... ]",
		Short: "Create or update a jobrunnergroup with variable expansion",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return apply(cfg, cmd.OutOrStdout(), cmd.ErrOrStderr(), args)
		},
	}

	cfg.AddFlags(cmd)

	return cmd
}

func apply(cfg *config.JobRunnerGroupApply, out, log io.Writer, args []string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	if cfg.Filename == "" {
		return errors.New("must specify jobrunnergroup request file with '-f' flag")
	}
	req, err := loadRunnerGroup(cfg.Filename, cfg.TemplateVals, false /*forDelete */)
	if err != nil {
		return err
	}

	params := runnergroups.NewApplyRunnergroupParams().
		WithOrgName(cfg.Org).
		WithRunnergroupName(req.Name).WithData(req)

	result, err := cfg.Client.RunnerGroups.ApplyRunnergroup(params, nil)
	if err != nil {
		return err
	}
	resp := result.Payload

	fmt.Fprintf(log, "Created runner %q (%q)\n\n", req.Name, cfg.RunnerGroupDashboardUrl(req.Name))

	return writeOutput(cfg, out, resp)
}

func writeOutput(cfg *config.JobRunnerGroupApply, out io.Writer, resp *models.RunnergroupsRunnerGroup) error {
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
