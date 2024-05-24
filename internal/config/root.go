package config

import (
	"fmt"
	"net/url"
	"path"
	"strconv"

	"github.com/signadot/cli/internal/utils/system"
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
	cmd.PersistentFlags().BoolVar(&c.Debug, "debug", false, "enable debug output")
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
		signadotDir, err := system.GetSignadotDir()
		if err != nil {
			return err
		}

		viper.AddConfigPath(signadotDir)
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

	if !c.Debug {
		c.Debug = viper.GetBool("debug")
	}

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

func (c *Root) RunnerGroupDashboardUrl(name string) *url.URL {
	u := *c.DashboardURL
	u.Path = path.Join(u.Path, "testing", "runner-groups", name)
	return &u
}

func (c *Root) JobDashboardUrl(name string) *url.URL {
	u := *c.DashboardURL
	u.Path = path.Join(u.Path, "testing", "jobs", name)
	return &u
}

func (c *Root) ArtifactDownloadUrl(org, jobName string, attemptID int64, fileName string) *url.URL {
	u := url.URL{
		Scheme:   "https",
		Path:     path.Join("api.staging.signadot.com/api/v2/orgs/", org, "artifacts", "jobs", jobName, "attempts", strconv.FormatInt(attemptID, 10), "objects/download"),
		RawQuery: "path=" + fileName,
	}
	return &u
	//https://api.staging.signadot.com/api/v2/orgs/signadot/artifacts/jobs/first-job/attempts/first-job-0/objects/download?path=/tmp/test.txt
}
