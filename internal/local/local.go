package local

import (
	"fmt"
	"path/filepath"

	rolloutapi "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	"github.com/signadot/cli/internal/config"
	sbmapi "github.com/signadot/cli/internal/locald/api/sandboxmanager"
	sbmgr "github.com/signadot/cli/internal/locald/sandboxmanager"
	"github.com/signadot/cli/internal/utils/system"
	"github.com/signadot/libconnect/common/processes"
	lcconfig "github.com/signadot/libconnect/config"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
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
	s := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(s); err != nil {
		return nil, fmt.Errorf("couldn't add client-go types to scheme: %w", err)
	}
	if err := rolloutapi.AddToScheme(s); err != nil {
		return nil, fmt.Errorf("couldn't add argo rollout types to scheme: %w", err)
	}
	return client.New(restConfig, client.Options{Scheme: s})
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
