package logs

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"

	"github.com/r3labs/sse/v2"
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func New(api *config.API) *cobra.Command {
	cfg := &config.Logs{API: api}

	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Inspect and manipulate artifact",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return list(cfg, cmd.OutOrStdout())
		},
	}

	cfg.AddFlags(cmd)

	return cmd
}

func list(cfg *config.Logs, out io.Writer) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	u, err := url.Parse(cfg.APIURL)
	if err != nil {
		return err
	}
	u.Path = "/api/v2/orgs/" + cfg.Org + "/jobs/" + cfg.Job + "/attempts/0/logs/stream"
	u.RawQuery = "type=" + cfg.Stream

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
		case event := <-events:
			type Log struct {
				Message string `json:"message"`
			}
			fmt.Printf("\tgot event %s\n", event.Event)

			switch string(event.Event) {
			case "message":
				var log Log
				if err := json.Unmarshal(event.Data, &log); err != nil {
					return err
				}

				fmt.Print(log.Message)
			case "error":
				return errors.New(string(event.Data))
			case "signal":

				switch string(event.Data) {
				case "EOF":
					fmt.Println()
					return nil
				case "RESTART":
					fmt.Println("\n\n-----------")
				default:
					return nil
				}
			}
		}
	}

}
