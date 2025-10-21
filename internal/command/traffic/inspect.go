package traffic

import (
	"fmt"
	"io"
	"os"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/poll"
	"github.com/spf13/cobra"
)

func newInspect(cfg *config.Traffic) *cobra.Command {
	inspectCfg := &config.TrafficInspect{
		Traffic: cfg,
	}

	cmd := &cobra.Command{
		Use:   "inspect --directory DIRECTORY [--wait]",
		Short: "Inspect traffic data from a directory",
		Long: `Inspect traffic data from a directory containing recorded traffic.

This command validates that the specified directory contains valid traffic data
by checking for the presence of meta files (meta.json, meta.yaml, meta.jsons, or meta.yamls).
If the directory doesn't contain any of these meta files, it will return an error 
indicating it's not a valid traffic directory.

When the --wait flag is used, the command will wait for the directory to contain
valid traffic data if it's initially empty, checking every second until meta files
are found.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return inspectTraffic(inspectCfg, cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}

	inspectCfg.AddFlags(cmd)
	return cmd
}

func inspectTraffic(cfg *config.TrafficInspect, w, wErr io.Writer) error {
	// Check if directory exists
	directoryInfo, err := os.Stat(cfg.Directory)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("directory does not exist: %s", cfg.Directory)
		}
		return fmt.Errorf("error accessing directory: %w", err)
	}
	if !directoryInfo.IsDir() {
		return fmt.Errorf("path is not a directory: %s", cfg.Directory)
	}

	// Look for meta.jsons or meta.yamls files in the directory
	hasMetaFile, err := hasMetaFile(cfg.Directory)
	if err != nil {
		return fmt.Errorf("error checking for meta files: %w", err)
	}

	if !hasMetaFile {
		if cfg.Wait {
			fmt.Fprintf(w, "Directory %s is empty, waiting for traffic data...\n", cfg.Directory)
			return waitForMetaFile(cfg.Directory, w)
		}
		return fmt.Errorf("Directory %s is empty", cfg.Directory)
	}

	fmt.Fprintf(w, "Directory %s contains valid traffic data\n", cfg.Directory)
	return nil
}

func hasMetaFile(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == "meta.json" || name == "meta.yaml" ||
			name == "meta.jsons" || name == "meta.yamls" {
			return true, nil
		}
	}

	return false, nil
}

func waitForMetaFile(dir string, w io.Writer) error {
	p := poll.NewPoll()

	return p.UntilWithError(func() (bool, error) {
		hasMetaFile, err := hasMetaFile(dir)
		if err != nil {
			return false, fmt.Errorf("error checking for meta files: %w", err)
		}
		if hasMetaFile {
			fmt.Fprintf(w, "Directory %s now contains valid traffic data\n", dir)
			return true, nil
		}
		return false, nil
	})
}
