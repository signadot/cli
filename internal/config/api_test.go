package config

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/signadot/cli/internal/auth"
	"github.com/spf13/viper"
	"github.com/zalando/go-keyring"
)

// An expired bearer token in the plain text store is refreshed (it used to
// fail with ErrAuthExpired), and the refresh token is kept when the server
// doesn't return a new one.
func TestInitRefreshesExpiredPlainTextAuth(t *testing.T) {
	keyring.MockInit()
	t.Setenv("HOME", t.TempDir())

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/auth/device/token/refresh" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"accessToken": "new", "expiresIn": 3600})
	}))
	defer srv.Close()
	viper.Set("api_url", srv.URL)
	defer viper.Reset()

	expired := time.Now().Add(-time.Minute)
	if err := auth.NewPlainTextStorage().Store(&auth.Auth{
		BearerToken: "old", RefreshToken: "refresh", OrgName: "org", ExpiresAt: &expired,
	}); err != nil {
		t.Fatal(err)
	}

	a := &API{}
	if err := a.init(); err != nil {
		t.Fatalf("init: %v", err)
	}
	if a.BearerToken != "new" {
		t.Errorf("bearer token = %q, want refreshed", a.BearerToken)
	}
	stored, err := auth.NewPlainTextStorage().Get()
	if err != nil || stored == nil {
		t.Fatalf("stored auth: %v %v", stored, err)
	}
	if stored.BearerToken != "new" || stored.RefreshToken != "refresh" {
		t.Errorf("stored = %+v, want new bearer token and kept refresh token", stored)
	}
}
