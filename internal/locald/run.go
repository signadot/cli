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

func RunSandboxManager(cfg *config.LocalDaemon, args []string) error {
	log := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	sbMgr, err := sbmgr.NewSandboxManager(cfg, args, log.With("locald-component", "sandbox-manager"))
	if err != nil {
		return err
	}
	return sbMgr.Run()
}

func RunAsRoot(cfg *config.LocalDaemon, args []string) error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("must run as root without --unpriveleged")
	}
	ctx := context.Background()
	log := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	rootMgr, err := rootmanager.NewRootManager(cfg, args, log)
	if err != nil {
		return err
	}
	return rootMgr.Run(ctx)
}
