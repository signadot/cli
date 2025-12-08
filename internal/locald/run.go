package locald

import (
	"context"
	"fmt"
	"os"

	"log/slog"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/locald/rootmanager"
	sbmgr "github.com/signadot/cli/internal/locald/sandboxmanager"
	"github.com/signadot/cli/internal/utils/system"
	"github.com/spf13/viper"
)

func RunSandboxManager(cfg *config.LocalDaemon, log *slog.Logger, args []string) error {
	// Initialize viper with the same config file used by the CLI
	// This ensures auth.ResolveAuth() reads from the correct config file
	if err := initViperFromCIConfig(cfg.ConnectInvocationConfig); err != nil {
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

// initViperFromCIConfig initializes viper with the config file path from ciConfig
func initViperFromCIConfig(ciConfig *config.ConnectInvocationConfig) error {
	if ciConfig == nil {
		return fmt.Errorf("ciConfig is nil")
	}

	// If ConfigFile is set, use it; otherwise use default location
	if ciConfig.ConfigFile != "" {
		viper.SetConfigFile(ciConfig.ConfigFile)
	} else {
		// Use default config location
		signadotDir, err := system.GetSignadotDir()
		if err != nil {
			return fmt.Errorf("failed to get signadot dir: %w", err)
		}
		viper.AddConfigPath(signadotDir)
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	viper.SetEnvPrefix("signadot")
	viper.AutomaticEnv()

	// Read config file (optional - may not exist)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found is OK - auth can come from keyring/env
	}

	return nil
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
