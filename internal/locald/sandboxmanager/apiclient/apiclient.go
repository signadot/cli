package apiclient

import (
	"fmt"
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
	} else if ciConfig != nil && ciConfig.APIKey != "" {
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
