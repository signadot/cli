package jobrunnergroup

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	runnergroups "github.com/signadot/go-sdk/client/runner_groups"
	"github.com/spf13/cobra"
)

func newGet(jobrunnergroup *config.JobRunnerGroup) *cobra.Command {
	cfg := &config.JobRunnerGroupGet{JobRunnerGroup: jobrunnergroup}

	cmd := &cobra.Command{
		Use:   "get NAME",
		Short: "Get jobrunnergroup",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return get(cfg, cmd.OutOrStdout(), args[0])
		},
	}

	return cmd
}

func get(cfg *config.JobRunnerGroupGet, out io.Writer, name string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	params := runnergroups.NewGetRunnergroupParams().WithOrgName(cfg.Org).WithRunnergroupName(name)
	resp, err := cfg.Client.RunnerGroups.GetRunnergroup(params, nil)
	if err != nil {
		return err
	}

	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return printRunnerGroupDetails(cfg.JobRunnerGroup, out, resp.Payload)
	case config.OutputFormatJSON:
		return print.RawJSON(out, resp.Payload)
	case config.OutputFormatYAML:
		return print.RawYAML(out, resp.Payload)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
