package config

import (
	"github.com/spf13/cobra"
)

type Artifact struct {
	*API
}

type ArtifactDownload struct {
	*Artifact

	// Flags
	Job        string
	OutputFile string

	//Attempt int64
}

func (c *ArtifactDownload) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&c.Job, "job", "j", "", "job name where to get the attempt artifact")
	cmd.MarkFlagRequired("job")

	cmd.Flags().StringVarP(&c.OutputFile, "output", "o", "", "path where the file would be downloaded")
	cmd.MarkFlagRequired("output")

	//cmd.Flags().Int64VarP(&c.Attempt, "attempt", "a", 0, "number of the attempt to get")
}
