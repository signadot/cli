package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/signadot/libconnect/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/client-go/util/homedir"
	"sigs.k8s.io/yaml"
)

type Local struct {
	*API
	LocalNet *config.Config `json:"local"`
}

func (l *Local) InitLocalConfig() error {
	p := filepath.Join(homedir.HomeDir(), ".signadot", "config.yaml")
	d, e := os.ReadFile(p)
	if e != nil {
		return e
	}
	ln := &config.Config{}
	if err := yaml.Unmarshal(d, ln); err != nil {
		return err
	}
	if ln == nil {
		return fmt.Errorf("no local section in $HOME/signadot/config.yaml")
	}
	if err := ln.Validate(); err != nil {
		return err
	}
	if ln.Debug == false {
		ln.Debug = viper.GetBool("debug")
	}

	if ln.Debug {
	}

	l.LocalNet = ln

	return nil
}

type LocalConnect struct {
	*Local

	// Flags
	NonInteractive bool
}

func (c *LocalConnect) AddFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&c.NonInteractive, "non-interactive", false, "run in background")
}

type LocalDisconnect struct {
	*Local
}
