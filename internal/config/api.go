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
	Org             string
	MaskedAPIKey    string
	APIURL          string
	ArtifactsAPIURL string

	// Runtime values
	Client *client.SignadotAPI `json:"-"`

	ApiKey    string
	UserAgent string
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

func (a *API) init() error {
	apiKey := viper.GetString("api_key")
	if apiKey == "" {
		return errors.New("Signadot API key must be specified through either the SIGNADOT_API_KEY env var or the 'api_key' field in ~/.signadot/config.yaml")
	} else {
		a.MaskedAPIKey = apiKey[:6] + "..."
	}
	a.ApiKey = apiKey

	a.Org = viper.GetString("org")
	if a.Org == "" {
		return errors.New("Signadot Org name must be specified through either the SIGNADOT_ORG env var or the 'org' field in ~/.signadot/config.yaml")
	}

	if apiURL := viper.GetString("api_url"); apiURL != "" {
		a.APIURL = apiURL

	} else {
		a.APIURL = "https://api.signadot.com"
	}

	// Allow defining a custom URL for artifacts (useful for local development).
	// Empty means using the API URL from above for accessing artifacts.
	a.ArtifactsAPIURL = viper.GetString("artifacts_api_url")

	a.UserAgent = fmt.Sprintf("signadot-cli:%s", buildinfo.Version)
	return nil
}

func (a *API) InitAPIConfig() error {
	if err := a.init(); err != nil {
		return err
	}

	return a.InitAPITransport()
}

func (a *API) GetBaseTransport() *transport.APIConfig {
	return &transport.APIConfig{
		APIKey:          a.ApiKey,
		APIURL:          a.APIURL,
		ArtifactsAPIURL: a.ArtifactsAPIURL,
		UserAgent:       a.UserAgent,
		Debug:           a.Debug,
	}
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
