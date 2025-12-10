package local

import (
	"fmt"
	"io"

	"github.com/fatih/color"
	"github.com/signadot/cli/internal/config"
	commonapi "github.com/signadot/cli/internal/locald/api"
	sbmapi "github.com/signadot/cli/internal/locald/api/sandboxmanager"
	sbmgr "github.com/signadot/cli/internal/locald/sandboxmanager"
	"github.com/signadot/cli/internal/utils/system"
	connectcfg "github.com/signadot/libconnect/config"
)

func printRawStatus(cfg *config.LocalStatus, out io.Writer, printer func(out io.Writer, v any) error,
	status *sbmapi.StatusResponse) error {
	// unmarshal the ci config
	ciConfig, err := sbmapi.ToCIConfig(status.CiConfig)
	if err != nil {
		return fmt.Errorf("couldn't unmarshal ci-config from sandboxmanager status, %v", err)
	}

	// convert the status into a map (useful to convert snake-case fields to camel-case,
	// format dates, etc)
	statusMap, err := sbmapi.StatusToMap(status)
	if err != nil {
		return err
	}

	type rawStatus struct {
		RuntimeConfig     any `json:"runtimeConfig,omitempty"`
		OperatorInfo      any `json:"operatorInfo,omitempty"`
		Localnet          any `json:"localnet,omitempty"`
		Hosts             any `json:"hosts,omitempty"`
		Portforward       any `json:"portforward,omitempty"`
		ControlPlaneProxy any `json:"controlPlaneProxy,omitempty"`
		SandboxesWatcher  any `json:"sandboxesWatcher,omitempty"`
		Sandboxes         any `json:"sandboxes,omitempty"`
		DevboxSession     any `json:"devboxSession,omitempty"`
	}

	rawSt := rawStatus{
		RuntimeConfig:     getRawRuntimeConfig(cfg, ciConfig),
		OperatorInfo:      getRawOperatorInfo(cfg, status.OperatorInfo),
		Localnet:          getRawLocalnet(cfg, ciConfig, status.Localnet, statusMap),
		Hosts:             getRawHosts(cfg, ciConfig, status.Hosts, statusMap),
		Portforward:       getRawPortforward(cfg, ciConfig, status.Portforward, statusMap),
		ControlPlaneProxy: getRawControlPlaneProxy(cfg, ciConfig, status.ControlPlaneProxy, statusMap),
		SandboxesWatcher:  getRawWatcher(cfg, status.Watcher, statusMap),
		Sandboxes:         statusMap["sandboxes"],
		DevboxSession:     getRawDevboxSession(cfg, status.DevboxSession, statusMap),
	}

	return printer(out, rawSt)
}

func getRawRuntimeConfig(cfg *config.LocalStatus, ciConfig *config.ConnectInvocationConfig) any {
	machineID, _ := system.GetMachineID()
	var runtimeConfig any

	if cfg.Details {
		// Details view
		type PrintableUser struct {
			UID      int    `json:"uid"`
			GID      int    `json:"gid"`
			Username string `json:"username"`
			UIDHome  string `json:"uidHome"`
		}

		type PrintableRuntimeConfig struct {
			RootDaemon       bool                         `json:"rootDaemon"`
			APIPort          uint16                       `json:"apiPort"`
			LocalNetPort     uint16                       `json:"localNetPort"`
			ConfigDir        string                       `json:"configDir"`
			User             *PrintableUser               `json:"user"`
			MachineID        string                       `json:"machineID"`
			ConnectionConfig *connectcfg.ConnectionConfig `json:"connectionConfig"`
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
			MachineID:        machineID,
			ConnectionConfig: ciConfig.ConnectionConfig,
			Debug:            ciConfig.Debug,
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

func getRawOperatorInfo(cfg *config.LocalStatus, info *commonapi.OperatorInfo) any {
	var operatorInfo any
	if info == nil {
		return operatorInfo
	}

	if cfg.Details {
		// Details view
		type PrintableOperatorInfo struct {
			Version   string `json:"version"`
			GitCommit string `json:"gitCommit"`
			BuildDate string `json:"buildDate"`
		}

		operatorInfo = &PrintableOperatorInfo{
			Version:   info.Version,
			GitCommit: info.GitCommit,
			BuildDate: info.BuildDate,
		}
	} else {
		// Standard view
		type PrintableOperatorInfo struct {
			Version string `json:"version"`
		}

		operatorInfo = &PrintableOperatorInfo{
			Version: info.Version,
		}
	}

	return operatorInfo
}

func getRawLocalnet(cfg *config.LocalStatus, ciConfig *config.ConnectInvocationConfig,
	localnet *commonapi.LocalNetStatus, statusMap map[string]any) any {
	if !ciConfig.WithRootManager {
		return localnet
	}

	if cfg.Details {
		// Details view
		return statusMap["localnet"]
	}

	// Standard view
	type PrintableLocalnet struct {
		Healthy         bool   `json:"healthy"`
		LastErrorReason string `json:"lastErrorReason,omitempty"`
	}

	result := &PrintableLocalnet{
		Healthy: false,
	}
	if localnet == nil || localnet.Health == nil {
		return result
	}
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
	return result
}

func getRawHosts(cfg *config.LocalStatus, ciConfig *config.ConnectInvocationConfig,
	hosts *commonapi.HostsStatus, statusMap map[string]any) any {
	if !ciConfig.WithRootManager {
		return hosts
	}

	if cfg.Details {
		// Details view
		return statusMap["hosts"]
	}

	// Standard view
	type PrintableHosts struct {
		Healthy         bool   `json:"healthy"`
		NumHosts        uint32 `json:"numHosts"`
		LastErrorReason string `json:"lastErrorReason,omitempty"`
	}

	result := &PrintableHosts{
		Healthy: false,
	}
	if hosts == nil || hosts.Health == nil {
		return result
	}
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
	return result
}

func getRawPortforward(cfg *config.LocalStatus, ciConfig *config.ConnectInvocationConfig,
	portforward *commonapi.PortForwardStatus, statusMap map[string]any) any {
	if ciConfig.ConnectionConfig.Type != connectcfg.PortForwardLinkType {
		return portforward
	}

	if cfg.Details {
		// Details view
		return statusMap["portforward"]
	}

	// Standard view
	type PrintablePortforward struct {
		Healthy         bool   `json:"healthy"`
		LocalAddress    string `json:"localAddress"`
		LastErrorReason string `json:"lastErrorReason,omitempty"`
	}

	result := &PrintablePortforward{
		Healthy: false,
	}
	if portforward == nil || portforward.Health == nil {
		return result
	}
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
	return result
}

func getRawControlPlaneProxy(cfg *config.LocalStatus, ciConfig *config.ConnectInvocationConfig,
	ctlPlaneProxy *commonapi.ControlPlaneProxyStatus, statusMap map[string]any) any {
	if ciConfig.ConnectionConfig.Type != connectcfg.ControlPlaneProxyLinkType {
		return ctlPlaneProxy
	}

	if cfg.Details {
		// Details view
		return statusMap["controlPlaneProxy"]
	}

	// Standard view
	type PrintableControlPlaneProxy struct {
		Healthy         bool   `json:"healthy"`
		LocalAddress    string `json:"localAddress"`
		LastErrorReason string `json:"lastErrorReason,omitempty"`
	}

	result := &PrintableControlPlaneProxy{
		Healthy: false,
	}
	if ctlPlaneProxy == nil || ctlPlaneProxy.Health == nil {
		return result
	}
	if ctlPlaneProxy.Health.Healthy {
		result = &PrintableControlPlaneProxy{
			Healthy:      true,
			LocalAddress: ctlPlaneProxy.LocalAddress,
		}
	} else {
		result = &PrintableControlPlaneProxy{
			Healthy:         false,
			LastErrorReason: ctlPlaneProxy.Health.LastErrorReason,
		}
	}
	return result
}

func getRawWatcher(cfg *config.LocalStatus, watcher *commonapi.WatcherStatus,
	statusMap map[string]any) any {
	if cfg.Details {
		// Details view
		return statusMap["watcher"]
	}

	// Standard view
	type PrintableWatcher struct {
		Healthy         bool   `json:"healthy"`
		LastErrorReason string `json:"lastErrorReason,omitempty"`
	}

	result := &PrintableWatcher{
		Healthy: false,
	}
	if watcher == nil || watcher.Health == nil {
		return result
	}
	if watcher.Health.Healthy {
		result = &PrintableWatcher{
			Healthy: true,
		}
	} else {
		result = &PrintableWatcher{
			Healthy:         false,
			LastErrorReason: watcher.Health.LastErrorReason,
		}
	}
	return result
}

func getRawDevboxSession(cfg *config.LocalStatus, devboxSession *commonapi.DevboxSessionStatus,
	statusMap map[string]any) any {
	if devboxSession == nil {
		return nil
	}

	if cfg.Details {
		// Details view - return full status from map
		return statusMap["devboxSession"]
	}

	// Standard view
	type PrintableDevboxSession struct {
		Healthy         bool   `json:"healthy"`
		SessionReleased bool   `json:"sessionReleased"`
		DevboxId        string `json:"devboxId,omitempty"`
		SessionId       string `json:"sessionId,omitempty"`
		LastErrorReason string `json:"lastErrorReason,omitempty"`
	}

	result := &PrintableDevboxSession{
		Healthy:         devboxSession.Healthy,
		SessionReleased: devboxSession.SessionReleased,
		DevboxId:        devboxSession.DevboxId,
		SessionId:       devboxSession.SessionId,
	}

	if devboxSession.LastErrorReason != "" {
		result.LastErrorReason = devboxSession.LastErrorReason
	}

	return result
}

func printLocalStatus(cfg *config.LocalStatus, out io.Writer, status *sbmapi.StatusResponse) error {
	ciConfig, err := sbmapi.ToCIConfig(status.CiConfig)
	if err != nil {
		return fmt.Errorf("couldn't unmarshal ci-config from sandbox manager status, %v", err)
	}
	connectErrs := sbmgr.CheckStatusConnectErrors(status, ciConfig)

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
	// Check devbox session status
	if status.DevboxSession != nil && status.DevboxSession.SessionReleased {
		printer.printErrors(append(connectErrs, fmt.Errorf("devbox session no longer available (released by another process)")))
		return nil
	}

	// print status
	if len(connectErrs) == 0 {
		printer.printSuccess()
	} else {
		printer.printErrors(connectErrs)
	}
	return nil
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
	machineID, _ := system.GetMachineID()

	var runtimeConfig string
	if p.ciConfig.WithRootManager {
		runtimeConfig = fmt.Sprintf("runtime config: cluster %s, running with root-daemon",
			p.white(p.ciConfig.ConnectionConfig.Cluster))
	} else {
		runtimeConfig = fmt.Sprintf("runtime config: cluster %s, running without root-daemon",
			p.white(p.ciConfig.ConnectionConfig.Cluster))
	}
	if p.cfg.Details {
		runtimeConfig += fmt.Sprintf(" (config-dir: %s, machine-id: %s)", p.ciConfig.SignadotDir, machineID)
	}
	p.printLine(p.out, 0, runtimeConfig, "*")
}

func (p *statusPrinter) printErrors(errs []error) {
	p.printLine(p.out, 0, "Local connection not healthy!", p.red("✗"))
	for _, err := range errs {
		p.printLine(p.out, 0, err.Error(), "*")
	}
}

func (p *statusPrinter) printSuccess() {
	p.printLine(p.out, 0, "Local connection healthy!", p.green("✓"))
	if p.status.OperatorInfo != nil {
		p.printOperatorInfo()
	}
	p.printDevboxSessionStatus()
	switch p.ciConfig.ConnectionConfig.Type {
	case connectcfg.PortForwardLinkType:
		p.printPortforwardStatus()
	case connectcfg.ControlPlaneProxyLinkType:
		p.printControlPlaneProxyStatus()
	}
	if p.ciConfig.WithRootManager {
		p.printLocalnetStatus()
		p.printHostsStatus()
	}
	p.printSandboxesWatcherStatus()
	p.printSandboxStatus()
}

func (p *statusPrinter) printOperatorInfo() {
	msg := fmt.Sprintf("operator version %s", p.status.OperatorInfo.Version)
	if p.cfg.Details {
		msg += fmt.Sprintf(" (git-commit: %s, build-date: %s)",
			p.status.OperatorInfo.GitCommit, p.status.OperatorInfo.BuildDate)
	}
	p.printLine(p.out, 1, msg, "*")
}

func (p *statusPrinter) printDevboxSessionStatus() {
	if p.status.DevboxSession == nil {
		return
	}

	ds := p.status.DevboxSession
	if ds.SessionReleased {
		msg := "devbox session no longer available"
		if ds.LastErrorReason != "" {
			msg += fmt.Sprintf(": %s", ds.LastErrorReason)
		}
		p.printLine(p.out, 1, msg, p.red("✗"))
	} else if ds.Healthy {
		msg := fmt.Sprintf("devbox session active (devbox: %s, session: %s)", ds.DevboxId, ds.SessionId)
		if p.cfg.Details && ds.SessionId != "" {
			msg = "devbox session active"
			p.printLine(p.out, 1, msg, p.green("✓"))
			p.printLine(p.out, 2, fmt.Sprintf("Devbox ID: %s", ds.DevboxId), "*")
			p.printLine(p.out, 2, fmt.Sprintf("Session ID: %s", ds.SessionId), "*")
		} else {
			p.printLine(p.out, 1, msg, p.green("✓"))
		}
	} else {
		msg := "devbox session unhealthy"
		if ds.LastErrorReason != "" {
			msg += fmt.Sprintf(": %s", ds.LastErrorReason)
		}
		p.printLine(p.out, 1, msg, p.red("✗"))
	}
}

func (p *statusPrinter) printPortforwardStatus() {
	p.printLine(p.out, 1, fmt.Sprintf("port-forward listening at %q", p.status.Portforward.LocalAddress), "*")
}

func (p *statusPrinter) printControlPlaneProxyStatus() {
	p.printLine(p.out, 1, fmt.Sprintf("control-plane proxy listening at %q", p.status.ControlPlaneProxy.LocalAddress), "*")
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

func (p *statusPrinter) printSandboxesWatcherStatus() {
	msg := "sandboxes watcher is not running"
	if p.status.Watcher != nil && p.status.Watcher.Health != nil {
		if p.status.Watcher.Health.Healthy {
			msg = "sandboxes watcher is running"
		} else {
			msg += fmt.Sprintf(" (%q)", p.status.Watcher.Health.LastErrorReason)
		}
	}
	p.printLine(p.out, 1, msg, "*")
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
					if localwl.Baseline.Kind == "Forward" {
						remoteAddr := fmt.Sprintf("fwd-%s-%s.signadot.svc:%d",
							sandbox.Name, localwl.Name, portMap.BaselinePort)
						p.printLine(p.out, 3, fmt.Sprintf("%s -> %s", remoteAddr, portMap.LocalAddress), "-")
					} else {
						p.printLine(p.out, 3, fmt.Sprintf("remote port %d -> %s",
							portMap.BaselinePort, portMap.LocalAddress), "-")
					}
				}
				if localwl.TunnelHealth.Healthy {
					p.printLine(p.out, 2, "connection ready", p.green("✓"))
				} else {
					p.printLine(p.out, 2, "connection not ready", p.red("✗"))
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
