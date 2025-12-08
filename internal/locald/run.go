package locald

import (
	"context"
	"fmt"
	"os"

	"log/slog"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/locald/rootmanager"
	sbmgr "github.com/signadot/cli/internal/locald/sandboxmanager"
)

func RunSandboxManager(cfg *config.LocalDaemon, log *slog.Logger, args []string) error {
	// Initialize viper with the same config file used by the CLI
	// This ensures auth.ResolveAuth() reads from the correct config file
	var configFile string
	if cfg.ConnectInvocationConfig != nil {
		configFile = cfg.ConnectInvocationConfig.ConfigFile
	}
	if err := config.InitViper(configFile); err != nil {
		log.Warn("Failed to initialize viper from ciConfig, auth may resolve incorrectly", "error", err)
	}

	ctx := context.Background()
	sbMgr, err := sbmgr.NewSandboxManager(cfg, args, log.With(
		"locald-component", "sandbox-manager",
		"pid", os.Getpid()))
	if err != nil {
		return fmt.Errorf("locald sandbox-manager error creating sandbox-manager: %w", err)
	}
	return sbMgr.Run(ctx)
}

func RunRootManager(cfg *config.LocalDaemon, log *slog.Logger, args []string) error {
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
