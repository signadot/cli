package local

import (
	"fmt"
	"io"

	"github.com/fatih/color"
	"github.com/signadot/cli/internal/config"
	commonapi "github.com/signadot/cli/internal/locald/api"
	sbmapi "github.com/signadot/cli/internal/locald/api/sandboxmanager"
	connectcfg "github.com/signadot/libconnect/config"
)

func printRawStatus(out io.Writer, printer func(out io.Writer, v any) error, status *sbmapi.StatusResponse) error {
	ciConfig, err := sbmapi.ToCIConfig(status.CiConfig)
	if err != nil {
		return fmt.Errorf("couldn't unmarshal ci-config from sandbox manager status, %v", err)
	}

	type rawStatus struct {
		CiConfig    *config.ConnectInvocationConfig `json:"ciConfig,omitempty"`
		Localnet    *commonapi.LocalNetStatus       `json:"localnet,omitempty"`
		Hosts       *commonapi.HostsStatus          `json:"hosts,omitempty"`
		Portforward *commonapi.PortForwardStatus    `json:"portforward,omitempty"`
		Sandboxes   []*commonapi.SandboxStatus      `json:"sandboxes,omitempty"`
	}

	rawSt := rawStatus{
		CiConfig:    ciConfig,
		Localnet:    status.Localnet,
		Hosts:       status.Hosts,
		Portforward: status.Portforward,
		Sandboxes:   status.Sandboxes,
	}
	return printer(out, rawSt)
}

func printLocalStatus(cfg *config.LocalStatus, out io.Writer, status *sbmapi.StatusResponse) error {
	ciConfig, err := sbmapi.ToCIConfig(status.CiConfig)
	if err != nil {
		return fmt.Errorf("couldn't unmarshal ci-config from sandbox manager status, %v", err)
	}

	errorLines := []string{}
	healthyLines := []string{}

	// check port forward status
	if ciConfig.ConnectionConfig.Type == connectcfg.PortForwardLinkType {
		msg, ok := checkPortforwardStatus(status.Portforward)
		if ok {
			healthyLines = append(healthyLines, msg)
		} else {
			errorLines = append(errorLines, msg)
		}
	}

	// check root manager (if running)
	if ciConfig.WithRootManager {
		// check localnet service
		msg, ok := checkLocalNetStatus(status.Localnet)
		if ok {
			healthyLines = append(healthyLines, msg)
		} else {
			errorLines = append(errorLines, msg)
		}

		// check hosts service
		msg, ok = checkHostsStatus(status.Hosts)
		if ok {
			healthyLines = append(healthyLines, msg)
		} else {
			errorLines = append(errorLines, msg)
		}
	}

	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	white := color.New(color.FgHiWhite, color.Bold).SprintFunc()

	if len(errorLines) == 0 {
		printLine(out, 0, fmt.Sprintf("connection healthy! %s", green("✓")), "*")
		for _, line := range healthyLines {
			printLine(out, 0, line, "*")
		}
		printLine(out, 0, "Local Sandboxes:", "*")
		if len(status.Sandboxes) == 0 {
			printLine(out, 1, "No active sandbox", "-")
		} else {
			for _, sandbox := range status.Sandboxes {
				printLine(out, 1, white(sandbox.Name), "-")
				printLine(out, 2, fmt.Sprintf("Routing Key: %s", sandbox.RoutingKey), "*")
				printLine(out, 2, "Local Workloads:", "*")
				for _, localwl := range sandbox.LocalWorkloads {
					printLine(out, 3, white(localwl.Name), "-")
					printLine(out, 4, fmt.Sprintf("%s/%s in namespace %q",
						localwl.Baseline.Kind, localwl.Baseline.Name, localwl.Baseline.Namespace), "*")
					for _, portMap := range localwl.WorkloadPortMapping {
						printLine(out, 5, fmt.Sprintf("port %d -> %s",
							portMap.BaselinePort, portMap.LocalAddress), "*")
					}
					if localwl.TunnelHealth.Healthy {
						printLine(out, 4, fmt.Sprintf("workload connected! %s", green("✓")), "*")
					} else {
						printLine(out, 4, fmt.Sprintf("workload not yet connected! %s", red("✗")), "*")
					}
				}
			}
		}
	} else {
		printLine(out, 0, fmt.Sprintf("connection not healthy! %s", red("✗")), "*")
		for _, line := range errorLines {
			printLine(out, 0, line, "*")
		}
	}
	return nil
}

func checkPortforwardStatus(portforward *commonapi.PortForwardStatus) (string, bool) {
	errorMsg := "failed to establish port-forward"
	if portforward != nil {
		if portforward.Health != nil {
			if portforward.Health.Healthy {
				return fmt.Sprintf("port-forward listening at %q", portforward.LocalAddress), true
			}
			if portforward.Health.LastErrorReason != "" {
				errorMsg += fmt.Sprintf(" (%q)", portforward.Health.LastErrorReason)
			}
		}
	}
	return errorMsg, false
}

func checkLocalNetStatus(localnet *commonapi.LocalNetStatus) (string, bool) {
	errorMsg := "failed to setup localnet"
	if localnet != nil {
		if localnet.Health != nil {
			if localnet.Health.Healthy {
				return "localnet has been configured", true
			}
			if localnet.Health.LastErrorReason != "" {
				errorMsg += fmt.Sprintf(" (%q)", localnet.Health.LastErrorReason)
			}
		}
	}
	return errorMsg, false
}

func checkHostsStatus(hosts *commonapi.HostsStatus) (string, bool) {
	errorMsg := "failed to configure hosts in /etc/hosts"
	if hosts != nil {
		if hosts.Health != nil {
			if hosts.Health.Healthy {
				return fmt.Sprintf("%d hosts accessible via /etc/hosts", hosts.NumHosts), true
			}
			if hosts.Health.LastErrorReason != "" {
				errorMsg += fmt.Sprintf(" (%q)", hosts.Health.LastErrorReason)
			}
		}
	}
	return errorMsg, false
}

func printLine(out io.Writer, idents int, line, prefix string) {
	for i := 0; i < idents; i++ {
		fmt.Fprintf(out, "    ")
	}
	if prefix != "" {
		fmt.Fprintf(out, "%s %s\n", prefix, line)
	} else {
		fmt.Fprintf(out, "%s\n", line)
	}
}
