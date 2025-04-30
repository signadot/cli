package auth

import (
	"errors"
	"fmt"
	"io"

	"github.com/fatih/color"
	"github.com/signadot/cli/internal/auth"
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

func newLogout(cfg *config.Auth) *cobra.Command {
	logoutCfg := &config.AuthLogout{Auth: cfg}

	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Log out from Signadot",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogout(logoutCfg, cmd.OutOrStdout())
		},
	}

	return cmd
}

func runLogout(cfg *config.AuthLogout, out io.Writer) error {
	if err := auth.DeleteToken(); err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return errors.New("you are already logged out")
		}
		return fmt.Errorf("failed to delete token: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Fprintf(out, "%s Successfully logged out\n", green("✓"))
	return nil
}
