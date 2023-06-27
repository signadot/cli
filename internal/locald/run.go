package locald

import (
	"fmt"
	"os"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/locald/rootmanager"
)

func RunSandboxManager(cfg *config.LocalDaemon, args []string) error {
	sbMgr, err := newSandboxManager(cfg, args)
	if err != nil {
		return err
	}
	return sbMgr.Run()
}

func RunAsRoot(cfg *config.LocalDaemon, args []string) error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("must run as root without --sandbox-manager=true")
	}
	rootMgr, err := rootmanager.NewRootManager(cfg, args)
	if err != nil {
		return err
	}
	return rootMgr.Run()
}
