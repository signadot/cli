package devbox

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	"github.com/signadot/go-sdk/client/devboxes"
	"github.com/spf13/cobra"
)

func newList(devbox *config.Devbox) *cobra.Command {
	cfg := &config.DevboxList{Devbox: devbox}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List devboxes",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return list(cfg, cmd.OutOrStdout())
		},
	}

	cfg.AddFlags(cmd)
	return cmd
}

func list(cfg *config.DevboxList, out io.Writer) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	params := devboxes.NewGetDevboxesParams().
		WithContext(ctx).
		WithOrgName(cfg.Org)
	if cfg.ShowAll {
		all := "true"
		params = params.WithAll(&all)
	}
	resp, err := cfg.Client.Devboxes.GetDevboxes(params)
	if err != nil {
		return err
	}

	devboxes := resp.Payload

	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return printDevboxTable(out, devboxes)
	case config.OutputFormatJSON:
		return print.RawJSON(out, devboxes)
	case config.OutputFormatYAML:
		return print.RawYAML(out, devboxes)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
