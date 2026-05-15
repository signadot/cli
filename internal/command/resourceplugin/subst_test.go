package resourceplugin

import (
	"strings"
	"testing"
)

func TestUnstructuredToResourcePlugin(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		in          map[string]any
		wantName    string
		wantVersion string
		wantErr     string // substring; empty means no error
	}{
		{
			name:     "bare name",
			in:       map[string]any{"name": "foo", "spec": map[string]any{}},
			wantName: "foo",
		},
		{
			name:        "version suffix in name",
			in:          map[string]any{"name": "foo@1.2.0", "spec": map[string]any{}},
			wantName:    "foo",
			wantVersion: "1.2.0",
		},
		{
			name:        "top-level version field",
			in:          map[string]any{"name": "foo", "version": "1.2.0", "spec": map[string]any{}},
			wantName:    "foo",
			wantVersion: "1.2.0",
		},
		{
			name:    "version in both forms is rejected even when equal",
			in:      map[string]any{"name": "foo@1.2.0", "version": "1.2.0", "spec": map[string]any{}},
			wantErr: "set in both",
		},
		{
			name:    "version in both forms with different values",
			in:      map[string]any{"name": "foo@1.2.0", "version": "2.0.0", "spec": map[string]any{}},
			wantErr: "set in both",
		},
		{
			name:    "non-string version errors loudly",
			in:      map[string]any{"name": "foo", "version": 1.2, "spec": map[string]any{}},
			wantErr: "must be a string",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rp, err := unstructuredToResourcePlugin(tc.in)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected error containing %q, got %q", tc.wantErr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			// rp.Name carries the combined wire form; split it back
			// to verify the loader's name/version resolution.
			gotName, gotVersion := splitNameVersion(rp.Name)
			if gotName != tc.wantName {
				t.Errorf("Name = %q, want %q", gotName, tc.wantName)
			}
			if gotVersion != tc.wantVersion {
				t.Errorf("Version = %q, want %q", gotVersion, tc.wantVersion)
			}
		})
	}
}

func TestSplitNameVersion(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in, wantName, wantVersion string
	}{
		{"foo", "foo", ""},
		{"foo@1.2.0", "foo", "1.2.0"},
		{"foo@latest", "foo", "latest"},
		{"foo@", "foo", ""},
		{"", "", ""},
	}
	for _, tc := range cases {
		name, version := splitNameVersion(tc.in)
		if name != tc.wantName || version != tc.wantVersion {
			t.Errorf("splitNameVersion(%q) = (%q, %q), want (%q, %q)", tc.in, name, version, tc.wantName, tc.wantVersion)
		}
	}
}
