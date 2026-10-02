package auth

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// Readers concurrent with writes must always see complete credentials.
func TestPlainTextConcurrentReadWrite(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	a := &Auth{BearerToken: strings.Repeat("t", 64<<10), OrgName: "org"}
	if err := storeAuthInPlainText(a); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	stop := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			if err := storeAuthInPlainText(a); err != nil {
				t.Error(err)
				return
			}
		}
	}()
	for i := 0; i < 500; i++ {
		got, err := getAuthFromPlainText()
		if err != nil || got == nil || got.OrgName != "org" {
			close(stop)
			wg.Wait()
			t.Fatalf("read %d: %+v, %v", i, got, err)
		}
	}
	close(stop)
	wg.Wait()
}

func TestPlainTextInvalidFileKept(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	p := filepath.Join(home, ".signadot", credentialsFileName)
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("{not json"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := getAuthFromPlainText(); err == nil {
		t.Fatal("expected error")
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("credentials file removed: %v", err)
	}
}
