package secret

import (
	"errors"
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	sdksecrets "github.com/signadot/go-sdk/client/secrets"
	"github.com/spf13/cobra"
)

func newDelete(secret *config.Secret) *cobra.Command {
	cfg := &config.SecretDelete{Secret: secret}

	cmd := &cobra.Command{
		Use:     "delete { NAME | -f FILENAME [--set var=val ...] }",
		Short:   "Delete a secret",
		Aliases: []string{"rm"},
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return deleteSecret(cfg, cmd.ErrOrStderr(), args)
		},
	}

	cfg.AddFlags(cmd)
	return cmd
}

func deleteSecret(cfg *config.SecretDelete, log io.Writer, args []string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}

	var name string
	if cfg.Filename == "" {
		if len(args) == 0 {
			return errors.New("must specify NAME or -f FILENAME")
		}
		if len(cfg.TemplateVals) != 0 {
			return errors.New("--set requires -f")
		}
		name = args[0]
	} else {
		if len(args) != 0 {
			return errors.New("must not provide NAME positional when -f is specified")
		}
		s, err := loadSecretFile(cfg.Filename, cfg.TemplateVals, true /* forDelete */)
		if err != nil {
			return err
		}
		name = s.Name
	}
	if name == "" {
		return errors.New("secret name is required")
	}

	params := sdksecrets.NewDeleteSecretParams().
		WithOrgName(cfg.Org).
		WithSecretName(name)
	if _, err := cfg.Client.Secrets.DeleteSecret(params, nil); err != nil {
		return err
	}
	fmt.Fprintf(log, "Deleted secret %q.\n\n", name)
	return nil
}
