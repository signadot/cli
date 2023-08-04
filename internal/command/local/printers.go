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

func printRawStatus(cfg *config.LocalStatus, out io.Writer, printer func(out io.Writer, v any) error,
	status *sbmapi.StatusResponse) error {
	// unmarshal the ci config
	ciConfig, err := sbmapi.ToCIConfig(status.CiConfig)
	if err != nil {
		return fmt.Errorf("couldn't unmarshal ci-config from sandbox manager status, %v", err)
	}

	// convert the status into a map (useful to convert snake-case fields to camel-case,
	// format dates, etc)
	statusMap, err := sbmapi.StatusToMap(status)
	if err != nil {
		return err
	}

	type rawStatus struct {
		RuntimeConfig any `json:"runtimeConfig,omitempty"`
		Localnet      any `json:"localnet,omitempty"`
		Hosts         any `json:"hosts,omitempty"`
		Portforward   any `json:"portforward,omitempty"`
		Sandboxes     any `json:"sandboxes,omitempty"`
	}

	rawSt := rawStatus{
		RuntimeConfig: getRawRuntimeConfig(cfg, ciConfig),
		Localnet:      getRawLocalnet(cfg, ciConfig, status.Localnet, statusMap),
		Hosts:         getRawHosts(cfg, ciConfig, status.Hosts, statusMap),
		Portforward:   getRawPortforward(cfg, ciConfig, status.Portforward, statusMap),
		Sandboxes:     statusMap["sandboxes"],
	}

	return printer(out, rawSt)
}

func getRawRuntimeConfig(cfg *config.LocalStatus, ciConfig *config.ConnectInvocationConfig) any {
	var runtimeConfig any

	if cfg.Details {
		// Details view
		type PrintableUser struct {
			UID      int    `json:"uid"`
			GID      int    `json:"gid"`
			Username string `json:"username"`
			UIDHome  string `json:"uidHome"`
		}

		type PrintableAPI struct {
			ConfigFile   string `json:"configFile"`
			Org          string `json:"org"`
			MaskedAPIKey string `json:"maskedAPIKey"`
			APIURL       string `json:"apiURL"`
		}

		type PrintableRuntimeConfig struct {
			RootDaemon       bool                         `json:"rootDaemon"`
			APIPort          uint16                       `json:"apiPort"`
			LocalNetPort     uint16                       `json:"localNetPort"`
			ConfigDir        string                       `json:"configDir"`
			User             *PrintableUser               `json:"user"`
			ConnectionConfig *connectcfg.ConnectionConfig `json:"connectionConfig"`
			API              *PrintableAPI                `json:"api"`
			Debug            bool                         `json:"debug"`
		}

		runtimeConfig = &PrintableRuntimeConfig{
			RootDaemon:   ciConfig.WithRootManager,
			APIPort:      ciConfig.APIPort,
			LocalNetPort: ciConfig.LocalNetPort,
			ConfigDir:    ciConfig.SignadotDir,
			User: &PrintableUser{
				UID:      ciConfig.User.UID,
				GID:      ciConfig.User.GID,
				Username: ciConfig.User.Username,
				UIDHome:  ciConfig.User.UIDHome,
			},
			ConnectionConfig: ciConfig.ConnectionConfig,
			API: &PrintableAPI{
				ConfigFile:   ciConfig.API.ConfigFile,
				Org:          ciConfig.API.Org,
				MaskedAPIKey: ciConfig.API.MaskedAPIKey,
				APIURL:       ciConfig.API.APIURL,
			},
			Debug: ciConfig.Debug,
		}
	} else {
		// Standard view
		type PrintableRuntimeConfig struct {
			RootDaemon       bool                         `json:"rootDaemon"`
			ConfigDir        string                       `json:"configDir"`
			ConnectionConfig *connectcfg.ConnectionConfig `json:"connectionConfig"`
		}

		runtimeConfig = &PrintableRuntimeConfig{
			RootDaemon:       ciConfig.WithRootManager,
			ConfigDir:        ciConfig.SignadotDir,
			ConnectionConfig: ciConfig.ConnectionConfig,
		}
	}

	return runtimeConfig
}

func getRawLocalnet(cfg *config.LocalStatus, ciConfig *config.ConnectInvocationConfig,
	localnet *commonapi.LocalNetStatus, statusMap map[string]any) any {
	var result any

	if !ciConfig.WithRootManager {
		return localnet
	}

	if cfg.Details {
		// Details view
		result = statusMap["localnet"]
	} else {
		// Standard view
		type PrintableLocalnet struct {
			Healthy         bool   `json:"healthy"`
			LastErrorReason string `json:"lastErrorReason,omitempty"`
		}

		result = &PrintableLocalnet{
			Healthy: false,
		}

		if localnet != nil {
			if localnet.Health != nil {
				if localnet.Health.Healthy {
					result = &PrintableLocalnet{
						Healthy: true,
					}
				} else {
					result = &PrintableLocalnet{
						Healthy:         false,
						LastErrorReason: localnet.Health.LastErrorReason,
					}
				}
			}
		}
	}
	return result
}

func getRawHosts(cfg *config.LocalStatus, ciConfig *config.ConnectInvocationConfig,
	hosts *commonapi.HostsStatus, statusMap map[string]any) any {
	var result any

	if !ciConfig.WithRootManager {
		return hosts
	}

	if cfg.Details {
		// Details view
		result = statusMap["hosts"]
	} else {
		// Standard view
		type PrintableHosts struct {
			Healthy         bool   `json:"healthy"`
			NumHosts        uint32 `json:"numHosts"`
			LastErrorReason string `json:"lastErrorReason,omitempty"`
		}

		result = &PrintableHosts{
			Healthy: false,
		}

		if hosts != nil {
			if hosts.Health != nil {
				if hosts.Health.Healthy {
					result = &PrintableHosts{
						Healthy:  true,
						NumHosts: hosts.NumHosts,
					}
				} else {
					result = &PrintableHosts{
						Healthy:         false,
						LastErrorReason: hosts.Health.LastErrorReason,
					}
				}
			}
		}
	}
	return result
}

func getRawPortforward(cfg *config.LocalStatus, ciConfig *config.ConnectInvocationConfig,
	portforward *commonapi.PortForwardStatus, statusMap map[string]any) any {
	var result any

	if ciConfig.ConnectionConfig.Type != connectcfg.PortForwardLinkType {
		return portforward
	}

	if cfg.Details {
		// Details view
		result = statusMap["portforward"]
	} else {
		// Standard view
		type PrintablePortforward struct {
			Healthy         bool   `json:"healthy"`
			LocalAddress    string `json:"localAddress"`
			LastErrorReason string `json:"lastErrorReason,omitempty"`
		}

		result = &PrintablePortforward{
			Healthy: false,
		}

		if portforward != nil {
			if portforward.Health != nil {
				if portforward.Health.Healthy {
					result = &PrintablePortforward{
						Healthy:      true,
						LocalAddress: portforward.LocalAddress,
					}
				} else {
					result = &PrintablePortforward{
						Healthy:         false,
						LastErrorReason: portforward.Health.LastErrorReason,
					}
				}
			}
		}
	}
	return result
}

func checkLocalStatusConnectErrors(ciConfig *config.ConnectInvocationConfig, status *sbmapi.StatusResponse) []error {
	var errs []error
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

func printLocalStatus(cfg *config.LocalStatus, out io.Writer, status *sbmapi.StatusResponse) error {
	ciConfig, err := sbmapi.ToCIConfig(status.CiConfig)
	if err != nil {
		return fmt.Errorf("couldn't unmarshal ci-config from sandbox manager status, %v", err)
	}
	connectErrs := checkLocalStatusConnectErrors(ciConfig, status)

	// create a printer
	printer := statusPrinter{
		cfg:      cfg,
		status:   status,
		ciConfig: ciConfig,
		out:      out,
		green:    color.New(color.FgGreen).SprintFunc(),
		red:      color.New(color.FgRed).SprintFunc(),
		white:    color.New(color.FgHiWhite, color.Bold).SprintFunc(),
	}
	// runtime config
	printer.printRuntimeConfig()
	// print status
	if len(connectErrs) == 0 {
		printer.printSuccess()
	} else {
		printer.printErrors(connectErrs)
	}
	return nil
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

type statusPrinter struct {
	cfg      *config.LocalStatus
	status   *sbmapi.StatusResponse
	ciConfig *config.ConnectInvocationConfig
	out      io.Writer
	green    func(a ...any) string
	red      func(a ...any) string
	white    func(a ...any) string
}

func (p *statusPrinter) printRuntimeConfig() {
	var runtimeConfig string
	if p.ciConfig.WithRootManager {
		runtimeConfig = fmt.Sprintf("runtime config: cluster %s, running with root-daemon",
			p.white(p.ciConfig.ConnectionConfig.Cluster))
	} else {
		runtimeConfig = fmt.Sprintf("runtime config: cluster %s, running without root-daemon",
			p.white(p.ciConfig.ConnectionConfig.Cluster))
	}
	if p.cfg.Details {
		runtimeConfig += fmt.Sprintf(" (config-dir: %s)", p.ciConfig.SignadotDir)
	}
	p.printLine(p.out, 0, runtimeConfig, "*")
}

func (p *statusPrinter) printErrors(errs []error) {
	p.printLine(p.out, 0, fmt.Sprintf("Local connection not healthy!"), p.red("✗"))
	for _, err := range errs {
		p.printLine(p.out, 0, err.Error(), "*")
	}
}

func (p *statusPrinter) printSuccess() {
	p.printLine(p.out, 0, fmt.Sprintf("Local connection healthy!"), p.green("✓"))
	if p.ciConfig.ConnectionConfig.Type == connectcfg.PortForwardLinkType {
		p.printPortforwardStatus()
	}
	if p.ciConfig.WithRootManager {
		p.printLocalnetStatus()
		p.printHostsStatus()
	}
	p.printSandboxStatus()
}

func (p *statusPrinter) printPortforwardStatus() {
	p.printLine(p.out, 1, fmt.Sprintf("port-forward listening at %q", p.status.Portforward.LocalAddress), "*")
}

func (p *statusPrinter) printLocalnetStatus() {
	p.printLine(p.out, 1, "localnet has been configured", "*")
	if p.cfg.Details {
		if len(p.status.Localnet.Cidrs) > 0 {
			p.printLine(p.out, 2, "CIDRs:", "*")
			for _, cidr := range p.status.Localnet.Cidrs {
				p.printLine(p.out, 3, cidr, "-")
			}
		}
		if len(p.status.Localnet.ExcludedCidrs) > 0 {
			p.printLine(p.out, 2, "Excluded CIDRs:", "*")
			for _, cidr := range p.status.Localnet.ExcludedCidrs {
				p.printLine(p.out, 3, cidr, "-")
			}
		}
	}
}

func (p *statusPrinter) printHostsStatus() {
	p.printLine(p.out, 1, fmt.Sprintf("%d hosts accessible via /etc/hosts", p.status.Hosts.NumHosts), "*")
}

func (p *statusPrinter) printSandboxStatus() {
	p.printLine(p.out, 0, "Connected Sandboxes:", "*")
	if len(p.status.Sandboxes) == 0 {
		p.printLine(p.out, 1, "No active sandbox", "-")
	} else {
		for _, sandbox := range p.status.Sandboxes {
			p.printLine(p.out, 1, p.white(sandbox.Name), "-")
			p.printLine(p.out, 2, fmt.Sprintf("Routing Key: %s", sandbox.RoutingKey), "*")
			for _, localwl := range sandbox.LocalWorkloads {
				p.printLine(p.out, 2,
					fmt.Sprintf("%s: routing from %s/%s in namespace %q",
						p.white(localwl.Name),
						localwl.Baseline.Kind,
						localwl.Baseline.Name,
						localwl.Baseline.Namespace), "-")
				for _, portMap := range localwl.WorkloadPortMapping {
					p.printLine(p.out, 3, fmt.Sprintf("remote port %d -> %s",
						portMap.BaselinePort, portMap.LocalAddress), "-")
				}
				if localwl.TunnelHealth.Healthy {
					p.printLine(p.out, 2, fmt.Sprintf("connection ready"), p.green("✓"))
				} else {
					p.printLine(p.out, 2, fmt.Sprintf("connection not ready"), p.red("✗"))
				}
			}
		}
	}
}

func (p *statusPrinter) printLine(out io.Writer, idents int, line, prefix string) {
	for i := 0; i < idents; i++ {
		fmt.Fprintf(out, "    ")
	}
	if prefix != "" {
		fmt.Fprintf(out, "%s %s\n", prefix, line)
	} else {
		fmt.Fprintf(out, "%s\n", line)
	}
}
