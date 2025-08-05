package config

import (
	"fmt"
	"os"
	"path/filepath"

	connectcfg "github.com/signadot/libconnect/config"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
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
	DaemonRun      bool
	RootManager    bool
	SandboxManager bool

	// Hidden Flags
	ConnectInvocationConfigFile string
	PProfAddr                   string
}

func (ld *LocalDaemon) InitLocalDaemon() error {
	var (
		ciBytes []byte
		err     error
	)

	if ld.ConnectInvocationConfigFile != "" {
		ciBytes, err = os.ReadFile(ld.ConnectInvocationConfigFile)
		if err != nil {
			return fmt.Errorf("error reading connect invocation config file %q: %w", ld.ConnectInvocationConfigFile, err)
		}
	} else {
		ciBytes = []byte(os.Getenv("SIGNADOT_LOCAL_CONNECT_INVOCATION_CONFIG"))
		if len(ciBytes) == 0 {
			return fmt.Errorf("expected $SIGNADOT_LOCAL_CONNECT_INVOCATION_CONFIG")
		}
	}
	ciConfig := &ConnectInvocationConfig{}
	if err := yaml.Unmarshal(ciBytes, ciConfig); err != nil {
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
	WithRootManager bool   `json:"withRootManager"`
	APIPort         uint16 `json:"apiPort"`
	LocalNetPort    uint16 `json:"localNetPort"`
	SignadotDir     string `json:"signadotDir"`

	User             *ConnectInvocationUser       `json:"user"`
	Env              []string                     `json:"env"`
	VirtualIPNet     string                       `json:"virtualIPNet"`
	ConnectionConfig *connectcfg.ConnectionConfig `json:"connectionConfig"`
	ProxyURL         string                       `json:"proxyURL"`
	APIKey           string                       `json:"apiKey"`
	Debug            bool                         `json:"debug"`
}

type ConnectInvocationUser struct {
	UID      int    `json:"uid"`
	GID      int    `json:"gid"`
	UIDHome  string `json:"uidHome"`
	UIDPath  string `json:"uidPath"`
	Username string `json:"username"`
}

func (ciConfig *ConnectInvocationConfig) GetPIDfile(isRootManager bool) string {
	if isRootManager {
		return filepath.Join(ciConfig.SignadotDir, RootManagerPIDFile)
	}
	return filepath.Join(ciConfig.SignadotDir, SandboxManagerPIDFile)
}

func (ciConfig *ConnectInvocationConfig) GetLogName(isRootManager bool) string {
	if isRootManager {
		return RootManagerLogFile
	}
	return SandboxManagerLogFile
}

func (c *LocalDaemon) AddFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&c.DaemonRun, "daemon", false, "run in background as daemon")
	cmd.Flags().BoolVar(&c.RootManager, "root-manager", false, "run the root-manager (privileged daemon)")
	cmd.Flags().BoolVar(&c.SandboxManager, "sandbox-manager", false, "run the sandbox-manager (unprivileged daemon)")

	cmd.Flags().StringVar(&c.ConnectInvocationConfigFile, "ci-config-file", "", "by-pass calling signadot local connect (hidden)")
	cmd.Flags().MarkHidden("ci-config-file")
	cmd.Flags().StringVar(&c.PProfAddr, "pprof", "", "pprof listen address")
	cmd.Flags().MarkHidden("pprof")
}
