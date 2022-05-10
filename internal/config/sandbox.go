package config

import "github.com/spf13/cobra"

type Sandbox struct {
	*Api
}

type SandboxCreate struct {
	*Sandbox

	// Flags
	Filename string
	Wait     bool
}

func (c *SandboxCreate) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&c.Filename, "filename", "f", "", "YAML or JSON file containing the sandbox creation request")
	cmd.Flags().BoolVar(&c.Wait, "wait", true, "wait for the sandbox status to be Ready before returning")
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
