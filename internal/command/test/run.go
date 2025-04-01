package test

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/repoconfig"
	"github.com/spf13/cobra"
)

func newRun(tConfig *config.Test) *cobra.Command {
	cfg := &config.TestRun{
		Test: tConfig,
	}
	cmd := &cobra.Command{
		Use:   "run <name>",
		Short: "Run a test",
		// Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cfg, cmd.OutOrStdout(), cmd.ErrOrStderr(), args)
		},
	}
	cfg.AddFlags(cmd)
	return cmd
}

func run(cfg *config.TestRun, wOut, wErr io.Writer, args []string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}

	// create a test finder
	tf, err := repoconfig.NewTestFinder(cfg.Directory)
	if err != nil {
		return err
	}

	// find tests
	testFiles, err := tf.FindTestFiles()
	if err != nil {
		return err
	}

	// debug print
	gitRepo := tf.GetGitRepo()
	if gitRepo != nil {
		fmt.Fprintf(wOut,
			"Git repo info:\n * path = %s\n * repo = %s\n * branch = %s\n * commit sha = %s\n\n",
			gitRepo.Path, gitRepo.Branch, gitRepo.Branch, gitRepo.CommitSHA)
	}

	fmt.Fprintf(wOut, "Found %d test files:\n", len(testFiles))
	for _, tf := range testFiles {
		fmt.Fprintf(wOut, " * Name %s\n", tf.Name)
		fmt.Fprintf(wOut, "   Path %s\n", tf.Path)
		if len(tf.Labels) > 0 {
			fmt.Fprintf(wOut, "   Labels:\n")
			for k, v := range tf.Labels {
				fmt.Fprintf(wOut, "      %s = %s\n", k, v)
			}
		}
	}
	return nil
}
