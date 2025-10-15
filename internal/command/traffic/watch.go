package traffic

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/poll"
	"github.com/signadot/cli/internal/trafficwatch"
	"github.com/signadot/cli/internal/utils/system"
	"github.com/signadot/go-sdk/client/sandboxes"
	twapi "github.com/signadot/libconnect/common/trafficwatch"
	"github.com/spf13/cobra"
)

func newWatch(cfg *config.Traffic) *cobra.Command {
	twCfg := &config.TrafficWatch{
		Traffic: cfg,
	}
	defaultDir := filepath.Join(system.GetSignadotDirGeneric(), trafficwatch.DefaultDirRelative)
	cmd := &cobra.Command{
		Use:   "watch --sandbox SANDBOX [ --short | --headers-only  ]",
		Short: `watches sandbox traffic`,
		Long: fmt.Sprintf(`watch
Provide a sandbox with --sandbox and watch its traffic. 

With --short, watch only reports request activity. If --to specifies a file,
request activity is sent in a json (or yaml) stream to it.  Otherwise, no
stream is recorded.

Without --short, watch produces output in a directory that will be populated
with a meta.jsons (or .yamls) file and subdirectories named by middleware
request ids.

By default, this directory is either %s-json 
or %s-yaml, depending on the output format.

Each subdirectory in turn will contain the files

- meta.json (or .yaml)
- request
- response

The request (and response) contains the wire format

- the protocol line which is terminated with '\r\n'.
- the headers each terminated  by '\r\n'
- the separator '\r\n'
- the body, unless run with --headers-only
`, defaultDir, defaultDir),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return watch(twCfg, defaultDir, cmd.OutOrStdout(), cmd.ErrOrStderr(), args)
		},
	}
	twCfg.AddFlags(cmd)
	return cmd
}

func watch(cfg *config.TrafficWatch, defaultDir string, w, wErr io.Writer, args []string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	if cfg.Sandbox == "" {
		return fmt.Errorf("must specify sandbox")
	}
	if cfg.Short && cfg.HeadersOnly {
		return fmt.Errorf("only one of --short or --headers-only can be provided")
	}
	if !cfg.Short && cfg.OutputDir == "" {
		signadotDir, err := system.GetSignadotDir()
		if err != nil {
			return err
		}
		dirSuffix := trafficwatch.FormatSuffix(cfg)
		if dirSuffix != "" {
			dirSuffix = "-" + dirSuffix[1:]
		}
		relDir := trafficwatch.DefaultDirRelative + dirSuffix
		cfg.OutputDir = filepath.Join(signadotDir, relDir)
		if cfg.Clean {
			if err := os.RemoveAll(cfg.OutputDir); err != nil {
				return fmt.Errorf("unable to clean up %s: %w", cfg.OutputDir)
			}
		}
		fmt.Fprintf(w, "Traffic will be written to %s.\n", cfg.OutputDir)
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
	log := getTerminalLogger(cfg, w)
	if !cfg.Short {
		if retErr = setupToDir(cfg.OutputDir); retErr != nil {
			return retErr
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()
	var tw *twapi.TrafficWatch
	tw, retErr = trafficwatch.GetTrafficWatch(context.Background(), cfg, log, routingKey)
	if retErr != nil {
		return retErr
	}
	if !cfg.NoInstrument {
		readiness := poll.NewPoll().Readiness(ctx, 5*time.Second, func() (ready bool, warn, fatal error) {
			return ckReady(cfg)
		})
		defer readiness.Stop()
		go readyLoop(ctx, log, tw, readiness)
	}

	if cfg.Short {
		out := "<none>"
		if cfg.OutputDir != "" {
			out = cfg.OutputDir
		}
		log.Info("watching sandbox request activity", "watch-options", getExpectedOpts(cfg).String(), "output", out)
		retErr = trafficwatch.ConsumeShort(ctx, log, cfg, tw)
	} else {
		log.Info("watching sandbox request activity and content", "watch-options", getExpectedOpts(cfg).String(), "output-dir", cfg.OutputDir)
		retErr = trafficwatch.ConsumeToDir(ctx, log, cfg, tw)
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
		// remove timestamps
		ReplaceAttr: func(attrs []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			if a.Key == slog.LevelKey {
				if a.Value.String() == "INFO" {
					return slog.Attr{}
				}
			}
			return a
		},
	}))
	return log.With("sandbox", cfg.Sandbox)
}

func setupToDir(toDir string) error {
	_, err := os.Stat(toDir)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err != nil {
		return os.MkdirAll(toDir, 0755)
	}
	return nil
}
