package logs

import (
	"context"
	"encoding/json"
	"errors"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/jclem/sseparser"
	"github.com/signadot/cli/internal/config"
	"github.com/signadot/go-sdk/client"
	"github.com/signadot/go-sdk/client/job_logs"
	"github.com/signadot/go-sdk/utils"
	"github.com/spf13/cobra"
)

func New(api *config.API) *cobra.Command {
	cfg := &config.Logs{API: api}

	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Display job logs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return showLogs(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr(), cfg)
		},
	}

	cfg.AddFlags(cmd)
	return cmd
}

func showLogs(ctx context.Context, outW, errW io.Writer, cfg *config.Logs) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
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

type event struct {
	Event string `sse:"event"`
	Data  string `sse:"data"`
}

type message struct {
	Message string `json:"message"`
	Cursor  string `json:"cursor"`
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
				lastCursor, err = parseSSEStream(reader, out)
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

func parseSSEStream(reader io.Reader, out io.Writer) (string, error) {
	scanner := sseparser.NewStreamScanner(reader)
	var lastCursor string

	for {
		// Then, we call `UnmarshalNext`, and log each completion chunk, until we
		// encounter an error or reach the end of the stream.
		var e event
		_, err := scanner.UnmarshalNext(&e)
		if err != nil {
			if errors.Is(err, sseparser.ErrStreamEOF) {
				err = nil
			}
			return lastCursor, err
		}

		switch e.Event {
		case "message":
			var m message
			err = json.Unmarshal([]byte(e.Data), &m)
			if err != nil {
				return lastCursor, err
			}
			if m.Message == "" {
				continue
			}
			out.Write([]byte(m.Message))

			lastCursor = m.Cursor
		case "error":
			return lastCursor, errors.New(string(e.Data))
		case "signal":
			switch e.Data {
			case "EOF":
				return lastCursor, nil
			case "RESTART":
				out.Write([]byte("\n\n-------------------------------------------------------------------------------\n"))
				out.Write([]byte("WARNING: The job execution has been restarted...\n\n"))
			}
		}
	}
}
