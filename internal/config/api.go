package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/signadot/cli/internal/auth"
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
	authInfo, err := auth.ResolveAuth()
	if err != nil {
		return fmt.Errorf("could not resolve auth: %w", err)
	}

	if authInfo == nil || (authInfo.APIKey == "" && authInfo.BearerToken == "") {
		return errors.New("No authentication found. Please log in using 'signadot auth login'")
	}
	if authInfo.ExpiresAt != nil && authInfo.ExpiresAt.Before(time.Now()) {
		return errors.New("Authentication expired. Please log in using 'signadot auth login'")
	}
	if authInfo.OrgName == "" {
		return errors.New("No organisation found. Please log in using 'signadot auth login'")
	}

	a.ApiKey = authInfo.APIKey
	a.BearerToken = authInfo.BearerToken
	a.Org = authInfo.OrgName

	// Init basic settings and return
	a.basicInit()
	return nil
}

func (a *API) basicInit() {
	if apiURL := viper.GetString("api_url"); apiURL != "" {
		a.APIURL = apiURL
	} else {
		a.APIURL = "https://api.signadot.com"
	}

	// Allow defining a custom URL for artifacts (useful for local development).
	// Empty means using the API URL from above for accessing artifacts.
	a.ArtifactsAPIURL = viper.GetString("artifacts_api_url")
	a.UserAgent = fmt.Sprintf("signadot-cli:%s", buildinfo.Version)
}

func (a *API) InitAPIConfig() error {
	if err := a.init(); err != nil {
		return err
	}

	return a.InitAPITransport()
}

func (a *API) InitUnauthAPIConfig() error {
	a.basicInit()
	return a.InitAPITransport()
}

func (a *API) InitAPIConfigWithApiKey(apiKey string) error {
	a.ApiKey = apiKey
	a.basicInit()
	return a.InitAPITransport()
}

func (a *API) InitAPIConfigWithBearerToken(bearerToken string) error {
	a.BearerToken = bearerToken
	a.basicInit()
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
