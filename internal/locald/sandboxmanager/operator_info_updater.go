package sandboxmanager

import (
	"context"
	"log/slog"
	"sync"
	"time"

	commonapi "github.com/signadot/cli/internal/locald/api"
	tunapiv1 "github.com/signadot/libconnect/apiv1"
	tunapiclient "github.com/signadot/libconnect/common/apiclient"
)

type operatorInfoUpdater struct {
	sync.Mutex

	log          *slog.Logger
	operatorInfo *commonapi.OperatorInfo
	isRunning    bool
}

func (oiu *operatorInfoUpdater) Get() *commonapi.OperatorInfo {
	oiu.Lock()
	defer oiu.Unlock()
	return oiu.operatorInfo
}

func (oiu *operatorInfoUpdater) Reset() {
	oiu.Lock()
	defer oiu.Unlock()
	oiu.operatorInfo = nil
}

func (oiu *operatorInfoUpdater) Reload(ctx context.Context, tunAPIClient tunapiclient.Client, force bool) {
	oiu.Lock()
	defer oiu.Unlock()

	if oiu.isRunning {
		// nothing to do here
		return
	}
	if !force && oiu.operatorInfo != nil {
		// we already have the operator info
		return
	}
	// reload the data
	go oiu.reload(ctx, tunAPIClient)
}

func (oiu *operatorInfoUpdater) reload(ctx context.Context, tunAPIClient tunapiclient.Client) {
	oiu.setIsRunning(true)
	defer oiu.setIsRunning(false)

	for {
		resp, err := tunAPIClient.GetOperatorInfo(ctx, &tunapiv1.GetOperatorInfoRRequest{})
		if err == nil {
			// update the operator info
			oiu.Lock()
			oiu.operatorInfo = &commonapi.OperatorInfo{
				Version:   resp.Version,
				GitCommit: resp.GitCommit,
				BuildDate: resp.BuildDate,
			}
			oiu.Unlock()
			return
		}

		// retry later
		oiu.log.Error("getting operator info", "error", err)
		select {
		case <-ctx.Done():
			return
		case <-time.After(3 * time.Second):
		}
	}
}

func (oiu *operatorInfoUpdater) setIsRunning(isRunning bool) {
	oiu.Lock()
	defer oiu.Unlock()
	oiu.isRunning = isRunning
}
