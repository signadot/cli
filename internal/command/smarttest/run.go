package smarttest

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/repoconfig"
	clusters "github.com/signadot/go-sdk/client/cluster"
	routegroups "github.com/signadot/go-sdk/client/route_groups"
	"github.com/signadot/go-sdk/client/sandboxes"
	"github.com/signadot/go-sdk/client/test_executions"
	"github.com/signadot/go-sdk/models"
	"github.com/spf13/cobra"
)

func newRun(tConfig *config.SmartTest) *cobra.Command {
	cfg := &config.SmartTestRun{
		SmartTest: tConfig,
	}
	cmd := &cobra.Command{
		Use:   "run <name>",
		Short: "Run tests",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd.Context(), cfg, cmd.OutOrStdout(), cmd.ErrOrStderr(), args)
		},
	}
	cfg.AddFlags(cmd)
	return cmd
}

func run(ctx context.Context, cfg *config.SmartTestRun, wOut, wErr io.Writer,
	args []string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	if err := validateRun(cfg); err != nil {
		return err
	}

	// create a test finder
	tf, err := repoconfig.NewTestFinder(cfg.Directory)
	if err != nil {
		return err
	}

	// find tests
	testFiles, err := tf.FindTestFiles()
	if err != nil {
		return err
	}
	if len(testFiles) == 0 {
		return errors.New("could not find any test")
	}

	// create a run ID
	runID := repoconfig.GenerateRunID()

	// get the git repo (if any)
	gitRepo := tf.GetGitRepo()

	// trigger test executions
	err = triggerTests(cfg, runID, gitRepo, testFiles)
	if err != nil {
		return err
	}

	var out *defaultRunOutput
	if cfg.OutputFormat == config.OutputFormatDefault {
		// create an output handler
		out = newDefaultRunOutput(cfg, wOut, runID)
		out.start()
	}

	var txs []*models.TestExecution
	if !cfg.NoWait {
		// wait until all test execution have completed
		txs, err = waitForTests(ctx, cfg, runID, out)
		if err != nil {
			return fmt.Errorf("error waiting for tests: %w", err)
		}
	} else {
		// get tests executions
		txs, err = getTestExecutionsForRunID(ctx, cfg.SmartTest, runID)
		if err != nil {
			return err
		}
	}

	if out != nil {
		// render the latest status of test executions
		out.renderTestXsTable(txs, "")

		if !cfg.NoWait {
			// render the test executions summary
			out.renderTestXsSummary(txs)
		}
		return nil
	}

	// render the structured output
	return structuredOutput(cfg, wOut, runID, txs)
}

func validateRun(cfg *config.SmartTestRun) error {
	count := 0
	if cfg.Cluster != "" {
		count++
	}
	if cfg.Sandbox != "" {
		count++
	}
	if cfg.RouteGroup != "" {
		count++
	}

	if count == 0 {
		return fmt.Errorf("you must specify one of '--cluster', '--sandbox' or '--route-group'")
	}
	if count > 1 {
		return fmt.Errorf("only one of '--cluster', '--sandbox' or '--route-group' should be specified")
	}

	// load the corresponding entity from the API
	if cfg.Sandbox != "" {
		params := sandboxes.NewGetSandboxParams().
			WithOrgName(cfg.Org).WithSandboxName(cfg.Sandbox)
		resp, err := cfg.Client.Sandboxes.GetSandbox(params, nil)
		if err != nil {
			return fmt.Errorf("failed to load sandbox %q: %v", cfg.Sandbox, err)
		}
		// store the cluster for later use
		cfg.Cluster = *resp.Payload.Spec.Cluster
	} else if cfg.RouteGroup != "" {
		params := routegroups.NewGetRoutegroupParams().
			WithOrgName(cfg.Org).WithRoutegroupName(cfg.RouteGroup)
		resp, err := cfg.Client.RouteGroups.GetRoutegroup(params, nil)
		if err != nil {
			return fmt.Errorf("failed to load routegroup %q: %v", cfg.RouteGroup, err)
		}
		// store the cluster for later use
		cfg.Cluster = resp.Payload.Spec.Cluster
	} else {
		// validate the cluster exists
		params := clusters.NewGetClusterParams().
			WithOrgName(cfg.Org).WithClusterName(cfg.Cluster)
		if _, err := cfg.Client.Cluster.GetCluster(params, nil); err != nil {
			return fmt.Errorf("failed to load cluster %q: %v", cfg.Cluster, err)
		}
	}

	return nil
}

func triggerTests(cfg *config.SmartTestRun, runID string,
	gitRepo *repoconfig.GitRepo, testFiles []repoconfig.TestFile) error {
	// define the execution context (common for all tests)
	ec := &models.TestExecutionContext{
		Cluster: cfg.Cluster,
		Publish: cfg.Publish,
	}
	if cfg.Sandbox != "" {
		ec.Routing = &models.JobRoutingContext{
			Sandbox: cfg.Sandbox,
		}
	} else if cfg.RouteGroup != "" {
		ec.Routing = &models.JobRoutingContext{
			Routegroup: cfg.RouteGroup,
		}
	}

	// define the common parts fields of the embedded spec
	spec := &models.ExecutionSpec{
		RunID: runID,
	}
	if gitRepo != nil {
		spec.Repo = gitRepo.Repo
		spec.Branch = gitRepo.Branch
		spec.CommitSHA = gitRepo.CommitSHA
	}

	for _, tf := range testFiles {
		if gitRepo != nil {
			// define the repo path
			repoPath, err := repoconfig.GetRelativePathFromGitRoot(gitRepo.Path, tf.Path)
			if err != nil {
				return err
			}
			spec.Path = repoPath
		}
		// define the test name
		spec.TestName = tf.Name
		// define the labels
		spec.Labels = tf.Labels
		for k, v := range cfg.Labels {
			spec.Labels[k] = v
		}
		// define the script
		scriptContent, err := os.ReadFile(tf.Path)
		if err != nil {
			return fmt.Errorf("failed to read test file %q: %w", tf.Path, err)
		}
		spec.Script = string(scriptContent)

		params := test_executions.NewCreateTestExecutionParams().
			WithOrgName(cfg.Org).
			WithData(&models.TestExecution{
				Spec: &models.TestExecutionSpec{
					EmbeddedSpec:     spec,
					ExecutionContext: ec,
				},
			})
		_, err = cfg.Client.TestExecutions.CreateTestExecution(params, nil)
		if err != nil {
			return fmt.Errorf("could not create test execution for %q: %w", tf.Path, err)
		}
	}

	return nil
}

func waitForTests(ctx context.Context, cfg *config.SmartTestRun,
	runID string, out *defaultRunOutput) ([]*models.TestExecution, error) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	var cancel context.CancelFunc
	if cfg.Timeout != 0 {
		ctx, cancel = context.WithTimeout(ctx, cfg.Timeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	if out != nil {
		// update test executions output
		go out.updateTestXTable(ctx)
	}

	for {
		// get all test executions
		txs, err := getTestExecutionsForRunID(ctx, cfg.SmartTest, runID)
		if err != nil {
			return nil, err
		}

		// update test executions in output manager
		if out != nil {
			out.setTestXs(txs)
		}

		// define if all tests have completed
		isComplete := true
		for _, tx := range txs {
			switch tx.Status.Phase {
			case "pending", "in_progress":
				isComplete = false
			}
		}
		if isComplete {
			return txs, nil
		}

		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func getTestExecutionsForRunID(ctx context.Context, cfg *config.SmartTest,
	runID string) ([]*models.TestExecution, error) {
	var (
		pageSize int64 = 100
		res      []*models.TestExecution
		cursor   *string
	)

	for {
		// prepare query parameters
		params := test_executions.NewQueryTestExecutionsParams().
			WithContext(ctx).
			WithOrgName(cfg.Org).
			WithRunID(&runID).
			WithPageSize(&pageSize)

		// add cursor if available for pagination
		if cursor != nil {
			params = params.WithCursor(cursor)
		}

		// query test executions
		result, err := cfg.Client.TestExecutions.QueryTestExecutions(params, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to query test executions: %w", err)
		}

		// add current page results to the collection
		for _, item := range result.Payload {
			tx := item.Execution
			if tx.Spec == nil || tx.Spec.EmbeddedSpec == nil {
				// this should never happen
				continue
			}
			res = append(res, tx)
		}

		// check if there are more pages
		if int64(len(result.Payload)) < pageSize {
			return res, nil
		}

		// define the next cursor
		cursor = &result.Payload[len(result.Payload)-1].Cursor
	}
}
