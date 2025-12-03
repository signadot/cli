package devbox

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/signadot/cli/internal/config"
	devboxpkg "github.com/signadot/cli/internal/devbox"
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

	// Check if a devbox ID file already exists
	idFile, err := devboxpkg.IDFile()
	if err != nil {
		return fmt.Errorf("failed to get devbox ID file path: %w", err)
	}

	existingID := ""
	if _, err := os.Stat(idFile); err == nil {
		// File exists, read the existing ID
		data, err := os.ReadFile(idFile)
		if err == nil {
			existingID = string(data)
			fmt.Fprintf(log, "Warning: devbox ID file already exists (ID: %s). It will be overwritten.\n", existingID)
		}
	}

	// Register the devbox
	devboxID, err := devboxpkg.RegisterDevbox(ctx, cfg.API, cfg.Claim, cfg.Name)
	if err != nil {
		return fmt.Errorf("failed to register devbox: %w", err)
	}

	// Save the devbox ID to file
	if err := os.WriteFile(idFile, []byte(devboxID), 0600); err != nil {
		return fmt.Errorf("failed to save devbox ID: %w", err)
	}

	// Get the devbox name that was used (might be hostname if not provided)
	devboxName := cfg.Name
	if devboxName == "" {
		hostname, err := os.Hostname()
		if err == nil {
			devboxName = hostname
		} else {
			devboxName = "unknown"
		}
	}

	// Print success message
	if existingID != "" && existingID != devboxID {
		fmt.Fprintf(out, "Successfully registered devbox (ID: %s, name: %s)\n", devboxID, devboxName)
		fmt.Fprintf(out, "Previous devbox ID (%s) has been replaced.\n", existingID)
	} else {
		fmt.Fprintf(out, "Successfully registered devbox (ID: %s, name: %s)\n", devboxID, devboxName)
	}

	return nil
}
