package artifact

import (
	"fmt"
	"github.com/signadot/cli/internal/config"
	"github.com/signadot/go-sdk/client/artifacts"
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

type ArtifactWriter struct {
	filename string
	file     *os.File
}

// Write implements the io.Writer interface
func (mw *ArtifactWriter) Write(p []byte) (n int, err error) {
	if mw.file == nil {
		return 0, fmt.Errorf("file not opened")
	}
	return mw.file.Write(p)
}

// UnmarshalText implements the encoding.TextUnmarshaler interface
func (mw *ArtifactWriter) UnmarshalText(text []byte) error {
	var err error
	mw.file, err = os.OpenFile(mw.filename, os.O_RDWR|os.O_CREATE, 0666)

	_, err = mw.Write(text)
	return err
}

func get(cfg *config.ArtifactDownload, out io.Writer, name string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}

	artifact := &ArtifactWriter{
		filename: cfg.OutputFile,
	}
	space := "system"

	params := artifacts.
		NewDownloadJobAttemptArtifactParams().
		WithOrgName(cfg.Org).
		WithJobName(cfg.Job).
		WithJobAttempt(0).
		WithPath(name).WithSpace(&space)
	status, partialContent, err := cfg.Client.Artifacts.DownloadJobAttemptArtifact(params, nil, artifact)
	if err != nil {
		return err
	}

	fmt.Println(status, partialContent)

	return nil
}
