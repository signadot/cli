package config

import "github.com/spf13/cobra"

// MCP represents the configuration for the mcp command
type MCP struct {
	*API
	DisableElicitation bool
}

// AddFlags adds flags for the mcp command.
func (c *MCP) AddFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&c.DisableElicitation, "disable-elicitation", false,
		"disable MCP elicitation independently of client support")
}
