package proxy

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/libconnect/proxy/grpcproxy"
	"github.com/signadot/libconnect/proxy/httpconnect"
	"github.com/signadot/libconnect/proxy/httpproxy"
	"github.com/signadot/libconnect/proxy/tcpproxy"
	"google.golang.org/grpc"
)

type proxyConfig struct {
	log        *slog.Logger
	cfg        *config.Proxy
	routingKey string
	cluster    string
	mapping    *config.ProxyMapping
}

type proxyServer struct {
	*proxyConfig

	proxyHeaders http.Header
	ln           net.Listener
	tcpProxy     *tcpproxy.Proxy
	httpServer   *http.Server
	grpcServer   *grpc.Server
}

func NewProxyServer(conf *proxyConfig) (*proxyServer, error) {
	// create a listener
	ln, err := net.Listen("tcp", conf.mapping.BindAddr)
	if err != nil {
		return nil, fmt.Errorf("couldn't listen at %s, %w", conf.mapping.BindAddr, err)
	}

	// setup needed header to send to the proxy
	proxyHeaders := http.Header{}
	proxyHeaders.Add(config.APIKeyHeader, conf.cfg.GetAPIKey())
	proxyHeaders.Add(config.ClusterHeader, conf.cluster)

	return &proxyServer{
		proxyConfig:  conf,
		ln:           ln,
		proxyHeaders: proxyHeaders,
	}, nil
}

func (p *proxyServer) Start() error {
	switch p.mapping.TargetProto {
	case "tcp":
		return p.runTCPServer()
	case "http":
		return p.runHTTPProxy()
	case "https":
		return p.runHTTPProxy()
	case "grpc":
		p.runGRPCProxy()
	}
	return nil
}

func (p *proxyServer) Shutdown(ctx context.Context) {
	if p.ln != nil {
		p.ln.Close()
	}
	if p.tcpProxy != nil {
		p.tcpProxy.Close()
	}
	if p.httpServer != nil {
		p.httpServer.Shutdown(ctx)
	}
	if p.grpcServer != nil {
		p.grpcServer.GracefulStop()
	}
}

func (p *proxyServer) runHTTPProxy() error {
	proxy, err := httpproxy.NewProxy(p.log, p.cfg.ProxyURL, p.proxyHeaders, p.mapping.GetTarget(), p.routingKey)
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", proxy.ProxyRequestHandler())
	p.httpServer = &http.Server{
		Handler: mux,
	}

	p.log.Info("starting http proxy", "targetURL", p.mapping.GetTarget(), "bindAddr", p.mapping.BindAddr)
	return p.httpServer.Serve(p.ln)
}

func (p *proxyServer) runGRPCProxy() error {
	proxy, err := grpcproxy.NewProxy(p.log, p.cfg.ProxyURL, p.proxyHeaders, p.mapping.GetTarget(), p.routingKey)
	if err != nil {
		return err
	}

	p.grpcServer = grpc.NewServer(
		grpc.UnknownServiceHandler(proxy.StreamHandler()),
	)

	p.log.Info("starting grpc proxy", "targetURL", p.mapping.GetTarget(), "bindAddr", p.mapping.BindAddr)
	return p.grpcServer.Serve(p.ln)
}

func (p *proxyServer) runTCPServer() error {
	dialer := httpconnect.NewDialer(p.log, p.cfg.ProxyURL, p.proxyHeaders)

	var err error
	p.tcpProxy, err = tcpproxy.NewProxy(&tcpproxy.Config{
		Log: p.log,
		Dialer: func(resolvData any) (net.Conn, error) {
			conn, err := dialer.DialContext(context.Background(), "tcp", p.mapping.TargetAddr)
			return conn, err
		},
	})
	if err != nil {
		return err
	}

	p.log.Info("starting tcp proxy", "targetURL", p.mapping.GetTarget(), "bindAddr", p.mapping.BindAddr)
	return p.tcpProxy.Serve(p.ln)
}
