package sandboxmanager

import (
	"context"
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

func GetStatus() (*sbmapi.StatusResponse, error) {
	// Get a sandbox manager API client
	grpcConn, err := grpc.Dial("127.0.0.1:6666", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("couldn't connect sandboxmanager: %w", err)
	}
	defer grpcConn.Close()

	// get the status
	sbManagerClient := sbmapi.NewSandboxManagerAPIClient(grpcConn)
	sbStatus, err := sbManagerClient.Status(context.Background(), &sbmapi.StatusRequest{})
	if err != nil {
		grpcStatus, ok := status.FromError(err)
		if ok {
			switch grpcStatus.Code() {
			case codes.Unavailable:
				return nil, fmt.Errorf("sandboxmanager is not running, start it with \"signadot local connect\"")
			}
		}
		return nil, fmt.Errorf("unable to get status from sandboxmanager: %w", err)
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

	// check port forward status
	if ciConfig.ConnectionConfig.Type == connectcfg.PortForwardLinkType {
		err := checkPortforwardStatus(status.Portforward)
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
	return fmt.Errorf(errorMsg)
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
	return fmt.Errorf(errorMsg)
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
	return fmt.Errorf(errorMsg)
}
