package secret

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/go-sdk/models"
	"github.com/spf13/viper"
	"github.com/zalando/go-keyring"
)

func TestUpdateKeepsDescription(t *testing.T) {
	cases := []struct {
		name           string
		description    string
		descriptionSet bool
		want           string
	}{
		{"not given", "", false, "current"},
		{"new", "new", true, "new"},
		{"explicitly empty", "", true, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			keyring.MockInit()
			t.Setenv("HOME", t.TempDir())
			var put *models.Secret
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.Method {
				case http.MethodGet:
					json.NewEncoder(w).Encode(&models.Secret{Name: "s", Description: "current"})
				case http.MethodPut:
					put = &models.Secret{}
					json.NewDecoder(r.Body).Decode(put)
					json.NewEncoder(w).Encode(put)
				}
			}))
			defer srv.Close()
			viper.Set("api_key", "key")
			viper.Set("org", "org")
			viper.Set("api_url", srv.URL)
			defer viper.Reset()

			cfg := &config.SecretUpdate{
				Secret:      &config.Secret{API: &config.API{}},
				Value:       "v",
				Description: c.description,
			}
			if err := update(cfg, c.descriptionSet, io.Discard, io.Discard, []string{"s"}); err != nil {
				t.Fatal(err)
			}
			if put == nil || put.Description != c.want {
				t.Fatalf("sent %+v, want description %q", put, c.want)
			}
		})
	}
}
