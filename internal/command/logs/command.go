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
	"github.com/spf13/cobra"
)

func New(api *config.API) *cobra.Command {
	cfg := &config.Logs{API: api}

	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Display job logs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return display(cmd.Context(), cfg, cmd.OutOrStdout())
		},
	}

	cfg.AddFlags(cmd)
	return cmd
}

type event struct {
	Event string `sse:"event"`
	Data  string `sse:"data"`
}

type message struct {
	Message string `json:"message"`
	Cursor  string `json:"cursor"`
}

func display(ctx context.Context, cfg *config.Logs, out io.Writer) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}

	params := job_logs.
		NewStreamJobAttemptLogsParams().
		WithContext(ctx).
		WithTimeout(0).
		WithOrgName(cfg.Org).
		WithJobName(cfg.Job).
		WithJobAttempt(0).
		WithType(&cfg.Stream)

	if cfg.TailLines > 0 {
		taillines := int64(cfg.TailLines)
		params.WithTailLines(&taillines)
	}

	// create a pipe for consuming the SSE stream
	reader, writer := io.Pipe()

	return cfg.APIClientWithCustomTransport(
		cfg.OverrideTransportClientConsumers(map[string]runtime.Consumer{
			"text/event-stream": runtime.ByteStreamConsumer(),
		}),
		func(c *client.SignadotAPI) error {
			errch := make(chan error)

			go func() {
				// parse the SSE stream
				err := parseSSEStream(reader, out)
				reader.Close()
				errch <- err
			}()

			go func() {
				// read the SSE stream
				_, err := c.JobLogs.StreamJobAttemptLogs(params, nil, writer)
				writer.Close()
				errch <- err
			}()

			return errors.Join(<-errch, <-errch)
		})
}

func parseSSEStream(reader io.Reader, out io.Writer) error {
	scanner := sseparser.NewStreamScanner(reader)
	for {
		// Then, we call `UnmarshalNext`, and log each completion chunk, until we
		// encounter an error or reach the end of the stream.
		var e event
		_, err := scanner.UnmarshalNext(&e)
		if err != nil {
			if errors.Is(err, sseparser.ErrStreamEOF) {
				err = nil
			}
			return err
		}

		switch e.Event {
		case "message":
			var m message
			err = json.Unmarshal([]byte(e.Data), &m)
			if err != nil {
				return err
			}
			if m.Message == "" {
				continue
			}
			out.Write([]byte(m.Message))
		case "error":
			return errors.New(string(e.Data))
		case "signal":
			switch e.Data {
			case "EOF":
				return nil
			case "RESTART":
				out.Write([]byte("\n\n-------------------------------------------------------------------------------\n"))
				out.Write([]byte("WARNING: The job execution has been restarted...\n\n"))
			}
		}
	}
}
