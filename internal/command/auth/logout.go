package auth

import (
	"errors"
	"fmt"
	"io"

	"github.com/fatih/color"
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
			return runLogout(logoutCfg, cmd.OutOrStdout())
		},
	}

	return cmd
}

func runLogout(cfg *config.AuthLogout, out io.Writer) error {
	authInfo, err := auth.ResolveAuth()
	if err != nil {
		return fmt.Errorf("could not resolve auth: %w", err)
	}
	if authInfo == nil {
		return errors.New("You are already logged out.")
	}
	if authInfo.Source == auth.ConfigAuthSource {
		return errors.New(`You are currently logged in using an API key specified via a configuration file
or environment variable. To log out, you must manually unset the environment variable
or remove the API key from the configuration file.`)
	}

	if err := auth.DeleteAuthFromKeyring(); err != nil {
		return fmt.Errorf("failed to delete auth info: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Fprintf(out, "%s Successfully logged out\n", green("✓"))
	return nil
}
