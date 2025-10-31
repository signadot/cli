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
		Long: `The 'override' command lets you intercept both HTTP and gRPC traffic coming into your sandbox and process it with a local service you specify (such as on your laptop).

How it works:
- Any HTTP or gRPC request to your sandbox is first routed to your local service.
- Your local service processes the request and replies.
- By default:
    * If your local service's response contains the header 'sd-override: true' (for HTTP), or the metadata 'sd-override: true' (for gRPC), the response from your local service is immediately sent back to the client—the request does NOT reach the original sandbox workload.
    * For all other responses (i.e., 'sd-override: true' not present), the request is forwarded to the original sandbox workload as usual, and its response is returned to the client.
- If your local service is unavailable or not running, all requests automatically fall through to the original sandbox workload.

Special case: Using '--except-status'
- When you provide the '--except-status' flag, you specify a list of HTTP status codes (such as 404,503).
- With this flag, all requests are overridden and served by your local service, EXCEPT when your local service returns one of the specified HTTP status codes.
- In those cases (when your local service replies with a status listed in '--except-status'), the request is forwarded to the original sandbox workload, and its response is returned to the client instead.

This setup allows flexible and powerful local testing of changes for both HTTP and gRPC services while letting you make exceptions for specific HTTP status codes as needed.`,
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
