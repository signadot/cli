package traffic

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/trafficwatch"
	"github.com/signadot/go-sdk/client/sandboxes"
	"github.com/spf13/cobra"
)

func newWatch(cfg *config.Traffic) *cobra.Command {
	twCfg := &config.TrafficWatch{
		Traffic: cfg,
	}
	cmd := &cobra.Command{
		Use:   "watch --sandbox SANDBOX [ --short | --headers-only  ]",
		Short: "watch sandbox traffic",
		Args:  cobra.NoArgs,
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
	routingKey := resp.Payload.RoutingKey
	log := getTerminalLogger(cfg, wErr)
	if !cfg.Short {
		if err := setupToDir(cfg.ToDir); err != nil {
			return err
		}
	}

	tw, err := trafficwatch.GetTrafficWatch(context.Background(), cfg, log, routingKey)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()

	if cfg.Short {
		return trafficwatch.ConsumeShort(ctx, log, tw, w)
	}
	return trafficwatch.ConsumeToDir(ctx, log, cfg, tw, w)
}

func getTerminalLogger(cfg *config.TrafficWatch, w io.Writer) *slog.Logger {
	logLevel := slog.LevelInfo
	if cfg.Debug {
		logLevel = slog.LevelDebug
	}
	log := slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{
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
