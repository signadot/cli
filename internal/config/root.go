package config

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Root struct {
	// Config file values
	DashboardURL *url.URL

	// Flags
	Debug        bool
	ConfigFile   string
	OutputFormat OutputFormat
}

func (c *Root) AddFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().Bool("debug", false, "enable debug output")
	viper.BindPFlag("debug", cmd.PersistentFlags().Lookup("debug"))

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
		// The config file is optional (required params (org, apikey) can
		// be set by env var instead).
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("error reading config file: %w", err)
		}
	}

	c.Debug = viper.GetBool("debug")

	if dashURL := viper.GetString("dashboard_url"); dashURL != "" {
		u, err := url.Parse(dashURL)
		if err != nil {
			return fmt.Errorf("invalid dashboard_url: %w", err)
		}
		c.DashboardURL = u
	} else {
		c.DashboardURL = &url.URL{
			Scheme: "https",
			Host:   "app.signadot.com",
		}
	}

	return nil
}

func (c *Root) SandboxDashboardURL(id string) *url.URL {
	u := *c.DashboardURL
	u.Path = path.Join(u.Path, "sandbox", "id", id)
	return &u
}
