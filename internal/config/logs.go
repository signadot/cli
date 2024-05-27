package config

import (
	"github.com/spf13/cobra"
)

type Logs struct {
	*API

	Job    string
	Stream string
}

func (c *Logs) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&c.Job, "job", "j", "", "job name where to get the attempt Logs")
	cmd.MarkFlagRequired("job")

	cmd.Flags().StringVarP(&c.Stream, "stream", "s", "stdout", "channel where get the logs stdout or stderr")
}
