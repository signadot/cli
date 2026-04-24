package secret

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	sdksecrets "github.com/signadot/go-sdk/client/secrets"
	"github.com/spf13/cobra"
)

func newList(secret *config.Secret) *cobra.Command {
	cfg := &config.SecretList{Secret: secret}

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List secrets (metadata only)",
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return list(cfg, cmd.OutOrStdout())
		},
	}

	return cmd
}

func list(cfg *config.SecretList, out io.Writer) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	params := sdksecrets.NewListSecretsParams().WithOrgName(cfg.Org)
	resp, err := cfg.Client.Secrets.ListSecrets(params, nil)
	if err != nil {
		return err
	}

	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return printSecretTable(out, resp.Payload)
	case config.OutputFormatJSON:
		return print.RawJSON(out, resp.Payload)
	case config.OutputFormatYAML:
		return print.RawYAML(out, resp.Payload)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
