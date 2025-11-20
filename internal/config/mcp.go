package config

// MCP represents the configuration for the mcp command
type MCP struct {
	*API
}

// MCPRun represents the configuration for running MCP
type MCPRun struct {
	*MCP
}
