package local

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/oklog/run"
	"github.com/signadot/cli/internal/config"
	clusters "github.com/signadot/go-sdk/client/cluster"
	routegroups "github.com/signadot/go-sdk/client/route_groups"
	"github.com/signadot/go-sdk/client/sandboxes"
	"github.com/signadot/libconnect/common/controlplaneproxy"
	"github.com/spf13/cobra"
)

func newProxy(localConfig *config.Local) *cobra.Command {
	cfg := &config.LocalProxy{
		Local: localConfig,
	}

	cmd := &cobra.Command{
		Use:   "proxy [--sandbox SANDBOX|--routegroup ROUTEGROUP|--cluster CLUSTER] --map <target-protocol>://<target-addr>|<bind-addr> [--map <target-protocol>://<target-addr>@<bind-addr>]",
		Short: "Proxy connections based on the specified mappings",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProxy(cmd, cmd.OutOrStdout(), cfg, args)
		},
	}
	cfg.AddFlags(cmd)

	return cmd
}

func runProxy(cmd *cobra.Command, out io.Writer, cfg *config.LocalProxy, args []string) error {
	ctx := context.Background()

	if err := cfg.InitLocalProxyConfig(); err != nil {
		return err
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	// define the cluster and routing key to use
	var cluster, routingKey string

	if cfg.Sandbox != "" {
		// resolve the sandbox
		params := sandboxes.NewGetSandboxParams().
			WithOrgName(cfg.Org).WithSandboxName(cfg.Sandbox)
		resp, err := cfg.Client.Sandboxes.GetSandbox(params, nil)
		if err != nil {
			return err
		}

		cluster = *resp.Payload.Spec.Cluster
		routingKey = resp.Payload.RoutingKey
	} else if cfg.RouteGroup != "" {
		// resolve the routegroup
		params := routegroups.NewGetRoutegroupParams().
			WithOrgName(cfg.Org).WithRoutegroupName(cfg.RouteGroup)
		resp, err := cfg.Client.RouteGroups.GetRoutegroup(params, nil)
		if err != nil {
			return err
		}

		cluster = resp.Payload.Spec.Cluster
		routingKey = resp.Payload.RoutingKey
	} else {
		// validate the cluster
		params := clusters.NewGetClusterParams().
			WithOrgName(cfg.Org).WithClusterName(cfg.Cluster)
		if _, err := cfg.Client.Cluster.GetCluster(params, nil); err != nil {
			return err
		}
		cluster = cfg.Cluster
	}

	logLevel := slog.LevelInfo
	if cfg.Debug {
		logLevel = slog.LevelDebug
	}
	log := slog.New(slog.NewTextHandler(out, &slog.HandlerOptions{
		Level: logLevel,
	}))

	var servers run.Group
	for i := range cfg.ProxyMappings {
		pm := &cfg.ProxyMappings[i]

		ctlPlaneProxy, err := controlplaneproxy.NewProxy(&controlplaneproxy.Config{
			Log:        log,
			ProxyURL:   cfg.ProxyURL,
			TargetURL:  pm.GetTarget(),
			Cluster:    cluster,
			RoutingKey: routingKey,
			BindAddr:   pm.BindAddr,
		}, cfg.GetAPIKey())
		if err != nil {
			return err
		}

		servers.Add(
			func() error { ctlPlaneProxy.Run(ctx); return nil },
			func(error) { ctlPlaneProxy.Close(ctx) },
		)
	}

	switch err := servers.Run().(type) {
	case run.SignalError:
		log.Info(fmt.Sprintf("Received %v signal. Shutdown complete.", err.Signal.String()))
		return nil
	default:
		return err
	}
}
