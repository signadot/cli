package logs

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"strconv"

	"github.com/r3labs/sse/v2"
	"github.com/signadot/cli/internal/config"
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

type log struct {
	Message string `json:"message"`
}

func display(ctx context.Context, cfg *config.Logs, out io.Writer) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	u, err := url.Parse(cfg.APIURL)
	if err != nil {
		return err
	}
	u.Path = "/api/v2/orgs/" + cfg.Org + "/jobs/" + cfg.Job + "/attempts/0/logs/stream"
	u.RawQuery = "type=" + cfg.Stream
	if cfg.TailLines > 0 {
		u.RawQuery += "&tailLines=" + strconv.FormatUint(uint64(cfg.TailLines), 10)
	}

	events := make(chan *sse.Event)

	client := sse.NewClient(u.String())
	client.Headers = map[string]string{
		"Signadot-Api-Key": cfg.ApiKey,
	}

	err = client.SubscribeChan("", events)
	if err != nil {
		return err
	}

	for {
		select {
		case event, ok := <-events:
			if !ok {
				return nil
			}

			switch string(event.Event) {
			case "message":
				var log log
				if err := json.Unmarshal(event.Data, &log); err != nil {
					return err
				}
				out.Write([]byte(log.Message))

			case "error":
				return errors.New(string(event.Data))

			case "signal":
				switch string(event.Data) {
				case "EOF":
					return nil
				case "RESTART":
					out.Write([]byte("\n\n--------------------------------------------------"))
				default:
					return nil
				}
			}

		case <-ctx.Done():
			return nil
		}
	}
}
