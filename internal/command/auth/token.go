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
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	if cfg.BearerToken == "" {
		return fmt.Errorf("no bearer token available (API key auth does not provide a token)")
	}
	fmt.Fprint(out, cfg.BearerToken)
	return nil
}
