package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/signadot/cli/internal/auth"
	"github.com/signadot/cli/internal/buildinfo"
	"github.com/signadot/go-sdk/client"
	sdkauth "github.com/signadot/go-sdk/client/auth"
	"github.com/signadot/go-sdk/transport"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

var (
	ErrAuthExpired    = errors.New("Authentication expired. Please log in using 'signadot auth login'")
	ErrAuthNoOrgFound = errors.New("No organisation found. Please log in using 'signadot auth login'")
	ErrAuthNoFound    = errors.New("No authentication found. Please log in using 'signadot auth login'")
)

type API struct {
	Root

	// Config file values
	Org             string
	MaskedAPIKey    string
	APIURL          string
	ArtifactsAPIURL string
	ProxyURL        string
	MCPURL          string

	// Runtime values
	Client *client.SignadotAPI `json:"-"`

	ApiKey      string
	BearerToken string
	UserAgent   string
}

// for error reporting, we select the config that
// the user controls.
func (a *API) MarshalJSON() ([]byte, error) {
	return a.marshal(json.Marshal)
}

func (a *API) MarshalYAML() ([]byte, error) {
	return a.marshal(yaml.Marshal)
}

func (a *API) marshal(marshaller func(interface{}) ([]byte, error)) ([]byte, error) {
	type T struct {
		Debug        bool
		ConfigFile   string
		Org          string
		MaskedAPIKey string
		APIURL       string
		ProxyURL     string
	}
	t := &T{
		Debug:        a.Debug,
		ConfigFile:   a.ConfigFile,
		Org:          a.Org,
		MaskedAPIKey: a.MaskedAPIKey,
		APIURL:       a.APIURL,
		ProxyURL:     a.ProxyURL,
	}
	return marshaller(t)
}

func (a *API) init() error {
	authInfo, err := auth.ResolveAuth()
	if err != nil {
		return fmt.Errorf("could not resolve auth: %w", err)
	}

	if authInfo == nil || (authInfo.APIKey == "" && authInfo.BearerToken == "") {
		return ErrAuthNoFound
	}
	if authInfo.ExpiresAt != nil && authInfo.ExpiresAt.Before(time.Now()) && authInfo.Source != auth.KeyringAuthSource {
		return ErrAuthExpired
	}
	if authInfo.OrgName == "" {
		return ErrAuthNoOrgFound
	}

	// Init basic settings and return
	if err := a.basicInit(); err != nil {
		return err
	}

	if authInfo.Source == auth.KeyringAuthSource {
		if err := a.checkKeyringAuth(authInfo); err != nil {
			return err
		}
	}

	a.ApiKey = authInfo.APIKey
	a.BearerToken = authInfo.BearerToken
	a.Org = authInfo.OrgName
	return nil
}

func (a *API) checkKeyringAuth(authInfo *auth.ResolvedAuth) error {

	// If the auth is expired, we need to refresh the token
	if authInfo.ExpiresAt != nil && time.Now().After(*authInfo.ExpiresAt) {
		if authInfo.RefreshToken == "" {
			return ErrAuthExpired
		}

		newAuthInfo, err := a.refreshKeyringAuth(authInfo)
		if err != nil {
			return err
		}
		*authInfo = *newAuthInfo
		return nil
	}

	// If using API key, just return
	if authInfo.APIKey != "" {
		return nil
	}

	return nil
}

func (a *API) refreshKeyringAuth(authInfo *auth.ResolvedAuth) (*auth.ResolvedAuth, error) {
	if err := a.InitUnauthAPIConfig(); err != nil {
		return nil, err
	}

	params := &sdkauth.AuthDeviceRefreshTokenParams{
		Data: authInfo.RefreshToken,
	}

	resp, err := a.Client.Auth.AuthDeviceRefreshToken(params)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	expiresAt := time.Now().Add(time.Duration(resp.Payload.ExpiresIn) * time.Second)
	authInfo.BearerToken = resp.Payload.AccessToken
	authInfo.RefreshToken = resp.Payload.RefreshToken
	authInfo.ExpiresAt = &expiresAt

	newAuthInfo := auth.Auth{
		APIKey:       authInfo.APIKey,
		BearerToken:  authInfo.BearerToken,
		RefreshToken: authInfo.RefreshToken,
		OrgName:      authInfo.OrgName,
		ExpiresAt:    &expiresAt,
	}

	// Store updated auth in the same location as the original
	var storage auth.Storage
	if authInfo.Source == auth.PlainTextAuthSource {
		storage = auth.NewPlainTextStorage()
	} else {
		storage = auth.NewKeyringStorage()
	}
	if err := storage.Store(&newAuthInfo); err != nil {
		return nil, fmt.Errorf("failed to store refreshed auth: %w", err)
	}

	return authInfo, nil
}

func (a *API) basicInit() error {
	if apiURL := viper.GetString("api_url"); apiURL != "" {
		a.APIURL = apiURL
	} else {
		a.APIURL = "https://api.signadot.com"
	}

	if proxyURL := viper.GetString("proxy_url"); proxyURL != "" {
		_, err := url.Parse(proxyURL)
		if err != nil {
			return fmt.Errorf("invalid proxy_url: %w", err)
		}
		a.ProxyURL = proxyURL
	} else {
		a.ProxyURL = "https://proxy.signadot.com"
	}

	if mcpURL := viper.GetString("mcp_url"); mcpURL != "" {
		a.MCPURL = mcpURL
	} else {
		a.MCPURL = "https://mcp.signadot.com"
	}

	// Allow defining a custom URL for artifacts (useful for local development).
	// Empty means using the API URL from above for accessing artifacts.
	a.ArtifactsAPIURL = viper.GetString("artifacts_api_url")
	a.UserAgent = fmt.Sprintf("signadot-cli:%s", buildinfo.Version)
	return nil
}

// RefreshAPIConfig refreshes the API config by re-initializing the API client
func (a *API) RefreshAPIConfig() error {
	return a.InitAPIConfig()
}

func (a *API) InitAPIConfig() error {
	if err := a.init(); err != nil {
		return err
	}

	return a.InitAPITransport()
}

func (a *API) InitUnauthAPIConfig() error {
	if err := a.basicInit(); err != nil {
		return err
	}
	return a.InitAPITransport()
}

func (a *API) InitAPIConfigWithApiKey(apiKey string) error {
	a.ApiKey = apiKey
	if err := a.basicInit(); err != nil {
		return err
	}
	return a.InitAPITransport()
}

func (a *API) InitAPIConfigWithBearerToken(bearerToken string) error {
	a.BearerToken = bearerToken
	if err := a.basicInit(); err != nil {
		return err
	}
	return a.InitAPITransport()
}

func (a *API) GetBaseTransport() *transport.APIConfig {
	cfg := &transport.APIConfig{
		APIURL:          a.APIURL,
		ArtifactsAPIURL: a.ArtifactsAPIURL,
		UserAgent:       a.UserAgent,
		Debug:           a.Debug,
	}

	// Prefer API key if present
	if a.ApiKey != "" {
		cfg.APIKey = a.ApiKey
	} else if a.BearerToken != "" {
		cfg.BearerToken = a.BearerToken
	}

	return cfg
}

func (a *API) InitAPITransport() error {
	// init API transport
	t, err := transport.InitAPITransport(a.GetBaseTransport())
	if err != nil {
		return err
	}

	// create an API client
	a.Client = client.New(t, nil)
	return nil
}

func (a *API) APIClientWithCustomTransport(conf *transport.APIConfig,
	execute func(client *client.SignadotAPI) error) error {
	if err := a.init(); err != nil {
		return err
	}

	t, err := transport.InitAPITransport(conf)
	if err != nil {
		return err
	}
	return execute(client.New(t, nil))
}
