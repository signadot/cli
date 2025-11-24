package devbox

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
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

	return cmd
}

func list(cfg *config.DevboxList, out io.Writer) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}

	// TODO: Implement API call to list devboxes
	// Example:
	// resp, err := cfg.Client.Devboxes.ListDevboxes(
	//     devboxes.NewListDevboxesParams().
	//         WithOrgName(cfg.Org).
	//         WithUser(cfg.User),
	//     nil,
	// )
	// if err != nil {
	//     return err
	// }

	// For now, return a placeholder
	fmt.Fprintln(out, "TODO: Implement devbox list API call")

	// TODO: Handle output format
	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		// return printDevboxTable(out, resp.Payload)
		return nil
	case config.OutputFormatJSON:
		// return print.RawJSON(out, resp.Payload)
		return print.RawJSON(out, []interface{}{})
	case config.OutputFormatYAML:
		// return print.RawYAML(out, resp.Payload)
		return print.RawYAML(out, []interface{}{})
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
