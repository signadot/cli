package local

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/local"
	"github.com/signadot/cli/internal/print"
	"github.com/spf13/cobra"
)

func newStatus(localConfig *config.Local) *cobra.Command {
	cfg := &config.LocalStatus{Local: localConfig}

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show status of the local machine's connection with cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(cfg, cmd.OutOrStdout(), args)
		},
	}
	cfg.AddFlags(cmd)

	return cmd
}

func runStatus(cfg *config.LocalStatus, out io.Writer, args []string) error {
	if err := cfg.InitLocalConfig(); err != nil {
		return err
	}
	status, err := local.GetLocalStatus()
	if err != nil {
		return err
	}
	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return printLocalStatus(cfg, out, status)
	case config.OutputFormatJSON:
		return printRawStatus(cfg, out, print.RawJSON, status)
	case config.OutputFormatYAML:
		return printRawStatus(cfg, out, print.RawK8SYAML, status)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
