package traffic

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/poll"
	"github.com/signadot/cli/internal/trafficwatch"
	"github.com/signadot/go-sdk/client/sandboxes"
	twapi "github.com/signadot/libconnect/common/trafficwatch"
	"github.com/spf13/cobra"
)

func newWatch(cfg *config.Traffic) *cobra.Command {
	twCfg := &config.TrafficWatch{
		Traffic: cfg,
	}
	cmd := &cobra.Command{
		Use:   "watch --sandbox SANDBOX [ --short | --headers-only  ]",
		Short: `watches sandbox traffic`,
		Long: `watch
Provide a sandbox with --sandbox and watch its traffic.  Console logging
is directed to stderr and a json stream (or yaml sequence of documents) describing 
requests received by the sandbox is directed to stdout.

With --short, watch only outputs the request descriptions.

Without --short, an output directory should be specified.  That directory
will be populated with subdirectories named middleware request id.  Each
subdirectory will contain the files

- meta.json (or .yaml)
- request
- response

The request contains either the request protocol line and headers, or also in
addition the body.  Run watch with --headers-only to skip the bodies.
`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return watch(twCfg, cmd.OutOrStdout(), cmd.ErrOrStderr(), args)
		},
	}
	twCfg.AddFlags(cmd)
	return cmd
}

func watch(cfg *config.TrafficWatch, w, wErr io.Writer, args []string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	if cfg.ToDir == "" && !cfg.Short {
		return fmt.Errorf("must specify output directory when running without --short")
	}
	if cfg.Sandbox == "" {
		return fmt.Errorf("must specify sandbox")
	}
	if cfg.Short && cfg.HeadersOnly {
		return fmt.Errorf("only one of --short or --headers-only can be provided")
	}
	params := sandboxes.NewGetSandboxParams().
		WithOrgName(cfg.Org).WithSandboxName(cfg.Sandbox)
	resp, err := cfg.Client.Sandboxes.GetSandbox(params, nil)
	if err != nil {
		return err
	}
	unedit, err := ensureHasTrafficWatchClientMW(cfg, w, resp.Payload)
	if err != nil {
		return err
	}
	// NOTE we should keep the single 'retErr' from here down
	var retErr error
	defer func() {
		retErr = errors.Join(retErr, unedit())
	}()
	if retErr = waitSandboxReady(cfg, wErr); retErr != nil {
		return retErr
	}
	routingKey := resp.Payload.RoutingKey
	log := getTerminalLogger(cfg, wErr)
	if !cfg.Short {
		if retErr = setupToDir(cfg.ToDir); retErr != nil {
			return retErr
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()
	readiness := poll.NewPoll().Readiness(ctx, 5*time.Second, func() (ready bool, warn, fatal error) {
		return ckReady(cfg)
	})
	defer readiness.Stop()

	var tw *twapi.TrafficWatch
	tw, retErr = trafficwatch.GetTrafficWatch(context.Background(), cfg, log, routingKey)
	if err != nil {
		return err
	}
	go readyLoop(ctx, log, tw, readiness)
	if cfg.Short {
		retErr = trafficwatch.ConsumeShort(ctx, log, tw, w)
	} else {
		retErr = trafficwatch.ConsumeToDir(ctx, log, cfg, tw, w)
	}
	return retErr
}

func getTerminalLogger(cfg *config.TrafficWatch, w io.Writer) *slog.Logger {
	logLevel := slog.LevelInfo
	if cfg.Debug {
		logLevel = slog.LevelDebug
	}
	log := slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: logLevel,
	}))
	return log
}

func setupToDir(toDir string) error {
	_, err := os.Stat(toDir)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err != nil {
		return os.Mkdir(toDir, 0755)
	}
	return nil
}
