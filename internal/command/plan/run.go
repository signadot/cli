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
	planlogs "github.com/signadot/go-sdk/client/plan_execution_logs"
	planexecs "github.com/signadot/go-sdk/client/plan_executions"
	sdkplans "github.com/signadot/go-sdk/client/plans"
	plantags "github.com/signadot/go-sdk/client/plan_tags"
	"github.com/signadot/go-sdk/models"
	"github.com/spf13/cobra"
)

func newRun(plan *config.Plan) *cobra.Command {
	cfg := &config.PlanRun{Plan: plan}

	cmd := &cobra.Command{
		Use:   "run [PLAN_ID] [--tag TAG_NAME] [--param key=value ...] [--param-secret param-name=secret-name ...]",
		Short: "Run a plan: create execution, wait for completion, print results",
		Long: `Creates an execution of a compiled plan and polls until completion.

Resolve the plan by ID (positional argument) or by tag name (--tag).
Use --attach to stream structured events (logs, outputs, result) to stdout.
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

	if cfg.Attach && cfg.OutputFormat == config.OutputFormatYAML {
		return fmt.Errorf("--attach does not support -o yaml; use -o json for structured output")
	}

	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}

	// Resolve and fetch plan.
	plan, err := resolvePlan(ctx, cfg, args)
	if err != nil {
		return err
	}
	planID := plan.ID
	planSpec := plan.Spec

	// Build params.
	params := buildParams(cfg.Params)
	if params == nil && (cfg.Sandbox != "" || cfg.RouteGroup != "") {
		params = make(map[string]any)
	}

	// Apply --sandbox / --route-group to params.
	if err := applyRoutingFlags(cfg, planSpec, params); err != nil {
		return err
	}

	secrets := buildSecrets(cfg.Secrets)
	for name := range secrets {
		if _, ok := params[name]; ok {
			return fmt.Errorf("param %q appears in both --param and --param-secret; specify only one", name)
		}
	}

	// Create execution.
	spec := &models.PlanExecutionSpec{
		PlanID:  planID,
		Cluster: cfg.Cluster,
		Params:  params,
		Secrets: secrets,
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
	// In --attach mode the "created" notice is emitted as a structured
	// event from inside attachExecution so it shares timestamping and
	// formatting with the rest of the stream. Otherwise print a plain
	// banner to stderr (only for default text output).
	if cfg.OutputFormat == config.OutputFormatDefault && !cfg.Attach {
		fmt.Fprintf(log, "Created execution %s for plan %s\n", execID, planID)
	}

	// Fire-and-forget mode.
	if !cfg.Wait {
		return writeRunOutput(cfg, out, createResp.Payload, planSpec)
	}

	// Wait for completion: attach streams structured events, otherwise poll with spinner.
	var exec *models.PlanExecution
	if cfg.Attach {
		exec, err = attachExecution(ctx, cfg, out, log, execID, planID)
	} else {
		exec, err = pollExecution(ctx, cfg, log, execID)
	}
	if err != nil {
		// On interrupt or timeout, try to cancel the execution.
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			fmt.Fprintf(log, "\nCancelling execution %s...\n", execID)
			cancelCtx, cancelCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancelCancel()
			cancelParams := planexecs.NewCancelPlanExecutionParams().
				WithContext(cancelCtx).
				WithOrgName(cfg.Org).
				WithExecutionID(execID)
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

	// In attach mode, events were already emitted to stdout. Just exit.
	if cfg.Attach {
		switch exec.Status.Phase {
		case models.PlansExecutionPhaseFailed:
			os.Exit(1)
		case models.PlansExecutionPhaseCancelled:
			os.Exit(2)
		}
		return nil
	}

	// Print result and exit with appropriate code.
	// On failure/cancellation, write details to stderr so stdout stays clean.
	switch exec.Status.Phase {
	case models.PlansExecutionPhaseFailed:
		if err := writeRunOutput(cfg, log, exec, planSpec); err != nil {
			fmt.Fprintf(log, "error rendering output: %v\n", err)
		}
		os.Exit(1)
	case models.PlansExecutionPhaseCancelled:
		if err := writeRunOutput(cfg, log, exec, planSpec); err != nil {
			fmt.Fprintf(log, "error rendering output: %v\n", err)
		}
		os.Exit(2)
	default:
		return writeRunOutput(cfg, out, exec, planSpec)
	}
	return nil
}

func resolvePlan(ctx context.Context, cfg *config.PlanRun, args []string) (*models.RunnablePlan, error) {
	if cfg.Tag != "" && len(args) > 0 {
		return nil, fmt.Errorf("specify either a plan ID argument or --tag, not both")
	}
	if cfg.Tag == "" && len(args) == 0 {
		return nil, fmt.Errorf("specify a plan ID argument or --tag")
	}

	if cfg.Tag != "" {
		tagParams := plantags.NewGetPlanTagParams().
			WithContext(ctx).
			WithOrgName(cfg.Org).
			WithPlanTagName(cfg.Tag)
		resp, err := cfg.Client.PlanTags.GetPlanTag(tagParams, nil)
		if err != nil {
			return nil, fmt.Errorf("resolving tag %q: %w", cfg.Tag, err)
		}
		if resp.Payload.Plan == nil {
			return nil, fmt.Errorf("tag %q has no plan", cfg.Tag)
		}
		return resp.Payload.Plan, nil
	}

	getParams := sdkplans.NewGetPlanParams().
		WithContext(ctx).
		WithOrgName(cfg.Org).
		WithPlanID(args[0])
	resp, err := cfg.Client.Plans.GetPlan(getParams, nil)
	if err != nil {
		return nil, fmt.Errorf("fetching plan: %w", err)
	}
	return resp.Payload, nil
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

func buildSecrets(tplVals config.TemplateVals) map[string]string {
	if len(tplVals) == 0 {
		return nil
	}
	secrets := make(map[string]string, len(tplVals))
	for _, tv := range tplVals {
		secrets[tv.Var] = tv.Val
	}
	return secrets
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

func applyRoutingFlags(cfg *config.PlanRun, planSpec *models.PlanSpec, params map[string]any) error {
	if cfg.Sandbox == "" && cfg.RouteGroup == "" {
		return nil
	}

	aff := planSpec.Cluster
	if aff == nil {
		if cfg.Sandbox != "" {
			return fmt.Errorf("plan does not accept a sandbox parameter")
		}
		return fmt.Errorf("plan does not accept a route group parameter")
	}

	if cfg.Sandbox != "" {
		if aff.FromAnyTarget != "" {
			raw, err := json.Marshal(models.RoutingTarget{Sandbox: cfg.Sandbox})
			if err != nil {
				return fmt.Errorf("marshalling routing target: %w", err)
			}
			params[aff.FromAnyTarget] = json.RawMessage(raw)
		} else if aff.FromSandbox != "" {
			params[aff.FromSandbox] = cfg.Sandbox
		} else {
			return fmt.Errorf("plan does not accept a sandbox parameter")
		}
	}

	if cfg.RouteGroup != "" {
		if aff.FromAnyTarget != "" {
			raw, err := json.Marshal(models.RoutingTarget{RouteGroup: cfg.RouteGroup})
			if err != nil {
				return fmt.Errorf("marshalling routing target: %w", err)
			}
			params[aff.FromAnyTarget] = json.RawMessage(raw)
		} else if aff.FromRouteGroup != "" {
			params[aff.FromRouteGroup] = cfg.RouteGroup
		} else {
			return fmt.Errorf("plan does not accept a route group parameter")
		}
	}

	return nil
}

// buildOutputEvent translates a plan-level PlanOutputStatus into the
// AttachEvent shape so an --attach text/JSON consumer can tell inline
// outputs (Kind=inline, Value set) from artifact outputs (Kind=artifact,
// Size/Ready/ContentType set) instead of seeing value=<nil> for the
// latter.
func buildOutputEvent(o *models.PlanOutputStatus) sdkprint.AttachEvent {
	evt := sdkprint.AttachEvent{
		Type: "output",
		Name: o.Name,
	}
	if o.StepRef != nil {
		evt.Step = o.StepRef.StepID
	}
	if o.Artifact != nil {
		evt.Kind = "artifact"
		evt.Size = o.Artifact.Size
		ready := o.Artifact.Ready
		evt.Ready = &ready
		if ct := o.Metadata["contentType"]; ct != "" {
			evt.ContentType = ct
		}
		evt.Error = o.Artifact.Error
	} else {
		evt.Kind = "inline"
		evt.Value = o.Value
	}
	return evt
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

	spinWriter := log
	if cfg.OutputFormat != config.OutputFormatDefault {
		spinWriter = io.Discard
	}
	spin := spinner.Start(spinWriter, "Execution")
	defer spin.Stop()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		params := planexecs.NewGetPlanExecutionParams().
			WithContext(ctx).
			WithOrgName(cfg.Org).
			WithExecutionID(execID)
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

func attachExecution(ctx context.Context, cfg *config.PlanRun, out, log io.Writer, execID, planID string) (*models.PlanExecution, error) {
	if cfg.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.Timeout)
		defer cancel()
	}

	jsonMode := cfg.OutputFormat == config.OutputFormatJSON
	aw := sdkprint.NewAttachWriter(out, jsonMode)

	// First event in the stream — same shape as the "Created execution"
	// stderr banner emitted in non-attach mode, so consumers piping
	// through jq see a consistent event sequence from creation to result.
	aw.Emit(sdkprint.AttachEvent{
		Type:   "created",
		ID:     execID,
		PlanID: planID,
	})

	// Stream aggregated logs in background, emitting structured events.
	logCtx, logCancel := context.WithCancel(ctx)
	defer logCancel()

	logDone := make(chan error, 1)
	go func() {
		transportCfg := cfg.GetBaseTransport()
		transportCfg.Consumers = map[string]runtime.Consumer{
			"text/event-stream": runtime.ByteStreamConsumer(),
		}
		err := cfg.APIClientWithCustomTransport(transportCfg,
			func(c *sdkclient.SignadotAPI) error {
				reader, writer := io.Pipe()
				errch := make(chan error, 2)

				go func() {
					_, err := sdkprint.ParseSSEAttach(reader, aw)
					if errors.Is(err, io.ErrClosedPipe) {
						err = nil
					}
					reader.Close()
					errch <- err
				}()

				go func() {
					params := planlogs.NewStreamPlanExecutionLogsParams().
						WithContext(logCtx).
						WithTimeout(0).
						WithOrgName(cfg.Org).
						WithExecutionID(execID)
					_, err := c.PlanExecutionLogs.StreamPlanExecutionLogs(params, nil, writer)
					if errors.Is(err, io.ErrClosedPipe) || errors.Is(err, context.Canceled) {
						err = nil
					}
					writer.Close()
					errch <- err
				}()

				return errors.Join(<-errch, <-errch)
			})
		logDone <- err
	}()

	// Poll for terminal phase.
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		params := planexecs.NewGetPlanExecutionParams().
			WithContext(ctx).
			WithOrgName(cfg.Org).
			WithExecutionID(execID)
		resp, err := cfg.Client.PlanExecutions.GetPlanExecution(params, nil)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				logCancel()
				<-logDone
				return nil, err
			}
		} else if isTerminal(resp.Payload.Status.Phase) {
			logCancel()
			<-logDone

			ex := resp.Payload
			// Emit output events for resolved plan-level outputs.
			if ex.Status != nil {
				for _, o := range ex.Status.Outputs {
					aw.Emit(buildOutputEvent(o))
				}
			}
			// Emit result event.
			resultEvent := sdkprint.AttachEvent{
				Type:  "result",
				ID:    ex.ID,
				Phase: string(ex.Status.Phase),
			}
			if ex.Status.Error != "" {
				resultEvent.Error = ex.Status.Error
			}
			aw.Emit(resultEvent)

			return ex, nil
		}

		select {
		case <-ticker.C:
		case <-ctx.Done():
			logCancel()
			<-logDone
			return nil, ctx.Err()
		}
	}
}

func writeRunOutput(cfg *config.PlanRun, out io.Writer, exec *models.PlanExecution, planSpec *models.PlanSpec) error {
	switch cfg.OutputFormat {
	case config.OutputFormatJSON:
		return sdkprint.RawJSON(out, exec)
	case config.OutputFormatYAML:
		return sdkprint.RawYAML(out, exec)
	default:
		return planexec.PrintRunResult(out, exec, planSpec)
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
					WithExecutionID(exec.ID).
					WithOutputName(o.Name)
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
