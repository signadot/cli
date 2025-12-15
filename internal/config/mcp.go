package config

import "github.com/spf13/cobra"

// MCP represents the configuration for the mcp command
type MCP struct {
	*API
}

// AddFlags adds flags for the mcp command.
func (c *MCP) AddFlags(cmd *cobra.Command) {
	// No additional flags needed - Debug is available via Root's persistent flags
}
