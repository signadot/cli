package config

import (
	"errors"
	"fmt"
	"net/url"

	oaclient "github.com/go-openapi/runtime/client"
	"github.com/signadot/cli/internal/hack"
	"github.com/signadot/go-sdk/client"
	"github.com/spf13/viper"
)

type Api struct {
	Root

	// Config file values
	Org string

	// Runtime values
	Client *client.SignadotAPI
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

	tc := client.DefaultTransportConfig()

	// Allow API URL to be overridden (e.g. for talking to dev/staging).
	if apiURL := viper.GetString("api_url"); apiURL != "" {
		u, err := url.Parse(apiURL)
		if err != nil {
			return fmt.Errorf("invalid api_url: %w", err)
		}
		tc.Host = u.Host
		tc.Schemes = []string{u.Scheme}
	}

	// Add auth info to every request.
	transport := oaclient.New(tc.Host, tc.BasePath, tc.Schemes)
	transport.DefaultAuthentication = oaclient.APIKeyAuth("signadot-api-key", "header", apiKey)
	transport.SetDebug(a.Debug)

	a.Client = client.New(hack.FixAPIErrors(transport), nil)

	return nil
}
