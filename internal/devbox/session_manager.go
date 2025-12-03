package devbox

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"log/slog"

	"github.com/signadot/cli/internal/auth"
	"github.com/signadot/cli/internal/buildinfo"
	"github.com/signadot/cli/internal/config"
	"github.com/signadot/go-sdk/client"
	sdkauth "github.com/signadot/go-sdk/client/auth"
	"github.com/signadot/go-sdk/client/devboxes"
	"github.com/signadot/go-sdk/transport"
	"github.com/spf13/viper"
)

const (
	// RenewalInterval is how often to renew the devbox session
	RenewalInterval = 1 * time.Minute
	// RenewalJitter adds randomness to avoid thundering herd
	RenewalJitter = 30 * time.Second
)

type SessionManager struct {
	log            *slog.Logger
	ciConfig       *config.ConnectInvocationConfig
	apiClient      *client.SignadotAPI
	renewalTicker  *time.Ticker
	doneCh         chan struct{}
	shutdownCh     chan struct{}
	sessionReleased bool
	lastError      error
	lastErrorTime  time.Time
	mu             sync.RWMutex
}

func NewSessionManager(log *slog.Logger, ciConfig *config.ConnectInvocationConfig, shutdownCh chan struct{}) (*SessionManager, error) {
	if ciConfig.DevboxID == "" || ciConfig.DevboxSessionID == "" {
		// No devbox session to manage
		return nil, nil
	}

	// Resolve auth dynamically
	authInfo, err := auth.ResolveAuth()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve auth: %w", err)
	}
	if authInfo == nil {
		return nil, fmt.Errorf("no auth found")
	}

	// Create API client with dynamic auth resolution
	apiClient, err := createAPIClient(ciConfig, authInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	dsm := &SessionManager{
		log:        log.With("component", "devbox-session-manager"),
		ciConfig:   ciConfig,
		apiClient:  apiClient,
		doneCh:     make(chan struct{}),
		shutdownCh: shutdownCh,
	}

	return dsm, nil
}

func createAPIClient(ciConfig *config.ConnectInvocationConfig, authInfo *auth.ResolvedAuth) (*client.SignadotAPI, error) {
	// Check if bearer token is expired and refresh if needed
	if authInfo != nil && authInfo.BearerToken != "" && authInfo.APIKey == "" {
		if authInfo.ExpiresAt != nil && time.Now().After(*authInfo.ExpiresAt) {
			if authInfo.RefreshToken != "" {
				// Refresh the token
				refreshedAuth, err := refreshBearerToken(authInfo)
				if err != nil {
					return nil, fmt.Errorf("failed to refresh bearer token: %w", err)
				}
				authInfo = refreshedAuth
			} else {
				return nil, fmt.Errorf("bearer token expired and no refresh token available")
			}
		}
	}

	// Get API URL from viper (similar to config.API.basicInit)
	apiURL := "https://api.signadot.com"
	if apiURLFromViper := viper.GetString("api_url"); apiURLFromViper != "" {
		apiURL = apiURLFromViper
	}

	// Create transport config
	cfg := &transport.APIConfig{
		APIURL:    apiURL,
		UserAgent: fmt.Sprintf("signadot-cli:%s", buildinfo.Version),
		Debug:     false,
	}

	// Set auth - prefer resolved auth, but fall back to CI config API key if available
	// (similar to how control plane proxy handles it)
	if authInfo != nil && authInfo.APIKey != "" {
		cfg.APIKey = authInfo.APIKey
	} else if authInfo != nil && authInfo.BearerToken != "" {
		cfg.BearerToken = authInfo.BearerToken
	} else if ciConfig.APIKey != "" {
		// Fall back to API key from CI config
		cfg.APIKey = ciConfig.APIKey
	} else {
		return nil, fmt.Errorf("no API key or bearer token found")
	}

	// Initialize transport
	t, err := transport.InitAPITransport(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to init API transport: %w", err)
	}

	// Create client
	return client.New(t, nil), nil
}

// refreshBearerToken refreshes an expired bearer token using the refresh token
func refreshBearerToken(authInfo *auth.ResolvedAuth) (*auth.ResolvedAuth, error) {
	// Create an unauthenticated API client for the refresh call
	apiURL := "https://api.signadot.com"
	if apiURLFromViper := viper.GetString("api_url"); apiURLFromViper != "" {
		apiURL = apiURLFromViper
	}

	cfg := &transport.APIConfig{
		APIURL:    apiURL,
		UserAgent: fmt.Sprintf("signadot-cli:%s", buildinfo.Version),
		Debug:     false,
	}

	t, err := transport.InitAPITransport(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to init unauthenticated API transport: %w", err)
	}

	unauthClient := client.New(t, nil)

	// Call the refresh endpoint
	params := &sdkauth.AuthDeviceRefreshTokenParams{
		Data: authInfo.RefreshToken,
	}

	resp, err := unauthClient.Auth.AuthDeviceRefreshToken(params)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	expiresAt := time.Now().Add(time.Duration(resp.Payload.ExpiresIn) * time.Second)
	
	// Update auth info with new tokens
	newAuthInfo := &auth.ResolvedAuth{
		Source: authInfo.Source,
		Auth: auth.Auth{
			APIKey:       authInfo.APIKey,
			BearerToken:  resp.Payload.AccessToken,
			RefreshToken: resp.Payload.RefreshToken,
			OrgName:      authInfo.OrgName,
			ExpiresAt:    &expiresAt,
		},
	}

	// Save the refreshed token back to storage
	if err := saveRefreshedAuth(newAuthInfo); err != nil {
		// Log but don't fail - we can still use the refreshed token for this request
		// The next ResolveAuth() call will get the old token, but it will be refreshed again
	}

	return newAuthInfo, nil
}

// saveRefreshedAuth saves the refreshed auth back to the storage (keyring or plaintext)
func saveRefreshedAuth(authInfo *auth.ResolvedAuth) error {
	switch authInfo.Source {
	case auth.KeyringAuthSource:
		keyringStorage := auth.NewKeyringStorage()
		return keyringStorage.Store(&authInfo.Auth)
	case auth.PlainTextAuthSource:
		plainTextStorage := auth.NewPlainTextStorage()
		return plainTextStorage.Store(&authInfo.Auth)
	default:
		// Config source doesn't need saving (it's in viper)
		return nil
	}
}

func (dsm *SessionManager) Start(ctx context.Context) {
	dsm.log.Info("Starting devbox session manager",
		"devboxID", dsm.ciConfig.DevboxID,
		"sessionID", dsm.ciConfig.DevboxSessionID)

	// Do initial renewal
	go dsm.renewLoop(ctx)

	// Set up periodic renewal
	interval := RenewalInterval + time.Duration(time.Now().UnixNano()%int64(RenewalJitter))
	dsm.renewalTicker = time.NewTicker(interval)
	defer dsm.renewalTicker.Stop()

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

func (dsm *SessionManager) renewLoop(ctx context.Context) {
	// Initial renewal
	dsm.renewSession(ctx)

	// Periodic renewals
	for {
		select {
		case <-dsm.doneCh:
			return
		case <-time.After(RenewalInterval):
			dsm.renewSession(ctx)
		}
	}
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

	// Recreate API client if needed (in case auth changed or token expired)
	apiClient, err := createAPIClient(dsm.ciConfig, authInfo)
	if err != nil {
		dsm.log.Error("Failed to recreate API client", "error", err)
		return
	}
	dsm.apiClient = apiClient

	params := devboxes.NewRenewDevboxParams().
		WithContext(ctx).
		WithOrgName(authInfo.OrgName).
		WithDevboxID(dsm.ciConfig.DevboxID)

	resp, err := dsm.apiClient.Devboxes.RenewDevbox(params)
	if err != nil {
		// Check if the error indicates the session was released by another process
		if dsm.isSessionReleasedError(err) {
			dsm.log.Warn("Devbox session was released by another process",
				"devboxID", dsm.ciConfig.DevboxID,
				"sessionID", dsm.ciConfig.DevboxSessionID)
			dsm.setSessionReleased(err)
			dsm.triggerShutdown()
			return
		}
		dsm.log.Error("Failed to renew devbox session", "error", err)
		dsm.setError(err)
		return
	}

	// Check response status code - 404 or similar might indicate session was released
	if resp.Code() == http.StatusNotFound {
		err := fmt.Errorf("devbox session not found (status %d)", resp.Code())
		dsm.log.Warn("Devbox session not found (likely released by another process)",
			"devboxID", dsm.ciConfig.DevboxID,
			"sessionID", dsm.ciConfig.DevboxSessionID)
		dsm.setSessionReleased(err)
		dsm.triggerShutdown()
		return
	}

	// Also verify the session ID matches by checking the devbox status
	if !dsm.verifySessionStillActive(ctx, authInfo.OrgName) {
		err := fmt.Errorf("devbox session ID mismatch")
		dsm.log.Warn("Devbox session ID mismatch (likely released by another process)",
			"devboxID", dsm.ciConfig.DevboxID,
			"sessionID", dsm.ciConfig.DevboxSessionID)
		dsm.setSessionReleased(err)
		dsm.triggerShutdown()
		return
	}

	dsm.log.Debug("Renewed devbox session",
		"devboxID", dsm.ciConfig.DevboxID,
		"sessionID", dsm.ciConfig.DevboxSessionID,
		"statusCode", resp.Code())
}

func (dsm *SessionManager) releaseSession() {
	dsm.log.Info("Releasing devbox session",
		"devboxID", dsm.ciConfig.DevboxID,
		"sessionID", dsm.ciConfig.DevboxSessionID)

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

	// Recreate API client if needed (in case auth changed or token expired)
	apiClient, err := createAPIClient(dsm.ciConfig, authInfo)
	if err != nil {
		dsm.log.Error("Failed to recreate API client for release", "error", err)
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
		"sessionID", dsm.ciConfig.DevboxSessionID,
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

// isSessionReleasedError checks if an error indicates the session was released
func (dsm *SessionManager) isSessionReleasedError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// Check for common error patterns that indicate session was released
	return strings.Contains(errStr, "404") ||
		strings.Contains(errStr, "not found") ||
		strings.Contains(errStr, "session") && strings.Contains(errStr, "released")
}

// verifySessionStillActive checks if the current session ID still matches the devbox's active session
func (dsm *SessionManager) verifySessionStillActive(ctx context.Context, orgName string) bool {
	params := devboxes.NewGetDevboxParams().
		WithContext(ctx).
		WithOrgName(orgName).
		WithDevboxID(dsm.ciConfig.DevboxID)

	resp, err := dsm.apiClient.Devboxes.GetDevbox(params)
	if err != nil {
		// If we can't check, assume it's still active to avoid false positives
		dsm.log.Debug("Failed to verify session status, assuming still active", "error", err)
		return true
	}

	if resp.Code() != http.StatusOK {
		// If we can't get the devbox, assume it's still active
		return true
	}

	session := resp.Payload.Status.Session
	if session == nil {
		// No active session means it was released
		return false
	}

	// Session ID mismatch means another process claimed/released it
	return session.ID == dsm.ciConfig.DevboxSessionID
}

// setSessionReleased marks the session as released
func (dsm *SessionManager) setSessionReleased(err error) {
	dsm.mu.Lock()
	defer dsm.mu.Unlock()
	dsm.sessionReleased = true
	dsm.lastError = err
	dsm.lastErrorTime = time.Now()
}

// setError records an error without marking session as released
func (dsm *SessionManager) setError(err error) {
	dsm.mu.Lock()
	defer dsm.mu.Unlock()
	dsm.lastError = err
	dsm.lastErrorTime = time.Now()
}

// WasSessionReleased returns whether the session was released
func (dsm *SessionManager) WasSessionReleased() bool {
	dsm.mu.RLock()
	defer dsm.mu.RUnlock()
	return dsm.sessionReleased
}

// GetStatus returns the current session status
func (dsm *SessionManager) GetStatus() (healthy bool, sessionReleased bool, devboxID string, sessionID string, lastError error, lastErrorTime time.Time) {
	dsm.mu.RLock()
	defer dsm.mu.RUnlock()
	
	if dsm.ciConfig == nil {
		return false, false, "", "", nil, time.Time{}
	}
	
	healthy = !dsm.sessionReleased && dsm.lastError == nil
	return healthy, dsm.sessionReleased, dsm.ciConfig.DevboxID, dsm.ciConfig.DevboxSessionID, dsm.lastError, dsm.lastErrorTime
}

// triggerShutdown closes the shutdown channel to trigger sandbox manager shutdown
func (dsm *SessionManager) triggerShutdown() {
	if dsm.shutdownCh == nil {
		return
	}
	select {
	case <-dsm.shutdownCh:
		// Already closed
	default:
		close(dsm.shutdownCh)
	}
}
