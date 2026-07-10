package local

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/signadot/cli/internal/config"
	commonapi "github.com/signadot/cli/internal/locald/api"
)

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
