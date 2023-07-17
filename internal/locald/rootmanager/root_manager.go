package rootmanager

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/hashicorp/go-multierror"
	"github.com/signadot/cli/internal/config"
	rootapi "github.com/signadot/cli/internal/locald/api/rootmanager"
	connectcfg "github.com/signadot/libconnect/config"
	"github.com/signadot/libconnect/fwdtun/etchosts"
	"github.com/signadot/libconnect/fwdtun/localnet"
	"golang.org/x/exp/slog"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"k8s.io/client-go/tools/clientcmd"
)

type rootManager struct {
	log        *slog.Logger
	conf       *config.ConnectInvocationConfig
	grpcServer *grpc.Server
	root       *rootServer
	sbmMonitor *sbmgrMonitor
	pfwMonitor *pfwMonitor
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
		conf:       cfg.ConnectInvocationConfig,
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

	// Run the sandbox manager
	go m.sbmMonitor.run()

	if m.conf.ConnectionConfig.Type == connectcfg.ProxyAddressLinkType {
		// Start localnet and etchost services
		m.runLocalnetService(ctx, m.conf.ConnectionConfig.ProxyAddress)
		m.runEtcHostsService(ctx, m.conf.ConnectionConfig.ProxyAddress)
	} else {
		// Start the port-forward monitor, who will be in charge of
		// starting/restarting the localnet and etchost services
		m.pfwMonitor = NewPortForwardMonitor(ctx, m)
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
	if m.pfwMonitor != nil {
		m.pfwMonitor.Stop()
	}
	me = multierror.Append(me, m.sbmMonitor.stop())
	me = multierror.Append(me, m.stopLocalnetService())
	me = multierror.Append(me, m.stopEtcHostsService())
	return me.ErrorOrNil()
}

func (m *rootManager) runAPIServer(ctx context.Context) error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", m.conf.LocalNetPort))
	if err != nil {
		return err
	}
	go m.grpcServer.Serve(ln)
	return nil
}

func (m *rootManager) runLocalnetService(ctx context.Context, socks5Addr string) {
	// Get the user from kubeconfig
	user := m.getK8SUser()
	m.log.Debug("current k8s user is", "user", user)

	// Start the localnet service
	localnetSVC := localnet.NewService(ctx, &localnet.ClientConfig{
		Log:        m.log,
		User:       user,
		SOCKS5Addr: socks5Addr,
		ListenAddr: "127.0.0.1:2223",
	}, m.conf.ConnectionConfig)

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

func (m *rootManager) runEtcHostsService(ctx context.Context, socks5Addr string) {
	// Start the etc hosts service
	etcHostsSVC := etchosts.NewEtcHosts(socks5Addr, m.getHostsFile(), m.log)

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

func (m *rootManager) getK8SUser() string {
	kubeConfig, err := clientcmd.LoadFromFile(m.conf.ConnectionConfig.GetKubeConfigPath())
	if err != nil {
		m.log.Error("couldn't load kubeconfig", "error", err)
		return ""
	}
	if k8sCtx, ok := kubeConfig.Contexts[m.conf.ConnectionConfig.KubeContext]; ok {
		return k8sCtx.AuthInfo
	}
	return ""
}

func (m *rootManager) getHostsFile() string {
	hostsFile := "/etc/hosts"
	if runtime.GOOS == "windows" {
		hostsFile = `C:\Windows\System32\Drivers\etc\hosts`
	}
	return hostsFile
}
