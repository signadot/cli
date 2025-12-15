package devbox

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"log/slog"

	"github.com/signadot/cli/internal/auth"
	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/locald/sandboxmanager/apiclient"
	"github.com/signadot/go-sdk/client/devboxes"
)

const (
	// RenewalInterval is how often to renew the devbox session
	RenewalInterval = 15 * time.Second
	// RenewalJitter adds randomness to avoid thundering herd
	RenewalJitter = 5 * time.Second
)

type SessionManager struct {
	log           *slog.Logger
	ciConfig      *config.ConnectInvocationConfig
	renewalTicker *time.Ticker
	doneCh        chan struct{}
	lastError     error
	lastErrorTime time.Time
	mu            sync.RWMutex
}

func NewSessionManager(log *slog.Logger, ciConfig *config.ConnectInvocationConfig) (*SessionManager, error) {
	if ciConfig.DevboxID == "" || ciConfig.DevboxSessionID == "" {
		return nil, fmt.Errorf("incomplete or absent  devbox session info")
	}

	// Resolve auth dynamically
	authInfo, err := auth.ResolveAuth()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve auth: %w", err)
	}
	if authInfo == nil {
		return nil, fmt.Errorf("no auth found")
	}

	log.Debug("NewSessionManager: auth resolved",
		"source", authInfo.Source,
		"orgName", authInfo.OrgName,
		"hasAPIKey", authInfo.APIKey != "",
		"hasBearerToken", authInfo.BearerToken != "",
		"hasExpiresAt", authInfo.ExpiresAt != nil,
		"expiresAt", authInfo.ExpiresAt)

	// Note: We don't store the API client - we create a fresh one for each request
	// to avoid stale connection errors after sleep periods

	dsm := &SessionManager{
		log:      log.With("component", "devbox-session-manager"),
		ciConfig: ciConfig,
		doneCh:   make(chan struct{}),
	}

	return dsm, nil
}

func (dsm *SessionManager) Start(ctx context.Context) {
	currentSessionID := dsm.ciConfig.DevboxSessionID

	dsm.log.Info("Starting devbox session manager",
		"devboxID", dsm.ciConfig.DevboxID,
		"sessionID", currentSessionID)

	// Do initial renewal immediately
	go dsm.renewSession(ctx)

	// Set up periodic renewal with jitter
	interval := RenewalInterval + time.Duration(time.Now().UnixNano()%int64(RenewalJitter))
	dsm.renewalTicker = time.NewTicker(interval)

	go func() {
		for {
			select {
			case <-dsm.doneCh:
				return
			case <-dsm.renewalTicker.C:
				dsm.renewSession(ctx)
			}
		}
	}()
}

func (dsm *SessionManager) renewSession(ctx context.Context) {
	// Resolve auth dynamically to get org
	authInfo, err := auth.ResolveAuth()
	if err != nil {
		dsm.log.Error("Failed to resolve auth for renewal", "error", err)
		return
	}
	if authInfo == nil || authInfo.OrgName == "" {
		dsm.log.Error("No org found in auth")
		return
	}

	dsm.log.Debug("renewSession: auth resolved",
		"source", authInfo.Source,
		"orgName", authInfo.OrgName,
		"hasAPIKey", authInfo.APIKey != "",
		"hasBearerToken", authInfo.BearerToken != "",
		"hasExpiresAt", authInfo.ExpiresAt != nil,
		"expiresAt", authInfo.ExpiresAt)

	// Create a fresh API client for each renewal request to avoid stale connection errors
	// after sleep periods. Each request gets a completely fresh client/transport.
	apiClient, err := apiclient.CreateAPIClientWithLogger(dsm.ciConfig, authInfo, dsm.log)
	if err != nil {
		dsm.log.Error("Failed to create API client", "error", err)
		return
	}

	params := devboxes.NewRenewDevboxParams().
		WithContext(ctx).
		WithOrgName(authInfo.OrgName).
		WithDevboxID(dsm.ciConfig.DevboxID).
		WithDevboxSessionID(dsm.ciConfig.DevboxSessionID)

	log := dsm.log.With("devboxID", dsm.ciConfig.DevboxID,
		"sessionID", dsm.ciConfig.DevboxSessionID,
		"orgName", authInfo.OrgName)

	log.Debug("renewSession: calling RenewDevbox")

	resp, err := apiClient.Devboxes.RenewDevbox(params)
	if err != nil {
		log.Error("Failed to renew devbox session", "error", err)
		dsm.setError(err)
		return
	}

	log.Debug("renewSession: RenewDevbox call succeeded",
		"statusCode", resp.Code())

	if resp.Code() == http.StatusOK {
		dsm.setError(nil)
	} else {
		dsm.setError(fmt.Errorf("error renewing devbox: %d %s", resp.Code(), http.StatusText(resp.Code())))
	}
}

func (dsm *SessionManager) releaseSession() {
	currentSessionID := dsm.ciConfig.DevboxSessionID

	dsm.log.Info("Releasing devbox session",
		"devboxID", dsm.ciConfig.DevboxID,
		"sessionID", currentSessionID)

	// Use a background context with timeout for release to ensure it completes
	// even if the original context is cancelled
	releaseCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Resolve auth dynamically to get org
	authInfo, err := auth.ResolveAuth()
	if err != nil {
		dsm.log.Error("Failed to resolve auth for release", "error", err)
		return
	}
	if authInfo == nil || authInfo.OrgName == "" {
		dsm.log.Error("No org found in auth")
		return
	}

	// Create a fresh API client for the release request
	apiClient, err := apiclient.CreateAPIClientWithLogger(dsm.ciConfig, authInfo, dsm.log)
	if err != nil {
		dsm.log.Error("Failed to create API client for release", "error", err)
		return
	}

	params := devboxes.NewReleaseDevboxParams().
		WithContext(releaseCtx).
		WithOrgName(authInfo.OrgName).
		WithDevboxID(dsm.ciConfig.DevboxID)

	resp, err := apiClient.Devboxes.ReleaseDevbox(params)
	if err != nil {
		dsm.log.Error("Failed to release devbox session", "error", err)
		return
	}

	dsm.log.Info("Released devbox session",
		"devboxID", dsm.ciConfig.DevboxID,
		"sessionID", currentSessionID,
		"statusCode", resp.Code())

}

func (dsm *SessionManager) Stop(ctx context.Context) {
	select {
	case <-dsm.doneCh:
	default:
		close(dsm.doneCh)
	}

	dsm.renewalTicker.Stop()

	// Release session on shutdown
	dsm.releaseSession()
}

// setError records an error
func (dsm *SessionManager) setError(err error) {
	dsm.mu.Lock()
	defer dsm.mu.Unlock()
	dsm.lastError = err
	dsm.lastErrorTime = time.Now()
}

// GetStatus returns the current session status.
func (dsm *SessionManager) GetStatus() (healthy bool, devboxID string, sessionID string, lastErrorTime time.Time, lastError error) {
	dsm.mu.RLock()
	defer dsm.mu.RUnlock()

	if dsm.ciConfig == nil {
		return false, "", "", time.Time{}, nil
	}

	healthy = dsm.lastError == nil
	return healthy, dsm.ciConfig.DevboxID, dsm.ciConfig.DevboxSessionID, dsm.lastErrorTime, dsm.lastError
}
