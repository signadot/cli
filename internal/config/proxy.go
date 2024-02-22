package config

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"regexp"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Proxy struct {
	*API

	ProxyURL string
}

func (p *Proxy) InitProxyConfig() error {
	if err := p.API.InitAPIConfig(); err != nil {
		return err
	}

	// Allow Proxy URL to be overridden (e.g. for talking to dev/staging).
	if proxyURL := viper.GetString("proxy_url"); proxyURL != "" {
		_, err := url.Parse(proxyURL)
		if err != nil {
			return fmt.Errorf("invalid proxy_url: %w", err)
		}
		p.ProxyURL = proxyURL

	} else {
		p.ProxyURL = "https://proxy.signadot.com"
	}
	return nil
}

func (p *Proxy) GetAPIKey() string {
	return viper.GetString("api_key")
}

type ProxyConnect struct {
	*Proxy

	// Flags
	Sandbox       string
	RouteGroup    string
	Cluster       string
	ProxyMappings ProxyMappings
}

func (pc *ProxyConnect) Validate() error {
	c := 0
	if pc.Sandbox != "" {
		c += 1
	}
	if pc.RouteGroup != "" {
		c += 1
	}
	if pc.Cluster != "" {
		c += 1
	}

	if c == 0 {
		return errors.New("you should specify one of '--sandbox', '--routegroup' or '--cluster'")
	}
	if c > 1 {
		return errors.New("only one of '--sandbox', '--routegroup' or '--cluster' should be specified")
	}

	for i := range pc.ProxyMappings {
		pm := &pc.ProxyMappings[i]
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
	return fmt.Sprintf("%s://%s|%s", pm.TargetProto, pm.TargetAddr, pm.BindAddr)
}

type ProxyMappings []ProxyMapping

func (pms *ProxyMappings) String() string {
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
func (pms *ProxyMappings) Set(arg string) error {
	regex := regexp.MustCompile(`^(.+?)://(.+?)\|(.+)$`)
	matches := regex.FindStringSubmatch(arg)
	if matches == nil || len(matches) != 4 {
		return fmt.Errorf("invalid format, expected \"<target-protocol>://<target-addr>|<bind-addr>\"")
	}

	*pms = append(*pms, ProxyMapping{
		TargetProto: matches[1],
		TargetAddr:  matches[2],
		BindAddr:    matches[3],
	})
	return nil
}

// Type is a no-op
func (pms *ProxyMappings) Type() string {
	return "string"
}

func (c *ProxyConnect) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&c.Sandbox, "sandbox", "s", "", "run the proxy in the context of the specificed sandbox")
	cmd.Flags().StringVarP(&c.RouteGroup, "routegroup", "r", "", "run the proxy in the context of the specificed routegroup")
	cmd.Flags().StringVarP(&c.Cluster, "cluster", "c", "", "target cluster")
	cmd.Flags().VarP(&c.ProxyMappings, "map", "m", "--map <target-protocol>://<target-addr>|<bind-addr>")
}
