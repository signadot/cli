package sandboxmanager

import (
	"context"
	"time"

	"log/slog"

	tunapiv1 "github.com/signadot/libconnect/apiv1"
	tunapiclient "github.com/signadot/libconnect/common/apiclient"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type sbMonitor struct {
	log          *slog.Logger
	routingKey   string
	tunAPIClient tunapiclient.Client
	eventFn      func(string, *tunapiv1.WatchSandboxResponse)
	doneCh       chan struct{}
}

func newSBMonitor(log *slog.Logger, sandboxName, routingKey string,
	tunAPIClient tunapiclient.Client, eventFn func(string, *tunapiv1.WatchSandboxResponse)) *sbMonitor {
	res := &sbMonitor{
		log:          log.With("sandbox", sandboxName),
		routingKey:   routingKey,
		tunAPIClient: tunAPIClient,
		eventFn:      eventFn,
		doneCh:       make(chan struct{}),
	}
	go res.monitor()
	return res
}

func (sbm *sbMonitor) stop() {
	select {
	case <-sbm.doneCh:
	default:
		close(sbm.doneCh)
	}
}

func (sbm *sbMonitor) monitor() {
	// setup context for grpc stream
	ctx, cancel := context.WithCancel(context.Background())

	// watch the given sandbox
	go sbm.watchSandbox(ctx)

	// wait until done
	<-sbm.doneCh
	// we are done, cancel the context
	cancel()
}

func (sbm *sbMonitor) watchSandbox(ctx context.Context) {
	// watch loop
	for {
		sbwClient, err := sbm.tunAPIClient.WatchSandbox(ctx, &tunapiv1.WatchSandboxRequest{
			RoutingKey: sbm.routingKey,
		})
		if err != nil {
			// don't retry if the context has been cancelled
			select {
			case <-ctx.Done():
				return
			default:
			}

			sbm.log.Error("error getting sb watch stream, retrying", "error", err)
			<-time.After(3 * time.Second)
			continue
		}

		sbm.log.Debug("successfully got sandbox watch client")
		err = sbm.readStream(ctx, sbwClient)
		if err == nil {
			// NotFound
			break
		}
	}

	// There is no sandbox, stop the monitor
	sbm.stop()
}

func (sbm *sbMonitor) readStream(ctx context.Context,
	sbwClient tunapiv1.TunnelAPI_WatchSandboxClient) error {
	var err error
	for {
		sbStatus, err := sbwClient.Recv()
		if err == nil {
			sbm.eventFn(sbm.routingKey, sbStatus)
			continue
		}
		// just return if the context has been cancelled
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		// extract the grpc status
		grpcStatus, ok := status.FromError(err)
		if !ok {
			sbm.log.Error("sandbox monitor grpc stream error: no status",
				"error", err)
			break
		}
		switch grpcStatus.Code() {
		case codes.OK:
			sbm.log.Debug("sandbox watch stream error code is ok")
			sbm.eventFn(sbm.routingKey, sbStatus)
			continue
		case codes.Internal:
			sbm.log.Error("sandbox watch: internal grpc error", "error", err)
			<-time.After(3 * time.Second)
		case codes.NotFound:
			sbm.log.Info("sandbox watch: sandbox not found")
			sbm.eventFn(sbm.routingKey, nil)
			err = nil
		default:
			sbm.log.Error("sandbox watch error", "error", err)
		}
		break
	}
	return err
}
