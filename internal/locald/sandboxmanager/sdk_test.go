package sandboxmanager

import (
	"strings"
	"testing"

	commonapi "github.com/signadot/cli/internal/locald/api"
)

func TestCheckLocalDNSStatus(t *testing.T) {
	tests := []struct {
		name       string
		in         *commonapi.LocalDNSStatus
		wantErr    bool
		wantSubstr string
	}{
		{"nil status", nil, true, "failed to configure local DNS resolver"},
		{"nil health", &commonapi.LocalDNSStatus{}, true, "failed to configure local DNS resolver"},
		{
			"healthy",
			&commonapi.LocalDNSStatus{Health: &commonapi.ServiceHealth{Healthy: true}},
			false, "",
		},
		{
			"unhealthy with reason",
			&commonapi.LocalDNSStatus{Health: &commonapi.ServiceHealth{
				Healthy: false, LastErrorReason: "bind failed"}},
			true, "bind failed",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkLocalDNSStatus(tt.in)
			if tt.wantErr != (err != nil) {
				t.Fatalf("checkLocalDNSStatus err = %v, wantErr = %v", err, tt.wantErr)
			}
			if err != nil && tt.wantSubstr != "" && !strings.Contains(err.Error(), tt.wantSubstr) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.wantSubstr)
			}
		})
	}
}

func TestCheckHostsStatus(t *testing.T) {
	tests := []struct {
		name    string
		in      *commonapi.HostsStatus
		wantErr bool
	}{
		{"nil status", nil, true},
		{"nil health", &commonapi.HostsStatus{}, true},
		{"healthy", &commonapi.HostsStatus{Health: &commonapi.ServiceHealth{Healthy: true}}, false},
		{"unhealthy", &commonapi.HostsStatus{Health: &commonapi.ServiceHealth{Healthy: false}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := checkHostsStatus(tt.in); tt.wantErr != (err != nil) {
				t.Errorf("checkHostsStatus err = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}
