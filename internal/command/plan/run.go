package plan

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/go-openapi/runtime"
	"github.com/signadot/cli/internal/command/planexec"
	"github.com/signadot/cli/internal/config"
	sdkprint "github.com/signadot/cli/internal/print"
	"github.com/signadot/cli/internal/spinner"
	sdkclient "github.com/signadot/go-sdk/client"
	planexecs "github.com/signadot/go-sdk/client/plan_executions"
	plantags "github.com/signadot/go-sdk/client/plan_tags"
	"github.com/signadot/go-sdk/models"
	"github.com/spf13/cobra"
)

func newRun(plan *config.Plan) *cobra.Command {
	cfg := &config.PlanRun{Plan: plan}

	cmd := &cobra.Command{
		Use:   "run [PLAN_ID] [--tag TAG_NAME] [--param key=value ...]",
		Short: "Run a plan: create execution, wait for completion, print results",
		Long: `Creates an execution of a compiled plan and polls until completion.

Resolve the plan by ID (positional argument) or by tag name (--tag).
Exit codes: 0 = completed, 1 = failed, 2 = cancelled.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPlan(cfg, cmd.OutOrStdout(), cmd.ErrOrStderr(), args)
		},
	}

	cfg.AddFlags(cmd)
	return cmd
}

func runPlan(cfg *config.PlanRun, out, log io.Writer, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer cancel()

	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}

	// Resolve plan ID.
	planID, err := resolvePlanID(cfg, args)
	if err != nil {
		return err
	}

	// Build params.
	params := buildParams(cfg.Params)

	// Create execution.
	spec := &models.PlanExecutionSpec{
		PlanID: planID,
		Params: params,
	}
	createParams := planexecs.NewCreatePlanExecutionParams().
		WithContext(ctx).
		WithOrgName(cfg.Org).
		WithData(spec)
	createResp, err := cfg.Client.PlanExecutions.CreatePlanExecution(createParams, nil)
	if err != nil {
		return fmt.Errorf("creating execution: %w", err)
	}
	execID := createResp.Payload.ID
	fmt.Fprintf(log, "Created execution %s for plan %s\n", execID, planID)

	// Fire-and-forget mode.
	if !cfg.Wait {
		return writeRunOutput(cfg, out, createResp.Payload)
	}

	// Poll for completion.
	exec, err := pollExecution(ctx, cfg, log, execID)
	if err != nil {
		// On interrupt, try to cancel the execution.
		if errors.Is(err, context.Canceled) {
			fmt.Fprintf(log, "\nInterrupted, cancelling execution %s...\n", execID)
			cancelCtx, cancelCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancelCancel()
			cancelParams := planexecs.NewCancelPlanExecutionParams().
				WithContext(cancelCtx).
				WithOrgName(cfg.Org).
				WithPlanExecutionID(execID)
			cfg.Client.PlanExecutions.CancelPlanExecution(cancelParams, nil)
			os.Exit(2)
		}
		return err
	}

	// Export outputs if --output-dir specified.
	if cfg.OutputDir != "" {
		if err := exportOutputs(cfg, log, exec); err != nil {
			fmt.Fprintf(log, "Warning: output export failed: %v\n", err)
		}
	}

	// Print result and exit with appropriate code.
	if err := writeRunOutput(cfg, out, exec); err != nil {
		return err
	}

	switch exec.Status.Phase {
	case models.PlansExecutionPhaseFailed:
		os.Exit(1)
	case models.PlansExecutionPhaseCancelled:
		os.Exit(2)
	}
	return nil
}

func resolvePlanID(cfg *config.PlanRun, args []string) (string, error) {
	if cfg.Tag != "" && len(args) > 0 {
		return "", fmt.Errorf("specify either a plan ID argument or --tag, not both")
	}
	if cfg.Tag == "" && len(args) == 0 {
		return "", fmt.Errorf("specify a plan ID argument or --tag")
	}
	if cfg.Tag != "" {
		params := plantags.NewGetPlanTagParams().
			WithOrgName(cfg.Org).
			WithPlanTagName(cfg.Tag)
		resp, err := cfg.Client.PlanTags.GetPlanTag(params, nil)
		if err != nil {
			return "", fmt.Errorf("resolving tag %q: %w", cfg.Tag, err)
		}
		if resp.Payload.Spec == nil || resp.Payload.Spec.PlanID == "" {
			return "", fmt.Errorf("tag %q has no plan ID", cfg.Tag)
		}
		return resp.Payload.Spec.PlanID, nil
	}
	return args[0], nil
}

func buildParams(tplVals config.TemplateVals) map[string]any {
	if len(tplVals) == 0 {
		return nil
	}
	params := make(map[string]any, len(tplVals))
	for _, tv := range tplVals {
		// If value looks like JSON, pass through as-is.
		v := tv.Val
		if looksLikeJSON(v) {
			var raw json.RawMessage
			if json.Unmarshal([]byte(v), &raw) == nil {
				params[tv.Var] = raw
				continue
			}
		}
		params[tv.Var] = v
	}
	return params
}

func looksLikeJSON(s string) bool {
	if len(s) == 0 {
		return false
	}
	switch s[0] {
	case '{', '[', '"':
		return true
	}
	switch s {
	case "true", "false", "null":
		return true
	}
	// Check if it's a number.
	if s[0] == '-' || (s[0] >= '0' && s[0] <= '9') {
		var v json.RawMessage
		return json.Unmarshal([]byte(s), &v) == nil
	}
	return false
}

func isTerminal(phase models.PlansExecutionPhase) bool {
	switch phase {
	case models.PlansExecutionPhaseCompleted,
		models.PlansExecutionPhaseFailed,
		models.PlansExecutionPhaseCancelled:
		return true
	}
	return false
}

func pollExecution(ctx context.Context, cfg *config.PlanRun, log io.Writer, execID string) (*models.PlanExecution, error) {
	if cfg.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.Timeout)
		defer cancel()
	}

	spin := spinner.Start(log, "Execution")
	defer spin.Stop()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		params := planexecs.NewGetPlanExecutionParams().
			WithContext(ctx).
			WithOrgName(cfg.Org).
			WithPlanExecutionID(execID)
		resp, err := cfg.Client.PlanExecutions.GetPlanExecution(params, nil)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				spin.StopFail()
				return nil, err
			}
			spin.Messagef("error: %v", err)
		} else {
			ex := resp.Payload
			if isTerminal(ex.Status.Phase) {
				switch ex.Status.Phase {
				case models.PlansExecutionPhaseCompleted:
					spin.StopMessage(string(ex.Status.Phase))
				default:
					spin.StopFail()
				}
				return ex, nil
			}
			msg := string(ex.Status.Phase)
			if sc := ex.Status.StepCounts; sc != nil {
				total := sc.Init + sc.Waiting + sc.Running + sc.Completed + sc.Failed + sc.Skipped
				msg = fmt.Sprintf("%s (%d/%d steps completed)", msg, sc.Completed, total)
			}
			spin.Message(msg)
		}

		select {
		case <-ticker.C:
		case <-ctx.Done():
			spin.StopFail()
			return nil, ctx.Err()
		}
	}
}

func writeRunOutput(cfg *config.PlanRun, out io.Writer, exec *models.PlanExecution) error {
	switch cfg.OutputFormat {
	case config.OutputFormatJSON:
		return sdkprint.RawJSON(out, exec)
	case config.OutputFormatYAML:
		return sdkprint.RawYAML(out, exec)
	default:
		return planexec.PrintRunResult(out, exec)
	}
}

func exportOutputs(cfg *config.PlanRun, log io.Writer, exec *models.PlanExecution) error {
	if exec.Status == nil || len(exec.Status.Outputs) == 0 {
		return nil
	}

	if err := os.MkdirAll(cfg.OutputDir, 0o755); err != nil {
		return err
	}

	transportCfg := cfg.GetBaseTransport()
	transportCfg.OverrideConsumers = true
	transportCfg.Consumers = map[string]runtime.Consumer{
		"*/*": runtime.ByteStreamConsumer(),
	}

	return cfg.APIClientWithCustomTransport(transportCfg,
		func(c *sdkclient.SignadotAPI) error {
			for _, o := range exec.Status.Outputs {
				outPath := filepath.Join(cfg.OutputDir, o.Name)
				f, err := os.Create(outPath)
				if err != nil {
					return fmt.Errorf("creating %s: %w", outPath, err)
				}
				params := planexecs.NewGetPlanExecutionOutputParams().
					WithOrgName(cfg.Org).
					WithPlanExecutionID(exec.ID).
					WithStepOutputName(o.Name)
				_, _, err = c.PlanExecutions.GetPlanExecutionOutput(params, nil, f)
				f.Close()
				if err != nil {
					return fmt.Errorf("downloading %q: %w", o.Name, err)
				}
				fmt.Fprintf(log, "Exported %s\n", outPath)
			}
			return nil
		})
}
