package planexec

import (
	"context"
	"errors"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-openapi/runtime"
	"github.com/signadot/cli/internal/config"
	sdkprint "github.com/signadot/cli/internal/print"
	"github.com/signadot/go-sdk/client"
	planlogs "github.com/signadot/go-sdk/client/plan_execution_logs"
	"github.com/spf13/cobra"
)

func newLogs(exec *config.PlanExecution) *cobra.Command {
	cfg := &config.PlanExecLogs{PlanExecution: exec}

	cmd := &cobra.Command{
		Use:   "logs EXECUTION_ID [STEP_ID]",
		Short: "Stream plan execution logs",
		Long: `Stream logs for a plan execution.

Without a step ID, streams aggregated logs for all steps.
With a step ID, streams logs for that specific step.`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return streamLogs(cfg, cmd.OutOrStdout(), args)
		},
	}

	cfg.AddFlags(cmd)
	return cmd
}

// ShowPlanLogs streams plan execution logs. Exported for use by signadot logs --plan.
func ShowPlanLogs(ctx context.Context, cfg *config.API, out io.Writer, execID, stepID, stream string, tailLines int) error {
	transportCfg := cfg.GetBaseTransport()
	transportCfg.Consumers = map[string]runtime.Consumer{
		"text/event-stream": runtime.ByteStreamConsumer(),
	}

	return cfg.APIClientWithCustomTransport(transportCfg,
		func(c *client.SignadotAPI) error {
			reader, writer := io.Pipe()

			errch := make(chan error, 2)

			go func() {
				_, err := sdkprint.ParseSSEStream(reader, out)
				if errors.Is(err, io.ErrClosedPipe) {
					err = nil
				}
				reader.Close()
				errch <- err
			}()

			go func() {
				var err error
				if stepID != "" {
					params := planlogs.NewStreamPlanExecutionStepLogsParams().
						WithContext(ctx).
						WithTimeout(0).
						WithOrgName(cfg.Org).
						WithExecutionID(execID).
						WithStepID(stepID).
						WithStream(stream)
					if tailLines > 0 {
						tl := int64(tailLines)
						params.WithTailLines(&tl)
					}
					_, err = c.PlanExecutionLogs.StreamPlanExecutionStepLogs(params, nil, writer)
				} else {
					params := planlogs.NewStreamPlanExecutionLogsParams().
						WithContext(ctx).
						WithTimeout(0).
						WithOrgName(cfg.Org).
						WithExecutionID(execID)
					if tailLines > 0 {
						tl := int64(tailLines)
						params.WithTailLines(&tl)
					}
					_, err = c.PlanExecutionLogs.StreamPlanExecutionLogs(params, nil, writer)
				}
				if errors.Is(err, io.ErrClosedPipe) {
					err = nil
				}
				writer.Close()
				errch <- err
			}()

			return errors.Join(<-errch, <-errch)
		})
}

func streamLogs(cfg *config.PlanExecLogs, out io.Writer, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer cancel()

	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}

	execID := args[0]
	stepID := ""
	if len(args) > 1 {
		stepID = args[1]
	}

	return ShowPlanLogs(ctx, cfg.API, out, execID, stepID, cfg.Stream, int(cfg.TailLines))
}
