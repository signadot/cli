package local

import (
	"context"
	"fmt"
	"time"

	"github.com/signadot/cli/internal/config"
	rmapi "github.com/signadot/cli/internal/locald/api/rootmanager"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func newDisconnect(localConfig *config.Local) *cobra.Command {
	cfg := &config.LocalDisconnect{Local: localConfig}
	_ = cfg

	cmd := &cobra.Command{
		Use:   "disconnect",
		Short: "disconnect local development with sandboxes",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDisconnect(cfg, args)
		},
	}

	return cmd
}

func runDisconnect(cfg *config.LocalDisconnect, args []string) error {
	if err := cfg.InitLocalConfig(); err != nil {
		return err
	}
	// Establish a connection with sandbox manager
	grpcConn, err := grpc.Dial("127.0.0.1:6667", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("couldn't connect sandbox manager api, %v", err)
	}
	defer grpcConn.Close()

	// Send the shutdown order
	rootManagerClient := rmapi.NewRootManagerAPIClient(grpcConn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err = rootManagerClient.Shutdown(ctx, &rmapi.ShutdownRequest{}); err != nil {
		return fmt.Errorf("error requesting shutdown in sandbox manager api: %v", err)
	}
	// TODO: monitor shutdown works
	return nil
}
