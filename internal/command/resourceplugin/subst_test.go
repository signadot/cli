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
			// The top-level `version:` field is no longer parsed — the
			// only supported form is the @ suffix on `name:`. A stray
			// `version:` should be ignored, not fold into rp.Name.
			name:     "top-level version field is ignored",
			in:       map[string]any{"name": "foo", "version": "1.2.0", "spec": map[string]any{}},
			wantName: "foo",
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
