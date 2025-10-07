package trafficwatch

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"

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

func ConsumeShort(ctx context.Context, log *slog.Logger, tw *trafficwatch.TrafficWatch, w io.Writer) error {
	waitDone := setupTW(ctx, tw, log)
	for meta := range tw.Meta {
		d, err := json.Marshal(meta)
		if err != nil {
			log.Warn("error unmarshaling request metadata", "error", err)
			continue
		}
		_, err = w.Write(d)
		if err != nil {
			log.Warn("unable to write request metadata", "error", err)
			continue
		}
	}
	<-waitDone
	return nil
}

func ConsumeToDir(ctx context.Context, log *slog.Logger, cfg *config.TrafficWatch, tw *trafficwatch.TrafficWatch, w io.Writer) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	waitDone := setupTW(ctx, tw, log)
	metaF, err := os.OpenFile(filepath.Join(cfg.ToDir, "meta.jsons"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
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
	fEnc := json.NewEncoder(metaF)
	oEnc := json.NewEncoder(w)
	for meta := range tw.Meta {
		if err := handleMetaToDir(cfg, log, fEnc, oEnc, meta); err != nil {
			return err
		}
	}
	<-waitDone
	return retErr
}

func handleMetaToDir(cfg *config.TrafficWatch, log *slog.Logger, fEnc, oEnc *json.Encoder, meta *api.RequestMetadata) error {
	if err := oEnc.Encode(meta); err != nil {
		return err
	}
	if err := fEnc.Encode(meta); err != nil {
		return err
	}
	p := filepath.Join(cfg.ToDir, meta.MiddlewareRequestID)
	if err := ensureDir(p); err != nil {
		return err
	}
	metaPath := filepath.Join(p, "meta.json")
	metaF, err := os.OpenFile(metaPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer metaF.Close()

	metaEnc := json.NewEncoder(metaF)
	metaEnc.SetIndent("", "  ")
	return metaEnc.Encode(meta)
}

func handleDataSource(cfg *config.TrafficWatch, s *trafficwatch.DataSource, errC chan error, what string) {
	defer s.R.Close()
	p := filepath.Join(cfg.ToDir, s.MiddlewareRequestID)
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
	signal.Notify(sigC, os.Interrupt)
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
		return os.Mkdir(p, 0755)
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
