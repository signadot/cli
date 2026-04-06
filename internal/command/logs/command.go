package logs

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/signadot/cli/internal/config"
	sdkprint "github.com/signadot/cli/internal/print"
	"github.com/signadot/go-sdk/client"
	"github.com/signadot/go-sdk/client/job_logs"
	planexeclogs "github.com/signadot/go-sdk/client/plan_execution_logs"
	"github.com/signadot/go-sdk/utils"
	"github.com/spf13/cobra"
)

func New(api *config.API) *cobra.Command {
	cfg := &config.Logs{API: api}

	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Display job or plan execution logs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return showLogs(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr(), cfg)
		},
	}

	cfg.AddFlags(cmd)
	return cmd
}

func showLogs(ctx context.Context, outW, errW io.Writer, cfg *config.Logs) error {
	if cfg.Job == "" && cfg.Plan == "" {
		return fmt.Errorf("must specify --job or --plan")
	}
	if cfg.Job != "" && cfg.Plan != "" {
		return fmt.Errorf("--job and --plan are mutually exclusive")
	}

	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}

	if cfg.Plan != "" {
		return showPlanLogs(ctx, outW, cfg)
	}

	var w io.Writer
	switch cfg.Stream {
	case utils.LogTypeStderr:
		w = errW
	default:
		w = outW
	}

	_, err := ShowLogs(ctx, cfg.API, w, cfg.Job, cfg.Stream, "", int(cfg.TailLines))
	return err
}

func showPlanLogs(ctx context.Context, out io.Writer, cfg *config.Logs) error {
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
				if cfg.Step != "" {
					params := planexeclogs.NewStreamPlanExecutionStepLogsParams().
						WithContext(ctx).
						WithTimeout(0).
						WithOrgName(cfg.Org).
						WithExecutionID(cfg.Plan).
						WithStepID(cfg.Step).
						WithStream(cfg.Stream)
					if cfg.TailLines > 0 {
						tl := int64(cfg.TailLines)
						params.WithTailLines(&tl)
					}
					_, err = c.PlanExecutionLogs.StreamPlanExecutionStepLogs(params, nil, writer)
				} else {
					params := planexeclogs.NewStreamPlanExecutionLogsParams().
						WithContext(ctx).
						WithTimeout(0).
						WithOrgName(cfg.Org).
						WithExecutionID(cfg.Plan)
					if cfg.TailLines > 0 {
						tl := int64(cfg.TailLines)
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

func ShowLogs(ctx context.Context, cfg *config.API, out io.Writer, jobName, stream, cursor string, tailLines int) (string, error) {
	params := job_logs.
		NewStreamJobAttemptLogsParams().
		WithContext(ctx).
		WithTimeout(0).
		WithOrgName(cfg.Org).
		WithJobName(jobName).
		WithJobAttempt(0).
		WithType(&stream)

	if tailLines > 0 {
		taillines := int64(tailLines)
		params.WithTailLines(&taillines)
	}

	if cursor != "" {
		params.WithCursor(&cursor)
	}

	// create a pipe for consuming the SSE stream
	reader, writer := io.Pipe()

	// create a custom transport to treat text/event-stream as a byte stream
	transportCfg := cfg.GetBaseTransport()
	transportCfg.Consumers = map[string]runtime.Consumer{
		"text/event-stream": runtime.ByteStreamConsumer(),
	}

	var lastCursor string

	err := cfg.APIClientWithCustomTransport(transportCfg,
		func(c *client.SignadotAPI) error {
			var err error
			errch := make(chan error)

			go func() {
				// parse the SSE stream
				lastCursor, err = sdkprint.ParseSSEStream(reader, out)
				if errors.Is(err, io.ErrClosedPipe) {
					err = nil // ignore ErrClosedPipe error
				}
				reader.Close()
				errch <- err
			}()

			go func() {
				// read the SSE stream
				_, err := c.JobLogs.StreamJobAttemptLogs(params, nil, writer)
				if errors.Is(err, io.ErrClosedPipe) {
					err = nil // ignore ErrClosedPipe error
				}
				writer.Close()
				errch <- err
			}()

			return errors.Join(<-errch, <-errch)
		})

	return lastCursor, err
}

