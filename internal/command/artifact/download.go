package artifact

import (
	"fmt"
	"github.com/go-openapi/runtime"
	"github.com/signadot/cli/internal/config"
	"github.com/signadot/go-sdk/client"
	"github.com/signadot/go-sdk/client/artifacts"
	"github.com/spf13/cobra"
	"io"
	"os"
	"path"
	"strings"
)

func newDownload(artifact *config.Artifact) *cobra.Command {
	cfg := &config.ArtifactDownload{Artifact: artifact}

	cmd := &cobra.Command{
		Use:   "download PATH",
		Short: "Download job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return download(cfg, cmd.OutOrStdout(), args[0])
		},
	}

	cfg.AddFlags(cmd)

	return cmd
}

func download(cfg *config.ArtifactDownload, out io.Writer, artifactPath string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}

	outputFilename := getOutputFilename(cfg, artifactPath)
	f, err := os.Create(outputFilename)
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

			fmt.Fprintf(out, "File saved successfully at %s\n", outputFilename)

			return nil
		})

	if err != nil {
		return err
	}

	return nil
}

func getOutputFilename(cfg *config.ArtifactDownload, artifactPath string) string {
	if len(cfg.OutputFile) != 0 {
		return cfg.OutputFile
	}

	return path.Base(artifactPath)
}
