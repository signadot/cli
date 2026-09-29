package artifact

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-openapi/runtime"
	"github.com/signadot/cli/internal/config"
	"github.com/signadot/go-sdk/client"
	"github.com/signadot/go-sdk/client/artifacts"
	"github.com/spf13/cobra"
)

func newDownload(artifact *config.Artifact) *cobra.Command {
	cfg := &config.ArtifactDownload{Artifact: artifact}

	cmd := &cobra.Command{
		Use:   "download PATH",
		Short: "Download job artifacts",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return download(cmd.Context(), cfg, cmd.OutOrStdout(), args[0])
		},
	}

	cfg.AddFlags(cmd)

	return cmd
}

func download(ctx context.Context, cfg *config.ArtifactDownload, out io.Writer, artifactPath string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}

	outputFilename := getOutputFilename(cfg, artifactPath)

	// If path starts with @ means is system based, otherwise user
	space := "user"
	if strings.HasPrefix(artifactPath, "@") {
		space = "system"
		artifactPath = strings.TrimPrefix(artifactPath, "@")
	}

	// Bound the time until the download starts, but not the download itself
	// (large artifacts can take longer than any fixed timeout).
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var startTimedOut atomic.Bool
	startTimer := time.AfterFunc(downloadStartTimeout, func() {
		startTimedOut.Store(true)
		cancel()
	})
	defer startTimer.Stop()

	params := artifacts.
		NewDownloadJobAttemptArtifactParams().
		WithContext(ctx).
		WithTimeout(0).
		WithOrgName(cfg.Org).
		WithJobName(cfg.Job).
		WithJobAttempt(0).
		WithPath(artifactPath).
		WithSpace(&space)

	// create a custom transport to treat everything as a byte stream
	transportCfg := cfg.GetBaseTransport()
	transportCfg.OverrideConsumers = true
	transportCfg.Consumers = map[string]runtime.Consumer{
		"*/*": runtime.ByteStreamConsumer(),
	}

	return cfg.APIClientWithCustomTransport(transportCfg,
		func(c *client.SignadotAPI) error {
			err := writeFileAtomic(outputFilename, func(w io.Writer) error {
				w = &firstWriteWriter{Writer: w, onFirst: func() { startTimer.Stop() }}
				_, _, err := c.Artifacts.DownloadJobAttemptArtifact(params, nil, w)
				return err
			})
			if err != nil {
				if startTimedOut.Load() {
					return fmt.Errorf("download did not start within %s: %w", downloadStartTimeout, err)
				}
				return err
			}

			fmt.Fprintf(out, "File saved successfully at %s\n", outputFilename)
			return nil
		})
}

const downloadStartTimeout = 4 * time.Minute

// writeFileAtomic writes filename with the content produced by write, via a
// temporary file in the same directory, so that on failure an existing file
// is left untouched and no partial file is left behind.
func writeFileAtomic(filename string, write func(w io.Writer) error) error {
	mode := os.FileMode(0644)
	if fi, err := os.Stat(filename); err == nil {
		mode = fi.Mode().Perm()
	}
	tmp, err := os.CreateTemp(filepath.Dir(filename), "."+filepath.Base(filename)+".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // no-op after a successful rename
	if err := write(tmp); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), filename)
}

// firstWriteWriter calls onFirst before the first write.
type firstWriteWriter struct {
	io.Writer
	onFirst func()
	once    sync.Once
}

func (w *firstWriteWriter) Write(p []byte) (int, error) {
	w.once.Do(w.onFirst)
	return w.Writer.Write(p)
}

func getOutputFilename(cfg *config.ArtifactDownload, artifactPath string) string {
	if len(cfg.OutputFile) != 0 {
		return cfg.OutputFile
	}

	return path.Base(artifactPath)
}
