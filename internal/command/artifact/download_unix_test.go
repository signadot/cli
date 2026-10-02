//go:build unix

package artifact

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// -o names where the output goes, devices included: /dev/null is written to,
// not replaced, and nothing is created beside it.
func TestDownloadWritesToADevice(t *testing.T) {
	cfg := fakeDownload(t, serveArtifact("content"))
	cfg.OutputFile = os.DevNull
	if err := download(context.Background(), cfg, io.Discard, "report.xml"); err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Lstat(os.DevNull); err != nil || fi.Mode()&os.ModeDevice == 0 {
		t.Fatalf("%s is no longer a device: %v %v", os.DevNull, fi.Mode(), err)
	}
}

// A new file gets the mode the umask allows, as any file created by name does.
func TestDownloadNewFileHonorsUmask(t *testing.T) {
	old := syscall.Umask(0o077)
	t.Cleanup(func() { syscall.Umask(old) })
	cfg := fakeDownload(t, serveArtifact("content"))
	cfg.OutputFile = filepath.Join(t.TempDir(), "report.xml")
	if err := download(context.Background(), cfg, io.Discard, "report.xml"); err != nil {
		t.Fatal(err)
	}
	if fi, _ := os.Stat(cfg.OutputFile); fi.Mode().Perm() != 0o600 {
		t.Errorf("mode = %v, want 0600 under umask 077", fi.Mode().Perm())
	}
}
