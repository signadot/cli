package config

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/go-openapi/runtime"
	"github.com/signadot/cli/internal/auth"
	"github.com/signadot/go-sdk/client"
	"github.com/spf13/viper"
)

type Api struct {
	Root

	// Config file values
	Org string

	// Runtime values
	Client   *client.SignadotAPI
	AuthInfo runtime.ClientAuthInfoWriter
}

func (a *Api) InitAPIConfig() error {

	a.Org = viper.GetString("org")
	if a.Org == "" {
		return errors.New("Signadot Org name must be specified through either the SIGNADOT_ORG env var or the 'org' field in ~/.signadot/config.yaml")
	}

	apiKey := viper.GetString("api_key")
	if apiKey == "" {
		return errors.New("Signadot API key must be specified through either the SIGNADOT_API_KEY env var or the 'api_key' field in ~/.signadot/config.yaml")
	}

	a.Client = client.Default
	a.AuthInfo = auth.Authenticator(apiKey)

	// Allow API URL to be overridden (e.g. for talking to dev/staging).
	if apiURL := viper.GetString("api_url"); apiURL != "" {
		u, err := url.Parse(apiURL)
		if err != nil {
			return fmt.Errorf("invalid api_url: %w", err)
		}
		a.Client = client.NewHTTPClientWithConfig(nil, &client.TransportConfig{
			Host:     u.Host,
			BasePath: client.DefaultBasePath,
			Schemes:  []string{u.Scheme},
		})
	}
	return nil
}
