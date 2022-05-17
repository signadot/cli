package config

import (
	"time"

	"github.com/spf13/cobra"
)

type Sandbox struct {
	*Api
}

type SandboxCreate struct {
	*Sandbox

	// Flags
	Filename    string
	Wait        bool
	WaitTimeout time.Duration
}

func (c *SandboxCreate) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&c.Filename, "filename", "f", "", "YAML or JSON file containing the sandbox creation request")
	cmd.Flags().BoolVar(&c.Wait, "wait", true, "wait for the sandbox status to be Ready before returning")
	cmd.Flags().DurationVar(&c.WaitTimeout, "wait-timeout", 5*time.Minute, "timeout when waiting for the sandbox to be Ready")
	cmd.MarkFlagRequired("filename")
}

type SandboxDelete struct {
	*Sandbox

	// Flags
	Filename string
}

func (c *SandboxDelete) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&c.Filename, "filename", "f", "", "optional YAML or JSON file containing the original sandbox creation request")
}

type SandboxGet struct {
	*Sandbox
}

type SandboxList struct {
	*Sandbox
}
