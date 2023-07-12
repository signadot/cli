package local

import (
	"fmt"
	"io"

	"github.com/fatih/color"
	"github.com/signadot/cli/internal/config"
	sbmapi "github.com/signadot/cli/internal/locald/api/sandboxmanager"
	connectcfg "github.com/signadot/libconnect/config"
)

func printLocalStatus(cfg *config.LocalStatus, out io.Writer, status *sbmapi.StatusResponse) error {
	ciConfig, err := sbmapi.ToCIConfig(status.CiConfig)
	if err != nil {
		return fmt.Errorf("couldn't unmarshal ci-config from sandbox manager status, %v", err)
	}

	errorLines := []string{}
	healthyLines := []string{}
	if ciConfig.ConnectionConfig.Type == connectcfg.PortForwardLinkType {
		if !status.Portforward.Health.Healthy {
			msg := "failed to establish port-forward"
			if status.Portforward.Health.LastErrorReason != "" {
				msg += fmt.Sprintf(" (%q)", status.Portforward.Health.LastErrorReason)
			}
			errorLines = append(errorLines, msg)
		} else {
			healthyLines = append(healthyLines,
				fmt.Sprintf("port-forward listening at %q", status.Portforward.LocalAddress))
		}
	}
	if ciConfig.WithRootManager {
		if !status.Localnet.Health.Healthy {
			msg := "failed to setup localnet"
			if status.Localnet.Health.LastErrorReason != "" {
				msg += fmt.Sprintf(" (%q)", status.Localnet.Health.LastErrorReason)
			}
			errorLines = append(errorLines, msg)
		}
		if !status.Hosts.Health.Healthy {
			msg := "failed to setup hosts in /etc/hosts"
			if status.Hosts.Health.LastErrorReason != "" {
				msg += fmt.Sprintf(" (%q)", status.Hosts.Health.LastErrorReason)
			}
			errorLines = append(errorLines, msg)
		} else {
			healthyLines = append(healthyLines,
				fmt.Sprintf("%d hosts accessible via /etc/hosts", status.Hosts.NumHosts))
		}
	}

	if len(errorLines) == 0 {
		fmt.Fprint(out, "* connection healthy! ")
		color.New(color.FgGreen).Fprintln(out, "✓")
		for _, line := range healthyLines {
			fmt.Fprintf(out, "* %s\n", line)
		}
		fmt.Fprint(out, "* Local Sandboxes:\n")
		if len(status.Sandboxes) == 0 {
			fmt.Fprintf(out, "\t* No active sandbox\n")
		} else {
			// for _, sandbox := range status.Sandboxes {

			// }
		}
	} else {
		fmt.Fprint(out, "* connection not healthy! ")
		color.New(color.FgRed).Fprintln(out, "✗")
		for _, line := range errorLines {
			fmt.Fprintf(out, "\t* %s\n", line)
		}
	}

	// fmt.Printf("errorLines = %v\n", errorLines)
	// fmt.Printf("healthyLines = %v\n", healthyLines)
	return nil
}
