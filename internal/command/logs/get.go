package logs

import (
	"fmt"
	"github.com/go-openapi/runtime"
	"github.com/signadot/cli/internal/config"
	"github.com/signadot/go-sdk/client"
	"github.com/signadot/go-sdk/client/artifacts"
	"github.com/spf13/cobra"
	"io"
	"os"
	"strings"
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
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}

	f, err := os.Create(cfg.OutputFile)
	if err != nil {
		return err
	}

	/*
		If path starts with @ means is system based, otherwise user
	*/
	space := "user"
	if strings.HasPrefix(artifactPath, "@") {
		space = "system"
		artifactPath = strings.TrimPrefix(artifactPath, "@")
	}

	params := artifacts.
		NewDownloadJobAttemptArtifactParams().
		WithOrgName(cfg.Org).
		WithJobName(cfg.Job).
		WithJobAttempt(0).
		WithPath(artifactPath).
		WithSpace(&space)

	err = cfg.APIClientWithCustomTransport(cfg.OverrideTransportClientConsumers(map[string]runtime.Consumer{
		runtime.TextMime: runtime.ByteStreamConsumer(),
	}),
		func(c *client.SignadotAPI) error {
			_, _, err = c.Artifacts.DownloadJobAttemptArtifact(params, nil, f)
			if err != nil {
				return err
			}

			fmt.Fprintf(out, "File saved successfully at %s\n", cfg.OutputFile)

			return nil
		})

	if err != nil {
		return err
	}

	return nil
}
