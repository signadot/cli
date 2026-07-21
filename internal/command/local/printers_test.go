package local

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/signadot/cli/internal/config"
	commonapi "github.com/signadot/cli/internal/locald/api"
	sbmapi "github.com/signadot/cli/internal/locald/api/sandboxmanager"
)

// TestLocalDNSStatusLineCrossVersion covers the CLI-vs-daemon version skew for
// the local-DNS status line: a new CLI rendering the status of a daemon that
// does (new) or does not (old, pre-host_count) report the host count. protobuf
// makes host_count decode as 0 against an older daemon, so the line must omit
// "(0 hosts)" rather than read as a discrepancy against `local hosts`.
func TestLocalDNSStatusLineCrossVersion(t *testing.T) {
	line := func(ldns *commonapi.LocalDNSStatus) string {
		var buf bytes.Buffer
		p := &statusPrinter{cfg: &config.LocalStatus{}, status: &sbmapi.StatusResponse{LocalDns: ldns}, out: &buf}
		p.printLocalDNSStatus()
		return buf.String()
	}

	t.Run("new daemon reports host count", func(t *testing.T) {
		got := line(&commonapi.LocalDNSStatus{
			Health: &commonapi.ServiceHealth{Healthy: true}, RecordCount: 147, HostCount: 49, BindAddr: "127.0.0.54:53",
		})
		if !strings.Contains(got, "147 names (49 hosts) resolvable via local DNS") {
			t.Errorf("got %q, want name and host counts", got)
		}
	})

	t.Run("old daemon (host_count absent -> 0) omits the host count", func(t *testing.T) {
		got := line(&commonapi.LocalDNSStatus{
			Health: &commonapi.ServiceHealth{Healthy: true}, RecordCount: 147, HostCount: 0, BindAddr: "127.0.0.54:53",
		})
		if !strings.Contains(got, "147 names resolvable via local DNS") {
			t.Errorf("got %q, want the plain name count", got)
		}
		if strings.Contains(got, "hosts)") {
			t.Errorf("got %q, want no host count for a pre-host_count daemon", got)
		}
	})

	t.Run("not running", func(t *testing.T) {
		if got := line(nil); !strings.Contains(got, "not running") {
			t.Errorf("got %q", got)
		}
	})
}

// TestGetRawHostsGating covers the section gating for the JSON/YAML status
// output: /etc/hosts status is omitted under --local-dns (where the root
// manager reports no hosts status) rather than emitted as a bogus failure.
func TestGetRawHostsGating(t *testing.T) {
	cfg := &config.LocalStatus{}
	statusMap := map[string]any{}
	hosts := &commonapi.HostsStatus{
		Health:   &commonapi.ServiceHealth{Healthy: true},
		NumHosts: 5,
	}

	t.Run("local-dns mode omits hosts", func(t *testing.T) {
		ci := &config.ConnectInvocationConfig{WithRootManager: true, EnableLocalDNS: true}
		if got := getRawHosts(cfg, ci, hosts, statusMap); got != nil {
			t.Errorf("getRawHosts under --local-dns = %v, want nil", got)
		}
	})

	t.Run("etchosts mode reports hosts", func(t *testing.T) {
		ci := &config.ConnectInvocationConfig{WithRootManager: true, EnableLocalDNS: false}
		got := getRawHosts(cfg, ci, hosts, statusMap)
		if got == nil {
			t.Fatal("getRawHosts in etchosts mode = nil, want a hosts section")
		}
		if js := mustJSON(t, got); !strings.Contains(js, `"healthy":true`) || !strings.Contains(js, `"numHosts":5`) {
			t.Errorf("unexpected hosts json: %s", js)
		}
	})

	t.Run("no root manager returns raw status", func(t *testing.T) {
		ci := &config.ConnectInvocationConfig{WithRootManager: false}
		if got := getRawHosts(cfg, ci, hosts, statusMap); got == nil {
			t.Error("getRawHosts without root manager = nil, want the raw status")
		}
	})
}

// TestGetRawLocalDNSGating is the inverse: local DNS status is present only when
// running under --local-dns with the root manager.
func TestGetRawLocalDNSGating(t *testing.T) {
	cfg := &config.LocalStatus{}
	statusMap := map[string]any{}
	ldns := &commonapi.LocalDNSStatus{
		Health:      &commonapi.ServiceHealth{Healthy: true},
		RecordCount: 7,
		BindAddr:    "127.0.0.54:53",
	}

	t.Run("local-dns mode reports localDNS", func(t *testing.T) {
		ci := &config.ConnectInvocationConfig{WithRootManager: true, EnableLocalDNS: true}
		got := getRawLocalDNS(cfg, ci, ldns, statusMap)
		if got == nil {
			t.Fatal("getRawLocalDNS under --local-dns = nil, want a localDNS section")
		}
		if js := mustJSON(t, got); !strings.Contains(js, `"healthy":true`) {
			t.Errorf("unexpected localDNS json: %s", js)
		}
	})

	t.Run("etchosts mode omits localDNS", func(t *testing.T) {
		ci := &config.ConnectInvocationConfig{WithRootManager: true, EnableLocalDNS: false}
		if got := getRawLocalDNS(cfg, ci, ldns, statusMap); got != nil {
			t.Errorf("getRawLocalDNS in etchosts mode = %v, want nil", got)
		}
	})

	t.Run("no root manager omits localDNS", func(t *testing.T) {
		ci := &config.ConnectInvocationConfig{WithRootManager: false, EnableLocalDNS: true}
		if got := getRawLocalDNS(cfg, ci, ldns, statusMap); got != nil {
			t.Errorf("getRawLocalDNS without root manager = %v, want nil", got)
		}
	})
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(b)
}
