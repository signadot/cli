package local

import (
	"bytes"
	"strings"
	"testing"
)

// TestPrintHosts checks the default table has a header row and one aligned
// NAME/IP row per host, in the order given (already name-sorted by the daemon).
func TestPrintHosts(t *testing.T) {
	var b bytes.Buffer
	err := printHosts(&b, []printableHost{
		{Name: "a.myns.svc.cluster.local", IP: "242.242.0.3"},
		{Name: "b.other.svc.cluster.local", IP: "242.242.0.4"},
	})
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(b.String(), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("got %d lines, want 3 (header + 2 rows):\n%s", len(lines), b.String())
	}
	if !strings.Contains(lines[0], "NAME") || !strings.Contains(lines[0], "IP") {
		t.Errorf("header line missing NAME/IP: %q", lines[0])
	}
	if !strings.HasPrefix(lines[1], "a.myns.svc.cluster.local") || !strings.Contains(lines[1], "242.242.0.3") {
		t.Errorf("row 1 unexpected: %q", lines[1])
	}
	if !strings.HasPrefix(lines[2], "b.other.svc.cluster.local") || !strings.Contains(lines[2], "242.242.0.4") {
		t.Errorf("row 2 unexpected: %q", lines[2])
	}
}

// TestPrintHostsEmpty: no hosts still emits just the header (and no panic).
func TestPrintHostsEmpty(t *testing.T) {
	var b bytes.Buffer
	if err := printHosts(&b, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), "NAME") {
		t.Errorf("empty output missing header: %q", b.String())
	}
}
