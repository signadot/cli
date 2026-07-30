package rootmanager

import (
	"net"
	"testing"
)

// TestPickIP covers the single-address selection GetHosts reports per host:
// IPv4 preferred, IPv6 fallback, empty when there is nothing to report.
func TestPickIP(t *testing.T) {
	v4 := net.ParseIP("242.242.0.3")
	v4b := net.ParseIP("242.242.0.9")
	v6 := net.ParseIP("fd00:5161::3")
	v6b := net.ParseIP("fd00:5161::9")

	tests := []struct {
		name string
		in   []net.IP
		want string
	}{
		{"nil", nil, ""},
		{"empty", []net.IP{}, ""},
		{"v4 only", []net.IP{v4}, "242.242.0.3"},
		{"v6 only", []net.IP{v6}, "fd00:5161::3"},
		{"v4 before v6", []net.IP{v4, v6}, "242.242.0.3"},
		{"v6 before v4 -> still v4", []net.IP{v6, v4}, "242.242.0.3"},
		{"two v6 -> first", []net.IP{v6, v6b}, "fd00:5161::3"},
		{"two v4 -> first", []net.IP{v4, v4b}, "242.242.0.3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pickIP(tt.in); got != tt.want {
				t.Errorf("pickIP(%v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
