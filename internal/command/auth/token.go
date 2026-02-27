package auth

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func newToken(cfg *config.Auth) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token",
		Short: "Print auth token to stdout",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runToken(cfg, cmd.OutOrStdout())
		},
	}
	return cmd
}

func runToken(cfg *config.Auth, out io.Writer) error {
	token, err := cfg.RefreshBearerToken()
	if err != nil {
		return fmt.Errorf("could not get token: %w", err)
	}
	fmt.Fprint(out, token)
	return nil
}
