package artifact

import (
	"fmt"
	"github.com/go-openapi/runtime"
	"github.com/signadot/cli/internal/config"
	"github.com/signadot/go-sdk/client/artifacts"
	"github.com/signadot/go-sdk/transport"
	"github.com/spf13/cobra"
	"io"
	"os"
)

func newGet(artifact *config.Artifact) *cobra.Command {
	cfg := &config.ArtifactDownload{Artifact: artifact}

	cmd := &cobra.Command{
		Use:   "get NAME",
		Short: "Get job",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return get(cfg, cmd.OutOrStdout(), args[0])
		},
	}

	cfg.AddFlags(cmd)

	return cmd
}

func get(cfg *config.ArtifactDownload, out io.Writer, artifactPath string) error {
	if err := cfg.InitAPIConfigWithCustomTransport(func(apiKey, apiUrl, artifactsUrl, userAgent string) *transport.APIConfig {
		return &transport.APIConfig{
			APIKey:          apiKey,
			APIURL:          apiUrl,
			ArtifactsAPIURL: artifactsUrl,
			UserAgent:       userAgent,
			Consumers: map[string]runtime.Consumer{
				// override text/plain consumer to consume logs as a stream
				runtime.TextMime: runtime.ByteStreamConsumer(),
			},
		}
	}); err != nil {
		return err
	}

	f, err := os.Create(cfg.OutputFile)
	if err != nil {
		return err
	}

	space := "system"

	params := artifacts.
		NewDownloadJobAttemptArtifactParams().
		WithOrgName(cfg.Org).
		WithJobName(cfg.Job).
		WithJobAttempt(0).
		WithPath(artifactPath).
		WithSpace(&space)

	_, _, err = cfg.Client.Artifacts.DownloadJobAttemptArtifact(params, nil, f)
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "File saved successfully at %s\n", cfg.OutputFile)

	return nil
}
