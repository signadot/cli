package config

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/signadot/go-sdk/client"
	"github.com/signadot/go-sdk/client/cluster"
	"github.com/signadot/go-sdk/transport"
	"github.com/spf13/cobra"
)

func TestParseHeaders(t *testing.T) {
	got, err := parseHeaders([]string{
		"signadot-client-context: integration=sandbox-action,integration-version=0.1.0",
		"X-Trace:  one ",
		"x-trace: two",
		"X-Empty:",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := http.Header{
		"Signadot-Client-Context": {"integration=sandbox-action,integration-version=0.1.0"},
		"X-Trace":                 {"one", "two"},
		"X-Empty":                 {""},
	}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for k, v := range want {
		if strings.Join(got[k], "|") != strings.Join(v, "|") {
			t.Errorf("%s: got %q, want %q", k, got[k], v)
		}
	}

	if h, err := parseHeaders(nil); h != nil || err != nil {
		t.Errorf("no flags: got %v, %v", h, err)
	}
}

func TestParseHeadersRefuses(t *testing.T) {
	for flag, want := range map[string]string{
		"no-colon":                     `want "Name: value"`,
		": value":                      "not a valid header name",
		"Bad Name: v":                  "not a valid header name",
		"X-A: one\r\nX-B: two":         "control character",
		"X-A: nul\x00here":             "control character",
		"X-A: del\x7fhere":             "control character",
		"X-A: tab\there is fine?":      "",
		"Accept: text/html":            "set by the CLI",
		"accept-encoding: gzip":        "set by the CLI",
		"Te: trailers":                 "set by the CLI",
		"Upgrade: websocket":           "set by the CLI",
		"authorization: Bearer x":      "set by the CLI",
		"Signadot-Api-Key: k":          "set by the CLI",
		"signadot-cluster-token: t":    "set by the CLI",
		"User-Agent: something-else/1": "set by the CLI",
		"Host: evil.example":           "set by the CLI",
	} {
		_, err := parseHeaders([]string{flag})
		if want == "" {
			if err != nil {
				t.Errorf("%q: got %v, want it accepted", flag, err)
			}
			continue
		}
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: got %v, want an error containing %q", flag, err, want)
		}
	}
}

// Through the real SDK client: the extra headers arrive, and neither the
// User-Agent nor the credentials are lost on the way.
func TestExtraHeadersReachTheAPI(t *testing.T) {
	var seen http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Clone()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
	}))
	defer srv.Close()

	for name, extra := range map[string]http.Header{
		"with --header":    {"Signadot-Client-Context": {"integration=sandbox-action"}},
		"without --header": nil,
	} {
		t.Run(name, func(t *testing.T) {
			a := &API{Root: Root{ExtraHeaders: extra}, APIURL: srv.URL, ApiKey: "k", UserAgent: "signadot-cli:test"}
			tr, err := transport.InitAPITransport(a.GetBaseTransport())
			if err != nil {
				t.Fatal(err)
			}
			if _, err := client.New(tr, nil).Cluster.ListClusters(cluster.NewListClustersParams().WithOrgName("o"), nil); err != nil {
				t.Fatal(err)
			}
			if got := seen.Get("Signadot-Client-Context"); got != extra.Get("Signadot-Client-Context") {
				t.Errorf("Signadot-Client-Context: got %q", got)
			}
			if got := seen.Get("User-Agent"); got != "signadot-cli:test" {
				t.Errorf("User-Agent: got %q", got)
			}
			if got := seen.Get("Signadot-Api-Key"); got != "k" {
				t.Errorf("Signadot-Api-Key: got %q", got)
			}
		})
	}
}

func TestHeaderFlagIsHidden(t *testing.T) {
	cmd := newTestCommand()
	f := cmd.PersistentFlags().Lookup("header")
	if f == nil || !f.Hidden {
		t.Fatalf("--header must be registered and hidden, got %+v", f)
	}
	if err := cmd.PersistentFlags().Parse([]string{"--header", "A: 1", "--header", "B: 2"}); err != nil {
		t.Fatal(err)
	}
}

func newTestCommand() *cobra.Command {
	c := &Root{}
	cmd := &cobra.Command{Use: "signadot"}
	c.AddFlags(cmd)
	return cmd
}
