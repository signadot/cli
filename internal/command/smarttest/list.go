package smarttest

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	"github.com/signadot/cli/internal/repoconfig"
	"github.com/spf13/cobra"
)

func newList(tConfig *config.SmartTest) *cobra.Command {
	cfg := &config.SmartTestList{
		SmartTest: tConfig,
	}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List local tests",
		RunE: func(cmd *cobra.Command, args []string) error {
			return list(cmd.Context(), cfg, cmd.OutOrStdout(), cmd.ErrOrStderr(), args)
		},
	}
	cfg.AddFlags(cmd)
	return cmd
}

func list(ctx context.Context, cfg *config.SmartTestList, wOut, wErr io.Writer,
	args []string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	if err := validateList(cfg); err != nil {
		return err
	}
	testFiles, _, err := testFilesAndRepo(cfg)
	if err != nil {
		return err
	}
	// render the structured output
	return listOutput(cfg, wOut, testFiles)
}

func testFilesAndRepo(cfg *config.SmartTestList) ([]repoconfig.TestFile, *repoconfig.GitRepo, error) {
	if cfg.File == "-" {
		host, err := os.Hostname()
		if err != nil {
			host = "unknown"
		}
		pid := strconv.Itoa(os.Getpid())
		return []repoconfig.TestFile{
			{
				Name:   host + "-" + "stdin-" + pid,
				Reader: os.Stdin,
			},
		}, nil, nil
	}
	// create a test finder
	// NOTE: at most one of cfg.{Dir,File} is non-empty
	tf, err := repoconfig.NewTestFinder(cfg.Directory+cfg.File, cfg.FilterLabels, cfg.WithoutLabels)
	if err != nil {
		return nil, nil, err
	}

	// find tests
	testFiles, err := tf.FindTestFiles()
	if err != nil {
		return nil, nil, fmt.Errorf("error finding test files: %w", err)
	}
	if len(testFiles) == 0 {
		return nil, nil, errors.New("could not find any test")
	}
	return testFiles, tf.GetGitRepo(), nil
}

func validateList(cfg *config.SmartTestList) error {
	if cfg.Directory != "" && cfg.File != "" {
		return fmt.Errorf("cannot specify both directory and file")
	}
	if cfg.Directory != "" {
		st, err := os.Stat(cfg.Directory)
		if err != nil {
			return fmt.Errorf("unable to stat input directory: %w", err)
		}
		if !st.IsDir() {
			return fmt.Errorf("%q is not a directory", cfg.Directory)
		}
	}
	if cfg.File != "" && cfg.File != "-" {
		st, err := os.Stat(cfg.File)
		if err != nil {
			return fmt.Errorf("unable to stat input file: %w", err)
		}
		if st.IsDir() {
			return fmt.Errorf("%q is not a file", cfg.File)
		}
	}

	return nil
}

func listOutput(cfg *config.SmartTestList, w io.Writer, tfs []repoconfig.TestFile) error {
	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		for i := range tfs {
			tf := &tfs[i]
			if _, err := fmt.Fprint(w, tf.Name+"\n"); err != nil {
				return err
			}
		}
		return nil
	case config.OutputFormatYAML:
		return print.RawYAML(w, tfs)
	case config.OutputFormatJSON:
		return print.RawJSON(w, tfs)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
