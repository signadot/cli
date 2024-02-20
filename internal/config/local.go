package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/signadot/libconnect/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"sigs.k8s.io/yaml"
)

const (
	DefaultVirtualIPNet = "242.242.0.1/16"
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
	if localConfig.Local.VirtualIPNet == "" {
		localConfig.Local.VirtualIPNet = DefaultVirtualIPNet
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
	Cluster      string
	Unprivileged bool
	NoWait       bool
	WaitTimeout  time.Duration

	// Hidden Flags
	DumpCIConfig bool
}

func (c *LocalConnect) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&c.Cluster, "cluster", "", "specify cluster connection config")

	cmd.Flags().BoolVar(&c.Unprivileged, "unprivileged", false, "run without root privileges")
	cmd.Flags().BoolVar(&c.NoWait, "no-wait", false, "don't wait for connection healthy")
	cmd.Flags().AddGoFlag(&flag.Flag{
		Name: "wait",
		Value: &waitFlagValue{
			negPointer: &c.NoWait,
		},
		DefValue: "true",
		Usage:    "wait for the connection to become healthy",
	})
	cmd.Flags().DurationVar(&c.WaitTimeout, "wait-timeout", 10*time.Second, "timeout to wait")

	cmd.Flags().BoolVar(&c.DumpCIConfig, "dump-ci-config", false, "dump connect invocation config")
	cmd.Flags().MarkHidden("dump-ci-config")
	cmd.Flags().MarkHidden("wait")
}

type waitFlagValue struct {
	negPointer *bool
}

func (w *waitFlagValue) Set(v string) error {
	b, err := strconv.ParseBool(v)
	if err != nil {
		return err
	}
	*w.negPointer = !b
	return nil
}

func (w *waitFlagValue) String() string {
	if w == nil || w.negPointer == nil {
		return "false"
	}
	return fmt.Sprintf("%t", *w.negPointer)
}

func (w *waitFlagValue) IsBoolFlag() bool {
	return true
}

type LocalDisconnect struct {
	*Local

	// Flags
	CleanLocalSandboxes bool
}

func (c *LocalDisconnect) AddFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&c.CleanLocalSandboxes, "clean-local-sandboxes", false, "clean active local sandboxes")
}

type LocalStatus struct {
	*Local

	// Flags
	Details bool
}

func (c *LocalStatus) AddFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&c.Details, "details", false, "display status details")
}
