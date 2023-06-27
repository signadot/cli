package locald

import (
	"fmt"
	"os"

	"github.com/signadot/cli/internal/config"
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
	// run unpriveleged
	return nil
}
