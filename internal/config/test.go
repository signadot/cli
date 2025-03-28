package config

import (
	"time"

	"github.com/spf13/cobra"
)

// Test represents the configuration for the test command
type Test struct {
	*API
}

// TestRun represents the configuration for running a test
type TestRun struct {
	*Test
	Directory  string
	Cluster    string
	Sandbox    string
	RouteGroup string
	Publish    bool
	Timeout    time.Duration
	Wait       bool
}

// AddFlags adds the flags for the test run command
func (c *TestRun) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&c.Directory, "directory", "d", "", "Base directory for finding tests")
	cmd.Flags().StringVar(&c.Cluster, "cluster", "", "Cluster where to run tests")
	cmd.Flags().StringVar(&c.Sandbox, "sandbox", "", "Sandbox where to run tests")
	cmd.Flags().StringVar(&c.RouteGroup, "route-group", "", "Route group where to run tests")
	cmd.Flags().BoolVar(&c.Publish, "publish", false, "Publish test results")
	cmd.Flags().DurationVar(&c.Timeout, "timeout", 0, "timeout when waiting for the tests to complete, if 0 is specified, no timeout will be applied (default 0)")
	cmd.Flags().BoolVar(&c.Wait, "wait", false, "waits until the tests are completed")
}
