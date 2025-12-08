package apiclient

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-openapi/runtime"
	"github.com/signadot/cli/internal/auth"
	"github.com/signadot/cli/internal/buildinfo"
	"github.com/signadot/cli/internal/config"
	"github.com/signadot/go-sdk/client"
	sdkauth "github.com/signadot/go-sdk/client/auth"
	"github.com/signadot/go-sdk/transport"
	"github.com/spf13/viper"
)

// CreateAPIClient creates a unified API client with proper auth resolution and token refresh.
// This is the unified mechanism for creating API clients that handles:
// - Dynamic auth resolution
// - Bearer token refresh
// - Proper API URL configuration
// - Fallback to CI config API key
func CreateAPIClient(ciConfig *config.ConnectInvocationConfig, authInfo *auth.ResolvedAuth) (*client.SignadotAPI, error) {
	return CreateAPIClientWithLogger(ciConfig, authInfo, nil)
}

const defaultAPIURL = "https://api.signadot.com"

// getLogger returns the provided logger or slog.Default() if nil
func getLogger(log *slog.Logger) *slog.Logger {
	if log != nil {
		return log
	}
	return slog.Default()
}

// resolveAPIURL resolves the API URL with priority: ciConfig > viper > default.
// Returns the resolved URL and its source for logging purposes.
func resolveAPIURL(ciConfig *config.ConnectInvocationConfig) (url, source string) {
	if ciConfig != nil && ciConfig.APIURL != "" {
		return ciConfig.APIURL, "ciConfig"
	}
	if apiURLFromViper := viper.GetString("api_url"); apiURLFromViper != "" {
		return apiURLFromViper, "viper"
	}
	return defaultAPIURL, "default"
}

// createTransportConfig creates a transport.APIConfig with the given API URL.
func createTransportConfig(apiURL string) *transport.APIConfig {
	return &transport.APIConfig{
		APIURL:    apiURL,
		UserAgent: fmt.Sprintf("signadot-cli:%s", buildinfo.Version),
		Debug:     false,
	}
}

// CreateAPIClientWithLogger creates a unified API client with logging support
func CreateAPIClientWithLogger(ciConfig *config.ConnectInvocationConfig, authInfo *auth.ResolvedAuth, log *slog.Logger) (*client.SignadotAPI, error) {
	log = getLogger(log)

	if authInfo == nil {
		log.Error("CreateAPIClient: authInfo is nil")
		return nil, fmt.Errorf("authInfo is required")
	}

	// Log auth info resolution
	authType := "none"
	if authInfo.APIKey != "" {
		authType = "api-key"
	} else if authInfo.BearerToken != "" {
		authType = "bearer-token"
		if authInfo.ExpiresAt != nil {
			now := time.Now()
			expired := now.After(*authInfo.ExpiresAt)
			log.Debug("CreateAPIClient: bearer token expiration check",
				"expiresAt", authInfo.ExpiresAt,
				"now", now,
				"expired", expired,
				"hasRefreshToken", authInfo.RefreshToken != "")
		}
	}
	log.Debug("CreateAPIClient: auth info resolved",
		"source", authInfo.Source,
		"authType", authType,
		"orgName", authInfo.OrgName)

	// Check if bearer token is expired and refresh if needed
	if authInfo.BearerToken != "" && authInfo.APIKey == "" && authInfo.ExpiresAt != nil && time.Now().After(*authInfo.ExpiresAt) {
		if authInfo.RefreshToken == "" {
			log.Error("CreateAPIClient: bearer token expired but no refresh token available")
			return nil, fmt.Errorf("bearer token expired and no refresh token available")
		}
		log.Debug("CreateAPIClient: bearer token expired, attempting refresh")
		refreshedAuth, err := refreshBearerToken(authInfo, log)
		if err != nil {
			log.Error("CreateAPIClient: failed to refresh bearer token", "error", err)
			return nil, fmt.Errorf("failed to refresh bearer token: %w", err)
		}
		log.Debug("CreateAPIClient: bearer token refreshed successfully")
		authInfo = refreshedAuth
	}

	// Get API URL - prefer ciConfig (passed from connect command), then viper, then default
	// Note: In daemon context, viper may not be initialized, so ciConfig.APIURL is preferred
	apiURL, apiURLSource := resolveAPIURL(ciConfig)
	log.Debug("CreateAPIClient: API URL resolved",
		"apiURL", apiURL,
		"source", apiURLSource,
		"ciConfigHasAPIURL", ciConfig != nil && ciConfig.APIURL != "",
		"viperHasAPIURL", viper.GetString("api_url") != "")

	// Create transport config
	cfg := createTransportConfig(apiURL)

	// Set auth - prefer resolved auth, but fall back to CI config API key if available
	// (similar to how control plane proxy handles it)
	var transportAuthType string
	switch {
	case authInfo.APIKey != "":
		cfg.APIKey = authInfo.APIKey
		transportAuthType = "resolved-api-key"
	case authInfo.BearerToken != "":
		cfg.BearerToken = authInfo.BearerToken
		transportAuthType = "resolved-bearer-token"
	case ciConfig != nil && ciConfig.APIKey != "":
		cfg.APIKey = ciConfig.APIKey
		transportAuthType = "ciConfig-api-key"
	default:
		log.Error("CreateAPIClient: no API key or bearer token found")
		return nil, fmt.Errorf("no API key or bearer token found")
	}

	log.Debug("CreateAPIClient: transport config created",
		"apiURL", cfg.APIURL,
		"authType", transportAuthType,
		"hasAPIKey", cfg.APIKey != "",
		"hasBearerToken", cfg.BearerToken != "")

	// Initialize transport
	t, err := transport.InitAPITransport(cfg)
	if err != nil {
		log.Error("CreateAPIClient: failed to init API transport", "error", err)
		return nil, fmt.Errorf("failed to init API transport: %w", err)
	}

	// Create client
	return client.New(t, nil), nil
}

// refreshBearerToken refreshes an expired bearer token using the refresh token
func refreshBearerToken(authInfo *auth.ResolvedAuth, log *slog.Logger) (*auth.ResolvedAuth, error) {
	log = getLogger(log)

	// Create an unauthenticated API client for the refresh call
	// Note: We don't use ciConfig here since this is a refresh call that happens
	// independently of the connect command context
	apiURL, apiURLSource := resolveAPIURL(nil)
	log.Debug("refreshBearerToken: using API URL for refresh",
		"apiURL", apiURL,
		"source", apiURLSource,
		"hasRefreshToken", authInfo.RefreshToken != "")

	cfg := createTransportConfig(apiURL)

	t, err := transport.InitAPITransport(cfg)
	if err != nil {
		log.Error("refreshBearerToken: failed to init unauthenticated API transport", "error", err)
		return nil, fmt.Errorf("failed to init unauthenticated API transport: %w", err)
	}

	unauthClient := client.New(t, nil)

	// Call the refresh endpoint
	params := &sdkauth.AuthDeviceRefreshTokenParams{
		Data: authInfo.RefreshToken,
	}

	log.Debug("refreshBearerToken: calling refresh endpoint", "apiURL", apiURL)
	resp, err := unauthClient.Auth.AuthDeviceRefreshToken(params)
	if err != nil {
		log.Error("refreshBearerToken: refresh call failed", "error", err, "apiURL", apiURL)
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	expiresAt := time.Now().Add(time.Duration(resp.Payload.ExpiresIn) * time.Second)

	log.Debug("refreshBearerToken: refresh successful",
		"expiresIn", resp.Payload.ExpiresIn,
		"expiresAt", expiresAt,
		"hasNewRefreshToken", resp.Payload.RefreshToken != "")

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
		log.Error("refreshBearerToken: failed to save refreshed auth", "error", err, "source", authInfo.Source)
		// Log but don't fail - we can still use the refreshed token for this request
		// The next ResolveAuth() call will get the old token, but it will be refreshed again
	} else {
		log.Debug("refreshBearerToken: saved refreshed auth to storage", "source", authInfo.Source)
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

// GetAuthHeaders returns HTTP headers using the unified auth resolution mechanism.
// This ensures consistent auth handling across all components that need to make API calls.
// It resolves auth dynamically and handles bearer tokens properly.
func GetAuthHeaders(ciConfig *config.ConnectInvocationConfig) (http.Header, error) {
	// Resolve auth dynamically
	authInfo, err := auth.ResolveAuth()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve auth: %w", err)
	}

	headers := http.Header{}
	if authInfo == nil {
		// Fall back to CI config API key if available
		if ciConfig != nil && ciConfig.APIKey != "" {
			headers.Set(transport.APIKeyHeader, ciConfig.APIKey)
		}
		return headers, nil
	}

	// Use resolved auth, with fallback to CI config
	if authInfo.APIKey != "" {
		headers.Set(transport.APIKeyHeader, authInfo.APIKey)
	} else if authInfo.BearerToken != "" {
		headers.Set(runtime.HeaderAuthorization, "Bearer "+authInfo.BearerToken)
	} else if ciConfig != nil && ciConfig.APIKey != "" {
		// Fall back to API key from CI config
		headers.Set(transport.APIKeyHeader, ciConfig.APIKey)
	}

	return headers, nil
}
