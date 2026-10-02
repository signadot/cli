package sandbox

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResetDefaultOutputDir(t *testing.T) {
	for _, noClobber := range []bool{false, true} {
		dir := filepath.Join(t.TempDir(), "files")
		f := filepath.Join(dir, "edited.conf")
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(f, []byte("local edit"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := resetDefaultOutputDir(dir, noClobber); err != nil {
			t.Fatal(err)
		}
		_, err := os.Stat(f)
		if kept := err == nil; kept != noClobber {
			t.Errorf("noClobber=%v: file kept=%v", noClobber, kept)
		}
	}
	// a missing directory is fine
	if err := resetDefaultOutputDir(filepath.Join(t.TempDir(), "missing"), false); err != nil {
		t.Fatal(err)
	}
}
