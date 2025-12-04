package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/signadot/libconnect/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"sigs.k8s.io/yaml"
)

var onlyDigitsString = regexp.MustCompile(`^[0-9]+$`)

const (
	DefaultVirtualIPNet = "242.242.0.1/16"
)

type Local struct {
	*API

	// initialized from ~/.signadot/config.yaml
	LocalConfig *config.Config
}

func (l *Local) InitLocalConfig() error {
	if err := l.API.InitAPIConfig(); err != nil {
		return err
	}

	type Tmp struct {
		Local *config.Config `json:"local"`
	}
	localConfig := &Tmp{}
	configFile := viper.ConfigFileUsed()
	if configFile == "" {
		return errors.New("config file needed for local configuration, see https://www.signadot.com/docs/getting-started/installation/signadot-cli")
	}
	d, e := os.ReadFile(configFile)
	if e != nil {
		return fmt.Errorf("error reading config file %q: %w", configFile, e)
	}
	if e := yaml.Unmarshal(d, localConfig); e != nil {
		return fmt.Errorf("error unmarshalling config file %q: %w", configFile, e)
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

func (l *Local) GetAPIKey() string {
	return viper.GetString("api_key")
}

type LocalConnect struct {
	*Local

	// Flags
	Cluster      string
	Devbox       string
	Unprivileged bool
	Wait         ConnectWait
	WaitTimeout  time.Duration

	// Hidden Flags
	DumpCIConfig bool
	PProfAddr    string
}

func (c *LocalConnect) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&c.Cluster, "cluster", "", "specify cluster connection config")
	cmd.Flags().StringVar(&c.Devbox, "devbox", "", "specify devbox ID to use for this connection")

	cmd.Flags().BoolVar(&c.Unprivileged, "unprivileged", false, "run without root privileges")
	cmd.Flags().Var(&c.Wait, "wait", "status to wait for while connecting {none,connect,sandboxes}")
	cmd.Flags().DurationVar(&c.WaitTimeout, "wait-timeout", 10*time.Second, "timeout to wait")

	cmd.Flags().BoolVar(&c.DumpCIConfig, "dump-ci-config", false, "dump connect invocation config")
	cmd.Flags().MarkHidden("dump-ci-config")
	cmd.Flags().Lookup("wait").NoOptDefVal = ConnectWaitConnect.String()
	cmd.Flags().StringVar(&c.PProfAddr, "pprof", "", "pprof listen address")
	cmd.Flags().MarkHidden("pprof")
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

	// Hidden Flags
	PProfAddr string
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
	// Allow routeGroup + cluster combination
	if c == 2 && lp.RouteGroup != "" && lp.Cluster != "" {
		return lp.validateProxyMappings()
	}
	if c > 1 {
		return errors.New("only one of '--sandbox', '--routegroup' or '--cluster' should be specified")
	}
	return lp.validateProxyMappings()
}

func (lp *LocalProxy) validateProxyMappings() error {
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
	cmd.Flags().StringVar(&lp.PProfAddr, "pprof", "", "pprof listen address")
	cmd.Flags().MarkHidden("pprof")
}

type LocalOverride struct {
	*Local
}

type LocalOverrideCreate struct {
	*LocalOverride

	// Flags
	Sandbox  string
	Port     int64
	To       string
	Workload string
	Detach   bool

	// Policy
	ExcludedStatusCodes []int `json:"excludedStatusCodes"`

	WaitTimeout time.Duration
}

func (lo *LocalOverrideCreate) AddFlags(cmd *cobra.Command) {
	// Flags
	cmd.Flags().StringVar(&lo.Sandbox, "sandbox", "",
		"name of the sandbox whose traffic will be overridden")

	cmd.Flags().StringVarP(&lo.Workload, "workload", "w", "",
		"name of the workload to override traffic for")

	cmd.Flags().Int64VarP(&lo.Port, "workload-port", "p", 0,
		"port on the sandbox workload to intercept traffic from")

	cmd.Flags().StringVar(&lo.To, "with", "",
		"target address of the override destination (e.g., localhost:9999) where traffic will be forwarded")

	cmd.Flags().BoolVarP(&lo.Detach, "detach", "d", false,
		"run in detached mode so the override remains active after the CLI session ends")

	cmd.Flags().DurationVar(&lo.WaitTimeout, "wait-timeout", 3*time.Minute,
		"maximum time to wait for the sandbox to become ready before failing")

	// Policy
	cmd.Flags().IntSliceVar(&lo.ExcludedStatusCodes, "except-status", []int{},
		"comma-separated list of HTTP status codes to bypass override. "+
			"Responses with these codes will fall through to the sandboxed destination (e.g., 404,503)")

	cmd.MarkFlagRequired("sandbox")
	cmd.MarkFlagRequired("workload")
	cmd.MarkFlagRequired("workload-port")
	cmd.MarkFlagRequired("with")
}

func (lo *LocalOverrideCreate) Validate() error {
	if lo.Sandbox == "" {
		return errors.New("--sandbox is required")
	}

	if lo.Workload == "" {
		return errors.New("--workload is required")
	}

	if lo.Port <= 0 || lo.Port > 65535 {
		return errors.New("--port must be a value between 1 and 65535")
	}

	if lo.To == "" {
		return errors.New("--to is required")
	}

	for _, code := range lo.ExcludedStatusCodes {
		if code < 100 || code > 599 {
			return errors.New("invalid except-status response code, should be between 100 and 599")
		}
	}

	to, err := parseTo(lo.To)
	if err != nil {
		return err
	}
	lo.To = to

	return nil
}

// parseTo allow to receive only the numeric port without the hostname
// and return the formatted string with the hostname
func parseTo(to string) (string, error) {
	if onlyDigitsString.MatchString(to) {
		if port, err := strconv.Atoi(to); err != nil || port <= 0 || port > 65535 {
			return "", fmt.Errorf("invalid port, should be a value between 1 and 65535")
		}

		return fmt.Sprintf("localhost:%s", to), nil
	}

	return to, nil
}

type LocalOverrideDelete struct {
	*LocalOverride

	// Flags
	Sandbox string
}

func (lod *LocalOverrideDelete) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&lod.Sandbox, "sandbox", "", "sandbox containing the override to delete")

	cmd.MarkFlagRequired("sandbox")
}

type LocalOverrideList struct {
	*LocalOverride
}
