package local

import (
	"fmt"
	"io"
	"net"

	"github.com/signadot/cli/internal/config"
	sbmgr "github.com/signadot/cli/internal/locald/sandboxmanager"
	"github.com/signadot/cli/internal/print"
	"github.com/spf13/cobra"
)

func newHosts(localConfig *config.Local) *cobra.Command {
	cfg := &config.LocalHosts{Local: localConfig}

	cmd := &cobra.Command{
		Use:   "hosts",
		Short: "List the cluster hosts resolvable from the local machine and their IP addresses",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHosts(cfg, cmd.OutOrStdout(), args)
		},
	}
	cfg.AddFlags(cmd)

	return cmd
}

// printableHost is the JSON/YAML shape of a single resolvable host. Each host
// carries exactly one address (see pickIP), which matches how the cluster
// assigns names: a single-stack cluster hands out one family, and even when
// both are present a resolver answers a name with one address at a time.
type printableHost struct {
	Name string `json:"name"`
	IP   string `json:"ip"`
}

func runHosts(cfg *config.LocalHosts, out io.Writer, args []string) error {
	if err := cfg.InitLocalConfig(); err != nil {
		return err
	}
	resp, err := sbmgr.GetHosts()
	if err != nil {
		return err
	}

	hosts := make([]printableHost, 0, len(resp.Entries))
	for _, e := range resp.Entries {
		ip := pickIP(e.Ips)
		if ip == "" {
			// A name with no usable address is not resolvable; skip it.
			continue
		}
		hosts = append(hosts, printableHost{Name: e.Name, IP: ip})
	}

	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return printHosts(out, hosts)
	case config.OutputFormatJSON:
		return print.RawJSON(out, hosts)
	case config.OutputFormatYAML:
		return print.RawK8SYAML(out, hosts)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}

// pickIP selects the single address to report for a host: the IPv4 address when
// one is present, otherwise the first IPv6 address (covering an IPv6-only
// cluster). It returns "" when there is no parseable address.
func pickIP(ips []string) string {
	var fallback string
	for _, ip := range ips {
		parsed := net.ParseIP(ip)
		if parsed == nil {
			continue
		}
		if parsed.To4() != nil {
			return ip
		}
		if fallback == "" {
			fallback = ip
		}
	}
	return fallback
}

// printHosts writes one "<fqdn> <ip>" line per host. The entries arrive already
// sorted by name from the root controller (see rootServer.GetHosts).
func printHosts(out io.Writer, hosts []printableHost) error {
	for _, h := range hosts {
		if _, err := fmt.Fprintf(out, "%s %s\n", h.Name, h.IP); err != nil {
			return err
		}
	}
	return nil
}
