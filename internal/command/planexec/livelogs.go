package planexec

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/go-openapi/runtime"
	"github.com/signadot/cli/internal/config"
	"github.com/signadot/go-sdk/client"
	planlogs "github.com/signadot/go-sdk/client/plan_execution_logs"
	"github.com/signadot/go-sdk/transport"
)

// LiveLogOptions selects the live log stream of a plan execution.
type LiveLogOptions struct {
	StepID    string // empty for the aggregated stream of all steps
	Stream    string // stdout or stderr (step streams only)
	TailLines int    // initial number of lines; 0 for all
}

// liveLogRetryDelay is the delay before reconnecting a live log stream.
var liveLogRetryDelay = 2 * time.Second

// FollowLiveLogs streams the live logs of a plan execution, parsing the SSE
// stream with parse (which returns the last cursor it saw). Unlike a single
// stream, it survives the stream ending early: it reconnects, resuming from
// the last cursor, when
//   - the execution has not been dispatched to a runner yet (404),
//   - the server cut the stream after 5 minutes without output (504), or
//   - the connection failed,
//
// and returns when the stream ends normally, the execution is finished (the
// runner no longer holds its live logs), or ctx is done.
func FollowLiveLogs(ctx context.Context, cfg *config.API, execID string, opts LiveLogOptions,
	parse func(io.Reader) (string, error)) error {
	cursor := ""
	for {
		c, err := streamLiveLogs(ctx, cfg, execID, opts, cursor, parse)
		if c != "" {
			cursor = c
			opts.TailLines = 0 // resume from the cursor, don't tail again
		}
		if err == nil || ctx.Err() != nil {
			return nil
		}
		if !retryLiveLogs(ctx, cfg, execID, err) {
			return err
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(liveLogRetryDelay):
		}
	}
}

func retryLiveLogs(ctx context.Context, cfg *config.API, execID string, err error) bool {
	var apiErr *transport.APIError
	if !errors.As(err, &apiErr) {
		// connection errors, SSE error events
		return true
	}
	switch {
	case apiErr.Code == http.StatusNotFound:
		// not dispatched yet, or finished: live logs are gone
		terminal, _, gerr := execTerminalPhase(ctx, cfg, execID)
		return gerr == nil && !terminal
	case apiErr.Code >= 500:
		return true
	}
	return false
}

// streamLiveLogs runs a single live log stream from cursor, returning the
// last cursor seen.
func streamLiveLogs(ctx context.Context, cfg *config.API, execID string, opts LiveLogOptions, cursor string,
	parse func(io.Reader) (string, error)) (string, error) {
	transportCfg := cfg.GetBaseTransport()
	transportCfg.Consumers = map[string]runtime.Consumer{
		"text/event-stream": runtime.ByteStreamConsumer(),
	}

	var lastCursor string
	err := cfg.APIClientWithCustomTransport(transportCfg,
		func(c *client.SignadotAPI) error {
			reader, writer := io.Pipe()
			errch := make(chan error, 2)

			go func() {
				var err error
				lastCursor, err = parse(reader)
				if errors.Is(err, io.ErrClosedPipe) {
					err = nil
				}
				reader.Close()
				errch <- err
			}()

			go func() {
				var cur *string
				if cursor != "" {
					cur = &cursor
				}
				var tl *int64
				if opts.TailLines > 0 {
					n := int64(opts.TailLines)
					tl = &n
				}
				var err error
				if opts.StepID != "" {
					params := planlogs.NewStreamPlanExecutionStepLogsParams().
						WithContext(ctx).
						WithTimeout(0).
						WithOrgName(cfg.Org).
						WithExecutionID(execID).
						WithStepID(opts.StepID).
						WithStream(opts.Stream).
						WithCursor(cur).
						WithTailLines(tl)
					_, err = c.PlanExecutionLogs.StreamPlanExecutionStepLogs(params, nil, writer)
				} else {
					params := planlogs.NewStreamPlanExecutionLogsParams().
						WithContext(ctx).
						WithTimeout(0).
						WithOrgName(cfg.Org).
						WithExecutionID(execID).
						WithCursor(cur).
						WithTailLines(tl)
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
	return lastCursor, err
}
