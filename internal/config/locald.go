package config

import (
	"encoding/json"
	"fmt"
	"os"

	connectcfg "github.com/signadot/libconnect/config"
	"github.com/spf13/cobra"
)

const (
	RootManagerPIDFile    = "rootmanager.pid"
	SandboxManagerPIDFile = "sandboxmanager.pid"
)

type LocalDaemon struct {
	*Local

	// config sent from `signadot local connect` in $SIGNADOT_LOCAL_CONNECT_CONFIG
	ConnectInvocationConfig *ConnectInvocationConfig

	// Hidden Flags
	ConnectInvocationConfigFile string
}

func (ld *LocalDaemon) InitLocalDaemon() error {
	var (
		ciBytes []byte
		err     error
	)
	if ld.ConnectInvocationConfigFile != "" {
		ciBytes, err = os.ReadFile(ld.ConnectInvocationConfigFile)
		if err != nil {
			return err
		}
	} else {
		ciBytes = []byte(os.Getenv("SIGNADOT_LOCAL_CONNECT_INVOCATION_CONFIG"))
		if len(ciBytes) == 0 {
			return fmt.Errorf("expected $SIGNADOT_LOCAL_CONNECT_INVOCATION_CONFIG")
		}
	}
	ciConfig := &ConnectInvocationConfig{}
	if err := json.Unmarshal(ciBytes, ciConfig); err != nil {
		return err
	}
	ld.ConnectInvocationConfig = ciConfig
	return nil

}

// ConnectInvocationConfig is the config for locald as computed
// by `signadot local connect` when `signadot local connect`
// is called.  This prevents racy behavior when the config file
// is edited and facilitates wrapping everything that needs to be
// passed in a json so we can evolve what needs to be passed
// without plumbing the command line
type ConnectInvocationConfig struct {
	Unpriveleged     bool                         `json:"unpriveleged"`
	Cluster          string                       `json:"cluster"`
	APIPort          uint16                       `json:"apiPort"`
	LocalNetPort     uint16                       `json:"localNetPort"`
	SignadotDir      string                       `json:"signadotDir"`
	UID              int                          `json:"uid"`
	ConnectionConfig *connectcfg.ConnectionConfig `json:"connectionConfig"`
	API              *API                         `json:"api"`
}

func (c *LocalDaemon) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&c.ConnectInvocationConfigFile, "connect-invocation-config-file", "", "by-pass calling signadot local connect (hidden)")
	cmd.Flags().MarkHidden("connect-invocation-config-file")
}
