package devbox

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func newDelete(devbox *config.Devbox) *cobra.Command {
	cfg := &config.DevboxDelete{Devbox: devbox}

	cmd := &cobra.Command{
		Use:   "delete NAME",
		Short: "Delete a devbox",
		Long: `Delete a devbox by name.

This will remove the devbox registration and any associated local state.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg.Name = args[0]
			return deleteDevbox(cfg, cmd.ErrOrStderr())
		},
	}

	cfg.AddFlags(cmd)

	return cmd
}

func deleteDevbox(cfg *config.DevboxDelete, log io.Writer) error {
	ctx, cancel := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer cancel()

	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}

	if cfg.Name == "" {
		return errors.New("devbox name is required")
	}

	// TODO: Implement API call to delete devbox
	// Example:
	// params := devboxes.NewDeleteDevboxParams().
	//     WithContext(ctx).
	//     WithOrgName(cfg.Org).
	//     WithUser(cfg.User).
	//     WithDevboxName(cfg.Name)
	//
	// _, err := cfg.Client.Devboxes.DeleteDevbox(params, nil)
	// if err != nil {
	//     return err
	// }

	_ = ctx // TODO: remove when implementing API call
	fmt.Fprintf(log, "TODO: Implement devbox delete API call\n")
	fmt.Fprintf(log, "Deleted devbox %q.\n\n", cfg.Name)

	// TODO: If this is the default devbox, clean up ~/.signadot/default-devbox
	// Example:
	// if isDefaultDevbox(cfg.Name) {
	//     if err := removeDefaultDevbox(); err != nil {
	//         fmt.Fprintf(log, "Warning: failed to remove default devbox config: %v\n", err)
	//     }
	// }

	// TODO: Wait for deletion with polling if cfg.Wait is true
	// Example:
	// if cfg.Wait {
	//     if err := waitForDeleted(ctx, cfg, log, cfg.Name); err != nil {
	//         return err
	//     }
	// }

	return nil
}

// TODO: Implement helper functions
// func isDefaultDevbox(name string) bool { ... }
// func removeDefaultDevbox() error { ... }
// func waitForDeleted(ctx context.Context, cfg *config.DevboxDelete, log io.Writer, name string) error { ... }
