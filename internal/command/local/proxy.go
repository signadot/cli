package local

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	_ "net/http/pprof"

	"github.com/oklog/run"
	"github.com/signadot/cli/internal/auth"
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
		Use:   "proxy [--sandbox SANDBOX|--routegroup ROUTEGROUP|--cluster CLUSTER] --map <target-protocol>://<target-addr>@<bind-addr> [--map <target-protocol>://<target-addr>@<bind-addr>]",
		Short: "Proxy connections based on the specified mappings",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProxy(cmd.OutOrStdout(), cfg)
		},
	}
	cfg.AddFlags(cmd)

	return cmd
}

func runProxy(out io.Writer, cfg *config.LocalProxy) error {
	ctx := context.Background()

	if err := cfg.InitLocalProxyConfig(); err != nil {
		return err
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	// define the cluster and routing key to use
	var cluster, routingKey string

	switch {
	case cfg.Sandbox != "":
		// resolve the sandbox
		params := sandboxes.NewGetSandboxParams().
			WithOrgName(cfg.Org).WithSandboxName(cfg.Sandbox)
		resp, err := cfg.Client.Sandboxes.GetSandbox(params, nil)
		if err != nil {
			return err
		}

		cluster = *resp.Payload.Spec.Cluster
		routingKey = resp.Payload.RoutingKey

	case cfg.RouteGroup != "":
		// resolve the routegroup
		params := routegroups.NewGetRoutegroupParams().
			WithOrgName(cfg.Org).WithRoutegroupName(cfg.RouteGroup)
		resp, err := cfg.Client.RouteGroups.GetRoutegroup(params, nil)
		if err != nil {
			return err
		}

		cluster = resp.Payload.Spec.Cluster
		routingKey = resp.Payload.RoutingKey
		if cluster == "" {
			// this is a multi-cluster RG, the cluster must be explicitly defined
			if cfg.Cluster == "" {
				return errors.New("--cluster must be specified in multi-cluster route groups")
			}
			// validate the cluster
			params := clusters.NewGetClusterParams().
				WithOrgName(cfg.Org).WithClusterName(cfg.Cluster)
			if _, err := cfg.Client.Cluster.GetCluster(params, nil); err != nil {
				return err
			}
			cluster = cfg.Cluster
		}

	default:
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
			Log:              log,
			ProxyURL:         cfg.ProxyURL,
			TargetURL:        pm.GetTarget(),
			Cluster:          cluster,
			RoutingKey:       routingKey,
			BindAddr:         pm.BindAddr,
			GetInjectHeaders: auth.GetHeaders,
		})
		if err != nil {
			return err
		}

		servers.Add(
			func() error { ctlPlaneProxy.Run(ctx); return nil },
			func(error) { ctlPlaneProxy.Close(ctx) },
		)
	}

	if cfg.PProfAddr != "" {
		go http.ListenAndServe(cfg.PProfAddr, nil)
	}

	switch err := servers.Run().(type) {
	case run.SignalError:
		log.Info(fmt.Sprintf("Received %v signal. Shutdown complete.", err.Signal.String()))
		return nil
	default:
		return err
	}
}
