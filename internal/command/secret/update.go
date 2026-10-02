package secret

import (
	"errors"
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	sdksecrets "github.com/signadot/go-sdk/client/secrets"
	"github.com/spf13/cobra"
)

func newUpdate(secret *config.Secret) *cobra.Command {
	cfg := &config.SecretUpdate{Secret: secret}

	cmd := &cobra.Command{
		Use:   "update { NAME --value VALUE | NAME --value-file PATH | NAME --value-stdin | -f FILENAME [--set var=val ...] } [--description TEXT]",
		Short: "Update an existing secret (value is required)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return update(cfg, cmd.Flags().Changed("description"), cmd.OutOrStdout(), cmd.ErrOrStderr(), args)
		},
	}

	cfg.AddFlags(cmd)
	return cmd
}

// descriptionSet reports whether --description was given (possibly empty).
func update(cfg *config.SecretUpdate, descriptionSet bool, out, log io.Writer, args []string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}

	s, err := buildSecretFromInputs(secretInputs{
		Args:        args,
		Filename:    cfg.Filename,
		TplVals:     cfg.TemplateVals,
		Value:       cfg.Value,
		ValueFile:   cfg.ValueFile,
		ValueStdin:  cfg.ValueStdin,
		Description: cfg.Description,
		Log:         log,
	})
	if err != nil {
		return err
	}
	if s.Name == "" {
		return errors.New("secret name is required")
	}
	if s.Value == "" {
		return errors.New("value is required; supply one of --value / --value-file / --value-stdin, or a file with -f")
	}

	// The update replaces the description, so keep the current one unless
	// a new one was given (with -f, the file is the whole secret).
	if cfg.Filename == "" && !descriptionSet {
		getParams := sdksecrets.NewGetSecretParams().
			WithOrgName(cfg.Org).
			WithSecretName(s.Name)
		cur, err := cfg.Client.Secrets.GetSecret(getParams, nil)
		if err != nil {
			return err
		}
		s.Description = cur.Payload.Description
	}

	params := sdksecrets.NewUpdateSecretParams().
		WithOrgName(cfg.Org).
		WithSecretName(s.Name).
		WithData(s)
	resp, err := cfg.Client.Secrets.UpdateSecret(params, nil)
	if err != nil {
		return err
	}

	fmt.Fprintf(log, "Updated secret %q\n\n", s.Name)
	return writeSecretOutput(cfg.OutputFormat, out, resp.Payload)
}
