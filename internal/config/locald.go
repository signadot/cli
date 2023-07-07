package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	connectcfg "github.com/signadot/libconnect/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	RootManagerPIDFile    = "rootmanager.pid"
	SandboxManagerPIDFile = "sandboxmanager.pid"

	RootManagerLogFile    = "root-manager.log"
	SandboxManagerLogFile = "sandbox-manager.log"
)

type LocalDaemon struct {
	// config sent from `signadot local connect` in $SIGNADOT_LOCAL_CONNECT_CONFIG
	ConnectInvocationConfig *ConnectInvocationConfig

	// Flags
	DaemonRun bool

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

	viper.Set("api_url", ciConfig.API.APIURL)
	viper.Set("api_key", ciConfig.APIKey)
	if err := ciConfig.API.InitAPITransport(ciConfig.APIKey); err != nil {
		return err
	}
	ld.ConnectInvocationConfig = ciConfig
	return nil

}

// ConnectInvocationConfig is the config for locald as computed by `signadot
// local connect` when `signadot local connect` is called.  This prevents racy
// behavior when the config file is edited, allows config to be computed by
// non-root user and used subsequently by root, and facilitates wrapping
// everything that needs to be passed in a json so we can evolve what needs to
// be passed without plumbing the command line
type ConnectInvocationConfig struct {
	Unprivileged     bool                         `json:"unprivileged"`
	Cluster          string                       `json:"cluster"`
	APIPort          uint16                       `json:"apiPort"`
	LocalNetPort     uint16                       `json:"localNetPort"`
	SignadotDir      string                       `json:"signadotDir"`
	UID              int                          `json:"uid"`
	GID              int                          `json:"gid"`
	UIDHome          string                       `json:"uidHome"`
	UIDPath          string                       `json:"uidPath"`
	ConnectionConfig *connectcfg.ConnectionConfig `json:"connectionConfig"`
	API              *API                         `json:"api"`
	APIKey           string                       `json:"apiKey"`
	Debug            bool                         `json:"debug"`
}

func (ciConfig *ConnectInvocationConfig) GetPIDfile() string {
	if !ciConfig.Unprivileged {
		return filepath.Join(ciConfig.SignadotDir, RootManagerPIDFile)
	}
	return filepath.Join(ciConfig.SignadotDir, SandboxManagerPIDFile)
}

func (ciConfig *ConnectInvocationConfig) GetLogName() string {
	if !ciConfig.Unprivileged {
		return RootManagerLogFile
	}
	return SandboxManagerLogFile
}

func (c *LocalDaemon) AddFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&c.DaemonRun, "daemon", false, "run in background as daemon")

	cmd.Flags().StringVar(&c.ConnectInvocationConfigFile, "connect-invocation-config-file", "", "by-pass calling signadot local connect (hidden)")
	cmd.Flags().MarkHidden("connect-invocation-config-file")
}
