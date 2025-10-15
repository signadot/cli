package trafficwatch

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/signadot/cli/internal/auth"
	"github.com/signadot/cli/internal/config"
	"github.com/signadot/libconnect/common/trafficwatch"
	"github.com/signadot/libconnect/common/trafficwatch/api"
	"github.com/signadot/libconnect/proxy/httpconnect"
)

func GetTrafficWatch(ctx context.Context, cfg *config.TrafficWatch, log *slog.Logger, rk string) (*trafficwatch.TrafficWatch, error) {
	watchOpts := getWatchOpts(cfg)
	hdrs, err := auth.GetHeaders()
	if err != nil {
		return nil, err
	}
	hdrs.Set("signadot-control", "true")
	twClient := &trafficwatch.WatchTrafficClient{
		Org: cfg.Org,
		Log: log,
	}
	dialer := httpconnect.NewDialer(log, cfg.ProxyURL, hdrs)
	conn, err := dialer.DialContext(ctx, "tcp", "trafficwatch-server.control:77")
	if err != nil {
		return nil, err
	}
	tw, err := twClient.Watch(ctx, conn, &api.RequestMetadata{
		RoutingKey:   rk,
		WatchOptions: *api.WatchAll(),
	}, watchOpts)
	if err != nil {
		return nil, err
	}
	return tw, nil
}

func getWatchOpts(cfg *config.TrafficWatch) *api.WatchOptions {
	if cfg.Short {
		return api.WatchShort()
	}
	if cfg.HeadersOnly {
		return api.WatchTruncate(0)
	}
	return api.WatchAll()
}

func ConsumeShort(ctx context.Context, log *slog.Logger, cfg *config.TrafficWatch, tw *trafficwatch.TrafficWatch) error {
	waitDone := setupTW(ctx, tw, log)
	var enc metaEncoder
	if cfg.OutputFile != "" {
		f, err := os.OpenFile(cfg.OutputFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer f.Close()
		enc = getMetaEncoder(f, cfg)
	}

	go encodeReqDones(tw.RequestDone, log, nil, enc)
	for meta := range tw.Meta {
		log.Info("incoming-request", "request", (*logMeta)(meta))
		if enc == nil {
			continue
		}
		err := enc.Encode(meta)
		if err != nil {
			log.Warn("error encoding request", "id", meta.MiddlewareRequestID, "error", err)
		}
	}
	<-waitDone
	return nil
}

func ConsumeToDir(ctx context.Context, log *slog.Logger, cfg *config.TrafficWatch, tw *trafficwatch.TrafficWatch) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	waitDone := setupTW(ctx, tw, log)
	suffix := StreamFormatSuffix(cfg)
	metaF, err := os.OpenFile(filepath.Join(cfg.OutputDir, "meta"+suffix), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer metaF.Close()

	dataSourceErrs := make(chan error, 1)

	go func() {
		for s := range tw.Requests {
			go handleDataSource(cfg, s, dataSourceErrs, "request")
		}
	}()
	go func() {
		for s := range tw.Responses {
			go handleDataSource(cfg, s, dataSourceErrs, "response")
		}

	}()
	var retErr error
	go func() {
		err := <-dataSourceErrs
		log.Error("error copying request/response", "error", err)
		retErr = err
		cancel()

	}()
	fEnc := getMetaEncoder(metaF, cfg)
	go encodeReqDones(tw.RequestDone, log, handleDir(cfg), fEnc)
	for meta := range tw.Meta {
		if err := handleMetaToDir(cfg, log, fEnc, meta); err != nil {
			return err
		}
	}
	<-waitDone
	return retErr
}

func handleMetaToDir(cfg *config.TrafficWatch, log *slog.Logger, fEnc metaEncoder, meta *api.RequestMetadata) error {
	log.Info("incoming-request", "request", (*logMeta)(meta))
	if err := fEnc.Encode(meta); err != nil {
		return err
	}
	p := filepath.Join(cfg.OutputDir, meta.MiddlewareRequestID)
	if err := ensureDir(p); err != nil {
		return err
	}
	suffix := ".json"
	if cfg.OutputFormat == config.OutputFormatYAML {
		suffix = ".yaml"
	}
	metaPath := filepath.Join(p, "meta"+suffix)
	metaF, err := os.OpenFile(metaPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer metaF.Close()

	metaEnc := getMetaEncoder(metaF, cfg)
	if metaEnc.j != nil {
		metaEnc.j.SetIndent("", "  ")
	}
	return metaEnc.Encode(meta)
}

func handleDataSource(cfg *config.TrafficWatch, s *trafficwatch.DataSource, errC chan error, what string) {
	defer s.R.Close()
	p := filepath.Join(cfg.OutputDir, s.MiddlewareRequestID)
	if err := ensureDir(p); err != nil {
		errC <- err
		return
	}
	p = filepath.Join(p, what)
	f, err := os.OpenFile(p, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		errC <- err
		return
	}
	_, err = io.Copy(f, s.R)
	if err != nil {
		if !errors.Is(err, os.ErrClosed) {
			errC <- err
		}
	}

}

func setupTW(ctx context.Context, tw *trafficwatch.TrafficWatch, log *slog.Logger) <-chan struct{} {
	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	res := make(chan struct{})
	go func() {
		select {
		case <-sigC:
			safeClose(tw.Close)
		case <-tw.Close:
			log.Info("server closed connection")
		case <-ctx.Done():
			log.Info("session timed out")
			safeClose(tw.Close)
		}
		signal.Ignore(os.Interrupt)
		close(res)
	}()
	return res
}

func ensureDir(p string) error {
	_, err := os.Stat(p)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err != nil {
		err := os.Mkdir(p, 0755)
		if err != nil && !os.IsExist(err) {
			return err
		}
	}
	return nil
}

func safeClose(c chan struct{}) {
	select {
	case <-c:
	default:
		close(c)
	}
}
