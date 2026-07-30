package rootmanager

import "testing"

// TestEvalRootServerHealth is the mode matrix behind checkRootServer: health
// must gate on the name-resolution service that is actually active for the mode
// (localdns under --local-dns, etchosts otherwise). The regression it guards
// against is the restart loop where --local-dns gated on the always-nil
// etcHostsSVC and restarted the services every cycle.
func TestEvalRootServerHealth(t *testing.T) {
	tests := []struct {
		name        string
		in          rootServerHealth
		wantOK      bool
		wantRestart bool
	}{
		// --- localdns mode ---
		{
			// The regression scenario: localdns healthy, etchosts absent
			// (nil->false). Must be healthy, NOT a restart.
			name: "localdns healthy, etchosts absent -> healthy",
			in: rootServerHealth{
				enableLocalDNS: true, localnetHealthy: true,
				localDNSHealthy: true, etcHostsHealthy: false, allowRestart: true,
			},
			wantOK: true, wantRestart: false,
		},
		{
			name: "localdns unhealthy past grace -> restart",
			in: rootServerHealth{
				enableLocalDNS: true, localnetHealthy: true,
				localDNSHealthy: false, etcHostsHealthy: false, allowRestart: true,
			},
			wantOK: true, wantRestart: true,
		},
		{
			name: "localdns unhealthy in grace -> wait",
			in: rootServerHealth{
				enableLocalDNS: true, localnetHealthy: true,
				localDNSHealthy: false, etcHostsHealthy: false, allowRestart: false,
			},
			wantOK: false, wantRestart: false,
		},
		{
			// etchosts health must be ignored in localdns mode.
			name: "localdns healthy, etchosts unhealthy -> healthy",
			in: rootServerHealth{
				enableLocalDNS: true, localnetHealthy: true,
				localDNSHealthy: true, etcHostsHealthy: false, allowRestart: true,
			},
			wantOK: true, wantRestart: false,
		},

		// --- etchosts mode ---
		{
			name: "etchosts healthy, localdns absent -> healthy",
			in: rootServerHealth{
				enableLocalDNS: false, localnetHealthy: true,
				localDNSHealthy: false, etcHostsHealthy: true, allowRestart: true,
			},
			wantOK: true, wantRestart: false,
		},
		{
			// localdns health must be ignored in etchosts mode.
			name: "etchosts unhealthy, localdns healthy -> restart",
			in: rootServerHealth{
				enableLocalDNS: false, localnetHealthy: true,
				localDNSHealthy: true, etcHostsHealthy: false, allowRestart: true,
			},
			wantOK: true, wantRestart: true,
		},
		{
			name: "etchosts unhealthy in grace -> wait",
			in: rootServerHealth{
				enableLocalDNS: false, localnetHealthy: true,
				localDNSHealthy: false, etcHostsHealthy: false, allowRestart: false,
			},
			wantOK: false, wantRestart: false,
		},

		// --- localnet gating (independent of mode) ---
		{
			name: "localnet unhealthy past grace -> restart",
			in: rootServerHealth{
				enableLocalDNS: true, localnetHealthy: false,
				localDNSHealthy: true, etcHostsHealthy: true, allowRestart: true,
			},
			wantOK: true, wantRestart: true,
		},
		{
			name: "localnet unhealthy in grace -> wait",
			in: rootServerHealth{
				enableLocalDNS: true, localnetHealthy: false,
				localDNSHealthy: true, etcHostsHealthy: true, allowRestart: false,
			},
			wantOK: false, wantRestart: false,
		},
		{
			name: "all healthy (localdns) -> healthy",
			in: rootServerHealth{
				enableLocalDNS: true, localnetHealthy: true,
				localDNSHealthy: true, etcHostsHealthy: false, allowRestart: false,
			},
			wantOK: true, wantRestart: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, restart := evalRootServerHealth(tt.in)
			if ok != tt.wantOK || restart != tt.wantRestart {
				t.Errorf("evalRootServerHealth(%+v) = (ok=%v, restart=%v), want (ok=%v, restart=%v)",
					tt.in, ok, restart, tt.wantOK, tt.wantRestart)
			}
		})
	}
}
