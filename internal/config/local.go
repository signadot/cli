package config

import (
	"fmt"
	"os"

	"github.com/signadot/libconnect/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"sigs.k8s.io/yaml"
)

type Local struct {
	*API

	// initialized from ~/.signadot/config.yaml
	LocalConfig *config.Config
}

func (l *Local) InitLocalConfig() error {
	if err := l.API.InitAPIConfig(); err != nil {
		return err
	}

	type Tmp struct {
		Local *config.Config `json:"local"`
	}
	localConfig := &Tmp{}
	d, e := os.ReadFile(viper.ConfigFileUsed())
	if e != nil {
		return e
	}
	if e := yaml.Unmarshal(d, localConfig); e != nil {
		return e
	}
	if localConfig.Local == nil {
		return fmt.Errorf("no local section in %s", viper.ConfigFileUsed())
	}
	if err := localConfig.Local.Validate(); err != nil {
		return err
	}
	if len(localConfig.Local.Connections) == 0 {
		return fmt.Errorf("no connections in local section in %s", viper.ConfigFileUsed())
	}
	if !localConfig.Local.Debug {
		localConfig.Local.Debug = l.Debug
	}
	l.LocalConfig = localConfig.Local
	return nil
}

func (l *Local) GetConnectionConfig(cluster string) (*config.ConnectionConfig, error) {
	conns := l.LocalConfig.Connections
	clusters := make([]string, len(conns))
	for i := range conns {
		clusters[i] = conns[i].Cluster
	}
	if cluster == "" {
		if len(conns) == 1 {
			return &conns[0], nil
		}
		return nil, fmt.Errorf("must specify --cluster=... (one of %v)", clusters)
	}
	for i := range conns {
		connConfig := &conns[i]
		if connConfig.Cluster == cluster {
			return connConfig, nil
		}
	}
	return nil, fmt.Errorf("no such cluster %q, expecting one of %v", cluster, clusters)
}

type LocalConnect struct {
	*Local

	// Flags
	NonInteractive bool
	Cluster        string

	// Hidden Flags
	Unprivileged bool
}

func (c *LocalConnect) AddFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&c.NonInteractive, "non-interactive", false, "run in background")
	cmd.Flags().StringVar(&c.Cluster, "cluster", "", "signadot cluster name")

	cmd.Flags().BoolVar(&c.Unprivileged, "unprivileged", false, "run without root priveleges")
	cmd.Flags().MarkHidden("unprivileged")
}

type LocalDisconnect struct {
	*Local
}

type LocalStatus struct {
	*Local
}
