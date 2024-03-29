package rootmanager

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"log/slog"

	"github.com/hashicorp/go-multierror"
	"github.com/signadot/cli/internal/config"
	rootapi "github.com/signadot/cli/internal/locald/api/rootmanager"
	"github.com/signadot/libconnect/apiv1"
	connectcfg "github.com/signadot/libconnect/config"
	"github.com/signadot/libconnect/fwdtun/etchosts"
	"github.com/signadot/libconnect/fwdtun/ipmap"
	"github.com/signadot/libconnect/fwdtun/localnet"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type rootManager struct {
	log        *slog.Logger
	ciConfig   *config.ConnectInvocationConfig
	grpcServer *grpc.Server
	root       *rootServer
	sbmMonitor *sbmgrMonitor
	tpMonitor  *tpMonitor
	shutdownCh chan struct{}
}

func NewRootManager(cfg *config.LocalDaemon, args []string, log *slog.Logger) (*rootManager, error) {
	shutdownCh := make(chan struct{})
	root := &rootServer{
		shutdownCh: shutdownCh,
	}
	grpcServer := grpc.NewServer()
	rootapi.RegisterRootManagerAPIServer(grpcServer, root)

	ciConfig := cfg.ConnectInvocationConfig
	log = log.With("locald-component", "root-manager")

	return &rootManager{
		log:        log,
		ciConfig:   ciConfig,
		grpcServer: grpcServer,
		root:       root,
		sbmMonitor: newSBMgrMonitor(ciConfig, log),
		shutdownCh: shutdownCh,
	}, nil
}

func (m *rootManager) Run(ctx context.Context) error {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Run the API server
	if err := m.runAPIServer(ctx); err != nil {
		return err
	}

	// start the ip mapping
	ipMap, err := ipmap.NewIPMap(m.ciConfig.VirtualIPNet)
	if err != nil {
		return err
	}

	// Run the sandbox manager
	go m.sbmMonitor.run()

	switch m.ciConfig.ConnectionConfig.Type {
	case connectcfg.PortForwardLinkType, connectcfg.ControlPlaneProxyLinkType:
		// Start the port-forward monitor, who will be in charge of
		// starting/restarting the localnet and etchost services
		m.tpMonitor = NewTunnelProxyMonitor(ctx, m, ipMap)
	default:
		// Start localnet and etchost services
		m.runLocalnetService(ctx, m.ciConfig.ConnectionConfig.ProxyAddress, ipMap)
		m.runEtcHostsService(ctx, m.ciConfig.ConnectionConfig.ProxyAddress, ipMap)
	}

	// Wait until termination
	select {
	case <-ctx.Done():
	case <-sigs:
	case <-m.shutdownCh:
	}

	// Clean up
	m.log.Info("Shutting down")
	var me *multierror.Error
	if m.tpMonitor != nil {
		m.tpMonitor.Stop()
	}
	me = multierror.Append(me, m.sbmMonitor.stop())
	me = multierror.Append(me, m.stopLocalnetService())
	me = multierror.Append(me, m.stopEtcHostsService())
	return me.ErrorOrNil()
}

func (m *rootManager) runAPIServer(ctx context.Context) error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", m.ciConfig.LocalNetPort))
	if err != nil {
		return err
	}
	go m.grpcServer.Serve(ln)
	return nil
}

func (m *rootManager) runLocalnetService(ctx context.Context, socks5Addr string, ipMap *ipmap.IPMap) {
	// Start the localnet service
	localnetSVC := localnet.NewService(ctx, ipMap, &localnet.ClientConfig{
		Log:          m.log,
		User:         m.ciConfig.User.Username,
		SOCKS5Addr:   socks5Addr,
		VirtualIPNet: m.ciConfig.VirtualIPNet,
		ListenAddr:   "127.0.0.1:2223",
	}, m.ciConfig.ConnectionConfig)

	// Register the localnet service in root api
	m.root.setLocalnetService(localnetSVC)
}

func (m *rootManager) stopLocalnetService() error {
	localnetSVC := m.root.getLocalnetService()
	if localnetSVC != nil {
		return localnetSVC.Close()
	}
	return nil
}

func (m *rootManager) runEtcHostsService(ctx context.Context, socks5Addr string, ipMap *ipmap.IPMap) {
	// Start the etc hosts service
	etcHostsSVC := etchosts.NewEtcHosts(socks5Addr, m.getHostsFile(), ipMap, m.xCIDRsFilter(), m.log)

	// Register the etc hosts service in root api
	m.root.setEtcHostsService(etcHostsSVC)
}

func (m *rootManager) stopEtcHostsService() error {
	etcHostsSVC := m.root.getEtcHostsService()
	if etcHostsSVC != nil {
		return etcHostsSVC.Close()
	}
	return nil
}

func (m *rootManager) xCIDRsFilter() func(*apiv1.GetDNSEntriesResponse_K8SService) bool {
	xCIDRsConfig := []string{}
	if m.ciConfig.ConnectionConfig.Outbound != nil {
		xCIDRsConfig = m.ciConfig.ConnectionConfig.Outbound.ExcludeCIDRs
	}
	xCIDRs := make([]net.IPNet, 0, len(xCIDRsConfig))
	for _, xc := range xCIDRsConfig {
		_, net, err := net.ParseCIDR(xc)
		if err != nil {
			m.log.Error("couldn't parse excluded CIDR", "cidr", xc)
			continue
		}
		xCIDRs = append(xCIDRs, *net)
	}
	xCIDRsFilter := func(s *apiv1.GetDNSEntriesResponse_K8SService) bool {
		ip := net.ParseIP(s.ServiceIp)
		for _, xNet := range xCIDRs {
			if xNet.Contains(ip) {
				m.log.Info("excluding dns entry for", "service", s.Name, "namespace", s.Namespace, "excluded-cidr", xNet.String())
				return false
			}
		}
		return true
	}
	return xCIDRsFilter
}

func (m *rootManager) getHostsFile() string {
	hostsFile := "/etc/hosts"
	if runtime.GOOS == "windows" {
		hostsFile = `C:\Windows\System32\Drivers\etc\hosts`
	}
	return hostsFile
}
