package traffic

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/poll"
	"github.com/signadot/cli/internal/trafficwatch"
	"github.com/signadot/cli/internal/tui"
	"github.com/signadot/cli/internal/utils"
	"github.com/signadot/cli/internal/utils/system"
	twapi "github.com/signadot/libconnect/common/trafficwatch"
	"github.com/spf13/cobra"
)

func newRecord(cfg *config.Traffic) *cobra.Command {
	twCfg := &config.TrafficWatch{
		Traffic: cfg,
	}
	defaultDir := filepath.Join(system.GetSignadotDirGeneric(), trafficwatch.DefaultDirRelative)
	cmd := &cobra.Command{
		Use:     "record --sandbox SANDBOX [ --short | --headers-only  ]",
		Aliases: []string{"r"},
		Short:   `records sandbox traffic`,
		Long: fmt.Sprintf(`record
Provide a sandbox with --sandbox and record its (incoming) traffic. 

With --short, record only reports request activity. If --output-file is
specified request activity is sent in a json (or yaml) stream to it.
Otherwise, no stream is recorded.

Without --short, record produces output in a directory that will be populated
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
			return record(cmd.Context(), twCfg, defaultDir,
				cmd.OutOrStdout(), cmd.ErrOrStderr(), args)
		},
	}
	twCfg.AddFlags(cmd)
	return cmd
}

func record(rootCtx context.Context, cfg *config.TrafficWatch, defaultDir string,
	w, wErr io.Writer, args []string) error {
	ctx, _ := signal.NotifyContext(rootCtx,
		os.Interrupt, syscall.SIGTERM, syscall.SIGTERM, syscall.SIGHUP)
	// set a timeout of 1h
	ctx, cancel := context.WithTimeout(ctx, time.Hour)
	defer cancel()

	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}

	// validations
	if cfg.Sandbox == "" {
		return fmt.Errorf("must specify sandbox")
	}
	if cfg.Short && cfg.HeadersOnly {
		return fmt.Errorf("only one of --short or --headers-only can be provided")
	}

	// define output dir
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
				return fmt.Errorf("unable to clean up %s: %w", cfg.OutputDir, err)
			}
		}
		fmt.Fprintf(w, "Traffic will be written to %s.\n", cfg.OutputDir)
	}

	// get the sandbox and ensure the trafficwatch middleware is present
	sb, err := utils.GetSandbox(ctx, cfg.API, cfg.Sandbox)
	if err != nil {
		return err
	}
	undo, err := ensureTrafficWatchMW(ctx, cfg, w, sb)
	if err != nil {
		return err
	}

	// NOTE we should keep the single 'retErr' from here down
	var retErr error
	defer func() {
		retErr = errors.Join(retErr, undo(rootCtx, w))
	}()

	if !cfg.NoInstrument {
		// wait until the sandbox is ready
		_, retErr = utils.WaitForSandboxReady(ctx, cfg.API, w, cfg.Sandbox, cfg.WaitTimeout)
		if retErr != nil {
			return retErr
		}
	}
	routingKey := sb.RoutingKey

	var logsFile string
	writer := w
	if cfg.TuiMode {
		f, err := os.CreateTemp("", "signadot-traffic-watch-*.log")
		if err != nil {
			return fmt.Errorf("error creating temp file: %w", err)
		}
		defer f.Close()
		writer = f
		logsFile = f.Name()
	}
	log := getTerminalLogger(cfg, writer)

	if !cfg.Short {
		if retErr = setupToDir(cfg.OutputDir); retErr != nil {
			return retErr
		}
	}

	// setup the traffic watch client
	var tw *twapi.TrafficWatch
	tw, retErr = trafficwatch.GetTrafficWatch(ctx, cfg, log, routingKey)
	if retErr != nil {
		return retErr
	}

	if !cfg.NoInstrument {
		// run the readiness loop
		readiness := poll.NewPoll().Readiness(ctx, 5*time.Second, func() (ready bool, warn, fatal error) {
			return ckReady(cfg)
		})
		defer readiness.Stop()
		go readyLoop(ctx, log, tw, readiness)
	}

	if cfg.TuiMode {
		go func() {
			if err := start(cfg, log, ctx, tw); err != nil {
				log.Error("error starting traffic watch", "error", err)
			}
		}()

		trafficWatch := tui.NewTrafficWatch(cfg.OutputDir, config.OutputFormatJSON, logsFile)
		if err := trafficWatch.Run(); err != nil {
			return err
		}
	} else {
		if err := start(cfg, log, ctx, tw); err != nil {
			return err
		}
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

func start(cfg *config.TrafficWatch, log *slog.Logger, ctx context.Context, tw *twapi.TrafficWatch) error {
	if cfg.Short {
		out := "<none>"
		if cfg.OutputFile != "" {
			out = cfg.OutputFile
		}
		log.Info("watching sandbox request activity", "watch-options", getExpectedOpts(cfg).String(), "output", out)
		return trafficwatch.ConsumeShort(ctx, log, cfg, tw)
	} else {
		log.Info("watching sandbox request activity and content", "watch-options", getExpectedOpts(cfg).String(), "output-dir", cfg.OutputDir)
		return trafficwatch.ConsumeToDir(ctx, log, cfg, tw)
	}
}
