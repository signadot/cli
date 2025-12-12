package devbox

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/signadot/cli/internal/config"
	devboxpkg "github.com/signadot/cli/internal/devbox"
	"github.com/signadot/cli/internal/print"
	"github.com/signadot/go-sdk/client/devboxes"
	"github.com/signadot/go-sdk/models"
	"github.com/spf13/cobra"
)

func newList(devbox *config.Devbox) *cobra.Command {
	cfg := &config.DevboxList{Devbox: devbox}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List devboxes",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return list(cfg, cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}

	cfg.AddFlags(cmd)
	return cmd
}

func list(cfg *config.DevboxList, out io.Writer, errOut io.Writer) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	params := devboxes.NewGetDevboxesParams().
		WithContext(ctx).
		WithOrgName(cfg.Org)
	if cfg.ShowAll {
		all := "true"
		params = params.WithAll(&all)
	}
	resp, err := cfg.Client.Devboxes.GetDevboxes(params)
	if err != nil {
		return err
	}

	devboxes := resp.Payload

	// Read the current devbox ID from file
	currentDevboxID, err := devboxpkg.GetDefaultDevboxID()
	if err != nil {
		fmt.Fprintf(errOut, "Warning: Could not read devbox ID from ~/.signadot/.devbox-id: %v\n", err)
		// Treat as absent (currentDevboxID is already empty)
	}

	// Check if current devbox ID is in the list
	if currentDevboxID != "" {
		var currentDevboxInList bool
		for _, db := range devboxes {
			if db.ID == currentDevboxID {
				currentDevboxInList = true
				break
			}
		}
		if !currentDevboxInList {
			ddb, err := get(ctx, cfg, currentDevboxID)
			if err != nil {
				fmt.Fprintf(errOut, "Warning: Could not fetch devbox %s from ~/.signadot/.devbox-id: %v\n", currentDevboxID, err)
				// set current to unknown
				currentDevboxID = ""
			} else {
				devboxes = append(devboxes, ddb)
			}
		}
	}

	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return printDevboxTable(out, devboxes, currentDevboxID)
	case config.OutputFormatJSON:
		return print.RawJSON(out, devboxes)
	case config.OutputFormatYAML:
		return print.RawYAML(out, devboxes)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}

func get(ctx context.Context, cfg *config.DevboxList, id string) (*models.Devbox, error) {

	getParams := devboxes.NewGetDevboxParams().
		WithContext(ctx).
		WithOrgName(cfg.Org).
		WithDevboxID(id)
	getResp, err := cfg.Client.Devboxes.GetDevbox(getParams)
	if err != nil {
		return nil, err
	}
	if getResp.Code() != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch devbox %s: status %d %s", id, getResp.Code(), http.StatusText(getResp.Code()))
	}
	if getResp.Payload == nil {
		return nil, errors.New("unexpected response from server without body")
	}
	return getResp.Payload, nil
}
