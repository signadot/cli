package locald

import (
	"context"
	"fmt"
	"os"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/locald/rootmanager"
	sbmgr "github.com/signadot/cli/internal/locald/sandboxmanager"
	"golang.org/x/exp/slog"
)

func RunSandboxManager(cfg *config.LocalDaemon, log *slog.Logger, args []string) error {
	ctx := context.Background()
	sbMgr, err := sbmgr.NewSandboxManager(cfg, args, log.With(
		"locald-component", "sandbox-manager",
		"pid", os.Getpid()))
	if err != nil {
		return err
	}
	return sbMgr.Run(ctx)
}

func RunAsRoot(cfg *config.LocalDaemon, log *slog.Logger, args []string) error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("must run as root without --unprivileged")
	}
	ctx := context.Background()
	rootMgr, err := rootmanager.NewRootManager(cfg, args, log)
	if err != nil {
		return err
	}
	return rootMgr.Run(ctx)
}
