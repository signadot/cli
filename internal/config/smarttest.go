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
	File       string
	Labels     TestExecLabels
	Cluster    string
	Sandbox    string
	RouteGroup string
	Publish    bool
	Timeout    time.Duration
	NoWait     bool
}

type TestExecLabels map[string]string

func (rl TestExecLabels) String() string {
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
		fmt.Fprintf(res, "%s=%s", key, rl[key])
	}
	return res.String()
}

func (rl TestExecLabels) Set(v string) error {
	key, val, ok := strings.Cut(v, "=")
	if !ok {
		return fmt.Errorf("%q should be in form <key>=<value>", v)
	}
	rl[key] = val
	return nil
}

func (tl TestExecLabels) Type() string {
	return "labels"
}

func (tl TestExecLabels) ToQueryFilter() []string {
	if len(tl) == 0 {
		return nil
	}
	result := make([]string, 0, len(tl))
	for k, v := range tl {
		result = append(result, k+":"+v)
	}
	return result
}

// AddFlags adds the flags for the test run command
func (c *SmartTestRun) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&c.Directory, "directory", "d", "", "base directory for finding tests")
	cmd.Flags().StringVarP(&c.File, "file", "f", "", "smart test file to run")
	cmd.Flags().StringVar(&c.Cluster, "cluster", "", "cluster where to run tests")
	cmd.Flags().StringVar(&c.Sandbox, "sandbox", "", "sandbox where to run tests")
	cmd.Flags().StringVar(&c.RouteGroup, "route-group", "", "route group where to run tests")
	cmd.Flags().BoolVar(&c.Publish, "publish", false, "publish test results")
	cmd.Flags().DurationVar(&c.Timeout, "timeout", 0, "timeout when waiting for the tests to complete, if 0 is specified, no timeout will be applied (default 0)")
	cmd.Flags().BoolVar(&c.NoWait, "no-wait", false, "do not wait until the tests are completed")

	c.Labels = make(map[string]string)
	cmd.Flags().Var(&c.Labels, "set-label", "set a label in form key=value for all test executions in the run (can be specified multiple times)")
}

type SmartTestExec struct {
	*SmartTest
}

type SmartTestExecGet struct {
	*SmartTestExec
}

type SmartTestExecList struct {
	*SmartTestExec
	TestName       string
	RunID          string
	Sandbox        string
	Repo           string
	RepoPath       string
	RepoCommitSHA  string
	ExecutionPhase string
	Labels         TestExecLabels
}

func (c *SmartTestExecList) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&c.TestName, "test-name", "", "filter test executions by test name")
	cmd.Flags().StringVar(&c.RunID, "run-id", "", "filter test executions by run ID")
	cmd.Flags().StringVar(&c.Sandbox, "sandbox", "", "filter test executions by sandbox name")
	cmd.Flags().StringVar(&c.Repo, "repo", "", "filter test executions by repository name")
	cmd.Flags().StringVar(&c.RepoPath, "repo-path", "", "filter test executions by repository path")
	cmd.Flags().StringVar(&c.RepoCommitSHA, "repo-commit-sha", "", "filter test executions by repository commit SHA")
	cmd.Flags().StringVar(&c.ExecutionPhase, "phase", "", "filter test executions by phase (one of 'pending', 'in_progress', 'succeeded', 'canceled' or 'failed')")

	c.Labels = make(map[string]string)
	cmd.Flags().Var(&c.Labels, "label", "filter test executions by label in the format key=value (can be specified multiple times)")
}

type SmartTestExecCancel struct {
	*SmartTestExec
	RunID string
}

func (c *SmartTestExecCancel) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&c.RunID, "run-id", "", "cancel all test executions in the run")
}
