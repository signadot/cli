package devbox

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/signadot/cli/internal/config"
	devboxpkg "github.com/signadot/cli/internal/devbox"
	"github.com/signadot/go-sdk/client/devboxes"
	"github.com/spf13/cobra"
)

func newDelete(devbox *config.Devbox) *cobra.Command {
	cfg := &config.DevboxDelete{Devbox: devbox}

	cmd := &cobra.Command{
		Use:   "delete ID",
		Short: "Delete a devbox",
		Long: `Delete a devbox by ID.

This will remove the devbox registration and any associated local state.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg.ID = args[0]
			return deleteDevbox(cfg, cmd.ErrOrStderr())
		},
	}

	cfg.AddFlags(cmd)

	return cmd
}

func deleteDevbox(cfg *config.DevboxDelete, log io.Writer) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}

	if cfg.ID == "" {
		return errors.New("devbox ID is required")
	}

	// Delete the devbox
	params := devboxes.NewDeleteDevboxParams().
		WithContext(ctx).
		WithOrgName(cfg.Org).
		WithDevboxID(cfg.ID)
	_, err := cfg.Client.Devboxes.DeleteDevbox(params)
	if err != nil {
		return fmt.Errorf("failed to delete devbox: %w", err)
	}

	// Check if this is the local devbox and clean up the ID file if so
	if err := cleanupLocalDevboxID(cfg.ID, log); err != nil {
		// Log warning but continue with deletion
		fmt.Fprintf(log, "Warning: failed to check local devbox ID file: %v\n", err)
	}

	fmt.Fprintf(log, "Deleted devbox (ID: %s).\n", cfg.ID)

	return nil
}

// cleanupLocalDevboxID checks if the deleted devbox is the local devbox and removes the ID file if so.
func cleanupLocalDevboxID(devboxID string, log io.Writer) error {
	idFile, err := devboxpkg.IDFile()
	if err != nil {
		return err
	}

	data, err := os.ReadFile(idFile)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, nothing to clean up
			return nil
		}
		return err
	}

	localID := strings.TrimSpace(string(data))
	if localID == devboxID {
		// This is the local devbox, remove the ID file
		if err := os.Remove(idFile); err != nil {
			return fmt.Errorf("failed to remove local devbox ID file: %w", err)
		}
		fmt.Fprintf(log, "Removed local devbox ID file.\n")
	}

	return nil
}
