package config

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// SmartTest represents the configuration for the test command
type SmartTest struct {
	*API
}

// SmartTestRun represents the configuration for running a test
type SmartTestRun struct {
	*SmartTest
	Directory  string
	Labels     RunLabels
	Cluster    string
	Sandbox    string
	RouteGroup string
	Publish    bool
	Timeout    time.Duration
	NoWait     bool
}

type RunLabels map[string]string

func (rl RunLabels) String() string {
	keys := make([]string, 0, len(rl))
	for k := range rl {
		keys = append(keys, k)
	}
	res := bytes.NewBuffer(nil)
	sort.Stable(sort.StringSlice(keys))
	for i, key := range keys {
		if i != 0 {
			fmt.Fprintf(res, ",")
		}
		fmt.Fprintf(res, "%s:%s", key, rl[key])
	}
	return res.String()
}

func (rl RunLabels) Set(v string) error {
	key, val, ok := strings.Cut(v, ":")
	if !ok {
		return fmt.Errorf("%q should be in form <key>:<value>", v)
	}
	rl[key] = val
	return nil
}

func (tl RunLabels) Type() string {
	return "labels"
}

// AddFlags adds the flags for the test run command
func (c *SmartTestRun) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&c.Directory, "directory", "d", "", "Base directory for finding tests")
	cmd.Flags().StringVar(&c.Cluster, "cluster", "", "Cluster where to run tests")
	cmd.Flags().StringVar(&c.Sandbox, "sandbox", "", "Sandbox where to run tests")
	cmd.Flags().StringVar(&c.RouteGroup, "route-group", "", "Route group where to run tests")
	cmd.Flags().BoolVar(&c.Publish, "publish", false, "Publish test results")
	cmd.Flags().DurationVar(&c.Timeout, "timeout", 0, "timeout when waiting for the tests to complete, if 0 is specified, no timeout will be applied (default 0)")
	cmd.Flags().BoolVar(&c.NoWait, "no-wait", false, "do not wait until the tests are completed")

	c.Labels = make(map[string]string)
	cmd.Flags().Var(c.Labels, "set-label", "set a label in form key:value for all test executions in the run")
}

type SmartTestGet struct {
	*SmartTest
}

type SmartTestList struct {
	*SmartTest
	TestName       string
	RunID          string
	Sandbox        string
	Repo           string
	RepoPath       string
	RepoCommitSHA  string
	ExecutionPhase string
	Labels         []string
}

func (c *SmartTestList) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&c.TestName, "test-name", "", "Filter test executions by test name")
	cmd.Flags().StringVar(&c.RunID, "run-id", "", "Filter test executions by run ID")
	cmd.Flags().StringVar(&c.Sandbox, "sandbox", "", "Filter test executions by sandbox name")
	cmd.Flags().StringVar(&c.Repo, "repo", "", "Filter test executions by repository name")
	cmd.Flags().StringVar(&c.RepoPath, "repo-path", "", "Filter test executions by repository path")
	cmd.Flags().StringVar(&c.RepoCommitSHA, "repo-commit-sha", "", "Filter test executions by repository commit SHA")
	cmd.Flags().StringVar(&c.ExecutionPhase, "phase", "", "Filter test executions by phase (one of 'pending', 'in_progress', 'succeeded', 'canceled' or 'failed')")
	cmd.Flags().StringArrayVar(&c.Labels, "label", []string{}, "Filter test executions by label in the format key:value (can be specified multiple times)")
}

type SmartTestCancel struct {
	*SmartTest
	RunID string
}

func (c *SmartTestCancel) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&c.RunID, "run-id", "", "Cancel all test executions of this run ID")
}
