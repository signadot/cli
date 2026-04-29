package secret

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	sdksecrets "github.com/signadot/go-sdk/client/secrets"
	"github.com/signadot/go-sdk/models"
	"github.com/spf13/cobra"
)

func newCreate(secret *config.Secret) *cobra.Command {
	cfg := &config.SecretCreate{Secret: secret}

	cmd := &cobra.Command{
		Use:   "create { NAME --value VALUE | NAME --value-file PATH | NAME --value-stdin | -f FILENAME [--set var=val ...] } [--description TEXT]",
		Short: "Create a new secret",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return create(cfg, cmd.OutOrStdout(), cmd.ErrOrStderr(), args)
		},
	}

	cfg.AddFlags(cmd)
	return cmd
}

func create(cfg *config.SecretCreate, out, log io.Writer, args []string) error {
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

	params := sdksecrets.NewCreateSecretParams().
		WithOrgName(cfg.Org).
		WithData(s)
	resp, err := cfg.Client.Secrets.CreateSecret(params, nil)
	if err != nil {
		return err
	}

	fmt.Fprintf(log, "Created secret %q\n\n", s.Name)
	return writeSecretOutput(cfg.OutputFormat, out, resp.Payload)
}

type secretInputs struct {
	Args        []string
	Filename    string
	TplVals     config.TemplateVals
	Value       string
	ValueFile   string
	ValueStdin  bool
	Description string
	Log         io.Writer
}

// buildSecretFromInputs resolves a *models.Secret from the combination of a
// positional NAME, value flags, and optional -f file. It enforces mutual
// exclusion between the file mode and the flat-CLI mode.
func buildSecretFromInputs(in secretInputs) (*models.Secret, error) {
	if in.Filename != "" {
		if len(in.Args) != 0 {
			return nil, errors.New("must not provide NAME positional when -f is specified")
		}
		if in.Value != "" || in.ValueFile != "" || in.ValueStdin {
			return nil, errors.New("must not combine -f with --value / --value-file / --value-stdin")
		}
		if in.Description != "" {
			return nil, errors.New("must not combine -f with --description")
		}
		if len(in.TplVals) != 0 && in.Filename == "" {
			return nil, errors.New("--set requires -f")
		}
		return loadSecretFile(in.Filename, in.TplVals, false /* forDelete */)
	}

	if len(in.Args) == 0 {
		return nil, errors.New("must specify NAME or -f FILENAME")
	}
	if len(in.TplVals) != 0 {
		return nil, errors.New("--set requires -f")
	}

	value, err := resolveValue(in.Value, in.ValueFile, in.ValueStdin, in.Log)
	if err != nil {
		return nil, err
	}
	return &models.Secret{
		Name:        in.Args[0],
		Description: in.Description,
		Value:       value,
	}, nil
}

// resolveValue reads the secret value from exactly one of the three flag sources.
// Returns "" when none are set; callers decide whether that's an error.
func resolveValue(literal, path string, fromStdin bool, log io.Writer) (string, error) {
	n := 0
	if literal != "" {
		n++
	}
	if path != "" {
		n++
	}
	if fromStdin {
		n++
	}
	if n > 1 {
		return "", errors.New("--value, --value-file, and --value-stdin are mutually exclusive")
	}

	switch {
	case literal != "":
		fmt.Fprintln(log, "warning: --value leaks the secret into shell history; prefer --value-file or --value-stdin")
		return literal, nil
	case path != "":
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("reading --value-file: %w", err)
		}
		return string(data), nil
	case fromStdin:
		fi, err := os.Stdin.Stat()
		if err == nil && (fi.Mode()&os.ModeCharDevice) != 0 {
			return "", errors.New("--value-stdin was given but stdin is a terminal")
		}
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("reading stdin: %w", err)
		}
		return string(data), nil
	default:
		return "", nil
	}
}

func writeSecretOutput(format config.OutputFormat, out io.Writer, s *models.Secret) error {
	switch format {
	case config.OutputFormatDefault:
		return nil
	case config.OutputFormatJSON:
		return print.RawJSON(out, s)
	case config.OutputFormatYAML:
		return print.RawYAML(out, s)
	default:
		return fmt.Errorf("unsupported output format: %q", format)
	}
}
