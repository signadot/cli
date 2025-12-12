package sandboxmanager

import (
	"context"
	"errors"
	"fmt"

	"github.com/signadot/cli/internal/config"
	commonapi "github.com/signadot/cli/internal/locald/api"
	sbmapi "github.com/signadot/cli/internal/locald/api/sandboxmanager"
	connectcfg "github.com/signadot/libconnect/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

var (
	ErrSandboxManagerUnavailable = errors.New(
		`sandboxmanager is not running, start it with "signadot local connect"`)
	ErrTunnelAPIMethodUnimplemented = errors.New("tunnel-api unsupported method")
)

func GetStatus() (*sbmapi.StatusResponse, error) {
	// get a sandbox manager API client
	grpcConn, err := connectSandboxManager()
	if err != nil {
		return nil, err
	}
	defer grpcConn.Close()

	// get the status
	sbManagerClient := sbmapi.NewSandboxManagerAPIClient(grpcConn)
	sbStatus, err := sbManagerClient.Status(context.Background(), &sbmapi.StatusRequest{})
	if err != nil {
		return nil, processGRPCError("unable to get status from sandboxmanager", err)
	}
	return sbStatus, nil
}

func CheckStatusConnectErrors(status *sbmapi.StatusResponse, ciConfig *config.ConnectInvocationConfig) []error {
	var errs []error

	// decode config (if needed)
	if ciConfig == nil {
		var err error
		ciConfig, err = sbmapi.ToCIConfig(status.CiConfig)
		if err != nil {
			return append(errs, fmt.Errorf("couldn't unmarshal ci-config from sandboxmanager status, %v", err))
		}
	}

	switch ciConfig.ConnectionConfig.Type {
	case connectcfg.PortForwardLinkType:
		// check port forward status
		err := checkPortforwardStatus(status.Portforward)
		if err != nil {
			errs = append(errs, err)
		}
	case connectcfg.ControlPlaneProxyLinkType:
		// check control-plane proxy status
		err := checkControlPlaneProxyStatus(status.ControlPlaneProxy)
		if err != nil {
			errs = append(errs, err)
		}
	}

	// check root manager (if running)
	if ciConfig.WithRootManager {
		// check localnet service
		err := checkLocalNetStatus(status.Localnet)
		if err != nil {
			errs = append(errs, err)
		}
		// check hosts service
		err = checkHostsStatus(status.Hosts)
		if err != nil {
			errs = append(errs, err)
		}
	}

	// check devbox session health
	err := checkDevboxSessionStatus(status.DevboxSession)
	if err != nil {
		errs = append(errs, err)
	}

	return errs
}

func checkPortforwardStatus(portforward *commonapi.PortForwardStatus) error {
	errorMsg := "failed to establish port-forward"
	if portforward != nil {
		if portforward.Health != nil {
			if portforward.Health.Healthy {
				return nil
			}
			if portforward.Health.LastErrorReason != "" {
				errorMsg += fmt.Sprintf(" (%q)", portforward.Health.LastErrorReason)
			}
		}
	}
	return errors.New(errorMsg)
}

func checkControlPlaneProxyStatus(ctlPlaneProxy *commonapi.ControlPlaneProxyStatus) error {
	errorMsg := "failed to establish control-plane proxy"
	if ctlPlaneProxy != nil {
		if ctlPlaneProxy.Health != nil {
			if ctlPlaneProxy.Health.Healthy {
				return nil
			}
			if ctlPlaneProxy.Health.LastErrorReason != "" {
				errorMsg += fmt.Sprintf(" (%q)", ctlPlaneProxy.Health.LastErrorReason)
			}
		}
	}
	return errors.New(errorMsg)
}

func checkLocalNetStatus(localnet *commonapi.LocalNetStatus) error {
	errorMsg := "failed to setup localnet"
	if localnet != nil {
		if localnet.Health != nil {
			if localnet.Health.Healthy {
				return nil
			}
			if localnet.Health.LastErrorReason != "" {
				errorMsg += fmt.Sprintf(" (%q)", localnet.Health.LastErrorReason)
			}
		}
	}
	return errors.New(errorMsg)
}

func checkHostsStatus(hosts *commonapi.HostsStatus) error {
	errorMsg := "failed to configure hosts in /etc/hosts"
	if hosts != nil {
		if hosts.Health != nil {
			if hosts.Health.Healthy {
				return nil
			}
			if hosts.Health.LastErrorReason != "" {
				errorMsg += fmt.Sprintf(" (%q)", hosts.Health.LastErrorReason)
			}
		}
	}
	return errors.New(errorMsg)
}

func checkDevboxSessionStatus(devboxSession *commonapi.DevboxSessionStatus) error {
	if devboxSession == nil {
		// Devbox session not initialized yet, not an error
		return nil
	}
	if !devboxSession.Healthy {
		errorMsg := "devbox session unhealthy"
		if devboxSession.LastErrorReason != "" {
			errorMsg += fmt.Sprintf(" (%q)", devboxSession.LastErrorReason)
		}
		return errors.New(errorMsg)
	}
	return nil
}

func IsWatcherRunning(status *sbmapi.StatusResponse) (bool, string) {
	if status == nil || status.Watcher == nil || status.Watcher.Health == nil {
		return false, ""
	}
	return status.Watcher.Health.Healthy, status.Watcher.Health.LastErrorReason
}

type ResourceOutput struct {
	Resource string `json:"resource"`
	Output   string `json:"output"`
	Value    string `json:"value"`
}

func GetResourceOutputs(ctx context.Context, sbRoutingKey string) ([]ResourceOutput, error) {
	grpcConn, err := connectSandboxManager()
	if err != nil {
		return nil, err
	}
	defer grpcConn.Close()

	// get the status
	sbManagerClient := sbmapi.NewSandboxManagerAPIClient(grpcConn)
	req := &sbmapi.GetResourceOutputsRequest{
		SandboxRoutingKey: sbRoutingKey,
	}
	resp, err := sbManagerClient.GetResourceOutputs(ctx, req)
	if err != nil {
		return nil, processGRPCError("unable to get resource outputs from sandboxmanager", err)
	}
	res := []ResourceOutput{}
	for _, ro := range resp.ResourceOutputs {
		for _, out := range ro.Outputs {
			res = append(res, ResourceOutput{Resource: ro.ResourceName, Output: out.Key, Value: out.Value})
		}
	}
	return res, nil
}

func connectSandboxManager() (*grpc.ClientConn, error) {
	grpcConn, err := grpc.NewClient("127.0.0.1:6666", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("couldn't connect sandboxmanager: %w", err)
	}
	return grpcConn, nil
}

// ValidateSandboxManager validates that sandboxmanager is running, connected to the right cluster,
// and returns the status and a sandbox with devbox session ID set if needed.
// This function is useful for operations that require local sandbox functionality.
func ValidateSandboxManager(expectedCluster *string) (*sbmapi.StatusResponse, error) {
	// Get sandboxmanager status
	status, err := GetStatus()
	if err != nil {
		return nil, err
	}

	// Parse CI config
	ciConfig, err := sbmapi.ToCIConfig(status.CiConfig)
	if err != nil {
		return nil, fmt.Errorf("couldn't unmarshal ci-config from sandboxmanager status, %v", err)
	}

	// Check for connection errors
	connectErrs := CheckStatusConnectErrors(status, ciConfig)
	if len(connectErrs) != 0 {
		return nil, fmt.Errorf("sandboxmanager is still starting")
	}

	// Validate cluster matches
	if expectedCluster != nil && *expectedCluster != ciConfig.ConnectionConfig.Cluster {
		return nil, fmt.Errorf("sandbox spec cluster %q does not match connected cluster (%q)",
			expectedCluster, ciConfig.ConnectionConfig.Cluster)
	}

	return status, nil
}

func processGRPCError(action string, err error) error {
	grpcStatus, ok := status.FromError(err)
	if ok {
		switch grpcStatus.Code() {
		case codes.Unavailable:
			return ErrSandboxManagerUnavailable
		case codes.Unimplemented:
			return fmt.Errorf("%w: consider upgrading the operator", ErrTunnelAPIMethodUnimplemented)
		}
	}
	return fmt.Errorf("%s: %w", action, err)
}
