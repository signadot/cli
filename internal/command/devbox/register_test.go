package devbox

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/signadot/cli/internal/config"
	"github.com/spf13/viper"
	"github.com/zalando/go-keyring"
)

// With auth from the environment only, ~/.signadot may not exist yet.
func TestRegisterCreatesSignadotDir(t *testing.T) {
	keyring.MockInit()
	home := t.TempDir()
	t.Setenv("HOME", home)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"id":"db1"}`)
	}))
	defer srv.Close()
	viper.Set("api_key", "key")
	viper.Set("org", "org")
	viper.Set("api_url", srv.URL)
	defer viper.Reset()

	cfg := &config.DevboxRegister{Devbox: &config.Devbox{API: &config.API{}}}
	if err := register(cfg, io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(home, ".signadot", ".devbox-id"))
	if err != nil || string(got) != "db1" {
		t.Fatalf("devbox id file: %q, %v", got, err)
	}
}
