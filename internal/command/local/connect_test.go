package local

import (
	"os"
	"strconv"
	"testing"

	"github.com/signadot/cli/internal/config"
)

func TestCheckNotConnected(t *testing.T) {
	for _, isRoot := range []bool{true, false} {
		dir := t.TempDir()
		if err := checkNotConnected(dir); err != nil {
			t.Fatalf("no pidfiles: %v", err)
		}
		// a live process (ours) holding either pidfile means connected
		pidFile := config.GetLocaldPIDfile(dir, isRoot)
		if err := os.WriteFile(pidFile, []byte(strconv.Itoa(os.Getpid())), 0600); err != nil {
			t.Fatal(err)
		}
		if err := checkNotConnected(dir); err == nil {
			t.Fatalf("root=%v pidfile held: expected already connected", isRoot)
		}
	}
}
