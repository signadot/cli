package signadot

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type RootCmd struct {
	*cobra.Command

	// Flags
	configFile   string
	outputFormat OutputFormat

	// Config File Values
	org    string
	apiKey string
}

func NewRootCmd() *RootCmd {
	c := &RootCmd{}
	cobra.OnInitialize(c.loadConfig)

	c.Command = &cobra.Command{
		Use:   "signadot",
		Short: "Command-line interface for Signadot",
	}

	c.PersistentFlags().StringVar(&c.configFile, "config", "", "config file (default is $HOME/.signadot/config.yaml)")
	c.PersistentFlags().VarP(&c.outputFormat, "output", "o", "output format (json|yaml)")

	// Subcommands
	addClusterCmd(c)
	addSandboxCmd(c)

	return c
}

func (c *RootCmd) loadConfig() {
	if c.configFile != "" {
		viper.SetConfigFile(c.configFile)
	} else {
		homeDir, err := os.UserHomeDir()
		cobra.CheckErr(err)

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
			cobra.CheckErr(err)
		}
	}

	c.org = viper.GetString("org")
	c.apiKey = viper.GetString("api_key")
}
