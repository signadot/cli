package local

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	sbmgr "github.com/signadot/cli/internal/locald/sandboxmanager"
	"github.com/signadot/cli/internal/print"
	"github.com/signadot/cli/internal/sdtab"
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
// carries exactly one address: the root controller assigns a single virtual
// address per name and reports it as HostEntry.Ip (IPv4-preferred).
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
		hosts = append(hosts, printableHost{Name: e.Name, IP: e.Ip})
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

type hostRow struct {
	Name string `sdtab:"NAME"`
	IP   string `sdtab:"IP"`
}

// printHosts renders the default aligned NAME/IP table, consistent with the
// other list commands (cluster, routegroup, ...). The entries arrive already
// sorted by name from the root controller (see rootServer.GetHosts).
func printHosts(out io.Writer, hosts []printableHost) error {
	t := sdtab.New[hostRow](out)
	t.AddHeader()
	for _, h := range hosts {
		t.AddRow(hostRow{Name: h.Name, IP: h.IP})
	}
	return t.Flush()
}
