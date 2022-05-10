package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Root struct {
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
			return err
		}
	}

	c.Debug = viper.GetBool("debug")

	return nil
}
