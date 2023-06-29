package rootmanager

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"

	"github.com/hashicorp/go-multierror"
	"github.com/signadot/cli/internal/config"
	rootapi "github.com/signadot/cli/internal/locald/api/rootmanager"
	"github.com/signadot/libconnect/common/processes"
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
	userInfo   *user.User
	grpcServer *grpc.Server
	root       *rootServer
	sbManager  *processes.RetryProcess
	pfwMonitor *pfwMonitor
}

func NewRootManager(cfg *config.LocalDaemon, args []string, log *slog.Logger) (*rootManager, error) {
	// Resolve the user info
	userInfo, err := user.LookupId(fmt.Sprintf("%d", cfg.ConnectInvocationConfig.UID))
	if err != nil {
		return nil, fmt.Errorf("invalid UID=%d, %w", cfg.ConnectInvocationConfig.UID, err)
	}

	root := &rootServer{}
	grpcServer := grpc.NewServer()
	rootapi.RegisterRootManagerAPIServer(grpcServer, root)

	return &rootManager{
		log:        log,
		conf:       cfg.ConnectInvocationConfig,
		userInfo:   userInfo,
		grpcServer: grpcServer,
		root:       root,
	}, nil
}

func (m *rootManager) Run(ctx context.Context) error {
	// Run the API server
	if err := m.runAPIServer(ctx); err != nil {
		return err
	}

	// Run the sandbox manager
	if err := m.runSandboxManager(ctx); err != nil {
		return err
	}

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
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	// Clean up
	var me *multierror.Error
	m.pfwMonitor.Stop()
	me = multierror.Append(me, m.stopLocalnetService())
	me = multierror.Append(me, m.stopEtcHostsService())
	me = multierror.Append(me, m.sbManager.Stop())
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

func (m *rootManager) runSandboxManager(ctx context.Context) (err error) {
	m.conf.Unpriveleged = true
	ciBytes, err := json.Marshal(m.conf)
	if err != nil {
		// should be impossible
		return err
	}

	m.sbManager, err = processes.NewRetryProcess(ctx, &processes.RetryProcessConf{
		Log: m.log,
		GetCmd: func() *exec.Cmd {
			cmdToRun := exec.Command(
				"sudo", "-n", "-u", fmt.Sprintf("#%d", m.conf.UID),
				"--preserve-env=SIGNADOT_LOCAL_CONNECT_INVOCATION_CONFIG",
				os.Args[0], "locald",
			)
			cmdToRun.Env = append(cmdToRun.Env, fmt.Sprintf("SIGNADOT_LOCAL_CONNECT_INVOCATION_CONFIG=%s", string(ciBytes)))
			cmdToRun.Stderr = os.Stderr
			cmdToRun.Stdout = os.Stdout
			cmdToRun.Stdin = os.Stdin
			// Prevent signaling the children
			cmdToRun.SysProcAttr = &syscall.SysProcAttr{
				Setpgid: true,
			}
			return cmdToRun
		},
		WritePID: func(pidFile string, pid int) error {
			// Write the pid
			if err := processes.WritePIDFile(pidFile, pid); err != nil {
				return err
			}
			// Set right ownership
			gid, _ := strconv.Atoi(m.userInfo.Gid)
			if err := os.Chown(pidFile, m.conf.UID, gid); err != nil {
				m.log.Warn("couldn't change ownership of pidfile", "error", err)
			}
			return nil
		},
		PIDFile: filepath.Join(m.conf.SignadotDir, config.SandboxManagerPIDFile),
	})
	return
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
		ListenAddr: ":8700",
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
