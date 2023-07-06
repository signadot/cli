package rootmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/signadot/cli/internal/config"
	sbmapi "github.com/signadot/cli/internal/locald/api/sandboxmanager"
	"github.com/signadot/libconnect/common/processes"
	"golang.org/x/exp/slog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type sbmgrMonitor struct {
	sync.Mutex
	log           *slog.Logger
	ciConfig      *config.ConnectInvocationConfig
	pidFile       string
	done, doneAck chan struct{}
	procDone      <-chan struct{}
	procPID       int
}

func newSBMgrMonitor(ciConfig *config.ConnectInvocationConfig, log *slog.Logger) *sbmgrMonitor {
	res := &sbmgrMonitor{
		ciConfig: ciConfig,
		log:      log,
		pidFile:  ciConfig.GetPIDfile(),
		done:     make(chan struct{}),
		doneAck:  make(chan struct{}),
	}
	return res
}

func (mon *sbmgrMonitor) getRunSandboxCmd(ciConfig *config.ConnectInvocationConfig) *exec.Cmd {
	mon.log.Debug("a")
	ciBytes, err := json.Marshal(ciConfig)
	if err != nil {
		mon.log.Error("ciconfig json", "error", err)
		panic(err)
	}
	mon.log.Debug("ci", "cfg", string(ciBytes))
	cmd := exec.Command(
		"sudo",
		"-n",
		"-u", fmt.Sprintf("#%d", ciConfig.UID),
		"--preserve-env=SIGNADOT_LOCAL_CONNECT_INVOCATION_CONFIG",
		os.Args[0],
		"locald",
		"--daemon",
	)
	mon.log.Debug("c")
	cmd.Env = append(cmd.Env,
		fmt.Sprintf("HOME=%s", ciConfig.UIDHome),
		fmt.Sprintf("PATH=%s", ciConfig.UIDPath),
		fmt.Sprintf("SIGNADOT_LOCAL_CONNECT_INVOCATION_CONFIG=%s", string(ciBytes)),
	)
	mon.log.Debug("d")
	return cmd
}

func (mon *sbmgrMonitor) run() {
	var (
		cmd      *exec.Cmd
		err      error
		ticker   = time.NewTicker(time.Second)
		procDone <-chan struct{}
		procPID  int
	)
	defer ticker.Stop()
	for {
		mon.log.Debug("sbmgr monitor: starting sandbox-manager cmd")
		cmd = mon.getRunSandboxCmd(mon.ciConfig)
		err = cmd.Run()
		if err != nil {
			mon.log.Error("error launching sandboxmanager", "error", err)
			goto tick
		}
		mon.log.Debug("sbmgr monitor: successfully ran sandbox-manager cmd")
		// if we get here, the command launched the sandbox manager
		// and it has an exclusive lock, or the pidfile doesn't exist
		// and the sandbox manager already crashed
		procDone, procPID, err = processes.MonitorPID(mon.pidFile, mon.log.With("subcomponent", "monitor-pid"))

		if err != nil {
			mon.log.Error("error getting monitor for pidfile", "error", err)
			goto tick
		}
		mon.setProcInfo(procDone, procPID)
		select {
		case <-procDone:
		case <-mon.done:
		}

	tick:
		select {
		case <-ticker.C:
		case <-mon.done:
			close(mon.doneAck)
			return
		}
	}
}

func (mon *sbmgrMonitor) setProcInfo(c <-chan struct{}, pid int) {
	mon.Lock()
	defer mon.Unlock()
	mon.procDone = c
	mon.procPID = pid
}

func (mon *sbmgrMonitor) getProcDone() (<-chan struct{}, int) {
	mon.Lock()
	defer mon.Unlock()
	return mon.procDone, mon.procPID
}

func (mon *sbmgrMonitor) stop() error {
	mon.log.Debug("sandbox manager shutdown")
	close(mon.done)
	<-mon.doneAck
	procDone, pid := mon.getProcDone()
	if procDone == nil {
		return nil
	}
	select {
	case <-procDone:
		return nil
	default:
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}

	// Establish a connection with sandbox manager
	grpcConn, err := grpc.Dial("127.0.0.1:6666", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("couldn't connect sandbox manager api, %v", err)
	}
	defer grpcConn.Close()

	// Send the shutdown order
	sbManagerClient := sbmapi.NewSandboxManagerAPIClient(grpcConn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err = sbManagerClient.Shutdown(ctx, &sbmapi.ShutdownRequest{}); err != nil {
		return fmt.Errorf("error requesting shutdown in sandbox manager api: %v", err)
	}

	// Wait until shutdown
	select {
	case <-procDone:
	case <-time.After(5 * time.Second):
		// Kill the process and wait until it's gone
		if err := process.Kill(); err != nil {
			return fmt.Errorf("failed to kill process: %w ", err)
		}
		<-procDone
	}

	return nil
}
