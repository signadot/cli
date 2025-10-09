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
	"github.com/signadot/cli/internal/spinner"
	"github.com/signadot/cli/internal/trafficwatch"
	"github.com/signadot/go-sdk/client/sandboxes"
	"github.com/signadot/go-sdk/models"
	twapi "github.com/signadot/libconnect/common/trafficwatch"
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
	unedit, err := ensureHasTrafficWatchClientMW(cfg, w, resp.Payload)
	if err != nil {
		return err
	}
	// NOTE we should keep the single 'retErr' from here down
	var retErr error
	defer func() {
		retErr = errors.Join(retErr, unedit())
	}()
	if retErr = waitSandboxReady(cfg, w); retErr != nil {
		return retErr
	}
	routingKey := resp.Payload.RoutingKey
	log := getTerminalLogger(cfg, wErr)
	if !cfg.Short {
		if err := setupToDir(cfg.ToDir); err != nil {
			return err
		}
	}

	var tw *twapi.TrafficWatch
	tw, retErr = trafficwatch.GetTrafficWatch(context.Background(), cfg, log, routingKey)
	if retErr != nil {
		return retErr
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()
	if cfg.Short {
		return trafficwatch.ConsumeShort(ctx, log, tw, w)
	}
	return trafficwatch.ConsumeToDir(ctx, log, cfg, tw, w)
}

func ensureHasTrafficWatchClientMW(cfg *config.TrafficWatch, w io.Writer, sb *models.Sandbox) (func() error, error) {
	for _, mw := range sb.Spec.Middleware {
		if mw.Name == "trafficwatch-client" {
			if cfg.Debug {
				fmt.Fprintf(w, "sandbox already has trafficwatch middleware\n")
			}
			return func() error { return nil }, nil
		}
	}

	if cfg.Debug {
		fmt.Fprintf(w, "adding trafficwatch-client middleware to sandbox\n")
	}

	sb.Spec.Middleware = append(sb.Spec.Middleware,
		&models.SandboxesMiddleware{
			Name: "trafficwatch-client",
			Match: []*models.SandboxesMiddlewareMatch{
				{
					Workload: "*",
				},
			},
		})
	if sb.Spec.Labels == nil {
		sb.Spec.Labels = map[string]string{}
	}
	sb.Spec.Labels["instrumentation.signadot.com/add-trafficwatch-client"] = time.Now().Format(time.RFC3339)
	params := sandboxes.NewApplySandboxParams().
		WithOrgName(cfg.Org).WithSandboxName(sb.Name).WithData(sb)
	_, err := cfg.Client.Sandboxes.ApplySandbox(params, nil)
	if err != nil {
		return func() error { return nil }, err
	}
	return uneditFunc(cfg, w), nil
}

func uneditFunc(cfg *config.TrafficWatch, w io.Writer) func() error {
	return func() error {
		params := sandboxes.NewGetSandboxParams().
			WithOrgName(cfg.Org).WithSandboxName(cfg.Sandbox)
		resp, err := cfg.Client.Sandboxes.GetSandbox(params, nil)
		if err != nil {
			return err
		}
		sb := resp.Payload
		if x := sb.Spec.Labels["instrumentation.signadot.com/add-trafficwatch-client"]; x == "" {
			return nil
		}
		delete(sb.Spec.Labels, "instrumentation.signadot.com/add-trafficwatch-client")
		j := 0
		for _, sbm := range sb.Spec.Middleware {
			if sbm.Name == "trafficwatch-client" {
				if len(sbm.Args) == 0 {
					if len(sbm.Match) == 1 {
						if sbm.Match[0].Workload == "*" {
							continue
						}
					}
				}
			}
			sb.Spec.Middleware[j] = sbm
			j++
		}
		sb.Spec.Middleware = sb.Spec.Middleware[:j]
		applyParams := sandboxes.NewApplySandboxParams().
			WithOrgName(cfg.Org).WithSandboxName(cfg.Sandbox).WithData(sb)
		_, err = cfg.Client.Sandboxes.ApplySandbox(applyParams, nil)
		return err
	}
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

func waitSandboxReady(cfg *config.TrafficWatch, w io.Writer) error {
	fmt.Fprintf(w, "Waiting (up to --wait-timeout=%v) for sandbox to be ready...\n", cfg.WaitTimeout)

	params := sandboxes.NewGetSandboxParams().
		WithOrgName(cfg.Org).
		WithSandboxName(cfg.Sandbox)

	spin := spinner.Start(w, "Sandbox status")
	defer spin.Stop()

	retry := poll.
		NewPoll().
		WithTimeout(cfg.WaitTimeout)
	var failedErr error
	err := retry.Until(func() bool {
		result, err := cfg.Client.Sandboxes.GetSandbox(params, nil)
		if err != nil {
			// Keep retrying in case it's a transient error.
			spin.Messagef("error: %v", err)
			return false
		}
		sb := result.Payload
		if !sb.Status.Ready {
			if sb.Status.Reason == "ResourceFailed" {
				failedErr = errors.New(sb.Status.Message)
				return true
			}
			spin.Messagef("Not Ready: %s", sb.Status.Message)
			return false
		}
		spin.StopMessagef("Ready: %s", sb.Status.Message)
		return true
	})
	if failedErr != nil {
		spin.StopFail()
		return failedErr
	}
	if err != nil {
		spin.StopFail()
		return err
	}
	return nil
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
