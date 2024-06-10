package config

import (
	"github.com/spf13/cobra"
)

type Logs struct {
	*API

	Job       string
	Stream    string
	TailLines uint
}

func (c *Logs) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&c.Job, "job", "j", "", "job name whose log lines will be displayed")
	cmd.MarkFlagRequired("job")

	cmd.Flags().StringVarP(&c.Stream, "stream", "s", "stdout", "stream from where to display log lines (stdout or stderr)")
	cmd.Flags().UintVarP(&c.TailLines, "tail", "t", 0, "lines of recent log file to display, defaults to 0, showing all log lines")
}
