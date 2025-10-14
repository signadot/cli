package local

import (
	"fmt"
	"path/filepath"

	"github.com/signadot/cli/internal/config"
	sbmapi "github.com/signadot/cli/internal/locald/api/sandboxmanager"
	sbmgr "github.com/signadot/cli/internal/locald/sandboxmanager"
	"github.com/signadot/cli/internal/utils/system"
	"github.com/signadot/libconnect/common/processes"
	lcconfig "github.com/signadot/libconnect/config"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetLocalKubeClient() (client.Client, error) {
	st, err := GetLocalStatus()
	if err != nil {
		return nil, err
	}
	ciConfig, err := sbmapi.ToCIConfig(st.CiConfig)
	if err != nil {
		return nil, fmt.Errorf("couldn't unmarshal ci-config from sandboxmanager status, %v", err)
	}
	if ciConfig.ConnectionConfig == nil {
		return nil, fmt.Errorf("no connection config")
	}
	connConfig := ciConfig.ConnectionConfig
	restConfig, err := lcconfig.GetRESTConfig(connConfig.GetKubeConfigPath(),
		connConfig.KubeContext)
	if err != nil {
		return nil, err
	}
	return client.New(restConfig, client.Options{})
}

func GetLocalStatus() (*sbmapi.StatusResponse, error) {
	// Get the signadot dir
	signadotDir, err := system.GetSignadotDir()
	if err != nil {
		return nil, err
	}

	// Make sure the sandbox manager is running
	pidfile := filepath.Join(signadotDir, config.SandboxManagerPIDFile)
	isRunning, err := processes.IsDaemonRunning(pidfile)
	if err != nil {
		return nil, err
	}
	if !isRunning {
		return nil, fmt.Errorf("signadot is not connected\n")
	}

	// Get the status from sandbox manager
	status, err := sbmgr.GetStatus()
	if err != nil {
		return nil, err
	}
	return status, nil
}
