package traffic

import (
	"fmt"
	"io"
	"os"

	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func newInspect(cfg *config.Traffic) *cobra.Command {
	inspectCfg := &config.TrafficInspect{
		Traffic: cfg,
	}

	cmd := &cobra.Command{
		Use:   "inspect --directory DIRECTORY",
		Short: "Inspect traffic data from a directory",
		Long: `Inspect traffic data from a directory containing recorded traffic.

This command validates that the specified directory contains valid traffic data
by checking for the presence of meta.json files. If the directory doesn't contain
meta.json files, it will return an error indicating it's not a valid traffic directory.`,
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
		return fmt.Errorf("directory is not a valid traffic directory: %s (no meta files found)", cfg.Directory)
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
		if entry.Name() == "meta.jsons" || entry.Name() == "meta.yamls" {
			return true, nil
		}
	}

	return false, nil
}
