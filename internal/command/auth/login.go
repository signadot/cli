package auth

import (
	"fmt"
	"time"

	"github.com/signadot/cli/internal/auth"
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func newLogin(cfg *config.Auth) *cobra.Command {
	loginCfg := &config.AuthLogin{Auth: cfg}

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in to Signadot",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogin(loginCfg)
		},
	}

	return cmd
}

func runLogin(cfg *config.AuthLogin) error {
	// TODO: Implement actual device flow
	fmt.Println("To authenticate, visit: https://activate.signadot.com")
	fmt.Println("And enter code: ABC-DEF")
	fmt.Println("\nWaiting for authentication...")

	// Simulate waiting for auth
	time.Sleep(2 * time.Second)

	// Store dummy token (will be replaced with real token later)
	token := "dummy.jwt.token"
	if err := auth.StoreToken(token); err != nil {
		return fmt.Errorf("failed to store token: %w", err)
	}

	fmt.Println("✓ Successfully logged in")
	return nil
}
