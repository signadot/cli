package bug

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/signadot/cli/internal/config"
)

// The bug report must include the build info and init error, not only the
// API config (whose MarshalJSON used to be promoted).
func TestBugConfigJSON(t *testing.T) {
	d, err := json.Marshal(&BugConfig{
		Config:    &config.API{Org: "my-org"},
		BuildInfo: "v1.2.3",
		Error:     "no auth",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"buildInfo":"v1.2.3"`, `"error":"no auth"`, `"Org":"my-org"`} {
		if !strings.Contains(string(d), want) {
			t.Errorf("missing %s in %s", want, d)
		}
	}
}
