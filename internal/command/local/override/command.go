package override

import (
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func New(local *config.Local) *cobra.Command {
	cfg := &config.LocalOverrideCreate{
		LocalOverride: &config.LocalOverride{Local: local},
	}

	cmd := &cobra.Command{
		Use:   "override --sandbox=<sandbox> [--workload=<workload>] --workload-port=<port> --with=<target> [--except-status=...] [--detach]",
		Short: "Override sandbox HTTP traffic using a local service",
		Long: `Using the 'override' command, when a request comes into the sandbox it is
delivered to the local service. The response of the local service determines
whether or not the request will subsequently be delivered to its original
destination (i.e. the sandbox workload).

When the request is not subsequently delivered to the original destination,
the response from the local service is the response returned to the client.

When the local service is not running, requests will be delivered to the
original destination after failing to communicate with the local service.

By default, overrides apply when the response from the local service
includes the header 'sd-override: true'.

You can use the '--except-status' flag to specify HTTP response codes that
should not be overridden. When set, all other traffic will be overridden
except for the specified status codes, which will fall through to the
original sandboxed destination.`,
		Example: `  # Override sandbox traffic from workload my-workload, port 8080 to localhost:9999
  signadot local override --sandbox=my-sandbox --workload=my-workload --workload-port=8080 --with=localhost:9999

  # Bypass override when the response returns 404 and 503
  signadot local override --sandbox=my-sandbox --workload=my-workload --workload-port=8080 --with=localhost:9999 --except-status=404,503

  # Keep the override active after the CLI session ends
  signadot local override --sandbox=my-sandbox --workload=my-workload --workload-port=8080 --with=localhost:9999 --detach

  # List all active overrides
  signadot local override list

  # Delete a specific override
  signadot local override delete <name> --sandbox=<sandbox>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOverride(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr(), cfg)
		},
	}

	// Add flags for the main override command
	cfg.AddFlags(cmd)

	// Subcommands
	cmd.AddCommand(
		newDelete(cfg.LocalOverride),
		newList(cfg.LocalOverride),
	)

	return cmd
}
