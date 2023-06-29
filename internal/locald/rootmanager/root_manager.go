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
	"strconv"
	"syscall"

	"github.com/signadot/cli/internal/config"
	rootapi "github.com/signadot/cli/internal/locald/api/rootmanager"
	"github.com/signadot/libconnect/common/processes"
	"golang.org/x/exp/slog"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type rootManager struct {
	log        *slog.Logger
	conf       *config.ConnectInvocationConfig
	userInfo   *user.User
	grpcServer *grpc.Server
}

func NewRootManager(cfg *config.LocalDaemon, args []string) (*rootManager, error) {
	// Resolve the user info
	userInfo, err := user.LookupId(fmt.Sprintf("%d", cfg.ConnectInvocationConfig.UID))
	if err != nil {
		return nil, fmt.Errorf("invalid UID=%d, %w", cfg.ConnectInvocationConfig.UID, err)
	}

	grpcServer := grpc.NewServer()
	rootapi.RegisterRootManagerAPIServer(grpcServer, &rootServer{})

	// TODO: define logging
	log := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	return &rootManager{
		log:        log.With("locald-component", "root-manager"),
		conf:       cfg.ConnectInvocationConfig,
		userInfo:   userInfo,
		grpcServer: grpcServer,
	}, nil
}

func (m *rootManager) Run(ctx context.Context) error {
	// Run the API server
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", m.conf.LocalNetPort))
	if err != nil {
		return err
	}
	go m.grpcServer.Serve(ln)

	// Run the sandbox manager
	m.conf.Unpriveleged = true
	ciBytes, err := json.Marshal(m.conf)
	if err != nil {
		// should be impossible
		return err
	}

	proc, err := processes.NewRetryProcess(ctx, &processes.RetryProcessConf{
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
	if err != nil {
		return err
	}

	// Wait until termination
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	return proc.Stop()
}
