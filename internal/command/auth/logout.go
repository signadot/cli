package auth

import (
	"fmt"

	"github.com/signadot/cli/internal/auth"
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func newLogout(cfg *config.Auth) *cobra.Command {
	logoutCfg := &config.AuthLogout{Auth: cfg}

	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Log out from Signadot",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogout(logoutCfg)
		},
	}

	return cmd
}

func runLogout(cfg *config.AuthLogout) error {
	if err := auth.DeleteToken(); err != nil {
		return fmt.Errorf("failed to delete token: %w", err)
	}

	fmt.Println("✓ Successfully logged out")
	return nil
}
