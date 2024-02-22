package proxy

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/oklog/run"
	"github.com/signadot/cli/internal/config"
	routegroups "github.com/signadot/go-sdk/client/route_groups"
	"github.com/signadot/go-sdk/client/sandboxes"
	"github.com/spf13/cobra"
)

func newConnect(proxyConfig *config.Proxy) *cobra.Command {
	cfg := &config.ProxyConnect{
		Proxy: proxyConfig,
	}

	cmd := &cobra.Command{
		Use:   "connect [--sandbox SANDBOX|--routegroup ROUTEGROUP|--cluster CLUSTER] --map <target-protocol>://<target-addr>|<bind-addr> [--map <target-protocol>://<target-addr>|<bind-addr>]",
		Short: "Proxy connections to the specified mappings",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConnect(cmd, cmd.OutOrStdout(), cfg, args)
		},
	}
	cfg.AddFlags(cmd)

	return cmd
}

func runConnect(cmd *cobra.Command, out io.Writer, cfg *config.ProxyConnect, args []string) error {
	if err := cfg.InitProxyConfig(); err != nil {
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

		proxyServer, err := NewProxyServer(&proxyConfig{
			log:        log,
			cfg:        cfg.Proxy,
			routingKey: routingKey,
			cluster:    cluster,
			mapping:    pm,
		})
		if err != nil {
			return err
		}
		servers.Add(
			proxyServer.Start,
			func(error) { proxyServer.Shutdown(context.Background()) },
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
