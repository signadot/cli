package devbox

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func newRegister(devbox *config.Devbox) *cobra.Command {
	cfg := &config.DevboxRegister{Devbox: devbox}

	cmd := &cobra.Command{
		Use:   "register",
		Short: "Register a devbox for local development",
		Long: `Register a devbox to associate this machine with your account.
This allows you to connect to remote clusters and use local development features.

If --name is not provided, a name will be automatically generated.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return register(cfg, cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}

	cfg.AddFlags(cmd)

	return cmd
}

func register(cfg *config.DevboxRegister, out, log io.Writer) error {
	ctx, cancel := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer cancel()

	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}

	// TODO: Get machine metadata (localMachineID, hostName, OS, CLIVersion)
	// This should come from system info utilities
	// Example:
	// machineID := getMachineID()
	// hostName, _ := os.Hostname()
	// osInfo := runtime.GOOS

	// TODO: Implement API call to claim/register devbox
	// Example:
	// claimReq := &models.DevboxClaimRequest{
	//     Machine: &models.Machine{
	//         Name:    cfg.Name, // Optional, may be empty
	//         Primary: cfg.Primary,
	//         Meta: &models.MachineMeta{
	//             LocalMachineID: machineID,
	//             HostName:       hostName,
	//             OS:             osInfo,
	//             CLIVersion:     version.Version,
	//         },
	//     },
	// }
	//
	// resp, err := cfg.Client.Devboxes.ClaimDevbox(
	//     devboxes.NewClaimDevboxParams().
	//         WithContext(ctx).
	//         WithOrgName(cfg.Org).
	//         WithUser(cfg.User).
	//         WithBody(claimReq),
	//     nil,
	// )
	// if err != nil {
	//     return err
	// }

	// TODO: Store the machine info in ~/.signadot/default-devbox
	// Example:
	// if err := saveDefaultDevbox(resp.Payload.Machine); err != nil {
	//     return err
	// }

	// For now, return a placeholder
	_ = ctx
	fmt.Fprintln(log, "TODO: Implement devbox register API call")
	if cfg.Name != "" {
		fmt.Fprintf(out, "Registered devbox with name: %s\n", cfg.Name)
	} else {
		fmt.Fprintln(out, "Registered devbox (name will be auto-generated)")
	}

	return nil
}

// TODO: Implement helper functions
// func getMachineID() string { ... }
// func saveDefaultDevbox(machine *models.Machine) error { ... }
// func loadDefaultDevbox() (*models.Machine, error) { ... }
