package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/go-openapi/runtime"
	"github.com/signadot/cli/internal/auth"
	"github.com/signadot/go-sdk/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Root struct {
	// Flags
	ConfigFile   string
	OutputFormat OutputFormat

	// Config file values
	Org string

	// Runtime values
	Client   *client.SignadotAPI
	AuthInfo runtime.ClientAuthInfoWriter
}

func (c *Root) AddFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&c.ConfigFile, "config", "", "config file (default is $HOME/.signadot/config.yaml)")
	cmd.PersistentFlags().VarP(&c.OutputFormat, "output", "o", "output format (json|yaml)")
}

func (c *Root) Init() {
	cobra.CheckErr(c.init())
}

func (c *Root) init() error {
	if c.ConfigFile != "" {
		viper.SetConfigFile(c.ConfigFile)
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		viper.AddConfigPath(filepath.Join(homeDir, ".signadot"))
		viper.SetConfigName("config") // Doesn't include extension.
		viper.SetConfigType("yaml")   // File name will be "config.yaml".
	}

	viper.SetEnvPrefix("signadot")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		// The config file is optional since required params (org, apikey) can
		// be set by env var instead.
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	c.Org = viper.GetString("org")
	if c.Org == "" {
		return errors.New("Signadot Org name must be specified through either the SIGNADOT_ORG env var or the 'org' field in ~/.signadot/config.yaml")
	}

	apiKey := viper.GetString("api_key")
	if apiKey == "" {
		return errors.New("Signadot API key must be specified through either the SIGNADOT_API_KEY env var or the 'api_key' field in ~/.signadot/config.yaml")
	}

	c.Client = client.Default
	c.AuthInfo = auth.Authenticator(apiKey)

	// Allow API URL to be overridden (e.g. for talking to dev/staging).
	if apiURL := viper.GetString("api_url"); apiURL != "" {
		u, err := url.Parse(apiURL)
		if err != nil {
			return fmt.Errorf("invalid api_url: %w", err)
		}
		c.Client = client.NewHTTPClientWithConfig(nil, &client.TransportConfig{
			Host:     u.Host,
			BasePath: client.DefaultBasePath,
			Schemes:  []string{u.Scheme},
		})
	}

	return nil
}
