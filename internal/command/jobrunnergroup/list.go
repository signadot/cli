package jobrunnergroup

import (
	"fmt"
	"io"

	runnergroups "github.com/signadot/go-sdk/client/runner_groups"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	"github.com/spf13/cobra"
)

func newList(jobrunnergroup *config.JobRunnerGroup) *cobra.Command {
	cfg := &config.JobRunnerGroupList{JobRunnerGroup: jobrunnergroup}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List runnergroups",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return list(cfg, cmd.OutOrStdout())
		},
	}

	return cmd
}

func list(cfg *config.JobRunnerGroupList, out io.Writer) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	resp, err := cfg.Client.RunnerGroups.ListRunnergroup(runnergroups.NewListRunnergroupParams().WithOrgName(cfg.Org), nil)
	if err != nil {
		return err
	}

	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return printRunnerGroupTable(cfg, out, resp.Payload)
	case config.OutputFormatJSON:
		return print.RawJSON(out, resp.Payload)
	case config.OutputFormatYAML:
		return print.RawYAML(out, resp.Payload)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
