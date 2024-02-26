package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	oaclient "github.com/go-openapi/runtime/client"
	"github.com/signadot/cli/internal/buildinfo"
	"github.com/signadot/cli/internal/hack"
	"github.com/signadot/go-sdk/client"
	"github.com/signadot/libconnect/common"
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

	tc := client.DefaultTransportConfig()

	// Allow API URL to be overridden (e.g. for talking to dev/staging).
	if apiURL := viper.GetString("api_url"); apiURL != "" {
		u, err := url.Parse(apiURL)
		if err != nil {
			return fmt.Errorf("invalid api_url: %w", err)
		}
		tc.Host = u.Host
		tc.Schemes = []string{u.Scheme}
		a.APIURL = apiURL

	} else {
		a.APIURL = "https://api.signadot.com"
	}

	// Add auth info to every request.
	transport := oaclient.New(tc.Host, tc.BasePath, tc.Schemes)
	transport.DefaultAuthentication = oaclient.APIKeyAuth(common.APIKeyHeader, "header", apiKey)
	transport.SetDebug(a.Debug)

	// Add User-Agent to every request
	transport.Transport = &userAgent{
		inner: transport.Transport,
		agent: fmt.Sprintf("signadot-cli:%s", buildinfo.Version),
	}

	a.Client = client.New(hack.FixAPIErrors(transport), nil)

	return nil
}

type userAgent struct {
	inner http.RoundTripper
	agent string
}

func (ua *userAgent) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("User-Agent", ua.agent)
	return ua.inner.RoundTrip(r)
}
