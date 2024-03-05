package config

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"time"

	"github.com/signadot/libconnect/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"sigs.k8s.io/yaml"
)

const (
	DefaultVirtualIPNet = "242.242.0.1/16"
)

type Local struct {
	*API

	ProxyURL string
	// initialized from ~/.signadot/config.yaml
	LocalConfig *config.Config
}

func (l *Local) InitLocalProxyConfig() error {
	var err error
	if err = l.API.InitAPIConfig(); err != nil {
		return err
	}
	if l.ProxyURL, err = l.GetProxyURL(); err != nil {
		return err
	}
	return nil
}

func (l *Local) InitLocalConfig() error {
	if err := l.InitLocalProxyConfig(); err != nil {
		return err
	}

	type Tmp struct {
		Local *config.Config `json:"local"`
	}
	localConfig := &Tmp{}
	d, e := os.ReadFile(viper.ConfigFileUsed())
	if e != nil {
		return e
	}
	if e := yaml.Unmarshal(d, localConfig); e != nil {
		return e
	}
	if localConfig.Local == nil {
		return fmt.Errorf("no local section in %s", viper.ConfigFileUsed())
	}
	if localConfig.Local.VirtualIPNet == "" {
		localConfig.Local.VirtualIPNet = DefaultVirtualIPNet
	}
	if err := localConfig.Local.Validate(); err != nil {
		return err
	}
	if len(localConfig.Local.Connections) == 0 {
		return fmt.Errorf("no connections in local section in %s", viper.ConfigFileUsed())
	}
	if !localConfig.Local.Debug {
		localConfig.Local.Debug = l.Debug
	}
	l.LocalConfig = localConfig.Local
	return nil
}

func (l *Local) GetConnectionConfig(cluster string) (*config.ConnectionConfig, error) {
	conns := l.LocalConfig.Connections
	clusters := make([]string, len(conns))
	for i := range conns {
		clusters[i] = conns[i].Cluster
	}
	if cluster == "" {
		if len(conns) == 1 {
			return &conns[0], nil
		}
		return nil, fmt.Errorf("must specify --cluster=... (one of %v)", clusters)
	}
	for i := range conns {
		connConfig := &conns[i]
		if connConfig.Cluster == cluster {
			return connConfig, nil
		}
	}
	return nil, fmt.Errorf("no such cluster %q, expecting one of %v", cluster, clusters)
}

func (l *Local) GetProxyURL() (string, error) {
	// Allow Proxy URL to be overridden (e.g. for talking to dev/staging).
	if proxyURL := viper.GetString("proxy_url"); proxyURL != "" {
		_, err := url.Parse(proxyURL)
		if err != nil {
			return "", fmt.Errorf("invalid proxy_url: %w", err)
		}
		return proxyURL, nil
	}
	return "https://proxy.signadot.com", nil
}

func (l *Local) GetAPIKey() string {
	return viper.GetString("api_key")
}

type LocalConnect struct {
	*Local

	// Flags
	Cluster      string
	Unprivileged bool
	Wait         ConnectWait
	WaitTimeout  time.Duration

	// Hidden Flags
	DumpCIConfig bool
}

func (c *LocalConnect) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&c.Cluster, "cluster", "", "specify cluster connection config")

	cmd.Flags().BoolVar(&c.Unprivileged, "unprivileged", false, "run without root privileges")
	cmd.Flags().Var(&c.Wait, "wait", "status to wait for while connecting {none,connect,sandboxes}")
	cmd.Flags().DurationVar(&c.WaitTimeout, "wait-timeout", 10*time.Second, "timeout to wait")

	cmd.Flags().BoolVar(&c.DumpCIConfig, "dump-ci-config", false, "dump connect invocation config")
	cmd.Flags().MarkHidden("dump-ci-config")
	cmd.Flags().Lookup("wait").NoOptDefVal = ConnectWaitConnect.String()
}

type ConnectWait int

const (
	ConnectWaitConnect ConnectWait = iota
	ConnectWaitSandboxes
	ConnectWaitNone
)

func (cw ConnectWait) String() string {
	return map[ConnectWait]string{
		ConnectWaitConnect:   "connect",
		ConnectWaitSandboxes: "sandboxes",
		ConnectWaitNone:      "none",
	}[cw]
}

func ParseConnectWait(v string) (ConnectWait, error) {
	cw, ok := map[string]ConnectWait{
		"connect":   ConnectWaitConnect,
		"sandboxes": ConnectWaitSandboxes,
		"none":      ConnectWaitNone,
	}[v]
	if !ok {
		return 0, fmt.Errorf("unknown connect wait value %q (should be connect, sandboxes, or none)", v)
	}
	return cw, nil
}

func (cw *ConnectWait) Set(v string) error {
	tmp, err := ParseConnectWait(v)
	if err != nil {
		return err
	}
	*cw = tmp
	return nil
}

func (cw *ConnectWait) Type() string {
	return "string"
}

type LocalDisconnect struct {
	*Local

	// Flags
	CleanLocalSandboxes bool
}

func (c *LocalDisconnect) AddFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&c.CleanLocalSandboxes, "clean-local-sandboxes", false, "clean active local sandboxes")
}

type LocalStatus struct {
	*Local

	// Flags
	Details bool
}

func (c *LocalStatus) AddFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&c.Details, "details", false, "display status details")
}

type LocalProxy struct {
	*Local

	// Flags
	Sandbox       string
	RouteGroup    string
	Cluster       string
	ProxyMappings []ProxyMapping
}

func (lp *LocalProxy) Validate() error {
	c := 0
	if lp.Sandbox != "" {
		c += 1
	}
	if lp.RouteGroup != "" {
		c += 1
	}
	if lp.Cluster != "" {
		c += 1
	}

	if c == 0 {
		return errors.New("you should specify one of '--sandbox', '--routegroup' or '--cluster'")
	}
	if c > 1 {
		return errors.New("only one of '--sandbox', '--routegroup' or '--cluster' should be specified")
	}

	for i := range lp.ProxyMappings {
		pm := &lp.ProxyMappings[i]
		if err := pm.Validate(); err != nil {
			return err
		}

	}
	return nil
}

type ProxyMapping struct {
	TargetProto string
	TargetAddr  string
	BindAddr    string
}

func (pm *ProxyMapping) Validate() error {
	switch pm.TargetProto {
	case "tcp":
	case "http":
	case "https":
	case "grpc":
	default:
		return fmt.Errorf("unsupported '%s' protocol, only 'tcp', 'http', 'https' and 'grpc' are supported", pm.TargetProto)
	}
	return nil
}

func (pm *ProxyMapping) GetTarget() string {
	return fmt.Sprintf("%s://%s", pm.TargetProto, pm.TargetAddr)
}

func (pm *ProxyMapping) String() string {
	return fmt.Sprintf("%s://%s@%s", pm.TargetProto, pm.TargetAddr, pm.BindAddr)
}

type proxyMappings []ProxyMapping

func (pms *proxyMappings) String() string {
	b := bytes.NewBuffer(nil)
	for i := range *pms {
		pm := &(*pms)[i]
		if i != 0 {
			fmt.Fprintf(b, " ")
		}
		fmt.Fprintf(b, "--map %s", pm)
	}
	return b.String()
}

// Set appends a new argument  to instance of Nargs
func (pms *proxyMappings) Set(arg string) error {
	regex := regexp.MustCompile(`^(.+?)://(.+?)\@(.+)$`)
	matches := regex.FindStringSubmatch(arg)
	if matches == nil || len(matches) != 4 {
		return fmt.Errorf("invalid format, expected \"<target-protocol>://<target-addr>@<bind-addr>\"")
	}

	*pms = append(*pms, ProxyMapping{
		TargetProto: matches[1],
		TargetAddr:  matches[2],
		BindAddr:    matches[3],
	})
	return nil
}

// Type is a no-op
func (pms *proxyMappings) Type() string {
	return "string"
}

func (lp *LocalProxy) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&lp.Sandbox, "sandbox", "s", "", "run the proxy in the context of the specificed sandbox")
	cmd.Flags().StringVarP(&lp.RouteGroup, "routegroup", "r", "", "run the proxy in the context of the specificed routegroup")
	cmd.Flags().StringVarP(&lp.Cluster, "cluster", "c", "", "target cluster")
	cmd.Flags().VarP((*proxyMappings)(&lp.ProxyMappings), "map", "m", "--map <target-protocol>://<target-addr>@<bind-addr>")
}
