package secret

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	sdksecrets "github.com/signadot/go-sdk/client/secrets"
	"github.com/spf13/cobra"
)

func newGet(secret *config.Secret) *cobra.Command {
	cfg := &config.SecretGet{Secret: secret}

	cmd := &cobra.Command{
		Use:   "get NAME",
		Short: "Get secret metadata (plaintext value is never returned)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return get(cfg, cmd.OutOrStdout(), args[0])
		},
	}

	return cmd
}

func get(cfg *config.SecretGet, out io.Writer, name string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	params := sdksecrets.NewGetSecretParams().
		WithOrgName(cfg.Org).
		WithSecretName(name)
	resp, err := cfg.Client.Secrets.GetSecret(params, nil)
	if err != nil {
		return err
	}

	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return printSecretDetails(out, resp.Payload)
	case config.OutputFormatJSON:
		return print.RawJSON(out, resp.Payload)
	case config.OutputFormatYAML:
		return print.RawYAML(out, resp.Payload)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
