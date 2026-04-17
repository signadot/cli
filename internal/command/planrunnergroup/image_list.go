package planrunnergroup

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	"github.com/signadot/cli/internal/sdtab"
	planrunnergroups "github.com/signadot/go-sdk/client/plan_runner_groups"
	"github.com/spf13/cobra"
)

func newImageList(prg *config.PlanRunnerGroup) *cobra.Command {
	cfg := &config.PlanRunnerGroupList{PlanRunnerGroup: prg}

	cmd := &cobra.Command{
		Use:   "list PRG_NAME",
		Short: "List images on a plan runner group",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return imageList(cfg, cmd.OutOrStdout(), args[0])
		},
	}
	return cmd
}

func imageList(cfg *config.PlanRunnerGroupList, out io.Writer, prgName string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	params := planrunnergroups.NewListPrgImagesParams().
		WithOrgName(cfg.Org).
		WithPlanRunnerGroupName(prgName)
	resp, err := cfg.Client.PlanRunnerGroups.ListPrgImages(params, nil)
	if err != nil {
		return err
	}

	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return printImageTable(out, resp)
	case config.OutputFormatJSON:
		return print.RawJSON(out, resp.Payload)
	case config.OutputFormatYAML:
		return print.RawYAML(out, resp.Payload)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}

type imageRow struct {
	Ref    string `sdtab:"REF"`
	Digest string `sdtab:"DIGEST"`
}

func printImageTable(out io.Writer, resp *planrunnergroups.ListPrgImagesOK) error {
	t := sdtab.New[imageRow](out)
	t.AddHeader()
	if resp.Payload != nil {
		for _, img := range resp.Payload.Images {
			if img == nil {
				continue
			}
			t.AddRow(imageRow{
				Ref:    img.Ref,
				Digest: img.Digest,
			})
		}
	}
	return t.Flush()
}
