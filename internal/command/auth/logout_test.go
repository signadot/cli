package auth

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/signadot/cli/internal/config"
	"github.com/spf13/viper"
	"github.com/zalando/go-keyring"
)

// An unreadable credentials file must not block the command that clears it.
func TestLogoutRemovesAnInvalidCredentialsFile(t *testing.T) {
	keyring.MockInit()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Cleanup(viper.Reset)
	p := filepath.Join(home, ".signadot", "credentials")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	if err := runLogout(&config.AuthLogout{Auth: &config.Auth{}}, &out); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Errorf("credentials file still there: %v", err)
	}
	if !strings.Contains(out.String(), "logged out") {
		t.Errorf("output: %q", out.String())
	}
}
