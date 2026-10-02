package artifact

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
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

	// Bound how long the download may go without data — before it starts,
	// and between writes once it has — but not the download itself: large
	// artifacts can take longer than any fixed timeout.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var idleTimedOut atomic.Bool
	idleTimer := time.AfterFunc(downloadIdleTimeout, func() {
		idleTimedOut.Store(true)
		cancel()
	})
	defer idleTimer.Stop()

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
			f := &outputFile{name: outputFilename, onWrite: func() { idleTimer.Reset(downloadIdleTimeout) }}
			_, _, err := c.Artifacts.DownloadJobAttemptArtifact(params, nil, f)
			if err != nil {
				f.abandon()
				switch {
				case idleTimedOut.Load() && f.f != nil:
					return fmt.Errorf("download stalled: no data for %s: %w", downloadIdleTimeout, err)
				case idleTimedOut.Load():
					return fmt.Errorf("download did not start within %s: %w", downloadIdleTimeout, err)
				}
				return err
			}
			if err := f.finish(); err != nil {
				return err
			}

			fmt.Fprintf(out, "File saved successfully at %s\n", outputFilename)
			return nil
		})
}

// downloadIdleTimeout is how long a download may go without data.
var downloadIdleTimeout = 4 * time.Minute

// outputFile is the output file, opened the way -o says — os.Create on the
// name, through whatever it is (a symlink, a device, a pipe) — when the first
// byte arrives. So a request that fails before any data leaves an existing
// file as it was, and one that fails partway leaves what arrived.
type outputFile struct {
	name    string
	onWrite func()
	f       *os.File
}

func (o *outputFile) Write(p []byte) (int, error) {
	o.onWrite()
	if o.f == nil {
		f, err := os.Create(o.name)
		if err != nil {
			return 0, err
		}
		o.f = f
	}
	return o.f.Write(p)
}

// finish closes the file after a download that succeeded, creating it first
// when the artifact was empty.
func (o *outputFile) finish() error {
	if o.f == nil {
		f, err := os.Create(o.name)
		if err != nil {
			return err
		}
		o.f = f
	}
	return o.f.Close()
}

// abandon closes the file after a download that failed, keeping what arrived.
func (o *outputFile) abandon() {
	if o.f != nil {
		o.f.Close()
	}
}

func getOutputFilename(cfg *config.ArtifactDownload, artifactPath string) string {
	if len(cfg.OutputFile) != 0 {
		return cfg.OutputFile
	}

	return path.Base(artifactPath)
}
