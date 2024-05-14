package config

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/signadot/cli/internal/buildinfo"
	"github.com/signadot/go-sdk/client"
	"github.com/signadot/go-sdk/transport"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

type API struct {
	Root

	// Config file values
	Org          string
	MaskedAPIKey string
	APIURL       string

	// Runtime values
	Client *client.SignadotAPI `json:"-"`
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
	}
	t := &T{
		Debug:        a.Debug,
		ConfigFile:   a.ConfigFile,
		Org:          a.Org,
		MaskedAPIKey: a.MaskedAPIKey,
		APIURL:       a.APIURL,
	}
	return marshaller(t)
}

func (a *API) InitAPIConfig() error {
	apiKey := viper.GetString("api_key")
	if apiKey == "" {
		return errors.New("Signadot API key must be specified through either the SIGNADOT_API_KEY env var or the 'api_key' field in ~/.signadot/config.yaml")
	} else {
		a.MaskedAPIKey = apiKey[:6] + "..."
	}

	a.Org = viper.GetString("org")
	if a.Org == "" {
		return errors.New("Signadot Org name must be specified through either the SIGNADOT_ORG env var or the 'org' field in ~/.signadot/config.yaml")
	}

	return a.InitAPITransport(apiKey)
}

func (a *API) InitAPITransport(apiKey string) error {
	// Allow API URL to be overridden (e.g. for talking to dev/staging).
	if apiURL := viper.GetString("api_url"); apiURL != "" {
		a.APIURL = apiURL

	} else {
		a.APIURL = "https://api.signadot.com"
	}
	// Allow defining a custom URL for artifacts (useful for local development).
	// Empty means using the API URL from above for accessing artifacts.
	artifactsAPIURL := viper.GetString("artifacts_api_url")

	// init API transport
	transport, err := transport.InitAPITransport(&transport.APIConfig{
		APIKey:          apiKey,
		APIURL:          a.APIURL,
		ArtifactsAPIURL: artifactsAPIURL,
		UserAgent:       fmt.Sprintf("signadot-cli:%s", buildinfo.Version),
		Debug:           a.Debug,
	})
	if err != nil {
		return err
	}

	// create an API client
	a.Client = client.New(transport, nil)
	return nil
}
